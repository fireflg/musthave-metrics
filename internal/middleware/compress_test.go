package middleware_test

import (
	"bytes"
	"compress/gzip"
	"github.com/fireflg/ago-musthave-metrics-tpl/internal/middleware"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGzipMiddleware_ResponseCompression(t *testing.T) {
	handler := middleware.GzipMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello world"))
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, "gzip", rr.Header().Get("Content-Encoding"))

	gr, err := gzip.NewReader(rr.Body)
	assert.NoError(t, err)
	defer gr.Close()

	data, err := io.ReadAll(gr)
	assert.NoError(t, err)
	assert.Equal(t, "hello world", string(data))
}

func TestGzipMiddleware_NoCompression(t *testing.T) {
	handler := middleware.GzipMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("plain text"))
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Empty(t, rr.Header().Get("Content-Encoding"))
	assert.Equal(t, "plain text", rr.Body.String())
}

func TestGzipMiddleware_RequestDecompression(t *testing.T) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	_, err := gw.Write([]byte("compressed body"))
	assert.NoError(t, err)
	gw.Close()

	handler := middleware.GzipMiddleware(func(w http.ResponseWriter, r *http.Request) {
		data, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.Equal(t, "compressed body", string(data))
		w.Write([]byte("ok"))
	})

	req := httptest.NewRequest(http.MethodPost, "/", &buf)
	req.Header.Set("Content-Encoding", "gzip")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, "ok", rr.Body.String())
}

func TestGzipMiddleware_RequestDecompressionError(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("not gzip"))
	req.Header.Set("Content-Encoding", "gzip")
	rr := httptest.NewRecorder()

	handler := middleware.GzipMiddleware(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	})

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}
