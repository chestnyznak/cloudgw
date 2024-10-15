package app

import (
	"context"
	"fmt"

	"go.fd.io/govpp/api"
	"go.fd.io/govpp/core"

	"git.crptech.ru/cloud/cloudgw/internal/repository/vpp"
	"git.crptech.ru/cloud/cloudgw/internal/repository/vpp/initialize"
	"git.crptech.ru/cloud/cloudgw/pkg/logger"
)

func initVPP(ctx context.Context, a *App) (api.Stream, func(), chan core.ConnectionEvent, error) {
	stream, disconnect, vppEvent, err := vpp.ConnectToVPPAPIAsync(ctx, a.Cfg.VPP.BinAPISock)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to connect to vpp stream api: %w", err)
	}

	version, err := vpp.GetVPPVersion(stream)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get vpp version: %w", err)
	}

	logger.Info("connected to vpp stream api", "vpp version", version)

	if err = initialize.ClearVPPConfig(stream, a.Cfg.VPP.MainInterfaceID, a.Storage.VPPVRFStorage); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to clear vpp config: %w", err)
	}

	logger.Info("vpp configuration cleared")

	if err = initialize.AddVPPInitConfig(stream, a.Storage.VPPVRFStorage, a.Cfg.VPP.MainInterfaceID, a.Cfg.VPP.TunDefaultGW); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to add vpp static config: %w", err)
	}

	logger.Info("vpp static config added")

	return stream, disconnect, vppEvent, nil
}
