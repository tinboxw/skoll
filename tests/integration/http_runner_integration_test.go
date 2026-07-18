package integration

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/bootstrap"
	"github.com/tinboxw/skoll/pkg/config"
	"github.com/tinboxw/skoll/pkg/security"
)

func TestRunnerHealthAndAuth(t *testing.T) {
	cfg := bootstrap.RuntimeConfig{
		AppConfig: config.AppConfig{
			Server: config.ServerConfig{Address: "127.0.0.1:18091", APIPrefix: "/skoll", ShutdownTimeout: 2 * time.Second},
			Store:  config.StoreConfig{Mode: "memory"},
			Security: config.SecurityConfig{
				JWTSecret: "integration-secret",
			},
			Log: config.LogConfig{Level: "error"},
		},
		AuthPolicy: bootstrap.AuthPolicy{Enabled: true, SkipPaths: map[string]struct{}{"/skoll/health": {}, "/skoll/ready": {}}},
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

	waitForUp(t, "http://127.0.0.1:18091/skoll/health")

	client := &http.Client{Timeout: 2 * time.Second}
	resp1, err := client.Get("http://127.0.0.1:18091/skoll/health")
	if err != nil {
		t.Fatalf("GET /skoll/health error: %v", err)
	}
	defer resp1.Body.Close()
	if resp1.StatusCode != http.StatusOK {
		t.Fatalf("/skoll/health status=%d", resp1.StatusCode)
	}

	resp2, err := client.Get("http://127.0.0.1:18091/skoll/v1/users")
	if err != nil {
		t.Fatalf("GET /skoll/v1/users error: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusUnauthorized {
		t.Fatalf("/skoll/v1/users expected 401, got %d", resp2.StatusCode)
	}
}

func TestRunnerPermissionCatalogVisibleAfterLogin(t *testing.T) {
	pluginsRoot := filepath.Join(t.TempDir(), "plugins")
	if err := os.MkdirAll(filepath.Join(pluginsRoot, "alpha"), 0o755); err != nil {
		t.Fatalf("create alpha dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(pluginsRoot, "beta"), 0o755); err != nil {
		t.Fatalf("create beta dir: %v", err)
	}

	alphaManifest := []byte("id: alpha\nname: \"Alpha\"\nversion: 0.1.0\napi_version: v1\ncompatibility_skoll: \">=1.0.0 <2.0.0\"\npermissions:\n  - \"alpha.read\"\n")
	betaManifest := []byte("id: beta\nname: \"Beta\"\nversion: 0.1.0\napi_version: v1\ncompatibility_skoll: \">=1.0.0 <2.0.0\"\npermissions:\n  - \"beta.read\"\n")
	if err := os.WriteFile(filepath.Join(pluginsRoot, "alpha", "plugin.yaml"), alphaManifest, 0o644); err != nil {
		t.Fatalf("write alpha manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginsRoot, "beta", "plugin.yaml"), betaManifest, 0o644); err != nil {
		t.Fatalf("write beta manifest: %v", err)
	}

	cfg := bootstrap.RuntimeConfig{
		AppConfig: config.AppConfig{
			Server: config.ServerConfig{Address: "127.0.0.1:18092", APIPrefix: "/skoll", ShutdownTimeout: 2 * time.Second},
			Store:  config.StoreConfig{Mode: "memory"},
			Security: config.SecurityConfig{
				JWTSecret: "integration-secret",
			},
			Log: config.LogConfig{Level: "error"},
			Dev: config.DevConfig{
				PortalEnabled: true,
				PluginsRoot:   pluginsRoot,
				PluginsRoots:  []string{pluginsRoot},
			},
		},
		AuthPolicy: bootstrap.AuthPolicy{Enabled: true, SkipPaths: map[string]struct{}{"/skoll/health": {}, "/skoll/ready": {}}},
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

	baseURL := "http://127.0.0.1:18092"
	waitForUp(t, baseURL+"/skoll/health")

	token := issueSuperAdminToken(t, "integration-secret")

	selfURL := baseURL + "/skoll/v1/plugins/dev/permission-catalog?pluginsRoot=" + url.QueryEscape(pluginsRoot) + "&pluginId=alpha"
	selfResp := callPermissionCatalog(t, selfURL, token)

	if len(selfResp.Data.Framework) == 0 {
		t.Fatalf("expected framework permissions to be non-empty")
	}
	if len(selfResp.Data.Suggested) == 0 {
		t.Fatalf("expected suggested permissions to be non-empty")
	}
	if !contains(selfResp.Data.Self, "alpha.read") {
		t.Fatalf("expected self permissions to include alpha.read, got %+v", selfResp.Data.Self)
	}
	if !containsPluginPermission(selfResp.Data.Plugins, "beta", "beta.read") {
		t.Fatalf("expected plugins group to include beta.read from beta plugin, got %+v", selfResp.Data.Plugins)
	}

	frameworkOnlyURL := baseURL + "/skoll/v1/plugins/dev/permission-catalog?pluginsRoot=" + url.QueryEscape(pluginsRoot)
	frameworkOnlyResp := callPermissionCatalog(t, frameworkOnlyURL, token)
	if len(frameworkOnlyResp.Data.Framework) == 0 {
		t.Fatalf("expected framework permissions when pluginId omitted")
	}
}

type permissionCatalogEnvelope struct {
	Code string `json:"code"`
	Data struct {
		Framework []string `json:"framework"`
		Self      []string `json:"self"`
		Suggested []string `json:"suggested"`
		Plugins   []struct {
			PluginID    string   `json:"pluginId"`
			Permissions []string `json:"permissions"`
		} `json:"plugins"`
	} `json:"data"`
}

func issueSuperAdminToken(t *testing.T, secret string) string {
	t.Helper()

	token, err := security.SignJWT(secret, security.JWTIdentity{Subject: "1", Role: "super_admin", Roles: []string{"super_admin"}}, time.Hour, time.Now().UTC())
	if err != nil {
		t.Fatalf("sign integration jwt: %v", err)
	}
	return token
}

func callPermissionCatalog(t *testing.T, endpoint, token string) permissionCatalogEnvelope {
	t.Helper()

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		t.Fatalf("build permission-catalog request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("permission-catalog request failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read permission-catalog response: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("permission-catalog status=%d body=%s", resp.StatusCode, string(body))
	}

	var parsed permissionCatalogEnvelope
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("decode permission-catalog response: %v body=%s", err, string(body))
	}
	if !strings.EqualFold(parsed.Code, "ok") {
		t.Fatalf("permission-catalog code=%s body=%s", parsed.Code, string(body))
	}
	return parsed
}

func contains(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func containsPluginPermission(groups []struct {
	PluginID    string   `json:"pluginId"`
	Permissions []string `json:"permissions"`
}, pluginID, permission string) bool {
	for _, group := range groups {
		if group.PluginID != pluginID {
			continue
		}
		for _, item := range group.Permissions {
			if item == permission {
				return true
			}
		}
	}
	return false
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
