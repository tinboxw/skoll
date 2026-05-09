package v1

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tinboxw/skoll/internal/plugin"
)

type fakePluginProvider struct {
	items []plugin.Info
}

func (f fakePluginProvider) List() []plugin.Info {
	return append([]plugin.Info(nil), f.items...)
}

func TestPluginHandlerListFromProvider(t *testing.T) {
	mux := http.NewServeMux()
	RegisterPluginRoutes(mux, fakePluginProvider{items: []plugin.Info{
		{ID: "demo", Name: "Demo", Version: "0.1.0", State: plugin.StateEnabled},
		{ID: "tool", Name: "Tool", Version: "0.2.0", State: plugin.StateInstalled},
	}})

	req := httptest.NewRequest(http.MethodGet, "/v1/plugins", nil)
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}

	var body struct {
		Code string         `json:"code"`
		Data []pluginRecord `json:"data"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if body.Code != "ok" {
		t.Fatalf("unexpected code: %s", body.Code)
	}
	if len(body.Data) != 2 {
		t.Fatalf("expected 2 records, got %d", len(body.Data))
	}
}

func TestPluginHandlerEnabledFilterFallback(t *testing.T) {
	mux := http.NewServeMux()
	RegisterPluginRoutes(mux, fakePluginProvider{items: []plugin.Info{
		{ID: "demo", Name: "Demo", Version: "0.1.0", State: plugin.StateInstalled},
	}})

	req := httptest.NewRequest(http.MethodGet, "/v1/plugins?enabled=true", nil)
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}

	var body struct {
		Data []pluginRecord `json:"data"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(body.Data) != 1 || body.Data[0].ID != "builtin-auth" {
		t.Fatalf("expected fallback builtin-auth, got %+v", body.Data)
	}
}
