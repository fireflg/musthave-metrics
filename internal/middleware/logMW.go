package middleware

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

func WithLogging(logger *zap.SugaredLogger) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			lrw := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			h.ServeHTTP(lrw, r)

			duration := time.Since(start)

			logger.Infoln(
				"method", r.Method,
				"uri", r.RequestURI,
				"status", lrw.statusCode,
				"duration", duration,
			)
		})
	}
}

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}
