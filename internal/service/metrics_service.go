package service

import (
	"context"
	"github.com/fireflg/ago-musthave-metrics-tpl/internal/config/server"
	models "github.com/fireflg/ago-musthave-metrics-tpl/internal/model"
	"github.com/fireflg/ago-musthave-metrics-tpl/internal/observer"
	_ "github.com/jackc/pgx/v5/stdlib"
	"time"
)

type MetricsService interface {
	SetMetric(ctx context.Context, metric models.Metrics) error
	SetMetricBatch(ctx context.Context, metrics []models.Metrics) error
	GetMetric(metricID, metricType string) (*models.Metrics, error)
	CheckRepository() error
}
type MetricsServiceImpl struct {
	repo     models.MetricsRepository
	Cfg      *server.Config
	observer *observer.Auditor
}

var _ MetricsService = (*MetricsServiceImpl)(nil)

func NewMetricsService(repo models.MetricsRepository, auditor *observer.Auditor) MetricsService {
	return &MetricsServiceImpl{repo: repo, observer: auditor}
}

func (m *MetricsServiceImpl) SetMetric(ctx context.Context, metric models.Metrics) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)

	defer cancel()
	if err := m.repo.SetMetric(ctx, metric); err != nil {
		return err
	}

	if m.observer != nil {
		m.observer.Notify(ctx, metric.ID)
	}

	return nil
}

func (m *MetricsServiceImpl) SetMetricBatch(ctx context.Context, metrics []models.Metrics) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := m.repo.SetMetrics(ctx, metrics); err != nil {
		return err
	}

	if m.observer != nil {
		ids := make([]string, len(metrics))
		for i, m := range metrics {
			ids[i] = m.ID
		}
		m.observer.NotifyBatch(ctx, ids)
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
