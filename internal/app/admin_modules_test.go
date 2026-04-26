package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tinboxw/skoll/internal/module/apiregistry"
	"github.com/tinboxw/skoll/internal/module/audit"
	"github.com/tinboxw/skoll/internal/module/config"
	"github.com/tinboxw/skoll/internal/module/dictionary"
	"github.com/tinboxw/skoll/internal/module/fileservice"
	"github.com/tinboxw/skoll/internal/module/jobscheduler"
	"github.com/tinboxw/skoll/internal/module/menu"
	"github.com/tinboxw/skoll/internal/module/modgenerator"
	"github.com/tinboxw/skoll/internal/module/pluginmgr"
	"github.com/tinboxw/skoll/internal/module/rbac"
	"github.com/tinboxw/skoll/internal/module/role"
	"github.com/tinboxw/skoll/internal/module/user"
)

type memoryFileBackend struct {
	items map[string][]byte
}

func (b *memoryFileBackend) Save(_ string, content []byte) (string, error) {
	if b.items == nil {
		b.items = make(map[string][]byte)
	}
	key := fmt.Sprintf("f-%d", len(b.items)+1)
	b.items[key] = append([]byte(nil), content...)
	return key, nil
}

func (b *memoryFileBackend) Open(storageKey string) ([]byte, error) {
	out, ok := b.items[storageKey]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return append([]byte(nil), out...), nil
}

func testAdminModuleServices() AdminModuleServices {
	return AdminModuleServices{
		Users:        user.NewService(),
		Roles:        role.NewService(),
		Menus:        menu.NewService(),
		Audit:        audit.NewService(),
		Configs:      config.NewService(),
		Dictionaries: dictionary.NewService(),
		Files:        fileservice.NewService(&memoryFileBackend{}),
		Jobs:         jobscheduler.NewService(),
		Generator:    modgenerator.NewService(),
		Plugins:      pluginmgr.NewService(),
		RBAC:         rbac.NewService(),
		APIs:         apiregistry.NewService(),
	}
}

