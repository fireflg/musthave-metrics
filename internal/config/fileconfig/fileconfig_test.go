package fileconfig_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fireflg/go-musthave-metrics-tpl/internal/config/fileconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

func TestLoadServerConfig(t *testing.T) {
	path := writeFile(t, `{
		"address": "localhost:8080",
		"restore": true,
		"store_interval": "1s",
		"store_file": "/path/to/file.db",
		"database_dsn": "postgres://x",
		"crypto_key": "/path/to/key.pem"
	}`)

	cfg, err := fileconfig.LoadServerConfig(path)
	require.NoError(t, err)

	require.NotNil(t, cfg.Address)
	assert.Equal(t, "localhost:8080", *cfg.Address)
	require.NotNil(t, cfg.Restore)
	assert.True(t, *cfg.Restore)
	require.NotNil(t, cfg.StoreInterval)
	assert.Equal(t, 1, int(time.Duration(*cfg.StoreInterval).Seconds()))
	require.NotNil(t, cfg.StoreFile)
	assert.Equal(t, "/path/to/file.db", *cfg.StoreFile)
	require.NotNil(t, cfg.DatabaseDSN)
	assert.Equal(t, "postgres://x", *cfg.DatabaseDSN)
	require.NotNil(t, cfg.CryptoKey)
	assert.Equal(t, "/path/to/key.pem", *cfg.CryptoKey)
}

func TestLoadServerConfig_PartialFile(t *testing.T) {
	path := writeFile(t, `{"address": "localhost:9090"}`)

	cfg, err := fileconfig.LoadServerConfig(path)
	require.NoError(t, err)

	require.NotNil(t, cfg.Address)
	assert.Equal(t, "localhost:9090", *cfg.Address)
	assert.Nil(t, cfg.Restore)
	assert.Nil(t, cfg.StoreInterval)
	assert.Nil(t, cfg.StoreFile)
	assert.Nil(t, cfg.DatabaseDSN)
	assert.Nil(t, cfg.CryptoKey)
}

func TestLoadAgentConfig(t *testing.T) {
	path := writeFile(t, `{
		"address": "localhost:8080",
		"report_interval": "1s",
		"poll_interval": "2s",
		"crypto_key": "/path/to/key.pem"
	}`)

	cfg, err := fileconfig.LoadAgentConfig(path)
	require.NoError(t, err)

	require.NotNil(t, cfg.Address)
	assert.Equal(t, "localhost:8080", *cfg.Address)
	require.NotNil(t, cfg.ReportInterval)
	assert.Equal(t, 1, int(time.Duration(*cfg.ReportInterval).Seconds()))
	require.NotNil(t, cfg.PollInterval)
	assert.Equal(t, 2, int(time.Duration(*cfg.PollInterval).Seconds()))
	require.NotNil(t, cfg.CryptoKey)
	assert.Equal(t, "/path/to/key.pem", *cfg.CryptoKey)
}

func TestLoadServerConfig_FileNotFound(t *testing.T) {
	_, err := fileconfig.LoadServerConfig("/nonexistent/config.json")
	assert.Error(t, err)
}

func TestLoadServerConfig_InvalidJSON(t *testing.T) {
	path := writeFile(t, `not json`)

	_, err := fileconfig.LoadServerConfig(path)
	assert.Error(t, err)
}

func TestLoadServerConfig_InvalidDuration(t *testing.T) {
	path := writeFile(t, `{"store_interval": "not-a-duration"}`)

	_, err := fileconfig.LoadServerConfig(path)
	assert.Error(t, err)
}

func TestResolveString(t *testing.T) {
	fileVal := "from-file"

	assert.Equal(t, "from-flag", fileconfig.ResolveString(true, "from-flag", "UNSET_ENV_VAR_XYZ", &fileVal, "default"))

	t.Setenv("RESOLVE_STRING_TEST", "from-env")
	assert.Equal(t, "from-env", fileconfig.ResolveString(false, "", "RESOLVE_STRING_TEST", &fileVal, "default"))

	assert.Equal(t, "from-file", fileconfig.ResolveString(false, "", "UNSET_ENV_VAR_XYZ", &fileVal, "default"))
	assert.Equal(t, "default", fileconfig.ResolveString(false, "", "UNSET_ENV_VAR_XYZ", nil, "default"))
}

func TestResolveBool(t *testing.T) {
	fileVal := true

	assert.False(t, fileconfig.ResolveBool(true, false, "UNSET_ENV_VAR_XYZ", &fileVal, false))

	t.Setenv("RESOLVE_BOOL_TEST", "true")
	assert.True(t, fileconfig.ResolveBool(false, false, "RESOLVE_BOOL_TEST", nil, false))

	assert.True(t, fileconfig.ResolveBool(false, false, "UNSET_ENV_VAR_XYZ", &fileVal, false))
	assert.False(t, fileconfig.ResolveBool(false, false, "UNSET_ENV_VAR_XYZ", nil, false))
}

func TestResolveDurationSeconds(t *testing.T) {
	fileVal := fileconfig.Duration(3_000_000_000) // 3s

	assert.Equal(t, 5, fileconfig.ResolveDurationSeconds(true, 5, "UNSET_ENV_VAR_XYZ", &fileVal, 0))

	t.Setenv("RESOLVE_DURATION_TEST", "7")
	assert.Equal(t, 7, fileconfig.ResolveDurationSeconds(false, 0, "RESOLVE_DURATION_TEST", &fileVal, 0))

	assert.Equal(t, 3, fileconfig.ResolveDurationSeconds(false, 0, "UNSET_ENV_VAR_XYZ", &fileVal, 0))
	assert.Equal(t, 42, fileconfig.ResolveDurationSeconds(false, 0, "UNSET_ENV_VAR_XYZ", nil, 42))
}
