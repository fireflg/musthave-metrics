package agent_test

import (
	"github.com/fireflg/ago-musthave-metrics-tpl/internal/agent"
	"runtime"
	"testing"
)

type fakeProvider struct{}

func (f *fakeProvider) Poll() runtime.MemStats {
	return runtime.MemStats{Alloc: 100, HeapAlloc: 200}
}

type fakeReporter struct {
	Reported map[string]float64
}

func newFakeReporter() *fakeReporter {
	return &fakeReporter{Reported: make(map[string]float64)}
}

func (r *fakeReporter) Report(metric string, value float64) error {
	r.Reported[metric] = value
	return nil
}

func TestMetrics_UpdateData(t *testing.T) {
	storage := agent.Metrics(make(map[string]float64))

	memStats := runtime.MemStats{Alloc: 100, HeapAlloc: 200}
	storage.UpdateData(memStats)

	if storage["Alloc"] != 100 {
		t.Fatalf("Alloc = %v, want 100", storage["Alloc"])
	}
	if storage["HeapAlloc"] != 200 {
		t.Fatalf("HeapAlloc = %v, want 200", storage["HeapAlloc"])
	}
	if storage["PollCount"] != 1 {
		t.Fatalf("PollCount = %v, want 1", storage["PollCount"])
	}
	if _, ok := storage["RandomValue"]; !ok {
		t.Fatalf("RandomValue key missing")
	}
}

func TestReporter(t *testing.T) {
	storage := agent.Metrics(make(map[string]float64))
	provider := &fakeProvider{}
	reporter := newFakeReporter()
	storage.UpdateData(provider.Poll())

	for metric, value := range storage {
		if err := reporter.Report(metric, value); err != nil {
			t.Fatalf("Failed to report metric: %v", err)
		}
	}

	if len(reporter.Reported) == 0 {
		t.Fatalf("No metrics reported")
	}
}
