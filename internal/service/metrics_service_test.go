package service_test

import (
	"context"
	models "github.com/fireflg/ago-musthave-metrics-tpl/internal/model"
	"github.com/fireflg/ago-musthave-metrics-tpl/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

type MockMetricsRepo struct {
	mock.Mock
}

func (m *MockMetricsRepo) SetCounter(ctx context.Context, id string, delta int64) error {
	args := m.Called(ctx, id, delta)
	return args.Error(0)
}

func (m *MockMetricsRepo) SetGauge(ctx context.Context, id string, value float64) error {
	args := m.Called(ctx, id, value)
	return args.Error(0)
}
func (m *MockMetricsRepo) SetMetric(ctx context.Context, metric models.Metrics) error {
	args := m.Called(ctx, metric)
	return args.Error(0)
}
func (m *MockMetricsRepo) GetCounter(ctx context.Context, id string) (int64, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockMetricsRepo) GetGauge(ctx context.Context, id string) (float64, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockMetricsRepo) Ping(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func TestSetMetric(t *testing.T) {
	repo := new(MockMetricsRepo)
	service := service.NewMetricsService(repo)

	delta := int64(10)
	repo.On("SetCounter", mock.Anything, "counter1", delta).Return(nil)
	err := service.SetMetric(models.Metrics{ID: "counter1", MType: "counter", Delta: &delta})
	assert.NoError(t, err)

	value := 1.23
	repo.On("SetGauge", mock.Anything, "gauge1", value).Return(nil)
	err = service.SetMetric(models.Metrics{ID: "gauge1", MType: "gauge", Value: &value})
	assert.NoError(t, err)

	err = service.SetMetric(models.Metrics{ID: "unknown", MType: "unknown"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown metric type")
}

func TestSetMetricBatch(t *testing.T) {
	repo := new(MockMetricsRepo)
	service := service.NewMetricsService(repo)

	delta := int64(5)
	value := 3.14

	repo.On("SetCounter", mock.Anything, "counter1", delta).Return(nil)
	repo.On("SetGauge", mock.Anything, "gauge1", value).Return(nil)

	metrics := []models.Metrics{
		{ID: "counter1", MType: "counter", Delta: &delta},
		{ID: "gauge1", MType: "gauge", Value: &value},
	}
	err := service.SetMetricBatch(metrics)
	assert.NoError(t, err)
}

func TestGetMetric(t *testing.T) {
	repo := new(MockMetricsRepo)
	service := service.NewMetricsService(repo)

	repo.On("GetCounter", mock.Anything, "counter1").Return(int64(42), nil)
	repo.On("GetGauge", mock.Anything, "gauge1").Return(3.14, nil)

	m, err := service.GetMetric("counter", "counter1")
	assert.NoError(t, err)
	assert.Equal(t, int64(42), *m.Delta)

	m, err = service.GetMetric("gauge", "gauge1")
	assert.NoError(t, err)
	assert.Equal(t, 3.14, *m.Value)

	_, err = service.GetMetric("unknown", "id")
	assert.Error(t, err)
}
