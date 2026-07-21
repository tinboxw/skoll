package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	domainrole "github.com/tinboxw/skoll/internal/domain/role"
	"github.com/tinboxw/skoll/internal/plugin"
	rolesvc "github.com/tinboxw/skoll/internal/service/role"
)

func TestPluginHandlerDevPermissionCatalogRequiresSuperAdmin(t *testing.T) {
	pluginsRoot := filepath.Join(t.TempDir(), "plugins")
	mux := http.NewServeMux()
	RegisterPluginRoutes(mux, &fakePluginManager{items: map[string]plugin.Info{}}, WithPluginDevPortal(true, pluginsRoot, []string{pluginsRoot}))

	catalogReq := httptest.NewRequest(http.MethodGet, "/v1/plugins/dev/permission-catalog?pluginsRoot="+filepath.ToSlash(pluginsRoot)+"&pluginId=demo", nil)
	catalogReq = withRole(catalogReq, "admin")
	catalogResp := httptest.NewRecorder()
	mux.ServeHTTP(catalogResp, catalogReq)

	if catalogResp.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for non super_admin, got %d body=%s", catalogResp.Code, catalogResp.Body.String())
	}
}

func TestPluginHandlerDevPortalRoutesWhenDisabled(t *testing.T) {
	pluginsRoot := filepath.Join(t.TempDir(), "plugins")
	mux := http.NewServeMux()
	RegisterPluginRoutes(mux, &fakePluginManager{items: map[string]plugin.Info{}}, WithPluginDevPortal(false, pluginsRoot, []string{pluginsRoot}))

	configReq := httptest.NewRequest(http.MethodGet, "/v1/plugins/dev/config", nil)
	configReq = withRole(configReq, "super_admin")
	configResp := httptest.NewRecorder()
	mux.ServeHTTP(configResp, configReq)
	if configResp.Code != http.StatusOK {
		t.Fatalf("expected dev config route to resolve even when disabled, got %d body=%s", configResp.Code, configResp.Body.String())
	}

	var configBody struct {
		Data devConfigResponse `json:"data"`
	}
	if err := json.Unmarshal(configResp.Body.Bytes(), &configBody); err != nil {
		t.Fatalf("decode config response: %v", err)
	}
	if configBody.Data.Enabled {
		t.Fatalf("expected enabled=false for disabled portal, got %+v", configBody.Data)
	}

	catalogReq := httptest.NewRequest(http.MethodGet, "/v1/plugins/dev/permission-catalog?pluginsRoot="+filepath.ToSlash(pluginsRoot), nil)
	catalogReq = withRole(catalogReq, "super_admin")
	catalogResp := httptest.NewRecorder()
	mux.ServeHTTP(catalogResp, catalogReq)
	if catalogResp.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for permission-catalog when portal disabled, got %d body=%s", catalogResp.Code, catalogResp.Body.String())
	}
}

func TestPluginHandlerDevPermissionCatalogWithoutPluginIDShowsFramework(t *testing.T) {
	pluginsRoot := filepath.Join(t.TempDir(), "plugins")
	mux := http.NewServeMux()
	RegisterPluginRoutes(mux, &fakePluginManager{items: map[string]plugin.Info{}}, WithPluginDevPortal(true, pluginsRoot, []string{pluginsRoot}))

	catalogReq := httptest.NewRequest(http.MethodGet, "/v1/plugins/dev/permission-catalog?pluginsRoot="+filepath.ToSlash(pluginsRoot), nil)
	catalogReq = withRole(catalogReq, "super_admin")
	catalogResp := httptest.NewRecorder()
	mux.ServeHTTP(catalogResp, catalogReq)

	if catalogResp.Code != http.StatusOK {
		t.Fatalf("expected 200 without pluginId, got %d body=%s", catalogResp.Code, catalogResp.Body.String())
	}

	var body struct {
		Data devPermissionCatalogResponse `json:"data"`
	}
	if err := json.Unmarshal(catalogResp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode catalog response: %v", err)
	}

	if body.Data.PluginID != "" {
		t.Fatalf("expected empty pluginId in response, got %q", body.Data.PluginID)
	}
	if !containsPermission(body.Data.Framework, "permission.manage") {
		t.Fatalf("expected framework permissions when pluginId omitted, got %+v", body.Data.Framework)
	}
}

