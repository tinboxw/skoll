package bootstrap

import (
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/tinboxw/skoll/internal/event"
	"github.com/tinboxw/skoll/pkg/config"
)

func TestBuildEventBusMemory(t *testing.T) {
	bus, err := buildEventBus(config.EventConfig{Mode: "memory"})
	if err != nil {
		t.Fatalf("buildEventBus memory error: %v", err)
	}
	if bus == nil {
		t.Fatalf("expected memory bus")
	}
}

func TestBuildEventBusRedis(t *testing.T) {
	srv, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}
	t.Cleanup(srv.Close)

	bus, err := buildEventBus(config.EventConfig{Mode: "redis", RedisAddr: srv.Addr(), ChannelPrefix: "skoll.events.test"})
	if err != nil {
		t.Fatalf("buildEventBus redis error: %v", err)
	}
	redisBus, ok := bus.(*event.RedisBus)
	if !ok {
		t.Fatalf("expected redis bus type, got %T", bus)
	}
	t.Cleanup(func() {
		_ = redisBus.Close()
	})
}

func TestBuildEventBusUnsupportedMode(t *testing.T) {
	if _, err := buildEventBus(config.EventConfig{Mode: "kafka"}); err == nil {
		t.Fatalf("expected unsupported mode error")
	}
}
