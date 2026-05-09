package store

import (
	"context"
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/domain/audit"
	"github.com/tinboxw/skoll/internal/domain/role"
	"github.com/tinboxw/skoll/internal/domain/shared"
	"github.com/tinboxw/skoll/internal/domain/user"
)

func TestNewBundleModes(t *testing.T) {
	cases := []struct {
		name string
		opts Options
	}{
		{name: "memory", opts: Options{Mode: ModeMemory}},
		{name: "mysql", opts: Options{Mode: ModeMySQL, PrimaryDSN: "mysql://demo", ClickHouseDSN: "clickhouse://demo"}},
		{name: "postgres", opts: Options{Mode: ModePostgres, PrimaryDSN: "postgres://demo", ClickHouseDSN: "clickhouse://demo"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b, err := NewBundle(tc.opts)
			if err != nil {
				t.Fatalf("NewBundle error: %v", err)
			}
			if b.Users == nil || b.Roles == nil || b.RBAC == nil || b.Audit == nil || b.UnitOfWork == nil {
				t.Fatalf("bundle has nil repositories")
			}

			assertUserRepoContract(t, b)
			assertRoleRepoContract(t, b)
			assertAuditRepoContract(t, b)
		})
	}
}

func TestNewBundleUnsupportedMode(t *testing.T) {
	if _, err := NewBundle(Options{Mode: Mode("bad")}); err == nil {
		t.Fatalf("expected unsupported mode error")
	}
}

func assertUserRepoContract(t *testing.T, b *Bundle) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC()
	u, err := user.New(shared.ID("u-contract"), "contract_user", "Contract User", "contract@example.com", now)
	if err != nil {
		t.Fatalf("user.New error: %v", err)
	}
	if err := b.Users.Save(ctx, u); err != nil {
		t.Fatalf("Users.Save error: %v", err)
	}

	got, err := b.Users.GetByID(ctx, u.ID)
	if err != nil {
		t.Fatalf("Users.GetByID error: %v", err)
	}
	if got == nil || got.ID != u.ID {
		t.Fatalf("unexpected user from repo")
	}
}

func assertRoleRepoContract(t *testing.T, b *Bundle) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC()
	r, err := role.New(shared.ID("r-contract"), "Contract Role", "contract_role", "for contract test", []string{"user:read"}, false, now)
	if err != nil {
		t.Fatalf("role.New error: %v", err)
	}
	if err := b.Roles.Save(ctx, r); err != nil {
		t.Fatalf("Roles.Save error: %v", err)
	}
	got, err := b.Roles.GetByID(ctx, r.ID)
	if err != nil {
		t.Fatalf("Roles.GetByID error: %v", err)
	}
	if got == nil || got.ID != r.ID {
		t.Fatalf("unexpected role from repo")
	}
}

func assertAuditRepoContract(t *testing.T, b *Bundle) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC()
	rec, err := audit.NewRecord(shared.ID("a-1"), shared.ID("u-contract"), "read", "user", "u-contract", map[string]any{"source": "test"}, now)
	if err != nil {
		t.Fatalf("audit.NewRecord error: %v", err)
	}
	if err := b.Audit.Append(ctx, rec); err != nil {
		t.Fatalf("Audit.Append error: %v", err)
	}
	got, err := b.Audit.GetByID(ctx, rec.ID)
	if err != nil {
		t.Fatalf("Audit.GetByID error: %v", err)
	}
	if got == nil || got.ID != rec.ID {
		t.Fatalf("unexpected audit record")
	}
}
