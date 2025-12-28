package file

import (
	"context"
	"encoding/json"
	"fmt"
	models "github.com/fireflg/ago-musthave-metrics-tpl/internal/model"
	"github.com/fireflg/ago-musthave-metrics-tpl/internal/repository/memory"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type FileRepository struct {
	storageInterval int
	storageRestore  bool
	storagePath     string
	memory.MemoryRepository
	mu sync.Mutex
}

func NewFileRepository(
	storagePath string,
	storageInterval int,
	storageRestore bool,
) *FileRepository {
	repo := &FileRepository{
		storagePath:     storagePath,
		storageInterval: storageInterval,
		storageRestore:  storageRestore,
		MemoryRepository: memory.MemoryRepository{
			Metrics: make(map[string]models.Metrics),
		},
	}
	err := repo.InitStorage()
	if err != nil {
		log.Printf("Error initializing file repository: %v", err)
	}
	return repo
}

func (f *FileRepository) GetCounter(ctx context.Context, name string) (int64, error) {
	return f.MemoryRepository.GetCounter(ctx, name)
}

func (f *FileRepository) GetGauge(ctx context.Context, name string) (float64, error) {
	return f.MemoryRepository.GetGauge(ctx, name)
}

func (f *FileRepository) SetGauge(ctx context.Context, name string, value float64) error {
	if err := f.MemoryRepository.SetGauge(ctx, name, value); err != nil {
		return err
	}
	if f.storageInterval == 0 {
		err := f.StoreMetrics()
		if err != nil {
			return err
		}
	}
	return nil
}

func (f *FileRepository) SetCounter(ctx context.Context, name string, value int64) error {
	if err := f.MemoryRepository.SetCounter(ctx, name, value); err != nil {
		return err
	}
	if f.storageInterval == 0 {
		err := f.StoreMetrics()
		if err != nil {
			return err
		}
	}
	return nil
}

func (f *FileRepository) SetMetric(ctx context.Context, metric models.Metrics) error {
	if err := f.MemoryRepository.SetMetric(ctx, metric); err != nil {
		return err
	}
	if f.storageInterval == 0 {
		err := f.StoreMetrics()
		if err != nil {
			return err
		}
	}
	return nil
}

func (f *FileRepository) Ping(ctx context.Context) error {
	return f.MemoryRepository.Ping(ctx)
}

func (f *FileRepository) InitStorage() error {
	if f.storageRestore {
		if err := f.RestoreMetrics(); err != nil {
			return err
		}
	}
	if f.storageInterval > 0 {
		go f.startPeriodicSave()
	}
	return nil
}

func (f *FileRepository) startPeriodicSave() {
	ticker := time.NewTicker(time.Duration(f.storageInterval) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if err := f.StoreMetrics(); err != nil {
			return
		}
	}
}

func (f *FileRepository) StoreMetrics() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	metrics := f.MemoryRepository.GetAllMetrics()
	data, err := json.MarshalIndent(metrics, "", "  ")
	if err != nil {
		return fmt.Errorf("StoreMetrics: marshal json: %w", err)
	}

	dir := filepath.Dir(f.storagePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("StoreMetrics: mkdir: %w", err)
	}

	if err := os.WriteFile(f.storagePath, data, 0644); err != nil {
		return fmt.Errorf("StoreMetrics: write file: %w", err)
	}

	return nil
}

func (f *FileRepository) RestoreMetrics() error {
	ctx := context.Background()
	if f.storagePath == "" {
		return nil
	}

	data, err := os.ReadFile(f.storagePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("RestoreMetrics: read file: %w", err)
	}

	var metricsArray map[string]models.Metrics
	if err := json.Unmarshal(data, &metricsArray); err != nil {
		return fmt.Errorf("RestoreMetrics: unmarshal json: %w", err)
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	for _, metric := range metricsArray {
		err := f.MemoryRepository.SetMetric(ctx, metric)
		if err != nil {
			return err
		}
	}
	return nil
}
