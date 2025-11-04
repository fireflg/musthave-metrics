package service

import (
	"errors"
	"fmt"
	models "github.com/fireflg/ago-musthave-metrics-tpl/internal/model"
	"log"
	"strconv"
)

type MetricsService interface {
	SetMetric(metricType string, metricName string, value string) error
	GetMetric(metricType string, metricName string) (value string, err error)
}

type MetricsStorage struct {
	Metrics []models.Metrics
}

var _ MetricsService = (*MetricsStorage)(nil)

func (m *MetricsStorage) GetMetric(metricType string, metricName string) (value string, err error) {
	if err := checkMetricType(metricType); err != nil {
		return "", err
	}
	for i := range m.Metrics {
		if m.Metrics[i].ID == metricName {
			convertedMetricValue := fmt.Sprintf("%g", *m.Metrics[i].Value)
			return convertedMetricValue, nil
		}

	}

	return "", errors.New("metric not found")
}

func (m *MetricsStorage) SetMetric(metricType string, metricName string, metricValue string) error {
	if err := checkMetricType(metricType); err != nil {
		return err
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
		Metrics: make([]models.Metrics, 0),
	}
}
