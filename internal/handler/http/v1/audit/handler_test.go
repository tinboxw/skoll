package audit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	domainaudit "github.com/tinboxw/skoll/internal/domain/audit"
	"github.com/tinboxw/skoll/internal/domain/shared"
	auditsvc "github.com/tinboxw/skoll/internal/service/audit"
)

type fakeAuditService struct {
	byActorItems []*domainaudit.Record
	byRangeItems []*domainaudit.Record
	byActorCalls int
	byRangeCalls int
}

func (f *fakeAuditService) Append(context.Context, string, string, string, string, map[string]any) (*domainaudit.Record, error) {
	return nil, nil
}

func (f *fakeAuditService) GetByID(context.Context, string) (*domainaudit.Record, error) {
	return nil, nil
}

func (f *fakeAuditService) ListByActor(context.Context, string, int) ([]*domainaudit.Record, error) {
	f.byActorCalls++
	return f.byActorItems, nil
}

func (f *fakeAuditService) ListByTimeRange(context.Context, time.Time, time.Time, int) ([]*domainaudit.Record, error) {
	f.byRangeCalls++
	return f.byRangeItems, nil
}

func (f *fakeAuditService) ClearByTimeRange(context.Context, time.Time, time.Time) (int, error) {
	return 0, nil
}

type fakeAuditEventService struct {
	filter auditsvc.EventFilter
	items  []*domainaudit.Event
	item   *domainaudit.Event
	getID  string
	getErr error
}

func (f *fakeAuditEventService) AppendEvent(context.Context, *domainaudit.Event) error {
	return nil
}

func (f *fakeAuditEventService) GetEventByID(_ context.Context, id string) (*domainaudit.Event, error) {
	f.getID = id
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.item, nil
}

func (f *fakeAuditEventService) ListEvents(_ context.Context, filter auditsvc.EventFilter) ([]*domainaudit.Event, error) {
	f.filter = filter
	return f.items, nil
}

func (f *fakeAuditEventService) ExportEventSourceData(context.Context, auditsvc.EventFilter) ([]auditsvc.EventSourceData, error) {
	return nil, nil
}

func TestAuditHandlerListEventsWithFilters(t *testing.T) {
	occurredAt := time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)
	event := mustAuditEvent(t, "event-1", domainaudit.EventTypeSecurity, "system.security.deny", "actor-1", "role", "delete", domainaudit.EventResultDenied, domainaudit.EventRiskHigh, occurredAt)
	event.Trace.RequestID = "req-1"
	svc := &fakeAuditEventService{items: []*domainaudit.Event{event}}
	h := &AuditHandler{events: svc}

	req := httptest.NewRequest(http.MethodGet, "/v1/audit?type=security&actorId=actor-1&action=system.security.deny&resourceType=role&resourceId=delete&result=denied&risk=high&from=2026-06-01T00:00:00Z&to=2026-06-30T00:00:00Z&offset=5&limit=250", nil)
	resp := httptest.NewRecorder()

	h.list(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}
	if svc.filter.Type != domainaudit.EventTypeSecurity ||
		svc.filter.ActorID != "actor-1" ||
		svc.filter.Action != domainaudit.AuditAction("system.security.deny") ||
		svc.filter.ResourceType != "role" ||
		svc.filter.ResourceID != "delete" ||
		svc.filter.Result != domainaudit.EventResultDenied ||
		svc.filter.Risk != domainaudit.EventRiskHigh ||
		svc.filter.Offset != 5 ||
		svc.filter.Limit != 200 {
		t.Fatalf("unexpected filter: %+v", svc.filter)
	}
	body := resp.Body.String()
	if !strings.Contains(body, `"items"`) || !strings.Contains(body, `"event-1"`) || !strings.Contains(body, `"offset":5`) || !strings.Contains(body, `"limit":200`) || !strings.Contains(body, `"requestId":"req-1"`) {
		t.Fatalf("unexpected body: %s", body)
	}
}

