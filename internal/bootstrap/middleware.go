package bootstrap

import (
	"net/http"
	"time"

	apperrors "github.com/tinboxw/skoll/pkg/errors"
	"github.com/tinboxw/skoll/pkg/logging"
)

func buildMiddlewareChain(next http.Handler, logger logging.Logger, policy AuthPolicy) http.Handler {
	h := recoverMiddleware(logger, next)
	h = accessLogMiddleware(logger, h)
	h = authGuardMiddleware(policy, h)
	return h
}

func authGuardMiddleware(policy AuthPolicy, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !policy.ShouldAuthenticate(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		if r.Header.Get("Authorization") == "" {
			apperrors.WriteHTTP(w, apperrors.New("unauthorized", "missing authorization header", nil))
			return
		}

		next.ServeHTTP(w, r)
	})
}

func accessLogMiddleware(logger logging.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		logger.Info("http_request", "method", r.Method, "path", r.URL.Path, "duration", time.Since(start).String())
	})
}

func recoverMiddleware(logger logging.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				logger.Error("panic recovered", "panic", rec)
				apperrors.WriteHTTP(w, apperrors.New("internal_error", "internal server error", nil))
			}
		}()

		next.ServeHTTP(w, r)
	})
}
