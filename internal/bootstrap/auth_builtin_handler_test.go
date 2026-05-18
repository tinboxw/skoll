package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	domainuser "github.com/tinboxw/skoll/internal/domain/user"
	"github.com/tinboxw/skoll/internal/store/memory"
	"github.com/tinboxw/skoll/pkg/logging"
)

type loginResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Token string `json:"token"`
		User  struct {
			Account string `json:"account"`
		} `json:"user"`
	} `json:"data"`
}

func newSeededAuthHandler(t *testing.T) (*builtinAuthHandler, *memory.UserStore) {
	t.Helper()

	users := memory.NewUserStore()
	roles := memory.NewRoleStore()
	rbac := memory.NewRBACStore()
	ensureBuiltinAuthData(context.Background(), logging.Discard(), users, roles, rbac)

	h := newBuiltinAuthHandler("test-secret", users, roles, rbac, nil)
	if h == nil {
		t.Fatal("expected builtin auth handler")
	}
	return h, users
}

func TestBuiltinAuthLoginSuccessWithSeededAdminCredentials(t *testing.T) {
	h, users := newSeededAuthHandler(t)

	seeded, err := users.GetByAccount(context.Background(), "admin")
	if err != nil {
		t.Fatalf("load seeded admin: %v", err)
	}
	if seeded == nil {
		t.Fatal("expected seeded admin user")
	}
	if !domainuser.VerifyPassword("Admin@123456", seeded.Password) {
		t.Fatalf("expected seeded admin password to match")
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewBufferString(`{"account":"admin","password":"Admin@123456"}`))
	resp := httptest.NewRecorder()

	h.handleLogin(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	var body loginResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if body.Data.Token == "" {
		t.Fatalf("expected token in response")
	}
	if body.Data.User.Account != "admin" {
		t.Fatalf("expected admin account in response, got %q", body.Data.User.Account)
	}
}

func TestBuiltinAuthLoginRejectsWrongPassword(t *testing.T) {
	h, _ := newSeededAuthHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewBufferString(`{"account":"admin","password":"admin"}`))
	resp := httptest.NewRecorder()

	h.handleLogin(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestBuiltinAuthLoginRejectsIdentifierFieldPayload(t *testing.T) {
	h, _ := newSeededAuthHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", bytes.NewBufferString(`{"identifier":"admin","password":"Admin@123456"}`))
	resp := httptest.NewRecorder()

	h.handleLogin(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", resp.Code, resp.Body.String())
	}
}
