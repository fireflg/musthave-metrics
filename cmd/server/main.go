package main

import (
	"github.com/fireflg/ago-musthave-metrics-tpl/internal/handler"
	"github.com/fireflg/ago-musthave-metrics-tpl/internal/service"
	"github.com/go-chi/chi/v5"
	"log"
	"net/http"
)

func ServerRouter() chi.Router {
	r := chi.NewRouter()
	metricsService := service.NewMetricsService()
	metricsHandler := handler.NewMetricsHandler(metricsService)
	r.Get("/value/{metricType}/{metricName}", metricsHandler.GetMetric)
	r.Post("/update/{metricType}/{metricName}/{metricValue}", metricsHandler.UpdateMetric)
	return r
}

func main() {
	r := ServerRouter()
	log.Fatal(http.ListenAndServe(":8080", r))
}
