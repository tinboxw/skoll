package bootstrap

import (
	"context"
	"fmt"

	"github.com/tinboxw/skoll/internal/integration/adminauth"
)

func resolveAdminNonceStore(authMode, authToken, hmacSecret, storeMode, redisAddr, redisPassword string, redisDB int, redisKeyPrefix string) (adminauth.ReplayNonceStore, func(context.Context) error, error) {
	if !isHMACFlowEnabled(authMode, authToken, hmacSecret) {
		return nil, nil, nil
	}

	resolvedMode := storeMode
	if resolvedMode == "" {
		resolvedMode = "auto"
	}

	switch resolvedMode {
	case "auto":
		if redisAddr == "" {
			return nil, nil, nil
		}
		cfg := adminauth.RedisNonceStoreConfig{Addr: redisAddr, Password: redisPassword, DB: redisDB, KeyPrefix: redisKeyPrefix}
		return adminauth.NewRedisNonceStoreFromConfig(cfg)
	case "memory":
		return nil, nil, nil
	case "redis":
		cfg := adminauth.RedisNonceStoreConfig{Addr: redisAddr, Password: redisPassword, DB: redisDB, KeyPrefix: redisKeyPrefix}
		return adminauth.NewRedisNonceStoreFromConfig(cfg)
	default:
		return nil, nil, fmt.Errorf("unsupported admin auth nonce store mode %q, valid: auto|memory|redis", resolvedMode)
	}
}

func isHMACFlowEnabled(mode, token, hmacSecret string) bool {
	switch mode {
	case "", "auto":
		return token == "" && hmacSecret != ""
	case "hmac-sha256":
		return hmacSecret != ""
	default:
		return false
	}
}

func validateAdminAuthPolicy(goAdminEnabled bool, goAdminMode, effectiveAdminAuthMode string, allowStaticTokenInProd bool) error {
	if !goAdminEnabled {
		return nil
	}
	if goAdminMode != "prod" {
		return nil
	}
	if effectiveAdminAuthMode != "static-token" {
		return nil
	}
	if allowStaticTokenInProd {
		return nil
	}
	return fmt.Errorf("static-token admin auth is not allowed in prod mode unless admin-auth-allow-static-token-in-prod=true")
}
