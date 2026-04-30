package persistent_test

import (
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/tinboxw/skoll/internal/module/audit"
	"github.com/tinboxw/skoll/internal/module/storageadapter/persistent"
	"github.com/tinboxw/skoll/internal/module/storageadapter/persistent/db"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	gdb, err := db.OpenSQLite("", db.Options{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		if sqlDB, err := gdb.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	return gdb
}

func TestAuditRepository_AppendQueryRecent(t *testing.T) {
	gdb := newTestDB(t)
	repo := persistent.NewAuditRepository(gdb)
	if err := db.Migrate(gdb, repo.Models()...); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	r1 := repo.Append("alice", "login", "user:1")
	r2 := repo.Append("bob", "logout", "user:2")
	r3 := repo.Append("alice", "create", "role:1")
	if r1.ID <= 0 || r2.ID <= r1.ID || r3.ID <= r2.ID {
		t.Fatalf("unexpected ids: %d %d %d", r1.ID, r2.ID, r3.ID)
	}
	if r3.CreatedAt.IsZero() {
		t.Fatalf("created_at not set")
	}

	recent := repo.Recent(2)
	if len(recent) != 2 || recent[0].ID != r3.ID || recent[1].ID != r2.ID {
		t.Fatalf("unexpected recent: %+v", recent)
	}

	q := repo.Query(audit.Query{Page: 1, Size: 10, Actor: "alice"})
	if q.Total != 2 || len(q.Items) != 2 {
		t.Fatalf("unexpected actor filter: %+v", q)
	}

	q = repo.Query(audit.Query{Page: 1, Size: 10, Q: "log"})
	if q.Total != 2 {
		t.Fatalf("expected 2 fuzzy matches for 'log', got %d", q.Total)
	}

	q = repo.Query(audit.Query{Page: 2, Size: 2})
	if q.Total != 3 || len(q.Items) != 1 {
		t.Fatalf("unexpected page=2 size=2 result: %+v", q)
	}

	q = repo.Query(audit.Query{Page: 99, Size: 10})
	if q.Total != 3 || len(q.Items) != 0 {
		t.Fatalf("expected empty page beyond range: %+v", q)
	}
}

func TestConfigRepository_SetGetList(t *testing.T) {
	gdb := newTestDB(t)
	repo := persistent.NewConfigRepository(gdb)
	if err := db.Migrate(gdb, repo.Models()...); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	if _, err := repo.Get("missing"); err == nil {
		t.Fatalf("expected not-found error")
	}

	a := repo.Set("system.theme", "aurora", "ui theme")
	if a.UpdatedAt.IsZero() || a.Value != "aurora" {
		t.Fatalf("unexpected entry: %+v", a)
	}
	b := repo.Set("system.locale", "zh_CN", "locale")
	_ = b

	got, err := repo.Get("system.theme")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Value != "aurora" || got.Description != "ui theme" {
		t.Fatalf("unexpected fetched entry: %+v", got)
	}

	// Update existing key
	updated := repo.Set("system.theme", "void", "ui theme v2")
	if updated.Value != "void" {
		t.Fatalf("update did not take: %+v", updated)
	}
	if !updated.UpdatedAt.After(a.UpdatedAt) && !updated.UpdatedAt.Equal(a.UpdatedAt) {
		t.Fatalf("updated_at regression: a=%v updated=%v", a.UpdatedAt, updated.UpdatedAt)
	}

	list := repo.List()
	if len(list) != 2 || list[0].Key != "system.locale" || list[1].Key != "system.theme" {
		t.Fatalf("unexpected list ordering: %+v", list)
	}
}

func TestDictionaryRepository_CRUDAndListByType(t *testing.T) {
	gdb := newTestDB(t)
	repo := persistent.NewDictionaryRepository(gdb)
	if err := db.Migrate(gdb, repo.Models()...); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	if _, err := repo.Get(999); err == nil {
		t.Fatalf("expected not-found for missing id")
	}

	en := repo.Create("status", "Enabled", "1", 10, true)
	dis := repo.Create("status", "Disabled", "0", 20, true)
	other := repo.Create("priority", "High", "h", 5, true)
	if en.ID <= 0 || dis.ID <= en.ID || other.ID <= dis.ID {
		t.Fatalf("unexpected ids: %d %d %d", en.ID, dis.ID, other.ID)
	}

	got, err := repo.Get(en.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Label != "Enabled" {
		t.Fatalf("unexpected get: %+v", got)
	}

	statusItems := repo.ListByType("status")
	if len(statusItems) != 2 || statusItems[0].ID != en.ID || statusItems[1].ID != dis.ID {
		t.Fatalf("unexpected status items: %+v", statusItems)
	}

	all := repo.List()
	if len(all) != 3 {
		t.Fatalf("expected 3 items, got %d", len(all))
	}
	// priority(5) sorts before status(10/20) by type alphabetical
	if all[0].Type != "priority" || all[1].ID != en.ID || all[2].ID != dis.ID {
		t.Fatalf("unexpected list ordering: %+v", all)
	}

	_ = time.Now
}
