package app

import (
	"context"
	"os"
	"sync"

	"go.fd.io/govpp/adapter/statsclient"
	vppapi "go.fd.io/govpp/api"
	"go.fd.io/govpp/core"

	"git.crptech.ru/cloud/cloudgw/internal/config"
	"git.crptech.ru/cloud/cloudgw/internal/monitor"
	"git.crptech.ru/cloud/cloudgw/internal/repository/imdb"
	"git.crptech.ru/cloud/cloudgw/internal/service"
	"git.crptech.ru/cloud/cloudgw/pkg/closer"
	"git.crptech.ru/cloud/cloudgw/pkg/logger"
	"git.crptech.ru/cloud/cloudgw/pkg/pyroscope"
	"github.com/osrg/gobgp/v3/pkg/server"
)

type App struct {
	Cfg       *config.Config
	Storage   *imdb.Storage
	BGPServer *server.BgpServer
	VPPStream *vppapi.Stream
	VPPEvent  chan core.ConnectionEvent
	VPPStats  *core.StatsConnection
}

func Init(ctx context.Context) *App {
	a := App{}

	var err error

	// config

	configPath := os.Getenv("CLOUDGW_CONFIG_PATH")

	if configPath == "" {
		configPath = "/etc/cloudgw/config.yml"
	}

	a.Cfg, err = config.ParseConfig(configPath)
	if err != nil {
		logger.Fatal("failed to parse config file", "file path", configPath, "error", err)
	}

	logger.Info("config file parsed successfully", "file", configPath)

	// global logger

	logger.Init(
		logger.WithLevel(a.Cfg.Logging.Level),
		logger.WithFormat(a.Cfg.Logging.Format),
		logger.WithOutput(a.Cfg.Logging.Output),
	)

	logger.Info("global logger initialized successfully", "level", a.Cfg.Logging.Level)

	// storages

	a.Storage, err = initStorages(a.Cfg)
	if err != nil {
		logger.Fatal("failed to initialize storages", "error", err)
	}

	// vpp bin and stats

	stream, vppDisconnect, vppEvent, err := initVPP(ctx, &a)
	if err != nil {
		logger.Fatal("failed to initialize vpp stream connection", "error", err)
	}

	a.VPPStream = &stream
	a.VPPEvent = vppEvent

	closer.Add(func() error {
		vppDisconnect()
		logger.Info("vpp stream api disconnecting")

		return nil
	})

	if a.Cfg.HTTP.Enable {
		vppStatsCli := statsclient.NewStatsClient("/run/vpp/stats.sock")

		vppStats, err := core.ConnectStats(vppStatsCli)
		if err != nil {
			logger.Error("failed to connect to vpp stats connection", "error", err)
		}

		a.VPPStats = vppStats

		closer.Add(func() error {
			vppStats.Disconnect()
			logger.Info("vpp stats api disconnecting")

			return nil
		})
	}

	// gobgp

	a.BGPServer, err = initGoBGPServer(ctx, a.Cfg)
	if err != nil {
		logger.Fatal("failed to initialize gobgp server", "error", err)
	}

	closer.Add(func() error {
		a.BGPServer.Stop()
		logger.Info("gobgp server shutting down")

		return nil
	})

	return &a
}

func Run(ctx context.Context, a *App) {
	defer func() {
		closer.CloseAll()
		closer.Wait()
	}()

	ctx, cancel := context.WithCancel(ctx)

	closer.Add(func() error {
		cancel()
		logger.Info("main context canceling")

		return nil
	})

	// pyroscope profiling

	if a.Cfg.Pyroscope.Enable {
		hostname, _ := os.Hostname()

		if err := pyroscope.EnablePyroscopeProfiling("cloudgw", hostname, a.Cfg.Pyroscope.URL); err != nil {
			logger.Error("failed to initiate pyroscope profiling", "error", err)
		}
	}

	// watch and handle bgp events from gobgp. NOTE: start watching before bgp peering to install routes correctly!

	service.HandleBGPUpdate(ctx, a.VPPStream, a.BGPServer, *a.Cfg, a.Storage)

	// gobgp peers

	deletePeers, err := ConfigureGoBGP(ctx, a.Storage, a.BGPServer)
	if err != nil {
		logger.Fatal(err.Error())
	}

	closer.Add(func() error {
		deletePeers()
		logger.Info("bgp peers deleting")

		return nil
	})

	// metrics

	if a.Cfg.HTTP.Enable {
		go initMetric(ctx, a)
	}

	// http server and prometheus metrics exposing

	if a.Cfg.HTTP.Enable {
		go initHTTPServer(ctx, a)
	}

	// monitor vpp main interface status

	bgpPeers := a.Storage.BGPPeerStorage.GetBGPPeers()

	if err = monitor.VPPInterfaceStatus(ctx, a.Cfg, bgpPeers, a.BGPServer); err != nil {
		logger.Error("vpp interface monitoring is not started, monitoring will be disabled", "error", err)
	}

	// monitor vpp connection status and stop the app if the connection failed

	var wg sync.WaitGroup

	wg.Add(1)

	go monitor.VPPConnStatus(ctx, a.VPPEvent, &wg)

	wg.Wait()
}
