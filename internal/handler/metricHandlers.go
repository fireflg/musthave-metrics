package handler

import (
	"bytes"
	"encoding/json"
	"github.com/fireflg/ago-musthave-metrics-tpl/internal/middleware"
	models "github.com/fireflg/ago-musthave-metrics-tpl/internal/model"
	"github.com/fireflg/ago-musthave-metrics-tpl/internal/service"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
	"io"
	"net/http"
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

	//r.Get("/value/{metricType}/{metricName}", m.GetMetric)
	//r.Post("/update/{metricType}/{metricName}/{metricValue}", m.UpdateMetric)
	r.Post("/update/", middleware.GzipMiddleware(m.UpdateMetricJSON))
	r.Post("/value/", middleware.GzipMiddleware(m.GetMetricJSON))
	r.Get("/ping", m.CheckDB)

	return r
}

func NewMetricsHandler(metricsManager service.MetricManagerImpl, logger *zap.SugaredLogger) *MetricsHandler {
	return &MetricsHandler{metricsManager: metricsManager, logger: logger}
}

//func (m *MetricsHandler) GetMetric(w http.ResponseWriter, r *http.Request) {
//	metricType := chi.URLParam(r, "metricType")
//	metricName := chi.URLParam(r, "metricName")
//
//	value, err := m.metricsManager.GetMetric(metricType, metricName)
//	if err != nil {
//		http.Error(w, err.Error(), http.StatusNotFound)
//		return
//	}
//
//	w.Header().Set("Content-Type", "text/plain")
//	w.WriteHeader(http.StatusOK)
//
//	strValue := strconv.FormatFloat(value, 'f', -1, 64)
//	_, _ = io.WriteString(w, strValue)
//}

//func (m *MetricsHandler) UpdateMetric(w http.ResponseWriter, r *http.Request) {
//	metricValueStr := chi.URLParam(r, "metricValue")
//	metricValue, err := strconv.ParseFloat(metricValueStr, 64)
//	if err != nil {
//		http.Error(w, "Invalid metric value", http.StatusBadRequest)
//		return
//	}
//
//	if err := m.metricsManager.SetMetric(
//		chi.URLParam(r, "metricType"),
//		chi.URLParam(r, "metricName"),
//		metricValue,
//	); err != nil {
//		http.Error(w, err.Error(), http.StatusBadRequest)
//		return
//	}
//
//	w.Header().Set("Content-Type", "text/plain")
//	w.WriteHeader(http.StatusOK)
//}

func (m *MetricsHandler) UpdateMetricJSON(w http.ResponseWriter, r *http.Request) {
	var metricReq models.Metric

	w.Header().Set("Content-Type", "application/json")

	raw, _ := io.ReadAll(r.Body)
	m.logger.Infof("RAW BODY: %s", raw)

	r.Body = io.NopCloser(bytes.NewBuffer(raw))

	if err := json.NewDecoder(r.Body).Decode(&metricReq); err != nil {
		m.logger.Errorf("Invalid JSON: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := m.metricsManager.SetMetric(metricReq); err != nil {
		m.logger.Errorf("Can't update metric: %v, name: %s", err, metricReq.ID)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		m.logger.Errorf("Failed to write response: %v", err)
	}
	m.logger.Infof("Updated metric: %s", metricReq.ID)
}

func (m *MetricsHandler) GetMetricJSON(w http.ResponseWriter, r *http.Request) {
	var metricReq models.Metric

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewDecoder(r.Body).Decode(&metricReq); err != nil {
		m.logger.Errorf("Invalid JSON: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	metric, err := m.metricsManager.GetMetric(metricReq.MType, metricReq.ID)
	if err != nil {
		m.logger.Errorf("Can't get metric: %v, name: %s", err, metricReq.ID)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var value interface{}
	if metric.MType == "counter" {
		value = *metric.Delta
	} else {
		value = *metric.Value
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"id":    metric.ID,
		"type":  metric.MType,
		"value": value,
	})
	m.logger.Infof("Getted metric: %s, value: %d, type: %s", metricReq.ID, value, metricReq.MType)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func (m *MetricsHandler) CheckDB(w http.ResponseWriter, r *http.Request) {
	err := m.metricsManager.CheckDBConn()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"status": "db error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
