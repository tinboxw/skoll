package memcached

import "time"

type ConfigCache struct {
	adapter *Adapter
}

func NewConfigCache(adapter *Adapter) *ConfigCache {
	return &ConfigCache{adapter: adapter}
}

func (c *ConfigCache) Set(key string, payload []byte, ttl time.Duration) {
	c.adapter.Set("config:"+key, payload, ttl)
}

func (c *ConfigCache) Get(key string) ([]byte, bool) {
	return c.adapter.Get("config:" + key)
}

func (c *ConfigCache) Delete(key string) {
	c.adapter.Delete("config:" + key)
}
