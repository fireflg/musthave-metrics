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
			if m.Metrics[i].Value == nil {
				return "", errors.New("metric value is nil")
			}
			switch metricType {
			case "counter":
				intVal := int64(*m.Metrics[i].Value)
				return fmt.Sprintf("%d", intVal), nil
			case "gauge":
				return strconv.FormatFloat(*m.Metrics[i].Value, 'f', -1, 64), nil
			default:
				return "", errors.New("unsupported metric type")
			}
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

	for i := range m.Metrics {
		if m.Metrics[i].ID == metricName {
			switch metricType {
			case "counter":
				if m.Metrics[i].Value == nil {
					m.Metrics[i].Value = new(float64)
				}
				*m.Metrics[i].Value += convertedMetricValue
				delta := int64(*m.Metrics[i].Value)
				m.Metrics[i].Delta = &delta

			case "gauge":
				m.Metrics[i].Value = &convertedMetricValue
			}
			log.Printf("set metric %s", metricName)
			return nil
		}
	}

	switch metricType {
	case "counter":
		val := convertedMetricValue
		delta := int64(val)
		m.Metrics = append(m.Metrics, models.Metrics{
			ID:    metricName,
			MType: metricType,
			Value: &val,
			Delta: &delta,
		})
	case "gauge":
		m.Metrics = append(m.Metrics, models.Metrics{
			ID:    metricName,
			MType: metricType,
			Value: &convertedMetricValue,
			Delta: new(int64),
		})
	}

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
