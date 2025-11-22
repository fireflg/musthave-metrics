package handler

import (
	"github.com/fireflg/ago-musthave-metrics-tpl/internal/service"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
	"io"
	"net/http"
	"time"
)

type MetricsHandler struct {
	service service.MetricsService
}

func NewMetricsHandler(service service.MetricsService) *MetricsHandler {
	return &MetricsHandler{
		service: service,
	}
}

func (h *MetricsHandler) GetMetric(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	value, err := h.service.GetMetric(chi.URLParam(r, "metricType"), chi.URLParam(r, "metricName"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)

	_, err = io.WriteString(w, value)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *MetricsHandler) UpdateMetric(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := h.service.SetMetric(chi.URLParam(r, "metricType"),
		chi.URLParam(r, "metricName"),
		chi.URLParam(r, "metricValue")); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
}

func WithLogging(logger *zap.SugaredLogger) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		logFn := func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			uri := r.RequestURI
			method := r.Method

			h.ServeHTTP(w, r)

			duration := time.Since(start)

			logger.Infoln(
				"uri", uri,
				"method", method,
				"duration", duration,
			)
		}
		return http.HandlerFunc(logFn)
	}
}
