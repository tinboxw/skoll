package role

import (
	"context"
	"strings"
	"testing"
	"time"

	domainrole "github.com/tinboxw/skoll/internal/domain/role"
	"github.com/tinboxw/skoll/internal/domain/shared"
	"github.com/tinboxw/skoll/internal/store/memory"
)

func TestRoleServiceGrantRevoke(t *testing.T) {
	svc := NewService(memory.NewRoleStore())

	r, err := svc.Create(context.Background(), CreateRoleInput{
		Name:        "Operator",
		Key:         "operator",
		Description: "ops",
		Permissions: []string{"user:read"},
	})
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}

	r, err = svc.Grant(context.Background(), r.ID.String(), "user:write")
	if err != nil {
		t.Fatalf("Grant error: %v", err)
	}
	if len(r.Permissions) < 2 {
		t.Fatalf("expected permissions to grow")
	}

	r, err = svc.Revoke(context.Background(), r.ID.String(), "user:write")
	if err != nil {
		t.Fatalf("Revoke error: %v", err)
	}
	for _, p := range r.Permissions {
		if p == "user:write" {
			t.Fatalf("permission should be revoked")
		}
	}
}

func TestRoleServiceValidationAndErrorPaths(t *testing.T) {
	svc := NewService(memory.NewRoleStore())

	t.Run("get requires id", func(t *testing.T) {
		_, err := svc.Get(context.Background(), "")
		if err == nil || !strings.Contains(strings.ToLower(err.Error()), "id is required") {
			t.Fatalf("expected id required error, got %v", err)
		}
	})

	t.Run("list rejects invalid pagination", func(t *testing.T) {
		_, err := svc.List(context.Background(), ListInput{Offset: -1, Limit: 10})
		if err == nil || !strings.Contains(strings.ToLower(err.Error()), "invalid pagination") {
			t.Fatalf("expected invalid pagination error, got %v", err)
		}
	})

	t.Run("update not found", func(t *testing.T) {
		_, err := svc.Update(context.Background(), UpdateRoleInput{ID: "missing", Name: "x"})
		if err == nil || !strings.Contains(strings.ToLower(err.Error()), "role not found") {
			t.Fatalf("expected role not found error, got %v", err)
		}
	})

	t.Run("delete built-in denied", func(t *testing.T) {
		builtIn, err := domainrole.New(shared.ID("builtin-role"), "Super Admin", "super_admin", "system", []string{"*"}, true, time.Now().UTC())
		if err != nil {
			t.Fatalf("new role error: %v", err)
		}

		store := memory.NewRoleStore()
		if err := store.Save(context.Background(), builtIn); err != nil {
			t.Fatalf("save built-in role error: %v", err)
		}
		svc2 := NewService(store)
		err = svc2.Delete(context.Background(), builtIn.ID.String())
		if err == nil || !strings.Contains(strings.ToLower(err.Error()), "built-in role") {
			t.Fatalf("expected built-in delete error, got %v", err)
		}
	})
}

func TestRoleServiceUpdateNormalizesKeyAndPermissions(t *testing.T) {
	store := memory.NewRoleStore()
	svc := NewService(store)

	created, err := svc.Create(context.Background(), CreateRoleInput{
		Name:        "Auditor",
		Key:         "auditor",
		Description: "desc",
		Permissions: []string{" user.read ", "user.read", "role.read"},
	})
	if err != nil {
		t.Fatalf("create role error: %v", err)
	}

	updated, err := svc.Update(context.Background(), UpdateRoleInput{
		ID:          created.ID.String(),
		Name:        "Auditor Team",
		Key:         "AUDITOR_TEAM",
		Description: " updated ",
		Permissions: []string{" role.read ", "user.read", "role.read"},
	})
	if err != nil {
		t.Fatalf("update role error: %v", err)
	}
	if updated.Key != "auditor_team" {
		t.Fatalf("expected normalized key auditor_team, got %q", updated.Key)
	}
	if len(updated.Permissions) != 2 {
		t.Fatalf("expected deduplicated permissions, got %v", updated.Permissions)
	}
}
