package main

import (
	"flag"
	"fmt"
	"github.com/fireflg/ago-musthave-metrics-tpl/internal/handler"
	"github.com/fireflg/ago-musthave-metrics-tpl/internal/service"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
	"log"
	"net/http"
	"os"
)

var flagRunAddr string

func parseServerParams() {
	address := os.Getenv("ADDRESS")
	if address == "" {
		flag.StringVar(&flagRunAddr, "a", ":8080", "address and port to run server")
	} else {
		flagRunAddr = address
	}
	if unknownFlag := flag.Args(); len(unknownFlag) > 0 {
		fmt.Fprintf(os.Stderr, "unknown flag(s): %v\n", unknownFlag)
		os.Exit(2)
	}
	flag.Parse()
}

func ServerRouter(logger *zap.SugaredLogger) chi.Router {
	r := chi.NewRouter()
	metricsService := service.NewMetricsService()
	r.Use(handler.WithLogging(logger))
	metricsHandler := handler.NewMetricsHandler(metricsService)
	r.Get("/value/{metricType}/{metricName}", metricsHandler.GetMetric)
	r.Post("/update/{metricType}/{metricName}/{metricValue}", metricsHandler.UpdateMetric)
	return r
}

func main() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	sugar := logger.Sugar()
	parseServerParams()
	r := ServerRouter(sugar)
	sugar.Infof("Running server on %s", flagRunAddr)
	log.Fatal(http.ListenAndServe(flagRunAddr, r))
}
