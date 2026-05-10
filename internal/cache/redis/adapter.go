package redis

import (
	"context"
	"fmt"
	"strings"
	"time"

	redisv9 "github.com/redis/go-redis/v9"
	"github.com/tinboxw/skoll/internal/cache/local"
)

type Adapter struct {
	addr     string
	client   *redisv9.Client
	fallback *local.LRUCache
}

func NewAdapter(addr string) (*Adapter, error) {
	if strings.TrimSpace(addr) == "" {
		return nil, fmt.Errorf("redis addr is required")
	}
	cleanAddr := strings.TrimSpace(addr)
	return &Adapter{
		addr:     cleanAddr,
		client:   redisv9.NewClient(&redisv9.Options{Addr: cleanAddr}),
		fallback: local.NewLRUCache(8192),
	}, nil
}

func (a *Adapter) Address() string { return a.addr }

func (a *Adapter) Set(key string, value []byte, ttl time.Duration) {
	if a == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	err := a.client.Set(ctx, key, value, ttl).Err()
	cancel()
	if err != nil {
		a.fallback.Set(key, value, ttl)
		return
	}
	a.fallback.Set(key, value, ttl)
}

func (a *Adapter) Get(key string) ([]byte, bool) {
	if a == nil {
		return nil, false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	value, err := a.client.Get(ctx, key).Bytes()
	cancel()
	if err == nil {
		return value, true
	}
	return a.fallback.Get(key)
}

func (a *Adapter) Delete(key string) {
	if a == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	_ = a.client.Del(ctx, key).Err()
	cancel()
	a.fallback.Delete(key)
}
