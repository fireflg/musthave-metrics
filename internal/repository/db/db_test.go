package db_test

import (
	"context"
	"github.com/fireflg/ago-musthave-metrics-tpl/internal/repository/db"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestSetAndGetGauge(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer mockDB.Close()

	repo := &db.PostgresRepository{DB: mockDB}

	mock.ExpectExec(regexp.QuoteMeta(
		`INSERT INTO metrics (id, type, value) VALUES ($1, 'gauge', $2)
         ON CONFLICT (id) DO UPDATE SET value = $2`)).
		WithArgs("gauge1", 1.23).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.SetGauge(context.Background(), "gauge1", 1.23)
	assert.NoError(t, err)

	rows := sqlmock.NewRows([]string{"value"}).AddRow(1.23)
	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT value FROM metrics WHERE id = $1 AND type = 'gauge'`)).
		WithArgs("gauge1").
		WillReturnRows(rows)

	val, err := repo.GetGauge(context.Background(), "gauge1")
	assert.NoError(t, err)
	assert.Equal(t, 1.23, val)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSetAndGetCounter(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer mockDB.Close()

	repo := &db.PostgresRepository{DB: mockDB}

	mock.ExpectExec(regexp.QuoteMeta(
		`INSERT INTO metrics AS m (id, type, delta) VALUES ($1, 'counter', $2)
         ON CONFLICT (id) 
         DO UPDATE SET delta = m.delta + EXCLUDED.delta`)).
		WithArgs("counter1", int64(10)).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.SetCounter(context.Background(), "counter1", 10)
	assert.NoError(t, err)

	rows := sqlmock.NewRows([]string{"delta"}).AddRow(10)
	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT m.delta FROM metrics m WHERE m.id = $1 AND m.type = 'counter'`)).
		WithArgs("counter1").
		WillReturnRows(rows)

	val, err := repo.GetCounter(context.Background(), "counter1")
	assert.NoError(t, err)
	assert.Equal(t, int64(10), val)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPing(t *testing.T) {
	mockDB, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	assert.NoError(t, err)
	defer mockDB.Close()

	repo := &db.PostgresRepository{DB: mockDB}
	mock.ExpectPing().WillReturnError(nil)
	err = repo.Ping(context.Background())
	assert.NoError(t, err)
}
