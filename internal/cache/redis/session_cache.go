package redis

import "time"

type SessionCache struct {
	adapter *Adapter
}

func NewSessionCache(adapter *Adapter) *SessionCache {
	return &SessionCache{adapter: adapter}
}

func (c *SessionCache) Set(sessionID string, payload []byte, ttl time.Duration) {
	c.adapter.Set("session:"+sessionID, payload, ttl)
}

func (c *SessionCache) Get(sessionID string) ([]byte, bool) {
	return c.adapter.Get("session:" + sessionID)
}

func (c *SessionCache) Delete(sessionID string) {
	c.adapter.Delete("session:" + sessionID)
}
