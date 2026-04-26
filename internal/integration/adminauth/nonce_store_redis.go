package adminauth

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	defaultRedisNonceKeyPrefix = "skoll:adminauth:nonce:"
	defaultRedisOpTimeout      = 200 * time.Millisecond
)

type redisNonceStoreClient interface {
	SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.BoolCmd
	Ping(ctx context.Context) *redis.StatusCmd
	Close() error
}

type redisNonceStore struct {
	client    redisNonceStoreClient
	keyPrefix string
	opTimeout time.Duration
}

type RedisNonceStoreConfig struct {
	Addr      string
	Password  string
	DB        int
	KeyPrefix string
}

func NewRedisNonceStoreFromConfig(cfg RedisNonceStoreConfig) (ReplayNonceStore, func(context.Context) error, error) {
	if cfg.Addr == "" {
		return nil, nil, fmt.Errorf("redis nonce store requires non-empty addr")
	}

	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), defaultRedisOpTimeout)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, nil, fmt.Errorf("redis nonce store ping failed: %w", err)
	}

	store := &redisNonceStore{
		client:    client,
		keyPrefix: normalizeRedisNonceKeyPrefix(cfg.KeyPrefix),
		opTimeout: defaultRedisOpTimeout,
	}
	closeFn := func(context.Context) error {
		return client.Close()
	}

	return store, closeFn, nil
}

func normalizeRedisNonceKeyPrefix(prefix string) string {
	if prefix == "" {
		return defaultRedisNonceKeyPrefix
	}
	return prefix
}

func (s *redisNonceStore) UseOnce(nonce string, _ time.Time, ttl time.Duration) bool {
	if nonce == "" {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.opTimeout)
	defer cancel()

	ok, err := s.client.SetNX(ctx, s.keyPrefix+nonce, "1", ttl).Result()
	if err != nil {
		return false
	}
	return ok
}
