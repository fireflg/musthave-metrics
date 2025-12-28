package memory

import (
	"context"
	"errors"
	"fmt"
	models "github.com/fireflg/ago-musthave-metrics-tpl/internal/model"
	"sync"
)

type MemoryRepository struct {
	Metrics map[string]models.Metrics
	mu      sync.Mutex
}

func NewMemoryRepository() models.MetricsRepository {
	return &MemoryRepository{
		Metrics: make(map[string]models.Metrics),
	}
}

func (m *MemoryRepository) SetGauge(ctx context.Context, name string, value float64) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("operation canceled: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	metric, exists := m.Metrics[name]
	if !exists {
		metric = models.Metrics{
			ID:    name,
			MType: "gauge",
		}
	}
	metric.Delta = nil
	metric.Value = &value

	m.Metrics[name] = metric
	return nil
}

func (m *MemoryRepository) SetCounter(ctx context.Context, name string, value int64) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("operation canceled: %w", err)
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	metric, exists := m.Metrics[name]
	if !exists {
		metric = models.Metrics{
			ID:    name,
			MType: "counter",
		}
	}

	var delta int64
	if metric.Delta != nil {
		delta = *metric.Delta
	}
	delta += value
	metric.Delta = &delta
	metric.Value = nil

	m.Metrics[name] = metric
	return nil
}

func (m *MemoryRepository) SetMetric(ctx context.Context, metric models.Metrics) error {
	if metric.ID == "" {
		return fmt.Errorf("metric ID is empty")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	switch metric.MType {
	case "counter":
		if metric.Delta == nil {
			return fmt.Errorf("counter metric delta is nil")
		}

		existing, exists := m.Metrics[metric.ID]
		if !exists {
			existing = models.Metrics{
				ID:    metric.ID,
				MType: "counter",
			}
		}

		var current int64
		if existing.Delta != nil {
			current = *existing.Delta
		}

		current += *metric.Delta
		existing.Delta = &current
		existing.Value = nil

		m.Metrics[metric.ID] = existing

	case "gauge":
		if metric.Value == nil {
			return fmt.Errorf("gauge metric value is nil")
		}

		m.Metrics[metric.ID] = models.Metrics{
			ID:    metric.ID,
			MType: "gauge",
			Value: metric.Value,
			Delta: nil,
		}

	default:
		return fmt.Errorf("unknown metric type: %s", metric.MType)
	}

	return nil
}

func (m *MemoryRepository) GetCounter(ctx context.Context, name string) (int64, error) {

	if err := ctx.Err(); err != nil {
		return 0, fmt.Errorf("operation canceled: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	metric, exists := m.Metrics[name]
	if !exists {
		return 0, errors.New("metric not found")
	}

	if metric.Delta == nil {
		return 0, errors.New("counter delta is nil")
	}
	return *metric.Delta, nil
}

func (m *MemoryRepository) GetGauge(ctx context.Context, name string) (float64, error) {
	if err := ctx.Err(); err != nil {
		return 0, fmt.Errorf("operation canceled: %w", err)
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	metric, exists := m.Metrics[name]
	if !exists {
		return 0, errors.New("metric not found")
	}

	if metric.Value == nil {
		return 0, errors.New("gauge value is nil")
	}
	return *metric.Value, nil

}

func (m *MemoryRepository) Ping(ctx context.Context) error {
	return nil
}

func (m *MemoryRepository) GetAllMetrics() map[string]models.Metrics {
	return m.Metrics
}
