package menu

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	domainmenu "github.com/tinboxw/skoll/internal/domain/menu"
	menusvc "github.com/tinboxw/skoll/internal/service/menu"
)

func TestMenuHandlerTreeFiltersAccess(t *testing.T) {
	svc := &fakeService{nodes: []domainmenu.MenuNode{mustNode(t, "system.users", "system", "/system/users", 10)}}
	mux := http.NewServeMux()
	RegisterMenuRoutes(mux, svc)

	req := httptest.NewRequest(http.MethodGet, "/v1/menus/tree?source=system&visible=true&roles=admin&permissions=system:user:list", nil)
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
	if svc.treeInput.Source != "system" || svc.treeInput.Visible == nil || !*svc.treeInput.Visible {
		t.Fatalf("tree input = %+v", svc.treeInput)
	}
	if len(svc.filterInput.Roles) != 1 || svc.filterInput.Roles[0] != "admin" {
		t.Fatalf("filter input = %+v", svc.filterInput)
	}
	body := decodeBody(t, resp)
	items := body["data"].(map[string]any)["items"].([]any)
	if len(items) != 1 || items[0].(map[string]any)["key"] != "system.users" {
		t.Fatalf("items = %+v", items)
	}
}

func TestMenuHandlerSaveReorderAndVisibility(t *testing.T) {
	svc := &fakeService{nodes: []domainmenu.MenuNode{mustNode(t, "system.users", "system", "/system/users", 10)}}
	mux := http.NewServeMux()
	RegisterMenuRoutes(mux, svc)

	saveBody := []byte(`{"items":[{"key":"system.users","parentKey":"system","source":"system","name":"Users","path":"/system/users","sort":10,"visible":true}]}`)
	saveReq := httptest.NewRequest(http.MethodPut, "/v1/menus", bytes.NewReader(saveBody))
	saveResp := httptest.NewRecorder()
	mux.ServeHTTP(saveResp, saveReq)
	if saveResp.Code != http.StatusOK || len(svc.mergedNodes) != 1 {
		t.Fatalf("save status=%d merged=%d body=%s", saveResp.Code, len(svc.mergedNodes), saveResp.Body.String())
	}

	reorderReq := httptest.NewRequest(http.MethodPost, "/v1/menus/reorder", bytes.NewReader([]byte(`{"parentKey":"system","orderedKeys":["system.users"]}`)))
	reorderResp := httptest.NewRecorder()
	mux.ServeHTTP(reorderResp, reorderReq)
	if reorderResp.Code != http.StatusOK || svc.reorderInput.ParentKey != "system" {
		t.Fatalf("reorder status=%d input=%+v", reorderResp.Code, svc.reorderInput)
	}

	visibilityReq := httptest.NewRequest(http.MethodPatch, "/v1/menus/visibility", bytes.NewReader([]byte(`{"key":"system.users","visible":false}`)))
	visibilityResp := httptest.NewRecorder()
	mux.ServeHTTP(visibilityResp, visibilityReq)
	if visibilityResp.Code != http.StatusOK {
		t.Fatalf("visibility status=%d body=%s", visibilityResp.Code, visibilityResp.Body.String())
	}
	if len(svc.mergedNodes) != 1 || svc.mergedNodes[0].Visible {
		t.Fatalf("merged visibility = %+v", svc.mergedNodes)
	}
}

func TestMenuHandlerFilterErrorReturnsForbidden(t *testing.T) {
	svc := &fakeService{filterErr: fmt.Errorf("forbidden")}
	mux := http.NewServeMux()
	RegisterMenuRoutes(mux, svc)

	req := httptest.NewRequest(http.MethodGet, "/v1/menus/tree?roles=missing", nil)
	resp := httptest.NewRecorder()
	mux.ServeHTTP(resp, req)
	if resp.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", resp.Code, resp.Body.String())
	}
}

type fakeService struct {
	nodes        []domainmenu.MenuNode
	treeInput    menusvc.TreeInput
	filterInput  menusvc.FilterInput
	mergedNodes  []domainmenu.MenuNode
	reorderInput menusvc.ReorderInput
	filterErr    error
}

func (s *fakeService) MergeNodes(_ context.Context, in menusvc.MergeNodesInput) ([]domainmenu.MenuNode, error) {
	s.mergedNodes = append([]domainmenu.MenuNode(nil), in.Nodes...)
	return s.mergedNodes, nil
}

func (s *fakeService) Tree(_ context.Context, in menusvc.TreeInput) ([]domainmenu.MenuNode, error) {
	s.treeInput = in
	return append([]domainmenu.MenuNode(nil), s.nodes...), nil
}

func (s *fakeService) Filter(_ context.Context, in menusvc.FilterInput) ([]domainmenu.MenuNode, error) {
	s.filterInput = in
	if s.filterErr != nil {
		return nil, s.filterErr
	}
	return append([]domainmenu.MenuNode(nil), in.Nodes...), nil
}

func (s *fakeService) Reorder(_ context.Context, in menusvc.ReorderInput) error {
	s.reorderInput = in
	return nil
}

func mustNode(t *testing.T, key string, parent string, path string, sort int) domainmenu.MenuNode {
	t.Helper()
	node, err := domainmenu.NewNode(domainmenu.NodeIdentity{
		Key:       key,
		ParentKey: parent,
		Source:    "system",
	}, domainmenu.NodeView{
		Name: key,
		Path: path,
	}, sort)
	if err != nil {
		t.Fatalf("NewNode() error = %v", err)
	}
	return node
}

func decodeBody(t *testing.T, resp *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v; %s", err, resp.Body.String())
	}
	return body
}
