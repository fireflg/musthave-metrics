package agent

import (
	"flag"
	"fmt"
	"github.com/caarlos0/env"
	"strings"
)

type Config struct {
	ServerURL      string `env:"ADDRESS" envDefault:"http://localhost:8080"`
	PollInterval   int    `env:"POLL_INTERVAL" envDefault:"2"`
	ReportInterval int    `env:"REPORT_INTERVAL" envDefault:"10"`
}

func LoadAgentConfig() (*Config, error) {
	var cfg Config

	err := env.Parse(&cfg)

	if err != nil {
		return nil, err
	}
	flag.StringVar(&cfg.ServerURL, "a", cfg.ServerURL, "Server address (default: from env or 'localhost:8080')")
	flag.IntVar(&cfg.PollInterval, "p", cfg.PollInterval, "Poll interval in seconds (default: from env or 10)")
	flag.IntVar(&cfg.ReportInterval, "r", cfg.ReportInterval, "Report interval in seconds (default: from env or 5)")

	if cfg.ServerURL != "" && !strings.HasPrefix(cfg.ServerURL, "http") {
		cfg.ServerURL = "http://" + cfg.ServerURL
	}

	if unknownFlags := flag.Args(); len(unknownFlags) > 0 {
		return nil, fmt.Errorf("invalid flags: %v", unknownFlags)
	}

	flag.Parse()

	return &cfg, nil
}
