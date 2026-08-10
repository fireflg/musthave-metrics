package service_test

import (
	"context"
	"errors"
	"testing"

	models "github.com/fireflg/go-musthave-metrics-tpl/internal/model"
	"github.com/fireflg/go-musthave-metrics-tpl/internal/observer"
	"github.com/fireflg/go-musthave-metrics-tpl/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type recordingObserver struct {
	single []string
	batch  [][]string
}

func (o *recordingObserver) Notify(_ context.Context, metricID string) {
	o.single = append(o.single, metricID)
}

func (o *recordingObserver) NotifyBatch(_ context.Context, metricIDs []string) {
	o.batch = append(o.batch, metricIDs)
}

func TestSetMetric_NotifiesObserver(t *testing.T) {
	repo := new(MockMetricsRepo)
	rec := &recordingObserver{}
	svc := service.NewMetricsService(repo, observer.Observers{rec})

	value := 1.0
	metric := models.Metrics{ID: "gauge1", MType: "gauge", Value: &value}
	repo.On("SetMetric", mock.Anything, metric).Return(nil)

	require.NoError(t, svc.SetMetric(context.Background(), metric))
	assert.Equal(t, []string{"gauge1"}, rec.single)

	repo.AssertExpectations(t)
}

func TestSetMetric_RepositoryErrorSkipsObserver(t *testing.T) {
	repo := new(MockMetricsRepo)
	rec := &recordingObserver{}
	svc := service.NewMetricsService(repo, observer.Observers{rec})

	metric := models.Metrics{ID: "gauge1", MType: "gauge"}
	repo.On("SetMetric", mock.Anything, metric).Return(errors.New("boom"))

	assert.ErrorContains(t, svc.SetMetric(context.Background(), metric), "boom")
	assert.Empty(t, rec.single)
}

func TestSetMetricBatch_NotifiesObserver(t *testing.T) {
	repo := new(MockMetricsRepo)
	rec := &recordingObserver{}
	svc := service.NewMetricsService(repo, observer.Observers{rec})

	delta := int64(1)
	metrics := []models.Metrics{
		{ID: "counter1", MType: "counter", Delta: &delta},
		{ID: "gauge1", MType: "gauge"},
	}
	repo.On("SetMetrics", mock.Anything, metrics).Return(nil)

	require.NoError(t, svc.SetMetricBatch(context.Background(), metrics))
	require.Len(t, rec.batch, 1)
	assert.Equal(t, []string{"counter1", "gauge1"}, rec.batch[0])
}

func TestSetMetricBatch_RepositoryError(t *testing.T) {
	repo := new(MockMetricsRepo)
	rec := &recordingObserver{}
	svc := service.NewMetricsService(repo, observer.Observers{rec})

	metrics := []models.Metrics{{ID: "gauge1", MType: "gauge"}}
	repo.On("SetMetrics", mock.Anything, metrics).Return(errors.New("batch failed"))

	assert.ErrorContains(t, svc.SetMetricBatch(context.Background(), metrics), "batch failed")
	assert.Empty(t, rec.batch)
}

func TestCheckRepository(t *testing.T) {
	repo := new(MockMetricsRepo)
	svc := service.NewMetricsService(repo, nil)

	repo.On("Ping", mock.Anything).Return(nil).Once()
	assert.NoError(t, svc.CheckRepository())

	repo.On("Ping", mock.Anything).Return(errors.New("db down")).Once()
	assert.ErrorContains(t, svc.CheckRepository(), "db down")

	repo.AssertExpectations(t)
}
