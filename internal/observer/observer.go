// Package observer предоставляет функциональность аудита для отслеживания изменений метрик.
//
// Observer поддерживает несколько бэкендов: файловый и HTTP.
// Каждый observer может использоваться независимо или в комбинации.
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

// ClientIP возвращает IP клиента из контекста.
func ClientIP(ctx context.Context) string {
	if v := ctx.Value(clientIPKey); v != nil {
		if ip, ok := v.(string); ok {
			return ip
		}
	}
	return ""
}

// WithClientIP возвращает новый контекст с установленным IP клиента.
func WithClientIP(ctx context.Context, ip string) context.Context {
	return context.WithValue(ctx, clientIPKey, ip)
}

// AuditEntry представляет одну запись аудита.
type AuditEntry struct {
	TS        int64    `json:"ts"`         // Время события аудита.
	Metrics   []string `json:"metrics"`    // Список ID метрик, которые были изменены.
	IPAddress string   `json:"ip_address"` // IP адрес клиента, вызвавшего изменение.
}

// Observer — интерфейс для наблюдателей аудита.
type Observer interface {
	// Notify уведомляет об изменении одной метрики.
	Notify(ctx context.Context, metricID string)
	// NotifyBatch уведомляет об изменении нескольких метрик.
	NotifyBatch(ctx context.Context, metricIDs []string)
}

// FileObserver записывает записи аудита в файл.
type FileObserver struct {
	filePath string
	file     *os.File
	mu       sync.Mutex
}

// NewFileObserver создает новый FileObserver.
func NewFileObserver(filePath string) (*FileObserver, error) {
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	return &FileObserver{filePath: filePath, file: file}, nil
}

// Close закрывает файл аудита.
func (f *FileObserver) Close() error {
	if f.file != nil {
		return f.file.Close()
	}
	return nil
}

// Notify записывает изменение одной метрики в файл аудита.
func (f *FileObserver) Notify(ctx context.Context, metricID string) {
	f.NotifyBatch(ctx, []string{metricID})
}

// NotifyBatch записывает изменения нескольких метрик в файл аудита.
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

// HTTPObserver отправляет записи аудита на удаленный URL с поддержкой ретраев.
type HTTPObserver struct {
	url    string
	client *retryablehttp.Client
}

// NewHTTPObserver создает новый HTTPObserver с поддержкой ретраев.
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

// Notify отправляет изменение одной метрики на удаленный URL.
func (h *HTTPObserver) Notify(ctx context.Context, metricID string) {
	h.NotifyBatch(ctx, []string{metricID})
}

// NotifyBatch отправляет изменения нескольких метрик на удаленный URL с автоматическими ретраями.
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

// Marshal сериализует AuditEntry в JSON.
func (e *AuditEntry) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

// Observers — список наблюдателей, которые могут уведомлять несколько бэкендов.
type Observers []Observer

// Notify вызывает Notify на всех наблюдателях.
func (o Observers) Notify(ctx context.Context, metricID string) {
	for _, observer := range o {
		observer.Notify(ctx, metricID)
	}
}

// NotifyBatch вызывает NotifyBatch на всех наблюдателях.
func (o Observers) NotifyBatch(ctx context.Context, metricIDs []string) {
	for _, observer := range o {
		observer.NotifyBatch(ctx, metricIDs)
	}
}

// Close закрывает всех наблюдателей, требующих очистки.
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

// NewObservers создает список наблюдателей на основе конфигурации.
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