package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/tinboxw/skoll/pkg/security"
)

func Auth(jwtSecret string, skipPaths ...string) func(http.Handler) http.Handler {
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
			token, err := parseBearerToken(r.Header.Get("Authorization"))
			if err != nil {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte("unauthorized"))
				return
			}
			claims, err := security.ParseJWT(jwtSecret, token)
			if err != nil {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte("unauthorized"))
				return
			}
			r = r.WithContext(security.WithJWTClaimsContext(r.Context(), claims))
			next.ServeHTTP(w, r)
		})
	}
}

func parseBearerToken(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", errors.New("missing authorization header")
	}
	parts := strings.SplitN(value, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", errors.New("invalid authorization scheme")
	}
	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", errors.New("missing bearer token")
	}
	return token, nil
}
