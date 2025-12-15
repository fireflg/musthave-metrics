package memory_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/fireflg/ago-musthave-metrics-tpl/internal/repository/memory"
)

func TestMemoryRepository_SetAndGetGauge(t *testing.T) {
	repo := memory.NewMemoryRepository()

	err := repo.SetGauge(context.Background(), "gauge1", 1.23)
	assert.NoError(t, err)

	val, err := repo.GetGauge(context.Background(), "gauge1")
	assert.NoError(t, err)
	assert.Equal(t, 1.23, val)
}

func TestMemoryRepository_SetAndGetCounter(t *testing.T) {
	repo := memory.NewMemoryRepository()

	err := repo.SetCounter(context.Background(), "counter1", 10)
	assert.NoError(t, err)

	err = repo.SetCounter(context.Background(), "counter1", 5)
	assert.NoError(t, err)

	val, err := repo.GetCounter(context.Background(), "counter1")
	assert.NoError(t, err)
	assert.Equal(t, int64(15), val)
}

func TestMemoryRepository_GetMissingMetric(t *testing.T) {
	repo := memory.NewMemoryRepository()

	_, err := repo.GetGauge(context.Background(), "unknown")
	assert.Error(t, err)

	_, err = repo.GetCounter(context.Background(), "unknown")
	assert.Error(t, err)
}
