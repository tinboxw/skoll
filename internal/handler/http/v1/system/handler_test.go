package system

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
	domainsystem "github.com/tinboxw/skoll/internal/domain/system"
	systemsvc "github.com/tinboxw/skoll/internal/service/system"
)

type fakeSystemService struct {
	items map[string]*domainsystem.Setting
}

func newFakeSystemService() *fakeSystemService {
	return &fakeSystemService{items: map[string]*domainsystem.Setting{}}
}

func (f *fakeSystemService) Upsert(_ context.Context, in systemsvc.UpsertInput) (*domainsystem.Setting, error) {
	now := time.Date(2026, time.June, 6, 12, 0, 0, 0, time.UTC)
	existing := f.items[in.Key]
	if existing != nil {
		existing.UpdateValue(in.Value, now)
		existing.Encrypted = in.Encrypted
		return existing, nil
	}
	item, err := domainsystem.NewSetting(shared.ID("setting-"+in.Key), in.Key, in.Value, in.Encrypted, now)
	if err != nil {
		return nil, err
	}
	f.items[in.Key] = item
	return item, nil
}

func (f *fakeSystemService) GetByKey(_ context.Context, key string) (*domainsystem.Setting, error) {
	return f.items[key], nil
}

func (f *fakeSystemService) List(_ context.Context, _ systemsvc.ListInput) ([]*domainsystem.Setting, error) {
	return nil, nil
}

func (f *fakeSystemService) Reset(_ context.Context) (int, error) {
	count := len(f.items)
	f.items = map[string]*domainsystem.Setting{}
	return count, nil
}

func TestSystemMenusDefaultAndOverride(t *testing.T) {
	svc := newFakeSystemService()
	mux := http.NewServeMux()
	RegisterSystemRoutes(mux, svc, nil)

	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/v1/system/menus", nil))
	if resp.Code != http.StatusOK {
		t.Fatalf("default menus status=%d body=%s", resp.Code, resp.Body.String())
	}
	var defaultBody struct {
		Data struct {
			Items      []MenuItem `json:"items"`
			Customized bool       `json:"customized"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &defaultBody); err != nil {
		t.Fatalf("decode default response: %v", err)
	}
	if defaultBody.Data.Customized {
		t.Fatalf("default menus should not be customized")
	}
	if len(defaultBody.Data.Items) == 0 {
		t.Fatalf("expected default menus")
	}

	payload := []byte(`{"items":[{"id":"custom","label":"Custom","path":"/skoll/custom","icon":"plugins","order":10,"visible":true,"requiredPermissions":["plugin.manage"]},{"id":"","label":"Bad","path":"bad"}]}`)
	putResp := httptest.NewRecorder()
	mux.ServeHTTP(putResp, httptest.NewRequest(http.MethodPut, "/v1/system/menus", bytes.NewReader(payload)))
	if putResp.Code != http.StatusOK {
		t.Fatalf("put menus status=%d body=%s", putResp.Code, putResp.Body.String())
	}
	var putBody struct {
		Data struct {
			Items      []MenuItem `json:"items"`
			Customized bool       `json:"customized"`
		} `json:"data"`
	}
	if err := json.Unmarshal(putResp.Body.Bytes(), &putBody); err != nil {
		t.Fatalf("decode put response: %v", err)
	}
	if !putBody.Data.Customized {
		t.Fatalf("put menus should be customized")
	}
	if len(putBody.Data.Items) != 1 || putBody.Data.Items[0].ID != "custom" {
		t.Fatalf("unexpected normalized menus: %+v", putBody.Data.Items)
	}
}

func TestSystemSettingsSchema(t *testing.T) {
	svc := newFakeSystemService()
	mux := http.NewServeMux()
	RegisterSystemRoutes(mux, svc, nil)

	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/v1/system/settings/schema", nil))
	if resp.Code != http.StatusOK {
		t.Fatalf("schema status=%d body=%s", resp.Code, resp.Body.String())
	}

	var body struct {
		Data SettingSchema `json:"data"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode schema response: %v", err)
	}
	if body.Data.TitleZhCN == "" || body.Data.TitleEnUS == "" {
		t.Fatalf("expected localized schema title, got %+v", body.Data)
	}
	if len(body.Data.Fields) < 4 {
		t.Fatalf("expected default schema fields, got %+v", body.Data.Fields)
	}
	if body.Data.Fields[0].Key != "audit.retention_days" || body.Data.Fields[0].Type != "number" {
		t.Fatalf("unexpected first schema field: %+v", body.Data.Fields[0])
	}
	if body.Data.Fields[0].Min == nil || *body.Data.Fields[0].Min != 1 {
		t.Fatalf("expected audit retention min rule, got %+v", body.Data.Fields[0].Min)
	}
	foundMenuTree := false
	for _, field := range body.Data.Fields {
		if field.Key == systemMenuSettingKey {
			foundMenuTree = true
			if field.Type != "textarea" {
				t.Fatalf("expected menu tree textarea field, got %+v", field)
			}
		}
	}
	if !foundMenuTree {
		t.Fatalf("expected menu tree setting field")
	}
}
