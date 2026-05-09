package audit

import (
	"context"
	"testing"

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
}
