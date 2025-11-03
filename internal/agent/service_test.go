package agent

import (
	"net/http"
	"runtime"
	"testing"
	"time"
)

func TestAgentConfig_PollMetrics(t *testing.T) {
	c := &AgentConfig{}
	got := c.PollMetrics()

	if got.Alloc == 0 && got.Sys == 0 {
		t.Errorf("PollMetrics() returned empty stats: %+v", got)
	}
}

func TestAgentConfig_UpdateMetrics(t *testing.T) {
	c := &AgentConfig{Metrics: Metrics{"PollCount": 3}}

	memStats := runtime.MemStats{
		Alloc:     123,
		HeapAlloc: 456,
	}

	got := c.UpdateMetrics(memStats)

	if got["Alloc"] != 123 {
		t.Errorf("Alloc = %v, want 123", got["Alloc"])
	}
	if got["HeapAlloc"] != 456 {
		t.Errorf("HeapAlloc = %v, want 456", got["HeapAlloc"])
	}
	if got["PollCount"] != 4 {
		t.Errorf("PollCount = %v, want 4", got["PollCount"])
	}
	if _, ok := got["RandomValue"]; !ok {
		t.Errorf("RandomValue key missing")
	}
}

func TestNewAgentService(t *testing.T) {
	client := http.Client{}
	url := "http://127.0.0.1:8080"
	got := NewAgentService(client, url)

	if got.ServerURL != url {
		t.Errorf("ServerUrl = %v, want %v", got.ServerURL, url)
	}
	if got.PollInterval != 2*time.Second {
		t.Errorf("PollInterval = %v, want 2s", got.PollInterval)
	}
	if got.ReportInterval != 10*time.Second {
		t.Errorf("ReportInterval = %v, want 10s", got.ReportInterval)
	}
	if got.Metrics == nil {
		t.Errorf("Metrics map is nil")
	}
}
