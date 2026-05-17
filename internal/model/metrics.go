package models

import "context"

// Metric type constants representing different metric kinds.
const (
	// Counter is a metric type that accumulates values over time.
	Counter = "counter"
	// Gauge is a metric type that represents a point-in-time value.
	Gauge = "gauge"
)

// Metrics represents a single metric with its value and metadata.
// Delta and Value are pointers to distinguish between zero values
// and unset values when JSON encoding.
//
// Example usage:
//
//	metric := models.Metrics{
//	    ID:    "memory_usage",
//	    MType: models.Gauge,
//	    Value: func() *float64 { v := 1024.0; return &v }(),
//	}
type Metrics struct {
	// ID is the unique identifier for the metric.
	ID string `json:"id"`
	// MType is the type of metric (counter or gauge).
	MType string `json:"type"`
	// Delta is the counter value (cumulative).
	Delta *int64 `json:"delta,omitempty"`
	// Value is the gauge value (point-in-time).
	Value *float64 `json:"value,omitempty"`
	// Hash is the HMAC signature for integrity verification.
	Hash string `json:"hash,omitempty"`
}

// MetricsRepository defines the interface for metrics storage operations.
// Implementations can use different backends (memory, file, database).
type MetricsRepository interface {
	// GetCounter retrieves a counter metric by name.
	GetCounter(ctx context.Context, name string) (int64, error)
	// SetCounter sets a counter metric value.
	SetCounter(ctx context.Context, name string, value int64) error
	// GetGauge retrieves a gauge metric by name.
	GetGauge(ctx context.Context, name string) (float64, error)
	// SetGauge sets a gauge metric value.
	SetGauge(ctx context.Context, name string, value float64) error
	// SetMetric sets a metric (gauge or counter) with automatic handling.
	SetMetric(ctx context.Context, metric Metrics) error
	// GetMetric retrieves a metric by ID and type.
	GetMetric(ctx context.Context, metricID, metricType string) (*Metrics, error)
	// SetMetrics sets multiple metrics in a batch operation.
	SetMetrics(ctx context.Context, metrics []Metrics) error
	// Ping checks the repository connectivity.
	Ping(ctx context.Context) error
}
