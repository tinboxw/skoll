package plugin

import (
	"testing"
	"time"
)

func TestNewPluginContextDefinesDefaultHostCapabilities(t *testing.T) {
	now := time.Date(2026, 7, 4, 11, 0, 0, 0, time.UTC)
	ctx := NewPluginContext(Info{
		ID:          "demo",
		Name:        "Demo",
		Version:     "1.0.0",
		I18nLocales: []string{"zh-CN", "en-US", "zh-CN"},
	}, "skoll", "en-US", now)

	if ctx.PluginID != "demo" || ctx.APIPrefix != "/skoll" || ctx.Locale != "en-US" {
		t.Fatalf("unexpected context identity: %#v", ctx)
	}
	if len(ctx.Locales) != 2 {
		t.Fatalf("expected deduplicated locales, got %#v", ctx.Locales)
	}
	for _, capability := range defaultHostCapabilities {
		if !ctx.HasCapability(capability) {
			t.Fatalf("expected capability %s", capability)
		}
	}
	if ctx.IssuedAt != now {
		t.Fatalf("expected issued time %s, got %s", now, ctx.IssuedAt)
	}
}

func TestPluginContextEndpointsUseCurrentContracts(t *testing.T) {
	ctx := NewPluginContext(Info{ID: "demo"}, "/skoll/", "", time.Time{})
	paths := map[string]struct{}{}
	for _, endpoint := range ctx.Endpoints {
		paths[endpoint.Method+" "+endpoint.Path] = struct{}{}
	}
	expected := []string{
		"GET /skoll/v1/auth/me",
		"GET /skoll/v1/system/settings/skoll.organization.departments",
		"GET /skoll/v1/system/dictionaries",
		"GET /skoll/v1/files",
		"GET /skoll/v1/audit",
		"GET /skoll/v1/plugins/demo/config",
		"PUT /skoll/v1/plugins/demo/config",
		"GET /skoll/v1/permissions",
	}
	for _, path := range expected {
		if _, ok := paths[path]; !ok {
			t.Fatalf("expected endpoint %s in %#v", path, ctx.Endpoints)
		}
	}
}
