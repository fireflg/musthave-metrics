package service_test

import (
	"context"
	"errors"
	"testing"

	models "github.com/fireflg/ago-musthave-metrics-tpl/internal/model"
	"github.com/fireflg/ago-musthave-metrics-tpl/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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

func (m *MockMetricsRepo) GetCounter(ctx context.Context, id string) (int64, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockMetricsRepo) GetGauge(ctx context.Context, id string) (float64, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockMetricsRepo) SetMetric(ctx context.Context, metric models.Metrics) error {
	args := m.Called(ctx, metric)
	return args.Error(0)
}

func (m *MockMetricsRepo) SetMetrics(ctx context.Context, metrics []models.Metrics) error {
	args := m.Called(ctx, metrics)
	return args.Error(0)
}

func (m *MockMetricsRepo) GetMetric(ctx context.Context, metricID, metricType string) (*models.Metrics, error) {
	args := m.Called(ctx, metricID, metricType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Metrics), args.Error(1)
}

func (m *MockMetricsRepo) Ping(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func TestSetMetric(t *testing.T) {
	repo := new(MockMetricsRepo)
	svc := service.NewMetricsService(repo)

	delta := int64(10)
	metricCounter := models.Metrics{
		ID:    "counter1",
		MType: "counter",
		Delta: &delta,
	}

	repo.On("SetMetric", mock.Anything, metricCounter).Return(nil)

	err := svc.SetMetric(metricCounter)
	assert.NoError(t, err)

	value := 3.14
	metricGauge := models.Metrics{
		ID:    "gauge1",
		MType: "gauge",
		Value: &value,
	}

	repo.On("SetMetric", mock.Anything, metricGauge).Return(nil)

	err = svc.SetMetric(metricGauge)
	assert.NoError(t, err)

	repo.AssertExpectations(t)
}

func TestSetMetricBatch(t *testing.T) {
	repo := new(MockMetricsRepo)
	svc := service.NewMetricsService(repo)

	delta := int64(5)
	value := 2.71
	metrics := []models.Metrics{
		{ID: "counter1", MType: "counter", Delta: &delta},
		{ID: "gauge1", MType: "gauge", Value: &value},
	}

	repo.On("SetMetrics", mock.Anything, metrics).Return(nil)

	err := svc.SetMetricBatch(metrics)
	assert.NoError(t, err)

	repo.AssertExpectations(t)
}

func TestGetMetric(t *testing.T) {
	repo := new(MockMetricsRepo)
	svc := service.NewMetricsService(repo)

	delta := int64(42)
	repo.On("GetMetric", mock.Anything, "counter1", "counter").
		Return(&models.Metrics{ID: "counter1", MType: "counter", Delta: &delta}, nil)

	value := 3.14
	repo.On("GetMetric", mock.Anything, "gauge1", "gauge").
		Return(&models.Metrics{ID: "gauge1", MType: "gauge", Value: &value}, nil)

	m, err := svc.GetMetric("counter1", "counter")
	assert.NoError(t, err)
	assert.NotNil(t, m.Delta)
	assert.Equal(t, int64(42), *m.Delta)

	m, err = svc.GetMetric("gauge1", "gauge")
	assert.NoError(t, err)
	assert.NotNil(t, m.Value)
	assert.Equal(t, 3.14, *m.Value)

	repo.On("GetMetric", mock.Anything, "unknown", "unknown").Return(nil, errors.New("not found"))
	_, err = svc.GetMetric("unknown", "unknown")
	assert.Error(t, err)

	repo.AssertExpectations(t)
}
