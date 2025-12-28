package service

import (
	"context"
	"fmt"
	"github.com/fireflg/ago-musthave-metrics-tpl/internal/config/server"
	models "github.com/fireflg/ago-musthave-metrics-tpl/internal/model"
	_ "github.com/jackc/pgx/v5/stdlib"
	"time"
)

type MetricsService interface {
	SetMetric(metric models.Metrics) error
	SetMetricBatch(metrics []models.Metrics) error
	GetMetric(metricType string, metricName string) (models.Metrics, error)
	CheckRepository() error
}
type MetricsServiceImpl struct {
	repo models.MetricsRepository
	Cfg  *server.Config
}

var _ MetricsService = (*MetricsServiceImpl)(nil)

func NewMetricsService(repo models.MetricsRepository) MetricsService {
	return &MetricsServiceImpl{repo: repo}
}

func (m *MetricsServiceImpl) SetMetric(metric models.Metrics) error {
	ctx := context.Background()
	if err := m.repo.SetMetric(ctx, metric); err != nil {
		return err
	}
	return nil
}

func (m *MetricsServiceImpl) SetMetricBatch(metrics []models.Metrics) error {
	for _, metric := range metrics {
		ctx := context.Background()
		if err := m.repo.SetMetric(ctx, metric); err != nil {
			return err
		}
	}
	return nil
}

func (m *MetricsServiceImpl) GetMetric(metricType string, metricName string) (models.Metrics, error) {
	switch metricType {
	case "counter":
		delta, err := m.repo.GetCounter(context.Background(), metricName)
		if err != nil {
			return models.Metrics{}, err
		}
		return models.Metrics{
			ID:    metricName,
			MType: "counter",
			Delta: &delta,
		}, nil

	case "gauge":
		value, err := m.repo.GetGauge(context.Background(), metricName)
		if err != nil {
			return models.Metrics{}, err
		}
		return models.Metrics{
			ID:    metricName,
			MType: "gauge",
			Value: &value,
		}, nil

	default:
		return models.Metrics{}, fmt.Errorf("unknown metric type: %s", metricType)
	}
}

func (m *MetricsServiceImpl) CheckRepository() error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	if err := m.repo.Ping(ctx); err != nil {
		return err
	}
	return nil
}
