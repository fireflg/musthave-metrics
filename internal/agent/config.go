package agent

import (
	"flag"
	"fmt"
	"strings"

	"github.com/caarlos0/env"
)

// Config содержит параметры конфигурации агента.
type Config struct {
	ServerURL      string `env:"ADDRESS" envDefault:"http://localhost:8080"` // ServerURL — эндпоинт сервера для отправки метрик.
	PollInterval   int    `env:"POLL_INTERVAL" envDefault:"0"`               // PollInterval — интервал в секундах для сбора метрик.
	ReportInterval int    `env:"REPORT_INTERVAL" envDefault:"10"`            // ReportInterval — интервал в секундах для отправки метрик.
	SecretKey      string `env:"KEY" envDefault:""`                          // SecretKey — HMAC ключ для подписи запросов.
	RateLimit      int    `env:"RATE_LIMIT" envDefault:"3"`                  // RateLimit — количество параллельных репортеров.
}

// LoadAgentConfig загружает конфигурацию агента из переменных окружения и флагов.
func LoadAgentConfig() (*Config, error) {
	var cfg Config

	err := env.Parse(&cfg)

	if err != nil {
		return nil, err
	}
	flag.StringVar(&cfg.ServerURL, "a", cfg.ServerURL, "Server address (default: from env or 'localhost:8080')")
	flag.IntVar(&cfg.PollInterval, "p", cfg.PollInterval, "Poll interval in seconds (default: from env or 10)")
	flag.IntVar(&cfg.ReportInterval, "r", cfg.ReportInterval, "Report interval in seconds (default: from env or 5)")
	flag.StringVar(&cfg.SecretKey, "k", cfg.SecretKey, "Hash key (default: env or 'key')")
	flag.IntVar(&cfg.RateLimit, "l", cfg.RateLimit, "Rate limit send to server")

	if unknownFlags := flag.Args(); len(unknownFlags) > 0 {
		return nil, fmt.Errorf("invalid flags: %v", unknownFlags)
	}

	flag.Parse()

	if !strings.Contains(cfg.ServerURL, "http://") {
		cfg.ServerURL = "http://" + cfg.ServerURL
	}

	return &cfg, nil
}