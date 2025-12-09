package main

import (
	"github.com/fireflg/ago-musthave-metrics-tpl/internal/handler"
	models "github.com/fireflg/ago-musthave-metrics-tpl/internal/model"
	"github.com/fireflg/ago-musthave-metrics-tpl/internal/service"
	"go.uber.org/zap"
	"log"
	"net/http"
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
	}
	storage.InitStorage()

	manager, err := service.NewMertricsManager(cfg, logger, storage)
	if err != nil {
		logger.Fatalf("Failed to initialize manager: %v", err)
	}

	metricsHandler := handler.NewMetricsHandler(*manager, logger)

	r := metricsHandler.ServerRouter()
	logger.Infof("Starting server on %s", cfg.RunAddr)
	err = http.ListenAndServe(cfg.RunAddr, r)
	if err != nil {
		logger.Fatal("server failed to start",
			zap.String("addr", cfg.RunAddr),
			zap.Error(err),
			zap.String("possible_causes",
				"port in use, insufficient privileges, invalid address format"))
	} else {
		logger.Fatalf("server stopped unexpectedly")
	}
}
