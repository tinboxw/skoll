package logging

import (
	"context"
	"testing"
)

func TestNewLoggerMethodsDoNotPanic(t *testing.T) {
	logger := New("debug")
	if logger == nil {
		t.Fatalf("expected non-nil logger")
	}

	logger.Debug("debug_message", "k1", "v1")
	logger.Info("info_message", "k2", 2)
	logger.Warn("warn_message", "k3", true)
	logger.Error("error_message", "k4")
}

func TestContextLoggerRoundTrip(t *testing.T) {
	fallback := Discard()
	ctx := context.Background()
	if got := FromContext(ctx, fallback); got != fallback {
		t.Fatalf("expected fallback logger")
	}

	bound := NewZapCompatibleLogger("info")
	ctx = ContextWithLogger(ctx, bound)
	if got := FromContext(ctx, fallback); got != bound {
		t.Fatalf("expected context logger")
	}
}

func TestKVBuildsZapFields(t *testing.T) {
	fields := KV("a", 1, "b", "x", 123, "ignored")
	if len(fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(fields))
	}
}
