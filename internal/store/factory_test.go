package store

import (
	"context"
	"os"
	"strconv"
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
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b, err := NewBundle(tc.opts)
			if err != nil {
				t.Fatalf("NewBundle error: %v", err)
			}
			if b.Users == nil || b.Roles == nil || b.RBAC == nil || b.Audit == nil || b.Permissions == nil || b.Menus == nil || b.UnitOfWork == nil {
				t.Fatalf("bundle has nil repositories")
			}

			assertUserRepoContract(t, b)
			assertRoleRepoContract(t, b)
			assertAuditRepoContract(t, b)
		})
	}
}

func TestNewBundleMySQLIntegration(t *testing.T) {
	dsn := os.Getenv("SKOLL_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("SKOLL_TEST_MYSQL_DSN is not set")
	}

	b, err := NewBundle(Options{Mode: ModeMySQL, PrimaryDSN: dsn, ClickHouseDSN: "clickhouse://demo"})
	if err != nil {
		t.Fatalf("NewBundle mysql error: %v", err)
	}
	if b.Users == nil || b.Roles == nil || b.RBAC == nil || b.System == nil || b.Permissions == nil || b.Menus == nil {
		t.Fatalf("mysql bundle has nil repositories")
	}

	assertUserRepoContract(t, b)
	assertRoleRepoContract(t, b)
	assertAuditRepoContract(t, b)
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
	runKey := strconv.FormatInt(now.UnixNano()&0x7fffffff, 36)
	u, err := user.New(
		shared.ID("u-contract-"+runKey),
		"contract_user_"+runKey,
		"Contract User",
		"contract+"+runKey+"@example.com",
		now,
	)
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
	runKey := strconv.FormatInt(now.UnixNano()&0x7fffffff, 36)
	r, err := role.New(
		shared.ID("r-contract-"+runKey),
		"Contract Role",
		"contract_role_"+runKey,
		"for contract test",
		[]string{"user:read"},
		false,
		now,
	)
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
	runKey := strconv.FormatInt(now.UnixNano()&0x7fffffff, 36)
	actorID := shared.ID("u-contract-" + runKey)
	targetID := "u-contract-" + runKey
	rec, err := audit.NewRecord(
		shared.ID("a-1-"+runKey),
		actorID,
		"read",
		"user",
		targetID,
		map[string]any{"source": "test"},
		now,
	)
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
