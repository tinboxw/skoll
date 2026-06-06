package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/plugin"
	"github.com/tinboxw/skoll/pkg/security"
)

type fakePluginManager struct {
	items     map[string]plugin.Info
	snapshots map[string]plugin.RegistrySnapshot
}

type noConfigUpdaterPluginManager struct {
	items map[string]plugin.Info
}

func (f *noConfigUpdaterPluginManager) Install(path string) (plugin.Info, error) {
	_ = path
	return plugin.Info{}, nil
}

func (f *noConfigUpdaterPluginManager) List() []plugin.Info {
	items := make([]plugin.Info, 0, len(f.items))
	for _, item := range f.items {
		items = append(items, item)
	}
	return items
}

func (f *noConfigUpdaterPluginManager) Enable(pluginID string) error {
	item, ok := f.items[pluginID]
	if !ok {
		return plugin.ErrPluginNotFound
	}
	item.State = plugin.StateEnabled
	f.items[pluginID] = item
	return nil
}

func (f *noConfigUpdaterPluginManager) Disable(pluginID string) error {
	item, ok := f.items[pluginID]
	if !ok {
		return plugin.ErrPluginNotFound
	}
	if item.SystemBuiltin || strings.EqualFold(item.Source, "builtin") {
		return plugin.ErrPluginSystemProtected
	}
	item.State = plugin.StateDisabled
	f.items[pluginID] = item
	return nil
}

func (f *noConfigUpdaterPluginManager) Uninstall(pluginID string) error {
	item, ok := f.items[pluginID]
	if !ok {
		return plugin.ErrPluginNotFound
	}
	if item.SystemBuiltin || strings.EqualFold(item.Source, "builtin") {
		return plugin.ErrPluginSystemProtected
	}
	item.State = plugin.StateUninstalled
	f.items[pluginID] = item
	return nil
}

func (f *noConfigUpdaterPluginManager) Get(pluginID string) (plugin.Info, error) {
	item, ok := f.items[pluginID]
	if !ok {
		return plugin.Info{}, plugin.ErrPluginNotFound
	}
	return item, nil
}

func (f *fakePluginManager) SavePluginConfig(pluginID string, config map[string]any) error {
	item, ok := f.items[pluginID]
	if !ok {
		return plugin.ErrPluginNotFound
	}
	raw, err := json.Marshal(config)
	if err != nil {
		return err
	}
	item.ConfigJSON = string(raw)
	f.items[pluginID] = item
	return nil
}

func (f *fakePluginManager) Install(path string) (plugin.Info, error) {
	_ = path
	return plugin.Info{}, nil
}

func (f *fakePluginManager) List() []plugin.Info {
	items := make([]plugin.Info, 0, len(f.items))
	for _, item := range f.items {
		items = append(items, item)
	}
	return items
}

func (f *fakePluginManager) Enable(pluginID string) error {
	item, ok := f.items[pluginID]
	if !ok {
		return plugin.ErrPluginNotFound
	}
	item.State = plugin.StateEnabled
	f.items[pluginID] = item
	return nil
}

func (f *fakePluginManager) Disable(pluginID string) error {
	item, ok := f.items[pluginID]
	if !ok {
		return plugin.ErrPluginNotFound
	}
	if item.SystemBuiltin || strings.EqualFold(item.Source, "builtin") {
		return plugin.ErrPluginSystemProtected
	}
	item.State = plugin.StateDisabled
	f.items[pluginID] = item
	return nil
}

func (f *fakePluginManager) Uninstall(pluginID string) error {
	item, ok := f.items[pluginID]
	if !ok {
		return plugin.ErrPluginNotFound
	}
	if item.SystemBuiltin || strings.EqualFold(item.Source, "builtin") {
		return plugin.ErrPluginSystemProtected
	}
	item.State = plugin.StateUninstalled
	f.items[pluginID] = item
	return nil
}

func (f *fakePluginManager) Get(pluginID string) (plugin.Info, error) {
	item, ok := f.items[pluginID]
	if !ok {
		return plugin.Info{}, plugin.ErrPluginNotFound
	}
	return item, nil
}

func (f *fakePluginManager) GetExtensionSnapshot(pluginID string) (plugin.RegistrySnapshot, bool) {
	s, ok := f.snapshots[pluginID]
	return s, ok
}

func withRole(req *http.Request, role string) *http.Request {
	claims := &security.JWTClaims{Subject: "u-1", Role: role}
	ctx := security.WithJWTClaimsContext(context.Background(), claims)
	return req.WithContext(ctx)
}

