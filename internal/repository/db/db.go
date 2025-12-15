package db

import (
	"context"
	"database/sql"
	models "github.com/fireflg/ago-musthave-metrics-tpl/internal/model"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(dsn string) models.MetricsRepository {
	db, _ := sql.Open("pgx", dsn)

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) Close() error {
	return r.db.Close()
}

func (r *PostgresRepository) GetGauge(ctx context.Context, name string) (float64, error) {
	var value float64
	row := r.db.QueryRowContext(ctx,
		`SELECT value FROM metrics WHERE id = $1 AND type = 'gauge'`, name)
	err := row.Scan(&value)
	if err != nil {
		return 0, err
	}
	return value, nil
}

func (r *PostgresRepository) SetGauge(ctx context.Context, name string, value float64) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO metrics (id, type, value) VALUES ($1, 'gauge', $2)
         ON CONFLICT (id) DO UPDATE SET value = $2`,
		name, value,
	)
	return err
}

func (r *PostgresRepository) Ping(ctx context.Context) error {
	return r.db.PingContext(ctx)
}

func (r *PostgresRepository) GetCounter(ctx context.Context, name string) (int64, error) {
	var value int64
	row := r.db.QueryRowContext(ctx,
		`SELECT m.delta FROM metrics m WHERE m.id = $1 AND m.type = 'counter'`,
		name)
	err := row.Scan(&value)
	if err != nil {
		return 0, err
	}

	return value, nil
}

func (r *PostgresRepository) SetCounter(ctx context.Context, name string, value int64) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO metrics AS m (id, type, delta) VALUES ($1, 'counter', $2)
         ON CONFLICT (id) 
         DO UPDATE SET delta = m.delta + EXCLUDED.delta`,
		name, value,
	)

	return err
}