func TestPluginHandlerDevPermissionCatalogAggregatesPermissions(t *testing.T) {
	pluginsRoot := filepath.Join(t.TempDir(), "plugins")
	mux := http.NewServeMux()
	RegisterPluginRoutes(mux, &fakePluginManager{items: map[string]plugin.Info{}}, WithPluginDevPortal(true, pluginsRoot, []string{pluginsRoot}))

	scaffoldTargetPayload := []byte(`{"pluginsRoot":"` + filepath.ToSlash(pluginsRoot) + `","pluginId":"alpha","pluginName":"Alpha"}`)
	scaffoldTargetReq := httptest.NewRequest(http.MethodPost, "/v1/plugins/dev/scaffold", bytes.NewReader(scaffoldTargetPayload))
	scaffoldTargetReq = withRole(scaffoldTargetReq, "super_admin")
	scaffoldTargetResp := httptest.NewRecorder()
	mux.ServeHTTP(scaffoldTargetResp, scaffoldTargetReq)
	if scaffoldTargetResp.Code != http.StatusCreated {
		t.Fatalf("scaffold target status=%d body=%s", scaffoldTargetResp.Code, scaffoldTargetResp.Body.String())
	}

	scaffoldOtherPayload := []byte(`{"pluginsRoot":"` + filepath.ToSlash(pluginsRoot) + `","pluginId":"beta","pluginName":"Beta"}`)
	scaffoldOtherReq := httptest.NewRequest(http.MethodPost, "/v1/plugins/dev/scaffold", bytes.NewReader(scaffoldOtherPayload))
	scaffoldOtherReq = withRole(scaffoldOtherReq, "super_admin")
	scaffoldOtherResp := httptest.NewRecorder()
	mux.ServeHTTP(scaffoldOtherResp, scaffoldOtherReq)
	if scaffoldOtherResp.Code != http.StatusCreated {
		t.Fatalf("scaffold other status=%d body=%s", scaffoldOtherResp.Code, scaffoldOtherResp.Body.String())
	}

	betaManifest := []byte("id: beta\nname: \"Beta\"\nversion: 0.1.0\napi_version: v1\npermissions:\n  - \"beta.read\"\n  - \"beta.write\"\n")
	if err := os.WriteFile(filepath.Join(pluginsRoot, "beta", "plugin.yaml"), betaManifest, 0o644); err != nil {
		t.Fatalf("write beta manifest: %v", err)
	}

	catalogReq := httptest.NewRequest(http.MethodGet, "/v1/plugins/dev/permission-catalog?pluginsRoot="+filepath.ToSlash(pluginsRoot)+"&pluginId=alpha", nil)
	catalogReq = withRole(catalogReq, "super_admin")
	catalogResp := httptest.NewRecorder()
	mux.ServeHTTP(catalogResp, catalogReq)
	if catalogResp.Code != http.StatusOK {
		t.Fatalf("catalog status=%d body=%s", catalogResp.Code, catalogResp.Body.String())
	}

	var body struct {
		Data devPermissionCatalogResponse `json:"data"`
	}
	if err := json.Unmarshal(catalogResp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode catalog response: %v", err)
	}

	if body.Data.Operation != "permission_catalog" || body.Data.Status != "ok" {
		t.Fatalf("unexpected envelope: %+v", body.Data)
	}
	if !containsPermission(body.Data.Self, "alpha.read") {
		t.Fatalf("expected self permissions to include alpha.read, got %+v", body.Data.Self)
	}
	if !containsPermission(body.Data.Framework, "permission.manage") {
		t.Fatalf("expected framework permissions to include permission.manage, got %+v", body.Data.Framework)
	}
	if len(body.Data.Plugins) == 0 || body.Data.Plugins[0].PluginID != "beta" {
		t.Fatalf("expected beta group in plugins list, got %+v", body.Data.Plugins)
	}
	if !containsPermission(body.Data.Plugins[0].Permissions, "beta.write") {
		t.Fatalf("expected beta.write in other plugin permissions, got %+v", body.Data.Plugins[0].Permissions)
	}
	if !containsPermission(body.Data.Suggested, "alpha.read") || !containsPermission(body.Data.Suggested, "beta.write") {
		t.Fatalf("expected suggested to contain aggregated permissions, got %+v", body.Data.Suggested)
	}
}

