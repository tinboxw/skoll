package middleware

import (
	"net/http"
	"strings"
)

func Auth(skipPaths ...string) func(http.Handler) http.Handler {
	skip := map[string]struct{}{"/health": {}, "/ready": {}, "/v1/plugins": {}, "/v1/auth": {}}
	for _, p := range skipPaths {
		skip[p] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for p := range skip {
				if r.URL.Path == p || strings.HasPrefix(r.URL.Path, p+"/") {
					next.ServeHTTP(w, r)
					return
				}
			}
			if r.Header.Get("Authorization") == "" {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte("unauthorized"))
				return
			}
			if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(r.Header.Get("Authorization"))), "bearer ") {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte("unauthorized"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
