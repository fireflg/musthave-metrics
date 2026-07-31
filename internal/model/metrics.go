package models

import "context"

// Константы типов метрик, представляющих различные виды метрик.
const (
	// Counter — тип метрики, накапливающей значения со временем.
	Counter = "counter"
	// Gauge — тип метрики, представляющей значение в конкретный момент времени.
	Gauge = "gauge"
)

// Metrics представляет одну метрику с её значением и метаданными.
// Delta и Value являются указателями для различения нулевых значений
// и неустановленных значений при JSON кодировании.
//
// Пример использования:
//
//	metric := models.Metrics{
//	    ID:    "memory_usage",
//	    MType: models.Gauge,
//	    Value: func() *float64 { v := 1024.0; return &v }(),
//	}
type Metrics struct {
	// ID — уникальный идентификатор метрики.
	ID string `json:"id"`
	// MType — тип метрики (counter или gauge).
	MType string `json:"type"`
	// Delta — значение счетчика (накопительное).
	Delta *int64 `json:"delta,omitempty"`
	// Value — значение gauge (значение в конкретный момент).
	Value *float64 `json:"value,omitempty"`
	// Hash — HMAC подпись для проверки целостности.
	Hash string `json:"hash,omitempty"`
}

// MetricsRepository определяет интерфейс для операций хранилища метрик.
// Реализации могут использовать различные бэкенды (memory, file, database).
type MetricsRepository interface {
	// GetCounter получает метрику counter по имени.
	GetCounter(ctx context.Context, name string) (int64, error)
	// SetCounter устанавливает значение метрики counter.
	SetCounter(ctx context.Context, name string, value int64) error
	// GetGauge получает метрику gauge по имени.
	GetGauge(ctx context.Context, name string) (float64, error)
	// SetGauge устанавливает значение метрики gauge.
	SetGauge(ctx context.Context, name string, value float64) error
	// SetMetric устанавливает метрику (gauge или counter) с автоматической обработкой.
	SetMetric(ctx context.Context, metric Metrics) error
	// GetMetric получает метрику по ID и типу.
	GetMetric(ctx context.Context, metricID, metricType string) (*Metrics, error)
	// SetMetrics устанавливает несколько метрик в пакетной операции.
	SetMetrics(ctx context.Context, metrics []Metrics) error
	// Ping проверяет доступность хранилища.
	Ping(ctx context.Context) error
}