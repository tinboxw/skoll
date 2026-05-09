package config

import (
	"errors"
	"fmt"
	"strings"
)

// Validate validates application configuration.
func Validate(cfg AppConfig) error {
	if strings.TrimSpace(cfg.Server.Address) == "" {
		return errors.New("server address is required")
	}
	if cfg.Server.ShutdownTimeout <= 0 {
		return errors.New("shutdown timeout must be > 0")
	}
	switch cfg.Store.Mode {
	case "memory", "mysql", "postgres":
	default:
		return fmt.Errorf("unsupported store mode: %q", cfg.Store.Mode)
	}
	if (cfg.Store.Mode == "mysql" || cfg.Store.Mode == "postgres") && strings.TrimSpace(cfg.Store.DSN) == "" {
		return errors.New("store dsn is required for sql modes")
	}
	if strings.TrimSpace(cfg.Security.JWTSecret) == "" {
		return errors.New("jwt secret is required")
	}
	return nil
}
