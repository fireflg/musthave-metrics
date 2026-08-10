package middleware

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"

	"go.uber.org/zap"
)

// SignMiddleware — HTTP мидлвар для проверки HMAC подписи.
func SignMiddleware(h http.HandlerFunc, secretKey string, logger *zap.SugaredLogger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		hashHeader := r.Header.Get("HashSHA256")

		if secretKey == "" || hashHeader == "" {
			logger.Debug("Signature check skipped")
			h.ServeHTTP(w, r)
			return
		}

		receivedSignature, err := hex.DecodeString(hashHeader)
		if err != nil {
			http.Error(w, "Invalid hash format", http.StatusBadRequest)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusInternalServerError)
			return
		}

		r.Body.Close()
		r.Body = io.NopCloser(bytes.NewBuffer(body))

		hsh := hmac.New(sha256.New, []byte(secretKey))
		hsh.Write(body)

		if !hmac.Equal(receivedSignature, hsh.Sum(nil)) {
			http.Error(w, "Invalid signature", http.StatusBadRequest)
			return
		}

		h.ServeHTTP(w, r)
	}
}
