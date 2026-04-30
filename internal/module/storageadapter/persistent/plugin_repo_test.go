package persistent_test

import (
	"testing"

	"github.com/tinboxw/skoll/internal/module/storageadapter/persistent"
	"github.com/tinboxw/skoll/internal/module/storageadapter/persistent/db"
)

func newPluginRepo(t *testing.T) *persistent.PluginRepository {
	t.Helper()
	gdb := newTestDB(t)
	if err := db.Migrate(gdb, (&persistent.PluginRepository{}).Models()...); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	repo, err := persistent.NewPluginRepository(gdb)
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	return repo
}

func TestPluginRepository_InstallEnableDisable(t *testing.T) {
	repo := newPluginRepo(t)
	m := repo.Install("auth", "1.0.0", []string{"login"})
	if m.Name != "auth" || !m.Enabled {
		t.Fatalf("install: %+v", m)
	}
	got, err := repo.Get("auth")
	if err != nil || got.Version != "1.0.0" {
		t.Fatalf("get: %+v err=%v", got, err)
	}
	if list := repo.List(); len(list) != 1 {
		t.Fatalf("list: %+v", list)
	}
	d, err := repo.Disable("auth")
	if err != nil || d.Enabled {
		t.Fatalf("disable: %+v err=%v", d, err)
	}
	e, err := repo.Enable("auth")
	if err != nil || !e.Enabled {
		t.Fatalf("enable: %+v err=%v", e, err)
	}
}

func TestPluginRepository_HooksAndTrustRoots(t *testing.T) {
	repo := newPluginRepo(t)
	repo.Install("auth", "1.0.0", []string{"login"})

	h, err := repo.RegisterHook("login", "auth", "1.0.0", 1, 5000, 3, true)
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if h.Order != 1 || h.RetryLimit != 3 {
		t.Fatalf("hook: %+v", h)
	}
	if hooks := repo.ListHooks(); len(hooks) != 1 {
		t.Fatalf("list hooks: %+v", hooks)
	}
	if _, err := repo.SetHookEnabled("login", "auth", false); err != nil {
		t.Fatalf("set enabled: %v", err)
	}
	if _, err := repo.SetHookOrder("login", "auth", 5); err != nil {
		t.Fatalf("set order: %v", err)
	}

	roots := repo.SetMarketplaceTrustRoots([]string{"key-1", "key-2"})
	if len(roots) != 2 {
		t.Fatalf("trust roots: %+v", roots)
	}
}

func TestPluginRepository_HydrateAcrossInstances(t *testing.T) {
	gdb := newTestDB(t)
	if err := db.Migrate(gdb, (&persistent.PluginRepository{}).Models()...); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	repo, err := persistent.NewPluginRepository(gdb)
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	repo.Install("auth", "1.0.0", []string{"login"})
	if _, err := repo.RegisterHook("login", "auth", "1.0.0", 1, 5000, 3, true); err != nil {
		t.Fatalf("register: %v", err)
	}
	repo.SetMarketplaceTrustRoots([]string{"key-1"})

	repo2, err := persistent.NewPluginRepository(gdb)
	if err != nil {
		t.Fatalf("hydrate: %v", err)
	}
	got, err := repo2.Get("auth")
	if err != nil || got.Version != "1.0.0" {
		t.Fatalf("hydrated plugin: %+v err=%v", got, err)
	}
	if hooks := repo2.ListHooks(); len(hooks) != 1 {
		t.Fatalf("hydrated hooks: %+v", hooks)
	}
	if roots := repo2.ListMarketplaceTrustRoots(); len(roots) != 1 {
		t.Fatalf("hydrated trust roots: %+v", roots)
	}
}
