package repository_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/fireflg/go-musthave-metrics-tpl/internal/config/server"
	"github.com/fireflg/go-musthave-metrics-tpl/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRepository_Memory(t *testing.T) {
	repo, err := repository.NewRepository(server.Config{StorageMode: "memory"})
	require.NoError(t, err)
	require.NotNil(t, repo)

	assert.NoError(t, repo.Ping(context.Background()))
}

func TestNewRepository_File(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metrics.json")

	repo, err := repository.NewRepository(server.Config{
		StorageMode:           "file",
		PersistentStoragePath: path,
	})
	require.NoError(t, err)
	require.NotNil(t, repo)

	require.NoError(t, repo.SetGauge(context.Background(), "g", 1.5))

	value, err := repo.GetGauge(context.Background(), "g")
	require.NoError(t, err)
	assert.Equal(t, 1.5, value)
}

func TestNewRepository_InvalidMode(t *testing.T) {
	_, err := repository.NewRepository(server.Config{StorageMode: "redis"})
	assert.ErrorContains(t, err, "invalid storage mode")
}

func TestNewRepository_PostgresUnreachable(t *testing.T) {
	repo, err := repository.NewRepository(server.Config{
		StorageMode: "db",
		DatabaseDSN: "postgres://user:pass@127.0.0.1:1/does-not-exist?connect_timeout=1",
	})
	require.NoError(t, err)
	require.NotNil(t, repo)

	assert.Error(t, repo.Ping(context.Background()))
}
