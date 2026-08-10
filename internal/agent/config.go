package agent

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fireflg/go-musthave-metrics-tpl/internal/config/fileconfig"
	"github.com/spf13/viper"
)

// Config содержит параметры конфигурации агента.
type Config struct {
	ServerURL      string
	PollInterval   int
	ReportInterval int
	SecretKey      string
	RateLimit      int
	CryptoKeyPath  string
	GRPCAddr       string
	ConfigPath     string
}

// LoadAgentConfig загружает конфигурацию агента из флагов, переменных
// окружения и JSON-файла конфигурации.
func LoadAgentConfig() (*Config, error) {
	var cfg Config

	flag.StringVar(&cfg.ServerURL, "a", "", "Server address (default: from env or 'localhost:8080')")
	flag.IntVar(&cfg.PollInterval, "p", 0, "Poll interval in seconds (default: from env or 2)")
	flag.IntVar(&cfg.ReportInterval, "r", 0, "Report interval in seconds (default: from env or 10)")
	flag.StringVar(&cfg.CryptoKeyPath, "crypto-key", "", "Path to public key file for encryption")
	flag.StringVar(&cfg.GRPCAddr, "g", "", "gRPC server address (empty = use HTTP transport)")
	flag.StringVar(&cfg.ConfigPath, "c", "", "Path to JSON config file")
	flag.StringVar(&cfg.ConfigPath, "config", "", "Path to JSON config file (alias for -c)")
	flag.StringVar(&cfg.SecretKey, "k", "", "Hash key (default: env or 'key')")
	flag.IntVar(&cfg.RateLimit, "l", 0, "Rate limit send to server")
	flag.Parse()

	if unknownFlags := flag.Args(); len(unknownFlags) > 0 {
		return nil, fmt.Errorf("invalid flags: %v", unknownFlags)
	}

	visited := map[string]bool{}
	flag.Visit(func(f *flag.Flag) { visited[f.Name] = true })

	if !visited["c"] && !visited["config"] {
		if v, ok := os.LookupEnv("CONFIG"); ok {
			cfg.ConfigPath = v
		}
	}

	v := viper.New()
	v.SetDefault("address", "http://localhost:8080")
	v.SetDefault("poll_interval", 2)
	v.SetDefault("report_interval", 10)
	v.SetDefault("crypto_key", "")
	v.SetDefault("grpc_address", "")
	v.SetDefault("key", "")
	v.SetDefault("rate_limit", 3)

	if cfg.ConfigPath != "" {
		fileCfg, err := fileconfig.LoadAgentConfig(cfg.ConfigPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load config file: %w", err)
		}
		applyAgentFileDefaults(v, fileCfg)
	}

	v.AutomaticEnv()

	cfg.ServerURL = resolveString(visited["a"], cfg.ServerURL, v, "address")
	cfg.PollInterval = resolveInt(visited["p"], cfg.PollInterval, v, "poll_interval")
	cfg.ReportInterval = resolveInt(visited["r"], cfg.ReportInterval, v, "report_interval")
	cfg.CryptoKeyPath = resolveString(visited["crypto-key"], cfg.CryptoKeyPath, v, "crypto_key")
	cfg.GRPCAddr = resolveString(visited["g"], cfg.GRPCAddr, v, "grpc_address")
	cfg.SecretKey = resolveString(visited["k"], cfg.SecretKey, v, "key")
	cfg.RateLimit = resolveInt(visited["l"], cfg.RateLimit, v, "rate_limit")

	if cfg.RateLimit < 1 {
		cfg.RateLimit = 1
	}

	if !strings.HasPrefix(cfg.ServerURL, "http://") && !strings.HasPrefix(cfg.ServerURL, "https://") {
		cfg.ServerURL = "http://" + cfg.ServerURL
	}

	return &cfg, nil
}

// applyAgentFileDefaults кладёт значения из файла в слой default
func applyAgentFileDefaults(v *viper.Viper, fileCfg fileconfig.AgentConfig) {
	if fileCfg.Address != nil {
		v.SetDefault("address", *fileCfg.Address)
	}
	if fileCfg.ReportInterval != nil {
		v.SetDefault("report_interval", int(time.Duration(*fileCfg.ReportInterval).Seconds()))
	}
	if fileCfg.PollInterval != nil {
		v.SetDefault("poll_interval", int(time.Duration(*fileCfg.PollInterval).Seconds()))
	}
	if fileCfg.CryptoKey != nil {
		v.SetDefault("crypto_key", *fileCfg.CryptoKey)
	}
	if fileCfg.GRPCAddress != nil {
		v.SetDefault("grpc_address", *fileCfg.GRPCAddress)
	}
}

func resolveString(flagSet bool, flagValue string, v *viper.Viper, key string) string {
	if flagSet {
		return flagValue
	}
	return v.GetString(key)
}

func resolveInt(flagSet bool, flagValue int, v *viper.Viper, key string) int {
	if flagSet {
		return flagValue
	}
	return v.GetInt(key)
}
