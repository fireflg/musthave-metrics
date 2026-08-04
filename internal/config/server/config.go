// Package server предоставляет конфигурацию сервера для сервиса метрик.
package server

import (
	"flag"
	"fmt"
	"os"

	"github.com/caarlos0/env"
	"github.com/fireflg/go-musthave-metrics-tpl/internal/config/fileconfig"
)

// Config содержит параметры конфигурации сервера.
type Config struct {
	RunAddr                   string // RunAddr — адрес и порт сервера. env: ADDRESS, флаг: -a.
	PersistentStorageInterval int    // PersistentStorageInterval — интервал периодического сохранения в секундах (0 для синхронного). env: STORE_INTERVAL, флаг: -i.
	PersistentStoragePath     string // PersistentStoragePath — путь к файлу хранения метрик. env: STORE_FILE, флаг: -f.
	PersistentStorageRestore  bool   // PersistentStorageRestore — флаг восстановления метрик при старте. env: RESTORE, флаг: -r.
	DatabaseDSN               string // DatabaseDSN — строка подключения к базе данных. env: DATABASE_DSN, флаг: -d.
	HashKey                   string `env:"HASH_KEY" envDefault:""`   // HashKey — HMAC ключ для проверки подписи запросов.
	AuditFile                 string `env:"AUDIT_FILE" envDefault:""` // AuditFile — путь к файлу аудита.
	AuditURL                  string `env:"AUDIT_URL" envDefault:""`  // AuditURL — URL для отправки логов аудита.
	CryptoKeyPath             string // CryptoKeyPath — путь к файлу с приватным ключом для расшифровки запросов. env: CRYPTO_KEY, флаг: -crypto-key.
	ConfigPath                string // ConfigPath — путь к JSON-файлу конфигурации. env: CONFIG, флаг: -c/-config.
	StorageMode               string // StorageMode — активный тип хранилища (db, file, memory).
}

// LoadAServerConfig загружает конфигурацию сервера из файла конфигурации,
// переменных окружения и флагов.
func LoadAServerConfig() (*Config, error) {
	var cfg Config

	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("failed to parse env vars: %w", err)
	}

	flag.StringVar(&cfg.RunAddr, "a", "", "Address and port to run server")
	flag.StringVar(&cfg.PersistentStoragePath, "f", "", "Path to store metrics")
	flag.IntVar(&cfg.PersistentStorageInterval, "i", 0, "Interval to store metrics in seconds (0 = sync save)")
	flag.BoolVar(&cfg.PersistentStorageRestore, "r", false, "Whether to restore metrics")
	flag.StringVar(&cfg.DatabaseDSN, "d", "", "Database connection string")
	flag.StringVar(&cfg.CryptoKeyPath, "crypto-key", "", "Path to private key file for decryption")
	flag.StringVar(&cfg.ConfigPath, "c", "", "Path to JSON config file")
	flag.StringVar(&cfg.ConfigPath, "config", "", "Path to JSON config file (alias for -c)")

	flag.StringVar(&cfg.HashKey, "k", cfg.HashKey, "Hash key")
	flag.StringVar(&cfg.AuditFile, "audit-file", cfg.AuditFile, "Path to audit log file")
	flag.StringVar(&cfg.AuditURL, "audit-url", cfg.AuditURL, "URL to send audit logs")
	flag.Parse()

	if unknownFlags := flag.Args(); len(unknownFlags) > 0 {
		return nil, fmt.Errorf("invalid flags: %v", unknownFlags)
	}

	visited := map[string]bool{}
	flag.Visit(func(f *flag.Flag) { visited[f.Name] = true })

	if !visited["c"] && !visited["config"] {
		cfg.ConfigPath = os.Getenv("CONFIG")
	}

	var fileCfg fileconfig.ServerConfig
	if cfg.ConfigPath != "" {
		var err error
		fileCfg, err = fileconfig.LoadServerConfig(cfg.ConfigPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load config file: %w", err)
		}
	}

	cfg.RunAddr = fileconfig.ResolveString(visited["a"], cfg.RunAddr, "ADDRESS", fileCfg.Address, ":8080")
	cfg.PersistentStoragePath = fileconfig.ResolveString(visited["f"], cfg.PersistentStoragePath, "STORE_FILE", fileCfg.StoreFile, "metrics.json")
	cfg.PersistentStorageInterval = fileconfig.ResolveDurationSeconds(visited["i"], cfg.PersistentStorageInterval, "STORE_INTERVAL", fileCfg.StoreInterval, 0)
	cfg.PersistentStorageRestore = fileconfig.ResolveBool(visited["r"], cfg.PersistentStorageRestore, "RESTORE", fileCfg.Restore, false)
	cfg.DatabaseDSN = fileconfig.ResolveString(visited["d"], cfg.DatabaseDSN, "DATABASE_DSN", fileCfg.DatabaseDSN, "")
	cfg.CryptoKeyPath = fileconfig.ResolveString(visited["crypto-key"], cfg.CryptoKeyPath, "CRYPTO_KEY", fileCfg.CryptoKey, "")

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
