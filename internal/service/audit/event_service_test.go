package audit

import (
	"context"
	"errors"
	"testing"
	"time"

	domainaudit "github.com/tinboxw/skoll/internal/domain/audit"
	"github.com/tinboxw/skoll/internal/domain/shared"
	auditrepo "github.com/tinboxw/skoll/internal/repository/audit"
)

type fakeEventRepo struct {
	appendErr error
	getErr    error
	listErr   error
	exportErr error

	appendCalled bool
	getID        shared.ID
	listFilter   auditrepo.EventFilter
	exportFilter auditrepo.EventFilter

	event   *domainaudit.Event
	events  []*domainaudit.Event
	sources []auditrepo.EventSourceData
}

func (f *fakeEventRepo) AppendEvent(_ context.Context, event *domainaudit.Event) error {
	f.appendCalled = true
	f.event = event
	return f.appendErr
}

func (f *fakeEventRepo) GetEventByID(_ context.Context, id shared.ID) (*domainaudit.Event, error) {
	f.getID = id
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.event, nil
}

func (f *fakeEventRepo) ListEvents(_ context.Context, filter auditrepo.EventFilter) ([]*domainaudit.Event, error) {
	f.listFilter = filter
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.events, nil
}

func (f *fakeEventRepo) ExportEventSourceData(_ context.Context, filter auditrepo.EventFilter) ([]auditrepo.EventSourceData, error) {
	f.exportFilter = filter
	if f.exportErr != nil {
		return nil, f.exportErr
	}
	return f.sources, nil
}

func TestEventServiceAppendEvent(t *testing.T) {
	repo := &fakeEventRepo{}
	svc := NewEventService(repo)
	event := mustAuditEvent(t)

	if err := svc.AppendEvent(context.Background(), event); err != nil {
		t.Fatalf("AppendEvent error: %v", err)
	}
	if !repo.appendCalled || repo.event != event {
		t.Fatalf("expected event to be appended")
	}
}

func TestEventServiceAppendEventValidation(t *testing.T) {
	repo := &fakeEventRepo{}
	svc := NewEventService(repo)

	if err := svc.AppendEvent(context.Background(), nil); err == nil {
		t.Fatalf("expected nil event error")
	}
	if repo.appendCalled {
		t.Fatalf("repo should not be called for nil event")
	}
}

func TestEventServiceGetEventByID(t *testing.T) {
	event := mustAuditEvent(t)
	repo := &fakeEventRepo{event: event}
	svc := NewEventService(repo)

	got, err := svc.GetEventByID(context.Background(), "event-1")
	if err != nil {
		t.Fatalf("GetEventByID error: %v", err)
	}
	if got != event || repo.getID != shared.ID("event-1") {
		t.Fatalf("expected event lookup by service id, got event=%+v id=%s", got, repo.getID)
	}
}

func TestEventServiceListEventsMapsFilter(t *testing.T) {
	now := time.Date(2026, time.June, 19, 10, 0, 0, 0, time.UTC)
	event := mustAuditEvent(t)
	repo := &fakeEventRepo{events: []*domainaudit.Event{event}}
	svc := NewEventService(repo)

	got, err := svc.ListEvents(context.Background(), EventFilter{
		Type:         domainaudit.EventTypeOperation,
		ActorID:      "user-1",
		Action:       domainaudit.AuditAction("rbac.role.create"),
		ResourceType: "role",
		ResourceID:   "role-1",
		Result:       domainaudit.EventResultSuccess,
		Risk:         domainaudit.EventRiskMedium,
		From:         now.Add(-time.Hour),
		To:           now,
		Offset:       20,
		Limit:        50,
	})
	if err != nil {
		t.Fatalf("ListEvents error: %v", err)
	}
	if len(got) != 1 || got[0] != event {
		t.Fatalf("expected service events, got %+v", got)
	}
	filter := repo.listFilter
	if filter.Type != domainaudit.EventTypeOperation ||
		filter.ActorID != shared.ID("user-1") ||
		filter.Action != domainaudit.AuditAction("rbac.role.create") ||
		filter.ResourceType != "role" ||
		filter.ResourceID != "role-1" ||
		filter.Result != domainaudit.EventResultSuccess ||
		filter.Risk != domainaudit.EventRiskMedium ||
		filter.Offset != 20 ||
		filter.Limit != 50 ||
		!filter.TimeRange.From.Equal(now.Add(-time.Hour)) ||
		!filter.TimeRange.To.Equal(now) {
		t.Fatalf("unexpected mapped filter: %+v", filter)
	}
}

