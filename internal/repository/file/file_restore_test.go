package file_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	models "github.com/fireflg/go-musthave-metrics-tpl/internal/model"
	"github.com/fireflg/go-musthave-metrics-tpl/internal/repository/file"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ptrFloat(v float64) *float64 { return &v }
func ptrInt(v int64) *int64       { return &v }

func TestFileRepository_SetMetricPersistsToDisk(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metrics.json")
	repo := file.NewFileRepository(path, 0, false)
	ctx := context.Background()

	require.NoError(t, repo.SetMetric(ctx, models.Metrics{ID: "g", MType: "gauge", Value: ptrFloat(3.5)}))

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var stored map[string]models.Metrics
	require.NoError(t, json.Unmarshal(data, &stored))
	require.Contains(t, stored, "g")
	assert.Equal(t, 3.5, *stored["g"].Value)
}

func TestFileRepository_SetMetricInvalid(t *testing.T) {
	repo := file.NewFileRepository(filepath.Join(t.TempDir(), "metrics.json"), 0, false)

	err := repo.SetMetric(context.Background(), models.Metrics{ID: "c", MType: "counter"})
	assert.Error(t, err, "counter без delta должен вернуть ошибку")
}

func TestFileRepository_GetMetric(t *testing.T) {
	repo := file.NewFileRepository(filepath.Join(t.TempDir(), "metrics.json"), 0, false)
	ctx := context.Background()

	require.NoError(t, repo.SetMetric(ctx, models.Metrics{ID: "c", MType: "counter", Delta: ptrInt(5)}))

	got, err := repo.GetMetric(ctx, "c", "counter")
	require.NoError(t, err)
	assert.Equal(t, "c", got.ID)
	assert.Equal(t, int64(5), *got.Delta)

	_, err = repo.GetMetric(ctx, "missing", "counter")
	assert.Error(t, err)
}

func TestFileRepository_SetMetricsPersistsBatch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metrics.json")
	repo := file.NewFileRepository(path, 0, false)
	ctx := context.Background()

	batch := []models.Metrics{
		{ID: "g", MType: "gauge", Value: ptrFloat(1)},
		{ID: "c", MType: "counter", Delta: ptrInt(2)},
	}
	require.NoError(t, repo.SetMetrics(ctx, batch))

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var stored map[string]models.Metrics
	require.NoError(t, json.Unmarshal(data, &stored))
	assert.Len(t, stored, 2)
}

func TestFileRepository_RestoreMetrics(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metrics.json")

	source := file.NewFileRepository(path, 0, false)
	ctx := context.Background()
	require.NoError(t, source.SetMetric(ctx, models.Metrics{ID: "g", MType: "gauge", Value: ptrFloat(7.5)}))
	require.NoError(t, source.SetMetric(ctx, models.Metrics{ID: "c", MType: "counter", Delta: ptrInt(3)}))

	restored := file.NewFileRepository(path, 0, true)

	gauge, err := restored.GetGauge(ctx, "g")
	require.NoError(t, err)
	assert.Equal(t, 7.5, gauge)

	counter, err := restored.GetCounter(ctx, "c")
	require.NoError(t, err)
	assert.Equal(t, int64(3), counter)
}

func TestFileRepository_RestoreMissingFileIsNotAnError(t *testing.T) {
	repo := file.NewFileRepository(filepath.Join(t.TempDir(), "absent.json"), 0, true)

	assert.NoError(t, repo.RestoreMetrics())
	assert.Empty(t, repo.GetAllMetrics())
}

func TestFileRepository_RestoreEmptyPath(t *testing.T) {
	repo := file.NewFileRepository("", 0, false)
	assert.NoError(t, repo.RestoreMetrics())
}

func TestFileRepository_RestoreBrokenJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metrics.json")
	require.NoError(t, os.WriteFile(path, []byte("{not json"), 0o644))

	repo := file.NewFileRepository(path, 0, false)
	assert.Error(t, repo.RestoreMetrics())
}

func TestFileRepository_StoreMetricsCreatesDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "dir", "metrics.json")
	repo := file.NewFileRepository(path, 0, false)

	require.NoError(t, repo.SetGauge(context.Background(), "g", 1))

	_, err := os.Stat(path)
	assert.NoError(t, err)
}

func TestFileRepository_PeriodicSave(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metrics.json")
	repo := file.NewFileRepository(path, 1, false)
	defer func() { _ = repo.Close() }()

	require.NoError(t, repo.SetGauge(context.Background(), "g", 42))

	_, err := os.Stat(path)
	require.True(t, os.IsNotExist(err), "при ненулевом интервале запись не должна быть синхронной")

	require.Eventually(t, func() bool {
		data, err := os.ReadFile(path)
		return err == nil && len(data) > 2
	}, 3*time.Second, 100*time.Millisecond)
}
