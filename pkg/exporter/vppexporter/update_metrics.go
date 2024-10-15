package vppexporter

import (
	"context"
	"time"

	"go.fd.io/govpp/api"
	"go.fd.io/govpp/binapi/ip"
	"go.fd.io/govpp/core"

	"git.crptech.ru/cloud/cloudgw/internal/model"
	"git.crptech.ru/cloud/cloudgw/internal/repository/vpp"
	"git.crptech.ru/cloud/cloudgw/pkg/logger"
)

// UpdateVPPInterfaceMetrics updates the VPPInterfaceMetrics var with VPP interface metrics every [poolingInterval] sec
func UpdateVPPInterfaceMetrics(ctx context.Context, vppStatsConn *core.StatsConnection, poolingInterval int) error {
	if vppStatsConn == nil {
		return nil
	}

	stats := new(api.InterfaceStats)

	if err := vppStatsConn.GetInterfaceStats(stats); err != nil {
		logger.Error("setting vpp interface stats failed", "error", err)

		return err
	}

	// initialize VPPInterfaceMetrics with all interfaces

	for _, i := range stats.Interfaces[1:] { // skip 'local0' interface with id=0
		VPPInterfaceMetrics[i.InterfaceIndex] = NewVPPInterfaceMetric(i.InterfaceName, float64(i.InterfaceIndex))
	}

	ticker := time.NewTicker(time.Duration(poolingInterval) * time.Second)

	done := make(chan bool)

	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("vpp interface metrics update stopped")

			return nil
		case <-ticker.C:
			// collect data
			if err := vppStatsConn.GetInterfaceStats(stats); err != nil {
				logger.Error("getting vpp interface stats failed", "error", err)

				return err
			}

			for _, i := range stats.Interfaces[1:] { // skip 'local0' with id=0
				VPPInterfaceMetrics[i.InterfaceIndex].SetVPPInterfaceMetric(
					float64(i.Rx.Packets),
					float64(i.Rx.Bytes),
					float64(i.RxErrors),
					float64(i.Tx.Packets),
					float64(i.Tx.Bytes),
					float64(i.TxErrors),
					float64(i.Drops),
				)
			}
		case <-done:
			return nil
		}
	}
}

// UpdateVPPUDPVRFMetrics updates the VPPUDPTunnelMetrics and VPPVRFMetrics every [poolingInterval] sec
func UpdateVPPUDPVRFMetrics(ctx context.Context, stream *api.Stream, poolingInterval int, vppVRFTables []*model.VPPVRFTable) error {
	ticker := time.NewTicker(time.Duration(poolingInterval) * time.Second)

	done := make(chan bool)

	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("vpp udp vrf metrics update stopped")

			return nil
		case <-ticker.C:
			udpCount, err := vpp.CountUDPTunnels(*stream)
			if err != nil {
				logger.Error("failed to count udp tunnels", "error", err)
			}

			VPPUDPTunnelMetrics.SetUDPTunnelTotal(udpCount)

			for _, vrf := range vppVRFTables {
				if vrf.ID == 0 { // skip global routing table
					continue
				}

				ipRouteCount, fipRouteCount, err := vpp.CountRoutesPerTable(*stream, ip.IPTable{TableID: vrf.ID})
				if err != nil {
					logger.Error("failed to count vpp routes", "error", err)
				}

				VPPVRFMetrics[vrf.ID].SetVPPRouteTotal(ipRouteCount, fipRouteCount)
			}
		case <-done:
			return nil
		}
	}
}
