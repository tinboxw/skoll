package bootstrap

import (
	"context"
	"testing"
	"time"
)

func TestValidateRuntimeConfig(t *testing.T) {
	cfg := RuntimeConfig{}
	if err := ValidateRuntimeConfig(cfg); err == nil {
		t.Fatalf("expected validation error")
	}
}

func TestRunnerStopsOnContextCancel(t *testing.T) {
	t.Setenv("SKOLL_SERVER_ADDRESS", ":18080")
	t.Setenv("SKOLL_SHUTDOWN_TIMEOUT", "2s")
	t.Setenv("SKOLL_STORE_MODE", "memory")
	runner, err := NewRunnerFromEnv()
	if err != nil {
		t.Fatalf("new runner: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- runner.Run(ctx) }()

	time.Sleep(150 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("runner returned error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("runner did not exit after cancel")
	}
}
