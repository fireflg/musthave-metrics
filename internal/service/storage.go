package service

import (
	"encoding/json"
	"errors"
	"fmt"
	models "github.com/fireflg/ago-musthave-metrics-tpl/internal/model"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Storage struct {
	mutex           sync.Mutex
	StorageInterval int
	StorageRestore  bool
	StoragePath     string
	Metrics         map[string]models.Metric
}

type MetricsStorage interface {
	StoreMetrics() error
	RestoreMetrics() error
	UpdateCounterMetricValue(metricName string, metricValue float64) error
	UpdateGaugeMetricValue(metricName string, metricValue float64) error
	GetCounterMetricValue(metricName string) (float64, error)
	GetGaugeMetricValue(metricName string) (float64, error)
}

var _ MetricsStorage = (*Storage)(nil)

func (s *Storage) GetGaugeMetricValue(metricName string) (float64, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	metric, exists := s.Metrics[metricName]
	if !exists {
		return 0, errors.New("metric not found")
	}
	if s.Metrics[metricName].MType == "gauge" {
		return *metric.Value, nil
	} else {
		return 0, errors.New("found metric, but wrong type")
	}
}

func (s *Storage) GetCounterMetricValue(metricName string) (float64, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	metric, exists := s.Metrics[metricName]
	if !exists {
		return 0, errors.New("metric not found")
	}
	if s.Metrics[metricName].MType == "counter" {
		return float64(*metric.Delta), nil
	} else {
		return 0, errors.New("found metric, but wrong type")
	}
}

func (s *Storage) UpdateGaugeMetricValue(metricName string, metricValue float64) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	metric, exists := s.Metrics[metricName]

	if exists && metric.MType != "gauge" {
		return fmt.Errorf("metric '%s' exists but has type '%s', expected 'gauge'",
			metricName, metric.MType)
	}

	if !exists {
		metric = models.Metric{
			ID:    metricName,
			MType: "gauge",
		}
	}

	metric.Delta = nil
	metric.Value = &metricValue

	s.Metrics[metricName] = metric
	return nil
}

func (s *Storage) UpdateCounterMetricValue(metricName string, metricValue float64) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	metric, exists := s.Metrics[metricName]

	if exists && metric.MType != "counter" {
		return fmt.Errorf("metric '%s' exists but has type '%s', expected 'counter'",
			metricName, metric.MType)
	}

	if !exists {
		metric = models.Metric{
			ID:    metricName,
			MType: "counter",
		}
	}

	if metricValue != float64(int64(metricValue)) {
		return fmt.Errorf("counter '%s' cannot accept fractional value %f",
			metricName, metricValue)
	}

	var delta int64
	if metric.Delta != nil {
		delta = *metric.Delta
	}

	delta += int64(metricValue)

	metric.Delta = &delta

	s.Metrics[metricName] = metric
	return nil
}

func (s *Storage) InitStorage() {
	if s.StorageRestore {
		err := s.RestoreMetrics()
		if err != nil {
			return
		}
	}
	if s.StorageInterval > 0 {
		go s.startPeriodicSave()
	}
}

func (s *Storage) startPeriodicSave() {
	ticker := time.NewTicker(time.Duration(s.StorageInterval) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if err := s.StoreMetrics(); err != nil {
		}
	}
}

func (s *Storage) StoreMetrics() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	metricsArray := make([]models.Metric, 0, len(s.Metrics))
	for _, m := range s.Metrics {
		metricsArray = append(metricsArray, m)
	}

	data, err := json.MarshalIndent(metricsArray, "", "  ")
	if err != nil {
		return fmt.Errorf("StoreMetrics: marshal json: %w", err)
	}

	dir := filepath.Dir(s.StoragePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("StoreMetrics: mkdir: %w", err)
	}

	if err := os.WriteFile(s.StoragePath, data, 0644); err != nil {
		return fmt.Errorf("StoreMetrics: write file: %w", err)
	}

	return nil
}

func (s *Storage) RestoreMetrics() error {
	if s.StoragePath == "" {
		return nil
	}

	data, err := os.ReadFile(s.StoragePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("RestoreMetrics: read file: %w", err)
	}

	var metricsArray []models.Metric
	if err := json.Unmarshal(data, &metricsArray); err != nil {
		return fmt.Errorf("RestoreMetrics: unmarshal json: %w", err)
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.Metrics = make(map[string]models.Metric, len(metricsArray))
	for _, m := range metricsArray {
		s.Metrics[m.ID] = m
	}

	return nil
}
