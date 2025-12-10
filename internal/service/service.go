package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"
	"time"
)

type MetricManagerImpl struct {
	cfg     *Config
	logger  *zap.SugaredLogger
	storage MetricsStorage
}

type MetricsManager interface {
	SetMetric(metricType string, metricName string, metricValue float64) error
	GetMetric(metricType string, metricName string) (float64, error)
	CheckDBConn() error
}

func NewMertricsManager(cfg *Config, logger *zap.SugaredLogger, storage MetricsStorage) (*MetricManagerImpl, error) {
	return &MetricManagerImpl{
		cfg:     cfg,
		logger:  logger,
		storage: storage,
	}, nil
}

func (m *MetricManagerImpl) SetMetric(metricType string, metricName string, metricValue float64) error {
	switch metricType {
	case "counter":
		if err := m.storage.UpdateCounterMetricValue(metricName, metricValue); err != nil {
			return err
		}
	case "gauge":
		if err := m.storage.UpdateGaugeMetricValue(metricName, metricValue); err != nil {
			return err
		}
	default:
		return errors.New("unknown metric type")
	}

	if m.cfg.PersistentStorageInterval == 0 {
		err := m.storage.StoreMetrics()
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *MetricManagerImpl) GetMetric(metricType string, metricName string) (float64, error) {
	getters := map[string]func(string) (float64, error){
		"gauge":   m.storage.GetGaugeMetricValue,
		"counter": m.storage.GetCounterMetricValue,
	}

	getFunc, ok := getters[metricType]
	if !ok {
		return 0, fmt.Errorf("unknown metric type: %s", metricType)
	}

	return getFunc(metricName)
}

func (m *MetricManagerImpl) CheckDBConn() error {
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second)
	defer cancel()

	db, err := sql.Open("pgx", m.cfg.DatabaseDSN)
	if err != nil {
		return fmt.Errorf("sql.Open error: %w", err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping error: %w", err)
	}

	return nil
}
