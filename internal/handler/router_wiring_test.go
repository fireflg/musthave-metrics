package handler_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fireflg/go-musthave-metrics-tpl/internal/crypto"
	"github.com/fireflg/go-musthave-metrics-tpl/internal/handler"
	models "github.com/fireflg/go-musthave-metrics-tpl/internal/model"
	"github.com/fireflg/go-musthave-metrics-tpl/internal/repository/memory"
	"github.com/fireflg/go-musthave-metrics-tpl/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func gzipBytes(t *testing.T, data []byte) []byte {
	t.Helper()

	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, err := zw.Write(data)
	require.NoError(t, err)
	require.NoError(t, zw.Close())
	return buf.Bytes()
}

func TestRouter_DecryptionIsWired(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	repo := memory.NewMemoryRepository()
	svc := service.NewMetricsService(repo, nil)
	r := handler.NewMetricsHandler(svc, zaptest.NewLogger(t).Sugar(), "", key, nil).ServerRouter()

	value := 5.5
	body, err := json.Marshal(models.Metrics{ID: "Alloc", MType: "gauge", Value: &value})
	require.NoError(t, err)

	encrypted, err := crypto.Encrypt(&key.PublicKey, gzipBytes(t, body))
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewReader(encrypted))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

	stored, err := repo.GetGauge(context.Background(), "Alloc")
	require.NoError(t, err)
	assert.Equal(t, 5.5, stored)
}

func TestRouter_DecryptionRejectsPlaintext(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	svc := service.NewMetricsService(memory.NewMemoryRepository(), nil)
	r := handler.NewMetricsHandler(svc, zaptest.NewLogger(t).Sugar(), "", key, nil).ServerRouter()

	rr := doRequest(t, r, http.MethodPost, "/update/", `{"id":"Alloc","type":"gauge","value":1}`)
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestRouter_TrustedSubnetIsWired(t *testing.T) {
	_, subnet, err := net.ParseCIDR("10.0.0.0/8")
	require.NoError(t, err)

	svc := service.NewMetricsService(memory.NewMemoryRepository(), nil)
	r := handler.NewMetricsHandler(svc, zaptest.NewLogger(t).Sugar(), "", nil, subnet).ServerRouter()

	t.Run("без заголовка", func(t *testing.T) {
		rr := doRequest(t, r, http.MethodPost, "/update/gauge/Alloc/1", "")
		assert.Equal(t, http.StatusForbidden, rr.Code)
	})

	t.Run("чужой IP", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/update/gauge/Alloc/1", nil)
		req.Header.Set("X-Real-IP", "192.168.1.1")

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusForbidden, rr.Code)
	})

	t.Run("доверенный IP", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/update/gauge/Alloc/1", nil)
		req.Header.Set("X-Real-IP", "10.1.2.3")

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
	})
}

type brokenRepo struct {
	models.MetricsRepository
	metric *models.Metrics
	pinged error
}

func (r *brokenRepo) GetMetric(_ context.Context, _, _ string) (*models.Metrics, error) {
	return r.metric, nil
}

func (r *brokenRepo) Ping(_ context.Context) error { return r.pinged }

func TestGetMetric_BrokenStoredValues(t *testing.T) {
	tests := []struct {
		name   string
		metric *models.Metrics
		path   string
		want   string
	}{
		{
			name:   "gauge без value",
			metric: &models.Metrics{ID: "Alloc", MType: "gauge"},
			path:   "/value/gauge/Alloc",
			want:   "gauge value is nil",
		},
		{
			name:   "counter без delta",
			metric: &models.Metrics{ID: "PollCount", MType: "counter"},
			path:   "/value/counter/PollCount",
			want:   "counter delta is nil",
		},
		{
			name:   "неизвестный тип",
			metric: &models.Metrics{ID: "X", MType: "histogram"},
			path:   "/value/histogram/X",
			want:   "unknown metric type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := service.NewMetricsService(&brokenRepo{metric: tt.metric}, nil)
			r := handler.NewMetricsHandler(svc, zaptest.NewLogger(t).Sugar(), "", nil, nil).ServerRouter()

			rr := doRequest(t, r, http.MethodGet, tt.path, "")
			assert.Equal(t, http.StatusInternalServerError, rr.Code)
			assert.Contains(t, rr.Body.String(), tt.want)
		})
	}
}

func TestCheckDB_RepositoryDown(t *testing.T) {
	svc := service.NewMetricsService(&brokenRepo{pinged: errors.New("db down")}, nil)
	r := handler.NewMetricsHandler(svc, zaptest.NewLogger(t).Sugar(), "", nil, nil).ServerRouter()

	rr := doRequest(t, r, http.MethodGet, "/ping", "")
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}
