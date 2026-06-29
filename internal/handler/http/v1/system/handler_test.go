package system

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
	domainsystem "github.com/tinboxw/skoll/internal/domain/system"
	systemsvc "github.com/tinboxw/skoll/internal/service/system"
)

type fakeSystemService struct {
	items     map[string]*domainsystem.Setting
	dictTypes map[string]*domainsystem.DictionaryType
	dictItems map[string]map[string]*domainsystem.DictionaryItem
}

func newFakeSystemService() *fakeSystemService {
	return &fakeSystemService{
		items:     map[string]*domainsystem.Setting{},
		dictTypes: map[string]*domainsystem.DictionaryType{},
		dictItems: map[string]map[string]*domainsystem.DictionaryItem{},
	}
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

func (f *fakeSystemService) SaveDictionaryType(_ context.Context, in systemsvc.DictionaryTypeInput) (*domainsystem.DictionaryType, error) {
	now := time.Date(2026, time.June, 6, 12, 0, 0, 0, time.UTC)
	id := shared.ID(in.ID)
	if id.IsZero() {
		id = shared.ID("type-" + in.Code)
	}
	item, err := domainsystem.NewDictionaryType(domainsystem.DictionaryTypeInput{
		ID:          id,
		Code:        in.Code,
		Name:        in.Name,
		Description: in.Description,
		Status:      domainsystem.DictionaryStatus(in.Status),
		Sort:        in.Sort,
		Builtin:     in.Builtin,
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		return nil, err
	}
	f.dictTypes[item.Code] = item
	return item, nil
}

func (f *fakeSystemService) GetDictionaryTypeByCode(_ context.Context, code string) (*domainsystem.DictionaryType, error) {
	return f.dictTypes[code], nil
}

func (f *fakeSystemService) ListDictionaryTypes(_ context.Context, _ systemsvc.DictionaryTypeListInput) ([]domainsystem.DictionaryType, error) {
	out := make([]domainsystem.DictionaryType, 0, len(f.dictTypes))
	for _, item := range f.dictTypes {
		out = append(out, *item)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Sort == out[j].Sort {
			return out[i].Code < out[j].Code
		}
		return out[i].Sort < out[j].Sort
	})
	return out, nil
}

func (f *fakeSystemService) DeleteDictionaryType(_ context.Context, id string) error {
	for code, item := range f.dictTypes {
		if item.ID.String() == id {
			delete(f.dictTypes, code)
			delete(f.dictItems, code)
			return nil
		}
	}
	return nil
}

func (f *fakeSystemService) SaveDictionaryItem(_ context.Context, in systemsvc.DictionaryItemInput) (*domainsystem.DictionaryItem, error) {
	now := time.Date(2026, time.June, 6, 12, 0, 0, 0, time.UTC)
	id := shared.ID(in.ID)
	if id.IsZero() {
		id = shared.ID("item-" + in.TypeCode + "-" + in.Value)
	}
	item, err := domainsystem.NewDictionaryItem(domainsystem.DictionaryItemInput{
		ID:        id,
		TypeCode:  in.TypeCode,
		Label:     in.Label,
		Value:     in.Value,
		Status:    domainsystem.DictionaryStatus(in.Status),
		Sort:      in.Sort,
		Builtin:   in.Builtin,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		return nil, err
	}
	if f.dictItems[item.TypeCode] == nil {
		f.dictItems[item.TypeCode] = map[string]*domainsystem.DictionaryItem{}
	}
	f.dictItems[item.TypeCode][item.Value] = item
	return item, nil
}

func (f *fakeSystemService) GetDictionaryItemByTypeAndValue(_ context.Context, typeCode, value string) (*domainsystem.DictionaryItem, error) {
	if f.dictItems[typeCode] == nil {
		return nil, nil
	}
	return f.dictItems[typeCode][value], nil
}

func (f *fakeSystemService) ListDictionaryItems(_ context.Context, in systemsvc.DictionaryItemListInput) ([]domainsystem.DictionaryItem, error) {
	itemsByValue := f.dictItems[in.TypeCode]
	out := make([]domainsystem.DictionaryItem, 0, len(itemsByValue))
	for _, item := range itemsByValue {
		out = append(out, *item)
	}
	return domainsystem.SortDictionaryItems(out), nil
}

func (f *fakeSystemService) DeleteDictionaryItem(_ context.Context, id string) error {
	for typeCode, itemsByValue := range f.dictItems {
		for value, item := range itemsByValue {
			if item.ID.String() == id {
				delete(f.dictItems[typeCode], value)
				return nil
			}
		}
	}
	return nil
}

func (f *fakeSystemService) RegisterConfigSchema(_ context.Context, in systemsvc.ConfigSchemaInput) (*systemsvc.ConfigSchema, error) {
	return &systemsvc.ConfigSchema{
		Scope:       in.Scope,
		Owner:       in.Owner,
		Title:       in.Title,
		TitleZhCN:   in.TitleZhCN,
		TitleEnUS:   in.TitleEnUS,
		Description: in.Description,
		Fields:      append([]systemsvc.ConfigField(nil), in.Fields...),
	}, nil
}

func (f *fakeSystemService) GetConfigSchema(_ context.Context, scope systemsvc.ConfigScope, _ string) (*systemsvc.ConfigSchema, error) {
	if scope != "" && scope != systemsvc.ConfigScopeSystem {
		return nil, nil
	}
	schema := systemsvc.DefaultSystemConfigSchema()
	return &schema, nil
}

func (f *fakeSystemService) ListConfigSchemas(_ context.Context, _ systemsvc.ConfigSchemaListInput) ([]systemsvc.ConfigSchema, error) {
	return []systemsvc.ConfigSchema{systemsvc.DefaultSystemConfigSchema()}, nil
}

func (f *fakeSystemService) ValidateConfigValues(_ context.Context, _ systemsvc.ConfigScope, _ string, values map[string]any) (map[string]any, error) {
	return values, nil
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

func TestSystemDictionariesDefaultOverrideAndLookup(t *testing.T) {
	svc := newFakeSystemService()
	mux := http.NewServeMux()
	RegisterSystemRoutes(mux, svc, nil)

	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, "/v1/system/dictionaries", nil))
	if resp.Code != http.StatusOK {
		t.Fatalf("empty dictionaries status=%d body=%s", resp.Code, resp.Body.String())
	}
	var emptyBody struct {
		Data struct {
			Items []DictionaryType `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &emptyBody); err != nil {
		t.Fatalf("decode empty dictionaries: %v", err)
	}
	if len(emptyBody.Data.Items) != 0 {
		t.Fatalf("expected empty dictionary list, got %+v", emptyBody.Data.Items)
	}

	payload := []byte(`{"items":[{"type":"order.status","name":"Order Status","description":"Order lifecycle","status":"enabled","order":30,"items":[{"label":"Paid","value":"paid","status":"enabled","order":20},{"label":"Pending","value":"pending","status":"enabled","order":10},{"label":"","value":"bad"}]},{"type":"","name":"Bad"}]}`)
	putResp := httptest.NewRecorder()
	mux.ServeHTTP(putResp, httptest.NewRequest(http.MethodPut, "/v1/system/dictionaries", bytes.NewReader(payload)))
	if putResp.Code != http.StatusOK {
		t.Fatalf("put dictionaries status=%d body=%s", putResp.Code, putResp.Body.String())
	}
	var putBody struct {
		Data struct {
			Items      []DictionaryType `json:"items"`
			Customized bool             `json:"customized"`
		} `json:"data"`
	}
	if err := json.Unmarshal(putResp.Body.Bytes(), &putBody); err != nil {
		t.Fatalf("decode put dictionaries: %v", err)
	}
	if len(putBody.Data.Items) != 1 || putBody.Data.Items[0].Code != "order.status" || putBody.Data.Items[0].Type != "order.status" {
		t.Fatalf("unexpected normalized dictionaries: %+v", putBody.Data.Items)
	}
	if got := putBody.Data.Items[0].Items; len(got) != 2 || got[0].Value != "pending" {
		t.Fatalf("expected sorted valid dictionary items, got %+v", got)
	}

	lookupResp := httptest.NewRecorder()
	mux.ServeHTTP(lookupResp, httptest.NewRequest(http.MethodGet, "/v1/system/dictionaries/order.status", nil))
	if lookupResp.Code != http.StatusOK {
		t.Fatalf("lookup dictionary status=%d body=%s", lookupResp.Code, lookupResp.Body.String())
	}
	var lookupBody struct {
		Data struct {
			Item DictionaryType `json:"item"`
		} `json:"data"`
	}
	if err := json.Unmarshal(lookupResp.Body.Bytes(), &lookupBody); err != nil {
		t.Fatalf("decode lookup dictionary: %v", err)
	}
	if lookupBody.Data.Item.Code != "order.status" || len(lookupBody.Data.Item.Items) != 2 {
		t.Fatalf("unexpected lookup dictionary: %+v", lookupBody.Data.Item)
	}

	filterResp := httptest.NewRecorder()
	mux.ServeHTTP(filterResp, httptest.NewRequest(http.MethodGet, "/v1/system/dictionaries?search=order&status=enabled", nil))
	if filterResp.Code != http.StatusOK {
		t.Fatalf("filter dictionary status=%d body=%s", filterResp.Code, filterResp.Body.String())
	}
	var filterBody struct {
		Data struct {
			Items []DictionaryType `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(filterResp.Body.Bytes(), &filterBody); err != nil {
		t.Fatalf("decode filter dictionary: %v", err)
	}
	if len(filterBody.Data.Items) != 1 {
		t.Fatalf("expected one filtered dictionary, got %+v", filterBody.Data.Items)
	}

	itemPayload := []byte(`{"label":"Refunded","status":"disabled","sort":30}`)
	itemResp := httptest.NewRecorder()
	mux.ServeHTTP(itemResp, httptest.NewRequest(http.MethodPut, "/v1/system/dictionaries/order.status/items/refunded", bytes.NewReader(itemPayload)))
	if itemResp.Code != http.StatusOK {
		t.Fatalf("put dictionary item status=%d body=%s", itemResp.Code, itemResp.Body.String())
	}

	itemsResp := httptest.NewRecorder()
	mux.ServeHTTP(itemsResp, httptest.NewRequest(http.MethodGet, "/v1/system/dictionaries/order.status/items?status=disabled&search=refund", nil))
	if itemsResp.Code != http.StatusOK {
		t.Fatalf("list dictionary items status=%d body=%s", itemsResp.Code, itemsResp.Body.String())
	}
	var itemsBody struct {
		Data struct {
			Items []DictionaryItem `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(itemsResp.Body.Bytes(), &itemsBody); err != nil {
		t.Fatalf("decode dictionary items: %v", err)
	}
	if len(itemsBody.Data.Items) != 1 || itemsBody.Data.Items[0].Value != "refunded" {
		t.Fatalf("unexpected filtered items: %+v", itemsBody.Data.Items)
	}

	deleteItemResp := httptest.NewRecorder()
	mux.ServeHTTP(deleteItemResp, httptest.NewRequest(http.MethodDelete, "/v1/system/dictionaries/order.status/items/refunded", nil))
	if deleteItemResp.Code != http.StatusOK {
		t.Fatalf("delete dictionary item status=%d body=%s", deleteItemResp.Code, deleteItemResp.Body.String())
	}

	missingResp := httptest.NewRecorder()
	mux.ServeHTTP(missingResp, httptest.NewRequest(http.MethodGet, "/v1/system/dictionaries/missing", nil))
	if missingResp.Code != http.StatusNotFound {
		t.Fatalf("missing dictionary status=%d body=%s", missingResp.Code, missingResp.Body.String())
	}

	deleteTypeResp := httptest.NewRecorder()
	mux.ServeHTTP(deleteTypeResp, httptest.NewRequest(http.MethodDelete, "/v1/system/dictionaries/order.status", nil))
	if deleteTypeResp.Code != http.StatusOK {
		t.Fatalf("delete dictionary type status=%d body=%s", deleteTypeResp.Code, deleteTypeResp.Body.String())
	}
}

func TestSystemOrganizationDefaultsAndOverride(t *testing.T) {
	svc := newFakeSystemService()
	mux := http.NewServeMux()
	RegisterSystemRoutes(mux, svc, nil)

	deptResp := httptest.NewRecorder()
	mux.ServeHTTP(deptResp, httptest.NewRequest(http.MethodGet, "/v1/system/departments", nil))
	if deptResp.Code != http.StatusOK {
		t.Fatalf("default departments status=%d body=%s", deptResp.Code, deptResp.Body.String())
	}
	var deptBody struct {
		Data struct {
			Items      []DepartmentRecord `json:"items"`
			Customized bool               `json:"customized"`
		} `json:"data"`
	}
	if err := json.Unmarshal(deptResp.Body.Bytes(), &deptBody); err != nil {
		t.Fatalf("decode default departments: %v", err)
	}
	if deptBody.Data.Customized || len(deptBody.Data.Items) == 0 {
		t.Fatalf("expected default non-customized departments, got %+v", deptBody.Data)
	}

	deptPayload := []byte(`{"items":[{"id":"dept-sales","name":"Sales","leader":"Alice","status":"enabled","order":20},{"id":"","name":"Bad"}]}`)
	deptPut := httptest.NewRecorder()
	mux.ServeHTTP(deptPut, httptest.NewRequest(http.MethodPut, "/v1/system/departments", bytes.NewReader(deptPayload)))
	if deptPut.Code != http.StatusOK {
		t.Fatalf("put departments status=%d body=%s", deptPut.Code, deptPut.Body.String())
	}
	var deptPutBody struct {
		Data struct {
			Items []DepartmentRecord `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(deptPut.Body.Bytes(), &deptPutBody); err != nil {
		t.Fatalf("decode put departments: %v", err)
	}
	if len(deptPutBody.Data.Items) != 1 || deptPutBody.Data.Items[0].ID != "dept-sales" {
		t.Fatalf("unexpected departments: %+v", deptPutBody.Data.Items)
	}

	posResp := httptest.NewRecorder()
	mux.ServeHTTP(posResp, httptest.NewRequest(http.MethodGet, "/v1/system/positions", nil))
	if posResp.Code != http.StatusOK {
		t.Fatalf("default positions status=%d body=%s", posResp.Code, posResp.Body.String())
	}

	posPayload := []byte(`{"items":[{"id":"pos-sales","code":"sales","name":"Sales","status":"enabled","order":30},{"id":"bad","code":"","name":"Bad"}]}`)
	posPut := httptest.NewRecorder()
	mux.ServeHTTP(posPut, httptest.NewRequest(http.MethodPut, "/v1/system/positions", bytes.NewReader(posPayload)))
	if posPut.Code != http.StatusOK {
		t.Fatalf("put positions status=%d body=%s", posPut.Code, posPut.Body.String())
	}
	var posPutBody struct {
		Data struct {
			Items []PositionRecord `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(posPut.Body.Bytes(), &posPutBody); err != nil {
		t.Fatalf("decode put positions: %v", err)
	}
	if len(posPutBody.Data.Items) != 1 || posPutBody.Data.Items[0].Code != "sales" {
		t.Fatalf("unexpected positions: %+v", posPutBody.Data.Items)
	}
}
