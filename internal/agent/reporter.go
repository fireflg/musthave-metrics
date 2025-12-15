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
	client.RetryMax = 15
	client.RetryWaitMin = 500 * time.Millisecond
	client.RetryWaitMax = 3 * time.Second
	client.Logger = nil

	return &Reporter{
		serverURL: serverURL,
		client:    client,
	}
}

func (r *Reporter) WaitServer(ctx context.Context) error {
	url := fmt.Sprintf("%s/", r.serverURL)
	req, err := retryablehttp.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
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

func (r *Reporter) Report(ctx context.Context, metrics Metrics) error {

	payload, err := r.makePayload(metrics)
	if err != nil {
		return err
	}

	compressed, err := r.compressPayload(payload)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/updates/", r.serverURL)

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

func (r *Reporter) makePayload(metrics Metrics) ([]byte, error) {
	var payloadMap []map[string]interface{}
	for k, v := range metrics {
		metricType := "gauge"
		if k == "PollCount" {
			metricType = "counter"
		}
		if metricType == "counter" {
			payloadMap = append(payloadMap, map[string]interface{}{
				"id":    k,
				"delta": int64(v),
				"type":  metricType,
			})
		} else {
			payloadMap = append(payloadMap, map[string]interface{}{
				"id":    k,
				"value": v,
				"type":  metricType,
			})
		}
	}

	payload, err := json.Marshal(payloadMap)
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
