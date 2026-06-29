package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRemoteMarketplaceAdapterFetchesAndValidatesIndex(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "application/json" {
			t.Fatalf("expected json accept header, got %q", r.Header.Get("Accept"))
		}
		if err := json.NewEncoder(w).Encode(validMarketplaceIndex()); err != nil {
			t.Fatalf("encode index: %v", err)
		}
	}))
	defer server.Close()

	index, err := NewRemoteMarketplaceAdapter(server.Client()).Fetch(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("fetch remote marketplace: %v", err)
	}
	if len(index.Plugins) != 1 || index.Plugins[0].ID != "demo" {
		t.Fatalf("unexpected index: %+v", index)
	}
}

func TestRemoteMarketplaceAdapterReturnsVisibleUnavailableError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "offline", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	_, err := NewRemoteMarketplaceAdapter(server.Client()).Fetch(context.Background(), server.URL)
	if !errors.Is(err, ErrRemoteMarketplaceUnavailable) {
		t.Fatalf("expected remote unavailable error, got %v", err)
	}
	if !strings.Contains(err.Error(), "status 503") {
		t.Fatalf("expected visible status in error, got %v", err)
	}
}

func TestMarketplaceCatalogServiceKeepsLocalCatalogWhenRemoteFails(t *testing.T) {
	root := t.TempDir()
	pluginDir := filepath.Join(root, "demo")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatalf("mkdir plugin: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin.yaml"), []byte(`id: demo
name: Demo Plugin
version: 0.2.0
api_version: v1
compatibility_skoll: ">=1.0.0 <2.0.0"
`), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "remote down", http.StatusBadGateway)
	}))
	defer server.Close()

	result, err := NewMarketplaceCatalogService(nil, NewRemoteMarketplaceAdapter(server.Client())).List(context.Background(), root, server.URL)
	if err != nil {
		t.Fatalf("list marketplace catalog: %v", err)
	}
	if len(result.Local.Items) != 1 || result.Local.Items[0].ID != "demo" {
		t.Fatalf("expected local catalog to survive remote failure, got %+v", result.Local.Items)
	}
	if result.Remote != nil || result.RemoteError == "" {
		t.Fatalf("expected visible remote error without remote index, got %+v", result)
	}
	if !strings.Contains(result.RemoteError, "status 502") {
		t.Fatalf("expected remote status in result error, got %q", result.RemoteError)
	}
}
