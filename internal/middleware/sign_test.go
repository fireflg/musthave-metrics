package middleware_test

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"github.com/fireflg/ago-musthave-metrics-tpl/internal/middleware"
	"go.uber.org/zap/zaptest"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSignMiddleware_ValidSignature(t *testing.T) {
	secret := "supersecret"
	logger := zaptest.NewLogger(t).Sugar()

	body := []byte(`{"test":"value"}`)

	hsh := hmac.New(sha256.New, []byte(secret))
	hsh.Write(body)
	signature := hex.EncodeToString(hsh.Sum(nil))

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(body))
	req.Header.Set("HashSHA256", signature)
	rr := httptest.NewRecorder()

	called := false
	handler := middleware.SignMiddleware(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}, secret, logger)

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rr.Code)
	}
	if !called {
		t.Fatal("expected handler to be called")
	}
}

func TestSignMiddleware_InvalidSignature(t *testing.T) {
	secret := "supersecret"
	logger := zaptest.NewLogger(t).Sugar()

	body := []byte(`{"test":"value"}`)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(body))
	req.Header.Set("HashSHA256", "deadbeef") // неверный хэш
	rr := httptest.NewRecorder()

	called := false
	handler := middleware.SignMiddleware(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}, secret, logger)

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 BadRequest, got %d", rr.Code)
	}
	if called {
		t.Fatal("handler should not be called on invalid signature")
	}
}

func TestSignMiddleware_MissingSignature(t *testing.T) {
	secret := "supersecret"
	logger := zaptest.NewLogger(t).Sugar()

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer([]byte(`data`)))
	rr := httptest.NewRecorder()

	called := false
	handler := middleware.SignMiddleware(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}, secret, logger)

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 BadRequest, got %d", rr.Code)
	}
	if called {
		t.Fatal("handler should not be called when signature missing")
	}
}

func TestSignMiddleware_NoSecretKey(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer([]byte(`data`)))
	rr := httptest.NewRecorder()

	called := false
	handler := middleware.SignMiddleware(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}, "", logger)

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rr.Code)
	}
	if !called {
		t.Fatal("handler should be called when secretKey is empty")
	}
}

func TestSignMiddleware_BodyRestored(t *testing.T) {
	secret := "key"
	logger := zaptest.NewLogger(t).Sugar()
	body := []byte("hello world")

	hsh := hmac.New(sha256.New, []byte(secret))
	hsh.Write(body)
	sign := hex.EncodeToString(hsh.Sum(nil))

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(body))
	req.Header.Set("HashSHA256", sign)
	rr := httptest.NewRecorder()

	handler := middleware.SignMiddleware(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		if string(data) != string(body) {
			t.Fatalf("expected body %s, got %s", body, data)
		}
		w.WriteHeader(http.StatusOK)
	}, secret, logger)

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rr.Code)
	}
}
