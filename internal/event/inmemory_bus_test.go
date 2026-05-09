package event

import (
	"context"
	"sync"
	"testing"
	"time"
)

type testEvent struct{ name string }

func (e testEvent) Name() string { return e.name }

func TestInMemoryBusPublishSubscribe(t *testing.T) {
	bus := NewInMemoryBus()
	var wg sync.WaitGroup
	wg.Add(1)

	unsub := bus.Subscribe("evt.test", func(_ context.Context, evt Event) error {
		if evt.Name() != "evt.test" {
			t.Fatalf("unexpected event: %s", evt.Name())
		}
		wg.Done()
		return nil
	})
	defer unsub()

	if err := bus.Publish(context.Background(), testEvent{name: "evt.test"}); err != nil {
		t.Fatalf("Publish error: %v", err)
	}

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("event handler timeout")
	}
}

func TestInMemoryBusUnsubscribe(t *testing.T) {
	bus := NewInMemoryBus()
	called := false
	unsub := bus.Subscribe("evt.once", func(_ context.Context, _ Event) error {
		called = true
		return nil
	})
	unsub()

	_ = bus.Publish(context.Background(), testEvent{name: "evt.once"})
	time.Sleep(50 * time.Millisecond)
	if called {
		t.Fatalf("handler should not be called after unsubscribe")
	}
}
