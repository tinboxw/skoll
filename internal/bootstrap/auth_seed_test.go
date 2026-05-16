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

	legacy, err := domainuser.New(shared.ID("legacy-admin"), "admin", "Legacy Admin", "admin@skoll.local", time.Now().UTC())
	if err != nil {
		t.Fatalf("new user error: %v", err)
	}
	legacyHash, err := domainuser.HashPassword("Legacy@123456")
	if err != nil {
		t.Fatalf("hash legacy password error: %v", err)
	}
	if err := legacy.SetPasswordHash(legacyHash.String()); err != nil {
		t.Fatalf("set legacy password error: %v", err)
	}
	legacy.Disable(time.Now().UTC())
	if err := users.Save(ctx, legacy); err != nil {
		t.Fatalf("save legacy user error: %v", err)
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
