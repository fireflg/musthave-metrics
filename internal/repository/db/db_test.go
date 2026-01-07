package db_test

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/fireflg/ago-musthave-metrics-tpl/internal/model"
	"github.com/fireflg/ago-musthave-metrics-tpl/internal/repository/db"
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
		`SELECT delta FROM metrics WHERE id = $1 AND type = 'counter'`)).
		WithArgs("counter1").
		WillReturnRows(rows)

	val, err := repo.GetCounter(context.Background(), "counter1")
	assert.NoError(t, err)
	assert.Equal(t, int64(10), val)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSetMetricAndGetMetric(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer mockDB.Close()

	repo := &db.PostgresRepository{DB: mockDB}

	delta := int64(5)
	metricCounter := models.Metrics{ID: "counter1", MType: "counter", Delta: &delta}

	mock.ExpectExec(regexp.QuoteMeta(
		`UPDATE metrics
			SET delta = COALESCE(delta, 0) + $2
			WHERE id = $1 AND type = 'counter'`)).
		WithArgs("counter1", delta).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta(
		`INSERT INTO metrics (id, type, delta)
				VALUES ($1, 'counter', $2)`)).
		WithArgs("counter1", delta).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.SetMetric(context.Background(), metricCounter)
	assert.NoError(t, err)

	rows := sqlmock.NewRows([]string{"delta"}).AddRow(delta)
	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT delta FROM metrics WHERE id=$1 AND type='counter'`)).
		WithArgs("counter1").
		WillReturnRows(rows)

	got, err := repo.GetMetric(context.Background(), "counter1", "counter")
	assert.NoError(t, err)
	assert.NotNil(t, got.Delta)
	assert.Equal(t, delta, *got.Delta)

	value := 3.14
	metricGauge := models.Metrics{ID: "gauge1", MType: "gauge", Value: &value}

	mock.ExpectExec(regexp.QuoteMeta(
		`UPDATE metrics
			SET value = $2
			WHERE id = $1 AND type = 'gauge'`)).
		WithArgs("gauge1", value).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta(
		`INSERT INTO metrics (id, type, value)
				VALUES ($1, 'gauge', $2)`)).
		WithArgs("gauge1", value).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.SetMetric(context.Background(), metricGauge)
	assert.NoError(t, err)

	rowsGauge := sqlmock.NewRows([]string{"value"}).AddRow(value)
	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT value FROM metrics WHERE id=$1 AND type='gauge'`)).
		WithArgs("gauge1").
		WillReturnRows(rowsGauge)

	gotGauge, err := repo.GetMetric(context.Background(), "gauge1", "gauge")
	assert.NoError(t, err)
	assert.NotNil(t, gotGauge.Value)
	assert.Equal(t, value, *gotGauge.Value)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSetMetricsBatch(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer mockDB.Close()

	repo := &db.PostgresRepository{DB: mockDB}

	metrics := []models.Metrics{
		{ID: "counter1", MType: "counter", Delta: ptrInt64(10)},
		{ID: "gauge1", MType: "gauge", Value: ptrFloat64(2.71)},
	}

	mock.ExpectBegin()

	mock.ExpectExec(regexp.QuoteMeta(
		`UPDATE metrics
				SET delta = COALESCE(delta, 0) + $2
				WHERE id = $1 AND type = 'counter'`)).
		WithArgs("counter1", int64(10)).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta(
		`INSERT INTO metrics (id, type, delta)
					VALUES ($1, 'counter', $2)`)).
		WithArgs("counter1", int64(10)).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectExec(regexp.QuoteMeta(
		`UPDATE metrics
				SET value = $2
				WHERE id = $1 AND type = 'gauge'`)).
		WithArgs("gauge1", 2.71).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta(
		`INSERT INTO metrics (id, type, value)
					VALUES ($1, 'gauge', $2)`)).
		WithArgs("gauge1", 2.71).
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectCommit()

	err = repo.SetMetrics(context.Background(), metrics)
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func ptrInt64(v int64) *int64       { return &v }
func ptrFloat64(v float64) *float64 { return &v }

func TestPing(t *testing.T) {
	mockDB, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	assert.NoError(t, err)
	defer mockDB.Close()

	repo := &db.PostgresRepository{DB: mockDB}

	mock.ExpectPing().WillReturnError(nil)
	err = repo.Ping(context.Background())
	assert.NoError(t, err)
}

func TestGetMetricUnknownType(t *testing.T) {
	mockDB, _, err := sqlmock.New()
	assert.NoError(t, err)
	defer mockDB.Close()

	repo := &db.PostgresRepository{DB: mockDB}

	_, err = repo.GetMetric(context.Background(), "id1", "unknown")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown metric type")
}
