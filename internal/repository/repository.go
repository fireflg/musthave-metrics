package repository

import (
	"errors"
	"github.com/fireflg/ago-musthave-metrics-tpl/internal/config/server"
	models "github.com/fireflg/ago-musthave-metrics-tpl/internal/model"
	"github.com/fireflg/ago-musthave-metrics-tpl/internal/repository/db"
	"github.com/fireflg/ago-musthave-metrics-tpl/internal/repository/file"
	"github.com/fireflg/ago-musthave-metrics-tpl/internal/repository/memory"
)

type StorageType string

const (
	StorageTypePostgres StorageType = "db"
	StorageTypeMemory   StorageType = "memory"
	StorageTypeFile     StorageType = "file"
)

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
