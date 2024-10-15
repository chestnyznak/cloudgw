// // Testing the correct operation of the clearing (ClearVPPConfig) and static configuration (addVPPInitConfig) functions
// // Need to manually synchronize the content of the function in the vpp_add_config.go and here
package vpp

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.fd.io/govpp/binapi/interface_types"

	"git.crptech.ru/cloud/cloudgw/internal/config"
	"git.crptech.ru/cloud/cloudgw/internal/model"
	"git.crptech.ru/cloud/cloudgw/internal/repository/vpp"
	"git.crptech.ru/cloud/cloudgw/pkg/logger"
	"git.crptech.ru/cloud/cloudgw/pkg/netutils"
	"git.crptech.ru/cloud/cloudgw/pkg/testcontainers"
	"git.crptech.ru/cloud/cloudgw/pkg/unixsockrelay"
)

func TestVPPStaticFunc(t *testing.T) {
	t.Parallel()

	var (
		vppAPITCPPort = "4445" // should be different from vpp_test container
		vppVRFTables  = make(map[uint32]*model.VPPVRFTable)
	)

	// initial shell command to add interfaces for VPP container and start unix sock listening to tcp relay

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

	// startup config

	cfg, err := config.ParseConfig(cloudgwConfigPath)
	if err != nil {
		logger.Fatal("failed to parse yaml config file", "error", err)
	}

	// prepare vpp routing tables and interfaces table

	// VPPVRFTable[0] is global routing table

	vppRoutingTbl := model.NewVPPVRFTable(
		"ipv4-VRF:0",
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
			model.UndefinedSubIf, // will be defined later when creating sub-interfaces
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

	vpp2Cont, err := testcontainers.CreateVppContainer(ctx, vppAPITCPPort)
	if err != nil {
		logger.Error("failed to create vpp container", "error", err)
		os.Exit(1)
	}

	defer vpp2Cont.DeleteVppContainer(ctx)

	time.Sleep(time.Second * 1)

	// run vp setup shell commands to prepare the VPP container

	for _, command := range vppStartupShellCmds {
		vpp2Cont.ExecCmd(ctx, []string{"/bin/sh", "-c", command})
		time.Sleep(time.Millisecond * 100)
	}

	time.Sleep(time.Second * 1)

	// to avoid interference with another container

	vppAPISockFile := vppAPITCPPort + "_" + cfg.VPP.BinAPISock

	// create local unix socket file on test host to communicate with the VPP container

	logger.Info("creating unix socket file")

	go unixsockrelay.UnixToTCPRelay(
		vppAPISockFile,
		vpp2Cont.GetIP(ctx),
		vppAPITCPPort)

	defer func() {
		_ = os.Remove(vppAPISockFile)
	}()

	time.Sleep(time.Second * 1)

	// testing connect to vp stream api through local host unix sock file

	vppStream, disconnect, _, err := vpp.ConnectToVPPAPIAsync(ctx, vppAPISockFile)

	defer disconnect()

	require.NoError(t, err)

	// testing getting vpp version

	version, err := vpp.GetVPPVersion(vppStream)

	require.NoError(t, err)
	require.Contains(t, version, testcontainers.VPPVersion)

	// ClearVPPConfig function with full empty vpp (cold start)

	// delete existing non-host ipv4 routes for all vrf (needed to correctly delete mpls local route)

	dumpedIPRoutes, err := vpp.DumpIPRoutes(vppStream)

	require.NoError(t, err)
	require.Equal(t, len(dumpedIPRoutes), 0)

	for _, route := range dumpedIPRoutes {
		err = vpp.AddDelIPRoute(vppStream, false, route)

		require.NoError(t, err)
	}

	// delete MPLS local-label routes for non-default VRF if they exists

	for _, table := range vppVRFTables {
		if table.ID == 0 {
			continue
		}

		err = vpp.AddDelMPLSLocalLabelRoute(vppStream, false, *table)
		require.Contains(t, err.Error(), "No such FIB / VRF") // VRF not exist yet
	}

	// delete all sub-interfaces

	numDeletedSubInterfaces, err := vpp.DelSubInterfaces(vppStream, cfg.VPP.MainInterfaceID)

	require.NoError(t, err)
	require.Equal(t, uint32(0), numDeletedSubInterfaces) // No sub-interface configured yet

	// get all existing udp tunnel ids

	dumpedUDPTunnels, err := vpp.DumpUDPTunnels(vppStream)

	require.NoError(t, err)
	require.Equal(t, 0, len(dumpedUDPTunnels))

	// get all existing ip/mpls routes to vm floating ip addresses

	dumpedFIPRoutes, err := vpp.DumpFIPRoutes(vppStream)

	require.NoError(t, err)
	require.Equal(t, 0, len(dumpedFIPRoutes))

	// unconditionally delete all existing matching routes to fips and udp tunnels
	// skipped as both dumpedUDPTunnels and dumpedFIPRoutes are zero

	// remove all "orphan" udp tunnel id records (records with no routes were found)
	// skipped as the dumpedUDPTunnels is already zero

	// reset vpp main interface configuration (delete IP address, disable)

	err = vpp.ResetMainInterface(vppStream, cfg.VPP.MainInterfaceID)

	require.NoError(t, err)

	// delete all vpp vrfs except default vrf

	for _, table := range vppVRFTables {
		if table.ID == 0 {
			continue
		}

		err = vpp.AddDelVRF(vppStream, false, *table)

		require.NoError(t, err)
	}

	// AddVPPInitConfig function

	// create vpp vrfs

	for _, table := range vppVRFTables {
		if table.ID == 0 {
			continue
		}

		err = vpp.AddDelVRF(vppStream, true, *table)

		require.NoError(t, err)
	}

	// create vpp mpls table

	err = vpp.AddDelMPLSTable(vppStream, true)

	require.NoError(t, err)

	// configure vpp main interface (ip address, enable, mpls support)

	err = vpp.SetupMainInterface(vppStream, *vppVRFTables[0])

	require.NoError(t, err)

	// create sub-interface to physical network for all vrf (except global routing table)

	for _, table := range vppVRFTables {
		if table.ID == 0 {
			continue
		}

		subIfID, err := vpp.AddSubInterface(vppStream, table)

		require.NoError(t, err)
		require.GreaterOrEqual(t, int(subIfID), 2)
	}

	// register mpls over udp decapsulation port 6635

	err = vpp.AddUDPDecap(vppStream)

	require.NoError(t, err)

	// create mpls local-label route to accept labeled traffic from vrouters

	for _, table := range vppVRFTables {
		if table.ID == 0 {
			continue
		}

		err = vpp.AddDelMPLSLocalLabelRoute(vppStream, true, *table)

		require.NoError(t, err)
	}

	// create default ipv4 route to vrouters from vpp global routing tables

	err = vpp.AddDelIPRoute(
		vppStream,
		true,
		model.VPPIPRoute{
			VRFID:           0,
			Prefix:          "0.0.0.0/0",
			NextHops:        []string{cfg.VPP.TunDefaultGW},
			MainInterfaceID: interface_types.InterfaceIndex(cfg.VPP.MainInterfaceID),
			SubInterfaceID:  model.UndefinedSubIf,
			TunnelIDs:       []uint32{model.UndefinedTunnelID},
			FIPMPLSLabels:   []uint32{model.UndefinedLabel},
		},
	)

	require.NoError(t, err)

	// create floating ip aggregated routes (used for blackHole routes)

	var vppAggrFIPRoutes []model.VPPIPRoute

	for _, vrf := range cfg.VRF {
		for _, prefix := range vrf.FIPPrefixes {
			mplsLocalLabel, err := netutils.MPLSLabel(vrf.LocalIP)

			require.NoError(t, err)

			vppAggrFIPRoute := model.NewVPPIPRoute(
				vrf.VRFID,
				interface_types.InterfaceIndex(cfg.VPP.MainInterfaceID),
				model.UndefinedSubIf,
				prefix,
				[]string{""},
				[]uint32{model.UndefinedTunnelID},
				[]uint32{mplsLocalLabel},
			)

			vppAggrFIPRoutes = append(vppAggrFIPRoutes, vppAggrFIPRoute)
		}
	}

	// add blackHole routes

	for _, route := range vppAggrFIPRoutes {
		err = vpp.AddBlackHoleIPRoute(vppStream, route)
		require.NoError(t, err)
	}

	// Testing ClearVPPConfig function (hot start/restart)

	// delete existing non-host IPv4 routes for all vrf (needed to correctly delete mpls local route)

	dumpedIPRoutes, err = vpp.DumpIPRoutes(vppStream)

	require.NoError(t, err)
	require.Greater(t, len(dumpedIPRoutes), 0)

	for _, route := range dumpedIPRoutes {
		err = vpp.AddDelIPRoute(vppStream, false, route)

		require.NoError(t, err)
	}

	// delete mpls local-label routes for non-default vrf if they exist

	for _, table := range vppVRFTables {
		if table.ID == 0 {
			continue
		}

		err = vpp.AddDelMPLSLocalLabelRoute(vppStream, false, *table)

		require.NoError(t, err)
	}

	// delete all sub-interfaces

	numDeletedSubInterfaces, err = vpp.DelSubInterfaces(vppStream, cfg.VPP.MainInterfaceID)

	require.NoError(t, err)
	require.Greater(t, numDeletedSubInterfaces, uint32(0))

	// get all existing udp tunnel ids

	dumpedUDPTunnels, err = vpp.DumpUDPTunnels(vppStream)

	require.NoError(t, err)
	require.Equal(t, 0, len(dumpedUDPTunnels))

	// get all existing ip/mpls routes to vm floating ip addresses

	dumpedFIPRoutes, err = vpp.DumpFIPRoutes(vppStream)

	require.NoError(t, err)
	require.Equal(t, 0, len(dumpedFIPRoutes))

	// unconditionally delete all existing matching routes to floating ips and UDP tunnels

	for _, routeRecord := range dumpedFIPRoutes {
		for _, udpRecord := range dumpedUDPTunnels {
			if contains(routeRecord.TunnelIDs, udpRecord.TunnelID) {
				err = vpp.AddDelFIPRoute(vppStream, false, &routeRecord) //nolint:gosec
				require.NoError(t, err)
				err = vpp.DelUDPTunnel(vppStream, udpRecord.TunnelID)
				require.NoError(t, err)
			}
		}
	}
	// remove all "single" udp Tunnel id records (records with no routes were found)

	dumpedUDPTunnels, err = vpp.DumpUDPTunnels(vppStream)

	require.NoError(t, err)
	require.Equal(t, 0, len(dumpedUDPTunnels))

	// reset vpp main interface configuration (delete ip address, disable)

	err = vpp.ResetMainInterface(vppStream, cfg.VPP.MainInterfaceID)

	require.NoError(t, err)

	// delete all vpp vrfs except default vrf

	for _, table := range vppVRFTables {
		if table.ID == 0 {
			continue
		}

		err = vpp.AddDelVRF(vppStream, false, *table)
		require.NoError(t, err)
	}
}

func contains(s []uint32, e uint32) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}

	return false
}
