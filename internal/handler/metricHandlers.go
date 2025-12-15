package handler

import (
	"encoding/json"
	"fmt"
	models "github.com/fireflg/ago-musthave-metrics-tpl/internal/model"
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
	var strValue string

	value, err := h.service.GetMetric(chi.URLParam(r, "metricType"), chi.URLParam(r, "metricName"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)

	if value.MType == "gauge" {
		strValue = strconv.FormatFloat(*value.Value, 'f', -1, 64)
	} else {
		strValue = strconv.FormatInt(*value.Delta, 10)
	}
	_, err = io.WriteString(w, strValue)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *MetricsHandler) UpdateMetric(w http.ResponseWriter, r *http.Request) {
	var metric models.Metrics
	metric.MType = chi.URLParam(r, "metricType")
	metric.ID = chi.URLParam(r, "metricName")

	if metric.MType != "gauge" && metric.MType != "counter" {
		http.Error(w, "Invalid metric type", http.StatusBadRequest)
		return
	}

	metricValueStr := chi.URLParam(r, "metricValue")
	if metricValueStr == "" {
		http.Error(w, "Metric value is required", http.StatusBadRequest)
		return
	}

	if metric.MType == "gauge" {
		floatValue, err := strconv.ParseFloat(metricValueStr, 64)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid gauge value: %v", err), http.StatusBadRequest)
			return
		}
		metric.Value = &floatValue
	} else {
		intValue, err := strconv.ParseInt(metricValueStr, 10, 64)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid counter value: %v", err), http.StatusBadRequest)
			return
		}
		metric.Delta = &intValue
	}
	if err := h.service.SetMetric(metric); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
}

func (h *MetricsHandler) UpdateMetricJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var metric models.Metrics

	if err := json.NewDecoder(r.Body).Decode(&metric); err != nil {
		w.WriteHeader(http.StatusBadRequest)
	}

	err := h.service.SetMetric(metric)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *MetricsHandler) GetMetricJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var metric models.Metrics

	if err := json.NewDecoder(r.Body).Decode(&metric); err != nil {
		w.WriteHeader(http.StatusBadRequest)
	}

	respRaw := map[string]interface{}{
		"id":   metric.ID,
		"type": metric.MType,
	}

	value, err := h.service.GetMetric(metric.MType, metric.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
	}

	if metric.MType == "gauge" {
		if value.Value == nil {
			http.Error(w, "value is nil", http.StatusInternalServerError)
			return
		}
		respRaw["value"] = *value.Value
	} else {
		if value.Delta == nil {
			http.Error(w, "delta is nil", http.StatusInternalServerError)
			return
		}
		respRaw["delta"] = *value.Delta
	}

	resp, err := json.Marshal(respRaw)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

func (h *MetricsHandler) UpdateMetricJSONBatch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var metrics []models.Metrics

	if err := json.NewDecoder(r.Body).Decode(&metrics); err != nil {
		w.WriteHeader(http.StatusBadRequest)
	}
	err := h.service.SetMetricBatch(metrics)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *MetricsHandler) CheckDB(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	err := h.service.CheckRepository()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusOK)
	}
}
