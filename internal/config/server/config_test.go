package server_test

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/fireflg/go-musthave-metrics-tpl/internal/config/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resetFlags() {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
}

func writeConfigFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
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
	assert.Equal(t, "", cfg.CryptoKeyPath)
	assert.Equal(t, "", cfg.TrustedSubnet)
	assert.Equal(t, "", cfg.GRPCAddr)
	assert.Equal(t, "memory", cfg.StorageMode)
}

func TestLoadAServerConfig_EnvVars(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()
	os.Args = []string{"cmd"}

	t.Setenv("ADDRESS", ":9090")
	t.Setenv("STORE_INTERVAL", "100")
	t.Setenv("STORE_FILE", "my_metrics.json")
	t.Setenv("RESTORE", "true")
	t.Setenv("DATABASE_DSN", "postgres://user:pass@localhost/db")
	t.Setenv("CRYPTO_KEY", "/path/to/private.pem")
	t.Setenv("TRUSTED_SUBNET", "192.168.0.0/24")
	t.Setenv("GRPC_ADDRESS", ":3200")

	resetFlags()

	cfg, err := server.LoadAServerConfig()
	assert.NoError(t, err)
	assert.Equal(t, ":9090", cfg.RunAddr)
	assert.Equal(t, 100, cfg.PersistentStorageInterval)
	assert.Equal(t, "my_metrics.json", cfg.PersistentStoragePath)
	assert.True(t, cfg.PersistentStorageRestore)
	assert.Equal(t, "postgres://user:pass@localhost/db", cfg.DatabaseDSN)
	assert.Equal(t, "/path/to/private.pem", cfg.CryptoKeyPath)
	assert.Equal(t, "192.168.0.0/24", cfg.TrustedSubnet)
	assert.Equal(t, ":3200", cfg.GRPCAddr)
	assert.Equal(t, "db", cfg.StorageMode)
}

func TestLoadAServerConfig_Flags(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()
	os.Args = []string{"cmd", "-a=:7070", "-i=50", "-f=flag_metrics.json", "-r=true", "-d=postgres://flag", "-crypto-key=/flag/key.pem", "-t=10.0.0.0/8", "-g=:3201"}

	resetFlags()

	cfg, err := server.LoadAServerConfig()
	assert.NoError(t, err)
	assert.Equal(t, ":7070", cfg.RunAddr)
	assert.Equal(t, 50, cfg.PersistentStorageInterval)
	assert.Equal(t, "flag_metrics.json", cfg.PersistentStoragePath)
	assert.True(t, cfg.PersistentStorageRestore)
	assert.Equal(t, "postgres://flag", cfg.DatabaseDSN)
	assert.Equal(t, "/flag/key.pem", cfg.CryptoKeyPath)
	assert.Equal(t, "10.0.0.0/8", cfg.TrustedSubnet)
	assert.Equal(t, ":3201", cfg.GRPCAddr)
}

func TestLoadAServerConfig_ConfigFile(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	path := writeConfigFile(t, `{
		"address": "localhost:8081",
		"restore": true,
		"store_interval": "5s",
		"store_file": "/path/to/file.db",
		"database_dsn": "postgres://file",
		"crypto_key": "/path/to/file_key.pem",
		"trusted_subnet": "172.16.0.0/12",
		"grpc_address": ":3202"
	}`)
	os.Args = []string{"cmd", "-c=" + path}

	resetFlags()

	cfg, err := server.LoadAServerConfig()
	assert.NoError(t, err)
	assert.Equal(t, "localhost:8081", cfg.RunAddr)
	assert.Equal(t, 5, cfg.PersistentStorageInterval)
	assert.Equal(t, "/path/to/file.db", cfg.PersistentStoragePath)
	assert.True(t, cfg.PersistentStorageRestore)
	assert.Equal(t, "postgres://file", cfg.DatabaseDSN)
	assert.Equal(t, "/path/to/file_key.pem", cfg.CryptoKeyPath)
	assert.Equal(t, "172.16.0.0/12", cfg.TrustedSubnet)
	assert.Equal(t, ":3202", cfg.GRPCAddr)
}

func TestLoadAServerConfig_ConfigFileViaEnv(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()
	os.Args = []string{"cmd"}

	path := writeConfigFile(t, `{"address": "localhost:9999"}`)
	t.Setenv("CONFIG", path)

	resetFlags()

	cfg, err := server.LoadAServerConfig()
	assert.NoError(t, err)
	assert.Equal(t, "localhost:9999", cfg.RunAddr)
}

func TestLoadAServerConfig_Priority(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	path := writeConfigFile(t, `{
		"address": "from-file:8080",
		"restore": false,
		"store_interval": "1s",
		"store_file": "from_file.json",
		"database_dsn": "from-file-dsn",
		"crypto_key": "from-file-key.pem"
	}`)

	os.Args = []string{"cmd", "-c=" + path, "-a=from-flag:8080"}
	t.Setenv("ADDRESS", "from-env:8080")

	resetFlags()

	cfg, err := server.LoadAServerConfig()
	require.NoError(t, err)
	assert.Equal(t, "from-flag:8080", cfg.RunAddr)

	assert.Equal(t, "from_file.json", cfg.PersistentStoragePath)
	require.NotEqual(t, "", cfg.DatabaseDSN)
}

func TestLoadAServerConfig_EnvBeatsFile(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	path := writeConfigFile(t, `{"store_file": "from_file.json"}`)
	os.Args = []string{"cmd", "-c=" + path}
	t.Setenv("STORE_FILE", "from_env.json")

	resetFlags()

	cfg, err := server.LoadAServerConfig()
	require.NoError(t, err)
	assert.Equal(t, "from_env.json", cfg.PersistentStoragePath)
}
