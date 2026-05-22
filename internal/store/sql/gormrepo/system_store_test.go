package gormrepo

import (
	"context"
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
	"github.com/tinboxw/skoll/internal/domain/system"
)

func TestSystemStore_GetByID(t *testing.T) {
	db := TestDB(t)
	store := NewSystemStore(db, nil)
	ctx := context.Background()

	testSetting := &system.Setting{
		ID:    shared.ID("1"),
		Key:   "app.name",
		Value: "Skoll Platform",
	}
	testSetting.Meta.Touch(time.Now())

	err := store.SaveSetting(ctx, testSetting)
	if err != nil {
		t.Fatalf("failed to save setting: %v", err)
	}

	t.Run("existing setting", func(t *testing.T) {
		result, err := store.GetSettingByID(ctx, shared.ID("1"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected setting, got nil")
		}
		if result.Value != "Skoll Platform" {
			t.Errorf("expected value 'Skoll Platform', got '%s'", result.Value)
		}
	})

	t.Run("non-existing setting", func(t *testing.T) {
		result, err := store.GetSettingByID(ctx, shared.ID("999"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != nil {
			t.Errorf("expected nil for non-existing setting, got %+v", result)
		}
	})
}

func TestSystemStore_GetByKey(t *testing.T) {
	db := TestDB(t)
	store := NewSystemStore(db, nil)
	ctx := context.Background()

	settings := []*system.Setting{
		{ID: shared.ID("s-1"), Key: "app.name", Value: "MyApp"},
		{ID: shared.ID("s-2"), Key: "app.version", Value: "1.0.0"},
	}
	for _, s := range settings {
		s.Meta.Touch(time.Now())
		if err := store.SaveSetting(ctx, s); err != nil {
			t.Fatalf("failed to save setting: %v", err)
		}
	}

	t.Run("existing key", func(t *testing.T) {
		result, err := store.GetSettingByKey(ctx, "app.name")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected setting, got nil")
		}
		if result.Value != "MyApp" {
			t.Errorf("expected value 'MyApp', got '%s'", result.Value)
		}
	})

	t.Run("non-existing key", func(t *testing.T) {
		result, err := store.GetSettingByKey(ctx, "app.nonexistent")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != nil {
			t.Errorf("expected nil, got %+v", result)
		}
	})
}

func TestSystemStore_List(t *testing.T) {
	db := TestDB(t)
	store := NewSystemStore(db, nil)
	ctx := context.Background()

	for i := 1; i <= 4; i++ {
		s := &system.Setting{
			ID:    shared.ID(string(rune('a'+i-1)) + "-setting"),
			Key:   "config." + string(rune('0'+i)),
			Value: "value-" + string(rune('0'+i)),
		}
		s.Meta.Touch(time.Now())
		if err := store.SaveSetting(ctx, s); err != nil {
			t.Fatalf("failed to save setting %d: %v", i, err)
		}
	}

	t.Run("list all with pagination", func(t *testing.T) {
		result, err := store.ListSettings(ctx, 0, 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 4 {
			t.Errorf("expected 4 settings, got %d", len(result))
		}
	})

	t.Run("with offset and limit", func(t *testing.T) {
		result, err := store.ListSettings(ctx, 1, 2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 2 {
			t.Errorf("expected 2 settings, got %d", len(result))
		}
	})
}

func TestSystemStore_SaveAndUpdate(t *testing.T) {
	db := TestDB(t)
	store := NewSystemStore(db, nil)
	ctx := context.Background()

	original := &system.Setting{
		ID:    shared.ID("2"),
		Key:   "updatable.key",
		Value: "original value",
	}
	original.Meta.Touch(time.Now())

	err := store.SaveSetting(ctx, original)
	if err != nil {
		t.Fatalf("failed to save setting: %v", err)
	}

	updated := &system.Setting{
		ID:    shared.ID("2"),
		Key:   "updatable.key",
		Value: "updated value",
	}
	updated.Meta.Touch(time.Now())

	err = store.SaveSetting(ctx, updated)
	if err != nil {
		t.Fatalf("failed to update setting: %v", err)
	}

	result, err := store.GetSettingByKey(ctx, "updatable.key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Value != "updated value" {
		t.Errorf("expected value 'updated value', got '%s'", result.Value)
	}
}

func TestSystemStore_Delete(t *testing.T) {
	db := TestDB(t)
	store := NewSystemStore(db, nil)
	ctx := context.Background()

	s := &system.Setting{
		ID:    shared.ID("3"),
		Key:   "deletable.key",
		Value: "to delete",
	}
	s.Meta.Touch(time.Now())

	if err := store.SaveSetting(ctx, s); err != nil {
		t.Fatalf("failed to save setting: %v", err)
	}

	err := store.DeleteSetting(ctx, shared.ID("3"))
	if err != nil {
		t.Fatalf("failed to delete setting: %v", err)
	}

	result, _ := store.GetSettingByID(ctx, shared.ID("3"))
	if result != nil {
		t.Errorf("expected nil after delete, got %+v", result)
	}
}
