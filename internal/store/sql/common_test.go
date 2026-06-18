package sql

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type testUserModel struct {
	ID    string `gorm:"primaryKey"`
	Name  string
	Score int
}

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		if isSQLiteCGODisabledError(err) {
			t.Skipf("sqlite test requires cgo: %v", err)
		}
		t.Fatalf("open sqlite error: %v", err)
	}
	if err := db.AutoMigrate(&testUserModel{}); err != nil {
		t.Fatalf("migrate error: %v", err)
	}
	return db
}

func isSQLiteCGODisabledError(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "requires cgo") || strings.Contains(msg, "cgo_enabled=0")
}

func TestParseDialect(t *testing.T) {
	t.Run("mysql", func(t *testing.T) {
		d, err := ParseDialect(" MySQL ")
		if err != nil {
			t.Fatalf("ParseDialect mysql error: %v", err)
		}
		if d != DialectMySQL {
			t.Fatalf("expected mysql dialect, got %q", d)
		}
	})

	t.Run("postgres", func(t *testing.T) {
		d, err := ParseDialect("POSTGRES")
		if err != nil {
			t.Fatalf("ParseDialect postgres error: %v", err)
		}
		if d != DialectPostgres {
			t.Fatalf("expected postgres dialect, got %q", d)
		}
	})

	t.Run("unsupported", func(t *testing.T) {
		_, err := ParseDialect("sqlite")
		if err == nil {
			t.Fatalf("expected unsupported dialect error")
		}
		if !strings.Contains(strings.ToLower(err.Error()), "unsupported sql dialect") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestNormalizeDSN(t *testing.T) {
	got := NormalizeDSN("  user:pass@tcp(127.0.0.1:3306)/db  ")
	if got != "user:pass@tcp(127.0.0.1:3306)/db" {
		t.Fatalf("unexpected normalized dsn: %q", got)
	}
}

func TestFirstWhere(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)

	if err := db.Create(&testUserModel{ID: "u1", Name: "alice", Score: 10}).Error; err != nil {
		t.Fatalf("seed error: %v", err)
	}

	t.Run("found", func(t *testing.T) {
		item, err := FirstWhere[testUserModel](ctx, db, "id = ?", "u1")
		if err != nil {
			t.Fatalf("FirstWhere error: %v", err)
		}
		if item == nil || item.Name != "alice" {
			t.Fatalf("unexpected item: %+v", item)
		}
	})

	t.Run("not found returns nil", func(t *testing.T) {
		item, err := FirstWhere[testUserModel](ctx, db, "id = ?", "missing")
		if err != nil {
			t.Fatalf("FirstWhere error: %v", err)
		}
		if item != nil {
			t.Fatalf("expected nil item for not found")
		}
	})
}

func TestListOrderedSaveAndDeleteByID(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)

	seed := []testUserModel{
		{ID: "u1", Name: "alice", Score: 20},
		{ID: "u2", Name: "bob", Score: 30},
		{ID: "u3", Name: "carol", Score: 10},
	}
	if err := db.Create(&seed).Error; err != nil {
		t.Fatalf("seed error: %v", err)
	}

	items, err := ListOrdered[testUserModel](ctx, db, "score desc", 1, 1)
	if err != nil {
		t.Fatalf("ListOrdered error: %v", err)
	}
	if len(items) != 1 || items[0].ID != "u1" {
		t.Fatalf("unexpected ordered result: %+v", items)
	}

	model := &testUserModel{ID: "u4", Name: "dave", Score: 5}
	if err := SaveModel(ctx, db, model); err != nil {
		t.Fatalf("SaveModel create error: %v", err)
	}
	model.Score = 99
	if err := SaveModel(ctx, db, model); err != nil {
		t.Fatalf("SaveModel update error: %v", err)
	}

	got, err := FirstWhere[testUserModel](ctx, db, "id = ?", "u4")
	if err != nil {
		t.Fatalf("FirstWhere error: %v", err)
	}
	if got == nil || got.Score != 99 {
		t.Fatalf("expected updated score 99, got %+v", got)
	}

	if err := DeleteByID[testUserModel](ctx, db, "u4"); err != nil {
		t.Fatalf("DeleteByID error: %v", err)
	}
	deleted, err := FirstWhere[testUserModel](ctx, db, "id = ?", "u4")
	if err != nil {
		t.Fatalf("FirstWhere after delete error: %v", err)
	}
	if deleted != nil {
		t.Fatalf("expected deleted model")
	}
}
