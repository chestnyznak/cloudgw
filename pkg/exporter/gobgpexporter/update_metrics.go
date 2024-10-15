package gobgpexporter

import (
	"context"
	"time"

	bgpapi "github.com/osrg/gobgp/v3/api"
	"github.com/osrg/gobgp/v3/pkg/server"

	"git.crptech.ru/cloud/cloudgw/internal/model"
	"git.crptech.ru/cloud/cloudgw/internal/repository/gobgp"
	"git.crptech.ru/cloud/cloudgw/pkg/logger"
)

// UpdateGoBGPMetrics updates the GoBGPPerPeerMetrics var with GoBGP metrics every [poolingInterval] sec
func UpdateGoBGPMetrics(ctx context.Context, bgpSrv *server.BgpServer, poolingInterval int, bgpPeers []*model.BGPPeer) error {
	// Initialize GoBGPPerPeerMetrics map with all VRFs
	for _, peer := range bgpPeers {
		GoBGPPerPeerMetrics[peer.PeerAddress] = NewGoBGPPerPeerMetric()
	}

	ticker := time.NewTicker(time.Duration(poolingInterval) * time.Second)

	done := make(chan bool)

	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("gobgp metrics update stopped")

			return nil
		case <-ticker.C:
			for _, peer := range bgpPeers {
				var safi bgpapi.Family_Safi

				if peer.PeerType == model.TF {
					safi = bgpapi.Family_SAFI_MPLS_VPN
				} else {
					safi = bgpapi.Family_SAFI_UNICAST
				}

				outputAdjIn, err := gobgp.UpdateGoBGPPerPeerMetrics(
					ctx,
					bgpSrv,
					bgpapi.TableType_ADJ_IN,
					bgpapi.Family_AFI_IP,
					safi,
					peer.PeerAddress,
				)
				if err != nil {
					return err
				}

				outputAdjOut, err := gobgp.UpdateGoBGPPerPeerMetrics(
					ctx,
					bgpSrv,
					bgpapi.TableType_ADJ_OUT,
					bgpapi.Family_AFI_IP,
					safi,
					peer.PeerAddress,
				)
				if err != nil {
					return err
				}

				if peer.PeerType == model.TF {
					GoBGPPerPeerMetrics[peer.PeerAddress].SetVpnv4UpdateRcvd(outputAdjIn)
					GoBGPPerPeerMetrics[peer.PeerAddress].SetVpnv4UpdateSent(outputAdjOut)
				} else {
					GoBGPPerPeerMetrics[peer.PeerAddress].SetIPv4UpdateRcvd(outputAdjIn)
					GoBGPPerPeerMetrics[peer.PeerAddress].SetIPv4UpdateSent(outputAdjOut)
				}
			}
		case <-done:
			return nil
		}
	}
}
