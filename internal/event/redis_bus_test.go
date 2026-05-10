package event

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
)

type distributedTestEvent struct {
	EventName string `json:"event_name"`
	Value     string `json:"value"`
}

func (e distributedTestEvent) Name() string { return e.EventName }

func TestRedisBusPublishSubscribeAcrossInstances(t *testing.T) {
	srv, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}
	t.Cleanup(srv.Close)

	busA, err := NewRedisBus(srv.Addr(), "skoll.events.test")
	if err != nil {
		t.Fatalf("new redis bus A: %v", err)
	}
	t.Cleanup(func() { _ = busA.Close() })

	busB, err := NewRedisBus(srv.Addr(), "skoll.events.test")
	if err != nil {
		t.Fatalf("new redis bus B: %v", err)
	}
	t.Cleanup(func() { _ = busB.Close() })

	received := make(chan RemoteEvent, 1)
	unsub := busB.Subscribe("evt.distributed", func(_ context.Context, evt Event) error {
		remote, ok := evt.(RemoteEvent)
		if !ok {
			t.Errorf("expected RemoteEvent, got %T", evt)
			return nil
		}
		received <- remote
		return nil
	})
	t.Cleanup(unsub)

	if err := busA.Publish(context.Background(), distributedTestEvent{EventName: "evt.distributed", Value: "hello"}); err != nil {
		t.Fatalf("publish: %v", err)
	}

	select {
	case remote := <-received:
		if remote.Name() != "evt.distributed" {
			t.Fatalf("unexpected event name: %s", remote.Name())
		}
		var payload map[string]any
		if err := json.Unmarshal(remote.Payload, &payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		if payload["value"] != "hello" {
			t.Fatalf("unexpected payload value: %v", payload["value"])
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting distributed event")
	}
}

func TestRedisBusLocalHandlerNotDuplicatedByLoopback(t *testing.T) {
	srv, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}
	t.Cleanup(srv.Close)

	bus, err := NewRedisBus(srv.Addr(), "skoll.events.test")
	if err != nil {
		t.Fatalf("new redis bus: %v", err)
	}
	t.Cleanup(func() { _ = bus.Close() })

	var calls atomic.Int32
	done := make(chan struct{}, 1)
	unsub := bus.Subscribe("evt.once", func(_ context.Context, _ Event) error {
		if calls.Add(1) == 1 {
			done <- struct{}{}
		}
		return nil
	})
	t.Cleanup(unsub)

	if err := bus.Publish(context.Background(), distributedTestEvent{EventName: "evt.once", Value: "v"}); err != nil {
		t.Fatalf("publish: %v", err)
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting local handler")
	}
	time.Sleep(200 * time.Millisecond)
	if got := calls.Load(); got != 1 {
		t.Fatalf("expected handler called once, got %d", got)
	}
}

func TestRedisBusUnsubscribe(t *testing.T) {
	srv, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}
	t.Cleanup(srv.Close)

	busA, err := NewRedisBus(srv.Addr(), "skoll.events.test")
	if err != nil {
		t.Fatalf("new redis bus A: %v", err)
	}
	t.Cleanup(func() { _ = busA.Close() })

	busB, err := NewRedisBus(srv.Addr(), "skoll.events.test")
	if err != nil {
		t.Fatalf("new redis bus B: %v", err)
	}
	t.Cleanup(func() { _ = busB.Close() })

	called := make(chan struct{}, 1)
	unsub := busB.Subscribe("evt.unsubscribe", func(_ context.Context, _ Event) error {
		called <- struct{}{}
		return nil
	})
	unsub()

	if err := busA.Publish(context.Background(), distributedTestEvent{EventName: "evt.unsubscribe", Value: "x"}); err != nil {
		t.Fatalf("publish: %v", err)
	}

	select {
	case <-called:
		t.Fatalf("handler should not be called after unsubscribe")
	case <-time.After(300 * time.Millisecond):
	}
}
