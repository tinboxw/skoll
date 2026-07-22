package plugin

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	domainaudit "github.com/tinboxw/skoll/internal/domain/audit"
	"github.com/tinboxw/skoll/internal/domain/shared"
	auditsvc "github.com/tinboxw/skoll/internal/service/audit"
	jobsvc "github.com/tinboxw/skoll/internal/service/job"
	"github.com/tinboxw/skoll/internal/store/clickhouse"
)

type diagnosticCatalogStub struct{ items map[string]Info }

func (s diagnosticCatalogStub) Get(pluginID string) (Info, error) {
	item, ok := s.items[pluginID]
	if !ok {
		return Info{}, ErrPluginNotFound
	}
	return item, nil
}

type diagnosticHealthStub struct{ report HealthReport }

type diagnosticEventServiceStub struct{ items []*domainaudit.Event }

func (s *diagnosticEventServiceStub) AppendEvent(_ context.Context, event *domainaudit.Event) error {
	s.items = append(s.items, event)
	return nil
}

func (s *diagnosticEventServiceStub) GetEventByID(_ context.Context, id string) (*domainaudit.Event, error) {
	for _, item := range s.items {
		if item.ID.String() == id {
			return item, nil
		}
	}
	return nil, nil
}

func (s *diagnosticEventServiceStub) ListEvents(context.Context, auditsvc.EventFilter) ([]*domainaudit.Event, error) {
	return append([]*domainaudit.Event(nil), s.items...), nil
}

func (s *diagnosticEventServiceStub) ExportEventSourceData(context.Context, auditsvc.EventFilter) ([]auditsvc.EventSourceData, error) {
	return []auditsvc.EventSourceData{}, nil
}

func (s diagnosticHealthStub) CheckPluginHealth(context.Context, string) (HealthReport, error) {
	return s.report, nil
}

func (s diagnosticHealthStub) PluginReadiness(context.Context) ReadinessReport {
	return ReadinessReport{Ready: s.report.Ready(), CheckedAt: s.report.CheckedAt, Plugins: []HealthReport{s.report}}
}

