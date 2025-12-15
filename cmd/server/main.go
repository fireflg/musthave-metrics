package main

import (
	"context"
	"github.com/fireflg/ago-musthave-metrics-tpl/internal/config/server"
	"github.com/fireflg/ago-musthave-metrics-tpl/internal/handler"
	"github.com/fireflg/ago-musthave-metrics-tpl/internal/middleware"
	"github.com/fireflg/ago-musthave-metrics-tpl/internal/repository"
	"github.com/fireflg/ago-musthave-metrics-tpl/internal/service"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func ServerRouter(metricsService service.MetricsService, logger *zap.SugaredLogger) chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.WithLogging(logger))

	r.Get("/", middleware.GzipMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<br>hi<br>"))
	}))
	metricsHandler := handler.NewMetricsHandler(metricsService)
	r.Get("/value/{metricType}/{metricName}", metricsHandler.GetMetric)
	r.Post("/update/{metricType}/{metricName}/{metricValue}", metricsHandler.UpdateMetric)
	r.Post("/update/", middleware.GzipMiddleware(metricsHandler.UpdateMetricJSON))
	r.Post("/updates/", middleware.GzipMiddleware(metricsHandler.UpdateMetricJSONBatch))
	r.Post("/value/", middleware.GzipMiddleware(metricsHandler.GetMetricJSON))
	r.Get("/ping", metricsHandler.CheckDB)

	return r
}

func main() {
	logger, err := zap.NewDevelopment()
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

	metricsService := service.NewMetricsService(repo)

	r := ServerRouter(metricsService, sugar)

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	srv := &http.Server{
		Addr:    cfg.RunAddr,
		Handler: r,
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

	logger.Info("Shutdown complete")
}
