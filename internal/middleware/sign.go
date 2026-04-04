package middleware

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"go.uber.org/zap"
	"io"
	"net/http"
)

func SignMiddleware(h http.HandlerFunc, secretKey string, logger *zap.SugaredLogger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if secretKey != "" {
			hashHeader := r.Header.Get("HashSHA256")

			if hashHeader != "" {
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
				expectedSignature := hsh.Sum(nil)

				if !hmac.Equal(receivedSignature, expectedSignature) {
					http.Error(w, "Invalid signature", http.StatusBadRequest)
					return
				}
			} else {
				http.Error(w, "Signature required", http.StatusBadRequest)
				return
			}
		} else {
			logger.Info("No secret key provided")
		}

		h.ServeHTTP(w, r)
	}
}