func TestDiagnosticsServiceLinksJobsAuditRoutesAndErrors(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 23, 9, 30, 0, 0, time.UTC)
	jobs := jobsvc.NewService(jobsvc.NewMemoryRepository(), func() time.Time { return now })
	if _, err := jobs.Schedule(ctx, jobsvc.ScheduleInput{
		ID: "plugin:pharma_oa:expiry-scan", Namespace: "plugin.pharma_oa", Kind: "expiry_scan",
		Payload: json.RawMessage(`{"scope":"tenant-a"}`), MaxAttempts: 1,
	}); err != nil {
		t.Fatalf("schedule job: %v", err)
	}
	if _, err := jobs.Schedule(ctx, jobsvc.ScheduleInput{
		ID: "plugin:other:unrelated", Namespace: "plugin.other", Kind: "unrelated",
		Payload: json.RawMessage(`{}`), MaxAttempts: 1,
	}); err != nil {
		t.Fatalf("schedule unrelated job: %v", err)
	}
	leased, err := jobs.LeaseDue(ctx, jobsvc.LeaseInput{Namespace: "plugin.pharma_oa", WorkerID: "worker-a", Limit: 1, LeaseDuration: time.Minute})
	if err != nil || len(leased) != 1 {
		t.Fatalf("lease job: items=%d err=%v", len(leased), err)
	}
	if _, err := jobs.Fail(ctx, jobsvc.FailInput{JobID: leased[0].ID, LeaseToken: leased[0].LeaseToken, Error: "qualification lookup failed"}); err != nil {
		t.Fatalf("fail job: %v", err)
	}

	auditService := auditsvc.NewService(clickhouse.NewAuditStore())
	if _, err := auditService.Append(ctx, "operator-1", "plugin.pharma_oa.job.fail", "plugin:pharma_oa:job", "expiry-scan", map[string]any{
		"pluginId": "pharma_oa", "result": "failure", "risk": "high",
	}); err != nil {
		t.Fatalf("append host audit: %v", err)
	}
	eventService := &diagnosticEventServiceStub{}
	action, err := domainaudit.ParseAuditAction("pharma_oa.customer.read")
	if err != nil {
		t.Fatal(err)
	}
	event, err := domainaudit.NewEvent(domainaudit.EventInput{
		ID: shared.ID("audit-route-1"), Type: domainaudit.EventTypePlugin, Action: action,
		Actor:    domainaudit.ActorRef{Type: "user", ID: shared.ID("operator-1")},
		Resource: domainaudit.ResourceRef{Type: "plugin_route", ID: "GET /v1/plugins/pharma_oa/api/customers", Name: "plugin.pharma_oa"},
		Result:   domainaudit.EventResultFailure, Risk: domainaudit.EventRiskMedium,
		Trace:      domainaudit.TraceContext{TraceID: "trace-1", RequestID: "request-1", Method: "GET", Path: "/v1/plugins/pharma_oa/api/customers"},
		OccurredAt: now.Add(-time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := eventService.AppendEvent(ctx, event); err != nil {
		t.Fatal(err)
	}

	service := NewDiagnosticsService(
		diagnosticCatalogStub{items: map[string]Info{"pharma_oa": {ID: "pharma_oa", State: StateEnabled}}},
		jobs, auditService, eventService,
		diagnosticHealthStub{report: HealthReport{PluginID: "pharma_oa", Status: HealthStatusUnhealthy, Code: "health_timeout", CheckedAt: now}},
	)
	service.now = func() time.Time { return now }
	snapshot, err := service.Inspect(ctx, "pharma_oa", DiagnosticQuery{Limit: 50})
	if err != nil {
		t.Fatalf("inspect diagnostics: %v", err)
	}
	if snapshot.Summary.TotalJobs != 1 || snapshot.Summary.DeadLetters != 1 {
		t.Fatalf("unexpected job summary: %+v", snapshot.Summary)
	}
	if len(snapshot.Audit) != 2 {
		t.Fatalf("expected host and route audit records, got %d", len(snapshot.Audit))
	}
	if len(snapshot.Errors) != 4 {
		t.Fatalf("expected process, job, host audit, and route errors, got %+v", snapshot.Errors)
	}
	var linkedRoute bool
	for _, item := range snapshot.Errors {
		if item.Correlation.TraceID == "trace-1" && item.Correlation.RequestID == "request-1" && item.Correlation.AuditID == "audit-route-1" {
			linkedRoute = true
		}
	}
	if !linkedRoute {
		t.Fatal("route error did not preserve trace, request, and audit correlation")
	}
}

func TestDiagnosticsServiceRetriesDeadLetterAsNewJob(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 7, 23, 10, 0, 0, 0, time.UTC)
	jobs := jobsvc.NewService(jobsvc.NewMemoryRepository(), func() time.Time { return now })
	_, _ = jobs.Schedule(ctx, jobsvc.ScheduleInput{ID: "plugin:pharma_oa:scan", Namespace: "plugin.pharma_oa", Kind: "scan", Payload: json.RawMessage(`{}`), MaxAttempts: 1})
	leased, _ := jobs.LeaseDue(ctx, jobsvc.LeaseInput{Namespace: "plugin.pharma_oa", WorkerID: "worker-a", Limit: 1, LeaseDuration: time.Minute})
	_, _ = jobs.Fail(ctx, jobsvc.FailInput{JobID: leased[0].ID, LeaseToken: leased[0].LeaseToken, Error: "failed"})

	service := NewDiagnosticsService(diagnosticCatalogStub{items: map[string]Info{"pharma_oa": {ID: "pharma_oa"}}}, jobs, nil, nil, nil)
	service.now = func() time.Time { return now }
	result, err := service.RetryDeadLetter(ctx, "pharma_oa", "scan")
	if err != nil {
		t.Fatalf("retry dead letter: %v", err)
	}
	if result.SourceJobID != "scan" || result.RetryJob.ID == "scan" || result.RetryJob.Status != string(jobsvc.StatusScheduled) {
		t.Fatalf("unexpected retry result: %+v", result)
	}
	original, err := jobs.Get(ctx, "plugin:pharma_oa:scan")
	if err != nil || original.Status != jobsvc.StatusDeadLetter {
		t.Fatalf("original dead letter was mutated: %+v err=%v", original, err)
	}
}
