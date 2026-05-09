package redis

import "time"

type UserCache struct {
	adapter *Adapter
}

func NewUserCache(adapter *Adapter) *UserCache {
	return &UserCache{adapter: adapter}
}

func (c *UserCache) Set(userID string, payload []byte, ttl time.Duration) {
	c.adapter.Set("user:"+userID, payload, ttl)
}

func (c *UserCache) Get(userID string) ([]byte, bool) {
	return c.adapter.Get("user:" + userID)
}

func (c *UserCache) Delete(userID string) {
	c.adapter.Delete("user:" + userID)
}
