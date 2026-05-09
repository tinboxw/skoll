package redis

import "time"

type PermissionCache struct {
	adapter *Adapter
}

func NewPermissionCache(adapter *Adapter) *PermissionCache {
	return &PermissionCache{adapter: adapter}
}

func (c *PermissionCache) Set(subject string, payload []byte, ttl time.Duration) {
	c.adapter.Set("perm:"+subject, payload, ttl)
}

func (c *PermissionCache) Get(subject string) ([]byte, bool) {
	return c.adapter.Get("perm:" + subject)
}

func (c *PermissionCache) Delete(subject string) {
	c.adapter.Delete("perm:" + subject)
}
