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
        delta INTEGER,
        value DOUBLE PRECISION,
        hash  VARCHAR(64)
    )`
	_, err := db.ExecContext(ctx, query)
	return err
}

func (r *PostgresRepository) GetGauge(ctx context.Context, name string) (float64, error) {
	const (
		maxRetries = 3
		retryDelay = 10 * time.Millisecond
	)

	var lastErr error

	for i := 0; i < maxRetries; i++ {
		var value float64

		row := r.DB.QueryRowContext(ctx,
			`SELECT value FROM metrics WHERE id = $1 AND type = 'gauge'`,
			name,
		)

		err := row.Scan(&value)
		if err == nil {
			return value, nil
		}

		if !errors.Is(err, sql.ErrNoRows) {
			return 0, err
		}

		lastErr = err

		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-time.After(retryDelay):
		}
	}

	return 0, lastErr
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
	const (
		maxRetries = 3
		retryDelay = 10 * time.Millisecond
	)

	var lastErr error

	for i := 0; i < maxRetries; i++ {
		var value int64

		row := r.DB.QueryRowContext(ctx,
			`SELECT m.delta FROM metrics m WHERE m.id = $1 AND m.type = 'counter'`,
			name,
		)

		err := row.Scan(&value)
		if err == nil {
			return value, nil
		}

		if !errors.Is(err, sql.ErrNoRows) {
			return 0, err
		}

		lastErr = err

		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-time.After(retryDelay):
		}
	}

	return 0, lastErr
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

		_, err := r.DB.ExecContext(ctx, `
			INSERT INTO metrics AS m (id, type, delta)
			VALUES ($1, 'counter', $2)
			ON CONFLICT (id)
			DO UPDATE SET delta = m.delta + EXCLUDED.delta
		`, metric.ID, *metric.Delta)
		return err

	case "gauge":
		if metric.Value == nil {
			return errors.New("gauge metric value is nil")
		}

		_, err := r.DB.ExecContext(ctx, `
			INSERT INTO metrics (id, type, value)
			VALUES ($1, 'gauge', $2)
			ON CONFLICT (id)
			DO UPDATE SET value = EXCLUDED.value
		`, metric.ID, *metric.Value)
		return err

	default:
		return fmt.Errorf("unknown metric type: %s", metric.MType)
	}
}
