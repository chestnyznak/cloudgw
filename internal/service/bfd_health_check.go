package service

import (
	"context"
	"strings"
	"syscall"

	"git.crptech.ru/cloud/cloudgw/internal/model"
	"git.crptech.ru/cloud/cloudgw/pkg/bfd"
	"git.crptech.ru/cloud/cloudgw/pkg/closer"
	"git.crptech.ru/cloud/cloudgw/pkg/logger"
)

// CheckBFDPeerStatus starts monitoring a peer by BFD, exit from program when BFD peer moved UP > DOWN
func CheckBFDPeerStatus(ctx context.Context, bgpPeer model.BGPPeer) {
	chBFDDone := make(chan struct{})

	control := bfd.NewControl(ctx, bgpPeer.BFDPeering.BFDLocalIP, syscall.AF_INET)

	control.AddSession(
		bgpPeer.BFDPeering.BFDPeerIP,
		false,
		bgpPeer.BFDPeering.BFDRxMin,
		bgpPeer.BFDPeering.BFDTxRate,
		bgpPeer.BFDPeering.BFDMultiplier,
		callbackBFDState,
		chBFDDone,
	)

	closer.Add(func() error {
		logger.Info("bfd session disconnecting", "peer ip", bgpPeer.BFDPeering.BFDPeerIP)

		return control.DelSession(bgpPeer.BFDPeering.BFDPeerIP)
	})

	select {
	case <-ctx.Done():
		logger.Info("closed context in bfd process detected, starting deleting sessions and exit")

		return
	case <-chBFDDone:
		logger.Info("closed channel from bfd peer detected, starting deleting bgp sessions and force exit", "peer", bgpPeer.BFDPeering.BFDPeerIP)
	}

	// close all and force exit

	closer.CloseAll()
	closer.Wait()

	logger.Fatal("exiting the app due to bfd peer failed", "peer ip", bgpPeer.BFDPeering.BFDPeerIP)
}

type bfdPeerState int

func (b bfdPeerState) String() string {
	return [...]string{"ADMINDOWN", "DOWN", "INIT", "UP"}[b]
}

// callbackBFDState shows BFD peer state when BGP peer state changed (you can use it for monitoring or event handling)
func callbackBFDState(ipAddr string, prevState, currState int) {
	logger.Info(
		"bfd state changed",
		"peer", ipAddr,
		"previous state", strings.ToLower(bfdPeerState(prevState).String()),
		"current state", strings.ToLower(bfdPeerState(currState).String()),
	)
}
