package audit

import (
	"context"
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/store/clickhouse"
)

func TestAuditServiceAppendAndList(t *testing.T) {
	svc := NewService(clickhouse.NewAuditStore())

	rec, err := svc.Append(context.Background(), "actor-1", "create", "user", "u-1", map[string]any{"ip": "127.0.0.1"})
	if err != nil {
		t.Fatalf("Append error: %v", err)
	}
	if rec.ID == "" {
		t.Fatalf("expected record id")
	}

	items, err := svc.ListByActor(context.Background(), "actor-1", 10)
	if err != nil {
		t.Fatalf("ListByActor error: %v", err)
	}
	if len(items) == 0 {
		t.Fatalf("expected audit items")
	}

	got, err := svc.GetByID(context.Background(), rec.ID.String())
	if err != nil {
		t.Fatalf("GetByID error: %v", err)
	}
	if got == nil || got.ID != rec.ID {
		t.Fatalf("expected matched audit record, got %+v", got)
	}

	from := rec.OccurredAt.Add(-time.Second)
	to := rec.OccurredAt.Add(time.Second)
	rangeItems, err := svc.ListByTimeRange(context.Background(), from, to, 10)
	if err != nil {
		t.Fatalf("ListByTimeRange error: %v", err)
	}
	if len(rangeItems) == 0 {
		t.Fatalf("expected range items")
	}

	deleted, err := svc.ClearByTimeRange(context.Background(), from, to)
	if err != nil {
		t.Fatalf("ClearByTimeRange error: %v", err)
	}
	if deleted <= 0 {
		t.Fatalf("expected deleted count > 0, got %d", deleted)
	}
}
