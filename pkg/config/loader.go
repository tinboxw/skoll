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
	Plugin   PluginConfig
	Security SecurityConfig
	Log      LogConfig
	Dev      DevConfig
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

type PluginConfig struct {
	Quota PluginQuotaConfig
}

type PluginQuotaConfig struct {
	RequestRate         float64
	RequestBurst        int
	RequestConcurrency  int
	HostCallRate        float64
	HostCallBurst       int
	HostCallConcurrency int
	QueryRate           float64
	QueryBurst          int
	QueryConcurrency    int
	MutationRate        float64
	MutationBurst       int
	MutationConcurrency int
	EventRate           float64
	EventBurst          int
	EventConcurrency    int
	JobRate             float64
	JobBurst            int
	JobConcurrency      int
	ExportRate          float64
	ExportBurst         int
	ExportConcurrency   int
	StorageRate         float64
	StorageBurst        int
	StorageConcurrency  int
	ProcessRate         float64
	ProcessBurst        int
	ProcessConcurrency  int
	MaxPendingEvents    int
	MaxPendingJobs      int
	MaxFileBytes        int64
	MaxStorageBytes     int64
	MaxRequestBytes     int64
	MaxResponseBytes    int64
	RequestTimeout      time.Duration
	ProcessMemoryBytes  int64
	ProcessMaxProcs     int
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

type DevConfig struct {
	PortalEnabled bool
	PluginsRoot   string
	PluginsRoots  []string
}

var defaultConfigCandidates = []string{
	"skoll.yaml",
	"skoll.yml",
	"config/skoll.yaml",
	"config/skoll.yml",
	"configs/skoll.yaml",
	"configs/skoll.yml",
}

const DefaultAPIBasePrefix = "/skoll"

// Load loads configuration with precedence: environment variables > config file > defaults.
func Load() (AppConfig, error) {
	v := viper.New()
	v.SetEnvPrefix("SKOLL")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	_ = v.BindEnv("api.base_prefix", "SKOLL_API_BASE_PREFIX")

	v.SetDefault("server.address", ":8080")
	v.SetDefault("api.base_prefix", DefaultAPIBasePrefix)
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
	v.SetDefault("plugin.quota.request_rate", 50)
	v.SetDefault("plugin.quota.request_burst", 100)
	v.SetDefault("plugin.quota.request_concurrency", 32)
	v.SetDefault("plugin.quota.host_call_rate", 100)
	v.SetDefault("plugin.quota.host_call_burst", 200)
	v.SetDefault("plugin.quota.host_call_concurrency", 64)
	v.SetDefault("plugin.quota.query_rate", 40)
	v.SetDefault("plugin.quota.query_burst", 80)
	v.SetDefault("plugin.quota.query_concurrency", 16)
	v.SetDefault("plugin.quota.mutation_rate", 20)
	v.SetDefault("plugin.quota.mutation_burst", 40)
	v.SetDefault("plugin.quota.mutation_concurrency", 8)
	v.SetDefault("plugin.quota.event_rate", 20)
	v.SetDefault("plugin.quota.event_burst", 40)
	v.SetDefault("plugin.quota.event_concurrency", 8)
	v.SetDefault("plugin.quota.job_rate", 10)
	v.SetDefault("plugin.quota.job_burst", 20)
	v.SetDefault("plugin.quota.job_concurrency", 4)
	v.SetDefault("plugin.quota.export_rate", 2)
	v.SetDefault("plugin.quota.export_burst", 4)
	v.SetDefault("plugin.quota.export_concurrency", 2)
	v.SetDefault("plugin.quota.storage_rate", 10)
	v.SetDefault("plugin.quota.storage_burst", 20)
	v.SetDefault("plugin.quota.storage_concurrency", 4)
	v.SetDefault("plugin.quota.process_rate", 1)
	v.SetDefault("plugin.quota.process_burst", 2)
	v.SetDefault("plugin.quota.process_concurrency", 1)
	v.SetDefault("plugin.quota.max_pending_events", 100)
	v.SetDefault("plugin.quota.max_pending_jobs", 100)
	v.SetDefault("plugin.quota.max_file_bytes", 16<<20)
	v.SetDefault("plugin.quota.max_storage_bytes", 512<<20)
	v.SetDefault("plugin.quota.max_request_bytes", 8<<20)
	v.SetDefault("plugin.quota.max_response_bytes", 16<<20)
	v.SetDefault("plugin.quota.request_timeout", "30s")
	v.SetDefault("plugin.quota.process_memory_bytes", 512<<20)
	v.SetDefault("plugin.quota.process_max_procs", 2)
	v.SetDefault("security.jwt_secret", "dev-secret-change-me")
	v.SetDefault("log.level", "info")
	v.SetDefault("log.dir", "log")
	v.SetDefault("log.file", "")
	v.SetDefault("log.plugin_per_file", false)
	v.SetDefault("dev.portal_enabled", false)
	v.SetDefault("dev.plugins_root", "plugins")
	v.SetDefault("server.port", "")

	if err := loadConfigFile(v); err != nil {
		return AppConfig{}, err
	}

	shutdownRaw := strings.TrimSpace(v.GetString("server.shutdown_timeout"))
	shutdownTimeout, err := time.ParseDuration(shutdownRaw)
	if err != nil {
		return AppConfig{}, fmt.Errorf("invalid SKOLL_SERVER_SHUTDOWN_TIMEOUT: %w", err)
	}
	pluginRequestTimeout, err := time.ParseDuration(strings.TrimSpace(v.GetString("plugin.quota.request_timeout")))
	if err != nil {
		return AppConfig{}, fmt.Errorf("invalid SKOLL_PLUGIN_QUOTA_REQUEST_TIMEOUT: %w", err)
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

	devRoots := parseDevPluginsRoots(v.GetString("dev.plugins_root"))
	devRoot := "plugins"
	if len(devRoots) > 0 {
		devRoot = devRoots[0]
	}

	return AppConfig{
		Server: ServerConfig{
			Address:         address,
			APIPrefix:       NormalizeAPIPrefix(resolveAPIPrefix(v)),
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
		Plugin: PluginConfig{Quota: PluginQuotaConfig{
			RequestRate: v.GetFloat64("plugin.quota.request_rate"), RequestBurst: v.GetInt("plugin.quota.request_burst"), RequestConcurrency: v.GetInt("plugin.quota.request_concurrency"),
			HostCallRate: v.GetFloat64("plugin.quota.host_call_rate"), HostCallBurst: v.GetInt("plugin.quota.host_call_burst"), HostCallConcurrency: v.GetInt("plugin.quota.host_call_concurrency"),
			QueryRate: v.GetFloat64("plugin.quota.query_rate"), QueryBurst: v.GetInt("plugin.quota.query_burst"), QueryConcurrency: v.GetInt("plugin.quota.query_concurrency"),
			MutationRate: v.GetFloat64("plugin.quota.mutation_rate"), MutationBurst: v.GetInt("plugin.quota.mutation_burst"), MutationConcurrency: v.GetInt("plugin.quota.mutation_concurrency"),
			EventRate: v.GetFloat64("plugin.quota.event_rate"), EventBurst: v.GetInt("plugin.quota.event_burst"), EventConcurrency: v.GetInt("plugin.quota.event_concurrency"),
			JobRate: v.GetFloat64("plugin.quota.job_rate"), JobBurst: v.GetInt("plugin.quota.job_burst"), JobConcurrency: v.GetInt("plugin.quota.job_concurrency"),
			ExportRate: v.GetFloat64("plugin.quota.export_rate"), ExportBurst: v.GetInt("plugin.quota.export_burst"), ExportConcurrency: v.GetInt("plugin.quota.export_concurrency"),
			StorageRate: v.GetFloat64("plugin.quota.storage_rate"), StorageBurst: v.GetInt("plugin.quota.storage_burst"), StorageConcurrency: v.GetInt("plugin.quota.storage_concurrency"),
			ProcessRate: v.GetFloat64("plugin.quota.process_rate"), ProcessBurst: v.GetInt("plugin.quota.process_burst"), ProcessConcurrency: v.GetInt("plugin.quota.process_concurrency"),
			MaxPendingEvents: v.GetInt("plugin.quota.max_pending_events"), MaxPendingJobs: v.GetInt("plugin.quota.max_pending_jobs"),
			MaxFileBytes: v.GetInt64("plugin.quota.max_file_bytes"), MaxStorageBytes: v.GetInt64("plugin.quota.max_storage_bytes"),
			MaxRequestBytes: v.GetInt64("plugin.quota.max_request_bytes"), MaxResponseBytes: v.GetInt64("plugin.quota.max_response_bytes"),
			RequestTimeout: pluginRequestTimeout, ProcessMemoryBytes: v.GetInt64("plugin.quota.process_memory_bytes"), ProcessMaxProcs: v.GetInt("plugin.quota.process_max_procs"),
		}},
		Security: SecurityConfig{
			JWTSecret: strings.TrimSpace(v.GetString("security.jwt_secret")),
		},
		Log: LogConfig{
			Level:         strings.ToLower(strings.TrimSpace(v.GetString("log.level"))),
			Dir:           strings.TrimSpace(v.GetString("log.dir")),
			File:          strings.TrimSpace(v.GetString("log.file")),
			PluginPerFile: v.GetBool("log.plugin_per_file"),
		},
		Dev: DevConfig{
			PortalEnabled: v.GetBool("dev.portal_enabled"),
			PluginsRoot:   devRoot,
			PluginsRoots:  devRoots,
		},
	}, nil
}

func parseDevPluginsRoots(raw string) []string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return []string{"plugins"}
	}
	parts := strings.FieldsFunc(trimmed, func(r rune) bool {
		return r == ';' || r == ','
	})
	roots := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		v := strings.TrimSpace(part)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		roots = append(roots, v)
	}
	if len(roots) == 0 {
		return []string{"plugins"}
	}
	return roots
}

func resolveAPIPrefix(v *viper.Viper) string {
	if v == nil {
		return DefaultAPIBasePrefix
	}
	vv := strings.TrimSpace(v.GetString("api.base_prefix"))
	if vv != "" {
		return vv
	}
	return DefaultAPIBasePrefix
}

func NormalizeAPIPrefix(raw string) string {
	v := strings.TrimSpace(raw)
	if v == "" {
		return DefaultAPIBasePrefix
	}
	if !strings.HasPrefix(v, "/") {
		v = "/" + v
	}
	v = strings.TrimRight(v, "/")
	if v == "" {
		return DefaultAPIBasePrefix
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
