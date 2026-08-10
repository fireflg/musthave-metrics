package middleware_test

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fireflg/go-musthave-metrics-tpl/internal/crypto"
	"github.com/fireflg/go-musthave-metrics-tpl/internal/middleware"
	"go.uber.org/zap/zaptest"
)

func TestDecryptMiddleware_ValidCiphertext(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	plaintext := []byte(`[{"id":"Alloc","type":"gauge","value":1}]`)
	ciphertext, err := crypto.Encrypt(&key.PublicKey, plaintext)
	if err != nil {
		t.Fatalf("failed to encrypt: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(ciphertext))
	rr := httptest.NewRecorder()

	called := false
	h := middleware.DecryptMiddleware(func(w http.ResponseWriter, r *http.Request) {
		called = true
		data, _ := io.ReadAll(r.Body)
		if string(data) != string(plaintext) {
			t.Fatalf("expected decrypted body %s, got %s", plaintext, data)
		}
		w.WriteHeader(http.StatusOK)
	}, key, logger)

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rr.Code)
	}
	if !called {
		t.Fatal("expected handler to be called")
	}
}

func TestDecryptMiddleware_InvalidCiphertext(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer([]byte("not encrypted data")))
	rr := httptest.NewRecorder()

	called := false
	h := middleware.DecryptMiddleware(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}, key, logger)

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 BadRequest, got %d", rr.Code)
	}
	if called {
		t.Fatal("handler should not be called on invalid ciphertext")
	}
}
