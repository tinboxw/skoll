package gormrepo

import (
	"context"
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/domain/audit"
	"github.com/tinboxw/skoll/internal/domain/shared"
)

func TestAuditStore_Append(t *testing.T) {
	db := TestDB(t)
	store := NewAuditStore(db)
	ctx := context.Background()

	record := &audit.Record{
		ID:         shared.ID("audit-1"),
		ActorID:    shared.ID("user-1"),
		Action:     "user.login",
		Resource:   "auth",
		OccurredAt: time.Now(),
	}

	err := store.Append(ctx, record)
	if err != nil {
		t.Fatalf("failed to append audit record: %v", err)
	}

	t.Run("retrieve by ID", func(t *testing.T) {
		result, err := store.GetByID(ctx, "audit-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected audit record, got nil")
		}
		if result.Action != "user.login" {
			t.Errorf("expected action 'user.login', got '%s'", result.Action)
		}
	})
}

func TestAuditStore_ListByActor(t *testing.T) {
	db := TestDB(t)
	store := NewAuditStore(db)
	ctx := context.Background()

	now := time.Now()
	for i := 0; i < 3; i++ {
		record := &audit.Record{
			ID:         shared.ID(string(rune('a'+i)) + "-audit"),
			ActorID:    shared.ID("user-1"),
			Action:     "action-" + string(rune('0'+i)),
			OccurredAt: now.Add(time.Duration(i) * time.Hour),
		}
		if err := store.Append(ctx, record); err != nil {
			t.Fatalf("failed to append record %d: %v", i, err)
		}
	}

	result, err := store.ListByActor(ctx, shared.ID("user-1"), 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 3 {
		t.Errorf("expected 3 records, got %d", len(result))
	}
}

func TestAuditStore_ListByTimeRange(t *testing.T) {
	db := TestDB(t)
	store := NewAuditStore(db)
	ctx := context.Background()

	now := time.Now()

	oldRecord := &audit.Record{
		ID:         shared.ID("old-audit"),
		ActorID:    shared.ID("user-1"),
		Action:     "old.action",
		OccurredAt: now.Add(-48 * time.Hour),
	}

	newRecord := &audit.Record{
		ID:         shared.ID("new-audit"),
		ActorID:    shared.ID("user-2"),
		Action:     "new.action",
		OccurredAt: now.Add(-1 * time.Hour),
	}

	store.Append(ctx, oldRecord)
	store.Append(ctx, newRecord)

	t.Run("query recent records", func(t *testing.T) {
		tr := shared.TimeRange{From: now.Add(-24 * time.Hour), To: now}
		result, err := store.ListByTimeRange(ctx, tr, 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 1 {
			t.Errorf("expected 1 recent record, got %d", len(result))
		}
	})

	t.Run("delete old records", func(t *testing.T) {
		tr := shared.TimeRange{To: now.Add(-24 * time.Hour)}
		count, err := store.DeleteByTimeRange(ctx, tr)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if count == 0 {
			t.Error("expected to delete at least 1 old record")
		}
	})
}
