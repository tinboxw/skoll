package pharmaoa

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	domainaudit "github.com/tinboxw/skoll/internal/domain/audit"
	domainfile "github.com/tinboxw/skoll/internal/domain/file"
	"github.com/tinboxw/skoll/internal/domain/shared"
	auditsvc "github.com/tinboxw/skoll/internal/service/audit"
	filesvc "github.com/tinboxw/skoll/internal/service/file"
)

type ReportExportType string
type ReportExportJobStatus string

const (
	ReportExportBusinessMetrics  ReportExportType = "business_metrics"
	ReportExportSalesTrend       ReportExportType = "sales_trend"
	ReportExportOperationalRisks ReportExportType = "operational_risks"

	ReportExportPending   ReportExportJobStatus = "pending"
	ReportExportRunning   ReportExportJobStatus = "running"
	ReportExportSucceeded ReportExportJobStatus = "succeeded"
	ReportExportFailed    ReportExportJobStatus = "failed"
)

var (
	ErrReportExportNotFound     = errors.New("report export job not found")
	ErrReportExportAccessDenied = errors.New("report export job access denied")
	ErrReportExportNotReady     = errors.New("report export is not ready")
	ErrReportExportRetryState   = errors.New("only failed report exports can be retried")
)

type ReportExportQuery struct {
	From              time.Time             `json:"from"`
	To                time.Time             `json:"to"`
	Bucket            BusinessMetricsBucket `json:"bucket"`
	QualificationDays int                   `json:"qualificationDays"`
}

