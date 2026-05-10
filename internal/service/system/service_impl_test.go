package system

import (
	"context"
	"testing"

	"github.com/tinboxw/skoll/internal/store/memory"
)

func TestSystemServiceUpsertAndList(t *testing.T) {
	svc := NewService(memory.NewSystemStore())

	first, err := svc.Upsert(context.Background(), UpsertInput{Key: "feature.alpha", Value: "on", Encrypted: false})
	if err != nil {
		t.Fatalf("Upsert create error: %v", err)
	}
	if first == nil || first.ID == "" {
		t.Fatalf("expected created setting")
	}

	second, err := svc.Upsert(context.Background(), UpsertInput{Key: "feature.alpha", Value: "off", Encrypted: true})
	if err != nil {
		t.Fatalf("Upsert update error: %v", err)
	}
	if second == nil || second.Value != "off" || !second.Encrypted {
		t.Fatalf("unexpected updated setting: %+v", second)
	}

	items, err := svc.List(context.Background(), ListInput{Offset: 0, Limit: 10})
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 setting, got %d", len(items))
	}
}
