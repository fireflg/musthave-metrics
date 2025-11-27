package service

import (
	"encoding/json"
	"errors"
	models "github.com/fireflg/ago-musthave-metrics-tpl/internal/model"
	"net/http"
	"sync"
)

type MetricsService interface {
	SetMetric(metricType string, metricName string, metricValue float64) error
	GetMetric(metricType string, metricName string) (value float64, err error)
	DecodeAndSetMetric(r *http.Request) error
	DecodeAndGetMetric(r *http.Request) ([]byte, error)
}

type MetricsStorage struct {
	Metrics map[string]models.Metrics
	Mutex   sync.Mutex
}

var _ MetricsService = (*MetricsStorage)(nil)

func (m *MetricsStorage) SetMetric(metricType string, metricName string, metricValue float64) error {
	m.Mutex.Lock()
	defer m.Mutex.Unlock()

	if err := checkMetricType(metricType); err != nil {
		return err
	}

	metric, exists := m.Metrics[metricName]
	if !exists {
		metric = models.Metrics{
			ID:    metricName,
			MType: metricType,
		}
	}

	switch metricType {
	case "counter":
		var delta int64
		if metric.Delta != nil {
			delta = *metric.Delta
		}
		delta += int64(metricValue)
		val := float64(delta)
		metric.Delta = &delta
		metric.Value = &val
	case "gauge":
		metric.Delta = nil
		metric.Value = &metricValue
	}

	m.Metrics[metricName] = metric
	return nil
}

func (m *MetricsStorage) GetMetric(metricType string, metricName string) (float64, error) {
	m.Mutex.Lock()
	defer m.Mutex.Unlock()

	if err := checkMetricType(metricType); err != nil {
		return 0, err
	}

	metric, exists := m.Metrics[metricName]
	if !exists {
		return 0, errors.New("metric not found")
	}

	switch metricType {
	case "gauge":
		if metric.Value == nil {
			return 0, errors.New("gauge value is nil")
		}
		return *metric.Value, nil
	case "counter":
		if metric.Delta == nil {
			return 0, errors.New("counter delta is nil")
		}
		return float64(*metric.Delta), nil
	default:
		return 0, errors.New("unknown metric type")
	}
}

func (m *MetricsStorage) DecodeAndSetMetric(r *http.Request) error {
	var metric models.Metrics
	if err := json.NewDecoder(r.Body).Decode(&metric); err != nil {
		return err
	}

	switch metric.MType {
	case "gauge":
		if metric.Value == nil {
			return errors.New("value required for gauge")
		}
		return m.SetMetric("gauge", metric.ID, *metric.Value)
	case "counter":
		if metric.Delta == nil {
			return errors.New("delta required for counter")
		}
		return m.SetMetric("counter", metric.ID, float64(*metric.Delta))
	default:
		return errors.New("unknown metric type")
	}
}

func (m *MetricsStorage) DecodeAndGetMetric(r *http.Request) ([]byte, error) {
	var metric models.Metrics
	if err := json.NewDecoder(r.Body).Decode(&metric); err != nil {
		return nil, err
	}

	m.Mutex.Lock()
	defer m.Mutex.Unlock()
	stored, ok := m.Metrics[metric.ID]
	if !ok {
		return nil, errors.New("metric not found")
	}

	resp := map[string]interface{}{
		"id":   stored.ID,
		"type": stored.MType,
	}

	switch stored.MType {
	case "gauge":
		if stored.Value == nil {
			return nil, errors.New("gauge value is nil")
		}
		resp["value"] = *stored.Value
	case "counter":
		if stored.Delta == nil {
			return nil, errors.New("counter delta is nil")
		}
		resp["delta"] = *stored.Delta
	default:
		return nil, errors.New("unknown metric type")
	}

	return json.Marshal(resp)
}

func checkMetricType(metricType string) error {
	if metricType != "gauge" && metricType != "counter" {
		return errors.New("invalid metric type. Use 'gauge' or 'counter'")
	}
	return nil
}

func NewMetricsService() *MetricsStorage {
	return &MetricsStorage{
		Metrics: make(map[string]models.Metrics),
	}
}