func TestPluginHandlerDevPermissionCatalogFrameworkFromBuiltInRoles(t *testing.T) {
	pluginsRoot := filepath.Join(t.TempDir(), "plugins")
	mux := http.NewServeMux()
	roleProvider := &fakeRoleCatalogProvider{
		roles: []*domainrole.Role{
			{BuiltIn: true, Permissions: []string{"framework.manage", "framework.read"}},
			{BuiltIn: false, Permissions: []string{"custom.role.permission"}},
		},
	}
	RegisterPluginRoutes(
		mux,
		&fakePluginManager{items: map[string]plugin.Info{}},
		WithPluginRoleCatalogProvider(roleProvider),
		WithPluginDevPortal(true, pluginsRoot, []string{pluginsRoot}),
	)

	scaffoldTargetPayload := []byte(`{"pluginsRoot":"` + filepath.ToSlash(pluginsRoot) + `","pluginId":"alpha","pluginName":"Alpha"}`)
	scaffoldTargetReq := httptest.NewRequest(http.MethodPost, "/v1/plugins/dev/scaffold", bytes.NewReader(scaffoldTargetPayload))
	scaffoldTargetReq = withRole(scaffoldTargetReq, "super_admin")
	scaffoldTargetResp := httptest.NewRecorder()
	mux.ServeHTTP(scaffoldTargetResp, scaffoldTargetReq)
	if scaffoldTargetResp.Code != http.StatusCreated {
		t.Fatalf("scaffold target status=%d body=%s", scaffoldTargetResp.Code, scaffoldTargetResp.Body.String())
	}

	catalogReq := httptest.NewRequest(http.MethodGet, "/v1/plugins/dev/permission-catalog?pluginsRoot="+filepath.ToSlash(pluginsRoot)+"&pluginId=alpha", nil)
	catalogReq = withRole(catalogReq, "super_admin")
	catalogResp := httptest.NewRecorder()
	mux.ServeHTTP(catalogResp, catalogReq)
	if catalogResp.Code != http.StatusOK {
		t.Fatalf("catalog status=%d body=%s", catalogResp.Code, catalogResp.Body.String())
	}

	var body struct {
		Data devPermissionCatalogResponse `json:"data"`
	}
	if err := json.Unmarshal(catalogResp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode catalog response: %v", err)
	}

	if !containsPermission(body.Data.Framework, "framework.manage") || !containsPermission(body.Data.Framework, "framework.read") {
		t.Fatalf("expected dynamic framework permissions from built-in roles, got %+v", body.Data.Framework)
	}
	if containsPermission(body.Data.Framework, "custom.role.permission") {
		t.Fatalf("expected non-built-in role permissions excluded from framework group, got %+v", body.Data.Framework)
	}
}

func TestPluginHandlerDevPermissionCatalogFrameworkFallbackWhenRoleCatalogFails(t *testing.T) {
	pluginsRoot := filepath.Join(t.TempDir(), "plugins")
	mux := http.NewServeMux()
	RegisterPluginRoutes(
		mux,
		&fakePluginManager{items: map[string]plugin.Info{}},
		WithPluginRoleCatalogProvider(&fakeRoleCatalogProvider{err: errors.New("role service unavailable")}),
		WithPluginDevPortal(true, pluginsRoot, []string{pluginsRoot}),
	)

	scaffoldTargetPayload := []byte(`{"pluginsRoot":"` + filepath.ToSlash(pluginsRoot) + `","pluginId":"alpha","pluginName":"Alpha"}`)
	scaffoldTargetReq := httptest.NewRequest(http.MethodPost, "/v1/plugins/dev/scaffold", bytes.NewReader(scaffoldTargetPayload))
	scaffoldTargetReq = withRole(scaffoldTargetReq, "super_admin")
	scaffoldTargetResp := httptest.NewRecorder()
	mux.ServeHTTP(scaffoldTargetResp, scaffoldTargetReq)
	if scaffoldTargetResp.Code != http.StatusCreated {
		t.Fatalf("scaffold target status=%d body=%s", scaffoldTargetResp.Code, scaffoldTargetResp.Body.String())
	}

	catalogReq := httptest.NewRequest(http.MethodGet, "/v1/plugins/dev/permission-catalog?pluginsRoot="+filepath.ToSlash(pluginsRoot)+"&pluginId=alpha", nil)
	catalogReq = withRole(catalogReq, "super_admin")
	catalogResp := httptest.NewRecorder()
	mux.ServeHTTP(catalogResp, catalogReq)
	if catalogResp.Code != http.StatusOK {
		t.Fatalf("catalog status=%d body=%s", catalogResp.Code, catalogResp.Body.String())
	}

	var body struct {
		Data devPermissionCatalogResponse `json:"data"`
	}
	if err := json.Unmarshal(catalogResp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode catalog response: %v", err)
	}

	if !containsPermission(body.Data.Framework, "permission.manage") {
		t.Fatalf("expected fallback framework permission seeds when role catalog fails, got %+v", body.Data.Framework)
	}
}

type fakeRoleCatalogProvider struct {
	roles []*domainrole.Role
	err   error
}

func (f *fakeRoleCatalogProvider) List(_ context.Context, _ rolesvc.ListInput) ([]*domainrole.Role, error) {
	if f == nil {
		return nil, nil
	}
	if f.err != nil {
		return nil, f.err
	}
	return f.roles, nil
}

func containsPermission(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
