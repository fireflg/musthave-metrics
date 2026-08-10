// Package server предоставляет конфигурацию сервера для сервиса метрик.
package server

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/fireflg/go-musthave-metrics-tpl/internal/config/fileconfig"
	"github.com/spf13/viper"
)

// Config содержит параметры конфигурации сервера.
type Config struct {
	RunAddr                   string
	PersistentStorageInterval int
	PersistentStoragePath     string
	PersistentStorageRestore  bool
	DatabaseDSN               string
	HashKey                   string
	AuditFile                 string
	AuditURL                  string
	CryptoKeyPath             string
	ConfigPath                string
	StorageMode               string
}

// LoadAServerConfig загружает конфигурацию сервера из флагов, переменных
// окружения и JSON-файла конфигурации.
func LoadAServerConfig() (*Config, error) {
	var cfg Config

	flag.StringVar(&cfg.RunAddr, "a", "", "Address and port to run server")
	flag.StringVar(&cfg.PersistentStoragePath, "f", "", "Path to store metrics")
	flag.IntVar(&cfg.PersistentStorageInterval, "i", 0, "Interval to store metrics in seconds (0 = sync save)")
	flag.BoolVar(&cfg.PersistentStorageRestore, "r", false, "Whether to restore metrics")
	flag.StringVar(&cfg.DatabaseDSN, "d", "", "Database connection string")
	flag.StringVar(&cfg.CryptoKeyPath, "crypto-key", "", "Path to private key file for decryption")
	flag.StringVar(&cfg.ConfigPath, "c", "", "Path to JSON config file")
	flag.StringVar(&cfg.ConfigPath, "config", "", "Path to JSON config file (alias for -c)")
	flag.StringVar(&cfg.HashKey, "k", "", "Hash key")
	flag.StringVar(&cfg.AuditFile, "audit-file", "", "Path to audit log file")
	flag.StringVar(&cfg.AuditURL, "audit-url", "", "URL to send audit logs")
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
	v.SetDefault("address", ":8080")
	v.SetDefault("store_file", "metrics.json")
	v.SetDefault("store_interval", 0)
	v.SetDefault("restore", false)
	v.SetDefault("database_dsn", "")
	v.SetDefault("crypto_key", "")
	v.SetDefault("hash_key", "")
	v.SetDefault("audit_file", "")
	v.SetDefault("audit_url", "")

	if cfg.ConfigPath != "" {
		fileCfg, err := fileconfig.LoadServerConfig(cfg.ConfigPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load config file: %w", err)
		}
		applyServerFileDefaults(v, fileCfg)
	}

	v.AutomaticEnv()

	cfg.RunAddr = resolveString(visited["a"], cfg.RunAddr, v, "address")
	cfg.PersistentStoragePath = resolveString(visited["f"], cfg.PersistentStoragePath, v, "store_file")
	cfg.PersistentStorageInterval = resolveInt(visited["i"], cfg.PersistentStorageInterval, v, "store_interval")
	cfg.PersistentStorageRestore = resolveBool(visited["r"], cfg.PersistentStorageRestore, v, "restore")
	cfg.DatabaseDSN = resolveString(visited["d"], cfg.DatabaseDSN, v, "database_dsn")
	cfg.CryptoKeyPath = resolveString(visited["crypto-key"], cfg.CryptoKeyPath, v, "crypto_key")
	cfg.HashKey = resolveString(visited["k"], cfg.HashKey, v, "hash_key")
	cfg.AuditFile = resolveString(visited["audit-file"], cfg.AuditFile, v, "audit_file")
	cfg.AuditURL = resolveString(visited["audit-url"], cfg.AuditURL, v, "audit_url")

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

// applyServerFileDefaults кладёт значения из файла в слой default
func applyServerFileDefaults(v *viper.Viper, fileCfg fileconfig.ServerConfig) {
	if fileCfg.Address != nil {
		v.SetDefault("address", *fileCfg.Address)
	}
	if fileCfg.Restore != nil {
		v.SetDefault("restore", *fileCfg.Restore)
	}
	if fileCfg.StoreInterval != nil {
		v.SetDefault("store_interval", int(time.Duration(*fileCfg.StoreInterval).Seconds()))
	}
	if fileCfg.StoreFile != nil {
		v.SetDefault("store_file", *fileCfg.StoreFile)
	}
	if fileCfg.DatabaseDSN != nil {
		v.SetDefault("database_dsn", *fileCfg.DatabaseDSN)
	}
	if fileCfg.CryptoKey != nil {
		v.SetDefault("crypto_key", *fileCfg.CryptoKey)
	}
}

func resolveString(flagSet bool, flagValue string, v *viper.Viper, key string) string {
	if flagSet {
		return flagValue
	}
	return v.GetString(key)
}

func resolveBool(flagSet bool, flagValue bool, v *viper.Viper, key string) bool {
	if flagSet {
		return flagValue
	}
	return v.GetBool(key)
}

func resolveInt(flagSet bool, flagValue int, v *viper.Viper, key string) int {
	if flagSet {
		return flagValue
	}
	return v.GetInt(key)
}
