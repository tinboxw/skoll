package bootstrap

import (
	"context"
	"testing"
)

func TestRunReturnsAfterCancel(t *testing.T) {
	r, err := NewRunnerFromEnv()
	if err != nil {
		t.Fatalf("NewRunnerFromEnv error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := r.Run(ctx); err != nil {
		t.Fatalf("Run error: %v", err)
	}
}
