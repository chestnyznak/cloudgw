package service

import (
	"context"

	"github.com/osrg/gobgp/v3/pkg/server"
	"go.fd.io/govpp/api"

	"git.crptech.ru/cloud/cloudgw/internal/config"
	"git.crptech.ru/cloud/cloudgw/internal/model"
	"git.crptech.ru/cloud/cloudgw/internal/repository/gobgp"
	"git.crptech.ru/cloud/cloudgw/internal/repository/imdb"
	"git.crptech.ru/cloud/cloudgw/internal/repository/vpp"
	"git.crptech.ru/cloud/cloudgw/pkg/gobgpapi"
	"git.crptech.ru/cloud/cloudgw/pkg/logger"
)

func DelFIPAndTunnelFromVPPAndStorage(
	ctx context.Context,
	vppStream *api.Stream,
	bgpSrv *server.BgpServer,
	cfg config.Config,
	vppIPRoute model.VPPIPRoute,
	appStorage *imdb.Storage,
	calculatedVPPVRF *model.VPPVRFTable,
	calculatedBGPVRF *model.BGPVRFTable,
) {
	// delete floating ip route from vpp
	err := vpp.AddDelFIPRoute(*vppStream, false, &vppIPRoute)
	if err != nil {
		logger.Error("failed to delete floating ip route from vpp", "prefix", vppIPRoute.Prefix, "error", err)
	} else {
		logger.Info("floating ip route deleted from vpp", "prefix", vppIPRoute.Prefix, "nh", vppIPRoute.NextHops[0])
	}

	// delete floating ip route from VPPFIPStorage

	err = appStorage.VPPFIPRouteStorage.DelFIPRoute(vppIPRoute.Prefix)
	if err != nil {
		logger.Error("failed to delete floating ip route from memory storage", "error", err)
	}

	// decrement floating ip served counters

	appStorage.VPPVRFStorage.DecFIPServed(vppIPRoute.VRFID)

	for _, nh := range vppIPRoute.NextHops {
		appStorage.VPPUDPTunnelStorage.DecFIPServed(nh)
	}

	// delete udp tunnel if no more floating ips served by the vrouter

	for i, nh := range vppIPRoute.NextHops {
		if appStorage.VPPUDPTunnelStorage.GetFIPServed(nh) == 0 {
			// delete udp tunnel from vpp
			if err = vpp.DelUDPTunnel(*vppStream, vppIPRoute.TunnelIDs[i]); err != nil {
				logger.Error("failed to delete udp tunnel from vpp", "vrouter", nh, "error", err)
			}

			// delete udp tunnel from memory VPPUDPTunnelStorage

			if err = appStorage.VPPUDPTunnelStorage.DelUDPTunnel(nh); err != nil {
				logger.Error("failed to delete udp tunnel from storage", "error", err)
			}
		}
	}

	// if it was last floating records in whole vrf, then withdraw the all aggregated prefixes from bgp table (from physical network) for specific vrf (floating ip /32 prefix always withdraw themselves)

	if appStorage.VPPVRFStorage.GetFIPServed(vppIPRoute.VRFID) == 0 {
		for _, fipAggrPrefix := range calculatedVPPVRF.FIPPrefixes {
			// create bgp nlri attributes structure for selected aggregated floating ip
			aggrFIPNLRIAttr := gobgpapi.NewBGPNLRIAttrs(
				fipAggrPrefix,
				calculatedVPPVRF.LocalAddr, // as the vpp handles the traffic
				0,                          // as vpnv4 belongs to vrf 0
				calculatedBGPVRF.RD,
				calculatedBGPVRF.ImportRT,
				[]uint32{model.UndefinedLabel},
			)

			if err = gobgp.AdvWdrawVpnv4Prefix(
				ctx,
				bgpSrv,
				WITHDRAW,
				aggrFIPNLRIAttr,
				cfg.TFController.BGPPeerASN,
			); err != nil {
				logger.Error("failed to withdraw vpnv4 prefix", "prefix", fipAggrPrefix, "error", err)
			}
		}
	}
}
