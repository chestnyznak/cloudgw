package monitor

import (
	"context"
	"sync"

	"go.fd.io/govpp/core"

	"git.crptech.ru/cloud/cloudgw/pkg/logger"
)

// VPPConnStatus checks if Bin API VPP connection status changed from "Connected" to "Disconnected"/"Failed".
func VPPConnStatus(ctx context.Context, vppEvt chan core.ConnectionEvent, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			logger.Info("closed context detected, stopping vpp api monitoring")

			return
		case currState := <-vppEvt:
			if currState.State.String() == "Disconnected" || currState.State.String() == "Failed" {
				logger.Error("vpp api connection status changed", "current state", currState.State)

				return
			}
		}
	}
}
