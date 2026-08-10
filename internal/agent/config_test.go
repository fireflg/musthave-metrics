package agent_test

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/fireflg/go-musthave-metrics-tpl/internal/agent"
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

func TestLoadAgentConfig_Defaults(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()
	os.Args = []string{"cmd"}

	resetFlags()

	cfg, err := agent.LoadAgentConfig()
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:8080", cfg.ServerURL)
	assert.Equal(t, 2, cfg.PollInterval)
	assert.Equal(t, 10, cfg.ReportInterval)
	assert.Equal(t, "", cfg.CryptoKeyPath)
	assert.Equal(t, "", cfg.GRPCAddr)
	assert.Equal(t, 3, cfg.RateLimit)
}

func TestLoadAgentConfig_HTTPSNotOverwritten(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()
	os.Args = []string{"cmd"}

	t.Setenv("ADDRESS", "https://example.com")

	resetFlags()

	cfg, err := agent.LoadAgentConfig()
	require.NoError(t, err)
	assert.Equal(t, "https://example.com", cfg.ServerURL)
}

func TestLoadAgentConfig_RateLimitClampedToOne(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()
	os.Args = []string{"cmd", "-l=0"}

	resetFlags()

	cfg, err := agent.LoadAgentConfig()
	require.NoError(t, err)
	assert.Equal(t, 1, cfg.RateLimit)
}

func TestLoadAgentConfig_EnvVars(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()
	os.Args = []string{"cmd"}

	t.Setenv("ADDRESS", "localhost:9090")
	t.Setenv("POLL_INTERVAL", "2")
	t.Setenv("REPORT_INTERVAL", "3")
	t.Setenv("CRYPTO_KEY", "/path/to/public.pem")
	t.Setenv("GRPC_ADDRESS", "localhost:3200")

	resetFlags()

	cfg, err := agent.LoadAgentConfig()
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:9090", cfg.ServerURL)
	assert.Equal(t, 2, cfg.PollInterval)
	assert.Equal(t, 3, cfg.ReportInterval)
	assert.Equal(t, "/path/to/public.pem", cfg.CryptoKeyPath)
	assert.Equal(t, "localhost:3200", cfg.GRPCAddr)
}

func TestLoadAgentConfig_Flags(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()
	os.Args = []string{"cmd", "-a=localhost:7070", "-p=4", "-r=5", "-crypto-key=/flag/public.pem", "-g=localhost:3201"}

	resetFlags()

	cfg, err := agent.LoadAgentConfig()
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:7070", cfg.ServerURL)
	assert.Equal(t, 4, cfg.PollInterval)
	assert.Equal(t, 5, cfg.ReportInterval)
	assert.Equal(t, "/flag/public.pem", cfg.CryptoKeyPath)
	assert.Equal(t, "localhost:3201", cfg.GRPCAddr)
}

func TestLoadAgentConfig_ConfigFile(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	path := writeConfigFile(t, `{
		"address": "localhost:8081",
		"report_interval": "6s",
		"poll_interval": "7s",
		"crypto_key": "/path/to/file_key.pem",
		"grpc_address": "localhost:3202"
	}`)

	os.Args = []string{"cmd", "-c=" + path}

	resetFlags()

	cfg, err := agent.LoadAgentConfig()
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:8081", cfg.ServerURL)
	assert.Equal(t, 7, cfg.PollInterval)
	assert.Equal(t, 6, cfg.ReportInterval)
	assert.Equal(t, "/path/to/file_key.pem", cfg.CryptoKeyPath)
	assert.Equal(t, "localhost:3202", cfg.GRPCAddr)
}

func TestLoadAgentConfig_ConfigFileViaEnv(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()
	os.Args = []string{"cmd"}

	path := writeConfigFile(t, `{"address": "localhost:9999"}`)
	t.Setenv("CONFIG", path)

	resetFlags()

	cfg, err := agent.LoadAgentConfig()
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:9999", cfg.ServerURL)
}

func TestLoadAgentConfig_Priority(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	path := writeConfigFile(t, `{
		"address": "from-file:8080",
		"report_interval": "1s",
		"poll_interval": "1s",
		"crypto_key": "from-file-key.pem"
	}`)

	os.Args = []string{"cmd", "-c=" + path, "-a=from-flag:8080"}
	t.Setenv("ADDRESS", "from-env:8080")
	t.Setenv("POLL_INTERVAL", "9")

	resetFlags()

	cfg, err := agent.LoadAgentConfig()
	require.NoError(t, err)
	assert.Equal(t, "http://from-flag:8080", cfg.ServerURL)

	assert.Equal(t, 9, cfg.PollInterval)

	assert.Equal(t, "from-file-key.pem", cfg.CryptoKeyPath)
}

func TestLoadAgentConfig_EnvBeatsFile(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	path := writeConfigFile(t, `{"report_interval": "1s"}`)
	os.Args = []string{"cmd", "-c=" + path}
	t.Setenv("REPORT_INTERVAL", "8")

	resetFlags()

	cfg, err := agent.LoadAgentConfig()
	require.NoError(t, err)
	assert.Equal(t, 8, cfg.ReportInterval)
}
