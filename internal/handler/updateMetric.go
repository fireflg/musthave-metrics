package handler

import (
	"github.com/fireflg/ago-musthave-metrics-tpl/internal/service"
	"github.com/go-chi/chi/v5"
	"io"
	"net/http"
)

type MetricsHandler struct {
	service service.MetricsService
}

func NewMetricsHandler(s service.MetricsService) *MetricsHandler {
	return &MetricsHandler{
		service: s,
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
