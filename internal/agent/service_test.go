package agent

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"reflect"
	"runtime"
	"testing"
)

func TestAgent_PollData(t *testing.T) {
	agent := NewAgent(&Config{}).(*Agent)
	stats := agent.PollData()
	if stats.Alloc == 0 && stats.Sys == 0 {
		t.Errorf("PollData returned empty stats")
	}
}

func TestAgent_UpdateData(t *testing.T) {
	agent := NewAgent(&Config{}).(*Agent)
	agent.metrics["PollCount"] = 5

	memStats := runtime.MemStats{
		Alloc:     100,
		HeapAlloc: 200,
	}
	agent.UpdateData(memStats)

	if agent.metrics["Alloc"] != 100 {
		t.Errorf("Alloc = %v, want 100", agent.metrics["Alloc"])
	}
	if agent.metrics["HeapAlloc"] != 200 {
		t.Errorf("HeapAlloc = %v, want 200", agent.metrics["HeapAlloc"])
	}
	if agent.metrics["PollCount"] != 6 {
		t.Errorf("PollCount = %v, want 6", agent.metrics["PollCount"])
	}
	if _, ok := agent.metrics["RandomValue"]; !ok {
		t.Errorf("RandomValue key missing")
	}
}

func TestAgent_MakePayload(t *testing.T) {
	agent := NewAgent(&Config{}).(*Agent)

	data, err := agent.MakePayload("Alloc", 123)
	if err != nil {
		t.Fatalf("MakePayload failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	expected := map[string]interface{}{"id": "Alloc", "value": 123.0, "type": "gauge"}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("got %v, want %v", result, expected)
	}

	data, _ = agent.MakePayload("PollCount", 10)
	json.Unmarshal(data, &result)
	if result["type"] != "counter" || result["delta"].(float64) != 10 {
		t.Errorf("counter payload incorrect: %v", result)
	}
}

func TestAgent_CompressPayload(t *testing.T) {
	agent := NewAgent(&Config{}).(*Agent)
	payload := []byte(`{"id":"test","value":42}`)

	compressed, err := agent.CompressPayload(payload)
	if err != nil {
		t.Fatalf("CompressPayload failed: %v", err)
	}

	r, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		t.Fatalf("gzip.NewReader failed: %v", err)
	}
	defer r.Close()

	var buf bytes.Buffer
	buf.ReadFrom(r)

	if !bytes.Equal(buf.Bytes(), payload) {
		t.Errorf("decompressed mismatch, got %s, want %s", buf.Bytes(), payload)
	}
}
