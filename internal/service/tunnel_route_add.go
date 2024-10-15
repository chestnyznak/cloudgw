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
	"git.crptech.ru/cloud/cloudgw/pkg/netutils"
)

func AddFIPAndTunnelInVPPAndStorage(
	ctx context.Context,
	stream api.Stream,
	bgpSrv *server.BgpServer,
	cfg config.Config,
	vppIPRoute model.VPPIPRoute,
	appStorage *imdb.Storage,
	calculatedVPPVRF *model.VPPVRFTable,
	calculatedBGPVRF *model.BGPVRFTable,
) {
	// check if udp tunnel already exist
	for i, nh := range vppIPRoute.NextHops {
		if appStorage.VPPUDPTunnelStorage.IsUDPTunnelExist(nh) {
			// if exists, update tunnel id for floating ip route in vppIPRoute as it unknown yet
			existedUDPTunnel := appStorage.VPPUDPTunnelStorage.GetUDPTunnel(nh)

			vppIPRoute.TunnelIDs[i] = existedUDPTunnel.TunnelID
		} else {
			// if not exist, create new udp tunnel and update tunnel id in vppIPRoute with created one
			newVPPUDPTunnel := model.NewVPPUDPTunnel(
				model.UndefinedTunnelID,
				netutils.Addr(cfg.VPP.TunLocalIP),
				nh,
				model.RandUDPTunnelSrcPort(),
			)

			if err := vpp.AddUDPTunnel(stream, &newVPPUDPTunnel); err != nil {
				logger.Error("failed to create udp tunnel in vpp", "dst ip", newVPPUDPTunnel.DstIP, "error", err)

				return
			}

			vppIPRoute.TunnelIDs[i] = newVPPUDPTunnel.TunnelID

			if err := appStorage.VPPUDPTunnelStorage.AddUDPTunnel(&newVPPUDPTunnel); err != nil {
				logger.Info("failed to create udp tunnel in memory storage", "dst ip", newVPPUDPTunnel.DstIP, "error", err)
			}
		}
	}

	// create vpp floating ip/mpls route for the floating ip in vpp and storage

	err := vpp.AddDelFIPRoute(stream, true, &vppIPRoute)
	if err != nil {
		logger.Error("failed to add floating ip route", "prefix", vppIPRoute.Prefix, "error", err)

		return
	}

	logger.Info("floating ip route created in vpp", "prefix", vppIPRoute.Prefix, "nh", vppIPRoute.NextHops[0], "label", vppIPRoute.MPLSLabels())

	if err = appStorage.VPPFIPRouteStorage.AddFIPRoute(&vppIPRoute); err != nil {
		logger.Error("failed to create floating ip route in memory storage", "fip", vppIPRoute.Prefix, "error", err)
	}

	// increment FIPServed counters

	appStorage.VPPVRFStorage.IncFIPServed(vppIPRoute.VRFID)

	for _, nh := range vppIPRoute.NextHops {
		appStorage.VPPUDPTunnelStorage.IncFIPServed(nh)
	}

	// advertise all aggregated prefixes for specific vrf to physical network vrf if at least one floating ip received from tungsten fabric

	if appStorage.VPPVRFStorage.GetFIPServed(vppIPRoute.VRFID) > 2 {
		return // already advertised
	}

	for _, fipAggrPrefix := range calculatedVPPVRF.FIPPrefixes {
		aggrFIPNLRIAttr := gobgpapi.NewBGPNLRIAttrs(
			fipAggrPrefix,
			calculatedVPPVRF.LocalAddr, // as the vpp handles the traffic
			0,
			calculatedBGPVRF.RD,
			calculatedBGPVRF.ImportRT, // because import to vrf where the rt import configured
			[]uint32{model.UndefinedLabel},
		)

		// add the aggregated floating ip prefix to bgp vpnv4 (vrf) rib (as adding ipv4 route to vrf doesn't work!)
		// gobgp sends the update only for first received vpnv4 floating ip address, so no need to suppress subsequent updates

		if err = gobgp.AdvWdrawVpnv4Prefix(
			ctx,
			bgpSrv,
			ADVERTISE,
			aggrFIPNLRIAttr,
			cfg.TFController.BGPPeerASN, // as the tungsten fabric is source of the floating ip
		); err != nil {
			logger.Error("failed to advertise vpnv4 prefix to physical network", "prefix", aggrFIPNLRIAttr.Prefix, "error", err)
		}
	}
}
