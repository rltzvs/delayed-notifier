package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

type responseWriterWithStatus struct {
	http.ResponseWriter
	status int
}

func (w *responseWriterWithStatus) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func LoggingMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := &responseWriterWithStatus{ResponseWriter: w, status: 200}
			next.ServeHTTP(rw, r)
			duration := time.Since(start)
			logger.Info("http request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", rw.status),
				slog.Duration("duration", duration),
			)
		})
	}
}
