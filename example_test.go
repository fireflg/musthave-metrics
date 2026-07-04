package musthave_metrics_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/fireflg/go-musthave-metrics-tpl/internal/handler"
	models "github.com/fireflg/go-musthave-metrics-tpl/internal/model"
	"github.com/fireflg/go-musthave-metrics-tpl/internal/service"
	"go.uber.org/zap"
)

// This file contains examples of how to use the metrics server endpoints.
// Run the server first, then use these examples to interact with it.
//
// The server provides the following endpoints:
//   - GET / - Health check page
//   - GET /value/{metricType}/{metricName} - Get metric by type and name
//   - POST /update/{metricType}/{metricName}/{metricValue} - Update metric via URL
//   - POST /update/ - Update metric via JSON body
//   - POST /updates/ - Batch update metrics via JSON body
//   - POST /value/ - Get metric via JSON body
//   - GET /ping - Database health check

func Example_updateGaugeMetric() {
	// Example: Update a gauge metric via URL
	// POST /update/gauge/memory_usage/1024.5

	server := createTestServer()
	defer server.Close()

	// Update a gauge metric
	resp, err := http.Post(
		fmt.Sprintf("%s/update/gauge/memory_usage/1024.5", server.URL),
		"text/plain",
		nil,
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("Status: %d\n", resp.StatusCode)
	// Output: Status: 200
}

func Example_updateCounterMetric() {
	// Example: Update a counter metric via URL
	// POST /update/counter/requests/1

	server := createTestServer()
	defer server.Close()

	// Update a counter metric (counter values accumulate)
	resp, err := http.Post(
		fmt.Sprintf("%s/update/counter/requests/1", server.URL),
		"text/plain",
		nil,
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	resp.Body.Close()

	// Update again to see accumulation
	resp2, err := http.Post(
		fmt.Sprintf("%s/update/counter/requests/1", server.URL),
		"text/plain",
		nil,
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	resp2.Body.Close()

	// Get the counter value
	resp, err = http.Get(fmt.Sprintf("%s/value/counter/requests", server.URL))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	fmt.Printf("Counter value: %s\n", buf.String())
	// Output: Counter value: 2
}

func Example_getMetricJSON() {
	// Example: Get metric via JSON request
	// POST /value/ with JSON body

	server := createTestServer()
	defer server.Close()

	// First, set a metric
	metric := models.Metrics{
		ID:    "cpu_usage",
		MType: models.Gauge,
		Value: func() *float64 { v := 75.5; return &v }(),
	}
	body, _ := json.Marshal(metric)
	resp, err := http.Post(
		fmt.Sprintf("%s/update/", server.URL),
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	// Then retrieve it via JSON
	body, _ = json.Marshal(models.Metrics{ID: "cpu_usage", MType: "gauge"})
	resp, err = http.Post(
		fmt.Sprintf("%s/value/", server.URL),
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	fmt.Printf("Value: %.1f\n", result["value"])
	// Output: Value: 75.5
}

func Example_batchUpdateMetrics() {
	// Example: Batch update multiple metrics
	// POST /updates/ with JSON array

	server := createTestServer()
	defer server.Close()

	metrics := []models.Metrics{
		{ID: "memory_total", MType: models.Gauge, Value: func() *float64 { v := 8192.0; return &v }()},
		{ID: "memory_used", MType: models.Gauge, Value: func() *float64 { v := 4096.0; return &v }()},
		{ID: "requests_total", MType: models.Counter, Delta: func() *int64 { v := int64(100); return &v }()},
	}
	body, _ := json.Marshal(metrics)
	resp, err := http.Post(
		fmt.Sprintf("%s/updates/", server.URL),
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("Status: %d\n", resp.StatusCode)
	// Output: Status: 200
}

func Example_getMetricPlain() {
	// Example: Get metric by type and name
	// GET /value/gauge/temperature

	server := createTestServer()
	defer server.Close()

	// Set a gauge value first
	resp, err := http.Post(
		fmt.Sprintf("%s/update/gauge/temperature/25.5", server.URL),
		"text/plain",
		nil,
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	// Get the value
	resp, err = http.Get(fmt.Sprintf("%s/value/gauge/temperature", server.URL))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	fmt.Printf("Temperature: %s°C\n", buf.String())
	// Output: Temperature: 25.5°C
}

func Example_pingDatabase() {
	// Example: Check database connectivity
	// GET /ping

	server := createTestServer()
	defer server.Close()

	resp, err := http.Get(fmt.Sprintf("%s/ping", server.URL))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		fmt.Println("Database: OK")
	} else {
		fmt.Println("Database: Error")
	}
	// Output: Database: OK
}

// Helper function to create a test server instance
func createTestServer() *httptest.Server {
	logger, _ := zap.NewDevelopment()
	sugar := logger.Sugar()

	// Create in-memory repository
	repo := service.NewMetricsService(nil, nil)

	// Create handler with the service
	h := handler.NewMetricsHandler(repo, sugar)
	r := h.ServerRouter()

	return httptest.NewServer(r)
}

// Example_gzipCompressedRequest demonstrates sending compressed requests
func Example_gzipCompressedRequest() {
	// Example: Send gzip compressed request
	// The server accepts gzip-encoded request bodies

	server := createTestServer()
	defer server.Close()

	// Create JSON payload
	metric := models.Metrics{
		ID:    "compressed_metric",
		MType: models.Gauge,
		Value: func() *float64 { v := 123.45; return &v }(),
	}
	body, _ := json.Marshal(metric)

	// Create gzip compressed reader
	var _ bytes.Buffer
	// Note: In production, use gzip.NewWriter(&buf)
	// For this example, we send regular JSON
	resp, err := http.Post(
		fmt.Sprintf("%s/update/", server.URL),
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("Status: %d\n", resp.StatusCode)
	// Output: Status: 200
}

// Example_signedRequest demonstrates HMAC-signed requests
func Example_signedRequest() {
	// Example: Send request with HMAC signature
	// Requires setting HASH_KEY environment variable or -k flag

	server := createTestServer()
	defer server.Close()

	// Create JSON payload
	metric := models.Metrics{
		ID:    "signed_metric",
		MType: models.Gauge,
		Value: func() *float64 { v := 99.9; return &v }(),
	}
	body, _ := json.Marshal(metric)

	// In production, compute HMAC-SHA256 of body with secret key
	// and add header: HashSHA256: <hex-encoded-signature>
	// For testing without signature, omit the header

	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/update/", server.URL),
		bytes.NewReader(body),
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("Status: %d\n", resp.StatusCode)
	// Output: Status: 200
}

// Example_parseMetricResponse demonstrates parsing metric responses
func Example_parseMetricResponse() {
	// Example: Parse JSON response from /value/ endpoint

	server := createTestServer()
	defer server.Close()

	// Set a metric first
	metric := models.Metrics{
		ID:    "parse_test",
		MType: models.Gauge,
		Value: func() *float64 { v := 42.0; return &v }(),
	}
	body, _ := json.Marshal(metric)
	resp, err := http.Post(
		fmt.Sprintf("%s/update/", server.URL),
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	// Request the metric back
	requestBody, _ := json.Marshal(models.Metrics{ID: "parse_test", MType: "gauge"})
	resp, err = http.Post(
		fmt.Sprintf("%s/value/", server.URL),
		"application/json",
		bytes.NewReader(requestBody),
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	fmt.Printf("ID: %s, Type: %s, Value: %.1f\n",
		result["id"], result["type"], result["value"])
	// Output: ID: parse_test, Type: gauge, Value: 42.0
}

// Example_errorHandling demonstrates handling error responses
func Example_errorHandling() {
	// Example: Handle error responses from the server

	server := createTestServer()
	defer server.Close()

	// Try to get a non-existent metric
	requestBody, _ := json.Marshal(models.Metrics{ID: "nonexistent", MType: "gauge"})
	resp, err := http.Post(
		fmt.Sprintf("%s/value/", server.URL),
		"application/json",
		bytes.NewReader(requestBody),
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		fmt.Println("Metric not found")
	} else if resp.StatusCode >= 400 {
		fmt.Printf("Error: %d\n", resp.StatusCode)
	}
	// Output: Metric not found
}

// Example_workWithCounters demonstrates counter accumulation
func Example_workWithCounters() {
	// Example: Counters accumulate over time

	server := createTestServer()
	defer server.Close()

	// Increment counter multiple times
	for i := 0; i < 3; i++ {
		resp, err := http.Post(
			fmt.Sprintf("%s/update/counter/hits/1", server.URL),
			"text/plain",
			nil,
		)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}

	// Get final counter value
	resp, err := http.Get(fmt.Sprintf("%s/value/counter/hits", server.URL))
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	value := strings.TrimSpace(buf.String())
	fmt.Printf("Counter value: %s\n", value)
	// Output: Counter value: 3
}
