package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// AppConfig stores runtime configuration resolved from environment variables.
type AppConfig struct {
	Server   ServerConfig
	Store    StoreConfig
	Cache    CacheConfig
	Event    EventConfig
	Security SecurityConfig
	Log      LogConfig
}

type ServerConfig struct {
	Address         string
	APIPrefix       string
	ShutdownTimeout time.Duration
}

type StoreConfig struct {
	Mode string
	DSN  string
}

type CacheConfig struct {
	Mode          string
	RedisAddr     string
	MemcachedAddr string
	LocalSize     int
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
	Level         string
	Dir           string
	File          string
	PluginPerFile bool
}

var defaultConfigCandidates = []string{
	"skoll.yaml",
	"skoll.yml",
	"config/skoll.yaml",
	"config/skoll.yml",
	"configs/skoll.yaml",
	"configs/skoll.yml",
}

// Load loads configuration with precedence: environment variables > config file > defaults.
func Load() (AppConfig, error) {
	v := viper.New()
	v.SetEnvPrefix("SKOLL")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	v.SetDefault("server.address", ":8080")
	v.SetDefault("server.api_prefix", "/api")
	v.SetDefault("server.shutdown_timeout", "10s")
	v.SetDefault("store.mode", "mysql")
	v.SetDefault("store.dsn", "root:root@tcp(127.0.0.1:3306)/skoll?charset=utf8mb4&parseTime=True&loc=Local")
	v.SetDefault("cache.mode", "memory")
	v.SetDefault("cache.redis_addr", "127.0.0.1:6379")
	v.SetDefault("cache.memcached_addr", "127.0.0.1:11211")
	v.SetDefault("cache.local_size", 4096)
	v.SetDefault("event.mode", "memory")
	v.SetDefault("event.redis_addr", "")
	v.SetDefault("event.channel_prefix", "skoll.events")
	v.SetDefault("security.jwt_secret", "dev-secret-change-me")
	v.SetDefault("log.level", "info")
	v.SetDefault("log.dir", "log")
	v.SetDefault("log.file", "")
	v.SetDefault("log.plugin_per_file", false)
	v.SetDefault("server.port", "")

	if err := loadConfigFile(v); err != nil {
		return AppConfig{}, err
	}

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
			APIPrefix:       normalizeAPIPrefix(v.GetString("server.api_prefix")),
			ShutdownTimeout: shutdownTimeout,
		},
		Store: StoreConfig{
			Mode: strings.ToLower(strings.TrimSpace(v.GetString("store.mode"))),
			DSN:  strings.TrimSpace(v.GetString("store.dsn")),
		},
		Cache: CacheConfig{
			Mode:          strings.ToLower(strings.TrimSpace(v.GetString("cache.mode"))),
			RedisAddr:     strings.TrimSpace(v.GetString("cache.redis_addr")),
			MemcachedAddr: strings.TrimSpace(v.GetString("cache.memcached_addr")),
			LocalSize:     v.GetInt("cache.local_size"),
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
			Level:         strings.ToLower(strings.TrimSpace(v.GetString("log.level"))),
			Dir:           strings.TrimSpace(v.GetString("log.dir")),
			File:          strings.TrimSpace(v.GetString("log.file")),
			PluginPerFile: v.GetBool("log.plugin_per_file"),
		},
	}, nil
}

func normalizeAPIPrefix(raw string) string {
	v := strings.TrimSpace(raw)
	if v == "" {
		return "/api"
	}
	if !strings.HasPrefix(v, "/") {
		v = "/" + v
	}
	v = strings.TrimRight(v, "/")
	if v == "" {
		return "/api"
	}
	return v
}

func loadConfigFile(v *viper.Viper) error {
	configFile := strings.TrimSpace(v.GetString("config.file"))
	if configFile != "" {
		v.SetConfigFile(configFile)
		if err := v.ReadInConfig(); err != nil {
			return fmt.Errorf("read config file %q: %w", configFile, err)
		}
		return nil
	}

	for _, candidate := range defaultConfigCandidates {
		if _, err := os.Stat(candidate); err == nil {
			v.SetConfigFile(candidate)
			if readErr := v.ReadInConfig(); readErr != nil {
				return fmt.Errorf("read config file %q: %w", candidate, readErr)
			}
			return nil
		}
	}

	return nil
}
