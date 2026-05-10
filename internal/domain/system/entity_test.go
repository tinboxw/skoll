package system

import (
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
)

func TestNewSetting(t *testing.T) {
	now := time.Date(2026, time.May, 10, 12, 0, 0, 0, time.UTC)

	t.Run("creates normalized setting", func(t *testing.T) {
		item, err := NewSetting(shared.ID("setting-1"), "  audit.retention.days ", "90", false, now)
		if err != nil {
			t.Fatalf("NewSetting returned error: %v", err)
		}
		if item.Key != "audit.retention.days" {
			t.Fatalf("unexpected key: %s", item.Key)
		}
		if item.Meta.CreatedAt.IsZero() || item.Meta.UpdatedAt.IsZero() {
			t.Fatalf("audit meta should be initialized")
		}
	})

	t.Run("rejects invalid id", func(t *testing.T) {
		_, err := NewSetting(shared.ID(""), "audit.retention.days", "90", false, now)
		if err == nil {
			t.Fatalf("expected error for empty id")
		}
	})

	t.Run("rejects invalid key", func(t *testing.T) {
		_, err := NewSetting(shared.ID("setting-1"), "!invalid", "90", false, now)
		if err == nil {
			t.Fatalf("expected error for invalid key")
		}
	})
}

func TestSettingUpdateValue(t *testing.T) {
	now := time.Date(2026, time.May, 10, 12, 0, 0, 0, time.UTC)
	later := now.Add(5 * time.Minute)

	item, err := NewSetting(shared.ID("setting-2"), "plugin.auto_enable", "true", false, now)
	if err != nil {
		t.Fatalf("NewSetting returned error: %v", err)
	}

	item.UpdateValue("false", later)
	if item.Value != "false" {
		t.Fatalf("unexpected value: %s", item.Value)
	}
	if !item.Meta.UpdatedAt.Equal(later) {
		t.Fatalf("unexpected UpdatedAt: %v", item.Meta.UpdatedAt)
	}
}
