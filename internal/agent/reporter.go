package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/hmac"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/fireflg/go-musthave-metrics-tpl/internal/crypto"
	"github.com/hashicorp/go-retryablehttp"
)

// probeTimeout ограничивает пробное соединение для определения локального IP.
const probeTimeout = 2 * time.Second

// Reporter отправляет метрики на удаленный сервер с поддержкой ретраев.
type Reporter struct {
	serverURL string
	client    *retryablehttp.Client
	secretKey string
	publicKey *rsa.PublicKey
	localIP   *lazyIP
}

// MetricsReporter определяет интерфейс для отправки метрик на сервер.
type MetricsReporter interface {
	// Report отправляет набор метрик на сервер.
	Report(ctx context.Context, metrics Metrics) error
	// WaitServer ожидает доступности сервера.
	WaitServer(ctx context.Context) error
}

// NewReporter создает новый экземпляр Reporter.
func NewReporter(serverURL string, secretKey string, publicKey *rsa.PublicKey) *Reporter {
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
		publicKey: publicKey,
		localIP:   newLazyIP(serverURL),
	}
}

// lazyIP определяет локальный IP при первом обращении.
type lazyIP struct {
	serverURL string
	once      sync.Once
	ip        string
}

func newLazyIP(serverURL string) *lazyIP {
	return &lazyIP{serverURL: serverURL}
}

func (l *lazyIP) get() string {
	l.once.Do(func() { l.ip = outboundIP(l.serverURL) })
	return l.ip
}

// outboundIP определяет IP-адрес хоста агента, который используется для
// исходящих соединений к серверу serverURL. Если определить его не удалось,
// возвращает пустую строку — заголовок X-Real-IP в этом случае не отправляется.
func outboundIP(serverURL string) string {
	host := serverURL
	if u, err := url.Parse(serverURL); err == nil && u.Host != "" {
		host = u.Host
	}

	if conn, err := net.DialTimeout("tcp", host, probeTimeout); err == nil {
		defer conn.Close()
		if tcpAddr, ok := conn.LocalAddr().(*net.TCPAddr); ok {
			return tcpAddr.IP.String()
		}
		return ""
	}

	conn, err := net.Dial("udp", host)
	if err != nil {
		return ""
	}
	defer conn.Close()

	udpAddr, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok {
		return ""
	}
	return udpAddr.IP.String()
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

	body := compressed
	if r.publicKey != nil {
		body, err = crypto.Encrypt(r.publicKey, compressed)
		if err != nil {
			return fmt.Errorf("failed to encrypt payload: %w", err)
		}
	}

	url := fmt.Sprintf("%s/updates/", r.serverURL)

	req, err := retryablehttp.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")

	if localIP := r.localIP.get(); localIP != "" {
		req.Header.Set("X-Real-IP", localIP)
	}

	if len(r.secretKey) > 0 {
		hash, err = r.signPayload(payload)
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
