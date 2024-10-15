package test

import (
	"context"
	"fmt"
	"testing"

	bgpapi "github.com/osrg/gobgp/v3/api"
	"github.com/phayes/freeport"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/anypb"

	"git.crptech.ru/cloud/cloudgw/internal/config"
	"git.crptech.ru/cloud/cloudgw/internal/model"
	"git.crptech.ru/cloud/cloudgw/internal/repository/gobgp"
	"git.crptech.ru/cloud/cloudgw/pkg/gobgpapi"
	"git.crptech.ru/cloud/cloudgw/pkg/logger"
)

func TestGoBGPServer(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	var (
		bgpVRFTables      = make(map[uint32]*model.BGPVRFTable)
		bgpPeers          = make(map[string]*model.BGPPeer)
		cloudgwConfigPath = "config.yml"
	)

	// config

	cfg, err := config.ParseConfig(cloudgwConfigPath)
	if err != nil {
		t.Error("failed to parse yaml config file: %w", err)
	}

	// prepare initial data

	// bgp routing tables (immutable table)

	for _, vrf := range cfg.VRF {
		routeTbl := model.NewBGPVRFTable(
			vrf.VRFName,
			vrf.VRFID,
			cfg.GoBGP.BGPLocalASN,
			vrf.BGPPeerASN,
			model.RD(cfg.GoBGP.RID, vrf.VRFID),
			[]*anypb.Any{model.RT(cfg.GoBGP.BGPLocalASN, vrf.VRFID)},
			[]*anypb.Any{model.RT(cfg.TFController.BGPPeerASN, vrf.VRFID)},
		)
		bgpVRFTables[vrf.VRFID] = &routeTbl
	}

	// bgp peers (mutable table)

	// tungsten fabric controllers

	for _, ip := range cfg.TFController.Address {
		bgpPeer := model.NewBGPPeer(
			model.TF,
			cfg.TFController.BGPPeerASN,
			ip,
			179,
			"",
			true,
			10,
			"", // should be empty!
			cfg.TFController.BGPKeepAlive,
			cfg.TFController.BGPHoldTimer,
		)

		bgpPeers[ip] = &bgpPeer
	}
	// physical networks
	for _, vrf := range cfg.VRF {
		bgpPeer := model.NewBGPPeer(
			model.PHYNET,
			vrf.BGPPeerASN,
			vrf.BGPPeerIP,
			179,
			vrf.BGPPassword,
			true,
			10,
			vrf.VRFName,
			vrf.BGPKeepAlive,
			vrf.BGPHoldTimer,
		)
		bgpPeers[vrf.BGPPeerIP] = &bgpPeer
	}

	// get free ports for local gobgp server (api and bgp ports)

	localFreeGoBGPAPIPort, _ := freeport.GetFreePort()
	localFreeGoBGPBGPPort, _ := freeport.GetFreePort()

	logger.Info("got local gobgp server api port", "port", localFreeGoBGPAPIPort)
	logger.Info("got local gobgp server bgp port", "port", localFreeGoBGPBGPPort)

	// start local gobgp server

	goBGPListenAddress := ":" + fmt.Sprint(localFreeGoBGPAPIPort)

	bgpSrv := gobgp.CreateGoBGPServer(goBGPListenAddress, "info", "json", "stdout")

	// connect to the local gobgp server using api

	goBGPAPIConn, ctx, cancel := gobgpapi.ConnectToGoBGPAPI(ctx, goBGPListenAddress, 5)

	defer cancel()

	// test setting and getting BGP local config

	err = gobgp.SetGoBGPLocalConfig(
		ctx,
		bgpSrv,
		cfg.GoBGP.BGPLocalASN,
		cfg.GoBGP.RID,
		int32(localFreeGoBGPBGPPort), // important to use any available port instead YAML port
	)
	require.NoError(t, err)

	responseLocalConf, err := gobgp.GetGoBGPLocalConfig(ctx, bgpSrv)

	require.NoError(t, err)
	require.Equal(t, cfg.GoBGP.RID, responseLocalConf.Global.RouterId)
	require.Equal(t, cfg.GoBGP.BGPLocalASN, responseLocalConf.Global.Asn)

	// test adding bgp vrfs

	for _, vrf := range bgpVRFTables {
		err := gobgp.AddGoBGPVRF(ctx, bgpSrv, vrf)

		require.NoError(t, err)

		responseVRF, err := gobgpapi.GetGoBGPVRFsByAPI(ctx, goBGPAPIConn, vrf.Name)

		require.NoError(t, err)
		require.Equal(t, vrf.Name, responseVRF.Name)
	}

	// test adding bgp peer

	peerID := 0

	for _, peer := range bgpPeers {
		err = gobgp.AddBGPPeer(ctx, bgpSrv, peer)

		require.NoError(t, err)

		responsePeers, err := gobgpapi.GetGoBGPPeersByAPI(ctx, goBGPAPIConn)

		require.NoError(t, err)
		require.Contains(t, "ACTIVE IDLE", (responsePeers[peerID].GetState().SessionState).String())

		peerID++
	}

	// test advertising ipv4 prefix

	err = gobgp.AdvWdrawIPv4Prefix(ctx, bgpSrv, true, gobgpapi.BGPNLRIAttrs{
		Prefix:  "10.0.0.0/24",
		NextHop: "192.0.0.1",
	})

	require.NoError(t, err)

	// test withdraw non-existed ipv4 prefix

	err = gobgp.AdvWdrawIPv4Prefix(ctx, bgpSrv, false, gobgpapi.BGPNLRIAttrs{
		Prefix:  "10.0.0.0/24",
		NextHop: "192.0.0.1",
	})

	require.NoError(t, err)

	// test advertising vpnv4 prefix

	err = gobgp.AdvWdrawVpnv4Prefix(
		ctx,
		bgpSrv,
		true,
		gobgpapi.BGPNLRIAttrs{
			Prefix:    "10.0.0.0/24",
			NextHop:   "192.0.0.1",
			MPLSLabel: []uint32{1000},
			RD:        model.RD(cfg.GoBGP.RID, 1),
			RT:        []*anypb.Any{model.RT(cfg.TFController.BGPPeerASN, 1)},
		},
		65001,
	)

	require.NoError(t, err)

	// test withdraw vpnv4 prefix

	err = gobgp.AdvWdrawVpnv4Prefix(
		ctx,
		bgpSrv,
		false,
		gobgpapi.BGPNLRIAttrs{
			Prefix:    "10.0.0.0/24",
			NextHop:   "192.0.0.1",
			MPLSLabel: []uint32{1000},
			RD:        model.RD(cfg.GoBGP.RID, 1),
			RT:        []*anypb.Any{model.RT(cfg.TFController.BGPPeerASN, 1)},
		},
		65001,
	)

	require.NoError(t, err)

	// test gobgp policing

	// test creating prefix sets (aka prefix-list)

	allHostRoutes, err := gobgp.CreateGoBGPPrefixSet(ctx, bgpSrv, "0.0.0.0/0", 32, 32)

	require.NoError(t, err)
	require.Equal(t, "0.0.0.0/0", allHostRoutes.Prefixes[0].IpPrefix)
	require.Equal(t, uint32(32), allHostRoutes.Prefixes[0].MaskLengthMin)
	require.Equal(t, uint32(32), allHostRoutes.Prefixes[0].MaskLengthMax)

	defaultRoute, err := gobgp.CreateGoBGPPrefixSet(ctx, bgpSrv, "0.0.0.0/0", 0, 0)

	require.NoError(t, err)
	require.Equal(t, "0.0.0.0/0", defaultRoute.Prefixes[0].IpPrefix)
	require.Equal(t, uint32(0), defaultRoute.Prefixes[0].MaskLengthMin)
	require.Equal(t, uint32(0), defaultRoute.Prefixes[0].MaskLengthMax)

	allExceptDefaultRoute, err := gobgp.CreateGoBGPPrefixSet(ctx, bgpSrv, "0.0.0.0/0", 1, 32)

	require.NoError(t, err)
	require.Equal(t, "0.0.0.0/0", allExceptDefaultRoute.Prefixes[0].IpPrefix)
	require.Equal(t, uint32(1), allExceptDefaultRoute.Prefixes[0].MaskLengthMin)
	require.Equal(t, uint32(32), allExceptDefaultRoute.Prefixes[0].MaskLengthMax)

	// test creating neighbor sets

	var (
		allTFControllerIPs []string
		allPhyNetRouterIPs []string
	)

	for peerIP, peer := range bgpPeers {
		switch peer.PeerType {
		case model.TF:
			allTFControllerIPs = append(allTFControllerIPs, peerIP+"/32")
		case model.PHYNET:
			allPhyNetRouterIPs = append(allPhyNetRouterIPs, peerIP+"/32")
		}
	}

	allTFControllers, err := gobgp.CreateGoBGPNeighborSet(ctx, bgpSrv, allTFControllerIPs)

	require.NoError(t, err)
	require.ElementsMatch(
		t,
		[]string{cfg.TFController.Address[0] + "/32", cfg.TFController.Address[1] + "/32", cfg.TFController.Address[2] + "/32"},
		allTFControllers.List)

	allPhyNetRouters, err := gobgp.CreateGoBGPNeighborSet(ctx, bgpSrv, allPhyNetRouterIPs)

	require.NoError(t, err)
	require.ElementsMatch(
		t, []string{cfg.VRF[0].BGPPeerIP + "/32", cfg.VRF[1].BGPPeerIP + "/32", cfg.VRF[2].BGPPeerIP + "/32"}, allPhyNetRouters.GetList())

	// test creating policy statements

	st1, err := gobgp.CreateGoBGPNPolicyStatements(
		allHostRoutes,
		allPhyNetRouters,
		bgpapi.RouteAction_REJECT,
	)

	require.NoError(t, err)
	require.Equal(t, "REJECT", st1.GetActions().RouteAction.String())
	require.Equal(t, allHostRoutes.Name, st1.Conditions.GetPrefixSet().Name)
	require.Equal(t, allPhyNetRouters.Name, st1.Conditions.GetNeighborSet().Name)

	st2, err := gobgp.CreateGoBGPNPolicyStatements(
		defaultRoute,
		allPhyNetRouters,
		bgpapi.RouteAction_REJECT,
	)

	require.NoError(t, err)
	require.Equal(t, "REJECT", st2.GetActions().RouteAction.String())
	require.Equal(t, defaultRoute.Name, st2.Conditions.GetPrefixSet().Name)
	require.Equal(t, allPhyNetRouters.Name, st2.Conditions.GetNeighborSet().Name)

	st3, err := gobgp.CreateGoBGPNPolicyStatements(
		allExceptDefaultRoute,
		allTFControllers,
		bgpapi.RouteAction_REJECT,
	)

	require.NoError(t, err)
	require.Equal(t, "REJECT", st3.GetActions().RouteAction.String())
	require.Equal(t, allExceptDefaultRoute.Name, st3.Conditions.GetPrefixSet().Name)
	require.Equal(t, allTFControllers.Name, st3.Conditions.GetNeighborSet().Name)

	// test applying as export policy and create gobgp police

	err = gobgp.CreateGoBGPPolicy(ctx, bgpSrv, allTFControllerIPs, allPhyNetRouterIPs)

	require.NoError(t, err)

	// test deleting bgp peering

	for _, peer := range bgpPeers {
		err = gobgp.DelBGPPeer(ctx, bgpSrv, peer)

		require.NoError(t, err)
	}

	// test deleting bgp vrfs

	for _, table := range bgpVRFTables {
		err = gobgp.DelGoBGPVRF(ctx, bgpSrv, table.Name)

		require.NoError(t, err)
	}
}
