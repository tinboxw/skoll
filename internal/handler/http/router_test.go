package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	auditsvc "github.com/tinboxw/skoll/internal/service/audit"
	rbacsvc "github.com/tinboxw/skoll/internal/service/rbac"
	rolesvc "github.com/tinboxw/skoll/internal/service/role"
	usersvc "github.com/tinboxw/skoll/internal/service/user"
	"github.com/tinboxw/skoll/internal/store"
)

func TestRouterUserCreateAndGet(t *testing.T) {
	bundle, err := store.NewBundle(store.Options{Mode: store.ModeMemory})
	if err != nil {
		t.Fatalf("store.NewBundle error: %v", err)
	}
	_ = auditsvc.NewService(bundle.Audit)
	userService := usersvc.NewService(bundle.Users, bundle.Audit, bundle.UnitOfWork)
	roleService := rolesvc.NewService(bundle.Roles)
	rbacService := rbacsvc.NewService(bundle.RBAC)

	router := NewRouter(Dependencies{UserService: userService, RoleService: roleService, RBACService: rbacService})

	createReq := map[string]any{
		"username":     "api_user",
		"displayName":  "API User",
		"email":        "api@example.com",
		"passwordHash": "1234567890abcdef",
		"actorID":      "admin-1",
	}
	raw, _ := json.Marshal(createReq)
	req := httptest.NewRequest(http.MethodPost, "/v1/users", bytes.NewReader(raw))
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", resp.Code, resp.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/v1/users?offset=0&limit=10", nil)
	listResp := httptest.NewRecorder()
	router.ServeHTTP(listResp, listReq)
	if listResp.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", listResp.Code, listResp.Body.String())
	}
}
