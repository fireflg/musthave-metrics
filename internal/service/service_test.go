package service

import (
	models "github.com/fireflg/ago-musthave-metrics-tpl/internal/model"
	"go.uber.org/zap"
	"os"
	"path/filepath"
	"testing"
)

func newManagerWithRealStorage(t *testing.T) (*MetricManagerImpl, *Storage) {
	logger, _ := zap.NewDevelopment()
	sugar := logger.Sugar()

	tmp := t.TempDir()
	file := filepath.Join(tmp, "metrics.json")

	st := &Storage{
		Metrics:     make(map[string]models.Metric),
		StoragePath: file,
	}

	cfg := &Config{
		PersistentStorageInterval: 0,
	}

	m, err := NewMertricsManager(cfg, sugar, st)
	if err != nil {
		t.Fatalf("cannot create manager: %v", err)
	}

	return m, st
}

func TestSetMetric_Gauge(t *testing.T) {
	m, st := newManagerWithRealStorage(t)

	err := m.SetMetric("gauge", "Alloc", 12.34)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	val, err := st.GetGaugeMetricValue("Alloc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != 12.34 {
		t.Fatalf("expected 12.34, got %f", val)
	}

	// проверяем, что store файл реально создался
	_, err = os.Stat(st.StoragePath)
	if err != nil {
		t.Fatalf("StoreMetrics did not write file: %v", err)
	}
}

func TestSetMetric_Counter(t *testing.T) {
	m, st := newManagerWithRealStorage(t)

	err := m.SetMetric("counter", "Requests", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	val, err := st.GetCounterMetricValue("Requests")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != 5 {
		t.Fatalf("expected 5, got %f", val)
	}
}

func TestSetMetric_UnknownType(t *testing.T) {
	m, _ := newManagerWithRealStorage(t)

	err := m.SetMetric("unknown", "X", 10)
	if err == nil {
		t.Fatal("expected error for unknown metric type")
	}
}

func TestGetMetric_Gauge(t *testing.T) {
	m, _ := newManagerWithRealStorage(t)

	_ = m.SetMetric("gauge", "Temp", 9.99)

	val, err := m.GetMetric("gauge", "Temp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if val != 9.99 {
		t.Fatalf("expected 9.99, got %f", val)
	}
}

func TestGetMetric_Counter(t *testing.T) {
	m, _ := newManagerWithRealStorage(t)

	_ = m.SetMetric("counter", "Hits", 3)

	val, err := m.GetMetric("counter", "Hits")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if val != 3 {
		t.Fatalf("expected 3, got %f", val)
	}
}

func TestGetMetric_NotFound(t *testing.T) {
	m, _ := newManagerWithRealStorage(t)

	_, err := m.GetMetric("gauge", "missing")
	if err == nil {
		t.Fatal("expected error for missing metric")
	}
}
