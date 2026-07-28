package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
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
	if cfg.Cache.Mode != "memory" {
		t.Fatalf("unexpected cache mode: %s", cfg.Cache.Mode)
	}
	if cfg.Plugin.Quota.RequestConcurrency != 32 || cfg.Plugin.Quota.MaxPendingJobs != 100 {
		t.Fatalf("unexpected plugin quota defaults: %+v", cfg.Plugin.Quota)
	}
}

func TestLoadPluginQuotaEnvOverrides(t *testing.T) {
	t.Setenv("SKOLL_PLUGIN_QUOTA_REQUEST_CONCURRENCY", "7")
	t.Setenv("SKOLL_PLUGIN_QUOTA_MAX_STORAGE_BYTES", "67108864")
	t.Setenv("SKOLL_PLUGIN_QUOTA_REQUEST_TIMEOUT", "12s")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load plugin quota env: %v", err)
	}
	if cfg.Plugin.Quota.RequestConcurrency != 7 ||
		cfg.Plugin.Quota.MaxStorageBytes != 67108864 ||
		cfg.Plugin.Quota.RequestTimeout != 12*time.Second {
		t.Fatalf("plugin quota env was not applied: %+v", cfg.Plugin.Quota)
	}
}

func TestLoadFromConfigFile(t *testing.T) {
	configFile := filepath.Join(t.TempDir(), "skoll.yaml")
	content := []byte("server:\n  address: :7000\ncache:\n  mode: redis\n  redis_addr: 127.0.0.1:6379\nevent:\n  mode: redis\n  redis_addr: 127.0.0.1:6379\n  channel_prefix: skoll.cfg.events\nsecurity:\n  jwt_secret: cfg-secret\n")
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
	if cfg.Cache.Mode != "redis" {
		t.Fatalf("unexpected cache mode: %s", cfg.Cache.Mode)
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

func TestLoadEnvOverridesCacheModeFromConfigFile(t *testing.T) {
	configFile := filepath.Join(t.TempDir(), "skoll.yaml")
	content := []byte("cache:\n  mode: redis\n  redis_addr: 127.0.0.1:6379\nsecurity:\n  jwt_secret: cfg-secret\n")
	if err := os.WriteFile(configFile, content, 0o644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	t.Setenv("SKOLL_CONFIG_FILE", configFile)
	t.Setenv("SKOLL_CACHE_MODE", "memory")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load with env override: %v", err)
	}
	if cfg.Cache.Mode != "memory" {
		t.Fatalf("env should override cache mode, got: %s", cfg.Cache.Mode)
	}
}

func TestLoadUsesUnifiedAPIBasePrefixEnv(t *testing.T) {
	t.Setenv("SKOLL_API_BASE_PREFIX", "/gateway")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load with unified api base prefix: %v", err)
	}
	if cfg.Server.APIPrefix != "/gateway" {
		t.Fatalf("expected unified api base prefix to win, got: %s", cfg.Server.APIPrefix)
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

func TestLoadParsesDevPluginsRootAllowlist(t *testing.T) {
	t.Setenv("SKOLL_DEV_PLUGINS_ROOT", "plugins;D:/workspace/skoll-apps;D:/workspace/skoll-apps")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load with dev allowlist: %v", err)
	}
	if cfg.Dev.PluginsRoot != "plugins" {
		t.Fatalf("unexpected default dev root: %s", cfg.Dev.PluginsRoot)
	}
	if len(cfg.Dev.PluginsRoots) != 2 {
		t.Fatalf("unexpected allowlist size: %d", len(cfg.Dev.PluginsRoots))
	}
	if cfg.Dev.PluginsRoots[1] != "D:/workspace/skoll-apps" {
		t.Fatalf("unexpected secondary allowlist root: %s", cfg.Dev.PluginsRoots[1])
	}
}

func TestLoadFallsBackToDefaultDevRootWhenEmpty(t *testing.T) {
	t.Setenv("SKOLL_DEV_PLUGINS_ROOT", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load with empty dev root: %v", err)
	}
	if cfg.Dev.PluginsRoot != "plugins" {
		t.Fatalf("unexpected fallback dev root: %s", cfg.Dev.PluginsRoot)
	}
	if len(cfg.Dev.PluginsRoots) != 1 || cfg.Dev.PluginsRoots[0] != "plugins" {
		t.Fatalf("unexpected fallback allowlist: %+v", cfg.Dev.PluginsRoots)
	}
}

func TestValidate(t *testing.T) {
	cfg := AppConfig{
		Server:   ServerConfig{Address: ":8080", ShutdownTimeout: 1},
		Store:    StoreConfig{Mode: "memory"},
		Cache:    CacheConfig{Mode: "memory", LocalSize: 256},
		Event:    EventConfig{Mode: "memory"},
		Plugin:   PluginConfig{Quota: validPluginQuota()},
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
		Cache:    CacheConfig{Mode: "memory", LocalSize: 256},
		Event:    EventConfig{Mode: "redis"},
		Plugin:   PluginConfig{Quota: validPluginQuota()},
		Security: SecurityConfig{JWTSecret: "s"},
	}
	if err := Validate(cfg); err == nil {
		t.Fatalf("expected redis event mode validation error")
	}
}

func TestValidateRejectsMissingPluginQuota(t *testing.T) {
	cfg := AppConfig{
		Server: ServerConfig{Address: ":8080", ShutdownTimeout: 1},
		Store:  StoreConfig{Mode: "memory"}, Cache: CacheConfig{Mode: "memory", LocalSize: 256},
		Event: EventConfig{Mode: "memory"}, Security: SecurityConfig{JWTSecret: "s"},
	}
	if err := Validate(cfg); err == nil {
		t.Fatal("missing plugin quota must fail validation")
	}
}

func validPluginQuota() PluginQuotaConfig {
	return PluginQuotaConfig{
		RequestRate: 1, RequestBurst: 1, RequestConcurrency: 1,
		HostCallRate: 1, HostCallBurst: 1, HostCallConcurrency: 1,
		QueryRate: 1, QueryBurst: 1, QueryConcurrency: 1,
		MutationRate: 1, MutationBurst: 1, MutationConcurrency: 1,
		EventRate: 1, EventBurst: 1, EventConcurrency: 1,
		JobRate: 1, JobBurst: 1, JobConcurrency: 1,
		ExportRate: 1, ExportBurst: 1, ExportConcurrency: 1,
		StorageRate: 1, StorageBurst: 1, StorageConcurrency: 1,
		ProcessRate: 1, ProcessBurst: 1, ProcessConcurrency: 1,
		MaxPendingEvents: 1, MaxPendingJobs: 1, MaxFileBytes: 1, MaxStorageBytes: 1,
		MaxRequestBytes: 1, MaxResponseBytes: 1, RequestTimeout: time.Second,
		ProcessMemoryBytes: 1, ProcessMaxProcs: 1,
	}
}
