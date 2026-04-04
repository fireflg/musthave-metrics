package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/go-retryablehttp"
	"net/http"
	"time"
)

type Reporter struct {
	serverURL string
	client    *retryablehttp.Client
	secretKey string
}

type MetricsReporter interface {
	Report(ctx context.Context, metrics Metrics) error
	WaitServer(ctx context.Context) error
}

func NewReporter(serverURL string, secretKey string) *Reporter {
	client := retryablehttp.NewClient()
	// Временный хардкод параметров
	client.RetryMax = 15
	client.RetryWaitMin = 500 * time.Millisecond
	client.RetryWaitMax = 3 * time.Second
	client.Logger = nil
	return &Reporter{
		serverURL: serverURL,
		client:    client,
		secretKey: secretKey,
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
	var hash string

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

	if len(r.secretKey) > 0 {
		hash, err = r.signPayload(payload)
		fmt.Println(hash)
		if err != nil {
			return err
		}
		req.Header.Set("HashSHA256", hash)
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

func (r *Reporter) signPayload(payload []byte) (string, error) {
	h := hmac.New(sha256.New, []byte(r.secretKey))

	n, err := h.Write(payload)
	if err != nil {
		return "", fmt.Errorf("failed to write to hmac: %w", err)
	}

	if n != len(payload) {
		return "", fmt.Errorf("partial write to hmac: wrote %d of %d bytes", n, len(payload))
	}
	signature := h.Sum(nil)

	signString := hex.EncodeToString(signature)

	return signString, nil
}
