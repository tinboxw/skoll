package permission

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	domainpermission "github.com/tinboxw/skoll/internal/domain/permission"
	permissionsvc "github.com/tinboxw/skoll/internal/service/permission"
)

func TestPermissionHandlerListAndDetail(t *testing.T) {
	svc := &fakeService{
		items: []domainpermission.PermissionResource{
			mustResource(t, "system:user:list", domainpermission.ResourceTypeAPI, "system", true),
		},
	}
	mux := http.NewServeMux()
	RegisterPermissionRoutes(mux, svc)

	req := httptest.NewRequest(http.MethodGet, "/v1/permissions?type=api&source=system&enabled=true&offset=0&limit=20", nil)
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", resp.Code, resp.Body.String())
	}
	if svc.listInput.Type != domainpermission.ResourceTypeAPI || svc.listInput.Source != "system" || svc.listInput.Enabled == nil || !*svc.listInput.Enabled {
		t.Fatalf("list input = %+v", svc.listInput)
	}
	body := decodeBody(t, resp)
	data := body["data"].(map[string]any)
	items := data["items"].([]any)
	if len(items) != 1 || items[0].(map[string]any)["key"] != "system:user:list" {
		t.Fatalf("unexpected items: %+v", items)
	}

	detailReq := httptest.NewRequest(http.MethodGet, "/v1/permissions/system:user:list", nil)
	detailResp := httptest.NewRecorder()
	mux.ServeHTTP(detailResp, detailReq)
	if detailResp.Code != http.StatusOK {
		t.Fatalf("detail status = %d body=%s", detailResp.Code, detailResp.Body.String())
	}

	missingReq := httptest.NewRequest(http.MethodGet, "/v1/permissions/missing", nil)
	missingResp := httptest.NewRecorder()
	mux.ServeHTTP(missingResp, missingReq)
	if missingResp.Code != http.StatusNotFound {
		t.Fatalf("missing status = %d body=%s", missingResp.Code, missingResp.Body.String())
	}
	missingBody := decodeBody(t, missingResp)
	if missingBody["code"] != "not_found" {
		t.Fatalf("missing body = %+v", missingBody)
	}
}

func TestPermissionHandlerEnableDisable(t *testing.T) {
	svc := &fakeService{}
	mux := http.NewServeMux()
	RegisterPermissionRoutes(mux, svc)

	enableReq := httptest.NewRequest(http.MethodPost, "/v1/permissions/system:user:list/enable", nil)
	enableResp := httptest.NewRecorder()
	mux.ServeHTTP(enableResp, enableReq)
	if enableResp.Code != http.StatusOK || svc.enabledKey != "system:user:list" {
		t.Fatalf("enable status=%d key=%q body=%s", enableResp.Code, svc.enabledKey, enableResp.Body.String())
	}

	disableReq := httptest.NewRequest(http.MethodPost, "/v1/permissions/system:user:list/disable", nil)
	disableResp := httptest.NewRecorder()
	mux.ServeHTTP(disableResp, disableReq)
	if disableResp.Code != http.StatusOK || svc.disabledKey != "system:user:list" {
		t.Fatalf("disable status=%d key=%q body=%s", disableResp.Code, svc.disabledKey, disableResp.Body.String())
	}
}

func TestPermissionHandlerDiff(t *testing.T) {
	svc := &fakeService{
		diff: &permissionsvc.DiffResult{
			Added: []domainpermission.PermissionResource{
				mustResource(t, "plugin.demo:report:list", domainpermission.ResourceTypeAPI, "plugin.demo", true),
			},
		},
	}
	mux := http.NewServeMux()
	RegisterPermissionRoutes(mux, svc)

	raw := []byte(`{"source":"plugin.demo","desired":[{"key":"plugin.demo:report:list","type":"api","module":"report","name":"Reports"}]}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/permissions/diff", bytes.NewReader(raw))
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
	if svc.diffInput.Source != "plugin.demo" || len(svc.diffInput.Desired) != 1 {
		t.Fatalf("diff input = %+v", svc.diffInput)
	}
	body := decodeBody(t, resp)
	added := body["data"].(map[string]any)["added"].([]any)
	if len(added) != 1 {
		t.Fatalf("added = %+v", added)
	}
}

type fakeService struct {
	items       []domainpermission.PermissionResource
	listInput   permissionsvc.ListResourcesInput
	enabledKey  string
	disabledKey string
	diffInput   permissionsvc.DiffResourcesInput
	diff        *permissionsvc.DiffResult
}

func (s *fakeService) RegisterResource(context.Context, permissionsvc.RegisterResourceInput) (*domainpermission.PermissionResource, error) {
	return nil, nil
}

func (s *fakeService) ListResources(_ context.Context, in permissionsvc.ListResourcesInput) ([]domainpermission.PermissionResource, error) {
	s.listInput = in
	return s.items, nil
}

func (s *fakeService) GetResource(_ context.Context, key string) (*domainpermission.PermissionResource, error) {
	for _, item := range s.items {
		if item.Key() == key {
			result := item
			return &result, nil
		}
	}
	return nil, nil
}

func (s *fakeService) EnableResource(_ context.Context, key string) error {
	s.enabledKey = key
	return nil
}

func (s *fakeService) DisableResource(_ context.Context, key string) error {
	s.disabledKey = key
	return nil
}

func (s *fakeService) DiffResources(_ context.Context, in permissionsvc.DiffResourcesInput) (*permissionsvc.DiffResult, error) {
	s.diffInput = in
	if s.diff != nil {
		return s.diff, nil
	}
	return &permissionsvc.DiffResult{}, nil
}

func mustResource(t *testing.T, key string, resourceType domainpermission.ResourceType, source string, enabled bool) domainpermission.PermissionResource {
	t.Helper()
	resource, err := domainpermission.NewResource(domainpermission.ResourceIdentity{
		Key:    key,
		Type:   resourceType,
		Module: "system",
		Source: source,
	}, "Permission")
	if err != nil {
		t.Fatalf("NewResource() error = %v", err)
	}
	resource.Enabled = enabled
	return resource
}

func decodeBody(t *testing.T, resp *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v; %s", err, resp.Body.String())
	}
	return body
}
