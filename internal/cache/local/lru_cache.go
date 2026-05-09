package local

import (
	"sync"
	"time"
)

type entry struct {
	value     []byte
	expiresAt time.Time
}

type LRUCache struct {
	mu       sync.RWMutex
	maxItems int
	items    map[string]entry
	order    []string
}

func NewLRUCache(maxItems int) *LRUCache {
	if maxItems <= 0 {
		maxItems = 1024
	}
	return &LRUCache{maxItems: maxItems, items: map[string]entry{}, order: make([]string, 0, maxItems)}
}

func (c *LRUCache) Set(key string, value []byte, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	expiresAt := time.Time{}
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl)
	}

	if _, exists := c.items[key]; !exists {
		c.order = append(c.order, key)
	}
	c.items[key] = entry{value: append([]byte(nil), value...), expiresAt: expiresAt}
	c.evictIfNeeded()
}

func (c *LRUCache) Get(key string) ([]byte, bool) {
	c.mu.RLock()
	e, ok := c.items[key]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if !e.expiresAt.IsZero() && time.Now().After(e.expiresAt) {
		c.Delete(key)
		return nil, false
	}
	return append([]byte(nil), e.value...), true
}

func (c *LRUCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
	for i, k := range c.order {
		if k == key {
			c.order = append(c.order[:i], c.order[i+1:]...)
			break
		}
	}
}

func (c *LRUCache) evictIfNeeded() {
	for len(c.items) > c.maxItems && len(c.order) > 0 {
		victim := c.order[0]
		c.order = c.order[1:]
		delete(c.items, victim)
	}
}
