// Package handler предоставляет HTTP обработчики для операций с метриками.
//
// Пакет реализует RESTful эндпоинты для получения и обновления
// метрик (gauges и counters) с поддержкой gzip сжатия
// и проверки HMAC подписи.
package handler

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/fireflg/go-musthave-metrics-tpl/internal/middleware"
	models "github.com/fireflg/go-musthave-metrics-tpl/internal/model"
	"github.com/fireflg/go-musthave-metrics-tpl/internal/service"
	"github.com/go-chi/chi/v5"

	"go.uber.org/zap"
)

// contextKey — пользовательский тип для избежания коллизий в значениях контекста.
type contextKey string

const clientIPKey contextKey = "client_ip"

// MetricsHandler обрабатывает HTTP запросы для операций с метриками.
type MetricsHandler struct {
	service   service.MetricsService
	logger    *zap.SugaredLogger
	secretKey string
	cryptoKey *rsa.PrivateKey
}

// NewMetricsHandler создает новый экземпляр MetricsHandler.
func NewMetricsHandler(service service.MetricsService, logger *zap.SugaredLogger, cryptoKey *rsa.PrivateKey) *MetricsHandler {
	return &MetricsHandler{service: service, logger: logger, cryptoKey: cryptoKey}
}

// ServerRouter возвращает chi Router со всеми настроенными эндпоинтами метрик.
//
// Эндпоинты:
//   - GET / - Страница проверки здоровья
//   - GET /value/{metricType}/{metricName} - Получить метрику по типу и имени
//   - POST /update/{metricType}/{metricName}/{metricValue} - Обновить одну метрику
//   - POST /update/ - Обновить метрику через JSON тело
//   - POST /updates/ - Пакетное обновление метрик через JSON
//   - POST /value/ - Получить метрику через JSON тело
//   - GET /ping - Проверка здоровья базы данных
func (h *MetricsHandler) ServerRouter() chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.WithLogging(h.logger))

	r.Get("/", middleware.GzipMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<br>hi<br>"))
	}))

	r.Get("/value/{metricType}/{metricName}", h.GetMetric)
	r.Post("/update/{metricType}/{metricName}/{metricValue}", middleware.SignMiddleware(h.UpdateMetric, h.secretKey, h.logger))
	r.Post("/update/", middleware.DecryptMiddleware(middleware.GzipMiddleware(middleware.SignMiddleware(h.UpdateMetricJSON, h.secretKey, h.logger)), h.cryptoKey, h.logger))
	r.Post("/updates/", middleware.DecryptMiddleware(middleware.GzipMiddleware(middleware.SignMiddleware(h.UpdateMetricJSONBatch, h.secretKey, h.logger)), h.cryptoKey, h.logger))
	r.Post("/value/", middleware.GzipMiddleware(h.GetMetricJSON))
	r.Get("/ping", h.CheckDB)
	return r
}

func (h *MetricsHandler) GetMetric(w http.ResponseWriter, r *http.Request) {
	var strValue string

	value, err := h.service.GetMetric(chi.URLParam(r, "metricName"), chi.URLParam(r, "metricType"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)

	switch value.MType {
	case "gauge":
		if value.Value == nil {
			http.Error(w, "gauge value is nil", http.StatusInternalServerError)
			return
		}
		strValue = strconv.FormatFloat(*value.Value, 'f', -1, 64)

	case "counter":
		if value.Delta == nil {
			http.Error(w, "counter delta is nil", http.StatusInternalServerError)
			return
		}
		strValue = strconv.FormatInt(*value.Delta, 10)

	default:
		http.Error(w, "unknown metric type", http.StatusInternalServerError)
		return
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

	ip := r.RemoteAddr

	ctx := context.WithValue(r.Context(), clientIPKey, ip)

	if err := h.service.SetMetric(ctx, metric); err != nil {
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

	h.logger.Infof("update metric %s type %s value %d, delta %d", metric.ID, metric.MType, metric.Value, metric.Delta)

	ip := r.RemoteAddr

	ctx := context.WithValue(r.Context(), clientIPKey, ip)

	err := h.service.SetMetric(ctx, metric)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		h.logger.Errorf("failed to update metric %s: %v", metric.ID, err)
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *MetricsHandler) GetMetricJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var metric models.Metrics
	if err := json.NewDecoder(r.Body).Decode(&metric); err != nil {
		h.logger.Warn("failed to decode request body", "error", err)
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	value, err := h.service.GetMetric(metric.ID, metric.MType)
	if err != nil {
		h.logger.Warnf("metric not found type %s id %s error %s", metric.MType, metric.ID, err)
		http.Error(w, "metric not found", http.StatusNotFound)
		return
	}

	resp := map[string]interface{}{
		"id":   metric.ID,
		"type": metric.MType,
	}

	if value.Delta != nil {
		resp["delta"] = *value.Delta
	}
	if value.Value != nil {
		resp["value"] = *value.Value
	}

	data, err := json.Marshal(resp)
	if err != nil {
		h.logger.Errorf("failed to marshal response, error: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func (h *MetricsHandler) UpdateMetricJSONBatch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var metrics []models.Metrics

	if err := json.NewDecoder(r.Body).Decode(&metrics); err != nil {
		w.WriteHeader(http.StatusBadRequest)
	}
	h.logger.Info("update metrics", zap.Any("metrics", metrics))

	ip := r.RemoteAddr

	ctx := context.WithValue(r.Context(), clientIPKey, ip)

	err := h.service.SetMetricBatch(ctx, metrics)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		h.logger.Errorf("failed to update metrics batch: %v", err)
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
