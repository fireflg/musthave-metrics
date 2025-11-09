package service

import (
	"errors"
	"fmt"
	models "github.com/fireflg/ago-musthave-metrics-tpl/internal/model"
	"log"
	"strconv"
	"sync"
)

type MetricsService interface {
	SetMetric(metricType string, metricName string, value string) error
	GetMetric(metricType string, metricName string) (value string, err error)
}

type MetricsStorage struct {
	Metrics map[string]models.Metrics
	Mutex   sync.Mutex
}

var _ MetricsService = (*MetricsStorage)(nil)

func (m *MetricsStorage) GetMetric(metricType string, metricName string) (value string, err error) {
	if err := checkMetricType(metricType); err != nil {
		return "", err
	}

	metric, exists := m.Metrics[metricName]
	if !exists {
		return "", errors.New("metric not found")
	}

	if metric.Value == nil {
		return "", errors.New("metric value is nil")
	}

	switch metricType {
	case "counter":
		intVal := int64(*metric.Value)
		return fmt.Sprintf("%d", intVal), nil
	case "gauge":
		return strconv.FormatFloat(*metric.Value, 'f', -1, 64), nil
	default:
		return "", errors.New("unsupported metric type")
	}
}

func (m *MetricsStorage) SetMetric(metricType string, metricName string, metricValue string) error {
	m.Mutex.Lock()
	defer m.Mutex.Unlock()

	if err := checkMetricType(metricType); err != nil {
		return err
	}

	convertedMetricValue, err := strconv.ParseFloat(metricValue, 64)
	if err != nil {
		return errors.New("only numbers allowed")
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
		delta += int64(convertedMetricValue)
		val := float64(delta)
		metric.Value = &val
		metric.Delta = &delta

	case "gauge":
		metric.Value = &convertedMetricValue
		metric.Delta = nil
	}

	m.Metrics[metricName] = metric

	log.Printf("set metric %s", metricName)
	return nil
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
