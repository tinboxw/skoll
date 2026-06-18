package memory

import (
	"context"
	"testing"
	"time"

	domainaudit "github.com/tinboxw/skoll/internal/domain/audit"
	"github.com/tinboxw/skoll/internal/domain/shared"
	auditrepo "github.com/tinboxw/skoll/internal/repository/audit"
)

func TestAuditEventStoreAppendAndGet(t *testing.T) {
	ctx := context.Background()
	store := NewAuditEventStore()
	event := mustAuditEvent(t, "audit-1", domainaudit.EventTypeOperation, "menu.node.update", "actor-1", "menu", "menu-1", domainaudit.EventResultSuccess, domainaudit.EventRiskMedium, time.Now())
	event.Metadata["owner"] = "platform"
	event.SourceData["raw"] = "source"

	if err := store.AppendEvent(ctx, event); err != nil {
		t.Fatalf("AppendEvent() error = %v", err)
	}
	event.Metadata["owner"] = "changed"
	event.SourceData["raw"] = "changed"

	got, err := store.GetEventByID(ctx, shared.ID("audit-1"))
	if err != nil {
		t.Fatalf("GetEventByID() error = %v", err)
	}
	if got == nil || got.ID != shared.ID("audit-1") {
		t.Fatalf("got = %+v", got)
	}
	if got.Metadata["owner"] != "platform" || got.SourceData["raw"] != "source" {
		t.Fatalf("store should clone event maps: metadata=%+v source=%+v", got.Metadata, got.SourceData)
	}

	got.Metadata["owner"] = "mutated"
	again, _ := store.GetEventByID(ctx, shared.ID("audit-1"))
	if again.Metadata["owner"] != "platform" {
		t.Fatalf("GetEventByID should return clone: %+v", again.Metadata)
	}
}

func TestAuditEventStoreListFiltersAndSorts(t *testing.T) {
	ctx := context.Background()
	store := NewAuditEventStore()
	base := time.Date(2026, time.June, 19, 12, 0, 0, 0, time.UTC)
	events := []*domainaudit.Event{
		mustAuditEvent(t, "audit-1", domainaudit.EventTypeOperation, "menu.node.update", "actor-1", "menu", "menu-1", domainaudit.EventResultSuccess, domainaudit.EventRiskMedium, base.Add(time.Minute)),
		mustAuditEvent(t, "audit-2", domainaudit.EventTypeSecurity, "rbac.role.grant", "actor-1", "role", "role-1", domainaudit.EventResultDenied, domainaudit.EventRiskHigh, base.Add(2*time.Minute)),
		mustAuditEvent(t, "audit-3", domainaudit.EventTypeOperation, "user.account.update", "actor-2", "user", "user-1", domainaudit.EventResultFailure, domainaudit.EventRiskMedium, base.Add(3*time.Minute)),
	}
	for _, event := range events {
		if err := store.AppendEvent(ctx, event); err != nil {
			t.Fatalf("AppendEvent() error = %v", err)
		}
	}

	items, err := store.ListEvents(ctx, auditrepo.EventFilter{ActorID: shared.ID("actor-1")})
	if err != nil {
		t.Fatalf("ListEvents(actor) error = %v", err)
	}
	assertAuditEventIDs(t, items, []string{"audit-2", "audit-1"})

	items, err = store.ListEvents(ctx, auditrepo.EventFilter{
		Type:         domainaudit.EventTypeOperation,
		ResourceType: " MENU ",
		ResourceID:   " menu-1 ",
		Result:       domainaudit.EventResultSuccess,
		Risk:         domainaudit.EventRiskMedium,
		TimeRange: shared.TimeRange{
			From: base,
			To:   base.Add(2 * time.Minute),
		},
	})
	if err != nil {
		t.Fatalf("ListEvents(filters) error = %v", err)
	}
	assertAuditEventIDs(t, items, []string{"audit-1"})

	items, err = store.ListEvents(ctx, auditrepo.EventFilter{Offset: 1, Limit: 1})
	if err != nil {
		t.Fatalf("ListEvents(page) error = %v", err)
	}
	assertAuditEventIDs(t, items, []string{"audit-2"})
}

func TestAuditEventStoreExportSourceData(t *testing.T) {
	ctx := context.Background()
	store := NewAuditEventStore()
	event := mustAuditEvent(t, "audit-1", domainaudit.EventTypeLogin, "auth.session.login", "actor-1", "auth", "session-1", domainaudit.EventResultSuccess, domainaudit.EventRiskLow, time.Now())
	event.SourceData["account"] = "admin"

	if err := store.AppendEvent(ctx, event); err != nil {
		t.Fatalf("AppendEvent() error = %v", err)
	}

	rows, err := store.ExportEventSourceData(ctx, auditrepo.EventFilter{Type: domainaudit.EventTypeLogin})
	if err != nil {
		t.Fatalf("ExportEventSourceData() error = %v", err)
	}
	if len(rows) != 1 || rows[0].EventID != shared.ID("audit-1") || rows[0].SourceData["account"] != "admin" {
		t.Fatalf("unexpected source rows: %+v", rows)
	}
	rows[0].SourceData["account"] = "changed"

	again, _ := store.ExportEventSourceData(ctx, auditrepo.EventFilter{Type: domainaudit.EventTypeLogin})
	if again[0].SourceData["account"] != "admin" {
		t.Fatalf("ExportEventSourceData should clone source data: %+v", again)
	}
}

func TestAuditEventStoreEmptyInputs(t *testing.T) {
	store := NewAuditEventStore()
	if err := store.AppendEvent(context.Background(), nil); err != nil {
		t.Fatalf("AppendEvent(nil) error = %v", err)
	}
	got, err := store.GetEventByID(context.Background(), "")
	if err != nil {
		t.Fatalf("GetEventByID(empty) error = %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil event, got %+v", got)
	}
}

func mustAuditEvent(t *testing.T, id string, eventType domainaudit.EventType, action string, actorID string, resourceType string, resourceID string, result domainaudit.EventResult, risk domainaudit.EventRisk, occurredAt time.Time) *domainaudit.Event {
	t.Helper()
	event, err := domainaudit.NewEvent(domainaudit.EventInput{
		ID:     shared.ID(id),
		Type:   eventType,
		Action: domainaudit.AuditAction(action),
		Actor: domainaudit.ActorRef{
			Type: "user",
			ID:   shared.ID(actorID),
		},
		Resource: domainaudit.ResourceRef{
			Type: resourceType,
			ID:   resourceID,
		},
		Result:     result,
		Risk:       risk,
		OccurredAt: occurredAt,
	})
	if err != nil {
		t.Fatalf("NewEvent() error = %v", err)
	}
	return event
}

func assertAuditEventIDs(t *testing.T, items []*domainaudit.Event, want []string) {
	t.Helper()
	if len(items) != len(want) {
		t.Fatalf("len = %d, want %d: %+v", len(items), len(want), items)
	}
	for i, item := range items {
		if item.ID.String() != want[i] {
			t.Fatalf("items[%d].ID = %q, want %q", i, item.ID, want[i])
		}
	}
}
