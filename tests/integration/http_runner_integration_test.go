package integration

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/bootstrap"
	"github.com/tinboxw/skoll/pkg/config"
)

func TestRunnerHealthAndAuth(t *testing.T) {
	cfg := bootstrap.RuntimeConfig{
		AppConfig: config.AppConfig{
			Server: config.ServerConfig{Address: "127.0.0.1:18091", ShutdownTimeout: 2 * time.Second},
			Store:  config.StoreConfig{Mode: "memory"},
			Security: config.SecurityConfig{
				JWTSecret: "integration-secret",
			},
			Log: config.LogConfig{Level: "error"},
		},
		AuthPolicy: bootstrap.AuthPolicy{Enabled: true, SkipPaths: map[string]struct{}{"/health": {}, "/ready": {}}},
	}

	runner, err := bootstrap.NewRunner(cfg)
	if err != nil {
		t.Fatalf("bootstrap.NewRunner error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- runner.Run(ctx) }()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-done:
			if err != nil {
				t.Fatalf("runner shutdown error: %v", err)
			}
		case <-time.After(3 * time.Second):
			t.Fatalf("runner shutdown timeout")
		}
	})

	waitForUp(t, "http://127.0.0.1:18091/health")

	client := &http.Client{Timeout: 2 * time.Second}
	resp1, err := client.Get("http://127.0.0.1:18091/health")
	if err != nil {
		t.Fatalf("GET /health error: %v", err)
	}
	defer resp1.Body.Close()
	if resp1.StatusCode != http.StatusOK {
		t.Fatalf("/health status=%d", resp1.StatusCode)
	}

	resp2, err := client.Get("http://127.0.0.1:18091/v1/users")
	if err != nil {
		t.Fatalf("GET /v1/users error: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusUnauthorized {
		t.Fatalf("/v1/users expected 401, got %d", resp2.StatusCode)
	}
}

func waitForUp(t *testing.T, url string) {
	t.Helper()
	client := &http.Client{Timeout: 300 * time.Millisecond}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			return
		}
		time.Sleep(30 * time.Millisecond)
	}
	t.Fatalf("server not up: %s", url)
}