func TestEventServiceListEventsInvalidRange(t *testing.T) {
	repo := &fakeEventRepo{}
	svc := NewEventService(repo)
	now := time.Date(2026, time.June, 19, 10, 0, 0, 0, time.UTC)

	_, err := svc.ListEvents(context.Background(), EventFilter{From: now, To: now.Add(-time.Second)})
	if !errors.Is(err, domainaudit.ErrInvalidTimeRange) {
		t.Fatalf("expected ErrInvalidTimeRange, got %v", err)
	}
	if repo.listFilter != (auditrepo.EventFilter{}) {
		t.Fatalf("repo should not receive invalid filter")
	}
}

func TestEventServiceExportEventSourceData(t *testing.T) {
	repo := &fakeEventRepo{sources: []auditrepo.EventSourceData{{
		EventID:    shared.ID("event-1"),
		SourceData: map[string]any{"before": "old"},
	}}}
	svc := NewEventService(repo)

	got, err := svc.ExportEventSourceData(context.Background(), EventFilter{ActorID: "user-1", Limit: 10})
	if err != nil {
		t.Fatalf("ExportEventSourceData error: %v", err)
	}
	if len(got) != 1 || got[0].EventID != "event-1" || got[0].SourceData["before"] != "old" {
		t.Fatalf("unexpected source data: %+v", got)
	}
	got[0].SourceData["before"] = "changed"
	if repo.sources[0].SourceData["before"] != "old" {
		t.Fatalf("service should not expose repository source map")
	}
	if repo.exportFilter.ActorID != shared.ID("user-1") || repo.exportFilter.Limit != 10 {
		t.Fatalf("unexpected export filter: %+v", repo.exportFilter)
	}
}

func TestEventServicePropagatesRepositoryErrors(t *testing.T) {
	wantErr := errors.New("repo failed")
	event := mustAuditEvent(t)

	if err := NewEventService(&fakeEventRepo{appendErr: wantErr}).AppendEvent(context.Background(), event); !errors.Is(err, wantErr) {
		t.Fatalf("expected append repo error, got %v", err)
	}
	if _, err := NewEventService(&fakeEventRepo{getErr: wantErr}).GetEventByID(context.Background(), "event-1"); !errors.Is(err, wantErr) {
		t.Fatalf("expected get repo error, got %v", err)
	}
	if _, err := NewEventService(&fakeEventRepo{listErr: wantErr}).ListEvents(context.Background(), EventFilter{}); !errors.Is(err, wantErr) {
		t.Fatalf("expected list repo error, got %v", err)
	}
	if _, err := NewEventService(&fakeEventRepo{exportErr: wantErr}).ExportEventSourceData(context.Background(), EventFilter{}); !errors.Is(err, wantErr) {
		t.Fatalf("expected export repo error, got %v", err)
	}
}

func TestEventServiceRequiresRepository(t *testing.T) {
	svc := NewEventService(nil)
	if err := svc.AppendEvent(context.Background(), mustAuditEvent(t)); !errors.Is(err, ErrEventRepositoryNotConfigured) {
		t.Fatalf("expected repository configuration error, got %v", err)
	}
	if _, err := svc.GetEventByID(context.Background(), "event-1"); !errors.Is(err, ErrEventRepositoryNotConfigured) {
		t.Fatalf("expected repository configuration error, got %v", err)
	}
	if _, err := svc.ListEvents(context.Background(), EventFilter{}); !errors.Is(err, ErrEventRepositoryNotConfigured) {
		t.Fatalf("expected repository configuration error, got %v", err)
	}
	if _, err := svc.ExportEventSourceData(context.Background(), EventFilter{}); !errors.Is(err, ErrEventRepositoryNotConfigured) {
		t.Fatalf("expected repository configuration error, got %v", err)
	}
}

func mustAuditEvent(t *testing.T) *domainaudit.Event {
	t.Helper()
	event, err := domainaudit.NewEvent(domainaudit.EventInput{
		ID:     shared.ID("event-1"),
		Type:   domainaudit.EventTypeOperation,
		Action: domainaudit.AuditAction("rbac.role.create"),
		Actor: domainaudit.ActorRef{
			Type: "user",
			ID:   shared.ID("user-1"),
			Name: "Admin",
		},
		Resource: domainaudit.ResourceRef{
			Type: "role",
			ID:   "role-1",
			Name: "Manager",
		},
		Result:     domainaudit.EventResultSuccess,
		Risk:       domainaudit.EventRiskMedium,
		OccurredAt: time.Date(2026, time.June, 19, 10, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("NewEvent error: %v", err)
	}
	return event
}
