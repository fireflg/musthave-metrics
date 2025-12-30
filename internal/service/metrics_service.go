package service

import (
	"context"
	"github.com/fireflg/ago-musthave-metrics-tpl/internal/config/server"
	models "github.com/fireflg/ago-musthave-metrics-tpl/internal/model"
	_ "github.com/jackc/pgx/v5/stdlib"
	"time"
)

type MetricsService interface {
	SetMetric(metric models.Metrics) error
	SetMetricBatch(metrics []models.Metrics) error
	GetMetric(metricID, metricType string) (*models.Metrics, error)
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
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := m.repo.SetMetric(ctx, metric); err != nil {
		return err
	}
	return nil
}

func (m *MetricsServiceImpl) SetMetricBatch(metrics []models.Metrics) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := m.repo.SetMetrics(ctx, metrics); err != nil {
		return err
	}
	return nil
}

func (m *MetricsServiceImpl) GetMetric(metricID, metricType string) (*models.Metrics, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	metric, err := m.repo.GetMetric(ctx, metricID, metricType)
	if err != nil {
		return nil, err
	}
	return metric, nil
}

func (m *MetricsServiceImpl) CheckRepository() error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	if err := m.repo.Ping(ctx); err != nil {
		return err
	}
	return nil
}
