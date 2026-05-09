package event

import (
	"context"
	"sync"
)

type InMemoryBus struct {
	mu       sync.RWMutex
	handlers map[string]map[int64]Handler
	nextID   int64
}

func NewInMemoryBus() *InMemoryBus {
	return &InMemoryBus{handlers: map[string]map[int64]Handler{}}
}

func (b *InMemoryBus) Publish(ctx context.Context, evt Event) error {
	if evt == nil {
		return nil
	}
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
	return nil
}

func (b *InMemoryBus) Subscribe(eventName string, handler Handler) (unsubscribe func()) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.handlers[eventName] == nil {
		b.handlers[eventName] = map[int64]Handler{}
	}
	b.nextID++
	id := b.nextID
	b.handlers[eventName][id] = handler

	return func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		if b.handlers[eventName] != nil {
			delete(b.handlers[eventName], id)
		}
	}
}
