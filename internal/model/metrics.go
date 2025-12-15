package models

import "context"

const (
	Counter = "counter"
	Gauge   = "gauge"
)

// NOTE: Не усложняем пример, вводя иерархическую вложенность структур.
// Органичиваясь плоской моделью.
// Delta и Value объявлены через указатели,
// что бы отличать значение "0", от не заданного значения
// и соответственно не кодировать в структуру.
//
//	type Metrics struct {
//		ID string `json:"id"`
//	}
type Metrics struct {
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
	Hash  string   `json:"hash,omitempty"`
}

type MetricsRepository interface {
	GetCounter(ctx context.Context, name string) (int64, error)
	SetCounter(ctx context.Context, name string, value int64) error
	GetGauge(ctx context.Context, name string) (float64, error)
	SetGauge(ctx context.Context, name string, value float64) error
	Ping(ctx context.Context) error
}
