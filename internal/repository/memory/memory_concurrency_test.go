package memory_test

import (
	"context"
	"sync"
	"testing"

	models "github.com/fireflg/go-musthave-metrics-tpl/internal/model"
	"github.com/fireflg/go-musthave-metrics-tpl/internal/repository/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetMetrics_ReturnsErrorOnInvalidMetric(t *testing.T) {
	repo := memory.NewMemoryRepository()

	value := 1.0
	err := repo.SetMetrics(context.Background(), []models.Metrics{
		{ID: "valid", MType: models.Gauge, Value: &value},
		{ID: "broken", MType: "unknown"},
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "broken")
}

func TestSetMetrics_ReturnsErrorOnNilDelta(t *testing.T) {
	repo := memory.NewMemoryRepository()

	err := repo.SetMetrics(context.Background(), []models.Metrics{
		{ID: "counter", MType: models.Counter},
	})

	require.Error(t, err)
}

func TestSetMetric_RejectsEmptyID(t *testing.T) {
	repo := memory.NewMemoryRepository()

	value := 1.0
	err := repo.SetMetric(context.Background(), models.Metrics{MType: models.Gauge, Value: &value})

	require.Error(t, err)
}

func TestSetMetric_RespectsCanceledContext(t *testing.T) {
	repo := memory.NewMemoryRepository()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	value := 1.0
	err := repo.SetMetric(ctx, models.Metrics{ID: "gauge", MType: models.Gauge, Value: &value})

	require.Error(t, err)
}

func TestGetMetric_ReturnsID(t *testing.T) {
	repo := memory.NewMemoryRepository()

	value := 42.0
	require.NoError(t, repo.SetMetric(context.Background(), models.Metrics{
		ID: "gauge", MType: models.Gauge, Value: &value,
	}))

	got, err := repo.GetMetric(context.Background(), "gauge", models.Gauge)
	require.NoError(t, err)
	assert.Equal(t, "gauge", got.ID)
	require.NotNil(t, got.Value)
	assert.Equal(t, 42.0, *got.Value)
}

func TestGetAllMetrics_IsSnapshot(t *testing.T) {
	repo := memory.NewMemoryRepository()

	value := 1.0
	require.NoError(t, repo.SetMetric(context.Background(), models.Metrics{
		ID: "gauge", MType: models.Gauge, Value: &value,
	}))

	withAll, ok := repo.(interface {
		GetAllMetrics() map[string]models.Metrics
	})
	require.True(t, ok)

	snapshot := withAll.GetAllMetrics()
	require.Len(t, snapshot, 1)

	newValue := 2.0
	require.NoError(t, repo.SetMetric(context.Background(), models.Metrics{
		ID: "another", MType: models.Gauge, Value: &newValue,
	}))

	assert.Len(t, snapshot, 1, "снимок не должен меняться после новых записей")
	assert.Len(t, withAll.GetAllMetrics(), 2)
}

func TestGetAllMetrics_ConcurrentWithWrites(t *testing.T) {
	repo := memory.NewMemoryRepository()

	withAll, ok := repo.(interface {
		GetAllMetrics() map[string]models.Metrics
	})
	require.True(t, ok)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		value := 1.0
		for i := 0; i < 1000; i++ {
			_ = repo.SetMetric(context.Background(), models.Metrics{
				ID: "gauge", MType: models.Gauge, Value: &value,
			})
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			for range withAll.GetAllMetrics() { //nolint:revive
			}
		}
	}()

	wg.Wait()
}
