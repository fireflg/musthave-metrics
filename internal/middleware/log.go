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

			lrw := &loggingResponseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			h.ServeHTTP(lrw, r)

			duration := time.Since(start)

			defer func() {
				if err := recover(); err != nil {
					logger.Errorw(
						"panic recovered",
						"method", r.Method,
						"uri", r.RequestURI,
						"panic", err,
					)
					http.Error(w, "internal server error", http.StatusInternalServerError)
				}
			}()

			if lrw.statusCode >= 400 {
				logger.Errorw(
					"http request error",
					"method", r.Method,
					"uri", r.RequestURI,
					"status", lrw.statusCode,
					"duration", duration,
				)
				return
			}

			logger.Infow(
				"http request",
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

func (lrw *loggingResponseWriter) Write(b []byte) (int, error) {
	return lrw.ResponseWriter.Write(b)
}
