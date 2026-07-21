package memory_test

import (
	"context"
	"testing"

	"github.com/fireflg/go-musthave-metrics-tpl/internal/repository/memory"
	models "github.com/fireflg/go-musthave-metrics-tpl/internal/model"
)

func BenchmarkSetGauge(b *testing.B) {
	repo := memory.NewMemoryRepository()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		name := "gauge_metric_" + string(rune('0'+i%10))
		repo.SetGauge(ctx, name, float64(i))
	}
}

func BenchmarkSetCounter(b *testing.B) {
	repo := memory.NewMemoryRepository()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		name := "counter_metric_" + string(rune('0'+i%10))
		repo.SetCounter(ctx, name, int64(i))
	}
}

func BenchmarkGetGauge(b *testing.B) {
	repo := memory.NewMemoryRepository()
	ctx := context.Background()

	// Pre-populate
	for i := 0; i < 100; i++ {
		repo.SetGauge(ctx, "gauge_metric", float64(i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		repo.GetGauge(ctx, "gauge_metric")
	}
}

func BenchmarkGetCounter(b *testing.B) {
	repo := memory.NewMemoryRepository()
	ctx := context.Background()

	// Pre-populate
	for i := 0; i < 100; i++ {
		repo.SetCounter(ctx, "counter_metric", int64(i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		repo.GetCounter(ctx, "counter_metric")
	}
}

func BenchmarkSetMetric_Gauge(b *testing.B) {
	repo := memory.NewMemoryRepository()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		value := float64(i)
		repo.SetMetric(ctx, models.Metrics{ID: "gauge_metric", MType: "gauge", Value: &value})
	}
}

func BenchmarkSetMetric_Counter(b *testing.B) {
	repo := memory.NewMemoryRepository()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		delta := int64(i)
		repo.SetMetric(ctx, models.Metrics{ID: "counter_metric", MType: "counter", Delta: &delta})
	}
}