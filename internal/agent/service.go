package agent

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
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

func (c *AgentConfig) Start() {
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

		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Printf("failed to report metrics: %d\n", resp.StatusCode)
		} else {
			log.Printf("successfully sent metric %s", lowercaseMetric)
		}
	}
}

func (c *AgentConfig) UpdateMetrics(memStats runtime.MemStats) Metrics {
	return Metrics{
		"Alloc":         float64(memStats.Alloc),
		"BuckHashSys":   float64(memStats.BuckHashSys),
		"Frees":         float64(memStats.Frees),
		"GCCPUFraction": memStats.GCCPUFraction,
		"HeapAlloc":     float64(memStats.HeapAlloc),
		"HeapIdle":      float64(memStats.HeapIdle),
		"HeapInuse":     float64(memStats.HeapInuse),
		"HeapReleased":  float64(memStats.HeapReleased),
		"HeapObjects":   float64(memStats.HeapObjects),
		"HeapSys":       float64(memStats.HeapSys),
		"LastGC":        float64(memStats.LastGC),
		"Lookups":       float64(memStats.Lookups),
		"MCacheInuse":   float64(memStats.MCacheInuse),
		"MCacheSys":     float64(memStats.MCacheSys),
		"MSpanInuse":    float64(memStats.MSpanInuse),
		"Mallocs":       float64(memStats.Mallocs),
		"NextGC":        float64(memStats.NextGC),
		"NumForcedGC":   float64(memStats.NumForcedGC),
		"NumGC":         float64(memStats.NumGC),
		"OtherSys":      float64(memStats.OtherSys),
		"PauseTotalNs":  float64(memStats.PauseTotalNs),
		"StackInuse":    float64(memStats.StackInuse),
		"Sys":           float64(memStats.Sys),
		"TotalAlloc":    float64(memStats.TotalAlloc),
		"RandomValue":   rand.ExpFloat64(),
		"PollCount":     c.Metrics["PollCount"] + 1,
	}
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
