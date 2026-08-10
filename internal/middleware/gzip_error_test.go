package middleware_test

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fireflg/go-musthave-metrics-tpl/internal/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestGzipMiddleware_ErrorResponseIsDecodable(t *testing.T) {
	h := middleware.GzipMiddleware(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	require.Equal(t, http.StatusInternalServerError, rr.Code)
	require.Equal(t, "gzip", rr.Header().Get("Content-Encoding"))

	zr, err := gzip.NewReader(rr.Body)
	require.NoError(t, err)
	defer zr.Close()

	body, err := io.ReadAll(zr)
	require.NoError(t, err)
	assert.Contains(t, string(body), "boom")
}

func TestGzipMiddleware_NoContentLengthOnCompressedResponse(t *testing.T) {
	h := middleware.GzipMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "5")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello"))
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	assert.Empty(t, rr.Header().Get("Content-Length"))
}

func TestWithLogging_RecoversPanic(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()

	h := middleware.WithLogging(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	require.NotPanics(t, func() { h.ServeHTTP(rr, req) })
	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestWithLogging_PassesThrough(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()

	h := middleware.WithLogging(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusTeapot, rr.Code)
	assert.Equal(t, "ok", rr.Body.String())
}
