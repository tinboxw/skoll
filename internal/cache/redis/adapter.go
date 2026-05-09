package redis

import (
	"fmt"
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/cache/local"
)

type Adapter struct {
	addr  string
	cache *local.LRUCache
}

func NewAdapter(addr string) (*Adapter, error) {
	if strings.TrimSpace(addr) == "" {
		return nil, fmt.Errorf("redis addr is required")
	}
	return &Adapter{addr: strings.TrimSpace(addr), cache: local.NewLRUCache(8192)}, nil
}

func (a *Adapter) Address() string { return a.addr }

func (a *Adapter) Set(key string, value []byte, ttl time.Duration) {
	a.cache.Set(key, value, ttl)
}

func (a *Adapter) Get(key string) ([]byte, bool) {
	return a.cache.Get(key)
}

func (a *Adapter) Delete(key string) {
	a.cache.Delete(key)
}
