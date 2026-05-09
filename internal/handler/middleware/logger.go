package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

func Logger() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			next.ServeHTTP(w, r)
			slog.Info("http_request", "method", r.Method, "path", r.URL.Path, "duration", time.Since(start).String())
		})
	}
}