func TestAuditHandlerListEventsRejectsInvalidFilter(t *testing.T) {
	h := &AuditHandler{events: &fakeAuditEventService{}}

	req := httptest.NewRequest(http.MethodGet, "/v1/audit?action=bad", nil)
	resp := httptest.NewRecorder()

	h.list(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestAuditHandlerGetEventDetail(t *testing.T) {
	occurredAt := time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)
	event := mustAuditEvent(t, "event-1", domainaudit.EventTypeOperation, "user.account.update", "actor-1", "user", "u1", domainaudit.EventResultSuccess, domainaudit.EventRiskMedium, occurredAt)
	event.Trace.RequestID = "req-1"
	event.Metadata = map[string]any{"status": "ok"}
	event.SourceData = map[string]any{
		"before": map[string]any{"name": "old"},
		"after":  map[string]any{"name": "new"},
	}
	svc := &fakeAuditEventService{item: event}
	h := &AuditHandler{events: svc}

	req := httptest.NewRequest(http.MethodGet, "/v1/audit/event-1", nil)
	req.SetPathValue("id", "event-1")
	resp := httptest.NewRecorder()

	h.get(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}
	if svc.getID != "event-1" {
		t.Fatalf("expected get id event-1, got %q", svc.getID)
	}
	body := resp.Body.String()
	for _, want := range []string{`"item"`, `"event-1"`, `"sourceData"`, `"diff"`, `"before"`, `"after"`, `"requestId":"req-1"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %s in body=%s", want, body)
		}
	}
}

func TestAuditHandlerGetEventDetailNotFound(t *testing.T) {
	h := &AuditHandler{events: &fakeAuditEventService{}}
	req := httptest.NewRequest(http.MethodGet, "/v1/audit/missing", nil)
	req.SetPathValue("id", "missing")
	resp := httptest.NewRecorder()

	h.get(resp, req)

	if resp.Code != http.StatusNotFound || !strings.Contains(resp.Body.String(), "not_found") {
		t.Fatalf("expected not_found, got %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestAuditHandlerGetEventDetailForbidden(t *testing.T) {
	h := &AuditHandler{events: &fakeAuditEventService{getErr: errAuditForbidden}}
	req := httptest.NewRequest(http.MethodGet, "/v1/audit/event-1", nil)
	req.SetPathValue("id", "event-1")
	resp := httptest.NewRecorder()

	h.get(resp, req)

	if resp.Code != http.StatusForbidden || !strings.Contains(resp.Body.String(), "forbidden") {
		t.Fatalf("expected forbidden, got %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestAuditHandlerListWithCombinedFilters(t *testing.T) {
	baseTime := time.Date(2026, 5, 18, 10, 0, 0, 0, time.UTC)
	h := &AuditHandler{service: &fakeAuditService{byRangeItems: []*domainaudit.Record{
		{ID: shared.ID("a1"), ActorID: shared.ID("1"), Action: "update", Resource: "user", OccurredAt: baseTime},
		{ID: shared.ID("a2"), ActorID: shared.ID("1"), Action: "update", Resource: "role", OccurredAt: baseTime},
		{ID: shared.ID("a3"), ActorID: shared.ID("2"), Action: "delete", Resource: "user", OccurredAt: baseTime},
	}}}

	req := httptest.NewRequest(http.MethodGet, "/v1/audit?from=2026-05-01T00:00:00Z&to=2026-05-31T00:00:00Z&actorId=1&action=update&resource=user&limit=10", nil)
	resp := httptest.NewRecorder()

	h.list(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}
	body := resp.Body.String()
	if !strings.Contains(body, "a1") {
		t.Fatalf("expected filtered record a1, body=%s", body)
	}
	if strings.Contains(body, "a2") || strings.Contains(body, "a3") {
		t.Fatalf("unexpected extra records in filtered response: %s", body)
	}
}

func TestFilterRecordsLimit(t *testing.T) {
	items := []*domainaudit.Record{
		{ID: shared.ID("a1"), ActorID: shared.ID("1"), Action: "update", Resource: "user"},
		{ID: shared.ID("a2"), ActorID: shared.ID("1"), Action: "update", Resource: "user"},
		{ID: shared.ID("a3"), ActorID: shared.ID("1"), Action: "update", Resource: "user"},
	}
	got := filterRecords(items, "1", "", "update", "user", 2)
	if len(got) != 2 {
		t.Fatalf("expected 2 items, got %d", len(got))
	}
	if got[0].ID.String() != "a1" || got[1].ID.String() != "a2" {
		t.Fatalf("unexpected order/items: %+v", got)
	}
}

func TestFilterRecordsByActorName(t *testing.T) {
	items := []*domainaudit.Record{
		{ID: shared.ID("a1"), ActorID: shared.ID("1"), Action: "login", Resource: "auth", Detail: map[string]any{"account": "admin"}},
		{ID: shared.ID("a2"), ActorID: shared.ID("2"), Action: "login", Resource: "auth", Detail: map[string]any{"account": "alice"}},
		{ID: shared.ID("a3"), ActorID: shared.ID("3"), Action: "login", Resource: "auth"},
	}

	got := filterRecords(items, "", "admin", "", "", 10)
	if len(got) != 1 {
		t.Fatalf("expected 1 item, got %d", len(got))
	}
	if got[0].ID.String() != "a1" {
		t.Fatalf("unexpected record: %+v", got[0])
	}
}

func TestAuditHandlerListActorOnlyUsesActorPath(t *testing.T) {
	svc := &fakeAuditService{byActorItems: []*domainaudit.Record{{ID: shared.ID("a1"), ActorID: shared.ID("1"), Action: "login", Resource: "auth"}}}
	h := &AuditHandler{service: svc}

	req := httptest.NewRequest(http.MethodGet, "/v1/audit?actorId=1&limit=10", nil)
	resp := httptest.NewRecorder()

	h.list(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}
	if svc.byActorCalls != 1 {
		t.Fatalf("expected ListByActor to be called once, got %d", svc.byActorCalls)
	}
	if svc.byRangeCalls != 0 {
		t.Fatalf("expected ListByTimeRange not called, got %d", svc.byRangeCalls)
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
		t.Fatalf("NewEvent error: %v", err)
	}
	return event
}
