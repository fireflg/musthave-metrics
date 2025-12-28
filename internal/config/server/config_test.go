package server_test

import (
	"flag"
	"github.com/fireflg/ago-musthave-metrics-tpl/internal/config/server"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func resetFlags() {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
}

func TestLoadAServerConfig_Defaults(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()
	os.Args = []string{"cmd"}

	resetFlags()

	cfg, err := server.LoadAServerConfig()
	assert.NoError(t, err)
	assert.Equal(t, ":8080", cfg.RunAddr)
	assert.Equal(t, 0, cfg.PersistentStorageInterval)
	assert.Equal(t, "metrics.json", cfg.PersistentStoragePath)
	assert.False(t, cfg.PersistentStorageRestore)
	assert.Equal(t, "", cfg.DatabaseDSN)
	assert.Equal(t, "memory", cfg.StorageMode)
}

func TestLoadAServerConfig_EnvVars(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()
	os.Args = []string{"cmd"}

	os.Setenv("ADDRESS", ":9090")
	os.Setenv("STORAGE_INTERVAL", "100")
	os.Setenv("FILE_STORAGE_PATH", "my_metrics.json")
	os.Setenv("RESTORE", "true")
	os.Setenv("DATABASE_DSN", "postgres://user:pass@localhost/db")
	defer func() {
		os.Unsetenv("ADDRESS")
		os.Unsetenv("STORAGE_INTERVAL")
		os.Unsetenv("FILE_STORAGE_PATH")
		os.Unsetenv("RESTORE")
		os.Unsetenv("DATABASE_DSN")
	}()

	resetFlags()

	cfg, err := server.LoadAServerConfig()
	assert.NoError(t, err)
	assert.Equal(t, ":9090", cfg.RunAddr)
	assert.Equal(t, 100, cfg.PersistentStorageInterval)
	assert.Equal(t, "my_metrics.json", cfg.PersistentStoragePath)
	assert.True(t, cfg.PersistentStorageRestore)
	assert.Equal(t, "postgres://user:pass@localhost/db", cfg.DatabaseDSN)
	assert.Equal(t, "db", cfg.StorageMode)
}
