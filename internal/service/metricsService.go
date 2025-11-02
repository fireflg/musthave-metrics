package service

import (
	"errors"
	models "github.com/fireflg/ago-musthave-metrics-tpl/internal/model"
	"strconv"
)

type MetricsService interface {
	SetMetric(metricType string, name string, value string) error
}

type MetricsStorage struct {
	Metrics []models.Metrics
}

func (m *MetricsStorage) SetMetric(metricType string, metricName string, metricValue string) error {
	if metricType != "gauge" && metricType != "counter" {
		return errors.New("invalid metric type. Use 'gauge' or 'counter'")
	}

	convertedMetricValue, err := strconv.ParseFloat(metricValue, 64)
	if err != nil {
		return errors.New("only numbers allowed")
	}

	var delta int64
	var found bool

	for i := range m.Metrics {
		if m.Metrics[i].ID == metricName {
			if m.Metrics[i].Value != nil {
				delta = int64(convertedMetricValue) - int64(*m.Metrics[i].Value)
			} else {
				delta = 0
			}
			m.Metrics[i].Delta = &delta
			m.Metrics[i].Value = &convertedMetricValue
			found = true
			break
		}
	}
	if !found {
		delta = 0
		m.Metrics = append(m.Metrics, models.Metrics{
			ID:    metricName,
			MType: metricType,
			Delta: &delta,
			Value: &convertedMetricValue,
			Hash:  "",
		})
		return nil
	}
	return nil
}

func NewMetricsService() *MetricsStorage {
	return &MetricsStorage{
		Metrics: make([]models.Metrics, 0),
	}
}
