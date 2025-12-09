package handler

import (
	"encoding/json"
	"github.com/fireflg/ago-musthave-metrics-tpl/internal/middleware"
	models "github.com/fireflg/ago-musthave-metrics-tpl/internal/model"
	"go.uber.org/zap"
	"io"
	"net/http"
	"strconv"

	"github.com/fireflg/ago-musthave-metrics-tpl/internal/service"
	"github.com/go-chi/chi/v5"
)

type MetricsHandler struct {
	metricsManager service.MetricManagerImpl
	logger         *zap.SugaredLogger
}

func (m *MetricsHandler) ServerRouter() chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.WithLogging(m.logger))

	r.Get("/", middleware.GzipMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<br>hi<br>"))
	}))

	r.Get("/value/{metricType}/{metricName}", m.GetMetric)
	r.Post("/update/{metricType}/{metricName}/{metricValue}", m.UpdateMetric)
	r.Post("/update/", middleware.GzipMiddleware(m.UpdateMetricJSON))
	r.Post("/value/", middleware.GzipMiddleware(m.GetMetricJSON))
	r.Get("/ping", m.CheckDB)

	return r
}

func NewMetricsHandler(metricsManager service.MetricManagerImpl, logger *zap.SugaredLogger) *MetricsHandler {
	return &MetricsHandler{metricsManager: metricsManager, logger: logger}
}

func (m *MetricsHandler) GetMetric(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	metricType := chi.URLParam(r, "metricType")
	metricName := chi.URLParam(r, "metricName")

	value, err := m.metricsManager.GetMetric(metricType, metricName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)

	strValue := strconv.FormatFloat(value, 'f', -1, 64)
	_, _ = io.WriteString(w, strValue)
}

func (m *MetricsHandler) UpdateMetric(w http.ResponseWriter, r *http.Request) {
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

	if err := m.metricsManager.SetMetric(
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

func (m *MetricsHandler) UpdateMetricJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method Not Allowed"})
		return
	}

	var metricReq models.Metric
	if err := json.NewDecoder(r.Body).Decode(&metricReq); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON"})
		return
	}

	switch metricReq.MType {
	case "counter":
		if err := m.metricsManager.SetMetric(metricReq.MType, metricReq.ID, float64(*metricReq.Delta)); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

	case "gauge":
		if err := m.metricsManager.SetMetric(metricReq.MType, metricReq.ID, *metricReq.Value); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Unknown metric type"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (m *MetricsHandler) GetMetricJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method Not Allowed"})
		return
	}

	var metricReq models.Metric
	if err := json.NewDecoder(r.Body).Decode(&metricReq); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON"})
		return
	}

	value, err := m.metricsManager.GetMetric(metricReq.MType, metricReq.ID)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	resp := map[string]interface{}{
		"id":    metricReq.ID,
		"type":  metricReq.MType,
		"value": value,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func (m *MetricsHandler) CheckDB(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Method Not Allowed"})
		return
	}

	err := m.metricsManager.CheckDBConn()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusOK)
	}

}
