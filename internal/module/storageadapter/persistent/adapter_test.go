package persistent_test

import (
	"testing"

	"github.com/tinboxw/skoll/internal/module/storageadapter/contracts"
	"github.com/tinboxw/skoll/internal/module/storageadapter/persistent"
)

func TestAdapter_WiresAllRepositories(t *testing.T) {
	gdb := newTestDB(t)
	a, err := persistent.NewAdapter(gdb, t.TempDir())
	if err != nil {
		t.Fatalf("new adapter: %v", err)
	}

	var _ contracts.Adapter = a

	// Smoke: one round-trip per repo group.
	u := a.Users().Create("alice", "alice@example.com")
	if u.ID == 0 {
		t.Fatalf("user create returned zero id: %+v", u)
	}
	r := a.Roles().Create("admin", []string{"all"})
	if r.ID == 0 {
		t.Fatalf("role create returned zero id: %+v", r)
	}
	m := a.Menus().Create("Home", "/", 1)
	if m.ID == 0 {
		t.Fatalf("menu create returned zero id: %+v", m)
	}
	if rec := a.Audit().Append("alice", "login", "system"); rec.ID == 0 {
		t.Fatalf("audit append returned zero id: %+v", rec)
	}
	if e := a.Configs().Set("ui.theme", "dark", "ui theme"); e.Key == "" {
		t.Fatalf("config set returned empty: %+v", e)
	}
	plug := a.Plugins().Install("auth", "1.0.0", []string{"login"})
	if plug.Name != "auth" {
		t.Fatalf("plugin install: %+v", plug)
	}

	// Hydration round-trip: building a fresh adapter on the same DB must
	// see committed state.
	a2, err := persistent.NewAdapter(gdb, t.TempDir())
	if err != nil {
		t.Fatalf("rehydrate adapter: %v", err)
	}
	got, err := a2.Users().Get(u.ID)
	if err != nil || got.Email != "alice@example.com" {
		t.Fatalf("user not hydrated: %+v err=%v", got, err)
	}
	gotPlug, err := a2.Plugins().Get("auth")
	if err != nil || gotPlug.Version != "1.0.0" {
		t.Fatalf("plugin not hydrated: %+v err=%v", gotPlug, err)
	}
}
