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

func (m *MockMetricsRepo) SetMetric(ctx context.Context, metric models.Metrics) error {
	args := m.Called(ctx, metric)
	return args.Error(0)
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

func (m *MockMetricsRepo) Ping(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func TestSetMetric(t *testing.T) {
	repo := new(MockMetricsRepo)
	svc := service.NewMetricsService(repo)

	delta := int64(10)

	repo.On(
		"SetMetric",
		mock.Anything,
		mock.MatchedBy(func(m models.Metrics) bool {
			return m.ID == "counter1" &&
				m.MType == "counter" &&
				m.Delta != nil &&
				*m.Delta == delta
		}),
	).Return(nil)

	err := svc.SetMetric(models.Metrics{
		ID:    "counter1",
		MType: "counter",
		Delta: &delta,
	})
	assert.NoError(t, err)

	value := 1.23

	repo.On(
		"SetMetric",
		mock.Anything,
		mock.MatchedBy(func(m models.Metrics) bool {
			return m.ID == "gauge1" &&
				m.MType == "gauge" &&
				m.Value != nil &&
				*m.Value == value
		}),
	).Return(nil)

	err = svc.SetMetric(models.Metrics{
		ID:    "gauge1",
		MType: "gauge",
		Value: &value,
	})
	assert.NoError(t, err)

	repo.On(
		"SetMetric",
		mock.Anything,
		mock.MatchedBy(func(m models.Metrics) bool {
			return m.MType == "unknown"
		}),
	).Return(errors.New("unknown metric type"))

	err = svc.SetMetric(models.Metrics{
		ID:    "unknown",
		MType: "unknown",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown")

	repo.AssertExpectations(t)
}

func TestSetMetricBatch(t *testing.T) {
	repo := new(MockMetricsRepo)
	svc := service.NewMetricsService(repo)

	delta := int64(5)
	value := 3.14

	repo.On(
		"SetMetric",
		mock.Anything,
		mock.AnythingOfType("models.Metrics"),
	).Return(nil).Twice()

	metrics := []models.Metrics{
		{
			ID:    "counter1",
			MType: "counter",
			Delta: &delta,
		},
		{
			ID:    "gauge1",
			MType: "gauge",
			Value: &value,
		},
	}

	err := svc.SetMetricBatch(metrics)
	assert.NoError(t, err)

	repo.AssertNumberOfCalls(t, "SetMetric", 2)
}

func TestGetMetric(t *testing.T) {
	repo := new(MockMetricsRepo)
	svc := service.NewMetricsService(repo)

	repo.On("GetCounter", mock.Anything, "counter1").
		Return(int64(42), nil)

	repo.On("GetGauge", mock.Anything, "gauge1").
		Return(3.14, nil)

	m, err := svc.GetMetric("counter", "counter1")
	assert.NoError(t, err)
	assert.NotNil(t, m.Delta)
	assert.Equal(t, int64(42), *m.Delta)

	m, err = svc.GetMetric("gauge", "gauge1")
	assert.NoError(t, err)
	assert.NotNil(t, m.Value)
	assert.Equal(t, 3.14, *m.Value)

	_, err = svc.GetMetric("unknown", "id")
	assert.Error(t, err)

	repo.AssertExpectations(t)
}
