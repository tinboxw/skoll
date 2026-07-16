package pharmaoa

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	domainaudit "github.com/tinboxw/skoll/internal/domain/audit"
	domainfile "github.com/tinboxw/skoll/internal/domain/file"
	"github.com/tinboxw/skoll/internal/domain/shared"
	filesvc "github.com/tinboxw/skoll/internal/service/file"
)

type reportMetricsFixture struct {
	snapshot *BusinessMetricsSnapshot
	err      error
	started  chan struct{}
	release  chan struct{}
}

func (f *reportMetricsFixture) Get(_ context.Context, _ BusinessMetricsInput) (*BusinessMetricsSnapshot, error) {
	if f.started != nil {
		close(f.started)
		<-f.release
	}
	if f.err != nil {
		return nil, f.err
	}
	return f.snapshot, nil
}

type reportFileFixture struct {
	mu     sync.Mutex
	input  filesvc.UploadInput
	body   string
	err    error
	fileID string
}

func (f *reportFileFixture) Upload(_ context.Context, in filesvc.UploadInput) (*domainfile.FileObject, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.input = in
	raw, _ := io.ReadAll(in.Body)
	f.body = string(raw)
	if f.err != nil {
		return nil, f.err
	}
	return &domainfile.FileObject{ID: shared.ID(f.fileID)}, nil
}

type reportAuditFixture struct {
	mu      sync.Mutex
	actions []string
}

func (f *reportAuditFixture) Append(_ context.Context, _, action, _, _ string, _ map[string]any) (*domainaudit.Record, error) {
	f.mu.Lock()
	f.actions = append(f.actions, action)
	f.mu.Unlock()
	return &domainaudit.Record{}, nil
}
func (*reportAuditFixture) GetByID(context.Context, string) (*domainaudit.Record, error) {
	return nil, nil
}
func (*reportAuditFixture) ListByActor(context.Context, string, int) ([]*domainaudit.Record, error) {
	return nil, nil
}
func (*reportAuditFixture) ListByTimeRange(context.Context, time.Time, time.Time, int) ([]*domainaudit.Record, error) {
	return nil, nil
}
func (*reportAuditFixture) ClearByTimeRange(context.Context, time.Time, time.Time) (int, error) {
	return 0, nil
}

func reportSnapshot() *BusinessMetricsSnapshot {
	from := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	return &BusinessMetricsSnapshot{
		Window:             BusinessMetricsWindow{From: from, To: from.AddDate(0, 0, 1), Bucket: BusinessMetricsBucketDay},
		StockAlerts:        BusinessMetricsStockAlerts{Active: 3, LowStock: 1, NearExpiry: 2},
		Qualifications:     BusinessMetricsQualifications{Expired: 2, Expiring: 4},
		ApprovalEfficiency: BusinessMetricsApprovalEfficiency{Pending: 5, CompletionRate: 75.5, AverageCompletionHours: 6.25},
		CustomerFollowUps:  BusinessMetricsCustomerFollowUps{Overdue: 3, CompletionRate: 60},
		SalesTrend:         BusinessMetricsSalesTrend{OrderCount: 2, AmountCents: 3234, Series: []BusinessMetricsSalesPoint{{StartedAt: from, OrderCount: 2, AmountCents: 3234}}},
		GeneratedAt:        from.Add(time.Hour),
	}
}

func synchronousReportService(metrics BusinessMetricsService, files ReportExportFileStore, audit *reportAuditFixture) *reportExportService {
	service := NewReportExportService(metrics, files, audit).(*reportExportService)
	service.nowFn = func() time.Time { return time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC) }
	service.dispatch = func(run func()) { run() }
	return service
}

