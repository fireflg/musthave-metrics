package service

import (
	"encoding/json"
	models "github.com/fireflg/ago-musthave-metrics-tpl/internal/model"
	"os"
	"path/filepath"
	"testing"
)

func newTestStorage() *Storage {
	return &Storage{
		Metrics: make(map[string]models.Metric),
	}
}

func TestUpdateGaugeMetricValue(t *testing.T) {
	s := newTestStorage()

	err := s.UpdateGaugeMetricValue("Alloc", 123.45)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m, ok := s.Metrics["Alloc"]
	if !ok {
		t.Fatalf("metric not stored")
	}

	if m.MType != "gauge" {
		t.Errorf("expected type gauge, got %s", m.MType)
	}

	if m.Value == nil || *m.Value != 123.45 {
		t.Errorf("unexpected gauge value: %v", m.Value)
	}
}

func TestUpdateGaugeWrongType(t *testing.T) {
	s := newTestStorage()
	_ = s.UpdateCounterMetricValue("PollCount", 1)

	err := s.UpdateGaugeMetricValue("PollCount", 1.23)
	if err == nil {
		t.Fatal("expected error when updating counter as gauge")
	}
}

func TestUpdateCounterMetricValue(t *testing.T) {
	s := newTestStorage()

	err := s.UpdateCounterMetricValue("Requests", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = s.UpdateCounterMetricValue("Requests", 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := s.Metrics["Requests"]
	if m.Delta == nil || *m.Delta != 8 {
		t.Errorf("expected delta 8, got %v", m.Delta)
	}
}

func TestUpdateCounterFractionError(t *testing.T) {
	s := newTestStorage()

	err := s.UpdateCounterMetricValue("Count", 1.5)
	if err == nil {
		t.Fatal("expected error for fractional value")
	}
}

func TestGetGaugeMetricValue(t *testing.T) {
	s := newTestStorage()
	_ = s.UpdateGaugeMetricValue("Alloc", 10.5)

	val, err := s.GetGaugeMetricValue("Alloc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if val != 10.5 {
		t.Errorf("expected 10.5, got %v", val)
	}
}

func TestGetCounterMetricValue(t *testing.T) {
	s := newTestStorage()
	_ = s.UpdateCounterMetricValue("Count", 5)

	val, err := s.GetCounterMetricValue("Count")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if val != 5 {
		t.Errorf("expected 5, got %v", val)
	}
}

func TestGetMetricNotFound(t *testing.T) {
	s := newTestStorage()

	_, err := s.GetGaugeMetricValue("Missing")
	if err == nil {
		t.Fatalf("expected error for missing metric")
	}
}

func TestStoreMetrics(t *testing.T) {
	tmp := t.TempDir()
	file := filepath.Join(tmp, "metrics.json")

	s := Storage{
		Metrics:     make(map[string]models.Metric),
		StoragePath: file,
	}

	_ = s.UpdateGaugeMetricValue("Alloc", 123)
	_ = s.UpdateCounterMetricValue("Count", 5)

	err := s.StoreMetrics()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("cannot read file: %v", err)
	}

	var arr []models.Metric
	if err := json.Unmarshal(data, &arr); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if len(arr) != 2 {
		t.Fatalf("expected 2 metrics, got %d", len(arr))
	}
}

func TestRestoreMetrics(t *testing.T) {
	tmp := t.TempDir()
	file := filepath.Join(tmp, "metrics.json")

	original := []models.Metric{
		{ID: "Alloc", MType: "gauge", Value: floatPtr(100)},
		{ID: "Count", MType: "counter", Delta: int64Ptr(42)},
	}

	data, _ := json.Marshal(original)
	_ = os.WriteFile(file, data, 0644)

	s := &Storage{
		Metrics:     make(map[string]models.Metric),
		StoragePath: file,
	}

	err := s.RestoreMetrics()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(s.Metrics) != 2 {
		t.Fatalf("expected 2 metrics, got %d", len(s.Metrics))
	}

	if *s.Metrics["Alloc"].Value != 100 {
		t.Errorf("wrong gauge value")
	}

	if *s.Metrics["Count"].Delta != 42 {
		t.Errorf("wrong counter delta")
	}
}

func floatPtr(v float64) *float64 { return &v }
func int64Ptr(v int64) *int64     { return &v }
