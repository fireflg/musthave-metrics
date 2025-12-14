package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	models "github.com/fireflg/ago-musthave-metrics-tpl/internal/model"
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
	SetMetric(metric models.Metric) error
	GetMetric(metricType string, metricName string) (string, error)
	CheckDBConn() error
}

func NewMertricsManager(cfg *Config, logger *zap.SugaredLogger, storage MetricsStorage) (*MetricManagerImpl, error) {
	return &MetricManagerImpl{
		cfg:     cfg,
		logger:  logger,
		storage: storage,
	}, nil
}

func (m *MetricManagerImpl) SetMetric(metric models.Metric) error {
	switch metric.MType {
	case "counter":
		if err := m.storage.UpdateCounterMetricValue(metric.ID, *metric.Delta); err != nil {
			return err
		}
	case "gauge":
		if err := m.storage.UpdateGaugeMetricValue(metric.ID, *metric.Value); err != nil {
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

func (m *MetricManagerImpl) GetMetric(metricType, metricName string) (*models.Metric, error) {
	switch metricType {
	case "counter":
		value, err := m.storage.GetCounterMetricValue(metricName)
		if err != nil {
			return nil, err
		}

		return &models.Metric{
			ID:    metricName,
			MType: "counter",
			Delta: &value,
		}, nil

	case "gauge":
		value, err := m.storage.GetGaugeMetricValue(metricName)
		if err != nil {
			return nil, err
		}

		return &models.Metric{
			ID:    metricName,
			MType: "gauge",
			Value: &value,
		}, nil

	default:
		return nil, fmt.Errorf("unknown metric type: %s", metricType)
	}
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
