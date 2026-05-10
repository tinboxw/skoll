package config

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// AppConfig stores runtime configuration resolved from environment variables.
type AppConfig struct {
	Server   ServerConfig
	Store    StoreConfig
	Event    EventConfig
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

type EventConfig struct {
	Mode          string
	RedisAddr     string
	ChannelPrefix string
}

type SecurityConfig struct {
	JWTSecret string
}

type LogConfig struct {
	Level string
}

// Load loads configuration via Viper using SKOLL_* environment variables.
func Load() (AppConfig, error) {
	v := viper.New()
	v.SetEnvPrefix("SKOLL")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	v.SetDefault("server.address", ":8080")
	v.SetDefault("server.shutdown_timeout", "10s")
	v.SetDefault("store.mode", "memory")
	v.SetDefault("store.dsn", "")
	v.SetDefault("event.mode", "memory")
	v.SetDefault("event.redis_addr", "")
	v.SetDefault("event.channel_prefix", "skoll.events")
	v.SetDefault("security.jwt_secret", "dev-secret-change-me")
	v.SetDefault("log.level", "info")
	v.SetDefault("server.port", "")

	shutdownRaw := strings.TrimSpace(v.GetString("server.shutdown_timeout"))
	shutdownTimeout, err := time.ParseDuration(shutdownRaw)
	if err != nil {
		return AppConfig{}, fmt.Errorf("invalid SKOLL_SERVER_SHUTDOWN_TIMEOUT: %w", err)
	}

	address := strings.TrimSpace(v.GetString("server.address"))
	if address == "" {
		address = ":8080"
	}

	portRaw := strings.TrimSpace(v.GetString("server.port"))
	if portRaw != "" {
		port, err := strconv.Atoi(portRaw)
		if err != nil || port <= 0 || port > 65535 {
			return AppConfig{}, fmt.Errorf("invalid SKOLL_SERVER_PORT: %q", portRaw)
		}
		address = fmt.Sprintf(":%d", port)
	}

	return AppConfig{
		Server: ServerConfig{
			Address:         address,
			ShutdownTimeout: shutdownTimeout,
		},
		Store: StoreConfig{
			Mode: strings.ToLower(strings.TrimSpace(v.GetString("store.mode"))),
			DSN:  strings.TrimSpace(v.GetString("store.dsn")),
		},
		Event: EventConfig{
			Mode:          strings.ToLower(strings.TrimSpace(v.GetString("event.mode"))),
			RedisAddr:     strings.TrimSpace(v.GetString("event.redis_addr")),
			ChannelPrefix: strings.TrimSpace(v.GetString("event.channel_prefix")),
		},
		Security: SecurityConfig{
			JWTSecret: strings.TrimSpace(v.GetString("security.jwt_secret")),
		},
		Log: LogConfig{
			Level: strings.ToLower(strings.TrimSpace(v.GetString("log.level"))),
		},
	}, nil
}
