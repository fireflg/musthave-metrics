// Package observer provides auditing functionality to track metric changes.
//
// The observer supports multiple backends: file and HTTP.
// Each observer can be used independently or combined.
package observer

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/hashicorp/go-retryablehttp"
)

type contextKey string

const clientIPKey contextKey = "client_ip"

func ClientIP(ctx context.Context) string {
	if v := ctx.Value(clientIPKey); v != nil {
		if ip, ok := v.(string); ok {
			return ip
		}
	}
	return ""
}

// AuditEntry represents a single audit record.
type AuditEntry struct {
	TS        int64    `json:"ts"`         // Timestamp of the audit event.
	Metrics   []string `json:"metrics"`    // List of metric IDs that were modified.
	IPAddress string   `json:"ip_address"` // Client IP address that triggered the event.
}

type Observer interface {
	Notify(ctx context.Context, metricID string)
	NotifyBatch(ctx context.Context, metricIDs []string)
}

type FileObserver struct {
	filePath string
	file     *os.File
	mu       sync.Mutex
}

func NewFileObserver(filePath string) (*FileObserver, error) {
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	return &FileObserver{filePath: filePath, file: file}, nil
}

func (f *FileObserver) Close() error {
	if f.file != nil {
		return f.file.Close()
	}
	return nil
}

func (f *FileObserver) Notify(ctx context.Context, metricID string) {
	f.NotifyBatch(ctx, []string{metricID})
}

func (f *FileObserver) NotifyBatch(ctx context.Context, metricIDs []string) {
	entry := AuditEntry{
		TS:        time.Now().Unix(),
		Metrics:   metricIDs,
		IPAddress: ClientIP(ctx),
	}

	data, err := entry.Marshal()
	if err != nil {
		log.Printf("Error marshaling audit entry: %v", err)
		return
	}

	data = append(data, '\n')

	f.mu.Lock()
	defer f.mu.Unlock()

	if _, err := f.file.Write(data); err != nil {
		log.Printf("Error writing to audit file: %v", err)
		return
	}

	log.Printf("Audit record written to file: %s", f.filePath)
}

type HTTPObserver struct {
	url    string
	client *retryablehttp.Client
}

func NewHTTPObserver(url string) *HTTPObserver {
	client := retryablehttp.NewClient()
	client.RetryMax = 3
	client.RetryWaitMin = 100 * time.Millisecond
	client.RetryWaitMax = 1 * time.Second
	client.HTTPClient.Timeout = 5 * time.Second

	return &HTTPObserver{
		url:    url,
		client: client,
	}
}

func (h *HTTPObserver) Notify(ctx context.Context, metricID string) {
	h.NotifyBatch(ctx, []string{metricID})
}

func (h *HTTPObserver) NotifyBatch(ctx context.Context, metricIDs []string) {
	entry := AuditEntry{
		TS:        time.Now().Unix(),
		Metrics:   metricIDs,
		IPAddress: ClientIP(ctx),
	}

	data, err := entry.Marshal()
	if err != nil {
		log.Printf("Error marshaling audit entry: %v", err)
		return
	}

	req, err := retryablehttp.NewRequestWithContext(ctx, http.MethodPost, h.url, data)
	if err != nil {
		log.Printf("Error creating audit request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.client.Do(req)
	if err != nil {
		log.Printf("Error sending audit to URL: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		log.Printf("Audit server returned error status: %d", resp.StatusCode)
		return
	}

	log.Printf("Audit record sent to URL: %s", h.url)
}

func (e *AuditEntry) Marshal() ([]byte, error) {
	return e.MarshalJSON()
}

func (e *AuditEntry) MarshalJSON() ([]byte, error) {
	return e.marshalJSONInternal()
}

func (e *AuditEntry) marshalJSONInternal() ([]byte, error) {
	return json.Marshal(e)
}

type Observers []Observer

func (o Observers) Notify(ctx context.Context, metricID string) {
	for _, observer := range o {
		observer.Notify(ctx, metricID)
	}
}

func (o Observers) NotifyBatch(ctx context.Context, metricIDs []string) {
	for _, observer := range o {
		observer.NotifyBatch(ctx, metricIDs)
	}
}

func (o Observers) Close() error {
	var errs []error
	for _, obs := range o {
		if f, ok := obs.(*FileObserver); ok {
			if err := f.Close(); err != nil {
				errs = append(errs, err)
			}
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func NewObservers(filePath, url string) (Observers, error) {
	var observers Observers

	if filePath != "" {
		obs, err := NewFileObserver(filePath)
		if err != nil {
			return nil, err
		}
		observers = append(observers, obs)
	}

	if url != "" {
		observers = append(observers, NewHTTPObserver(url))
	}

	return observers, nil
}