type ReportExportJobLog struct {
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"createdAt"`
}

type ReportExportJob struct {
	ID          string                `json:"id"`
	ReportType  ReportExportType      `json:"reportType"`
	Status      ReportExportJobStatus `json:"status"`
	Query       ReportExportQuery     `json:"query"`
	OwnerID     string                `json:"ownerId"`
	FileID      string                `json:"fileId"`
	Filename    string                `json:"filename"`
	ContentType string                `json:"contentType"`
	Size        int64                 `json:"size"`
	RowCount    int                   `json:"rowCount"`
	Error       string                `json:"error"`
	RetryCount  int                   `json:"retryCount"`
	Logs        []ReportExportJobLog  `json:"logs"`
	CreatedAt   time.Time             `json:"createdAt"`
	StartedAt   *time.Time            `json:"startedAt,omitempty"`
	CompletedAt *time.Time            `json:"completedAt,omitempty"`
}

type ReportExportCreateInput struct {
	ReportType        ReportExportType
	From              time.Time
	To                time.Time
	Bucket            BusinessMetricsBucket
	QualificationDays int
	ActorID           string
}

type ReportExportFile struct {
	Filename    string
	ContentType string
	Body        []byte
}

type ReportExportService interface {
	Queue(ctx context.Context, in ReportExportCreateInput) (*ReportExportJob, error)
	List(ctx context.Context, actorID string) ([]*ReportExportJob, error)
	Get(ctx context.Context, id, actorID string) (*ReportExportJob, error)
	Retry(ctx context.Context, id, actorID string) (*ReportExportJob, error)
	Download(ctx context.Context, id, actorID string) (*ReportExportFile, error)
}

type ReportExportFileStore interface {
	Upload(ctx context.Context, in filesvc.UploadInput) (*domainfile.FileObject, error)
}

type reportExportService struct {
	mu       sync.RWMutex
	jobs     map[string]*ReportExportJob
	content  map[string][]byte
	metrics  BusinessMetricsService
	files    ReportExportFileStore
	audit    auditsvc.Service
	nowFn    func() time.Time
	dispatch func(func())
	counter  int64
}

func NewReportExportService(metrics BusinessMetricsService, files ReportExportFileStore, audit auditsvc.Service) ReportExportService {
	return &reportExportService{
		jobs: map[string]*ReportExportJob{}, content: map[string][]byte{}, metrics: metrics, files: files, audit: audit,
		nowFn: func() time.Time { return time.Now().UTC() }, dispatch: func(run func()) { go run() },
	}
}

func (s *reportExportService) Queue(ctx context.Context, in ReportExportCreateInput) (*ReportExportJob, error) {
	if s == nil || s.metrics == nil || s.files == nil {
		return nil, fmt.Errorf("report export dependencies are required")
	}
	metricsInput, err := s.normalizeInput(in)
	if err != nil {
		return nil, err
	}
	now := s.nowFn().UTC()
	s.mu.Lock()
	s.counter++
	id := fmt.Sprintf("report-export-%d-%d", now.UnixNano(), s.counter)
	job := &ReportExportJob{
		ID: id, ReportType: in.ReportType, Status: ReportExportPending,
		Query:   ReportExportQuery{From: metricsInput.From, To: metricsInput.To, Bucket: metricsInput.Bucket, QualificationDays: metricsInput.QualificationDays},
		OwnerID: metricsInput.ActorID, ContentType: "text/csv", CreatedAt: now,
		Logs: []ReportExportJobLog{{Level: "info", Message: "report export queued", CreatedAt: now}},
	}
	s.jobs[id] = cloneReportExportJob(job)
	s.mu.Unlock()
	s.appendAudit(ctx, in.ActorID, "pharma_oa.report_export.queue", id, map[string]any{"reportType": in.ReportType, "bucket": metricsInput.Bucket})
	asyncContext := context.WithoutCancel(ctx)
	s.dispatch(func() { s.execute(asyncContext, id, false) })
	return cloneReportExportJob(job), nil
}

func (s *reportExportService) List(ctx context.Context, actorID string) ([]*ReportExportJob, error) {
	actorID = strings.TrimSpace(actorID)
	if actorID == "" {
		return nil, fmt.Errorf("actorId is required")
	}
	s.mu.RLock()
	out := make([]*ReportExportJob, 0, len(s.jobs))
	for _, job := range s.jobs {
		if job.OwnerID == actorID {
			out = append(out, cloneReportExportJob(job))
		}
	}
	s.mu.RUnlock()
	sortReportExportJobs(out)
	s.appendAudit(ctx, actorID, "pharma_oa.report_export.read_list", "report-export-jobs", map[string]any{"count": len(out)})
	return out, nil
}

func (s *reportExportService) Get(ctx context.Context, id, actorID string) (*ReportExportJob, error) {
	job, err := s.getOwned(id, actorID)
	if err != nil {
		return nil, err
	}
	s.appendAudit(ctx, actorID, "pharma_oa.report_export.read", job.ID, map[string]any{"status": job.Status})
	return job, nil
}

func (s *reportExportService) Retry(ctx context.Context, id, actorID string) (*ReportExportJob, error) {
	job, err := s.getOwned(id, actorID)
	if err != nil {
		return nil, err
	}
	if job.Status != ReportExportFailed {
		return nil, ErrReportExportRetryState
	}
	now := s.nowFn().UTC()
	s.mu.Lock()
	stored := s.jobs[job.ID]
	stored.Status = ReportExportPending
	stored.Error = ""
	stored.FileID, stored.Filename, stored.Size, stored.RowCount = "", "", 0, 0
	stored.StartedAt, stored.CompletedAt = nil, nil
	stored.RetryCount++
	stored.Logs = append(stored.Logs, ReportExportJobLog{Level: "info", Message: "report export retry queued", CreatedAt: now})
	delete(s.content, job.ID)
	retried := cloneReportExportJob(stored)
	s.mu.Unlock()
	s.appendAudit(ctx, actorID, "pharma_oa.report_export.retry", job.ID, map[string]any{"retryCount": retried.RetryCount})
	asyncContext := context.WithoutCancel(ctx)
	s.dispatch(func() { s.execute(asyncContext, job.ID, true) })
	return retried, nil
}

func (s *reportExportService) Download(ctx context.Context, id, actorID string) (*ReportExportFile, error) {
	job, err := s.getOwned(id, actorID)
	if err != nil {
		return nil, err
	}
	if job.Status != ReportExportSucceeded || job.FileID == "" {
		return nil, ErrReportExportNotReady
	}
	s.mu.RLock()
	body, ok := s.content[job.ID]
	s.mu.RUnlock()
	if !ok {
		return nil, ErrReportExportNotReady
	}
	s.appendAudit(ctx, actorID, "pharma_oa.report_export.download", job.ID, map[string]any{"fileId": job.FileID, "size": len(body)})
	return &ReportExportFile{Filename: job.Filename, ContentType: job.ContentType, Body: append([]byte(nil), body...)}, nil
}

func (s *reportExportService) execute(ctx context.Context, id string, retry bool) {
	job, err := s.start(id)
	if err != nil {
		return
	}
	s.appendAudit(ctx, job.OwnerID, "pharma_oa.report_export.run", job.ID, map[string]any{"reportType": job.ReportType, "retry": retry})
	snapshot, err := s.metrics.Get(ctx, BusinessMetricsInput{From: job.Query.From, To: job.Query.To, Bucket: job.Query.Bucket, QualificationDays: job.Query.QualificationDays, ActorID: job.OwnerID})
	if err != nil {
		s.fail(ctx, job.ID, job.OwnerID, err, retry)
		return
	}
	body, rows, err := buildReportCSV(job.ReportType, snapshot)
	if err != nil {
		s.fail(ctx, job.ID, job.OwnerID, err, retry)
		return
	}
	hash := fmt.Sprintf("%x", sha256.Sum256(body))
	filename := fmt.Sprintf("%s_%s.csv", job.ReportType, s.nowFn().UTC().Format("20060102_150405"))
	object, err := s.files.Upload(ctx, filesvc.UploadInput{
		Key: fmt.Sprintf("pharma-oa/reports/%s.csv", job.ID), Name: filename, Size: int64(len(body)), MIME: "text/csv", Hash: hash, Body: bytes.NewReader(body),
		Owner: domainfile.OwnerRef{Type: "user", ID: shared.ID(job.OwnerID)}, Visibility: domainfile.VisibilityPrivate, StorageDriver: "local",
		Source: domainfile.SourceRef{Module: "pharma_oa", PluginID: "pharma_oa"}, Metadata: map[string]string{"job_id": job.ID, "report_type": string(job.ReportType)},
		Actor: domainaudit.ActorRef{Type: "user", ID: shared.ID(job.OwnerID)}, AuditMetadata: map[string]any{"jobId": job.ID, "reportType": job.ReportType},
	})
	if err != nil {
		s.fail(ctx, job.ID, job.OwnerID, err, retry)
		return
	}
	now := s.nowFn().UTC()
	s.mu.Lock()
	stored := s.jobs[job.ID]
	stored.Status, stored.FileID, stored.Filename = ReportExportSucceeded, object.ID.String(), filename
	stored.Size, stored.RowCount, stored.Error = int64(len(body)), rows, ""
	stored.CompletedAt = timePointer(now)
	stored.Logs = append(stored.Logs, ReportExportJobLog{Level: "info", Message: "report export completed", CreatedAt: now})
	s.content[job.ID] = append([]byte(nil), body...)
	s.mu.Unlock()
	s.appendAudit(ctx, job.OwnerID, "pharma_oa.report_export.complete", job.ID, map[string]any{"fileId": object.ID.String(), "rows": rows, "size": len(body), "retry": retry})
}

func (s *reportExportService) start(id string) (*ReportExportJob, error) {
	now := s.nowFn().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	job := s.jobs[id]
	if job == nil {
		return nil, ErrReportExportNotFound
	}
	if job.Status != ReportExportPending {
		return nil, fmt.Errorf("report export is not pending")
	}
	job.Status, job.StartedAt = ReportExportRunning, timePointer(now)
	job.Logs = append(job.Logs, ReportExportJobLog{Level: "info", Message: "report export started", CreatedAt: now})
	return cloneReportExportJob(job), nil
}

func (s *reportExportService) fail(ctx context.Context, id, actorID string, cause error, retry bool) {
	now := s.nowFn().UTC()
	s.mu.Lock()
	job := s.jobs[id]
	if job != nil {
		job.Status, job.Error, job.CompletedAt = ReportExportFailed, cause.Error(), timePointer(now)
		job.Logs = append(job.Logs, ReportExportJobLog{Level: "error", Message: "report export failed", CreatedAt: now})
	}
	s.mu.Unlock()
	s.appendAudit(ctx, actorID, "pharma_oa.report_export.fail", id, map[string]any{"error": cause.Error(), "retry": retry})
}

func (s *reportExportService) getOwned(id, actorID string) (*ReportExportJob, error) {
	id, actorID = strings.TrimSpace(id), strings.TrimSpace(actorID)
	if actorID == "" {
		return nil, fmt.Errorf("actorId is required")
	}
	s.mu.RLock()
	job := cloneReportExportJob(s.jobs[id])
	s.mu.RUnlock()
	if job == nil {
		return nil, ErrReportExportNotFound
	}
	if job.OwnerID != actorID {
		return nil, ErrReportExportAccessDenied
	}
	return job, nil
}

func (s *reportExportService) normalizeInput(in ReportExportCreateInput) (BusinessMetricsInput, error) {
	switch in.ReportType {
	case ReportExportBusinessMetrics, ReportExportSalesTrend, ReportExportOperationalRisks:
	default:
		return BusinessMetricsInput{}, fmt.Errorf("reportType must be business_metrics, sales_trend, or operational_risks")
	}
	metricsInput := BusinessMetricsInput{From: in.From, To: in.To, Bucket: in.Bucket, QualificationDays: in.QualificationDays, ActorID: strings.TrimSpace(in.ActorID)}
	if err := normalizeBusinessMetricsInput(&metricsInput, s.nowFn().UTC()); err != nil {
		return BusinessMetricsInput{}, err
	}
	if metricsInput.ActorID == "" {
		return BusinessMetricsInput{}, fmt.Errorf("actorId is required")
	}
	return metricsInput, nil
}

func (s *reportExportService) appendAudit(ctx context.Context, actorID, action, id string, detail map[string]any) {
	if s.audit != nil {
		_, _ = s.audit.Append(ctx, strings.TrimSpace(actorID), action, "pharma_oa_report_export_job", id, detail)
	}
}

func buildReportCSV(reportType ReportExportType, snapshot *BusinessMetricsSnapshot) ([]byte, int, error) {
	if snapshot == nil {
		return nil, 0, fmt.Errorf("business metrics snapshot is required")
	}
	var rows [][]string
	switch reportType {
	case ReportExportBusinessMetrics:
		rows = [][]string{{"section", "metric", "value", "unit"},
			{"window", "from", snapshot.Window.From.Format(time.RFC3339), ""}, {"window", "to", snapshot.Window.To.Format(time.RFC3339), ""},
			{"stock", "active_alerts", strconv.Itoa(snapshot.StockAlerts.Active), "count"}, {"stock", "low_stock", strconv.Itoa(snapshot.StockAlerts.LowStock), "count"}, {"stock", "over_stock", strconv.Itoa(snapshot.StockAlerts.OverStock), "count"}, {"stock", "near_expiry", strconv.Itoa(snapshot.StockAlerts.NearExpiry), "count"},
			{"qualification", "expired", strconv.Itoa(snapshot.Qualifications.Expired), "count"}, {"qualification", "expiring", strconv.Itoa(snapshot.Qualifications.Expiring), "count"},
			{"approval", "completion_rate", decimal(snapshot.ApprovalEfficiency.CompletionRate), "percent"}, {"approval", "average_completion_hours", decimal(snapshot.ApprovalEfficiency.AverageCompletionHours), "hours"},
			{"follow_up", "completion_rate", decimal(snapshot.CustomerFollowUps.CompletionRate), "percent"}, {"follow_up", "overdue", strconv.Itoa(snapshot.CustomerFollowUps.Overdue), "count"},
			{"sales", "order_count", strconv.Itoa(snapshot.SalesTrend.OrderCount), "count"}, {"sales", "amount", strconv.FormatInt(snapshot.SalesTrend.AmountCents, 10), "cents"}}
	case ReportExportSalesTrend:
		rows = [][]string{{"started_at", "order_count", "amount_cents"}}
		for _, point := range snapshot.SalesTrend.Series {
			rows = append(rows, []string{point.StartedAt.Format(time.RFC3339), strconv.Itoa(point.OrderCount), strconv.FormatInt(point.AmountCents, 10)})
		}
	case ReportExportOperationalRisks:
		rows = [][]string{{"category", "risk", "count"},
			{"stock", "low_stock", strconv.Itoa(snapshot.StockAlerts.LowStock)}, {"stock", "over_stock", strconv.Itoa(snapshot.StockAlerts.OverStock)}, {"stock", "near_expiry", strconv.Itoa(snapshot.StockAlerts.NearExpiry)},
			{"qualification", "expired", strconv.Itoa(snapshot.Qualifications.Expired)}, {"qualification", "expiring", strconv.Itoa(snapshot.Qualifications.Expiring)},
			{"customer_follow_up", "overdue", strconv.Itoa(snapshot.CustomerFollowUps.Overdue)}, {"purchase_approval", "pending", strconv.Itoa(snapshot.ApprovalEfficiency.Pending)}}
	default:
		return nil, 0, fmt.Errorf("unsupported report type")
	}
	var output bytes.Buffer
	writer := csv.NewWriter(&output)
	if err := writer.WriteAll(rows); err != nil {
		return nil, 0, err
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, 0, err
	}
	return output.Bytes(), len(rows) - 1, nil
}

func cloneReportExportJob(job *ReportExportJob) *ReportExportJob {
	if job == nil {
		return nil
	}
	out := *job
	out.Logs = append([]ReportExportJobLog(nil), job.Logs...)
	if job.StartedAt != nil {
		out.StartedAt = timePointer(*job.StartedAt)
	}
	if job.CompletedAt != nil {
		out.CompletedAt = timePointer(*job.CompletedAt)
	}
	return &out
}

func sortReportExportJobs(jobs []*ReportExportJob) {
	for i := 1; i < len(jobs); i++ {
		for j := i; j > 0 && jobs[j].CreatedAt.After(jobs[j-1].CreatedAt); j-- {
			jobs[j], jobs[j-1] = jobs[j-1], jobs[j]
		}
	}
}

func timePointer(value time.Time) *time.Time { return &value }
func decimal(value float64) string           { return strconv.FormatFloat(value, 'f', -1, 64) }
