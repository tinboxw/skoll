package cache

import (
	"fmt"
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/cache/local"
	"github.com/tinboxw/skoll/internal/cache/memcached"
	"github.com/tinboxw/skoll/internal/cache/redis"
)

type Mode string

const (
	ModeLocal     Mode = "local"
	ModeRedis     Mode = "redis"
	ModeMemcached Mode = "memcached"
)

type Options struct {
	Mode          Mode
	RedisAddr     string
	MemcachedAddr string
	LocalSize     int
}

type BytesCache interface {
	Set(string, []byte, time.Duration)
	Get(string) ([]byte, bool)
	Delete(string)
}

type Bundle struct {
	Session    BytesCache
	User       BytesCache
	Permission BytesCache
	Page       BytesCache
	Config     BytesCache
}

type localNamespaceCache struct {
	cache  *local.LRUCache
	prefix string
}

func (c *localNamespaceCache) Set(key string, value []byte, ttl time.Duration) {
	c.cache.Set(c.prefix+key, value, ttl)
}

func (c *localNamespaceCache) Get(key string) ([]byte, bool) {
	return c.cache.Get(c.prefix + key)
}

func (c *localNamespaceCache) Delete(key string) {
	c.cache.Delete(c.prefix + key)
}

func NewBundle(opts Options) (*Bundle, error) {
	switch Mode(strings.ToLower(strings.TrimSpace(string(opts.Mode)))) {
	case ModeLocal:
		lru := local.NewLRUCache(opts.LocalSize)
		return &Bundle{
			Session:    &localNamespaceCache{cache: lru, prefix: "session:"},
			User:       &localNamespaceCache{cache: lru, prefix: "user:"},
			Permission: &localNamespaceCache{cache: lru, prefix: "perm:"},
			Page:       &localNamespaceCache{cache: lru, prefix: "page:"},
			Config:     &localNamespaceCache{cache: lru, prefix: "config:"},
		}, nil
	case ModeRedis:
		a, err := redis.NewAdapter(opts.RedisAddr)
		if err != nil {
			return nil, err
		}
		return &Bundle{
			Session:    redis.NewSessionCache(a),
			User:       redis.NewUserCache(a),
			Permission: redis.NewPermissionCache(a),
			Page:       &localNamespaceCache{cache: local.NewLRUCache(2048), prefix: "page:"},
			Config:     &localNamespaceCache{cache: local.NewLRUCache(2048), prefix: "config:"},
		}, nil
	case ModeMemcached:
		a, err := memcached.NewAdapter(opts.MemcachedAddr)
		if err != nil {
			return nil, err
		}
		return &Bundle{
			Session:    &localNamespaceCache{cache: local.NewLRUCache(2048), prefix: "session:"},
			User:       &localNamespaceCache{cache: local.NewLRUCache(2048), prefix: "user:"},
			Permission: &localNamespaceCache{cache: local.NewLRUCache(2048), prefix: "perm:"},
			Page:       memcached.NewPageCache(a),
			Config:     memcached.NewConfigCache(a),
		}, nil
	default:
		return nil, fmt.Errorf("unsupported cache mode: %q", opts.Mode)
	}
}
