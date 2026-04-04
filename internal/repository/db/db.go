package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	models "github.com/fireflg/ago-musthave-metrics-tpl/internal/model"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
	"log"
	"time"
)

type PostgresRepository struct {
	DB *sql.DB
}

func NewPostgresRepository(dsn string) models.MetricsRepository {
	db, _ := sql.Open("pgx", dsn)

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := createMetricsTableIfNotExists(db); err != nil {
		log.Printf("Warning: failed to create metrics table: %v", err)
	}

	return &PostgresRepository{DB: db}
}

func (r *PostgresRepository) Close() error {
	return r.DB.Close()
}

func createMetricsTableIfNotExists(db *sql.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	query := `
    CREATE TABLE IF NOT EXISTS metrics (
        id    VARCHAR(255) PRIMARY KEY,
        type VARCHAR(255) NOT NULL,
        delta BIGINT,
        value DOUBLE PRECISION,
        hash  VARCHAR(64)
    )`
	_, err := db.ExecContext(ctx, query)
	return err
}

func (r *PostgresRepository) GetGauge(ctx context.Context, name string) (float64, error) {
	var value float64

	err := r.DB.QueryRowContext(ctx,
		`SELECT value FROM metrics WHERE id = $1 AND type = 'gauge'`,
		name,
	).Scan(&value)

	if err != nil {
		return 0, err
	}

	return value, nil
}

func (r *PostgresRepository) SetGauge(ctx context.Context, name string, value float64) error {
	_, err := r.DB.ExecContext(ctx,
		`INSERT INTO metrics (id, type, value) VALUES ($1, 'gauge', $2)
         ON CONFLICT (id) DO UPDATE SET value = $2`,
		name, value,
	)
	return err
}

func (r *PostgresRepository) Ping(ctx context.Context) error {
	return r.DB.PingContext(ctx)
}

func (r *PostgresRepository) GetCounter(ctx context.Context, name string) (int64, error) {
	var value int64

	err := r.DB.QueryRowContext(ctx,
		`SELECT delta FROM metrics WHERE id = $1 AND type = 'counter'`,
		name,
	).Scan(&value)

	if err != nil {
		return 0, err
	}

	return value, nil
}

func (r *PostgresRepository) SetCounter(ctx context.Context, name string, value int64) error {
	_, err := r.DB.ExecContext(ctx,
		`INSERT INTO metrics AS m (id, type, delta) VALUES ($1, 'counter', $2)
         ON CONFLICT (id) 
         DO UPDATE SET delta = m.delta + EXCLUDED.delta`,
		name, value,
	)

	return err
}
func (r *PostgresRepository) SetMetric(ctx context.Context, metric models.Metrics) error {
	if metric.ID == "" {
		return errors.New("metric ID is empty")
	}

	switch metric.MType {
	case "counter":
		if metric.Delta == nil {
			return errors.New("counter metric delta is nil")
		}

		res, err := r.DB.ExecContext(ctx, `
			UPDATE metrics
			SET delta = COALESCE(delta, 0) + $2
			WHERE id = $1 AND type = 'counter'
		`, metric.ID, *metric.Delta)
		if err != nil {
			return err
		}

		rowsAffected, _ := res.RowsAffected()
		if rowsAffected == 0 {
			_, err := r.DB.ExecContext(ctx, `
				INSERT INTO metrics (id, type, delta)
				VALUES ($1, 'counter', $2)
			`, metric.ID, *metric.Delta)
			if err != nil {
				return err
			}
		}

	case "gauge":
		if metric.Value == nil {
			return errors.New("gauge metric value is nil")
		}

		res, err := r.DB.ExecContext(ctx, `
			UPDATE metrics
			SET value = $2
			WHERE id = $1 AND type = 'gauge'
		`, metric.ID, *metric.Value)
		if err != nil {
			return err
		}

		rowsAffected, _ := res.RowsAffected()
		if rowsAffected == 0 {
			_, err := r.DB.ExecContext(ctx, `
				INSERT INTO metrics (id, type, value)
				VALUES ($1, 'gauge', $2)
			`, metric.ID, *metric.Value)
			if err != nil {
				return err
			}
		}

	default:
		return fmt.Errorf("unknown metric type: %s", metric.MType)
	}

	return nil
}

func (r *PostgresRepository) SetMetrics(ctx context.Context, metrics []models.Metrics) error {
	if len(metrics) == 0 {
		return nil
	}

	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, metric := range metrics {
		if metric.ID == "" {
			return errors.New("metric ID is empty")
		}

		switch metric.MType {
		case "counter":
			if metric.Delta == nil {
				return errors.New("counter metric delta is nil")
			}

			res, err := tx.ExecContext(ctx, `
				UPDATE metrics
				SET delta = COALESCE(delta, 0) + $2
				WHERE id = $1 AND type = 'counter'
			`, metric.ID, *metric.Delta)
			if err != nil {
				return err
			}

			rowsAffected, _ := res.RowsAffected()
			if rowsAffected == 0 {
				_, err := tx.ExecContext(ctx, `
					INSERT INTO metrics (id, type, delta)
					VALUES ($1, 'counter', $2)
				`, metric.ID, *metric.Delta)
				if err != nil {
					return err
				}
			}

		case "gauge":
			if metric.Value == nil {
				return errors.New("gauge metric value is nil")
			}

			res, err := tx.ExecContext(ctx, `
				UPDATE metrics
				SET value = $2
				WHERE id = $1 AND type = 'gauge'
			`, metric.ID, *metric.Value)
			if err != nil {
				return err
			}

			rowsAffected, _ := res.RowsAffected()
			if rowsAffected == 0 {
				_, err := tx.ExecContext(ctx, `
					INSERT INTO metrics (id, type, value)
					VALUES ($1, 'gauge', $2)
				`, metric.ID, *metric.Value)
				if err != nil {
					return err
				}
			}

		default:
			return fmt.Errorf("unknown metric type: %s", metric.MType)
		}
	}

	return tx.Commit()
}

func (r *PostgresRepository) GetMetric(ctx context.Context, metricID, metricType string) (*models.Metrics, error) {

	m := &models.Metrics{MType: metricType, ID: metricID}

	switch metricType {
	case "counter":
		var v int64
		err := r.DB.QueryRowContext(ctx, `
            SELECT delta FROM metrics WHERE id=$1 AND type='counter'
        `, metricID).Scan(&v)

		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, err
			}
			return nil, err
		}

		m.Delta = &v

	case "gauge":
		var v float64
		err := r.DB.QueryRowContext(ctx, `
            SELECT value FROM metrics WHERE id=$1 AND type='gauge'
        `, metricID).Scan(&v)

		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, err
			}
			return nil, err
		}

		m.Value = &v

	default:
		return nil, fmt.Errorf("unknown metric type: %s", metricType)
	}

	return m, nil
}
