package middleware

import (
	"net/http"
	"time"
)

func RateLimit(rps int, burst int) func(http.Handler) http.Handler {
	if rps <= 0 {
		rps = 50
	}
	if burst <= 0 {
		burst = rps
	}

	tokens := make(chan struct{}, burst)
	for i := 0; i < burst; i++ {
		tokens <- struct{}{}
	}
	ticker := time.NewTicker(time.Second / time.Duration(rps))
	go func() {
		for range ticker.C {
			select {
			case tokens <- struct{}{}:
			default:
			}
		}
	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			select {
			case <-tokens:
				next.ServeHTTP(w, r)
			default:
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte("rate limited"))
			}
		})
	}
}
