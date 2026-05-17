// Package repository provides data storage implementations for metrics.
//
// The package implements the repository pattern to support different
// storage backends: memory, file, and PostgreSQL database.
package repository

import (
	"errors"

	"github.com/fireflg/go-musthave-metrics-tpl/internal/config/server"
	models "github.com/fireflg/go-musthave-metrics-tpl/internal/model"
	"github.com/fireflg/go-musthave-metrics-tpl/internal/repository/db"
	"github.com/fireflg/go-musthave-metrics-tpl/internal/repository/file"
	"github.com/fireflg/go-musthave-metrics-tpl/internal/repository/memory"
)

// StorageType represents the type of storage backend.
type StorageType string

// Storage type constants.
const (
	StorageTypePostgres StorageType = "db"     // PostgreSQL database storage.
	StorageTypeMemory   StorageType = "memory" // In-memory storage.
	StorageTypeFile     StorageType = "file"   // File-based storage.
)

// NewRepository creates a new repository instance based on the configuration.
// Returns an error if the storage mode is invalid.
func NewRepository(cfg server.Config) (models.MetricsRepository, error) {
	switch cfg.StorageMode {
	case string(StorageTypePostgres):
		return db.NewPostgresRepository(cfg.DatabaseDSN), nil
	case string(StorageTypeMemory):
		return memory.NewMemoryRepository(), nil
	case string(StorageTypeFile):
		return file.NewFileRepository(cfg.PersistentStoragePath, cfg.PersistentStorageInterval, cfg.PersistentStorageRestore), nil
	default:
		return nil, errors.New("invalid storage mode")
	}
}
