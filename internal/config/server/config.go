package server

import (
	"flag"
	"fmt"
	"github.com/caarlos0/env"
)

type Config struct {
	RunAddr                   string `env:"RUN_ADDR" envDefault:":8080"`
	PersistentStorageInterval int    `env:"STORAGE_INTERVAL" envDefault:"300"`
	PersistentStoragePath     string `env:"FILE_STORAGE_PATH" envDefault:"metrics.json"`
	PersistentStorageRestore  bool   `env:"RESTORE" envDefault:"false"`
	DatabaseDSN               string `env:"DATABASE_DSN" envDefault:""`
	StorageMode               string
}

func LoadAServerConfig() (*Config, error) {
	var cfg Config

	err := env.Parse(&cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to parse env vars: %w", err)
	}

	flag.StringVar(&cfg.RunAddr, "a", cfg.RunAddr, "Address and port to run server")
	flag.StringVar(&cfg.PersistentStoragePath, "f", cfg.PersistentStoragePath, "Path to store metrics")
	flag.IntVar(&cfg.PersistentStorageInterval, "i", cfg.PersistentStorageInterval, "Interval to store metrics in seconds (0 = sync save)")
	flag.BoolVar(&cfg.PersistentStorageRestore, "r", cfg.PersistentStorageRestore, "Whether to restore metrics")
	flag.StringVar(&cfg.DatabaseDSN, "d", cfg.DatabaseDSN, "Database connection string")
	flag.Parse()

	if unknownFlags := flag.Args(); len(unknownFlags) > 0 {
		return nil, fmt.Errorf("invalid flags: %v", unknownFlags)
	}

	switch {
	case cfg.DatabaseDSN != "":
		cfg.StorageMode = "db"
	case cfg.PersistentStoragePath != "" && cfg.PersistentStoragePath != "metrics.json":
		cfg.StorageMode = "file"
	default:
		cfg.StorageMode = "memory"
	}

	return &cfg, nil
}
