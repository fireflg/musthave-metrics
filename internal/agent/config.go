package agent

import (
	"flag"
	"fmt"
	"strings"

	"github.com/caarlos0/env"
)

// Config holds agent configuration parameters.
type Config struct {
	ServerURL      string `env:"ADDRESS" envDefault:"http://localhost:8080"` // ServerURL is the server endpoint to report metrics to.
	PollInterval   int    `env:"POLL_INTERVAL" envDefault:"0"`               // PollInterval is the interval in seconds to collect metrics.
	ReportInterval int    `env:"REPORT_INTERVAL" envDefault:"10"`             // ReportInterval is the interval in seconds to report metrics.
	SecretKey      string `env:"KEY" envDefault:""`                           // SecretKey is the HMAC key for request signing.
	RateLimit      int    `env:"RATE_LIMIT" envDefault:"3"`                  // RateLimit is the number of concurrent reporters.
}

// LoadAgentConfig loads agent configuration from environment variables and flags.
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
