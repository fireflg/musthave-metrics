package main

import (
	"github.com/fireflg/ago-musthave-metrics-tpl/internal/handler"
	"github.com/fireflg/ago-musthave-metrics-tpl/internal/service"
	"net/http"
)

func main() {
	metricsService := service.NewMetricsService()
	metricsHandler := handler.NewMetricsHandler(metricsService)
	mux := http.NewServeMux()
	mux.Handle("POST /update/{metricType}/{metricName}/{metricValue}",
		http.HandlerFunc(metricsHandler.UpdateMetric))
	err := http.ListenAndServe(`:8080`, mux)
	if err != nil {
		panic(err)
	}
}
