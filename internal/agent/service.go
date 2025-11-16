package agent

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"reflect"
	"runtime"
	"strings"
	"time"
)

type AgentService interface {
	Start()
	UpdateMetrics(memStats runtime.MemStats) Metrics
	PollMetrics() runtime.MemStats
	ReportMetrics()
}

type AgentConfig struct {
	ServerURL      string
	HTTPClient     http.Client
	PollInterval   time.Duration
	ReportInterval time.Duration
	Metrics        Metrics
}

func (c *AgentConfig) Start(ctx context.Context) {
	pollTicker := time.NewTicker(c.PollInterval)
	reportTicker := time.NewTicker(c.ReportInterval)
	defer pollTicker.Stop()
	defer reportTicker.Stop()

	log.Printf("agent started: poll=%v, report=%v", c.PollInterval, c.ReportInterval)

	for {
		select {
		case <-pollTicker.C:
			log.Printf("poll metrics")
			metrics := c.PollMetrics()
			c.Metrics = c.UpdateMetrics(metrics)

		case <-reportTicker.C:
			log.Printf("send metrics")
			c.ReportMetrics()
		case <-ctx.Done():
			log.Printf("agent stopped")
			return
		}
	}
}

func (c *AgentConfig) PollMetrics() runtime.MemStats {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	return memStats
}

func (c *AgentConfig) ReportMetrics() {
	for metric, value := range c.Metrics {
		metricType := "gauge"
		if metric == "PollCount" {
			metricType = "counter"
		}
		lowercaseMetric := strings.ToLower(metric)
		url := fmt.Sprintf("%s/update/%s/%s/%v", c.ServerURL, metricType, lowercaseMetric, value)
		log.Printf("make request %s", url)
		resp, err := c.HTTPClient.Post(url, "text/plain", strings.NewReader(""))
		if err != nil {
			log.Printf("error reporting metrics: %v\n", err)
			continue
		}

		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Printf("failed to report metrics: %d\n", resp.StatusCode)
		} else {
			log.Printf("successfully sent metric %s", lowercaseMetric)
		}
	}
}

func (c *AgentConfig) UpdateMetrics(memStats runtime.MemStats) Metrics {
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

	return metrics
}

func NewAgentService(client http.Client, serverBaseURL string, poolInterval int, reportInterval int) *AgentConfig {
	return &AgentConfig{
		ServerURL:      serverBaseURL,
		HTTPClient:     client,
		PollInterval:   time.Second * time.Duration(poolInterval),
		ReportInterval: time.Second * time.Duration(reportInterval),
		Metrics:        Metrics{},
	}
}
