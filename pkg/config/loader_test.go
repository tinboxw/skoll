package config

import "testing"

func TestLoad(t *testing.T) {
	t.Setenv("SKOLL_SERVER_ADDRESS", ":9090")
	t.Setenv("SKOLL_STORE_MODE", "memory")
	t.Setenv("SKOLL_JWT_SECRET", "secret")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load from env: %v", err)
	}
	if cfg.Server.Address != ":9090" {
		t.Fatalf("unexpected address: %s", cfg.Server.Address)
	}
}

func TestValidate(t *testing.T) {
	cfg := AppConfig{
		Server:   ServerConfig{Address: ":8080", ShutdownTimeout: 1},
		Store:    StoreConfig{Mode: "memory"},
		Security: SecurityConfig{JWTSecret: "s"},
	}
	if err := Validate(cfg); err != nil {
		t.Fatalf("validate: %v", err)
	}
}
