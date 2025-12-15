package file_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/fireflg/ago-musthave-metrics-tpl/internal/repository/file"
)

func TestFileRepository_SetAndGetGauge(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "metrics*.json")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	repo := file.NewFileRepository(tmpFile.Name(), 0, false)

	err = repo.SetGauge(context.Background(), "gauge1", 1.23)
	assert.NoError(t, err)

	val, err := repo.GetGauge(context.Background(), "gauge1")
	assert.NoError(t, err)
	assert.Equal(t, 1.23, val)
}

func TestFileRepository_SetAndGetCounter(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "metrics*.json")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	repo := file.NewFileRepository(tmpFile.Name(), 0, false)

	err = repo.SetCounter(context.Background(), "counter1", 10)
	assert.NoError(t, err)

	val, err := repo.GetCounter(context.Background(), "counter1")
	assert.NoError(t, err)
	assert.Equal(t, int64(10), val)
}

func TestFileRepository_Ping(t *testing.T) {
	repo := file.NewFileRepository("", 0, false)
	err := repo.Ping(context.Background())
	assert.NoError(t, err)
}
