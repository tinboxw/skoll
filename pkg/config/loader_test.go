package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	t.Setenv("SKOLL_SERVER_ADDRESS", ":9090")
	t.Setenv("SKOLL_STORE_MODE", "memory")
	t.Setenv("SKOLL_EVENT_MODE", "redis")
	t.Setenv("SKOLL_EVENT_REDIS_ADDR", "127.0.0.1:6379")
	t.Setenv("SKOLL_EVENT_CHANNEL_PREFIX", "skoll.test.events")
	t.Setenv("SKOLL_SECURITY_JWT_SECRET", "secret")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load from env: %v", err)
	}
	if cfg.Server.Address != ":9090" {
		t.Fatalf("unexpected address: %s", cfg.Server.Address)
	}
	if cfg.Event.Mode != "redis" {
		t.Fatalf("unexpected event mode: %s", cfg.Event.Mode)
	}
	if cfg.Event.RedisAddr != "127.0.0.1:6379" {
		t.Fatalf("unexpected event redis addr: %s", cfg.Event.RedisAddr)
	}
	if cfg.Event.ChannelPrefix != "skoll.test.events" {
		t.Fatalf("unexpected event channel prefix: %s", cfg.Event.ChannelPrefix)
	}
}

func TestLoadFromConfigFile(t *testing.T) {
	configFile := filepath.Join(t.TempDir(), "skoll.yaml")
	content := []byte("server:\n  address: :7000\nevent:\n  mode: redis\n  redis_addr: 127.0.0.1:6379\n  channel_prefix: skoll.cfg.events\nsecurity:\n  jwt_secret: cfg-secret\n")
	if err := os.WriteFile(configFile, content, 0o644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	t.Setenv("SKOLL_CONFIG_FILE", configFile)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load from config file: %v", err)
	}
	if cfg.Server.Address != ":7000" {
		t.Fatalf("unexpected address: %s", cfg.Server.Address)
	}
	if cfg.Event.Mode != "redis" {
		t.Fatalf("unexpected event mode: %s", cfg.Event.Mode)
	}
	if cfg.Event.RedisAddr != "127.0.0.1:6379" {
		t.Fatalf("unexpected event redis addr: %s", cfg.Event.RedisAddr)
	}
}

func TestLoadEnvOverridesConfigFile(t *testing.T) {
	configFile := filepath.Join(t.TempDir(), "skoll.yaml")
	content := []byte("server:\n  address: :7000\nstore:\n  mode: memory\nsecurity:\n  jwt_secret: cfg-secret\n")
	if err := os.WriteFile(configFile, content, 0o644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	t.Setenv("SKOLL_CONFIG_FILE", configFile)
	t.Setenv("SKOLL_SERVER_ADDRESS", ":7001")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load with env override: %v", err)
	}
	if cfg.Server.Address != ":7001" {
		t.Fatalf("env should override config file, got: %s", cfg.Server.Address)
	}
}

func TestLoadFromDefaultConfigCandidate(t *testing.T) {
	dir := t.TempDir()
	configFile := filepath.Join(dir, "skoll.yaml")
	content := []byte("server:\n  address: :7010\nsecurity:\n  jwt_secret: cfg-secret\n")
	if err := os.WriteFile(configFile, content, 0o644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	t.Chdir(dir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load from default candidate: %v", err)
	}
	if cfg.Server.Address != ":7010" {
		t.Fatalf("unexpected address: %s", cfg.Server.Address)
	}
}

func TestValidate(t *testing.T) {
	cfg := AppConfig{
		Server:   ServerConfig{Address: ":8080", ShutdownTimeout: 1},
		Store:    StoreConfig{Mode: "memory"},
		Event:    EventConfig{Mode: "memory"},
		Security: SecurityConfig{JWTSecret: "s"},
	}
	if err := Validate(cfg); err != nil {
		t.Fatalf("validate: %v", err)
	}
}

func TestValidateRedisEventModeRequiresAddr(t *testing.T) {
	cfg := AppConfig{
		Server:   ServerConfig{Address: ":8080", ShutdownTimeout: 1},
		Store:    StoreConfig{Mode: "memory"},
		Event:    EventConfig{Mode: "redis"},
		Security: SecurityConfig{JWTSecret: "s"},
	}
	if err := Validate(cfg); err == nil {
		t.Fatalf("expected redis event mode validation error")
	}
}
