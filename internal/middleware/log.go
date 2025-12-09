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

			fields := []interface{}{
				"method", r.Method,
				"uri", r.RequestURI,
				"status", lrw.statusCode,
				"duration", duration,
			}

			if lrw.statusCode >= 400 {
				logger.Errorw("request error", fields...)
			} else {
				logger.Infow("request", fields...)
			}
		})
	}
}

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode  int
	wroteHeader bool
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	if !lrw.wroteHeader {
		lrw.statusCode = code
		lrw.wroteHeader = true
	}
	lrw.ResponseWriter.WriteHeader(code)
}
