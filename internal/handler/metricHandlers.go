package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/fireflg/ago-musthave-metrics-tpl/internal/service"
	"github.com/go-chi/chi/v5"
)

type MetricsHandler struct {
	service service.MetricsService
}

func NewMetricsHandler(service service.MetricsService) *MetricsHandler {
	return &MetricsHandler{service: service}
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

	strValue := strconv.FormatFloat(value, 'f', -1, 64)
	_, err = io.WriteString(w, strValue)
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

	metricValueStr := chi.URLParam(r, "metricValue")
	metricValue, err := strconv.ParseFloat(metricValueStr, 64)
	if err != nil {
		http.Error(w, "Invalid metric value", http.StatusBadRequest)
		return
	}

	if err := h.service.SetMetric(
		chi.URLParam(r, "metricType"),
		chi.URLParam(r, "metricName"),
		metricValue,
	); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
}

func (h *MetricsHandler) UpdateMetricJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method Not Allowed"})
		return
	}

	if err := h.service.DecodeAndSetMetric(r); err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *MetricsHandler) GetMetricJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method Not Allowed"})
		return
	}

	resp, err := h.service.DecodeAndGetMetric(r)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

func (h *MetricsHandler) CheckDB(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method Not Allowed"})
		return
	}

	err := h.service.CheckDBConn()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusOK)
	}

}
