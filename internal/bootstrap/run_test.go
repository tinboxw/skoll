package bootstrap

import (
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/tinboxw/skoll/pkg/config"
)

func TestNewRunnerRejectsInvalidConfig(t *testing.T) {
	cfg := RuntimeConfig{
		AppConfig: appConfigForTest("invalid:host:8080"),
		AuthPolicy: AuthPolicy{
			Enabled:   false,
			SkipPaths: map[string]struct{}{},
		},
	}

	if _, err := NewRunner(cfg); err == nil {
		t.Fatalf("expected config validation error")
	}
}

func TestRunReturnsAfterCancel(t *testing.T) {
	cfg := RuntimeConfig{
		AppConfig: appConfigForTest("127.0.0.1:18081"),
		AuthPolicy: AuthPolicy{
			Enabled:   false,
			SkipPaths: map[string]struct{}{"/api/health": {}},
		},
	}

	r, err := NewRunner(cfg)
	if err != nil {
		t.Fatalf("NewRunner error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	if err := r.Run(ctx); err != nil {
		t.Fatalf("Run error: %v", err)
	}
}

func TestHealthEndpointNoAuth(t *testing.T) {
	cfg := RuntimeConfig{
		AppConfig: appConfigForTest("127.0.0.1:18082"),
		AuthPolicy: AuthPolicy{
			Enabled:   true,
			SkipPaths: map[string]struct{}{"/api/health": {}},
		},
	}

	r, err := NewRunner(cfg)
	if err != nil {
		t.Fatalf("NewRunner error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- r.Run(ctx) }()
	waitForServer(t, "http://127.0.0.1:18082/api/health")

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://127.0.0.1:18082/api/health")
	if err != nil {
		cancel()
		t.Fatalf("GET /api/health: %v", err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		cancel()
		t.Fatalf("status=%d body=%s", resp.StatusCode, string(body))
	}

	cancel()
	if err := <-done; err != nil {
		t.Fatalf("Run error: %v", err)
	}
}

func TestProtectedEndpointRequiresAuth(t *testing.T) {
	cfg := RuntimeConfig{
		AppConfig: appConfigForTest("127.0.0.1:18083"),
		AuthPolicy: AuthPolicy{
			Enabled:   true,
			SkipPaths: map[string]struct{}{"/api/health": {}},
		},
	}

	r, err := NewRunner(cfg)
	if err != nil {
		t.Fatalf("NewRunner error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- r.Run(ctx) }()
	waitForServer(t, "http://127.0.0.1:18083/api/health")

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://127.0.0.1:18083/protected")
	if err != nil {
		cancel()
		t.Fatalf("GET /protected: %v", err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })

	if resp.StatusCode != http.StatusUnauthorized {
		cancel()
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}

	cancel()
	if err := <-done; err != nil {
		t.Fatalf("Run error: %v", err)
	}
}

func appConfigForTest(addr string) config.AppConfig {
	return config.AppConfig{
		Server: config.ServerConfig{
			Address:         addr,
			APIPrefix:       "/api",
			ShutdownTimeout: 2 * time.Second,
		},
		Store: config.StoreConfig{Mode: "memory"},
		Security: config.SecurityConfig{
			JWTSecret: "test-secret",
		},
		Log: config.LogConfig{Level: "error"},
	}
}

func waitForServer(t *testing.T, url string) {
	t.Helper()
	client := &http.Client{Timeout: 200 * time.Millisecond}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("server did not become ready: %s", url)
}
