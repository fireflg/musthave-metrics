// Package repository предоставляет реализации хранилища данных для метрик.
//
// Пакет реализует паттерн репозитория для поддержки различных
// бэкендов хранения: memory, file и PostgreSQL база данных.
package repository

import (
	"errors"

	"github.com/fireflg/go-musthave-metrics-tpl/internal/config/server"
	models "github.com/fireflg/go-musthave-metrics-tpl/internal/model"
	"github.com/fireflg/go-musthave-metrics-tpl/internal/repository/db"
	"github.com/fireflg/go-musthave-metrics-tpl/internal/repository/file"
	"github.com/fireflg/go-musthave-metrics-tpl/internal/repository/memory"
)

// StorageType представляет тип бэкенда хранения.
type StorageType string

// Константы типов хранилища.
const (
	StorageTypePostgres StorageType = "db"     // Хранилище в PostgreSQL базе данных.
	StorageTypeMemory   StorageType = "memory" // Хранилище в памяти.
	StorageTypeFile     StorageType = "file"   // Файловый бэкенд.
)

// NewRepository создает новый экземпляр репозитория на основе конфигурации.
// Возвращает ошибку, если режим хранилища невалиден.
func NewRepository(cfg server.Config) (models.MetricsRepository, error) {
	switch cfg.StorageMode {
	case string(StorageTypePostgres):
		return db.NewPostgresRepository(cfg.DatabaseDSN)
	case string(StorageTypeMemory):
		return memory.NewMemoryRepository(), nil
	case string(StorageTypeFile):
		return file.NewFileRepository(cfg.PersistentStoragePath, cfg.PersistentStorageInterval, cfg.PersistentStorageRestore), nil
	default:
		return nil, errors.New("invalid storage mode")
	}
}