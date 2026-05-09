package memcached

import "time"

type PageCache struct {
	adapter *Adapter
}

func NewPageCache(adapter *Adapter) *PageCache {
	return &PageCache{adapter: adapter}
}

func (c *PageCache) Set(path string, payload []byte, ttl time.Duration) {
	c.adapter.Set("page:"+path, payload, ttl)
}

func (c *PageCache) Get(path string) ([]byte, bool) {
	return c.adapter.Get("page:" + path)
}

func (c *PageCache) Delete(path string) {
	c.adapter.Delete("page:" + path)
}
