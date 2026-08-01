package main

import (
	"context"
	"github.com/fireflg/go-musthave-metrics-tpl/internal/buildinfo"
	"github.com/fireflg/go-musthave-metrics-tpl/internal/config/server"
	"github.com/fireflg/go-musthave-metrics-tpl/internal/handler"
	"github.com/fireflg/go-musthave-metrics-tpl/internal/observer"
	"github.com/fireflg/go-musthave-metrics-tpl/internal/repository"
	"github.com/fireflg/go-musthave-metrics-tpl/internal/service"
	"go.uber.org/zap"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	buildinfo.Print()

	logger, err := zap.NewProduction()
	if err != nil {
		panic("failed to initialize logger: " + err.Error())
	}
	defer logger.Sync()
	sugar := logger.Sugar()

	cfg, err := server.LoadAServerConfig()
	if err != nil {
		logger.Fatal("Failed to load config", zap.Error(err))
	}

	repo, err := repository.NewRepository(*cfg)
	if err != nil {
		logger.Fatal("Failed to initialize repository", zap.Error(err))
	}

	var observers observer.Observers
	if cfg.AuditFile != "" || cfg.AuditURL != "" {
		observers, err = observer.NewObservers(cfg.AuditFile, cfg.AuditURL)
		if err != nil {
			logger.Fatal("Failed to initialize observer", zap.Error(err))
		}
		sugar.Infow("Audit enabled", "file", cfg.AuditFile, "url", cfg.AuditURL)
	}

	metricsService := service.NewMetricsService(repo, observers)
	metricsHandler := handler.NewMetricsHandler(metricsService, logger.Sugar())
	r := metricsHandler.ServerRouter()

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	srv := &http.Server{
		Addr:        cfg.RunAddr,
		Handler:     r,
		IdleTimeout: 10 * time.Second,
		ReadTimeout: 10 * time.Second,
	}

	go func() {
		sugar.Infof("Starting server on %s", cfg.RunAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal(
				"server failed to start",
				zap.String("addr", cfg.RunAddr),
				zap.Error(err),
			)
		}
	}()

	<-ctx.Done()

	logger.Info("Starting graceful shutdown...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown failed", zap.Error(err))
	}

	if observers != nil {
		if err := observers.Close(); err != nil {
			logger.Error("observer shutdown failed", zap.Error(err))
		}
	}

	logger.Info("Shutdown complete")
}
