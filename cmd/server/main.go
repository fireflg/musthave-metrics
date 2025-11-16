package main

import (
	"flag"
	"fmt"
	"github.com/fireflg/ago-musthave-metrics-tpl/internal/handler"
	"github.com/fireflg/ago-musthave-metrics-tpl/internal/service"
	"github.com/go-chi/chi/v5"
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

func ServerRouter() chi.Router {
	r := chi.NewRouter()
	metricsService := service.NewMetricsService()
	metricsHandler := handler.NewMetricsHandler(metricsService)
	r.Get("/value/{metricType}/{metricName}", metricsHandler.GetMetric)
	r.Post("/update/{metricType}/{metricName}/{metricValue}", metricsHandler.UpdateMetric)
	return r
}

func main() {
	parseServerParams()
	r := ServerRouter()
	fmt.Println("Running server on", flagRunAddr)
	log.Fatal(http.ListenAndServe(flagRunAddr, r))
}
