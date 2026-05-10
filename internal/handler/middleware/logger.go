package middleware

import (
	"net/http"
	"time"

	"github.com/tinboxw/skoll/pkg/logging"
)

func Logger() func(http.Handler) http.Handler {
	logger := logging.New("info")
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			next.ServeHTTP(w, r)
			logger.Info("http_request", "method", r.Method, "path", r.URL.Path, "duration", time.Since(start).String())
		})
	}
}
