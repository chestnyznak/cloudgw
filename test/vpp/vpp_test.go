// Testing the correct operation of all VPP functions using Bin API
package vpp

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.fd.io/govpp/binapi/interface_types"
	"go.fd.io/govpp/binapi/ip"

	"git.crptech.ru/cloud/cloudgw/internal/config"
	"git.crptech.ru/cloud/cloudgw/internal/model"
	"git.crptech.ru/cloud/cloudgw/internal/repository/vpp"
	"git.crptech.ru/cloud/cloudgw/pkg/logger"
	"git.crptech.ru/cloud/cloudgw/pkg/netutils"
	"git.crptech.ru/cloud/cloudgw/pkg/testcontainers"
	"git.crptech.ru/cloud/cloudgw/pkg/unixsockrelay"
)

var cloudgwConfigPath = "config.yml"

func TestVPPFunc(t *testing.T) {
	t.Parallel()

	var (
		vppAPITCPPort        = "9568"           // should be different from vpp_static_test container
		vRouterAddr          = "192.0.0.99"     // fake vRouter serving the VM with floating ip
		vmFIPAddress         = "172.16.0.99/32" // fake VM with assigned floating ip in vrf id = 1
		vmMPLSLabel   uint32 = 1000             // value for vrf id = 1
		vppVRFTables         = make(map[uint32]*model.VPPVRFTable)
	)

	// initial shell command to add interfaces for vpp container and start Unix sock listening to tcp relay

	vppStartupShellCmds := []string{
		"ip link add name contif type veth peer name vppif",
		"ip link add link contif name contsubif type vlan id 4000",
		"ip link set dev contif up",
		"ip link set dev vppif up",
		"ip link set dev contsubif up",
		"vppctl create host-interface name vppif",
		"ip addr add 192.0.0.254/24 dev contif",
		"ip addr add 192.0.1.254/24 dev contsubif",
		"apt update && apt install socat -y",
		"socat TCP-LISTEN:" + vppAPITCPPort + ",fork UNIX-CONNECT:/run/vpp/api.sock &",
	}

	// import startup config from YAML

	cfg, err := config.ParseConfig(cloudgwConfigPath)
	if err != nil {
		logger.Fatal("failed to parsing the yaml config file", "error", err)
	}

	// prepare map with vpp routing and interfaces tables

	// VPPVRFTable[0] is global routing table

	vppRoutingTbl := model.NewVPPVRFTable(
		"",
		0,
		interface_types.InterfaceIndex(cfg.VPP.MainInterfaceID),
		model.UndefinedSubIf,
		0,
		netutils.Addr(cfg.VPP.TunLocalIP),
		netutils.MaskLen(cfg.VPP.TunLocalIP),
		cfg.VPP.TunDefaultGW,
		model.UndefinedLabel,
		nil,
	)
	vppVRFTables[0] = &vppRoutingTbl

	// VPPVRFTable[1, ...] - all other vrfs

	for _, vrf := range cfg.VRF {
		mplsLocalLabel, err := netutils.MPLSLabel(vrf.LocalIP)
		if err != nil {
			logger.Fatal("failed to create mpls local label", "error", err)
		}

		vppRoutingTbl := model.NewVPPVRFTable(
			vrf.VRFName,
			vrf.VRFID,
			interface_types.InterfaceIndex(cfg.VPP.MainInterfaceID),
			model.UndefinedSubIf, // will be defined when create sub-interfaces
			vrf.VLANID,
			netutils.Addr(vrf.LocalIP),
			netutils.MaskLen(vrf.LocalIP),
			vrf.BGPPeerIP,
			mplsLocalLabel,
			vrf.FIPPrefixes,
		)
		vppVRFTables[vrf.VRFID] = &vppRoutingTbl
	}

	// create a vpp container

	logger.Info("creating context and containers")

	ctx := context.Background()

	vpp1Cont, err := testcontainers.CreateVppContainer(ctx, vppAPITCPPort)
	if err != nil {
		logger.Error("failed to create vpp container", "error", err)
		os.Exit(1)
	}

	defer vpp1Cont.DeleteVppContainer(ctx)

	time.Sleep(time.Second * 1)

	// run vpp setup shell commands to prepare the vpp container

	for _, command := range vppStartupShellCmds {
		vpp1Cont.ExecCmd(ctx, []string{"/bin/sh", "-c", command})
		time.Sleep(time.Millisecond * 100)
	}

	time.Sleep(time.Second * 1)

	// to avoid interference with another container

	vppAPISockFile := vppAPITCPPort + "_" + cfg.VPP.BinAPISock

	// create local unix socket file on test host to communicate with the vpp container

	logger.Info("creating unix socket file")

	go unixsockrelay.UnixToTCPRelay(
		vppAPISockFile,
		vpp1Cont.GetIP(ctx),
		vppAPITCPPort)

	defer func() {
		_ = os.Remove(vppAPISockFile)
	}()

	time.Sleep(time.Second * 1)

	// testing connect to vpp api through local host unix sock file

	vppStream, disconnect, _, err := vpp.ConnectToVPPAPIAsync(ctx, vppAPISockFile)
	defer disconnect()
	require.NoError(t, err)

	// testing getting vpp version

	version, err := vpp.GetVPPVersion(vppStream)
	require.NoError(t, err)
	require.Contains(t, version, testcontainers.VPPVersion)

	logger.Info("got vpp version", "version", version)

	// testing create/delete mpls table

	err = vpp.AddDelMPLSTable(vppStream, true)
	require.NoError(t, err)

	err = vpp.AddDelMPLSTable(vppStream, false)
	require.NoError(t, err)
	err = vpp.AddDelMPLSTable(vppStream, true)
	require.NoError(t, err)

	// testing getting interface ids

	ifs, err := vpp.GetInterfaceIDs(vppStream)
	require.NoError(t, err)
	require.Equal(t, 2, len(ifs)) // loopback interface (0) and main interface (1), totally 2

	// testing setup/reset main interface

	err = vpp.SetupMainInterface(vppStream, *vppVRFTables[0])
	require.NoError(t, err)

	err = vpp.ResetMainInterface(vppStream, 1)
	require.NoError(t, err)

	err = vpp.SetupMainInterface(vppStream, *vppVRFTables[0])
	require.NoError(t, err)

	// testing add/del vrfF routing table (one vrf is enough)

	err = vpp.AddDelVRF(vppStream, true, *vppVRFTables[1])
	require.NoError(t, err)

	err = vpp.AddDelVRF(vppStream, false, *vppVRFTables[1])
	require.NoError(t, err)

	err = vpp.AddDelVRF(vppStream, true, *vppVRFTables[1])
	require.NoError(t, err)

	// testing add/del sub-interface

	subIfID, err := vpp.AddSubInterface(vppStream, vppVRFTables[1])
	require.NoError(t, err)
	require.Equal(t, 2, int(subIfID))

	deletedSubInterfaceIDs, err := vpp.DelSubInterfaces(vppStream, cfg.VPP.MainInterfaceID)
	require.NoError(t, err)
	require.Equal(t, uint32(1), deletedSubInterfaceIDs)

	_, err = vpp.AddSubInterface(vppStream, vppVRFTables[1])
	require.NoError(t, err)

	// testing add/del/count/check empty/dump udp uncap

	isUDPEmpty, err := vpp.CheckUDPTunnelTableEmpty(vppStream)
	require.NoError(t, err)
	require.True(t, isUDPEmpty)

	udpTunnelNums, err := vpp.CountUDPTunnels(vppStream)
	require.NoError(t, err)
	require.Equal(t, float64(0), udpTunnelNums) // as udp tunnel table is empty

	testUDPTunnel := model.NewVPPUDPTunnel(
		model.UndefinedTunnelID,
		netutils.Addr(cfg.VPP.TunLocalIP),
		vRouterAddr,
		model.RandUDPTunnelSrcPort(),
	)

	err = vpp.AddUDPTunnel(vppStream, &testUDPTunnel)
	require.NoError(t, err)
	require.Less(t, testUDPTunnel.TunnelID, uint32(4294967295))
	require.Less(t, testUDPTunnel.SrcPort, uint16(65535))
	require.Greater(t, testUDPTunnel.SrcPort, uint16(49151))

	udpTunnelNums, err = vpp.CountUDPTunnels(vppStream)
	require.NoError(t, err)
	require.Equal(t, float64(1), udpTunnelNums) // udp tunnel has 1 element

	isUDPEmpty, err = vpp.CheckUDPTunnelTableEmpty(vppStream)
	require.NoError(t, err)
	require.False(t, isUDPEmpty)

	dumpedUDPTunnel, err := vpp.DumpUDPTunnels(vppStream)
	require.NoError(t, err)
	require.Equal(t, testUDPTunnel.TunnelID, dumpedUDPTunnel[0].TunnelID)

	err = vpp.DelUDPTunnel(vppStream, dumpedUDPTunnel[0].TunnelID)
	require.NoError(t, err)

	isUDPEmpty, err = vpp.CheckUDPTunnelTableEmpty(vppStream)
	require.NoError(t, err)
	require.True(t, isUDPEmpty)

	err = vpp.AddUDPTunnel(vppStream, &testUDPTunnel)
	require.NoError(t, err)

	// testing add add decap

	err = vpp.AddUDPDecap(vppStream)
	require.NoError(t, err)

	// testing add/del/lookup vm floating ip route (/32 to the vm, vrf id = 1) with 2 nextHops

	err = vpp.AddUDPTunnel(vppStream, &model.VPPUDPTunnel{
		RoutingTableID: 0,
		SrcIP:          netutils.Addr(cfg.VPP.TunLocalIP),
		DstIP:          "192.0.0.111",
		SrcPort:        model.RandUDPTunnelSrcPort(),
		DstPort:        6635,
	})
	require.NoError(t, err)

	err = vpp.AddUDPTunnel(vppStream, &model.VPPUDPTunnel{
		RoutingTableID: 0,
		SrcIP:          netutils.Addr(cfg.VPP.TunLocalIP),
		DstIP:          "192.0.0.222",
		SrcPort:        model.RandUDPTunnelSrcPort(),
		DstPort:        6635,
	})
	require.NoError(t, err)

	vmFIPRoute := model.NewVPPIPRoute(
		cfg.VRF[0].VRFID,
		interface_types.InterfaceIndex(cfg.VPP.MainInterfaceID),
		2,
		"192.168.0.100/32",
		[]string{"1.1.1.1", "2.2.2.2"},
		[]uint32{1, 2},
		[]uint32{1111, 2222},
	)

	require.NoError(t, err)

	err = vpp.AddDelFIPRoute(vppStream, true, &vmFIPRoute)
	require.NoError(t, err)

	dumpedFIPRoute, err := vpp.DumpFIPRoutes(vppStream)
	require.NoError(t, err)
	require.Equal(t, 1, len(dumpedFIPRoute))
	require.Equal(t, 2, len(dumpedFIPRoute[0].NextHops))
	require.Equal(t, 2, len(dumpedFIPRoute[0].TunnelIDs))
	require.Equal(t, 2, len(dumpedFIPRoute[0].FIPMPLSLabels))

	isFound, err := vpp.LookupFIPRoute(vppStream, &vmFIPRoute)
	require.NoError(t, err)
	require.True(t, isFound)
	require.Equal(t, uint32(1), vmFIPRoute.TunnelIDs[0])

	err = vpp.AddDelFIPRoute(vppStream, false, &vmFIPRoute)
	require.NoError(t, err)

	dumpedFIPRoute, err = vpp.DumpFIPRoutes(vppStream)
	require.NoError(t, err)
	require.Equal(t, 0, len(dumpedFIPRoute))

	// testing add/del/lookup vp floating ip route (/32 to the vm, vrf id = 1) with one nexthop

	vmFIPRoute = model.NewVPPIPRoute(
		cfg.VRF[0].VRFID,
		interface_types.InterfaceIndex(cfg.VPP.MainInterfaceID),
		2,
		vmFIPAddress,
		[]string{vRouterAddr},
		[]uint32{testUDPTunnel.TunnelID},
		[]uint32{vmMPLSLabel},
	)
	err = vpp.AddDelFIPRoute(vppStream, true, &vmFIPRoute)
	require.NoError(t, err)

	dumpedFIPRoute, err = vpp.DumpFIPRoutes(vppStream)
	require.NoError(t, err)
	require.Equal(t, vmFIPAddress, dumpedFIPRoute[0].Prefix)

	incompleteVMFIPRoute := model.NewVPPIPRoute(
		cfg.VRF[0].VRFID,
		model.UndefinedMainIf,
		model.UndefinedSubIf,
		vmFIPAddress,
		[]string{""},
		[]uint32{model.UndefinedTunnelID},
		[]uint32{model.UndefinedLabel},
	)
	isFound, err = vpp.LookupFIPRoute(vppStream, &incompleteVMFIPRoute)
	require.NoError(t, err)
	require.True(t, isFound)
	require.Equal(t, testUDPTunnel.TunnelID, incompleteVMFIPRoute.TunnelIDs[0])

	// testing add/del mpls local label (one vrf is enough)

	err = vpp.AddDelMPLSLocalLabelRoute(vppStream, true, *vppVRFTables[1])
	require.NoError(t, err)

	err = vpp.AddDelMPLSLocalLabelRoute(vppStream, false, *vppVRFTables[1])
	require.NoError(t, err)

	err = vpp.AddDelMPLSLocalLabelRoute(vppStream, true, *vppVRFTables[1])
	require.NoError(t, err)

	// testing add/del/get/dump/count ip routes and tables

	tables, err := vpp.GetRoutingTables(vppStream)
	require.NoError(t, err)
	require.Greater(t, len(tables), 1) // Global + VRF(s)

	// in global routing table (id=0)

	err = vpp.AddDelIPRoute(vppStream, true, model.VPPIPRoute{
		VRFID:           0,
		Prefix:          "0.0.0.0/0",
		NextHops:        []string{cfg.VPP.TunDefaultGW},
		MainInterfaceID: interface_types.InterfaceIndex(cfg.VPP.MainInterfaceID),
		SubInterfaceID:  model.UndefinedSubIf,
		TunnelIDs:       []uint32{model.UndefinedTunnelID},
		FIPMPLSLabels:   []uint32{model.UndefinedLabel},
	})
	require.NoError(t, err)

	// in vrf (id=1)

	err = vpp.AddDelIPRoute(vppStream, true, model.VPPIPRoute{
		VRFID:           cfg.VRF[0].VRFID,
		Prefix:          "10.0.0.0/24",
		NextHops:        []string{cfg.VRF[0].BGPPeerIP},
		MainInterfaceID: interface_types.InterfaceIndex(cfg.VPP.MainInterfaceID),
		SubInterfaceID:  2,
		TunnelIDs:       []uint32{model.UndefinedTunnelID},
		FIPMPLSLabels:   []uint32{model.UndefinedLabel},
	})
	require.NoError(t, err)

	allFIPRoutes, err := vpp.DumpFIPRoutes(vppStream)
	require.NoError(t, err)
	require.Greater(t, len(allFIPRoutes), 0)

	dumpedIPRoutes, err := vpp.DumpIPRoutes(vppStream)
	require.NoError(t, err)
	require.Greater(t, len(dumpedIPRoutes), 0)

	ipRouteCount, fipRouteCount, err := vpp.CountRoutesPerTable(vppStream, ip.IPTable{
		TableID: cfg.VRF[0].VRFID,
		IsIP6:   false,
		Name:    cfg.VRF[0].VRFName,
	})
	require.NoError(t, err)
	require.Equal(t, float64(2), ipRouteCount)  // 1 created ip + 1 connected
	require.Equal(t, float64(1), fipRouteCount) // 1 floating ip route record

	err = vpp.AddDelIPRoute(vppStream, false, model.VPPIPRoute{
		VRFID:           0,
		Prefix:          "0.0.0.0/0",
		NextHops:        []string{cfg.VPP.TunDefaultGW},
		MainInterfaceID: interface_types.InterfaceIndex(cfg.VPP.MainInterfaceID),
		SubInterfaceID:  model.UndefinedSubIf,
		TunnelIDs:       []uint32{model.UndefinedTunnelID},
		FIPMPLSLabels:   []uint32{model.UndefinedLabel},
	})
	require.NoError(t, err)

	err = vpp.AddDelIPRoute(vppStream, false, model.VPPIPRoute{
		VRFID:           cfg.VRF[0].VRFID,
		Prefix:          "10.0.0.0/24",
		NextHops:        []string{cfg.VRF[0].BGPPeerIP},
		MainInterfaceID: interface_types.InterfaceIndex(cfg.VPP.MainInterfaceID),
		SubInterfaceID:  2,
		TunnelIDs:       []uint32{model.UndefinedTunnelID},
		FIPMPLSLabels:   []uint32{model.UndefinedLabel},
	})
	require.NoError(t, err)

	// testing add black-hole ip route

	err = vpp.AddBlackHoleIPRoute(vppStream, model.VPPIPRoute{
		VRFID:           cfg.VRF[0].VRFID,
		Prefix:          cfg.VRF[0].FIPPrefixes[0],
		NextHops:        []string{""},
		MainInterfaceID: interface_types.InterfaceIndex(cfg.VPP.MainInterfaceID),
		SubInterfaceID:  model.UndefinedSubIf,
		TunnelIDs:       []uint32{model.UndefinedTunnelID},
		FIPMPLSLabels:   []uint32{model.UndefinedLabel},
	})
	require.NoError(t, err)
}
