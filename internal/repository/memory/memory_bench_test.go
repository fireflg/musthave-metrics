package memory_test

import (
	"testing"

	models "github.com/fireflg/go-musthave-metrics-tpl/internal/model"
	"github.com/fireflg/go-musthave-metrics-tpl/internal/repository/memory"
)

func BenchmarkSetGauge(b *testing.B) {
	b.StopTimer()
	repo := memory.NewMemoryRepository()
	ctx := b.Context()
	var i int64
	b.StartTimer()

	for b.Loop() {
		repo.SetGauge(ctx, "gauge_metric", float64(i))
		i++
	}
}

func BenchmarkSetCounter(b *testing.B) {
	b.StopTimer()
	repo := memory.NewMemoryRepository()
	ctx := b.Context()
	var i int64
	b.StartTimer()

	for b.Loop() {
		repo.SetCounter(ctx, "counter_metric", i)
		i++
	}
}

func BenchmarkGetGauge(b *testing.B) {
	b.StopTimer()
	repo := memory.NewMemoryRepository()
	ctx := b.Context()

	for i := 0; i < 100; i++ {
		repo.SetGauge(ctx, "gauge_metric", float64(i))
	}
	b.StartTimer()

	for b.Loop() {
		repo.GetGauge(ctx, "gauge_metric")
	}
}

func BenchmarkGetCounter(b *testing.B) {
	b.StopTimer()
	repo := memory.NewMemoryRepository()
	ctx := b.Context()

	for i := 0; i < 100; i++ {
		repo.SetCounter(ctx, "counter_metric", int64(i))
	}
	b.StartTimer()

	for b.Loop() {
		repo.GetCounter(ctx, "counter_metric")
	}
}

func BenchmarkSetMetric_Gauge(b *testing.B) {
	b.StopTimer()
	repo := memory.NewMemoryRepository()
	ctx := b.Context()
	var i int64
	b.StartTimer()

	for b.Loop() {
		value := float64(i)
		repo.SetMetric(ctx, models.Metrics{ID: "gauge_metric", MType: "gauge", Value: &value})
		i++
	}
}

func BenchmarkSetMetric_Counter(b *testing.B) {
	b.StopTimer()
	repo := memory.NewMemoryRepository()
	ctx := b.Context()
	var i int64
	b.StartTimer()

	for b.Loop() {
		repo.SetMetric(ctx, models.Metrics{ID: "counter_metric", MType: "counter", Delta: &i})
		i++
	}
}
