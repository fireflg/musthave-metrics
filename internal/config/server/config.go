// Package server provides server configuration for the metrics service.
package server

import (
	"flag"
	"fmt"

	"github.com/caarlos0/env"
)

// Config holds server configuration parameters.
type Config struct {
	RunAddr                   string `env:"ADDRESS" envDefault:":8080"`                  // RunAddr is the server address and port.
	PersistentStorageInterval int    `env:"STORAGE_INTERVAL" envDefault:"0"`             // PersistentStorageInterval is the interval for periodic storage saves (0 for sync).
	PersistentStoragePath     string `env:"FILE_STORAGE_PATH" envDefault:"metrics.json"` // PersistentStoragePath is the path to the metrics storage file.
	PersistentStorageRestore  bool   `env:"RESTORE" envDefault:"false"`                  // PersistentStorageRestore indicates whether to restore metrics on startup.
	DatabaseDSN               string `env:"DATABASE_DSN" envDefault:""`                  // DatabaseDSN is the database connection string.
	HashKey                   string `env:"HASH_KEY" envDefault:""`                      // HashKey is the HMAC key for request signature verification.
	AuditFile                 string `env:"AUDIT_FILE" envDefault:""`                    // AuditFile is the path to the audit log file.
	AuditURL                  string `env:"AUDIT_URL" envDefault:""`                     // AuditURL is the URL to send audit logs to.
	StorageMode               string // StorageMode is the active storage type (db, file, memory).
}

// LoadAServerConfig loads server configuration from environment variables and flags.
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
	flag.StringVar(&cfg.HashKey, "k", cfg.HashKey, "Hash key")
	flag.StringVar(&cfg.AuditFile, "audit-file", cfg.AuditFile, "Path to audit log file")
	flag.StringVar(&cfg.AuditURL, "audit-url", cfg.AuditURL, "URL to send audit logs")
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
