package file_test

import (
	"context"
	"path/filepath"
	"sync"
	"testing"

	models "github.com/fireflg/go-musthave-metrics-tpl/internal/model"
	"github.com/fireflg/go-musthave-metrics-tpl/internal/repository/file"
	"github.com/stretchr/testify/require"
)

func TestStoreMetrics_ConcurrentWithSetMetric(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metrics.json")
	repo := file.NewFileRepository(path, 0, false)
	t.Cleanup(func() { _ = repo.Close() })

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		value := 1.0
		for i := 0; i < 500; i++ {
			_ = repo.MemoryRepository.SetMetric(context.Background(), models.Metrics{
				ID: "gauge", MType: models.Gauge, Value: &value,
			})
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 500; i++ {
			require.NoError(t, repo.StoreMetrics())
		}
	}()

	wg.Wait()
}

func TestClose_Idempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metrics.json")
	repo := file.NewFileRepository(path, 1, false)

	require.NoError(t, repo.Close())
	require.NotPanics(t, func() {
		require.NoError(t, repo.Close())
	})
}

func TestClose_WithoutPeriodicSave(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metrics.json")
	repo := file.NewFileRepository(path, 0, false)

	require.NotPanics(t, func() {
		require.NoError(t, repo.Close())
	})
}

func TestSetMetrics_PropagatesError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metrics.json")
	repo := file.NewFileRepository(path, 0, false)
	t.Cleanup(func() { _ = repo.Close() })

	err := repo.SetMetrics(context.Background(), []models.Metrics{
		{ID: "broken", MType: "unknown"},
	})
	require.Error(t, err)
}
