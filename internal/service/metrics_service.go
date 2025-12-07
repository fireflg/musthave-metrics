package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	_ "github.com/jackc/pgx/v5/stdlib"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	models "github.com/fireflg/ago-musthave-metrics-tpl/internal/model"
)

type MetricsService interface {
	SetMetric(metricType string, metricName string, metricValue float64) error
	GetMetric(metricType string, metricName string) (value float64, err error)
	DecodeAndSetMetric(r *http.Request) error
	DecodeAndGetMetric(r *http.Request) ([]byte, error)
	CheckDBConn() error
}

type MetricsStorage struct {
	Metrics map[string]models.Metrics
	mutex   sync.Mutex
	DBDSN   string
}

var _ MetricsService = (*MetricsStorage)(nil)

func (m *MetricsStorage) SetMetric(metricType string, metricName string, metricValue float64) error {

	if err := checkMetricType(metricType); err != nil {
		return err
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	metric, exists := m.Metrics[metricName]
	if !exists {
		metric = models.Metrics{
			ID:    metricName,
			MType: metricType,
		}
	}

	switch metricType {
	case "counter":
		var delta int64
		if metric.Delta != nil {
			delta = *metric.Delta
		}
		delta += int64(metricValue)
		val := float64(delta)
		metric.Delta = &delta
		metric.Value = &val
	case "gauge":
		metric.Delta = nil
		metric.Value = &metricValue
	}

	m.Metrics[metricName] = metric
	return nil
}

func (m *MetricsStorage) GetMetric(metricType string, metricName string) (float64, error) {
	if err := checkMetricType(metricType); err != nil {
		return 0, err
	}
	m.mutex.Lock()
	defer m.mutex.Unlock()

	metric, exists := m.Metrics[metricName]
	if !exists {
		return 0, errors.New("metric not found")
	}

	switch metricType {
	case "gauge":
		if metric.Value == nil {
			return 0, errors.New("gauge value is nil")
		}
		return *metric.Value, nil
	case "counter":
		if metric.Delta == nil {
			return 0, errors.New("counter delta is nil")
		}
		return float64(*metric.Delta), nil
	default:
		return 0, errors.New("unknown metric type")
	}
}

func (m *MetricsStorage) DecodeAndSetMetric(r *http.Request) error {
	var metric models.Metrics
	if err := json.NewDecoder(r.Body).Decode(&metric); err != nil {
		return err
	}

	switch metric.MType {
	case "gauge":
		if metric.Value == nil {
			return errors.New("value required for gauge")
		}
		return m.SetMetric("gauge", metric.ID, *metric.Value)
	case "counter":
		if metric.Delta == nil {
			return errors.New("delta required for counter")
		}
		return m.SetMetric("counter", metric.ID, float64(*metric.Delta))
	default:
		return errors.New("unknown metric type")
	}
}

func (m *MetricsStorage) DecodeAndGetMetric(r *http.Request) ([]byte, error) {
	var metric models.Metrics
	if err := json.NewDecoder(r.Body).Decode(&metric); err != nil {
		return nil, err
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()
	stored, ok := m.Metrics[metric.ID]
	if !ok {
		return nil, errors.New("metric not found")
	}

	resp := map[string]interface{}{
		"id":   stored.ID,
		"type": stored.MType,
	}

	switch stored.MType {
	case "gauge":
		if stored.Value == nil {
			return nil, errors.New("gauge value is nil")
		}
		resp["value"] = *stored.Value
	case "counter":
		if stored.Delta == nil {
			return nil, errors.New("counter delta is nil")
		}
		resp["delta"] = *stored.Delta
	default:
		return nil, errors.New("unknown metric type")
	}

	return json.Marshal(resp)
}

func checkMetricType(metricType string) error {
	if metricType != "gauge" && metricType != "counter" {
		return errors.New("invalid metric type. Use 'gauge' or 'counter'")
	}
	return nil
}

func NewMetricsService(dbDSN string) *MetricsStorage {
	return &MetricsStorage{
		Metrics: make(map[string]models.Metrics),
		DBDSN:   dbDSN,
	}
}

func (m *MetricsStorage) RestoreMetrics(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("RestoreMetrics: file not found %s, skipping restore", filePath)
			return nil
		}
		return fmt.Errorf("RestoreMetrics: read file: %w", err)
	}

	var metricsArray []models.Metrics
	if err := json.Unmarshal(data, &metricsArray); err != nil {
		return fmt.Errorf("RestoreMetrics: json unmarshal: %w", err)
	}

	m.Metrics = make(map[string]models.Metrics, len(metricsArray))

	for _, metric := range metricsArray {
		m.Metrics[metric.ID] = metric
	}
	log.Printf("RestoreMetrics: restored %d metrics from %s", len(metricsArray), filePath)
	return nil
}

func (m *MetricsStorage) SaveMetrics(ctx context.Context, filePath string, storeInterval int) error {
	if storeInterval < 0 {
		return fmt.Errorf("storeInterval must be >= 0, got %d", storeInterval)
	}

	if storeInterval == 0 {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				if err := m.saveOnce(filePath); err != nil {
					return err
				}
			}
		}
	}

	storeTicker := time.NewTicker(time.Duration(storeInterval) * time.Second)
	defer storeTicker.Stop()

	for {
		select {
		case <-storeTicker.C:
			if err := m.saveOnce(filePath); err != nil {
				return err
			}

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (m *MetricsStorage) saveOnce(filePath string) error {
	m.mutex.Lock()

	metricsSlice := make([]map[string]interface{}, 0, len(m.Metrics))
	for _, metric := range m.Metrics {
		item := map[string]interface{}{
			"id":   metric.ID,
			"type": metric.MType,
		}

		switch metric.MType {
		case "counter":
			item["delta"] = metric.Delta
		case "gauge":
			item["value"] = metric.Value
		}
		metricsSlice = append(metricsSlice, item)
	}

	data, err := json.MarshalIndent(metricsSlice, "", "  ")
	m.mutex.Unlock()
	if err != nil {
		return fmt.Errorf("json marshal failed: %w", err)
	}

	if err := m.saveMetricsToFile(filePath, data); err != nil {
		return fmt.Errorf("write failed: %w", err)
	}

	return nil
}

func (m *MetricsStorage) saveMetricsToFile(filePath string, data []byte) error {
	if filePath == "" {
		return fmt.Errorf("filePath is empty")
	}

	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	fd, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer fd.Close()

	_, err = fd.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write file %s: %w", filePath, err)
	}

	return nil
}

func (m *MetricsStorage) CheckDBConn() error {
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second)
	defer cancel()

	db, err := sql.Open("pgx", m.DBDSN)
	if err != nil {
		return fmt.Errorf("sql.Open error: %w", err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping error: %w", err)
	}

	return nil
}
