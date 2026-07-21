package bootstrap

import (
	"context"
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
	domainuser "github.com/tinboxw/skoll/internal/domain/user"
	"github.com/tinboxw/skoll/internal/store/memory"
	"github.com/tinboxw/skoll/pkg/logging"
)

func TestEnsureBuiltinAuthDataResetsSeedAdminPassword(t *testing.T) {
	ctx := context.Background()
	users := memory.NewUserStore()
	roles := memory.NewRoleStore()
	rbac := memory.NewRBACStore()
	logger := logging.Discard()

	existing, err := domainuser.New(shared.ID("existing-admin"), "admin", "Existing Admin", "admin@skoll.local", time.Now().UTC())
	if err != nil {
		t.Fatalf("new user error: %v", err)
	}
	existingHash, err := domainuser.HashPassword("Existing@123456")
	if err != nil {
		t.Fatalf("hash existing password error: %v", err)
	}
	if err := existing.SetPasswordHash(existingHash.String()); err != nil {
		t.Fatalf("set existing password error: %v", err)
	}
	existing.Disable(time.Now().UTC())
	if err := users.Save(ctx, existing); err != nil {
		t.Fatalf("save existing user error: %v", err)
	}

	ensureBuiltinAuthData(ctx, logger, users, roles, rbac)

	admin, err := users.GetByAccount(ctx, "admin")
	if err != nil {
		t.Fatalf("load admin error: %v", err)
	}
	if admin == nil {
		t.Fatal("expected admin user")
	}
	if !domainuser.VerifyPassword("Admin@123456", admin.Password) {
		t.Fatalf("expected seeded admin password to be reset")
	}
	if admin.Status != domainuser.StatusActive {
		t.Fatalf("expected admin status active, got %s", admin.Status)
	}
}
