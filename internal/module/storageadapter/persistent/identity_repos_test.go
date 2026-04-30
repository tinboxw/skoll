package persistent_test

import (
	"testing"

	"github.com/tinboxw/skoll/internal/module/storageadapter/persistent"
	"github.com/tinboxw/skoll/internal/module/storageadapter/persistent/db"
)

func TestRoleRepository_CreateGetList(t *testing.T) {
	gdb := newTestDB(t)
	repo := persistent.NewRoleRepository(gdb)
	if err := db.Migrate(gdb, repo.Models()...); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	if _, err := repo.Get(404); err == nil {
		t.Fatalf("expected not-found")
	}

	a := repo.Create("admin", []string{"user.read", "user.write"})
	b := repo.Create("viewer", nil)
	if a.ID <= 0 || b.ID <= a.ID {
		t.Fatalf("unexpected ids: %d %d", a.ID, b.ID)
	}

	got, err := repo.Get(a.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Name != "admin" || len(got.Permissions) != 2 || got.Permissions[0] != "user.read" {
		t.Fatalf("unexpected role: %+v", got)
	}

	gotB, err := repo.Get(b.ID)
	if err != nil {
		t.Fatalf("get b: %v", err)
	}
	if gotB.Permissions != nil {
		t.Fatalf("expected nil permissions, got %+v", gotB.Permissions)
	}

	list := repo.List()
	if len(list) != 2 || list[0].ID != a.ID || list[1].ID != b.ID {
		t.Fatalf("unexpected list: %+v", list)
	}
}

func TestMenuRepository_CreateGetListOrdering(t *testing.T) {
	gdb := newTestDB(t)
	repo := persistent.NewMenuRepository(gdb)
	if err := db.Migrate(gdb, repo.Models()...); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	if _, err := repo.Get(404); err == nil {
		t.Fatalf("expected not-found")
	}

	dash := repo.Create("Dashboard", "/dashboard", 20)
	sys := repo.Create("System", "/system", 10)
	tools := repo.Create("Tools", "/tools", 10) // same order as sys -> ID tiebreak

	got, err := repo.Get(dash.ID)
	if err != nil || got.Path != "/dashboard" {
		t.Fatalf("get dashboard: %+v err=%v", got, err)
	}

	list := repo.List()
	if len(list) != 3 {
		t.Fatalf("expected 3 menus, got %d", len(list))
	}
	if list[0].ID != sys.ID || list[1].ID != tools.ID || list[2].ID != dash.ID {
		t.Fatalf("unexpected menu ordering: %+v", list)
	}
}

func TestAPIRegistryRepository_RegisterAndQuery(t *testing.T) {
	gdb := newTestDB(t)
	repo := persistent.NewAPIRegistryRepository(gdb)
	if err := db.Migrate(gdb, repo.Models()...); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	repo.RegisterMany([]string{"GET /admin/v1/users", "POST:/admin/v1/users", "  ", "bogus"})
	if !repo.Exists("GET:/admin/v1/users") {
		t.Fatalf("expected GET entry registered")
	}
	if !repo.Exists("POST /admin/v1/users") {
		t.Fatalf("expected POST entry registered (alt form)")
	}
	if repo.Exists("DELETE:/admin/v1/users") {
		t.Fatalf("did not expect DELETE entry")
	}

	// Re-register identical entries: must remain idempotent.
	repo.RegisterMany([]string{"GET:/admin/v1/users"})
	list := repo.List()
	if len(list) != 2 {
		t.Fatalf("expected 2 unique entries, got %+v", list)
	}
	if list[0] != "GET:/admin/v1/users" || list[1] != "POST:/admin/v1/users" {
		t.Fatalf("unexpected sorted list: %+v", list)
	}
}
