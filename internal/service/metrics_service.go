// Package service предоставляет бизнес-логику для управления метриками.
//
// Уровень сервиса координирует работу между HTTP обработчиками и репозиториями данных,
// предоставляя методы для получения и установки метрик с поддержкой наблюдателей для
// отслеживания изменений.
package service

import (
	"context"
	"time"

	models "github.com/fireflg/go-musthave-metrics-tpl/internal/model"
	"github.com/fireflg/go-musthave-metrics-tpl/internal/observer"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// MetricsService defines the interface for metrics business operations.
type MetricsService interface {
	// SetMetric stores a single metric.
	SetMetric(ctx context.Context, metric models.Metrics) error
	// SetMetricBatch stores multiple metrics in one operation.
	SetMetricBatch(ctx context.Context, metrics []models.Metrics) error
	// GetMetric retrieves a metric by its ID and type.
	GetMetric(metricID, metricType string) (*models.Metrics, error)
	// CheckRepository verifies the repository is accessible.
	CheckRepository() error
}

// MetricsServiceImpl implements MetricsService with repository storage
// and optional observer for change notifications.
type MetricsServiceImpl struct {
	repo     models.MetricsRepository
	observer observer.Observers
}

// Verify MetricsServiceImpl implements MetricsService interface.
var _ MetricsService = (*MetricsServiceImpl)(nil)

// NewMetricsService creates a new MetricsService instance.
func NewMetricsService(repo models.MetricsRepository, observers observer.Observers) MetricsService {
	return &MetricsServiceImpl{repo: repo, observer: observers}
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
