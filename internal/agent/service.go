package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"reflect"
	"runtime"
	"time"

	"go.uber.org/zap"
)

type Metrics map[string]float64

type Agent struct {
	cfg     *Config
	metrics Metrics
}

type AgentService interface {
	Start(ctx context.Context)
	PollData() runtime.MemStats
	Report(metric string, payload []byte) error
	MakePayload(metric string, value float64) ([]byte, error)
	UpdateData(memStats runtime.MemStats)
	CompressPayload(payload []byte) ([]byte, error)
}

var _ AgentService = (*Agent)(nil)

type Config struct {
	ServerURL      string
	HTTPClient     http.Client
	PollInterval   time.Duration
	ReportInterval time.Duration
	Logger         *zap.SugaredLogger
}

func NewAgent(cfg *Config) AgentService {
	if cfg.Logger == nil {
		l, _ := zap.NewProduction()
		cfg.Logger = l.Sugar()
	}
	return &Agent{
		cfg:     cfg,
		metrics: Metrics{},
	}
}

func (a *Agent) Start(ctx context.Context) {
	pollTicker := time.NewTicker(a.cfg.PollInterval)
	reportTicker := time.NewTicker(a.cfg.ReportInterval)
	defer pollTicker.Stop()
	defer reportTicker.Stop()

	a.cfg.Logger.Infof("agent started")

	for {
		select {
		case <-pollTicker.C:
			a.UpdateData(a.PollData())

		case <-reportTicker.C:
			for metric, value := range a.metrics {
				payload, err := a.MakePayload(metric, value)
				if err != nil {
					a.cfg.Logger.Warnw("failed to make payload", "metric", metric, "value", value, "error", err)
					continue
				}

				compressedPayload, err := a.CompressPayload(payload)
				if err != nil {
					a.cfg.Logger.Warnw("failed to compress payload", "metric", metric, "value", value, "error", err)
					continue
				}

				err = a.Report(metric, compressedPayload)
				if err != nil {
					a.cfg.Logger.Warnw("failed to report metric", "metric", metric, "value", value, "error", err)
				}
			}

		case <-ctx.Done():
			a.cfg.Logger.Infof("agent stopped")
			return
		}
	}
}

func (a *Agent) PollData() runtime.MemStats {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	return memStats
}

func (a *Agent) Report(metric string, payload []byte) error {
	const maxRetries = 5
	backoff := 100 * time.Millisecond
	url := fmt.Sprintf("%s/update/", a.cfg.ServerURL)

	for attempt := 1; attempt <= maxRetries; attempt++ {
		req, err := http.NewRequest("POST", url, bytes.NewReader(payload))
		if err != nil {
			a.cfg.Logger.Warnw("failed to create request", "metric", metric, "attempt", attempt, "error", err)
			continue
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Content-Encoding", "gzip")

		resp, err := a.cfg.HTTPClient.Do(req)
		if err == nil && resp != nil && resp.StatusCode == http.StatusOK {
			a.cfg.Logger.Infof("successfully sent metric %s", metric)
			resp.Body.Close()
			break
		}

		if err != nil {
			a.cfg.Logger.Warnw("send error", "metric", metric, "attempt", attempt, "error", err)
		} else {
			a.cfg.Logger.Warnw("bad status", "metric", metric, "attempt", attempt, "status", resp.StatusCode)
			resp.Body.Close()
		}

		if attempt == maxRetries {
			a.cfg.Logger.Errorw("failed to send metric after retries", "metric", metric, "attempts", maxRetries)
			break
		}

		time.Sleep(backoff)
		backoff *= 2
	}

	return nil
}

func (a *Agent) MakePayload(metric string, value float64) ([]byte, error) {
	metricType := "gauge"
	if metric == "PollCount" {
		metricType = "counter"
	}

	var payload []byte
	var err error

	if metricType == "counter" {
		payload, err = json.Marshal(map[string]interface{}{
			"id":    metric,
			"delta": int64(value),
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
		a.cfg.Logger.Warnw("failed to marshal payload", "metric", metric, "error", err)
		return nil, err
	}
	return payload, nil
}

func (a *Agent) UpdateData(memStats runtime.MemStats) {
	v := reflect.ValueOf(memStats)

	for _, name := range memStatFields {
		field := v.FieldByName(name)
		if field.IsValid() {
			switch field.Kind() {
			case reflect.Uint64:
				a.metrics[name] = float64(field.Uint())
			case reflect.Float64:
				a.metrics[name] = field.Float()
			case reflect.Int64:
				a.metrics[name] = float64(field.Int())
			default:
				a.metrics[name] = 0
			}
			continue
		}
		a.metrics[name] = 0
	}

	a.metrics["RandomValue"] = rand.ExpFloat64()
	a.metrics["PollCount"]++
}

func (a *Agent) CompressPayload(payload []byte) ([]byte, error) {
	var compressedBuf bytes.Buffer

	gzipWriter := gzip.NewWriter(&compressedBuf)

	_, err := gzipWriter.Write(payload)

	if err != nil {
		a.cfg.Logger.Warnw("failed to write compressed payload", "error", err)
		return nil, err
	}

	if err = gzipWriter.Close(); err != nil {
		a.cfg.Logger.Warnw("failed to close gzip writer", "error", err)
		return nil, err
	}

	return compressedBuf.Bytes(), nil
}
