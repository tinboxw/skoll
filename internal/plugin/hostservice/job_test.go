package hostservice

import (
	"context"
	"testing"
	"time"

	auditsvc "github.com/tinboxw/skoll/internal/service/audit"
	jobsvc "github.com/tinboxw/skoll/internal/service/job"
	"github.com/tinboxw/skoll/internal/store/clickhouse"
	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

func TestJobServiceLeasesOnlyBoundPluginNamespace(t *testing.T) {
	now := time.Date(2026, 7, 22, 8, 0, 0, 0, time.UTC)
	backend := jobsvc.NewService(jobsvc.NewMemoryRepository(), func() time.Time { return now })
	auditBackend := auditsvc.NewService(clickhouse.NewAuditStore())
	auditA, _ := NewAuditService("plugin_a", auditBackend)
	auditB, _ := NewAuditService("plugin_b", auditBackend)
	serviceA, err := NewJobService("plugin_a", backend, auditA)
	if err != nil {
		t.Fatalf("NewJobService A error: %v", err)
	}
	serviceB, err := NewJobService("plugin_b", backend, auditB)
	if err != nil {
		t.Fatalf("NewJobService B error: %v", err)
	}
	operationCtx, err := pluginsdk.WithOperationContext(context.Background(), pluginsdk.OperationContext{
		CorrelationID: "operation-job-42",
		RequestID:     "request-job-42",
	})
	if err != nil {
		t.Fatal(err)
	}
	scheduled, err := serviceA.Schedule(operationCtx, pluginsdk.JobScheduleInput{ID: "sync", Kind: "sync", MaxAttempts: 1})
	if err != nil {
		t.Fatalf("Schedule A error: %v", err)
	}
	if scheduled.CorrelationID != "operation-job-42" {
		t.Fatalf("scheduled correlation=%q", scheduled.CorrelationID)
	}
	if _, err := serviceB.Schedule(context.Background(), pluginsdk.JobScheduleInput{ID: "sync", Kind: "sync", MaxAttempts: 1}); err != nil {
		t.Fatalf("Schedule B error: %v", err)
	}
	leased, err := serviceA.LeaseDue(context.Background(), pluginsdk.JobLeaseInput{WorkerID: "worker", Limit: 10, LeaseDuration: time.Minute})
	if err != nil || len(leased) != 1 || leased[0].ID != "sync" || leased[0].LeaseOwner != "worker" {
		t.Fatalf("LeaseDue A items=%+v err=%v", leased, err)
	}
	if leased[0].CorrelationID != "operation-job-42" {
		t.Fatalf("leased correlation=%q", leased[0].CorrelationID)
	}
	remaining, err := serviceB.List(context.Background(), pluginsdk.JobQuery{Status: pluginsdk.JobStatusScheduled})
	if err != nil || len(remaining) != 1 {
		t.Fatalf("plugin B job was leased by A: items=%+v err=%v", remaining, err)
	}
}
