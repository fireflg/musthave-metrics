package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/go-retryablehttp"
	"net/http"
	"time"
)

type Reporter struct {
	serverURL string
	client    *retryablehttp.Client
}

func NewReporter(serverURL string) *Reporter {
	client := retryablehttp.NewClient()
	// Временный хардкод параметров
	client.RetryMax = 10
	client.RetryWaitMin = 200 * time.Millisecond
	client.RetryWaitMax = 3 * time.Second
	client.Logger = nil

	return &Reporter{
		serverURL: serverURL,
		client:    client,
	}
}

func (r *Reporter) Report(ctx context.Context, metric string, value float64) error {

	payload, err := r.makePayload(metric, value)
	if err != nil {
		return err
	}

	compressed, err := r.compressPayload(payload)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/update/", r.serverURL)

	req, err := retryablehttp.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(compressed))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")

	resp, err := r.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	return nil
}

func (r *Reporter) makePayload(metric string, value float64) ([]byte, error) {
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
		return nil, err
	}
	return payload, nil
}

func (r *Reporter) compressPayload(payload []byte) ([]byte, error) {
	var compressedBuf bytes.Buffer

	gzipWriter := gzip.NewWriter(&compressedBuf)

	_, err := gzipWriter.Write(payload)

	if err != nil {
		return nil, err
	}

	if err = gzipWriter.Close(); err != nil {
		return nil, err
	}

	return compressedBuf.Bytes(), nil
}