func TestAdminUserRoutes_CreateListGet(t *testing.T) {
	srv := New(":0", "test-version")
	srv.MountAdminModuleRoutes(testAdminModuleServices(), nil)

	createReq := httptest.NewRequest(http.MethodPost, "/admin/v1/users", strings.NewReader(`{"name":"alice","email":"alice@example.com"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(createRR, createReq)
	if createRR.Code != http.StatusCreated {
		t.Fatalf("expected create status 201, got %d", createRR.Code)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/admin/v1/users", nil)
	listRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(listRR, listReq)
	if listRR.Code != http.StatusOK {
		t.Fatalf("expected list status 200, got %d", listRR.Code)
	}

	var users []map[string]any
	if err := json.Unmarshal(listRR.Body.Bytes(), &users); err != nil {
		t.Fatalf("unmarshal list response failed: %v", err)
	}
	if len(users) != 1 {
		t.Fatalf("expected 1 user, got %d", len(users))
	}

	getReq := httptest.NewRequest(http.MethodGet, "/admin/v1/users/1", nil)
	getRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(getRR, getReq)
	if getRR.Code != http.StatusOK {
		t.Fatalf("expected get status 200, got %d", getRR.Code)
	}
	if !strings.Contains(getRR.Body.String(), `"Email":"alice@example.com"`) {
		t.Fatalf("expected user email in response, got %s", getRR.Body.String())
	}
}

func TestAdminRoleAndMenuRoutes(t *testing.T) {
	srv := New(":0", "test-version")
	srv.MountAdminModuleRoutes(testAdminModuleServices(), nil)

	roleReq := httptest.NewRequest(http.MethodPost, "/admin/v1/roles", strings.NewReader(`{"name":"ops","permissions":["user.read","menu.read"]}`))
	roleReq.Header.Set("Content-Type", "application/json")
	roleRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(roleRR, roleReq)
	if roleRR.Code != http.StatusCreated {
		t.Fatalf("expected role create status 201, got %d", roleRR.Code)
	}

	menuReq := httptest.NewRequest(http.MethodPost, "/admin/v1/menus", strings.NewReader(`{"title":"Dashboard","path":"/dashboard","order":1}`))
	menuReq.Header.Set("Content-Type", "application/json")
	menuRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(menuRR, menuReq)
	if menuRR.Code != http.StatusCreated {
		t.Fatalf("expected menu create status 201, got %d", menuRR.Code)
	}

	roleGetReq := httptest.NewRequest(http.MethodGet, "/admin/v1/roles/1", nil)
	roleGetRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(roleGetRR, roleGetReq)
	if roleGetRR.Code != http.StatusOK {
		t.Fatalf("expected role get status 200, got %d", roleGetRR.Code)
	}

	menuListReq := httptest.NewRequest(http.MethodGet, "/admin/v1/menus", nil)
	menuListRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(menuListRR, menuListReq)
	if menuListRR.Code != http.StatusOK {
		t.Fatalf("expected menu list status 200, got %d", menuListRR.Code)
	}
}

func TestAdminAuditRoutes_RecentWithLimit(t *testing.T) {
	srv := New(":0", "test-version")
	srv.MountAdminModuleRoutes(testAdminModuleServices(), nil)

	for i := 0; i < 3; i++ {
		payload := []byte(`{"actor":"system","action":"create","target":"user"}`)
		req := httptest.NewRequest(http.MethodPost, "/admin/v1/audit-logs", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		srv.httpServer.Handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusCreated {
			t.Fatalf("expected audit append status 201, got %d", rr.Code)
		}
	}

	recentReq := httptest.NewRequest(http.MethodGet, "/admin/v1/audit-logs?limit=2", nil)
	recentRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(recentRR, recentReq)
	if recentRR.Code != http.StatusOK {
		t.Fatalf("expected recent status 200, got %d", recentRR.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(recentRR.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal recent response failed: %v", err)
	}
	items, ok := payload["items"].([]any)
	if !ok {
		t.Fatalf("expected items array payload, got %v", payload)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 recent records, got %d", len(items))
	}

	filteredReq := httptest.NewRequest(http.MethodGet, "/admin/v1/audit-logs?page=1&size=10&actor=system&action=create&q=user", nil)
	filteredRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(filteredRR, filteredReq)
	if filteredRR.Code != http.StatusOK {
		t.Fatalf("expected filtered status 200, got %d", filteredRR.Code)
	}
	if !strings.Contains(filteredRR.Body.String(), `"total":3`) {
		t.Fatalf("expected filtered total field in response, got %s", filteredRR.Body.String())
	}
}

func TestAdminRoutes_AuthWrapper(t *testing.T) {
	srv := New(":0", "test-version")
	wrapper := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-Admin-Token") != "secret" {
				respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
	srv.MountAdminModuleRoutes(testAdminModuleServices(), wrapper)

	unauthReq := httptest.NewRequest(http.MethodGet, "/admin/v1/users", nil)
	unauthRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(unauthRR, unauthReq)
	if unauthRR.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized status 401, got %d", unauthRR.Code)
	}

	authReq := httptest.NewRequest(http.MethodGet, "/admin/v1/users", nil)
	authReq.Header.Set("X-Admin-Token", "secret")
	authRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(authRR, authReq)
	if authRR.Code != http.StatusOK {
		t.Fatalf("expected authorized status 200, got %d", authRR.Code)
	}
}

func BenchmarkAdminUsersListEndpoint(b *testing.B) {
	srv := New(":0", "bench")
	users := user.NewService()
	for i := 0; i < 100; i++ {
		users.Create("user", "user@example.com")
	}
	services := testAdminModuleServices()
	services.Users = users
	srv.MountAdminModuleRoutes(services, nil)

	req := httptest.NewRequest(http.MethodGet, "/admin/v1/users", nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		srv.httpServer.Handler.ServeHTTP(rr, req)
	}
}

func TestRoleBindingRoutes_MenuAndAPI(t *testing.T) {
	srv := New(":0", "test-version")
	srv.MountAdminModuleRoutes(testAdminModuleServices(), nil)

	createRoleReq := httptest.NewRequest(http.MethodPost, "/admin/v1/roles", strings.NewReader(`{"name":"ops","permissions":["user.read"]}`))
	createRoleReq.Header.Set("Content-Type", "application/json")
	createRoleRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(createRoleRR, createRoleReq)
	if createRoleRR.Code != http.StatusCreated {
		t.Fatalf("expected role create status 201, got %d", createRoleRR.Code)
	}

	createMenuReq1 := httptest.NewRequest(http.MethodPost, "/admin/v1/menus", strings.NewReader(`{"title":"Dashboard","path":"/dashboard","order":1}`))
	createMenuReq1.Header.Set("Content-Type", "application/json")
	createMenuRR1 := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(createMenuRR1, createMenuReq1)
	if createMenuRR1.Code != http.StatusCreated {
		t.Fatalf("expected menu create status 201, got %d", createMenuRR1.Code)
	}

	createMenuReq2 := httptest.NewRequest(http.MethodPost, "/admin/v1/menus", strings.NewReader(`{"title":"System","path":"/system","order":2}`))
	createMenuReq2.Header.Set("Content-Type", "application/json")
	createMenuRR2 := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(createMenuRR2, createMenuReq2)
	if createMenuRR2.Code != http.StatusCreated {
		t.Fatalf("expected menu create status 201, got %d", createMenuRR2.Code)
	}

	setMenusReq := httptest.NewRequest(http.MethodPut, "/admin/v1/roles/1/menus", strings.NewReader(`{"menu_ids":[2,1,1]}`))
	setMenusReq.Header.Set("Content-Type", "application/json")
	setMenusRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(setMenusRR, setMenusReq)
	if setMenusRR.Code != http.StatusOK {
		t.Fatalf("expected set role menus status 200, got %d", setMenusRR.Code)
	}

	getMenusReq := httptest.NewRequest(http.MethodGet, "/admin/v1/roles/1/menus", nil)
	getMenusRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(getMenusRR, getMenusReq)
	if getMenusRR.Code != http.StatusOK {
		t.Fatalf("expected get role menus status 200, got %d", getMenusRR.Code)
	}
	if !strings.Contains(getMenusRR.Body.String(), `"menu_ids":[1,2]`) {
		t.Fatalf("expected sorted deduplicated menu ids, got %s", getMenusRR.Body.String())
	}

	setAPIsReq := httptest.NewRequest(http.MethodPut, "/admin/v1/roles/1/apis", strings.NewReader(`{"apis":["POST:/admin/v1/users","GET:/admin/v1/users","GET:/admin/v1/users"]}`))
	setAPIsReq.Header.Set("Content-Type", "application/json")
	setAPIsRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(setAPIsRR, setAPIsReq)
	if setAPIsRR.Code != http.StatusOK {
		t.Fatalf("expected set role apis status 200, got %d", setAPIsRR.Code)
	}

	getAPIsReq := httptest.NewRequest(http.MethodGet, "/admin/v1/roles/1/apis", nil)
	getAPIsRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(getAPIsRR, getAPIsReq)
	if getAPIsRR.Code != http.StatusOK {
		t.Fatalf("expected get role apis status 200, got %d", getAPIsRR.Code)
	}
	if !strings.Contains(getAPIsRR.Body.String(), `"apis":["GET:/admin/v1/users","POST:/admin/v1/users"]`) {
		t.Fatalf("expected sorted deduplicated api bindings, got %s", getAPIsRR.Body.String())
	}

	listRegistryReq := httptest.NewRequest(http.MethodGet, "/admin/v1/apis", nil)
	listRegistryRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(listRegistryRR, listRegistryReq)
	if listRegistryRR.Code != http.StatusOK {
		t.Fatalf("expected list api registry status 200, got %d", listRegistryRR.Code)
	}
	if !strings.Contains(listRegistryRR.Body.String(), `"GET:/admin/v1/roles/{id}/apis"`) {
		t.Fatalf("expected registered apis in response, got %s", listRegistryRR.Body.String())
	}

	invalidAPIReq := httptest.NewRequest(http.MethodPut, "/admin/v1/roles/1/apis", strings.NewReader(`{"apis":["DELETE:/admin/v1/users"]}`))
	invalidAPIReq.Header.Set("Content-Type", "application/json")
	invalidAPIRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(invalidAPIRR, invalidAPIReq)
	if invalidAPIRR.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid api binding status 400, got %d", invalidAPIRR.Code)
	}
}

func TestConfigAndDictionaryRoutes(t *testing.T) {
	srv := New(":0", "test-version")
	srv.MountAdminModuleRoutes(testAdminModuleServices(), nil)

	configReq := httptest.NewRequest(http.MethodPost, "/admin/v1/configs", strings.NewReader(`{"key":"system.theme","value":"aurora","description":"ui theme"}`))
	configReq.Header.Set("Content-Type", "application/json")
	configRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(configRR, configReq)
	if configRR.Code != http.StatusCreated {
		t.Fatalf("expected config create status 201, got %d", configRR.Code)
	}

	configGetReq := httptest.NewRequest(http.MethodGet, "/admin/v1/configs/system.theme", nil)
	configGetRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(configGetRR, configGetReq)
	if configGetRR.Code != http.StatusOK {
		t.Fatalf("expected config get status 200, got %d", configGetRR.Code)
	}

	dictReq := httptest.NewRequest(http.MethodPost, "/admin/v1/dictionaries", strings.NewReader(`{"type":"status","label":"Enabled","value":"1","sort":10}`))
	dictReq.Header.Set("Content-Type", "application/json")
	dictRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(dictRR, dictReq)
	if dictRR.Code != http.StatusCreated {
		t.Fatalf("expected dictionary create status 201, got %d", dictRR.Code)
	}

	dictListReq := httptest.NewRequest(http.MethodGet, "/admin/v1/dictionaries?type=status", nil)
	dictListRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(dictListRR, dictListReq)
	if dictListRR.Code != http.StatusOK {
		t.Fatalf("expected dictionary list status 200, got %d", dictListRR.Code)
	}
	if !strings.Contains(dictListRR.Body.String(), `"Type":"status"`) {
		t.Fatalf("expected filtered dictionary items, got %s", dictListRR.Body.String())
	}

	dictGetReq := httptest.NewRequest(http.MethodGet, "/admin/v1/dictionaries/1", nil)
	dictGetRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(dictGetRR, dictGetReq)
	if dictGetRR.Code != http.StatusOK {
		t.Fatalf("expected dictionary get status 200, got %d", dictGetRR.Code)
	}
}

func TestFileRoutes_UploadListGetDownload(t *testing.T) {
	srv := New(":0", "test-version")
	srv.MountAdminModuleRoutes(testAdminModuleServices(), nil)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "hello.txt")
	if err != nil {
		t.Fatalf("create form file failed: %v", err)
	}
	if _, err := part.Write([]byte("hello skoll")); err != nil {
		t.Fatalf("write multipart content failed: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer failed: %v", err)
	}

	uploadReq := httptest.NewRequest(http.MethodPost, "/admin/v1/files", &body)
	uploadReq.Header.Set("Content-Type", writer.FormDataContentType())
	uploadRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(uploadRR, uploadReq)
	if uploadRR.Code != http.StatusCreated {
		t.Fatalf("expected upload status 201, got %d", uploadRR.Code)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/admin/v1/files", nil)
	listRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(listRR, listReq)
	if listRR.Code != http.StatusOK {
		t.Fatalf("expected file list status 200, got %d", listRR.Code)
	}
	if !strings.Contains(listRR.Body.String(), `"Name":"hello.txt"`) {
		t.Fatalf("expected uploaded file in list, got %s", listRR.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/admin/v1/files/1", nil)
	getRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(getRR, getReq)
	if getRR.Code != http.StatusOK {
		t.Fatalf("expected file get status 200, got %d", getRR.Code)
	}

	downloadReq := httptest.NewRequest(http.MethodGet, "/admin/v1/files/1/download", nil)
	downloadRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(downloadRR, downloadReq)
	if downloadRR.Code != http.StatusOK {
		t.Fatalf("expected file download status 200, got %d", downloadRR.Code)
	}
	if downloadRR.Body.String() != "hello skoll" {
		t.Fatalf("unexpected downloaded body: %s", downloadRR.Body.String())
	}
}

func TestJobRoutes_CreateRunHistory(t *testing.T) {
	srv := New(":0", "test-version")
	srv.MountAdminModuleRoutes(testAdminModuleServices(), nil)

	createReq := httptest.NewRequest(http.MethodPost, "/admin/v1/jobs", strings.NewReader(`{"name":"daily-sync","schedule":"0 0 * * *"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(createRR, createReq)
	if createRR.Code != http.StatusCreated {
		t.Fatalf("expected create job status 201, got %d", createRR.Code)
	}

	runReq := httptest.NewRequest(http.MethodPost, "/admin/v1/jobs/1/run", nil)
	runRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(runRR, runReq)
	if runRR.Code != http.StatusOK {
		t.Fatalf("expected run job status 200, got %d", runRR.Code)
	}

	historyReq := httptest.NewRequest(http.MethodGet, "/admin/v1/jobs/1/history?limit=10", nil)
	historyRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(historyRR, historyReq)
	if historyRR.Code != http.StatusOK {
		t.Fatalf("expected job history status 200, got %d", historyRR.Code)
	}
	if !strings.Contains(historyRR.Body.String(), `"Status":"success"`) {
		t.Fatalf("expected successful run in history, got %s", historyRR.Body.String())
	}
}

func TestGeneratorRoutes_ModuleScaffoldPreview(t *testing.T) {
	srv := New(":0", "test-version")
	srv.MountAdminModuleRoutes(testAdminModuleServices(), nil)

	req := httptest.NewRequest(http.MethodPost, "/admin/v1/generator/modules", strings.NewReader(`{"module":"billing"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected generator status 200, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), `"module":"billing"`) {
		t.Fatalf("expected module name in generator result, got %s", rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"internal/module/billing/service.go"`) {
		t.Fatalf("expected generated service artifact path, got %s", rr.Body.String())
	}
}

func TestPluginRoutes_InstallAndToggle(t *testing.T) {
	srv := New(":0", "test-version")
	srv.MountAdminModuleRoutes(testAdminModuleServices(), nil)

	installReq := httptest.NewRequest(http.MethodPost, "/admin/v1/plugins/manifests", strings.NewReader(`{"name":"audit-ext","version":"1.0.0","hooks":["on_boot"]}`))
	installReq.Header.Set("Content-Type", "application/json")
	installRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(installRR, installReq)
	if installRR.Code != http.StatusCreated {
		t.Fatalf("expected install status 201, got %d", installRR.Code)
	}

	disableReq := httptest.NewRequest(http.MethodPost, "/admin/v1/plugins/audit-ext/disable", nil)
	disableRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(disableRR, disableReq)
	if disableRR.Code != http.StatusOK {
		t.Fatalf("expected disable status 200, got %d", disableRR.Code)
	}
	if !strings.Contains(disableRR.Body.String(), `"enabled":false`) {
		t.Fatalf("expected plugin disabled, got %s", disableRR.Body.String())
	}

	enableReq := httptest.NewRequest(http.MethodPost, "/admin/v1/plugins/audit-ext/enable", nil)
	enableRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(enableRR, enableReq)
	if enableRR.Code != http.StatusOK {
		t.Fatalf("expected enable status 200, got %d", enableRR.Code)
	}
	if !strings.Contains(enableRR.Body.String(), `"enabled":true`) {
		t.Fatalf("expected plugin enabled, got %s", enableRR.Body.String())
	}

	packageReq := httptest.NewRequest(http.MethodPost, "/admin/v1/plugins/packages/install", strings.NewReader(`{"name":"audit-ext","version":"1.1.0","package_url":"https://example.com/plugins/audit-ext-1.1.0.tgz","package_hash":"sha256:abcd","hooks":["on_boot"]}`))
	packageReq.Header.Set("Content-Type", "application/json")
	packageRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(packageRR, packageReq)
	if packageRR.Code != http.StatusCreated {
		t.Fatalf("expected package install status 201, got %d", packageRR.Code)
	}
	if !strings.Contains(packageRR.Body.String(), `"package_url":"https://example.com/plugins/audit-ext-1.1.0.tgz"`) {
		t.Fatalf("expected package metadata in response, got %s", packageRR.Body.String())
	}

	versionReq := httptest.NewRequest(http.MethodPost, "/admin/v1/plugins/audit-ext/version-check", strings.NewReader(`{"latest_version":"1.2.0"}`))
	versionReq.Header.Set("Content-Type", "application/json")
	versionRR := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(versionRR, versionReq)
	if versionRR.Code != http.StatusOK {
		t.Fatalf("expected version check status 200, got %d", versionRR.Code)
	}
	if !strings.Contains(versionRR.Body.String(), `"update_available":true`) {
		t.Fatalf("expected update availability in response, got %s", versionRR.Body.String())
	}
}
