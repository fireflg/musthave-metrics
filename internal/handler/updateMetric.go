package handler

import (
	"github.com/fireflg/ago-musthave-metrics-tpl/internal/service"
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

func (h *MetricsHandler) UpdateMetric(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	metricType := r.PathValue("metricType")
	metricName := r.PathValue("metricName")
	metricValue := r.PathValue("metricValue")

	if err := h.service.SetMetric(metricType, metricName, metricValue); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}
