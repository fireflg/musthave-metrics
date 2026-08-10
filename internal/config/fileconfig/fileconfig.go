// Package fileconfig предоставляет загрузку конфигурации агента и сервера
// из JSON-файла
package fileconfig

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Duration оборачивает time.Duration/
type Duration time.Duration

// UnmarshalJSON разбирает строковое представление длительности.
func (d *Duration) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	parsed, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", s, err)
	}

	*d = Duration(parsed)
	return nil
}

// ServerConfig описывает поля файла конфигурации сервера.
type ServerConfig struct {
	Address       *string   `json:"address"`
	Restore       *bool     `json:"restore"`
	StoreInterval *Duration `json:"store_interval"`
	StoreFile     *string   `json:"store_file"`
	DatabaseDSN   *string   `json:"database_dsn"`
	CryptoKey     *string   `json:"crypto_key"`
}

// AgentConfig описывает поля файла конфигурации агента.
type AgentConfig struct {
	Address        *string   `json:"address"`
	ReportInterval *Duration `json:"report_interval"`
	PollInterval   *Duration `json:"poll_interval"`
	CryptoKey      *string   `json:"crypto_key"`
}

// LoadServerConfig читает и разбирает файл конфигурации сервера.
func LoadServerConfig(path string) (ServerConfig, error) {
	var cfg ServerConfig

	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("failed to parse config file: %w", err)
	}

	return cfg, nil
}

// LoadAgentConfig читает и разбирает файл конфигурации агента.
func LoadAgentConfig(path string) (AgentConfig, error) {
	var cfg AgentConfig

	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("failed to parse config file: %w", err)
	}

	return cfg, nil
}
