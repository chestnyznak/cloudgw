package app

import (
	"context"

	"git.crptech.ru/cloud/cloudgw/pkg/exporter/gobgpexporter"
	"git.crptech.ru/cloud/cloudgw/pkg/exporter/vppexporter"
	"git.crptech.ru/cloud/cloudgw/pkg/logger"
)

func initMetric(ctx context.Context, a *App) {
	vppPollTimer := a.Cfg.VPP.MetricPollingInterval
	gobgpPollTimer := a.Cfg.GoBGP.MetricPollingInterval

	vppVRFs := a.Storage.VPPVRFStorage.GetVRFs()

	for _, vrf := range vppVRFs {
		if vrf.ID == 0 {
			continue
		}

		vppexporter.VPPVRFMetrics[vrf.ID] = vppexporter.NewVPPVRFMetric(vrf.Name)
	}

	// update metrics

	bgpPeers := a.Storage.BGPPeerStorage.GetBGPPeers()

	go func() {
		if err := gobgpexporter.UpdateGoBGPMetrics(ctx, a.BGPServer, gobgpPollTimer, bgpPeers); err != nil {
			logger.Error("failed to update gobgp metrics", "error", err)
		}
	}()

	go func() {
		if err := vppexporter.UpdateVPPInterfaceMetrics(ctx, a.VPPStats, vppPollTimer); err != nil {
			logger.Error("failed to update vpp interface metrics", "error", err)
		}
	}()

	go func() {
		if err := vppexporter.UpdateVPPUDPVRFMetrics(ctx, a.VPPStream, vppPollTimer, vppVRFs); err != nil {
			logger.Error("failed to update vpp tunnel and route metrics", "error", err)
		}
	}()
}