func TestReportExportStoresPrivateExactCentCSVAndAuditsLifecycle(t *testing.T) {
	files := &reportFileFixture{fileID: "file-report-1"}
	audit := &reportAuditFixture{}
	service := synchronousReportService(&reportMetricsFixture{snapshot: reportSnapshot()}, files, audit)
	queued, err := service.Queue(context.Background(), ReportExportCreateInput{ReportType: ReportExportBusinessMetrics, ActorID: "manager-1"})
	if err != nil || queued.Status != ReportExportPending {
		t.Fatalf("queue: job=%+v err=%v", queued, err)
	}
	job, err := service.Get(context.Background(), queued.ID, "manager-1")
	if err != nil || job.Status != ReportExportSucceeded || job.FileID != "file-report-1" || job.RowCount != 14 {
		t.Fatalf("completed job: %+v err=%v", job, err)
	}
	if !strings.Contains(files.body, "sales,amount,3234,cents") {
		t.Fatalf("exact cent row missing: %s", files.body)
	}
	if files.input.Visibility != domainfile.VisibilityPrivate || files.input.Owner.ID != "manager-1" || files.input.Source.PluginID != "pharma_oa" {
		t.Fatalf("unsafe file metadata: %+v", files.input)
	}
	download, err := service.Download(context.Background(), queued.ID, "manager-1")
	if err != nil || string(download.Body) != files.body || download.ContentType != "text/csv" {
		t.Fatalf("download mismatch: %+v err=%v", download, err)
	}
	audit.mu.Lock()
	actions := strings.Join(audit.actions, ",")
	audit.mu.Unlock()
	for _, action := range []string{"pharma_oa.report_export.queue", "pharma_oa.report_export.run", "pharma_oa.report_export.complete", "pharma_oa.report_export.read", "pharma_oa.report_export.download"} {
		if !strings.Contains(actions, action) {
			t.Fatalf("missing audit %s in %s", action, actions)
		}
	}
}

func TestReportExportReturnsBeforeBackgroundWorkCompletes(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})
	service := NewReportExportService(&reportMetricsFixture{snapshot: reportSnapshot(), started: started, release: release}, &reportFileFixture{fileID: "file-async"}, nil)
	queued, err := service.Queue(context.Background(), ReportExportCreateInput{ReportType: ReportExportSalesTrend, ActorID: "manager-1"})
	if err != nil || queued.Status != ReportExportPending {
		t.Fatalf("queue: %+v err=%v", queued, err)
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("background report did not start")
	}
	running, err := service.Get(context.Background(), queued.ID, "manager-1")
	if err != nil || running.Status != ReportExportRunning {
		t.Fatalf("expected running job: %+v err=%v", running, err)
	}
	close(release)
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		job, _ := service.Get(context.Background(), queued.ID, "manager-1")
		if job.Status == ReportExportSucceeded {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("background report did not complete")
}

func TestReportExportFailedJobRetriesAndRejectsCrossOwnerAccess(t *testing.T) {
	files := &reportFileFixture{fileID: "file-retry", err: errors.New("storage offline")}
	audit := &reportAuditFixture{}
	service := synchronousReportService(&reportMetricsFixture{snapshot: reportSnapshot()}, files, audit)
	queued, err := service.Queue(context.Background(), ReportExportCreateInput{ReportType: ReportExportOperationalRisks, ActorID: "manager-1"})
	if err != nil {
		t.Fatal(err)
	}
	failed, _ := service.Get(context.Background(), queued.ID, "manager-1")
	if failed.Status != ReportExportFailed || failed.Error == "" {
		t.Fatalf("expected failed job: %+v", failed)
	}
	if _, err := service.Get(context.Background(), queued.ID, "other-user"); !errors.Is(err, ErrReportExportAccessDenied) {
		t.Fatalf("expected access denied, got %v", err)
	}
	files.err = nil
	if _, err := service.Retry(context.Background(), queued.ID, "manager-1"); err != nil {
		t.Fatal(err)
	}
	completed, _ := service.Get(context.Background(), queued.ID, "manager-1")
	if completed.Status != ReportExportSucceeded || completed.RetryCount != 1 {
		t.Fatalf("retry did not complete: %+v", completed)
	}
	if _, err := service.Retry(context.Background(), queued.ID, "manager-1"); !errors.Is(err, ErrReportExportRetryState) {
		t.Fatalf("expected terminal retry rejection, got %v", err)
	}
}

func TestReportExportRejectsUnsupportedOrUnboundedInput(t *testing.T) {
	service := synchronousReportService(&reportMetricsFixture{snapshot: reportSnapshot()}, &reportFileFixture{fileID: "file"}, nil)
	now := time.Now().UTC()
	for _, input := range []ReportExportCreateInput{
		{ReportType: "unknown", ActorID: "manager"},
		{ReportType: ReportExportBusinessMetrics, From: now.AddDate(-2, 0, 0), To: now, ActorID: "manager"},
		{ReportType: ReportExportBusinessMetrics},
	} {
		if _, err := service.Queue(context.Background(), input); err == nil {
			t.Fatalf("expected validation error for %+v", input)
		}
	}
}
