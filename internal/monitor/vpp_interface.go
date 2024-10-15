package monitor

import (
	"context"

	"github.com/osrg/gobgp/v3/pkg/server"

	"git.crptech.ru/cloud/cloudgw/internal/config"
	"git.crptech.ru/cloud/cloudgw/internal/model"
	"git.crptech.ru/cloud/cloudgw/internal/repository/gobgp"
	"git.crptech.ru/cloud/cloudgw/pkg/logger"
	"git.crptech.ru/cloud/cloudgw/pkg/netutils"
)

func VPPInterfaceStatus(ctx context.Context, cfg *config.Config, bgpPeers []*model.BGPPeer, bgpSrv *server.BgpServer) error {
	if cfg.VPP.InterfaceMonitorEnable {
		monitoredAddress := []string{netutils.Addr(cfg.VPP.TunLocalIP)}

		go ProbeVPPInterface(ctx, bgpSrv, bgpPeers, monitoredAddress, 2, 10)

		logger.Info("vpp main interface monitoring started", "monitored addresses", monitoredAddress)
	}

	return nil
}

// ProbeVPPInterface checks vpp interface(s) availability and exit from the program if any ip address of vpp are not available
func ProbeVPPInterface(
	ctx context.Context,
	bgpSrv *server.BgpServer,
	bgpPeers []*model.BGPPeer,
	monitoredIPAddress []string,
	pingDuration uint,
	maxFailCount int,
) {
	for {
		for _, address := range monitoredIPAddress {
			ok, err := netutils.IsAddressAlive(address, pingDuration)
			if err != nil {
				logger.Error("failed to probe ip address", "address", address, "error", err)

				break
			}

			if !ok { //nolint:nestif
				failCount := 1
				// counting failed attempts
				for i := 0; i < maxFailCount; i++ {
					ok, err := netutils.IsAddressAlive(address, pingDuration)
					if err != nil {
						logger.Error("failed to check is address alive", "address", address, "error", err)

						break
					}

					if ok {
						logger.Error(
							"ip address is dead less then deadline interval, skip and continue",
							"address", address,
							"fail count", failCount,
							"deadline time", int(pingDuration)*maxFailCount,
						)

						break
					}

					failCount++
				}

				// stop app if limit exceeded

				if failCount >= maxFailCount {
					logger.Error(
						"ip address is dead more then deadline interval",
						"address", address,
						"fail count", failCount,
						"deadline time", int(pingDuration)*maxFailCount,
					)

					for _, peer := range bgpPeers {
						if err := gobgp.DelBGPPeer(ctx, bgpSrv, peer); err != nil {
							logger.Info(
								"failed to delete bgp peer",
								"peer address", peer.PeerAddress,
								"error", err,
							)
						}
					}

					logger.Fatal("exit from app due to vpp interface monitoring failed")
				}
			}
		}
	}
}
