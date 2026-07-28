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
	if strings.TrimSpace(cfg.Server.APIPrefix) != "" && !strings.HasPrefix(strings.TrimSpace(cfg.Server.APIPrefix), "/") {
		return errors.New("server api prefix must start with /")
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
	if err := validatePluginQuota(cfg.Plugin.Quota); err != nil {
		return err
	}
	if strings.TrimSpace(cfg.Security.JWTSecret) == "" {
		return errors.New("jwt secret is required")
	}
	if cfg.Dev.PortalEnabled {
		if strings.TrimSpace(cfg.Dev.PluginsRoot) == "" {
			return errors.New("dev plugins root is required when dev portal is enabled")
		}
		if len(cfg.Dev.PluginsRoots) == 0 {
			return errors.New("at least one dev plugins root is required when dev portal is enabled")
		}
	}
	return nil
}

func validatePluginQuota(quota PluginQuotaConfig) error {
	rates := []struct {
		name        string
		rate        float64
		burst       int
		concurrency int
	}{
		{"request", quota.RequestRate, quota.RequestBurst, quota.RequestConcurrency},
		{"host call", quota.HostCallRate, quota.HostCallBurst, quota.HostCallConcurrency},
		{"query", quota.QueryRate, quota.QueryBurst, quota.QueryConcurrency},
		{"mutation", quota.MutationRate, quota.MutationBurst, quota.MutationConcurrency},
		{"event", quota.EventRate, quota.EventBurst, quota.EventConcurrency},
		{"job", quota.JobRate, quota.JobBurst, quota.JobConcurrency},
		{"export", quota.ExportRate, quota.ExportBurst, quota.ExportConcurrency},
		{"storage", quota.StorageRate, quota.StorageBurst, quota.StorageConcurrency},
		{"process", quota.ProcessRate, quota.ProcessBurst, quota.ProcessConcurrency},
	}
	for _, item := range rates {
		if item.rate <= 0 || item.burst <= 0 || item.concurrency <= 0 {
			return fmt.Errorf("plugin %s quota rate, burst, and concurrency must be positive", item.name)
		}
	}
	if quota.MaxPendingEvents < 1 || quota.MaxPendingEvents > 199 {
		return errors.New("plugin pending event quota must be between 1 and 199")
	}
	if quota.MaxPendingJobs < 1 || quota.MaxPendingJobs > 499 {
		return errors.New("plugin pending job quota must be between 1 and 499")
	}
	if quota.MaxFileBytes <= 0 || quota.MaxStorageBytes < quota.MaxFileBytes {
		return errors.New("plugin storage quota requires positive file bytes within total bytes")
	}
	if quota.MaxRequestBytes <= 0 || quota.MaxResponseBytes <= 0 || quota.RequestTimeout <= 0 {
		return errors.New("plugin request byte limits and timeout must be positive")
	}
	if quota.ProcessMemoryBytes <= 0 || quota.ProcessMaxProcs <= 0 {
		return errors.New("plugin process memory and processor quotas must be positive")
	}
	return nil
}
