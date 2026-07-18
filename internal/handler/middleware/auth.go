package middleware

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/tinboxw/skoll/pkg/config"
	"github.com/tinboxw/skoll/pkg/security"
)

func Auth(jwtSecret string, skipPaths ...string) func(http.Handler) http.Handler {
	apiPrefix := resolveMiddlewareAPIPrefixFromEnv()
	skip := map[string]struct{}{
		apiPrefix + "/health":     {},
		apiPrefix + "/ready":      {},
		apiPrefix + "/docs":       {},
		apiPrefix + "/v1/plugins": {},
		apiPrefix + "/v1/auth":    {},
	}
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
				writeUnauthorized(w)
				return
			}
			claims, err := security.ParseJWT(jwtSecret, token)
			if err != nil {
				writeUnauthorized(w)
				return
			}
			r = r.WithContext(security.WithJWTClaimsContext(r.Context(), claims))
			next.ServeHTTP(w, r)
		})
	}
}

func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"code":    "unauthorized",
		"message": "invalid or missing access token",
	})
}

func resolveMiddlewareAPIPrefixFromEnv() string {
	if v := strings.TrimSpace(os.Getenv("SKOLL_API_BASE_PREFIX")); v != "" {
		return config.NormalizeAPIPrefix(v)
	}
	if v := strings.TrimSpace(os.Getenv("SKOLL_SERVER_API_PREFIX")); v != "" {
		return config.NormalizeAPIPrefix(v)
	}
	return config.DefaultAPIBasePrefix
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
