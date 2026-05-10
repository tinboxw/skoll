package event

import (
	"context"
	crand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	redisv9 "github.com/redis/go-redis/v9"
)

const defaultRedisChannelPrefix = "skoll.events"

type RemoteEvent struct {
	EventName   string
	Payload     json.RawMessage
	PublishedAt time.Time
}

func (e RemoteEvent) Name() string { return e.EventName }

type redisEnvelope struct {
	EventName   string          `json:"event"`
	Source      string          `json:"source"`
	Payload     json.RawMessage `json:"payload"`
	PublishedAt time.Time       `json:"published_at"`
}

type redisSubscription struct {
	cancel func()
	pubsub *redisv9.PubSub
}

type RedisBus struct {
	client        *redisv9.Client
	channelPrefix string
	instanceID    string

	mu            sync.RWMutex
	handlers      map[string]map[int64]Handler
	subscriptions map[string]*redisSubscription
	nextID        int64
}

func NewRedisBus(addr, channelPrefix string) (*RedisBus, error) {
	cleanAddr := strings.TrimSpace(addr)
	if cleanAddr == "" {
		return nil, fmt.Errorf("redis addr is required")
	}

	prefix := strings.TrimSpace(channelPrefix)
	if prefix == "" {
		prefix = defaultRedisChannelPrefix
	}

	client := redisv9.NewClient(&redisv9.Options{Addr: cleanAddr})
	pingCtx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	err := client.Ping(pingCtx).Err()
	cancel()
	if err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	return &RedisBus{
		client:        client,
		channelPrefix: prefix,
		instanceID:    newInstanceID(),
		handlers:      map[string]map[int64]Handler{},
		subscriptions: map[string]*redisSubscription{},
	}, nil
}

func (b *RedisBus) Publish(ctx context.Context, evt Event) error {
	if b == nil || evt == nil {
		return nil
	}

	b.dispatch(ctx, evt)

	payload, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("marshal event payload: %w", err)
	}
	msg, err := json.Marshal(redisEnvelope{
		EventName:   evt.Name(),
		Source:      b.instanceID,
		Payload:     payload,
		PublishedAt: time.Now().UTC(),
	})
	if err != nil {
		return fmt.Errorf("marshal redis envelope: %w", err)
	}

	pubCtx := ctx
	if pubCtx == nil {
		pubCtx = context.Background()
	}
	if _, hasDeadline := pubCtx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		pubCtx, cancel = context.WithTimeout(pubCtx, 500*time.Millisecond)
		defer cancel()
	}

	if err := b.client.Publish(pubCtx, b.channel(evt.Name()), msg).Err(); err != nil {
		return fmt.Errorf("publish to redis: %w", err)
	}
	return nil
}

func (b *RedisBus) Subscribe(eventName string, handler Handler) (unsubscribe func()) {
	if b == nil || handler == nil {
		return func() {}
	}
	eventName = strings.TrimSpace(eventName)
	if eventName == "" {
		return func() {}
	}

	b.mu.Lock()
	if b.handlers[eventName] == nil {
		b.handlers[eventName] = map[int64]Handler{}
	}
	b.nextID++
	id := b.nextID
	b.handlers[eventName][id] = handler
	needSubscribe := b.subscriptions[eventName] == nil
	b.mu.Unlock()

	if needSubscribe {
		b.ensureSubscription(eventName)
	}

	return func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		if b.handlers[eventName] != nil {
			delete(b.handlers[eventName], id)
			if len(b.handlers[eventName]) == 0 {
				delete(b.handlers, eventName)
				if sub := b.subscriptions[eventName]; sub != nil {
					sub.cancel()
					_ = sub.pubsub.Close()
					delete(b.subscriptions, eventName)
				}
			}
		}
	}
}

func (b *RedisBus) Close() error {
	if b == nil {
		return nil
	}

	b.mu.Lock()
	subs := make([]*redisSubscription, 0, len(b.subscriptions))
	for _, sub := range b.subscriptions {
		subs = append(subs, sub)
	}
	b.subscriptions = map[string]*redisSubscription{}
	b.handlers = map[string]map[int64]Handler{}
	b.mu.Unlock()

	for _, sub := range subs {
		sub.cancel()
		_ = sub.pubsub.Close()
	}
	return b.client.Close()
}

func (b *RedisBus) ensureSubscription(eventName string) {
	ctx, cancel := context.WithCancel(context.Background())
	pubsub := b.client.Subscribe(ctx, b.channel(eventName))
	if _, err := pubsub.Receive(ctx); err != nil {
		cancel()
		_ = pubsub.Close()
		return
	}

	b.mu.Lock()
	if b.subscriptions[eventName] != nil {
		b.mu.Unlock()
		cancel()
		_ = pubsub.Close()
		return
	}
	b.subscriptions[eventName] = &redisSubscription{cancel: cancel, pubsub: pubsub}
	b.mu.Unlock()

	go b.consumeSubscription(ctx, pubsub)
}

func (b *RedisBus) consumeSubscription(ctx context.Context, pubsub *redisv9.PubSub) {
	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			if strings.TrimSpace(msg.Payload) == "" {
				continue
			}
			var env redisEnvelope
			if err := json.Unmarshal([]byte(msg.Payload), &env); err != nil {
				continue
			}
			if env.Source == b.instanceID || strings.TrimSpace(env.EventName) == "" {
				continue
			}
			b.dispatch(ctx, RemoteEvent{EventName: env.EventName, Payload: env.Payload, PublishedAt: env.PublishedAt})
		}
	}
}

func (b *RedisBus) dispatch(ctx context.Context, evt Event) {
	b.mu.RLock()
	set := b.handlers[evt.Name()]
	copied := make([]Handler, 0, len(set))
	for _, h := range set {
		copied = append(copied, h)
	}
	b.mu.RUnlock()

	for _, h := range copied {
		h := h
		go func() {
			_ = h(ctx, evt)
		}()
	}
}

func (b *RedisBus) channel(eventName string) string {
	return b.channelPrefix + "." + eventName
}

func newInstanceID() string {
	var token [8]byte
	if _, err := crand.Read(token[:]); err == nil {
		return hex.EncodeToString(token[:])
	}
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
