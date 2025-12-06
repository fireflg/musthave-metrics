package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/fireflg/ago-musthave-metrics-tpl/internal/handler"
	"github.com/fireflg/ago-musthave-metrics-tpl/internal/middleware"
	"github.com/fireflg/ago-musthave-metrics-tpl/internal/service"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

var (
	flagRunAddr         string
	flagStoreInterval   int
	flagFileStoragePath string
	flagRestore         bool
)

func parseServerParams() {
	address := os.Getenv("ADDRESS")
	if address != "" {
		flagRunAddr = address
	} else {
		flag.StringVar(&flagRunAddr, "a", ":8080", "address and port to run server")
	}

	envFilePath := os.Getenv("FILE_STORAGE_PATH")
	if envFilePath != "" {
		flagFileStoragePath = envFilePath
	} else {
		defaultFile := "metrics.json"
		flag.StringVar(&flagFileStoragePath, "f", defaultFile, "path to store metrics")
	}

	envStoreInterval := os.Getenv("STORE_INTERVAL")
	if envStoreInterval != "" {
		if val, err := strconv.Atoi(envStoreInterval); err == nil && val >= 0 {
			flagStoreInterval = val
		} else {
			fmt.Fprintf(os.Stderr, "Invalid STORE_INTERVAL=%s, must be >=0\n", envStoreInterval)
			os.Exit(2)
		}
	} else {
		flag.IntVar(&flagStoreInterval, "i", 300, "interval to store metrics in seconds (0 = sync save)")
	}

	envRestore := os.Getenv("RESTORE")
	if envRestore != "" {
		flagRestore = envRestore == "true" || envRestore == "1"
	} else {
		flag.BoolVar(&flagRestore, "r", false, "restore metrics from file on server start")
	}

	flag.Parse()

	if unknownFlag := flag.Args(); len(unknownFlag) > 0 {
		fmt.Fprintf(os.Stderr, "unknown flag(s): %v\n", unknownFlag)
		os.Exit(2)
	}

	if flagFileStoragePath == "" {
		fmt.Fprintln(os.Stderr, "file storage path cannot be empty")
		os.Exit(2)
	}
}

func ServerRouter(metricsService *service.MetricsStorage, logger *zap.SugaredLogger) chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.WithLogging(logger))

	r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("pong"))
	})
	r.Get("/", middleware.GzipMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<br>hi<br>"))
	}))
	metricsHandler := handler.NewMetricsHandler(metricsService)
	r.Get("/value/{metricType}/{metricName}", metricsHandler.GetMetric)
	r.Post("/update/{metricType}/{metricName}/{metricValue}", metricsHandler.UpdateMetric)
	r.Post("/update/", middleware.GzipMiddleware(metricsHandler.UpdateMetricJSON))
	r.Post("/value/", middleware.GzipMiddleware(metricsHandler.GetMetricJSON))

	return r
}

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic("failed to initialize logger: " + err.Error())
	}
	defer logger.Sync()
	sugar := logger.Sugar()

	parseServerParams()
	sugar.Infof("Server parameters: addr=%s, storage=%s, restore=%v, interval=%d",
		flagRunAddr, flagFileStoragePath, flagRestore, flagStoreInterval)

	metricsService := service.NewMetricsService()

	if flagRestore {
		if err := metricsService.RestoreMetrics(flagFileStoragePath); err != nil {
			logger.Error("failed to restore metrics",
				zap.String("file", flagFileStoragePath),
				zap.Error(err))
		} else {
			logger.Info("metrics successfully restored", zap.String("file", flagFileStoragePath))
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := metricsService.SaveMetrics(ctx, flagFileStoragePath, flagStoreInterval); err != nil &&
			!errors.Is(err, context.Canceled) {

			sugar.Error("SaveMetrics stopped", "error", err)
		}
	}()

	r := ServerRouter(metricsService, sugar)

	sugar.Infof("Starting server on %s", flagRunAddr)
	err = http.ListenAndServe(flagRunAddr, r)
	if err != nil {
		// Логируем подробно, почему сервер не стартует
		logger.Fatal("server failed to start",
			zap.String("addr", flagRunAddr),
			zap.Error(err),
			zap.String("possible_causes",
				"port in use, insufficient privileges, invalid address format"))
	}
}
