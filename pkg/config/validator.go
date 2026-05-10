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
	switch strings.ToLower(strings.TrimSpace(cfg.Cache.Mode)) {
	case "", "memory", "local":
		if cfg.Cache.LocalSize <= 0 {
			cfg.Cache.LocalSize = 4096
		}
	case "redis":
		if strings.TrimSpace(cfg.Cache.RedisAddr) == "" {
			return errors.New("cache redis addr is required for redis mode")
		}
	case "memcached":
		if strings.TrimSpace(cfg.Cache.MemcachedAddr) == "" {
			return errors.New("cache memcached addr is required for memcached mode")
		}
	default:
		return fmt.Errorf("unsupported cache mode: %q", cfg.Cache.Mode)
	}
	switch strings.ToLower(strings.TrimSpace(cfg.Event.Mode)) {
	case "", "memory":
	case "redis":
		if strings.TrimSpace(cfg.Event.RedisAddr) == "" {
			return errors.New("event redis addr is required for redis mode")
		}
	default:
		return fmt.Errorf("unsupported event mode: %q", cfg.Event.Mode)
	}
	if strings.TrimSpace(cfg.Security.JWTSecret) == "" {
		return errors.New("jwt secret is required")
	}
	return nil
}
