package config

import "testing"

func TestLoad(t *testing.T) {
	t.Setenv("SKOLL_SERVER_ADDRESS", ":9090")
	t.Setenv("SKOLL_STORE_MODE", "memory")
	t.Setenv("SKOLL_EVENT_MODE", "redis")
	t.Setenv("SKOLL_EVENT_REDIS_ADDR", "127.0.0.1:6379")
	t.Setenv("SKOLL_EVENT_CHANNEL_PREFIX", "skoll.test.events")
	t.Setenv("SKOLL_JWT_SECRET", "secret")

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
