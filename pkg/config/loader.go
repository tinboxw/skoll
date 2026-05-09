package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// AppConfig stores runtime configuration resolved from environment variables.
type AppConfig struct {
	Server   ServerConfig
	Store    StoreConfig
	Security SecurityConfig
	Log      LogConfig
}

type ServerConfig struct {
	Address         string
	ShutdownTimeout time.Duration
}

type StoreConfig struct {
	Mode string
	DSN  string
}

type SecurityConfig struct {
	JWTSecret string
}

type LogConfig struct {
	Level string
}

// LoadFromEnv loads configuration from SKOLL_* environment variables.
func LoadFromEnv() (AppConfig, error) {
	cfg := AppConfig{
		Server: ServerConfig{
			Address:         getenvDefault("SKOLL_SERVER_ADDRESS", ":8080"),
			ShutdownTimeout: 10 * time.Second,
		},
		Store: StoreConfig{
			Mode: strings.ToLower(getenvDefault("SKOLL_STORE_MODE", "memory")),
			DSN:  os.Getenv("SKOLL_STORE_DSN"),
		},
		Security: SecurityConfig{
			JWTSecret: getenvDefault("SKOLL_JWT_SECRET", "dev-secret-change-me"),
		},
		Log: LogConfig{
			Level: strings.ToLower(getenvDefault("SKOLL_LOG_LEVEL", "info")),
		},
	}

	if raw := os.Getenv("SKOLL_SHUTDOWN_TIMEOUT"); strings.TrimSpace(raw) != "" {
		d, err := time.ParseDuration(raw)
		if err != nil {
			return AppConfig{}, fmt.Errorf("invalid SKOLL_SHUTDOWN_TIMEOUT: %w", err)
		}
		cfg.Server.ShutdownTimeout = d
	}

	if raw := os.Getenv("SKOLL_SERVER_PORT"); strings.TrimSpace(raw) != "" {
		port, err := strconv.Atoi(raw)
		if err != nil || port <= 0 || port > 65535 {
			return AppConfig{}, fmt.Errorf("invalid SKOLL_SERVER_PORT: %q", raw)
		}
		cfg.Server.Address = fmt.Sprintf(":%d", port)
	}

	return cfg, nil
}

func getenvDefault(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}
