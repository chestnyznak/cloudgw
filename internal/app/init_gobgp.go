package app

import (
	"context"
	"fmt"

	"github.com/osrg/gobgp/v3/pkg/server"

	"git.crptech.ru/cloud/cloudgw/internal/config"
	"git.crptech.ru/cloud/cloudgw/internal/model"
	"git.crptech.ru/cloud/cloudgw/internal/repository/gobgp"
	"git.crptech.ru/cloud/cloudgw/internal/repository/imdb"
	"git.crptech.ru/cloud/cloudgw/pkg/exporter/gobgpexporter"
	"git.crptech.ru/cloud/cloudgw/pkg/logger"
)

func initGoBGPServer(ctx context.Context, cfg *config.Config) (*server.BgpServer, error) {
	bgpSrv := gobgp.CreateGoBGPServer(
		cfg.GoBGP.GRPCListenAddress,
		cfg.Logging.Level,
		cfg.Logging.Format,
		cfg.Logging.Output,
	)

	if err := gobgp.SetGoBGPLocalConfig(
		ctx, bgpSrv,
		cfg.GoBGP.BGPLocalASN,
		cfg.GoBGP.RID,
		cfg.GoBGP.BGPLocalPort,
	); err != nil {
		return nil, fmt.Errorf("failed to create local bgp configuration: %w", err)
	}

	return bgpSrv, nil
}

func ConfigureGoBGP(ctx context.Context, storage *imdb.Storage, bgpSrv *server.BgpServer) (func(), error) {
	gobgpVRFs := storage.BGPVRFStorage.GetVRFs()

	for _, vrf := range gobgpVRFs {
		if err := gobgp.AddGoBGPVRF(ctx, bgpSrv, vrf); err != nil {
			return nil, fmt.Errorf("failed to create bgp vrf %q: %w", vrf.Name, err)
		}

		gobgpexporter.GoBGPGeneralMetrics.IncVRFCount()
	}

	// add bgp peers (tungsten fabric and physical network) and update metrics

	bgpPeers := storage.BGPPeerStorage.GetBGPPeers()

	gobgpexporter.GoBGPGeneralMetrics.PeerCount = int32(len(bgpPeers))

	for _, peer := range bgpPeers {
		if err := gobgp.AddBGPPeer(ctx, bgpSrv, peer); err != nil {
			return nil, fmt.Errorf("failed to create bgp peer %s: %w", peer.PeerAddress, err)
		}
	}

	// delete bgp peer before shutdown to avoid traffic black-holing

	deletePeers := func() {
		for _, peer := range bgpPeers {
			_ = gobgp.DelBGPPeer(ctx, bgpSrv, peer)
		}
	}

	// bgp policy

	var (
		tfServerIPs []string
		phyNetIPs   []string
	)

	for _, peer := range bgpPeers {
		switch peer.PeerType {
		case model.TF:
			tfServerIPs = append(tfServerIPs, peer.PeerAddress+"/32")
		case model.PHYNET:
			phyNetIPs = append(phyNetIPs, peer.PeerAddress+"/32")
		}
	}

	if err := gobgp.CreateGoBGPPolicy(ctx, bgpSrv, tfServerIPs, phyNetIPs); err != nil {
		logger.Error("failed to create bgp policy", "error", err)
	}

	return deletePeers, nil
}
