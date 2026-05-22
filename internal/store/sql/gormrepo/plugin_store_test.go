package gormrepo

import (
	"context"
	"testing"

	"github.com/tinboxw/skoll/internal/plugin"
)

func TestPluginStore_Get(t *testing.T) {
	db := TestDB(t)
	store := NewPluginStore(db)
	ctx := context.Background()

	testPlugin := plugin.Info{
		ID:          "plugin-1",
		Name:        "Test Plugin",
		Version:     "1.0.0",
		Description: "A test plugin",
		State:       plugin.StateEnabled,
	}

	err := store.Save(ctx, testPlugin)
	if err != nil {
		t.Fatalf("failed to save plugin: %v", err)
	}

	t.Run("existing plugin", func(t *testing.T) {
		result, err := store.Get(ctx, "plugin-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected plugin, got nil")
		}
		if result.Name != "Test Plugin" {
			t.Errorf("expected name 'Test Plugin', got '%s'", result.Name)
		}
	})

	t.Run("non-existing plugin", func(t *testing.T) {
		result, err := store.Get(ctx, "nonexistent")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != nil {
			t.Errorf("expected nil for non-existing plugin, got %+v", result)
		}
	})
}

func TestPluginStore_List(t *testing.T) {
	db := TestDB(t)
	store := NewPluginStore(db)
	ctx := context.Background()

	for i := 1; i <= 3; i++ {
		p := plugin.Info{
			ID:          string(rune('a'+i-1)) + "-plugin",
			Name:        "Plugin " + string(rune('0'+i)),
			Version:     "1.0.0",
			Description: "Description " + string(rune('0'+i)),
			State:       plugin.StateEnabled,
		}
		if err := store.Save(ctx, p); err != nil {
			t.Fatalf("failed to save plugin %d: %v", i, err)
		}
	}

	result, err := store.List(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 3 {
		t.Errorf("expected 3 plugins, got %d", len(result))
	}
}

func TestPluginStore_SaveAndUpdate(t *testing.T) {
	db := TestDB(t)
	store := NewPluginStore(db)
	ctx := context.Background()

	original := plugin.Info{
		ID:      "updatable-plugin",
		Name:    "Original Name",
		Version: "1.0.0",
		State:   plugin.StateDisabled,
	}

	err := store.Save(ctx, original)
	if err != nil {
		t.Fatalf("failed to save plugin: %v", err)
	}

	updated := plugin.Info{
		ID:      "updatable-plugin",
		Name:    "Updated Name",
		Version: "2.0.0",
		State:   plugin.StateEnabled,
	}

	err = store.Save(ctx, updated)
	if err != nil {
		t.Fatalf("failed to update plugin: %v", err)
	}

	result, err := store.Get(ctx, "updatable-plugin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "Updated Name" {
		t.Errorf("expected name 'Updated Name', got '%s'", result.Name)
	}
	if result.Version != "2.0.0" {
		t.Errorf("expected version '2.0.0', got '%s'", result.Version)
	}
}

func TestPluginStore_Delete(t *testing.T) {
	db := TestDB(t)
	store := NewPluginStore(db)
	ctx := context.Background()

	p := plugin.Info{
		ID:    "deletable-plugin",
		Name:  "To Delete",
		State: plugin.StateEnabled,
	}

	if err := store.Save(ctx, p); err != nil {
		t.Fatalf("failed to save plugin: %v", err)
	}

	err := store.Delete(ctx, "deletable-plugin")
	if err != nil {
		t.Fatalf("failed to delete plugin: %v", err)
	}

	result, _ := store.Get(ctx, "deletable-plugin")
	if result != nil {
		t.Errorf("expected nil after delete, got %+v", result)
	}
}
