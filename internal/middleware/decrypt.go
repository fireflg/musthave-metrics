package middleware

import (
	"bytes"
	"crypto/rsa"
	"io"
	"net/http"

	"github.com/fireflg/go-musthave-metrics-tpl/internal/crypto"
	"go.uber.org/zap"
)

// DecryptMiddleware — HTTP мидлварь для расшифровки тела запроса приватным
// ключом privateKey. Расшифровка выполняется до распаковки gzip, так как
// агент шифрует уже сжатые данные.
func DecryptMiddleware(h http.HandlerFunc, privateKey *rsa.PrivateKey, logger *zap.SugaredLogger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusInternalServerError)
			return
		}

		decrypted, err := crypto.Decrypt(privateKey, body)
		if err != nil {
			logger.Errorw("Failed to decrypt request body", "error", err)
			http.Error(w, "Failed to decrypt request body", http.StatusBadRequest)
			return
		}

		r.Body = io.NopCloser(bytes.NewReader(decrypted))
		r.ContentLength = int64(len(decrypted))

		h.ServeHTTP(w, r)
	}
}
