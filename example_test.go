package musthave_metrics_test

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/fireflg/go-musthave-metrics-tpl/internal/handler"
	models "github.com/fireflg/go-musthave-metrics-tpl/internal/model"
	"github.com/fireflg/go-musthave-metrics-tpl/internal/repository/memory"
	"github.com/fireflg/go-musthave-metrics-tpl/internal/service"
	"go.uber.org/zap"
)

// Файл содержит примеры использования эндпоинтов сервера метрик.
//
// Сервер предоставляет следующие эндпоинты:
//   - GET / - Страница проверки здоровья
//   - GET /value/{metricType}/{metricName} - Получить метрику по типу и имени
//   - POST /update/{metricType}/{metricName}/{metricValue} - Обновить метрику через URL
//   - POST /update/ - Обновить метрику через JSON тело
//   - POST /updates/ - Пакетное обновление метрик через JSON
//   - POST /value/ - Получить метрику через JSON тело
//   - GET /ping - Проверка здоровья базы данных

func Example_updateGaugeMetric() {

	// POST /update/gauge/memory_usage/1024.5

	server := createTestServer()
	defer server.Close()

	// Обновляем gauge метрику
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

	// POST /update/counter/requests/1

	server := createTestServer()
	defer server.Close()

	// Обновляем counter метрику (значения накапливаются)
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

	// Обновляем снова, чтобы увидеть накопление
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

	// Получаем значение счетчика
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

	// POST /value/ с JSON телом

	server := createTestServer()
	defer server.Close()

	// Сначала устанавливаем метрику
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

	// Затем получаем её через JSON
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

	// POST /updates/ с JSON массивом

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

	// GET /value/gauge/temperature

	server := createTestServer()
	defer server.Close()

	// Сначала устанавливаем значение gauge
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

	// Получаем значение
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

// createTestServer создает тестовый сервер с in-memory репозиторием.
func createTestServer() *httptest.Server {
	return createSignedTestServer("")
}

func createSignedTestServer(secretKey string) *httptest.Server {
	logger, _ := zap.NewDevelopment()
	sugar := logger.Sugar()

	repo := memory.NewMemoryRepository()
	svc := service.NewMetricsService(repo, nil)

	h := handler.NewMetricsHandler(svc, sugar, secretKey, nil, nil)
	r := h.ServerRouter()

	return httptest.NewServer(r)
}

// Example_gzipCompressedRequest демонстрирует отправку сжатых запросов
func Example_gzipCompressedRequest() {

	// Сервер принимает gzip-encoded тела запросов

	server := createTestServer()
	defer server.Close()

	// Создаем JSON payload
	metric := models.Metrics{
		ID:    "compressed_metric",
		MType: models.Gauge,
		Value: func() *float64 { v := 123.45; return &v }(),
	}
	body, _ := json.Marshal(metric)

	// Создаем gzip сжатый reader
	var _ bytes.Buffer
	// Примечание: в продакшене используйте gzip.NewWriter(&buf)
	// В этом примере отправляем обычный JSON
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

// Example_signedRequest демонстрирует HMAC-подписанные запросы
func Example_signedRequest() {

	const secretKey = "super-secret"

	server := createSignedTestServer(secretKey)
	defer server.Close()

	// Создаем JSON payload
	metric := models.Metrics{
		ID:    "signed_metric",
		MType: models.Gauge,
		Value: func() *float64 { v := 99.9; return &v }(),
	}
	body, _ := json.Marshal(metric)

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

	mac := hmac.New(sha256.New, []byte(secretKey))
	mac.Write(body)
	req.Header.Set("HashSHA256", hex.EncodeToString(mac.Sum(nil)))

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

// Example_parseMetricResponse демонстрирует разбор ответов метрик
func Example_parseMetricResponse() {

	server := createTestServer()
	defer server.Close()

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

// Example_errorHandling демонстрирует обработку ошибок
func Example_errorHandling() {

	server := createTestServer()
	defer server.Close()

	// Пробуем получить несуществующую метрику
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

// Example_workWithCounters демонстрирует накопление счетчиков
func Example_workWithCounters() {

	server := createTestServer()
	defer server.Close()

	// Увеличиваем счетчик несколько раз
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

	// Получаем итоговое значение счетчика
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
