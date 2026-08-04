package agent

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/caarlos0/env"
	"github.com/fireflg/go-musthave-metrics-tpl/internal/config/fileconfig"
)

// Config содержит параметры конфигурации агента.
type Config struct {
	ServerURL      string // ServerURL — эндпоинт сервера для отправки метрик. env: ADDRESS, флаг: -a.
	PollInterval   int    // PollInterval — интервал в секундах для сбора метрик. env: POLL_INTERVAL, флаг: -p.
	ReportInterval int    // ReportInterval — интервал в секундах для отправки метрик. env: REPORT_INTERVAL, флаг: -r.
	SecretKey      string `env:"KEY" envDefault:""`         // SecretKey — HMAC ключ для подписи запросов.
	RateLimit      int    `env:"RATE_LIMIT" envDefault:"3"` // RateLimit — количество параллельных репортеров.
	CryptoKeyPath  string // CryptoKeyPath — путь к файлу с публичным ключом для шифрования запросов. env: CRYPTO_KEY, флаг: -crypto-key.
	ConfigPath     string // ConfigPath — путь к JSON-файлу конфигурации. env: CONFIG, флаг: -c/-config.
}

// LoadAgentConfig загружает конфигурацию агента из файла конфигурации,
// переменных окружения и флагов. Приоритет источников (от высшего к
// низшему): флаг, переменная окружения, файл конфигурации, значение по
// умолчанию.
func LoadAgentConfig() (*Config, error) {
	var cfg Config

	if err := env.Parse(&cfg); err != nil {
		return nil, err
	}

	flag.StringVar(&cfg.ServerURL, "a", "", "Server address (default: from env or 'localhost:8080')")
	flag.IntVar(&cfg.PollInterval, "p", 0, "Poll interval in seconds (default: from env or 10)")
	flag.IntVar(&cfg.ReportInterval, "r", 0, "Report interval in seconds (default: from env or 5)")
	flag.StringVar(&cfg.CryptoKeyPath, "crypto-key", "", "Path to public key file for encryption")
	flag.StringVar(&cfg.ConfigPath, "c", "", "Path to JSON config file")
	flag.StringVar(&cfg.ConfigPath, "config", "", "Path to JSON config file (alias for -c)")

	flag.StringVar(&cfg.SecretKey, "k", cfg.SecretKey, "Hash key (default: env or 'key')")
	flag.IntVar(&cfg.RateLimit, "l", cfg.RateLimit, "Rate limit send to server")

	flag.Parse()

	if unknownFlags := flag.Args(); len(unknownFlags) > 0 {
		return nil, fmt.Errorf("invalid flags: %v", unknownFlags)
	}

	visited := map[string]bool{}
	flag.Visit(func(f *flag.Flag) { visited[f.Name] = true })

	if !visited["c"] && !visited["config"] {
		cfg.ConfigPath = os.Getenv("CONFIG")
	}

	var fileCfg fileconfig.AgentConfig
	if cfg.ConfigPath != "" {
		var err error
		fileCfg, err = fileconfig.LoadAgentConfig(cfg.ConfigPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load config file: %w", err)
		}
	}

	cfg.ServerURL = fileconfig.ResolveString(visited["a"], cfg.ServerURL, "ADDRESS", fileCfg.Address, "http://localhost:8080")
	cfg.PollInterval = fileconfig.ResolveDurationSeconds(visited["p"], cfg.PollInterval, "POLL_INTERVAL", fileCfg.PollInterval, 0)
	cfg.ReportInterval = fileconfig.ResolveDurationSeconds(visited["r"], cfg.ReportInterval, "REPORT_INTERVAL", fileCfg.ReportInterval, 10)
	cfg.CryptoKeyPath = fileconfig.ResolveString(visited["crypto-key"], cfg.CryptoKeyPath, "CRYPTO_KEY", fileCfg.CryptoKey, "")

	if !strings.Contains(cfg.ServerURL, "http://") {
		cfg.ServerURL = "http://" + cfg.ServerURL
	}

	return &cfg, nil
}
