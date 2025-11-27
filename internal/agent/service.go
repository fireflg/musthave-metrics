package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"log"
	"math/rand"
	"net/http"
	"reflect"
	"runtime"
	"time"
)

type Service interface {
	Start()
	UpdateMetrics(memStats runtime.MemStats) Metrics
	PollMetrics() runtime.MemStats
	ReportMetrics()
}

type Config struct {
	ServerURL      string
	HTTPClient     http.Client
	PollInterval   time.Duration
	ReportInterval time.Duration
	Metrics        Metrics
}

func (c *Config) Start(ctx context.Context, logger *zap.SugaredLogger) {
	pollTicker := time.NewTicker(c.PollInterval)
	reportTicker := time.NewTicker(c.ReportInterval)
	defer pollTicker.Stop()
	defer reportTicker.Stop()

	logger.Infof("agent started: poll=%v, report=%v", c.PollInterval, c.ReportInterval)

	for {
		select {
		case <-pollTicker.C:
			logger.Infof("poll metrics")
			metrics := c.PollMetrics()
			c.Metrics = c.UpdateMetrics(metrics)

		case <-reportTicker.C:
			logger.Infof("send metrics")
			c.ReportMetrics()
		case <-ctx.Done():
			logger.Infof("agent stopped")
			return
		}
	}
}

func (c *Config) PollMetrics() runtime.MemStats {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	return memStats
}

func (c *Config) ReportMetrics() {
	for metric, value := range c.Metrics {
		metricType := "gauge"
		if metric == "PollCount" {
			metricType = "counter"
		}

		url := fmt.Sprintf("%s/update/", c.ServerURL)

		var payload []byte
		var err error
		if metricType == "counter" {
			delta := int64(value)
			payload, err = json.Marshal(map[string]interface{}{
				"id":    metric,
				"delta": delta,
				"type":  metricType,
			})
		} else {
			payload, err = json.Marshal(map[string]interface{}{
				"id":    metric,
				"value": value,
				"type":  metricType,
			})
		}
		if err != nil {
			log.Printf("marshal error: %v", err)
			continue
		}

		const maxRetries = 5
		backoff := 100 * time.Millisecond

		for attempt := 1; attempt <= maxRetries; attempt++ {
			resp, err := c.HTTPClient.Post(url, "application/json", bytes.NewReader(payload))
			if err == nil && resp != nil && resp.StatusCode == http.StatusOK {
				log.Printf("successfully sent %s", metric)
				resp.Body.Close()
				break
			}

			if err != nil {
				log.Printf("send error (attempt %d/%d): %v", attempt, maxRetries, err)
			} else {
				log.Printf("bad status %d (attempt %d/%d) for metric %s", resp.StatusCode, attempt, maxRetries, metric)
				resp.Body.Close()
			}

			if attempt == maxRetries {
				log.Printf("failed to send %s after %d attempts", metric, maxRetries)
				break
			}

			time.Sleep(backoff)
			backoff *= 2
		}
	}
}

func (c *Config) UpdateMetrics(memStats runtime.MemStats) Metrics {
	metrics := Metrics{}

	v := reflect.ValueOf(memStats)
	for _, name := range memStatFields {
		field := v.FieldByName(name)
		if field.IsValid() {
			switch field.Kind() {
			case reflect.Uint64:
				metrics[name] = float64(field.Uint())
			case reflect.Float64:
				metrics[name] = field.Float()
			case reflect.Int64:
				metrics[name] = float64(field.Int())
			default:
				metrics[name] = 0.0
			}
		} else {
			metrics[name] = 0.0
		}
	}

	metrics["RandomValue"] = rand.ExpFloat64()
	metrics["PollCount"] = c.Metrics["PollCount"] + 1
	c.Metrics["PollCount"] = metrics["PollCount"]

	return metrics
}

func NewAgentService(client http.Client, serverBaseURL string, poolInterval int, reportInterval int) *Config {
	return &Config{
		ServerURL:      serverBaseURL,
		HTTPClient:     client,
		PollInterval:   time.Second * time.Duration(poolInterval),
		ReportInterval: time.Second * time.Duration(reportInterval),
		Metrics:        Metrics{},
	}
}