func TestPluginHandlerListFromProvider(t *testing.T) {
	mux := http.NewServeMux()
	RegisterPluginRoutes(mux, &fakePluginManager{items: map[string]plugin.Info{
		"demo": {ID: "demo", Name: "Demo", Version: "0.1.0", State: plugin.StateEnabled},
		"tool": {ID: "tool", Name: "Tool", Version: "0.2.0", State: plugin.StateInstalled},
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
	if body.Data[0].UIMode == "" || body.Data[1].UIMode == "" {
		t.Fatalf("expected ui mode in list payload: %+v", body.Data)
	}
	if body.Data[0].Level == "" || body.Data[0].MountPolicy == "" {
		t.Fatalf("expected level/mountPolicy in list payload: %+v", body.Data[0])
	}
}

func TestPluginHandlerEnabledFilterFallback(t *testing.T) {
	mux := http.NewServeMux()
	RegisterPluginRoutes(mux, &fakePluginManager{items: map[string]plugin.Info{
		"demo": {ID: "demo", Name: "Demo", Version: "0.1.0", State: plugin.StateInstalled},
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
	if body.Data[0].Level != string(plugin.LevelSystem) {
		t.Fatalf("unexpected fallback level: %s", body.Data[0].Level)
	}
}

func TestPluginHandlerStateOperations(t *testing.T) {
	mgr := &fakePluginManager{items: map[string]plugin.Info{
		"demo": {ID: "demo", Name: "Demo", Version: "0.1.0", State: plugin.StateInstalled},
	}}
	mux := http.NewServeMux()
	RegisterPluginRoutes(mux, mgr)

	enableReq := httptest.NewRequest(http.MethodPost, "/v1/plugins/demo/enable", nil)
	enableResp := httptest.NewRecorder()
	mux.ServeHTTP(enableResp, enableReq)
	if enableResp.Code != http.StatusOK {
		t.Fatalf("enable status=%d body=%s", enableResp.Code, enableResp.Body.String())
	}

	disableReq := httptest.NewRequest(http.MethodPost, "/v1/plugins/demo/disable", nil)
	disableResp := httptest.NewRecorder()
	mux.ServeHTTP(disableResp, disableReq)
	if disableResp.Code != http.StatusOK {
		t.Fatalf("disable status=%d body=%s", disableResp.Code, disableResp.Body.String())
	}

	uninstallReq := httptest.NewRequest(http.MethodDelete, "/v1/plugins/demo", nil)
	uninstallResp := httptest.NewRecorder()
	mux.ServeHTTP(uninstallResp, uninstallReq)
	if uninstallResp.Code != http.StatusOK {
		t.Fatalf("uninstall status=%d body=%s", uninstallResp.Code, uninstallResp.Body.String())
	}
}

func TestPluginHandlerProtectedBuiltinActions(t *testing.T) {
	mgr := &fakePluginManager{items: map[string]plugin.Info{
		"builtin-auth": {ID: "builtin-auth", Name: "Builtin Auth", Version: "1.0.0", State: plugin.StateEnabled, Source: "builtin", SystemBuiltin: true},
	}}
	mux := http.NewServeMux()
	RegisterPluginRoutes(mux, mgr)

	disableReq := httptest.NewRequest(http.MethodPost, "/v1/plugins/builtin-auth/disable", nil)
	disableResp := httptest.NewRecorder()
	mux.ServeHTTP(disableResp, disableReq)
	if disableResp.Code != http.StatusForbidden {
		t.Fatalf("disable builtin status=%d body=%s", disableResp.Code, disableResp.Body.String())
	}

	uninstallReq := httptest.NewRequest(http.MethodDelete, "/v1/plugins/builtin-auth", nil)
	uninstallResp := httptest.NewRecorder()
	mux.ServeHTTP(uninstallResp, uninstallReq)
	if uninstallResp.Code != http.StatusForbidden {
		t.Fatalf("uninstall builtin status=%d body=%s", uninstallResp.Code, uninstallResp.Body.String())
	}
}

func TestPluginHandlerInstallAndValidate(t *testing.T) {
	tmp := t.TempDir()
	manifestDir := filepath.Join(tmp, "sample")
	if err := os.MkdirAll(manifestDir, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	manifest := strings.Join([]string{
		"id: sample",
		"name: Sample Plugin",
		"version: 0.1.0",
		"permissions:",
		"  - menu.read",
	}, "\n")
	if err := os.WriteFile(filepath.Join(manifestDir, "plugin.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest failed: %v", err)
	}

	mgr := &fakePluginManager{items: map[string]plugin.Info{}}
	mux := http.NewServeMux()
	RegisterPluginRoutes(mux, mgr)

	validatePayload := []byte(`{"path":"` + filepath.ToSlash(manifestDir) + `"}`)
	validateReq := httptest.NewRequest(http.MethodPost, "/v1/plugins/validate", bytes.NewReader(validatePayload))
	validateResp := httptest.NewRecorder()
	mux.ServeHTTP(validateResp, validateReq)
	if validateResp.Code != http.StatusOK {
		t.Fatalf("validate status=%d body=%s", validateResp.Code, validateResp.Body.String())
	}

	installReq := httptest.NewRequest(http.MethodPost, "/v1/plugins/install", bytes.NewReader(validatePayload))
	installResp := httptest.NewRecorder()
	mux.ServeHTTP(installResp, installReq)
	if installResp.Code != http.StatusCreated {
		t.Fatalf("install status=%d body=%s", installResp.Code, installResp.Body.String())
	}
}

func TestPluginHandlerInstallValidateBadRequest(t *testing.T) {
	mux := http.NewServeMux()
	RegisterPluginRoutes(mux, &fakePluginManager{items: map[string]plugin.Info{}})

	for _, path := range []string{"/v1/plugins/install", "/v1/plugins/validate"} {
		req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader([]byte(`{"path":""}`)))
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("%s expected 400 got %d body=%s", path, resp.Code, resp.Body.String())
		}
	}
}

func TestPluginHandlerDevPortalRoutesDisabled(t *testing.T) {
	mux := http.NewServeMux()
	RegisterPluginRoutes(mux, &fakePluginManager{items: map[string]plugin.Info{}})

	req := httptest.NewRequest(http.MethodPost, "/v1/plugins/dev/scaffold", bytes.NewReader([]byte(`{}`)))
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)
	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected 404 when dev portal disabled, got %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestPluginHandlerDevPortalRequiresSuperAdmin(t *testing.T) {
	tmp := t.TempDir()
	mux := http.NewServeMux()
	RegisterPluginRoutes(mux, &fakePluginManager{items: map[string]plugin.Info{}}, WithPluginDevPortal(true, tmp, []string{tmp}))

	payload := []byte(`{"pluginsRoot":"` + filepath.ToSlash(tmp) + `","pluginId":"demo","pluginName":"Demo"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/plugins/dev/scaffold", bytes.NewReader(payload))
	req = withRole(req, "admin")
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)
	if resp.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for non super_admin, got %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestPluginHandlerDevPortalScaffoldRejectsUnexpectedRoot(t *testing.T) {
	allowedRoot := filepath.Join(t.TempDir(), "plugins")
	otherRoot := filepath.Join(t.TempDir(), "other")
	mux := http.NewServeMux()
	RegisterPluginRoutes(mux, &fakePluginManager{items: map[string]plugin.Info{}}, WithPluginDevPortal(true, allowedRoot, []string{allowedRoot}))

	payload := []byte(`{"pluginsRoot":"` + filepath.ToSlash(otherRoot) + `","pluginId":"demo","pluginName":"Demo"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/plugins/dev/scaffold", bytes.NewReader(payload))
	req = withRole(req, "super_admin")
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for non-whitelisted root, got %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestPluginHandlerDevPortalScaffoldAndValidateAll(t *testing.T) {
	pluginsRoot := filepath.Join(t.TempDir(), "plugins")
	mux := http.NewServeMux()
	RegisterPluginRoutes(mux, &fakePluginManager{items: map[string]plugin.Info{}}, WithPluginDevPortal(true, pluginsRoot, []string{pluginsRoot}))

	scaffoldPayload := []byte(`{"pluginsRoot":"` + filepath.ToSlash(pluginsRoot) + `","pluginId":"demo-plugin","pluginName":"Demo Plugin"}`)
	scaffoldReq := httptest.NewRequest(http.MethodPost, "/v1/plugins/dev/scaffold", bytes.NewReader(scaffoldPayload))
	scaffoldReq = withRole(scaffoldReq, "super_admin")
	scaffoldResp := httptest.NewRecorder()
	mux.ServeHTTP(scaffoldResp, scaffoldReq)
	if scaffoldResp.Code != http.StatusCreated {
		t.Fatalf("scaffold status=%d body=%s", scaffoldResp.Code, scaffoldResp.Body.String())
	}
	var scaffoldBody struct {
		Data struct {
			Operation string `json:"operation"`
			Status    string `json:"status"`
			PluginID  string `json:"pluginId"`
		} `json:"data"`
	}
	if err := json.Unmarshal(scaffoldResp.Body.Bytes(), &scaffoldBody); err != nil {
		t.Fatalf("decode scaffold response: %v", err)
	}
	if scaffoldBody.Data.Operation != "scaffold" || scaffoldBody.Data.Status != "ok" || scaffoldBody.Data.PluginID != "demo-plugin" {
		t.Fatalf("unexpected scaffold data: %+v", scaffoldBody.Data)
	}

	manifestPath := filepath.Join(pluginsRoot, "demo-plugin", "plugin.yaml")
	if _, err := os.Stat(manifestPath); err != nil {
		t.Fatalf("expected scaffold manifest created: %v", err)
	}

	validatePayload := []byte(`{"pluginsRoot":"` + filepath.ToSlash(pluginsRoot) + `"}`)
	validateReq := httptest.NewRequest(http.MethodPost, "/v1/plugins/dev/validate-all", bytes.NewReader(validatePayload))
	validateReq = withRole(validateReq, "super_admin")
	validateResp := httptest.NewRecorder()
	mux.ServeHTTP(validateResp, validateReq)
	if validateResp.Code != http.StatusOK {
		t.Fatalf("validate-all status=%d body=%s", validateResp.Code, validateResp.Body.String())
	}

	var body struct {
		Data struct {
			Operation string `json:"operation"`
			Status    string `json:"status"`
			Summary   struct {
				Total   int `json:"total"`
				Valid   int `json:"valid"`
				Invalid int `json:"invalid"`
			} `json:"summary"`
		} `json:"data"`
	}
	if err := json.Unmarshal(validateResp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode validate-all response: %v", err)
	}
	if body.Data.Operation != "validate_all" || body.Data.Status != "ok" {
		t.Fatalf("unexpected validate-all envelope: %+v", body.Data)
	}
	if body.Data.Summary.Total != 1 || body.Data.Summary.Valid != 1 || body.Data.Summary.Invalid != 0 {
		t.Fatalf("unexpected validate-all result: %+v", body.Data)
	}
}

func TestPluginHandlerDevPortalAllowsSecondaryRoot(t *testing.T) {
	primaryRoot := filepath.Join(t.TempDir(), "plugins-primary")
	secondaryRoot := filepath.Join(t.TempDir(), "plugins-secondary")
	mux := http.NewServeMux()
	RegisterPluginRoutes(
		mux,
		&fakePluginManager{items: map[string]plugin.Info{}},
		WithPluginDevPortal(true, primaryRoot, []string{primaryRoot, secondaryRoot}),
	)

	payload := []byte(`{"pluginsRoot":"` + filepath.ToSlash(secondaryRoot) + `","pluginId":"demo-secondary","pluginName":"Demo Secondary"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/plugins/dev/scaffold", bytes.NewReader(payload))
	req = withRole(req, "super_admin")
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected scaffold success on secondary root, got %d body=%s", resp.Code, resp.Body.String())
	}

	manifestPath := filepath.Join(secondaryRoot, "demo-secondary", "plugin.yaml")
	if _, err := os.Stat(manifestPath); err != nil {
		t.Fatalf("expected scaffold manifest created in secondary root: %v", err)
	}
}

func TestPluginHandlerDevPortalScaffoldRepositoryMode(t *testing.T) {
	pluginsRoot := filepath.Join(t.TempDir(), "plugins")
	mux := http.NewServeMux()
	RegisterPluginRoutes(mux, &fakePluginManager{items: map[string]plugin.Info{}}, WithPluginDevPortal(true, pluginsRoot, []string{pluginsRoot}))

	payload := []byte(`{"pluginsRoot":"` + filepath.ToSlash(pluginsRoot) + `","pluginId":"repo-plugin","pluginName":"Repo Plugin","mode":"repository"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/plugins/dev/scaffold", bytes.NewReader(payload))
	req = withRole(req, "super_admin")
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected scaffold repository mode success, got %d body=%s", resp.Code, resp.Body.String())
	}

	mainPath := filepath.Join(pluginsRoot, "repo-plugin", "backend", "cmd", "repo-plugin", "main.go")
	if _, err := os.Stat(mainPath); err != nil {
		t.Fatalf("expected repository mode main.go created: %v", err)
	}
	frontendPkg := filepath.Join(pluginsRoot, "repo-plugin", "frontend", "package.json")
	if _, err := os.Stat(frontendPkg); err != nil {
		t.Fatalf("expected repository mode frontend package.json created: %v", err)
	}
}

func TestPluginHandlerDevPortalScaffoldRejectsInvalidMode(t *testing.T) {
	pluginsRoot := filepath.Join(t.TempDir(), "plugins")
	mux := http.NewServeMux()
	RegisterPluginRoutes(mux, &fakePluginManager{items: map[string]plugin.Info{}}, WithPluginDevPortal(true, pluginsRoot, []string{pluginsRoot}))

	payload := []byte(`{"pluginsRoot":"` + filepath.ToSlash(pluginsRoot) + `","pluginId":"bad-mode","pluginName":"Bad Mode","mode":"unknown"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/plugins/dev/scaffold", bytes.NewReader(payload))
	req = withRole(req, "super_admin")
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid mode bad request, got %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestPluginHandlerDevPortalConfigProjectsRemoveAndPackage(t *testing.T) {
	pluginsRoot := filepath.Join(t.TempDir(), "plugins")
	mgr := &fakePluginManager{items: map[string]plugin.Info{}}
	mux := http.NewServeMux()
	RegisterPluginRoutes(mux, mgr, WithPluginDevPortal(true, pluginsRoot, []string{pluginsRoot}))

	scaffoldPayload := []byte(`{"pluginsRoot":"` + filepath.ToSlash(pluginsRoot) + `","pluginId":"dev-work","pluginName":"Dev Work"}`)
	scaffoldReq := httptest.NewRequest(http.MethodPost, "/v1/plugins/dev/scaffold", bytes.NewReader(scaffoldPayload))
	scaffoldReq = withRole(scaffoldReq, "super_admin")
	scaffoldResp := httptest.NewRecorder()
	mux.ServeHTTP(scaffoldResp, scaffoldReq)
	if scaffoldResp.Code != http.StatusCreated {
		t.Fatalf("scaffold status=%d body=%s", scaffoldResp.Code, scaffoldResp.Body.String())
	}

	configReq := httptest.NewRequest(http.MethodGet, "/v1/plugins/dev/config", nil)
	configReq = withRole(configReq, "super_admin")
	configResp := httptest.NewRecorder()
	mux.ServeHTTP(configResp, configReq)
	if configResp.Code != http.StatusOK {
		t.Fatalf("config status=%d body=%s", configResp.Code, configResp.Body.String())
	}

	projectsPayload := []byte(`{"pluginsRoot":"` + filepath.ToSlash(pluginsRoot) + `"}`)
	projectsReq := httptest.NewRequest(http.MethodPost, "/v1/plugins/dev/projects", bytes.NewReader(projectsPayload))
	projectsReq = withRole(projectsReq, "super_admin")
	projectsResp := httptest.NewRecorder()
	mux.ServeHTTP(projectsResp, projectsReq)
	if projectsResp.Code != http.StatusOK {
		t.Fatalf("projects status=%d body=%s", projectsResp.Code, projectsResp.Body.String())
	}
	var projectsBody struct {
		Data devListProjectsResponse `json:"data"`
	}
	if err := json.Unmarshal(projectsResp.Body.Bytes(), &projectsBody); err != nil {
		t.Fatalf("decode projects response: %v", err)
	}
	if len(projectsBody.Data.Projects) != 1 || projectsBody.Data.Projects[0].PluginID != "dev-work" {
		t.Fatalf("unexpected projects: %+v", projectsBody.Data.Projects)
	}

	packagePayload := []byte(`{"pluginsRoot":"` + filepath.ToSlash(pluginsRoot) + `","pluginId":"dev-work"}`)
	packageReq := httptest.NewRequest(http.MethodPost, "/v1/plugins/dev/package", bytes.NewReader(packagePayload))
	packageReq = withRole(packageReq, "super_admin")
	packageResp := httptest.NewRecorder()
	mux.ServeHTTP(packageResp, packageReq)
	if packageResp.Code != http.StatusOK {
		t.Fatalf("package status=%d body=%s", packageResp.Code, packageResp.Body.String())
	}
	var packageBody struct {
		Data devPackageProjectResponse `json:"data"`
	}
	if err := json.Unmarshal(packageResp.Body.Bytes(), &packageBody); err != nil {
		t.Fatalf("decode package response: %v", err)
	}
	if _, err := os.Stat(packageBody.Data.ArtifactPath); err != nil {
		t.Fatalf("expected artifact file, got err=%v", err)
	}

	removePayload := []byte(`{"pluginsRoot":"` + filepath.ToSlash(pluginsRoot) + `","pluginId":"dev-work","removeFiles":true}`)
	removeReq := httptest.NewRequest(http.MethodPost, "/v1/plugins/dev/remove", bytes.NewReader(removePayload))
	removeReq = withRole(removeReq, "super_admin")
	removeResp := httptest.NewRecorder()
	mux.ServeHTTP(removeResp, removeReq)
	if removeResp.Code != http.StatusOK {
		t.Fatalf("remove status=%d body=%s", removeResp.Code, removeResp.Body.String())
	}
	if _, err := os.Stat(filepath.Join(pluginsRoot, "dev-work")); !os.IsNotExist(err) {
		t.Fatalf("expected project directory removed, err=%v", err)
	}
}

func TestPluginHandlerDevPortalManifestPipelineAndRollout(t *testing.T) {
	pluginsRoot := filepath.Join(t.TempDir(), "plugins")
	mgr := &fakePluginManager{items: map[string]plugin.Info{}}
	mux := http.NewServeMux()
	RegisterPluginRoutes(mux, mgr, WithPluginDevPortal(true, pluginsRoot, []string{pluginsRoot}))

	scaffoldPayload := []byte(`{"pluginsRoot":"` + filepath.ToSlash(pluginsRoot) + `","pluginId":"dev-edit","pluginName":"Dev Edit"}`)
	scaffoldReq := httptest.NewRequest(http.MethodPost, "/v1/plugins/dev/scaffold", bytes.NewReader(scaffoldPayload))
	scaffoldReq = withRole(scaffoldReq, "super_admin")
	scaffoldResp := httptest.NewRecorder()
	mux.ServeHTTP(scaffoldResp, scaffoldReq)
	if scaffoldResp.Code != http.StatusCreated {
		t.Fatalf("scaffold status=%d body=%s", scaffoldResp.Code, scaffoldResp.Body.String())
	}

	manifestGetReq := httptest.NewRequest(http.MethodGet, "/v1/plugins/dev/manifest?pluginsRoot="+filepath.ToSlash(pluginsRoot)+"&pluginId=dev-edit", nil)
	manifestGetReq = withRole(manifestGetReq, "super_admin")
	manifestGetResp := httptest.NewRecorder()
	mux.ServeHTTP(manifestGetResp, manifestGetReq)
	if manifestGetResp.Code != http.StatusOK {
		t.Fatalf("manifest get status=%d body=%s", manifestGetResp.Code, manifestGetResp.Body.String())
	}

	manifest := strings.Join([]string{
		"id: dev-edit",
		"name: \"Dev Edit Updated\"",
		"version: 0.1.1",
		"api_version: v1",
		"compatibility_skoll: \">=1.0.0 <2.0.0\"",
		"migration_version: v0.1.0",
		"ui_mode: separated",
		"level: system",
		"mount_policy: admin",
		"ui_nav_position: none",
		"ui_open_mode: integrated",
		"ui_tab_mode: optional",
		"i18n_locales:",
		"  - zh-CN",
		"  - en-US",
		"permissions:",
		"  - \"dev-edit.read\"",
	}, "\n") + "\n"

	manifestPutPayload, _ := json.Marshal(devManifestRequest{PluginsRoot: filepath.ToSlash(pluginsRoot), PluginID: "dev-edit", Manifest: manifest})
	manifestPutReq := httptest.NewRequest(http.MethodPut, "/v1/plugins/dev/manifest", bytes.NewReader(manifestPutPayload))
	manifestPutReq = withRole(manifestPutReq, "super_admin")
	manifestPutResp := httptest.NewRecorder()
	mux.ServeHTTP(manifestPutResp, manifestPutReq)
	if manifestPutResp.Code != http.StatusOK {
		t.Fatalf("manifest put status=%d body=%s", manifestPutResp.Code, manifestPutResp.Body.String())
	}

	manifestValidateReq := httptest.NewRequest(http.MethodPost, "/v1/plugins/dev/manifest/validate", bytes.NewReader(manifestPutPayload))
	manifestValidateReq = withRole(manifestValidateReq, "super_admin")
	manifestValidateResp := httptest.NewRecorder()
	mux.ServeHTTP(manifestValidateResp, manifestValidateReq)
	if manifestValidateResp.Code != http.StatusOK {
		t.Fatalf("manifest validate status=%d body=%s", manifestValidateResp.Code, manifestValidateResp.Body.String())
	}

	pipelinePayload := []byte(`{"pluginsRoot":"` + filepath.ToSlash(pluginsRoot) + `","pluginId":"dev-edit"}`)
	pipelineReq := httptest.NewRequest(http.MethodPost, "/v1/plugins/dev/pipeline", bytes.NewReader(pipelinePayload))
	pipelineReq = withRole(pipelineReq, "super_admin")
	pipelineResp := httptest.NewRecorder()
	mux.ServeHTTP(pipelineResp, pipelineReq)
	if pipelineResp.Code != http.StatusOK {
		t.Fatalf("pipeline status=%d body=%s", pipelineResp.Code, pipelineResp.Body.String())
	}
	var pipelineBody struct {
		Data devPipelineResponse `json:"data"`
	}
	if err := json.Unmarshal(pipelineResp.Body.Bytes(), &pipelineBody); err != nil {
		t.Fatalf("decode pipeline response: %v", err)
	}
	if pipelineBody.Data.Status != "ok" || len(pipelineBody.Data.Steps) < 2 {
		t.Fatalf("unexpected pipeline response: %+v", pipelineBody.Data)
	}

	// Add runtime item for rollout persistence path.
	mgr.items["dev-edit"] = plugin.Info{ID: "dev-edit", Name: "Dev Edit", Version: "0.1.1", State: plugin.StateEnabled, ConfigJSON: "{}"}

	rolloutPayload := []byte(`{"pluginId":"dev-edit","percent":20}`)
	rolloutReq := httptest.NewRequest(http.MethodPost, "/v1/plugins/dev/rollout", bytes.NewReader(rolloutPayload))
	rolloutReq = withRole(rolloutReq, "super_admin")
	rolloutResp := httptest.NewRecorder()
	mux.ServeHTTP(rolloutResp, rolloutReq)
	if rolloutResp.Code != http.StatusOK {
		t.Fatalf("rollout status=%d body=%s", rolloutResp.Code, rolloutResp.Body.String())
	}
	var rolloutBody struct {
		Data devRolloutResponse `json:"data"`
	}
	if err := json.Unmarshal(rolloutResp.Body.Bytes(), &rolloutBody); err != nil {
		t.Fatalf("decode rollout response: %v", err)
	}
	if strings.TrimSpace(rolloutBody.Data.Task.TaskID) == "" {
		t.Fatalf("expected rollout task id in response")
	}

	rollbackPayload := []byte(`{"pluginId":"dev-edit"}`)
	rollbackReq := httptest.NewRequest(http.MethodPost, "/v1/plugins/dev/rollback", bytes.NewReader(rollbackPayload))
	rollbackReq = withRole(rollbackReq, "super_admin")
	rollbackResp := httptest.NewRecorder()
	mux.ServeHTTP(rollbackResp, rollbackReq)
	if rollbackResp.Code != http.StatusOK {
		t.Fatalf("rollback status=%d body=%s", rollbackResp.Code, rollbackResp.Body.String())
	}
	var rollbackBody struct {
		Data devRolloutResponse `json:"data"`
	}
	if err := json.Unmarshal(rollbackResp.Body.Bytes(), &rollbackBody); err != nil {
		t.Fatalf("decode rollback response: %v", err)
	}
	if strings.TrimSpace(rollbackBody.Data.Task.TaskID) == "" {
		t.Fatalf("expected rollback task id in response")
	}

	taskListReq := httptest.NewRequest(http.MethodGet, "/v1/plugins/dev/rollout-tasks?pluginId=dev-edit", nil)
	taskListReq = withRole(taskListReq, "super_admin")
	taskListResp := httptest.NewRecorder()
	mux.ServeHTTP(taskListResp, taskListReq)
	if taskListResp.Code != http.StatusOK {
		t.Fatalf("rollout task list status=%d body=%s", taskListResp.Code, taskListResp.Body.String())
	}
	var taskListBody struct {
		Data devListRolloutTasksResponse `json:"data"`
	}
	if err := json.Unmarshal(taskListResp.Body.Bytes(), &taskListBody); err != nil {
		t.Fatalf("decode rollout task list response: %v", err)
	}
	if len(taskListBody.Data.Tasks) < 2 {
		t.Fatalf("expected at least 2 rollout tasks, got %d", len(taskListBody.Data.Tasks))
	}

	taskID := strings.TrimSpace(rollbackBody.Data.Task.TaskID)
	taskDetailReq := httptest.NewRequest(http.MethodGet, "/v1/plugins/dev/rollout-tasks/"+taskID, nil)
	taskDetailReq = withRole(taskDetailReq, "super_admin")
	taskDetailResp := httptest.NewRecorder()
	mux.ServeHTTP(taskDetailResp, taskDetailReq)
	if taskDetailResp.Code != http.StatusOK {
		t.Fatalf("rollout task detail status=%d body=%s", taskDetailResp.Code, taskDetailResp.Body.String())
	}
	var taskDetailBody struct {
		Data devGetRolloutTaskResponse `json:"data"`
	}
	if err := json.Unmarshal(taskDetailResp.Body.Bytes(), &taskDetailBody); err != nil {
		t.Fatalf("decode rollout task detail response: %v", err)
	}
	if taskDetailBody.Data.Task.TaskID != taskID {
		t.Fatalf("unexpected rollout task detail: %+v", taskDetailBody.Data.Task)
	}

	taskLogsReq := httptest.NewRequest(http.MethodGet, "/v1/plugins/dev/rollout-tasks/"+taskID+"/logs", nil)
	taskLogsReq = withRole(taskLogsReq, "super_admin")
	taskLogsResp := httptest.NewRecorder()
	mux.ServeHTTP(taskLogsResp, taskLogsReq)
	if taskLogsResp.Code != http.StatusOK {
		t.Fatalf("rollout task logs status=%d body=%s", taskLogsResp.Code, taskLogsResp.Body.String())
	}
	var taskLogsBody struct {
		Data devRolloutTaskLogsResponse `json:"data"`
	}
	if err := json.Unmarshal(taskLogsResp.Body.Bytes(), &taskLogsBody); err != nil {
		t.Fatalf("decode rollout task logs response: %v", err)
	}
	if len(taskLogsBody.Data.Logs) == 0 {
		t.Fatalf("expected rollout task logs")
	}
}

func TestPluginHandlerDevPortalRolloutRequiresConfigUpdater(t *testing.T) {
	mux := http.NewServeMux()
	RegisterPluginRoutes(mux, &noConfigUpdaterPluginManager{items: map[string]plugin.Info{
		"dev-edit": {ID: "dev-edit", Name: "Dev Edit", Version: "0.1.1", State: plugin.StateEnabled, ConfigJSON: "{}"},
	}}, WithPluginDevPortal(true, "plugins", []string{"plugins"}))

	rolloutPayload := []byte(`{"pluginId":"dev-edit","percent":20}`)
	rolloutReq := httptest.NewRequest(http.MethodPost, "/v1/plugins/dev/rollout", bytes.NewReader(rolloutPayload))
	rolloutReq = withRole(rolloutReq, "super_admin")
	rolloutResp := httptest.NewRecorder()
	mux.ServeHTTP(rolloutResp, rolloutReq)

	if rolloutResp.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request when updater missing, got status=%d body=%s", rolloutResp.Code, rolloutResp.Body.String())
	}
	if !strings.Contains(rolloutResp.Body.String(), "runtime config updater is required") {
		t.Fatalf("unexpected rollout error body: %s", rolloutResp.Body.String())
	}
}

func TestPluginHandlerDevPortalReleaseOrderFlow(t *testing.T) {
	pluginsRoot := filepath.Join(t.TempDir(), "plugins")
	mgr := &fakePluginManager{items: map[string]plugin.Info{}}
	mux := http.NewServeMux()
	RegisterPluginRoutes(mux, mgr, WithPluginDevPortal(true, pluginsRoot, []string{pluginsRoot}))

	scaffoldPayload := []byte(`{"pluginsRoot":"` + filepath.ToSlash(pluginsRoot) + `","pluginId":"release-demo","pluginName":"Release Demo"}`)
	scaffoldReq := httptest.NewRequest(http.MethodPost, "/v1/plugins/dev/scaffold", bytes.NewReader(scaffoldPayload))
	scaffoldReq = withRole(scaffoldReq, "super_admin")
	scaffoldResp := httptest.NewRecorder()
	mux.ServeHTTP(scaffoldResp, scaffoldReq)
	if scaffoldResp.Code != http.StatusCreated {
		t.Fatalf("scaffold status=%d body=%s", scaffoldResp.Code, scaffoldResp.Body.String())
	}

	createPayload := []byte(`{"pluginsRoot":"` + filepath.ToSlash(pluginsRoot) + `","pluginId":"release-demo","releaseVersion":"1.2.0","changelog":"release note"}`)
	createReq := httptest.NewRequest(http.MethodPost, "/v1/plugins/dev/release-orders", bytes.NewReader(createPayload))
	createReq = withRole(createReq, "super_admin")
	createResp := httptest.NewRecorder()
	mux.ServeHTTP(createResp, createReq)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("create release order status=%d body=%s", createResp.Code, createResp.Body.String())
	}

	var createBody struct {
		Data devCreateReleaseOrderResponse `json:"data"`
	}
	if err := json.Unmarshal(createResp.Body.Bytes(), &createBody); err != nil {
		t.Fatalf("decode create release order response: %v", err)
	}
	if createBody.Data.Order.OrderID == "" {
		t.Fatalf("expected order id in create response")
	}
	if createBody.Data.Order.OrderStatus != devReleaseOrderStatusPending {
		t.Fatalf("unexpected order status after create: %+v", createBody.Data.Order)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/v1/plugins/dev/release-orders?pluginsRoot="+filepath.ToSlash(pluginsRoot)+"&pluginId=release-demo", nil)
	listReq = withRole(listReq, "super_admin")
	listResp := httptest.NewRecorder()
	mux.ServeHTTP(listResp, listReq)
	if listResp.Code != http.StatusOK {
		t.Fatalf("list release order status=%d body=%s", listResp.Code, listResp.Body.String())
	}

	var listBody struct {
		Data devListReleaseOrderResponse `json:"data"`
	}
	if err := json.Unmarshal(listResp.Body.Bytes(), &listBody); err != nil {
		t.Fatalf("decode list release order response: %v", err)
	}
	if len(listBody.Data.Orders) != 1 {
		t.Fatalf("expected 1 order in list, got %d", len(listBody.Data.Orders))
	}

	approvePayload := []byte(`{"pluginsRoot":"` + filepath.ToSlash(pluginsRoot) + `","pluginId":"release-demo","comment":"looks good"}`)
	approveReq := httptest.NewRequest(http.MethodPost, "/v1/plugins/dev/release-orders/"+createBody.Data.Order.OrderID+"/approve", bytes.NewReader(approvePayload))
	approveReq = withRole(approveReq, "super_admin")
	approveResp := httptest.NewRecorder()
	mux.ServeHTTP(approveResp, approveReq)
	if approveResp.Code != http.StatusOK {
		t.Fatalf("approve release order status=%d body=%s", approveResp.Code, approveResp.Body.String())
	}

	var approveBody struct {
		Data devReviewReleaseOrderResponse `json:"data"`
	}
	if err := json.Unmarshal(approveResp.Body.Bytes(), &approveBody); err != nil {
		t.Fatalf("decode approve release order response: %v", err)
	}
	if approveBody.Data.Order.OrderStatus != devReleaseOrderStatusApproved {
		t.Fatalf("unexpected status after approve: %+v", approveBody.Data.Order)
	}

	secondCreateReq := httptest.NewRequest(http.MethodPost, "/v1/plugins/dev/release-orders", bytes.NewReader([]byte(`{"pluginsRoot":"`+filepath.ToSlash(pluginsRoot)+`","pluginId":"release-demo","releaseVersion":"1.3.0"}`)))
	secondCreateReq = withRole(secondCreateReq, "super_admin")
	secondCreateResp := httptest.NewRecorder()
	mux.ServeHTTP(secondCreateResp, secondCreateReq)
	if secondCreateResp.Code != http.StatusCreated {
		t.Fatalf("create second release order status=%d body=%s", secondCreateResp.Code, secondCreateResp.Body.String())
	}
	var secondCreateBody struct {
		Data devCreateReleaseOrderResponse `json:"data"`
	}
	if err := json.Unmarshal(secondCreateResp.Body.Bytes(), &secondCreateBody); err != nil {
		t.Fatalf("decode second create response: %v", err)
	}

	rejectPayload := []byte(`{"pluginsRoot":"` + filepath.ToSlash(pluginsRoot) + `","pluginId":"release-demo","comment":"missing rollback checklist"}`)
	rejectReq := httptest.NewRequest(http.MethodPost, "/v1/plugins/dev/release-orders/"+secondCreateBody.Data.Order.OrderID+"/reject", bytes.NewReader(rejectPayload))
	rejectReq = withRole(rejectReq, "super_admin")
	rejectResp := httptest.NewRecorder()
	mux.ServeHTTP(rejectResp, rejectReq)
	if rejectResp.Code != http.StatusOK {
		t.Fatalf("reject release order status=%d body=%s", rejectResp.Code, rejectResp.Body.String())
	}
	var rejectBody struct {
		Data devReviewReleaseOrderResponse `json:"data"`
	}
	if err := json.Unmarshal(rejectResp.Body.Bytes(), &rejectBody); err != nil {
		t.Fatalf("decode reject response: %v", err)
	}
	if rejectBody.Data.Order.OrderStatus != devReleaseOrderStatusRejected {
		t.Fatalf("unexpected status after reject: %+v", rejectBody.Data.Order)
	}

	packagePayload := []byte(`{"pluginsRoot":"` + filepath.ToSlash(pluginsRoot) + `","pluginId":"release-demo"}`)
	packageReq := httptest.NewRequest(http.MethodPost, "/v1/plugins/dev/package", bytes.NewReader(packagePayload))
	packageReq = withRole(packageReq, "super_admin")
	packageResp := httptest.NewRecorder()
	mux.ServeHTTP(packageResp, packageReq)
	if packageResp.Code != http.StatusOK {
		t.Fatalf("package status=%d body=%s", packageResp.Code, packageResp.Body.String())
	}

	var packageBody struct {
		Data devPackageProjectResponse `json:"data"`
	}
	if err := json.Unmarshal(packageResp.Body.Bytes(), &packageBody); err != nil {
		t.Fatalf("decode package response: %v", err)
	}
	if strings.TrimSpace(packageBody.Data.ArtifactPath) == "" {
		t.Fatalf("expected artifact path in package response")
	}

	executePayload := []byte(`{"pluginsRoot":"` + filepath.ToSlash(pluginsRoot) + `","pluginId":"release-demo","targetEnv":"staging","artifactPath":"` + filepath.ToSlash(packageBody.Data.ArtifactPath) + `"}`)
	executeReq := httptest.NewRequest(http.MethodPost, "/v1/plugins/dev/release-orders/"+createBody.Data.Order.OrderID+"/execute", bytes.NewReader(executePayload))
	executeReq = withRole(executeReq, "super_admin")
	executeResp := httptest.NewRecorder()
	mux.ServeHTTP(executeResp, executeReq)
	if executeResp.Code != http.StatusAccepted {
		t.Fatalf("execute release order status=%d body=%s", executeResp.Code, executeResp.Body.String())
	}

	var executeBody struct {
		Data devExecuteReleaseOrderResponse `json:"data"`
	}
	if err := json.Unmarshal(executeResp.Body.Bytes(), &executeBody); err != nil {
		t.Fatalf("decode execute release response: %v", err)
	}
	taskID := strings.TrimSpace(executeBody.Data.Task.TaskID)
	if taskID == "" {
		t.Fatalf("expected task id in execute response")
	}

	var taskDetailBody struct {
		Data devGetReleaseTaskResponse `json:"data"`
	}
	for attempt := 0; attempt < 20; attempt++ {
		taskDetailReq := httptest.NewRequest(http.MethodGet, "/v1/plugins/dev/release-tasks/"+taskID+"?pluginsRoot="+filepath.ToSlash(pluginsRoot), nil)
		taskDetailReq = withRole(taskDetailReq, "super_admin")
		taskDetailResp := httptest.NewRecorder()
		mux.ServeHTTP(taskDetailResp, taskDetailReq)
		if taskDetailResp.Code != http.StatusOK {
			t.Fatalf("task detail status=%d body=%s", taskDetailResp.Code, taskDetailResp.Body.String())
		}
		if err := json.Unmarshal(taskDetailResp.Body.Bytes(), &taskDetailBody); err != nil {
			t.Fatalf("decode task detail response: %v", err)
		}
		if taskDetailBody.Data.Task.TaskStatus == devReleaseTaskStatusSuccess {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if taskDetailBody.Data.Task.TaskStatus != devReleaseTaskStatusSuccess {
		t.Fatalf("expected task success, got %+v", taskDetailBody.Data.Task)
	}

	taskListReq := httptest.NewRequest(http.MethodGet, "/v1/plugins/dev/release-tasks?pluginsRoot="+filepath.ToSlash(pluginsRoot)+"&pluginId=release-demo&orderId="+createBody.Data.Order.OrderID, nil)
	taskListReq = withRole(taskListReq, "super_admin")
	taskListResp := httptest.NewRecorder()
	mux.ServeHTTP(taskListResp, taskListReq)
	if taskListResp.Code != http.StatusOK {
		t.Fatalf("task list status=%d body=%s", taskListResp.Code, taskListResp.Body.String())
	}
	var taskListBody struct {
		Data devListReleaseTasksResponse `json:"data"`
	}
	if err := json.Unmarshal(taskListResp.Body.Bytes(), &taskListBody); err != nil {
		t.Fatalf("decode task list response: %v", err)
	}
	if len(taskListBody.Data.Tasks) == 0 {
		t.Fatalf("expected at least one task in list")
	}

	taskLogsReq := httptest.NewRequest(http.MethodGet, "/v1/plugins/dev/release-tasks/"+taskID+"/logs?pluginsRoot="+filepath.ToSlash(pluginsRoot), nil)
	taskLogsReq = withRole(taskLogsReq, "super_admin")
	taskLogsResp := httptest.NewRecorder()
	mux.ServeHTTP(taskLogsResp, taskLogsReq)
	if taskLogsResp.Code != http.StatusOK {
		t.Fatalf("task logs status=%d body=%s", taskLogsResp.Code, taskLogsResp.Body.String())
	}
	var taskLogsBody struct {
		Data devReleaseTaskLogsResponse `json:"data"`
	}
	if err := json.Unmarshal(taskLogsResp.Body.Bytes(), &taskLogsBody); err != nil {
		t.Fatalf("decode task logs response: %v", err)
	}
	if len(taskLogsBody.Data.Logs) == 0 {
		t.Fatalf("expected task logs")
	}
}

func TestPluginHandlerDevPortalReleaseTaskConflict(t *testing.T) {
	pluginsRoot := filepath.Join(t.TempDir(), "plugins")
	mgr := &fakePluginManager{items: map[string]plugin.Info{}}
	mux := http.NewServeMux()
	RegisterPluginRoutes(mux, mgr, WithPluginDevPortal(true, pluginsRoot, []string{pluginsRoot}))

	scaffoldPayload := []byte(`{"pluginsRoot":"` + filepath.ToSlash(pluginsRoot) + `","pluginId":"release-conflict","pluginName":"Release Conflict"}`)
	scaffoldReq := httptest.NewRequest(http.MethodPost, "/v1/plugins/dev/scaffold", bytes.NewReader(scaffoldPayload))
	scaffoldReq = withRole(scaffoldReq, "super_admin")
	scaffoldResp := httptest.NewRecorder()
	mux.ServeHTTP(scaffoldResp, scaffoldReq)
	if scaffoldResp.Code != http.StatusCreated {
		t.Fatalf("scaffold status=%d body=%s", scaffoldResp.Code, scaffoldResp.Body.String())
	}

	createPayload := []byte(`{"pluginsRoot":"` + filepath.ToSlash(pluginsRoot) + `","pluginId":"release-conflict","releaseVersion":"1.0.0"}`)
	createReq := httptest.NewRequest(http.MethodPost, "/v1/plugins/dev/release-orders", bytes.NewReader(createPayload))
	createReq = withRole(createReq, "super_admin")
	createResp := httptest.NewRecorder()
	mux.ServeHTTP(createResp, createReq)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("create release order status=%d body=%s", createResp.Code, createResp.Body.String())
	}
	var createBody struct {
		Data devCreateReleaseOrderResponse `json:"data"`
	}
	if err := json.Unmarshal(createResp.Body.Bytes(), &createBody); err != nil {
		t.Fatalf("decode create response: %v", err)
	}

	approvePayload := []byte(`{"pluginsRoot":"` + filepath.ToSlash(pluginsRoot) + `","pluginId":"release-conflict"}`)
	approveReq := httptest.NewRequest(http.MethodPost, "/v1/plugins/dev/release-orders/"+createBody.Data.Order.OrderID+"/approve", bytes.NewReader(approvePayload))
	approveReq = withRole(approveReq, "super_admin")
	approveResp := httptest.NewRecorder()
	mux.ServeHTTP(approveResp, approveReq)
	if approveResp.Code != http.StatusOK {
		t.Fatalf("approve release order status=%d body=%s", approveResp.Code, approveResp.Body.String())
	}

	conflictTask := devReleaseTaskRecord{
		TaskID:         "rt-running-conflict",
		OrderID:        createBody.Data.Order.OrderID,
		PluginID:       "release-conflict",
		ReleaseVersion: "1.0.0",
		TargetEnv:      "staging",
		ArtifactPath:   filepath.Join(pluginsRoot, "_dist", "release-conflict-1.0.0.zip"),
		TaskStatus:     devReleaseTaskStatusRunning,
		CreatedBy:      "tester",
		CreatedAt:      time.Now().UTC().Format(time.RFC3339),
	}
	if err := os.MkdirAll(filepath.Join(pluginsRoot, "_dist"), 0o755); err != nil {
		t.Fatalf("mkdir dist: %v", err)
	}
	if err := os.WriteFile(conflictTask.ArtifactPath, []byte("dummy"), 0o644); err != nil {
		t.Fatalf("write artifact: %v", err)
	}
	h := &PluginHandler{}
	if err := h.saveDevReleaseTasks(pluginsRoot, "release-conflict", []devReleaseTaskRecord{conflictTask}); err != nil {
		t.Fatalf("save conflict task: %v", err)
	}

	executePayload := []byte(`{"pluginsRoot":"` + filepath.ToSlash(pluginsRoot) + `","pluginId":"release-conflict","targetEnv":"staging","artifactPath":"` + filepath.ToSlash(conflictTask.ArtifactPath) + `"}`)
	executeReq := httptest.NewRequest(http.MethodPost, "/v1/plugins/dev/release-orders/"+createBody.Data.Order.OrderID+"/execute", bytes.NewReader(executePayload))
	executeReq = withRole(executeReq, "super_admin")
	executeResp := httptest.NewRecorder()
	mux.ServeHTTP(executeResp, executeReq)
	if executeResp.Code != http.StatusConflict {
		t.Fatalf("expected conflict status, got=%d body=%s", executeResp.Code, executeResp.Body.String())
	}
}

func TestPluginHandlerDebugAndLogs(t *testing.T) {
	t.Cleanup(func() {
		_ = os.RemoveAll("log")
	})

	if err := os.MkdirAll("log", 0o755); err != nil {
		t.Fatalf("mkdir log dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join("log", "demo.log"), []byte("line-1\nline-2\n"), 0o644); err != nil {
		t.Fatalf("write log file: %v", err)
	}

	mgr := &fakePluginManager{items: map[string]plugin.Info{
		"demo": {
			ID:          "demo",
			Name:        "Demo",
			Version:     "0.1.0",
			Description: "demo plugin",
			State:       plugin.StateEnabled,
			Permissions: []string{"menu.read"},
			Dependencies: []plugin.Dependency{
				{ID: "core", Version: "1.0.0"},
			},
			Source: "plugins/demo",
		},
	}, snapshots: map[string]plugin.RegistrySnapshot{
		"demo": {
			Routes: []plugin.RouteExtension{{Method: "GET", Path: "/v1/demo"}},
		},
	}}
	mux := http.NewServeMux()
	RegisterPluginRoutes(mux, mgr, WithPluginLogTarget("info", "log", "", true))

	debugReq := httptest.NewRequest(http.MethodGet, "/v1/plugins/demo/debug", nil)
	debugResp := httptest.NewRecorder()
	mux.ServeHTTP(debugResp, debugReq)
	if debugResp.Code != http.StatusOK {
		t.Fatalf("debug status=%d body=%s", debugResp.Code, debugResp.Body.String())
	}

	var debugBody struct {
		Data pluginDebugRecord `json:"data"`
	}
	if err := json.Unmarshal(debugResp.Body.Bytes(), &debugBody); err != nil {
		t.Fatalf("decode debug error: %v", err)
	}
	if debugBody.Data.ID != "demo" || debugBody.Data.State != string(plugin.StateEnabled) {
		t.Fatalf("unexpected debug data: %+v", debugBody.Data)
	}
	if debugBody.Data.Extensions == nil {
		t.Fatalf("expected extension snapshot in debug payload")
	}

	logsReq := httptest.NewRequest(http.MethodGet, "/v1/plugins/demo/logs", nil)
	logsResp := httptest.NewRecorder()
	mux.ServeHTTP(logsResp, logsReq)
	if logsResp.Code != http.StatusOK {
		t.Fatalf("logs status=%d body=%s", logsResp.Code, logsResp.Body.String())
	}

	var logsBody struct {
		Data pluginLogsRecord `json:"data"`
	}
	if err := json.Unmarshal(logsResp.Body.Bytes(), &logsBody); err != nil {
		t.Fatalf("decode logs error: %v", err)
	}
	if logsBody.Data.Content != "line-1\nline-2" {
		t.Fatalf("unexpected logs content: %q", logsBody.Data.Content)
	}
}

func TestPluginHandlerLogsFromUnifiedFile(t *testing.T) {
	tmp := t.TempDir()
	logFile := filepath.Join(tmp, "skoll.log")
	content := strings.Join([]string{
		"2026-01-01T00:00:00Z action=install result=ok plugin_id=demo message=installed",
		"2026-01-01T00:00:01Z action=enable result=ok plugin_id=other message=enabled",
		"2026-01-01T00:00:02Z action=disable result=ok plugin_id=demo message=disabled",
	}, "\n")
	if err := os.WriteFile(logFile, []byte(content+"\n"), 0o644); err != nil {
		t.Fatalf("write unified log file: %v", err)
	}

	mgr := &fakePluginManager{items: map[string]plugin.Info{
		"demo": {ID: "demo", Name: "Demo", Version: "0.1.0", State: plugin.StateEnabled},
	}}
	mux := http.NewServeMux()
	RegisterPluginRoutes(mux, mgr, WithPluginLogTarget("info", filepath.Dir(logFile), filepath.Base(logFile), false))

	logsReq := httptest.NewRequest(http.MethodGet, "/v1/plugins/demo/logs", nil)
	logsResp := httptest.NewRecorder()
	mux.ServeHTTP(logsResp, logsReq)
	if logsResp.Code != http.StatusOK {
		t.Fatalf("logs status=%d body=%s", logsResp.Code, logsResp.Body.String())
	}

	var logsBody struct {
		Data pluginLogsRecord `json:"data"`
	}
	if err := json.Unmarshal(logsResp.Body.Bytes(), &logsBody); err != nil {
		t.Fatalf("decode logs error: %v", err)
	}
	if !strings.Contains(logsBody.Data.Content, "plugin_id=demo") {
		t.Fatalf("expected filtered demo logs, got: %q", logsBody.Data.Content)
	}
	if strings.Contains(logsBody.Data.Content, "plugin_id=other") {
		t.Fatalf("unexpected logs from other plugin: %q", logsBody.Data.Content)
	}
}

func TestPluginHandlerNotFound(t *testing.T) {
	mux := http.NewServeMux()
	RegisterPluginRoutes(mux, &fakePluginManager{items: map[string]plugin.Info{}})

	for _, tc := range []struct {
		method string
		path   string
	}{
		{method: http.MethodPost, path: "/v1/plugins/missing/enable"},
		{method: http.MethodPost, path: "/v1/plugins/missing/disable"},
		{method: http.MethodDelete, path: "/v1/plugins/missing"},
		{method: http.MethodGet, path: "/v1/plugins/missing/debug"},
	} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		resp := httptest.NewRecorder()
		mux.ServeHTTP(resp, req)
		if resp.Code != http.StatusNotFound {
			t.Fatalf("%s %s status=%d body=%s", tc.method, tc.path, resp.Code, resp.Body.String())
		}
	}

	logsReq := httptest.NewRequest(http.MethodGet, "/v1/plugins/missing/logs", nil)
	logsResp := httptest.NewRecorder()
	mux.ServeHTTP(logsResp, logsReq)
	if logsResp.Code != http.StatusNotFound {
		t.Fatalf("logs not found status=%d body=%s", logsResp.Code, logsResp.Body.String())
	}
}

func TestPluginHandlerPageAndAssets(t *testing.T) {
	tmp := t.TempDir()
	pluginDir := filepath.Join(tmp, "demo-frontend")
	if err := os.MkdirAll(filepath.Join(pluginDir, "static"), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "static", "index.html"), []byte("<html><head></head><body><script src=\"./app.js\"></script></body></html>"), 0o644); err != nil {
		t.Fatalf("write index failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "static", "app.js"), []byte("console.log('ok')"), 0o644); err != nil {
		t.Fatalf("write app.js failed: %v", err)
	}

	mgr := &fakePluginManager{items: map[string]plugin.Info{
		"demo-frontend": {
			ID:            "demo-frontend",
			Name:          "Demo Frontend",
			Version:       "0.1.0",
			State:         plugin.StateEnabled,
			Source:        pluginDir,
			UIMode:        plugin.UIModeFrontendOnly,
			FrontendEntry: "/plugins/demo-frontend",
		},
	}}

	mux := http.NewServeMux()
	RegisterPluginRoutes(mux, mgr)

	pageReq := httptest.NewRequest(http.MethodGet, "/v1/plugins/demo-frontend/page", nil)
	pageResp := httptest.NewRecorder()
	mux.ServeHTTP(pageResp, pageReq)
	if pageResp.Code != http.StatusOK {
		t.Fatalf("page status=%d body=%s", pageResp.Code, pageResp.Body.String())
	}
	if !strings.Contains(pageResp.Body.String(), "/v1/plugins/demo-frontend/assets/") {
		t.Fatalf("expected injected base href in page body: %s", pageResp.Body.String())
	}

	assetReq := httptest.NewRequest(http.MethodGet, "/v1/plugins/demo-frontend/assets/app.js", nil)
	assetResp := httptest.NewRecorder()
	mux.ServeHTTP(assetResp, assetReq)
	if assetResp.Code != http.StatusOK {
		t.Fatalf("asset status=%d body=%s", assetResp.Code, assetResp.Body.String())
	}
	if !strings.Contains(assetResp.Body.String(), "console.log('ok')") {
		t.Fatalf("unexpected asset body: %s", assetResp.Body.String())
	}
}

func TestPluginHandlerPageDemoBackendMonolith(t *testing.T) {
	tmp := t.TempDir()
	pluginDir := filepath.Join(tmp, "demo-backend")
	if err := os.MkdirAll(filepath.Join(pluginDir, "static"), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pluginDir, "static", "index.html"), []byte("<html><head></head><body>Demo Backend Plugin</body></html>"), 0o644); err != nil {
		t.Fatalf("write index failed: %v", err)
	}

	mgr := &fakePluginManager{items: map[string]plugin.Info{
		"demo-backend": {ID: "demo-backend", Name: "Demo Backend", Version: "0.1.0", State: plugin.StateEnabled, Source: pluginDir, UIMode: plugin.UIModeMonolith},
	}}
	mux := http.NewServeMux()
	RegisterPluginRoutes(mux, mgr)

	req := httptest.NewRequest(http.MethodGet, "/v1/plugins/demo-backend/page", nil)
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200 for demo-backend page, got %d body=%s", resp.Code, resp.Body.String())
	}
	if !strings.Contains(resp.Body.String(), "Demo Backend Plugin") {
		t.Fatalf("expected demo-backend html content, got: %s", resp.Body.String())
	}
}

func TestFakePluginManagerImplementsManager(t *testing.T) {
	var _ PluginManager = (*fakePluginManager)(nil)
	var _ PluginExtensionSnapshotProvider = (*fakePluginManager)(nil)
	var _ PluginConfigUpdater = (*fakePluginManager)(nil)
	t.Log("compile assertions passed")
}
