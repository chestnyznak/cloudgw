package app

import (
	"fmt"

	"go.fd.io/govpp/binapi/interface_types"
	"google.golang.org/protobuf/types/known/anypb"

	"git.crptech.ru/cloud/cloudgw/internal/config"
	"git.crptech.ru/cloud/cloudgw/internal/model"
	"git.crptech.ru/cloud/cloudgw/internal/repository/imdb"
	"git.crptech.ru/cloud/cloudgw/pkg/netutils"
)

func initStorages(cfg *config.Config) (*imdb.Storage, error) {
	BGPPeerStorage, err := initBGPPeerStorage(cfg)
	if err != nil {
		return nil, err
	}

	BGPVRFStorage, err := initBGPVRFStorage(cfg)
	if err != nil {
		return nil, err
	}

	VPPVRFStorage, err := initVPPVRFStorage(cfg)
	if err != nil {
		return nil, err
	}

	VPPFIPRouteStorage := imdb.NewVPPFIPRouteStorage()

	VPPUDPTunnelStorage := imdb.NewVPPUDPTunnelStorage()

	storage := imdb.Storage{
		BGPPeerStorage:      BGPPeerStorage,
		BGPVRFStorage:       BGPVRFStorage,
		VPPVRFStorage:       VPPVRFStorage,
		VPPFIPRouteStorage:  VPPFIPRouteStorage,
		VPPUDPTunnelStorage: VPPUDPTunnelStorage,
	}

	return &storage, nil
}

func initBGPPeerStorage(cfg *config.Config) (*imdb.BGPPeerStorage, error) {
	peerStorage := imdb.NewBGPPeerStorage()

	// tungsten fabric controllers
	for _, ip := range cfg.TFController.Address {
		bgpPeer := model.NewBGPPeer(
			model.TF,
			cfg.TFController.BGPPeerASN,
			ip,
			179,
			"",
			true,
			cfg.TFController.BGPTTL,
			"", // should be empty!
			cfg.TFController.BGPKeepAlive,
			cfg.TFController.BGPHoldTimer,
		)

		if err := peerStorage.AddBGPPeer(&bgpPeer); err != nil {
			return nil, fmt.Errorf("failed to add bgp peer %s: %w", ip, err)
		}
	}

	// physical network
	for _, vrf := range cfg.VRF {
		bgpPeer := model.NewBGPPeer(
			model.PHYNET,
			vrf.BGPPeerASN,
			vrf.BGPPeerIP,
			179,
			vrf.BGPPassword,
			true,
			vrf.BGPTTL,
			vrf.VRFName,
			vrf.BGPKeepAlive,
			vrf.BGPHoldTimer,
		)

		bfdPeering := model.NewBFDPeer(
			vrf.BFDEnable,
			vrf.BGPPeerIP,
			vrf.BFDLocalIP,
			vrf.BFDTxRate,
			vrf.BFDRxMin,
			vrf.BFDMultiplier,
		)

		bgpPeer.BFDPeering = &bfdPeering

		if err := peerStorage.AddBGPPeer(&bgpPeer); err != nil {
			return nil, fmt.Errorf("failed to add bgp peer ip %s: %w", vrf.BGPPeerIP, err)
		}
	}

	bgpPeers := peerStorage.GetBGPPeers()

	if len(bgpPeers) < 2 {
		return nil, fmt.Errorf("found %d bgp peers (needed at least 2)", len(bgpPeers))
	}

	return peerStorage, nil
}

func initBGPVRFStorage(cfg *config.Config) (*imdb.BGPVRFStorage, error) {
	bgpVrfStorage := imdb.NewBGPVRFStorage() // routing tables without grt

	for _, vrf := range cfg.VRF {
		vrfTbl := model.NewBGPVRFTable(
			vrf.VRFName,
			vrf.VRFID,
			cfg.GoBGP.BGPLocalASN,
			vrf.BGPPeerASN,
			model.RD(cfg.GoBGP.RID, vrf.VRFID),
			[]*anypb.Any{model.RT(
				cfg.TFController.BGPPeerASN,
				vrf.VRFID,
			)},
			[]*anypb.Any{model.RT(
				cfg.TFController.BGPPeerASN,
				vrf.VRFID,
			)},
		)

		if err := bgpVrfStorage.AddVRF(&vrfTbl); err != nil {
			return nil, fmt.Errorf("failed to add bgp vrf %s: %w", vrf.VRFName, err)
		}
	}

	bgpVRFs := bgpVrfStorage.GetVRFs()

	if len(bgpVRFs) < 1 {
		return nil, fmt.Errorf("found %d bgp vrfs (needed at least 1)", len(bgpVRFs))
	}

	return bgpVrfStorage, nil
}

func initVPPVRFStorage(cfg *config.Config) (*imdb.VPPVRFStorage, error) {
	VPPVRFStorage := imdb.NewVPPVRFStorage() // routing tables with grt

	// global routing table (id = 0)
	VPPRoutingTbl := model.NewVPPVRFTable(
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

	if err := VPPVRFStorage.AddVRF(&VPPRoutingTbl); err != nil {
		return nil, fmt.Errorf("failed to add vpp vrf id %d: %w", VPPRoutingTbl.ID, err)
	}

	// vrfs (id = 1, ...)
	for _, vrf := range cfg.VRF {
		mplsLocalLabel, err := netutils.MPLSLabel(vrf.LocalIP)
		if err != nil {
			return nil, fmt.Errorf("failed to create mpls local label: %w", err)
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

		if err = VPPVRFStorage.AddVRF(&vppRoutingTbl); err != nil {
			return nil, fmt.Errorf("failed to add vpp vrf id %d: %w", VPPRoutingTbl.ID, err)
		}
	}

	if len(VPPVRFStorage.GetVRFs()) < 2 {
		return nil, fmt.Errorf("found %d vrf(s) in storage (need at least 2)", len(VPPVRFStorage.GetVRFs()))
	}

	return VPPVRFStorage, nil
}
