package adminauth

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

type fakeRedisNonceClient struct {
	mu       sync.Mutex
	keys     map[string]time.Time
	pingErr  error
	setNXErr error
}

func (c *fakeRedisNonceClient) SetNX(_ context.Context, key string, _ interface{}, expiration time.Duration) *redis.BoolCmd {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.setNXErr != nil {
		return redis.NewBoolResult(false, c.setNXErr)
	}
	now := time.Now()
	if exp, ok := c.keys[key]; ok && exp.After(now) {
		return redis.NewBoolResult(false, nil)
	}
	c.keys[key] = now.Add(expiration)
	return redis.NewBoolResult(true, nil)
}

func (c *fakeRedisNonceClient) Ping(_ context.Context) *redis.StatusCmd {
	if c.pingErr != nil {
		return redis.NewStatusResult("", c.pingErr)
	}
	return redis.NewStatusResult("PONG", nil)
}

func (c *fakeRedisNonceClient) Close() error {
	return nil
}

func TestRedisNonceStoreUseOnce(t *testing.T) {
	client := &fakeRedisNonceClient{keys: make(map[string]time.Time)}
	store := &redisNonceStore{client: client, keyPrefix: "test:", opTimeout: 50 * time.Millisecond}

	if !store.UseOnce("n1", time.Now(), time.Minute) {
		t.Fatalf("expected first nonce use to pass")
	}
	if store.UseOnce("n1", time.Now(), time.Minute) {
		t.Fatalf("expected replay nonce use to fail")
	}
	if store.UseOnce("", time.Now(), time.Minute) {
		t.Fatalf("expected empty nonce to fail")
	}
}

func TestRedisNonceStoreUseOnceError(t *testing.T) {
	client := &fakeRedisNonceClient{keys: make(map[string]time.Time), setNXErr: errors.New("boom")}
	store := &redisNonceStore{client: client, keyPrefix: "test:", opTimeout: 50 * time.Millisecond}

	if store.UseOnce("n1", time.Now(), time.Minute) {
		t.Fatalf("expected nonce use to fail when redis returns error")
	}
}

func TestNormalizeRedisNonceKeyPrefix(t *testing.T) {
	if got := normalizeRedisNonceKeyPrefix(""); got != defaultRedisNonceKeyPrefix {
		t.Fatalf("expected default key prefix, got %q", got)
	}
	if got := normalizeRedisNonceKeyPrefix("custom:"); got != "custom:" {
		t.Fatalf("expected custom key prefix, got %q", got)
	}
}
