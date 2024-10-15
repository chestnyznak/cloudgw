package app

import (
	"context"
	"errors"
	"net/http"

	controller "git.crptech.ru/cloud/cloudgw/internal/controller/http"
	"git.crptech.ru/cloud/cloudgw/pkg/closer"
	"git.crptech.ru/cloud/cloudgw/pkg/logger"
)

func initHTTPServer(ctx context.Context, a *App) {
	engine := controller.NewRouter(a.Storage, *a.VPPStream)

	srv := http.Server{
		Addr:    a.Cfg.HTTP.Address,
		Handler: engine,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("failed to start http server", "error", err)
		}
	}()

	closer.Add(func() error {
		logger.Info("http server shutting down")

		return srv.Shutdown(ctx)
	})

	logger.Info("http server and prometheus metric exposing started")
}
