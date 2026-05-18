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
)

type fakeAuditService struct {
	byActorItems []*domainaudit.Record
	byRangeItems []*domainaudit.Record
}

func (f *fakeAuditService) Append(context.Context, string, string, string, string, map[string]any) (*domainaudit.Record, error) {
	return nil, nil
}

func (f *fakeAuditService) GetByID(context.Context, string) (*domainaudit.Record, error) {
	return nil, nil
}

func (f *fakeAuditService) ListByActor(context.Context, string, int) ([]*domainaudit.Record, error) {
	return f.byActorItems, nil
}

func (f *fakeAuditService) ListByTimeRange(context.Context, time.Time, time.Time, int) ([]*domainaudit.Record, error) {
	return f.byRangeItems, nil
}

func (f *fakeAuditService) ClearByTimeRange(context.Context, time.Time, time.Time) (int, error) {
	return 0, nil
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
	got := filterRecords(items, "1", "update", "user", 2)
	if len(got) != 2 {
		t.Fatalf("expected 2 items, got %d", len(got))
	}
	if got[0].ID.String() != "a1" || got[1].ID.String() != "a2" {
		t.Fatalf("unexpected order/items: %+v", got)
	}
}
