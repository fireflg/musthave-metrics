package main

import (
	"context"
	"errors"
	"github.com/fireflg/ago-musthave-metrics-tpl/internal/handler"
	models "github.com/fireflg/ago-musthave-metrics-tpl/internal/model"
	"github.com/fireflg/ago-musthave-metrics-tpl/internal/service"
	"go.uber.org/zap"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	l, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("can't initialize zap logger: %v", err)
	}
	logger := l.Sugar()
	defer logger.Sync()

	cfg, err := service.LoadAServerConfig()
	if err != nil {
		logger.Fatal("Failed to load config", zap.Error(err))
	}

	storage := &service.Storage{
		Metrics:         make(map[string]models.Metric),
		StorageInterval: cfg.PersistentStorageInterval,
		StorageRestore:  cfg.PersistentStorageRestore,
		StoragePath:     cfg.PersistentStoragePath,
		Logger:          logger,
	}

	storage.InitStorage()

	manager, err := service.NewMertricsManager(cfg, logger, storage)
	if err != nil {
		logger.Fatal("Failed to initialize manager", zap.Error(err))
	}

	metricsHandler := handler.NewMetricsHandler(*manager, logger)
	router := metricsHandler.ServerRouter()

	server := &http.Server{
		Addr:    cfg.RunAddr,
		Handler: router,
	}

	logger.Infof("Starting server on %s", cfg.RunAddr)

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("HTTP server failed", zap.Error(err))
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	logger.Info("Shutdown signal received...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Graceful shutdown failed", zap.Error(err))
	} else {
		logger.Info("Graceful shutdown complete")
	}
}
