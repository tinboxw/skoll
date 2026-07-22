package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	domainaudit "github.com/tinboxw/skoll/internal/domain/audit"
	auditsvc "github.com/tinboxw/skoll/internal/service/audit"
	jobsvc "github.com/tinboxw/skoll/internal/service/job"
)

const (
	diagnosticDefaultLimit = 50
	diagnosticMaxLimit     = 200
)

type DiagnosticQuery struct {
	JobStatus   jobsvc.Status
	AuditResult string
	Correlation string
	Limit       int
}

type DiagnosticSnapshot struct {
	PluginID   string                  `json:"pluginId"`
	CapturedAt time.Time               `json:"capturedAt"`
	Health     HealthReport            `json:"health"`
	Summary    DiagnosticSummary       `json:"summary"`
	Jobs       []DiagnosticJob         `json:"jobs"`
	Audit      []DiagnosticAuditRecord `json:"audit"`
	Errors     []DiagnosticError       `json:"errors"`
}

type DiagnosticSummary struct {
	TotalJobs    int `json:"totalJobs"`
	ActiveJobs   int `json:"activeJobs"`
	DeadLetters  int `json:"deadLetters"`
	AuditEvents  int `json:"auditEvents"`
	FailureCount int `json:"failureCount"`
}

type DiagnosticJob struct {
	ID             string     `json:"id"`
	Kind           string     `json:"kind"`
	Status         string     `json:"status"`
	RunAt          time.Time  `json:"runAt"`
	MaxAttempts    int        `json:"maxAttempts"`
	AttemptCount   int        `json:"attemptCount"`
	LastError      string     `json:"lastError,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
	CompletedAt    *time.Time `json:"completedAt,omitempty"`
	DeadLetteredAt *time.Time `json:"deadLetteredAt,omitempty"`
	CanRetry       bool       `json:"canRetry"`
}

type DiagnosticAuditRecord struct {
	ID           string         `json:"id"`
	Source       string         `json:"source"`
	Action       string         `json:"action"`
	Result       string         `json:"result"`
	Risk         string         `json:"risk"`
	ActorID      string         `json:"actorId"`
	ResourceType string         `json:"resourceType"`
	ResourceID   string         `json:"resourceId"`
	OccurredAt   time.Time      `json:"occurredAt"`
	TraceID      string         `json:"traceId,omitempty"`
	RequestID    string         `json:"requestId,omitempty"`
	Method       string         `json:"method,omitempty"`
	Path         string         `json:"path,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

type DiagnosticCorrelation struct {
	ProcessID string `json:"processId,omitempty"`
	RouteID   string `json:"routeId,omitempty"`
	JobID     string `json:"jobId,omitempty"`
	AuditID   string `json:"auditId,omitempty"`
	RequestID string `json:"requestId,omitempty"`
	TraceID   string `json:"traceId,omitempty"`
}

type DiagnosticError struct {
	ID          string                `json:"id"`
	Category    string                `json:"category"`
	Severity    string                `json:"severity"`
	Summary     string                `json:"summary"`
	OccurredAt  time.Time             `json:"occurredAt"`
	Correlation DiagnosticCorrelation `json:"correlation"`
}

type DiagnosticRetryResult struct {
	OperationID string        `json:"operationId"`
	CompletedAt time.Time     `json:"completedAt"`
	SourceJobID string        `json:"sourceJobId"`
	RetryJob    DiagnosticJob `json:"retryJob"`
}

type diagnosticPluginCatalog interface {
	Get(pluginID string) (Info, error)
}

type diagnosticJobService interface {
	Schedule(context.Context, jobsvc.ScheduleInput) (jobsvc.Job, error)
	Get(context.Context, string) (jobsvc.Job, error)
	List(context.Context, jobsvc.Filter) ([]jobsvc.Job, error)
}

type DiagnosticsService struct {
	plugins diagnosticPluginCatalog
	jobs    diagnosticJobService
	audit   auditsvc.Service
	events  auditsvc.EventService
	health  HealthProvider
	now     func() time.Time
}

func NewDiagnosticsService(plugins diagnosticPluginCatalog, jobs diagnosticJobService, audit auditsvc.Service, events auditsvc.EventService, health HealthProvider) *DiagnosticsService {
	return &DiagnosticsService{plugins: plugins, jobs: jobs, audit: audit, events: events, health: health, now: func() time.Time { return time.Now().UTC() }}
}

func (s *DiagnosticsService) Inspect(ctx context.Context, pluginID string, query DiagnosticQuery) (DiagnosticSnapshot, error) {
	pluginID, err := s.validatePlugin(pluginID)
	if err != nil {
		return DiagnosticSnapshot{}, err
	}
	if s.jobs == nil || s.audit == nil || s.events == nil || s.health == nil {
		return DiagnosticSnapshot{}, errors.New("plugin diagnostics dependencies are not configured")
	}
	limit, err := diagnosticLimit(query.Limit)
	if err != nil {
		return DiagnosticSnapshot{}, err
	}
	if query.JobStatus != "" && !validDiagnosticJobStatus(query.JobStatus) {
		return DiagnosticSnapshot{}, fmt.Errorf("invalid job status")
	}
	query.AuditResult = strings.ToLower(strings.TrimSpace(query.AuditResult))
	if query.AuditResult != "" && query.AuditResult != "success" && query.AuditResult != "failure" && query.AuditResult != "denied" {
		return DiagnosticSnapshot{}, fmt.Errorf("invalid audit result")
	}

	capturedAt := s.now().UTC()
	health, healthErr := s.health.CheckPluginHealth(ctx, pluginID)
	if healthErr != nil {
		return DiagnosticSnapshot{}, healthErr
	}
	jobs, err := s.jobs.List(ctx, jobsvc.Filter{Namespace: pluginJobNamespace(pluginID), Status: query.JobStatus, Limit: limit})
	if err != nil {
		return DiagnosticSnapshot{}, err
	}
	auditItems, err := s.auditRecords(ctx, pluginID, capturedAt, limit)
	if err != nil {
		return DiagnosticSnapshot{}, err
	}
	eventItems, err := s.eventRecords(ctx, pluginID, limit)
	if err != nil {
		return DiagnosticSnapshot{}, err
	}
	auditItems = append(auditItems, eventItems...)
	sort.SliceStable(auditItems, func(i, j int) bool { return auditItems[i].OccurredAt.After(auditItems[j].OccurredAt) })
	if len(auditItems) > limit {
		auditItems = auditItems[:limit]
	}
	if query.AuditResult != "" {
		auditItems = filterAuditResult(auditItems, query.AuditResult)
	}

	jobItems := make([]DiagnosticJob, 0, len(jobs))
	for _, item := range jobs {
		jobItems = append(jobItems, diagnosticJob(pluginID, item))
	}
	errorsFound := diagnosticErrors(pluginID, health, jobItems, auditItems)
	correlation := strings.ToLower(strings.TrimSpace(query.Correlation))
	if correlation != "" {
		jobItems = filterDiagnosticJobs(jobItems, correlation)
		auditItems = filterDiagnosticAudit(auditItems, correlation)
		errorsFound = filterDiagnosticErrors(errorsFound, correlation)
	}

	return DiagnosticSnapshot{
		PluginID: pluginID, CapturedAt: capturedAt, Health: health,
		Summary: diagnosticSummary(jobItems, auditItems, errorsFound),
		Jobs:    jobItems, Audit: auditItems, Errors: errorsFound,
	}, nil
}

func (s *DiagnosticsService) RetryDeadLetter(ctx context.Context, pluginID, localJobID string) (DiagnosticRetryResult, error) {
	pluginID, err := s.validatePlugin(pluginID)
	if err != nil {
		return DiagnosticRetryResult{}, err
	}
	if s.jobs == nil {
		return DiagnosticRetryResult{}, errors.New("plugin job diagnostics are not configured")
	}
	localJobID = strings.TrimSpace(localJobID)
	if localJobID == "" || strings.ContainsAny(localJobID, "/\\") {
		return DiagnosticRetryResult{}, fmt.Errorf("invalid job id")
	}
	source, err := s.jobs.Get(ctx, pluginJobID(pluginID, localJobID))
	if err != nil {
		return DiagnosticRetryResult{}, err
	}
	if source.Namespace != pluginJobNamespace(pluginID) {
		return DiagnosticRetryResult{}, fmt.Errorf("job is outside plugin namespace")
	}
	if source.Status != jobsvc.StatusDeadLetter {
		return DiagnosticRetryResult{}, fmt.Errorf("only dead-letter jobs can be retried")
	}
	now := s.now().UTC()
	retryLocalID := fmt.Sprintf("%s-retry-%d", localJobID, now.UnixMilli())
	retried, err := s.jobs.Schedule(ctx, jobsvc.ScheduleInput{
		ID: pluginJobID(pluginID, retryLocalID), Namespace: source.Namespace, Kind: source.Kind,
		Payload: append(json.RawMessage(nil), source.Payload...), RunAt: now, MaxAttempts: source.MaxAttempts,
	})
	if err != nil {
		return DiagnosticRetryResult{}, err
	}
	return DiagnosticRetryResult{
		OperationID: fmt.Sprintf("plugin-job-retry-%d", now.UnixNano()), CompletedAt: now,
		SourceJobID: localJobID, RetryJob: diagnosticJob(pluginID, retried),
	}, nil
}

func (s *DiagnosticsService) validatePlugin(pluginID string) (string, error) {
	pluginID = strings.ToLower(strings.TrimSpace(pluginID))
	if pluginID == "" {
		return "", ErrPluginNotFound
	}
	if s == nil || s.plugins == nil {
		return "", errors.New("plugin diagnostics catalog is not configured")
	}
	if _, err := s.plugins.Get(pluginID); err != nil {
		return "", err
	}
	return pluginID, nil
}

func (s *DiagnosticsService) auditRecords(ctx context.Context, pluginID string, now time.Time, limit int) ([]DiagnosticAuditRecord, error) {
	items, err := s.audit.ListByTimeRange(ctx, now.Add(-90*24*time.Hour), now, diagnosticMaxLimit)
	if err != nil {
		return nil, err
	}
	out := make([]DiagnosticAuditRecord, 0, min(len(items), limit))
	for _, item := range items {
		if item == nil || !diagnosticRecordBelongsToPlugin(item.Action, item.Resource, item.Detail, pluginID) {
			continue
		}
		result, _ := item.Detail["result"].(string)
		risk, _ := item.Detail["risk"].(string)
		out = append(out, DiagnosticAuditRecord{
			ID: item.ID.String(), Source: "host", Action: item.Action, Result: result, Risk: risk,
			ActorID: item.ActorID.String(), ResourceType: item.Resource, ResourceID: item.ResourceID,
			OccurredAt: item.OccurredAt, Metadata: cloneDiagnosticMap(item.Detail),
		})
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (s *DiagnosticsService) eventRecords(ctx context.Context, pluginID string, limit int) ([]DiagnosticAuditRecord, error) {
	items, err := s.events.ListEvents(ctx, auditsvc.EventFilter{Type: domainaudit.EventTypePlugin, Limit: diagnosticMaxLimit})
	if err != nil {
		return nil, err
	}
	out := make([]DiagnosticAuditRecord, 0, min(len(items), limit))
	for _, item := range items {
		if item == nil || !diagnosticEventBelongsToPlugin(item, pluginID) {
			continue
		}
		out = append(out, DiagnosticAuditRecord{
			ID: item.ID.String(), Source: "event", Action: item.Action.String(), Result: string(item.Result), Risk: string(item.Risk),
			ActorID: item.Actor.ID.String(), ResourceType: item.Resource.Type, ResourceID: item.Resource.ID,
			OccurredAt: item.OccurredAt, TraceID: item.Trace.TraceID, RequestID: item.Trace.RequestID,
			Method: item.Trace.Method, Path: item.Trace.Path, Metadata: cloneDiagnosticMap(item.Metadata),
		})
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func diagnosticRecordBelongsToPlugin(action, resource string, detail map[string]any, pluginID string) bool {
	if value, _ := detail["pluginId"].(string); strings.EqualFold(strings.TrimSpace(value), pluginID) {
		return true
	}
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(action)), "plugin."+pluginID+".") ||
		strings.HasPrefix(strings.ToLower(strings.TrimSpace(resource)), "plugin:"+pluginID+":")
}

func diagnosticEventBelongsToPlugin(item *domainaudit.Event, pluginID string) bool {
	if item.Resource.Type == "plugin" && strings.EqualFold(item.Resource.ID, pluginID) {
		return true
	}
	if item.Action.Module() == pluginID || strings.Contains(strings.ToLower(item.Trace.Path), "/plugins/"+pluginID+"/") {
		return true
	}
	for _, source := range []map[string]any{item.Metadata, item.SourceData} {
		if value, _ := source["pluginId"].(string); strings.EqualFold(strings.TrimSpace(value), pluginID) {
			return true
		}
		if value, _ := source["source"].(string); strings.EqualFold(strings.TrimPrefix(strings.TrimSpace(value), "plugin."), pluginID) {
			return true
		}
	}
	return false
}

func diagnosticJob(pluginID string, item jobsvc.Job) DiagnosticJob {
	return DiagnosticJob{
		ID: strings.TrimPrefix(item.ID, pluginJobID(pluginID, "")), Kind: item.Kind, Status: string(item.Status), RunAt: item.RunAt,
		MaxAttempts: item.MaxAttempts, AttemptCount: item.AttemptCount, LastError: item.LastError,
		CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt, CompletedAt: item.CompletedAt, DeadLetteredAt: item.DeadLetteredAt,
		CanRetry: item.Status == jobsvc.StatusDeadLetter,
	}
}

func diagnosticErrors(pluginID string, health HealthReport, jobs []DiagnosticJob, audit []DiagnosticAuditRecord) []DiagnosticError {
	out := make([]DiagnosticError, 0)
	if !health.Ready() {
		out = append(out, DiagnosticError{
			ID: "process:" + pluginID + ":" + health.CheckedAt.UTC().Format(time.RFC3339Nano), Category: "process", Severity: "critical",
			Summary: health.Code, OccurredAt: health.CheckedAt, Correlation: DiagnosticCorrelation{ProcessID: pluginID},
		})
	}
	for _, item := range jobs {
		if item.Status != string(jobsvc.StatusDeadLetter) && strings.TrimSpace(item.LastError) == "" {
			continue
		}
		out = append(out, DiagnosticError{
			ID: "job:" + item.ID + ":" + item.UpdatedAt.UTC().Format(time.RFC3339Nano), Category: "job", Severity: "high",
			Summary: item.LastError, OccurredAt: item.UpdatedAt, Correlation: DiagnosticCorrelation{JobID: item.ID},
		})
	}
	for _, item := range audit {
		if item.Result != "failure" && item.Result != "denied" {
			continue
		}
		category := "audit"
		routeID := ""
		if item.ResourceType == "plugin_route" || item.Path != "" {
			category = "route"
			routeID = strings.TrimSpace(item.Method + " " + item.Path)
		}
		summary := item.Action
		if value, _ := item.Metadata["error"].(string); strings.TrimSpace(value) != "" {
			summary = strings.TrimSpace(value)
		}
		out = append(out, DiagnosticError{
			ID: "audit:" + item.ID, Category: category, Severity: diagnosticAuditSeverity(item.Risk), Summary: summary,
			OccurredAt: item.OccurredAt, Correlation: DiagnosticCorrelation{RouteID: routeID, AuditID: item.ID, RequestID: item.RequestID, TraceID: item.TraceID},
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].OccurredAt.After(out[j].OccurredAt) })
	return out
}

func diagnosticSummary(jobs []DiagnosticJob, audit []DiagnosticAuditRecord, errorsFound []DiagnosticError) DiagnosticSummary {
	summary := DiagnosticSummary{TotalJobs: len(jobs), AuditEvents: len(audit), FailureCount: len(errorsFound)}
	for _, item := range jobs {
		switch item.Status {
		case string(jobsvc.StatusScheduled), string(jobsvc.StatusRunning), string(jobsvc.StatusRetryWait):
			summary.ActiveJobs++
		case string(jobsvc.StatusDeadLetter):
			summary.DeadLetters++
		}
	}
	return summary
}

func diagnosticAuditSeverity(risk string) string {
	switch risk {
	case "critical", "high":
		return risk
	default:
		return "medium"
	}
}

func filterAuditResult(items []DiagnosticAuditRecord, result string) []DiagnosticAuditRecord {
	out := make([]DiagnosticAuditRecord, 0, len(items))
	for _, item := range items {
		if item.Result == result {
			out = append(out, item)
		}
	}
	return out
}

func filterDiagnosticJobs(items []DiagnosticJob, query string) []DiagnosticJob {
	out := make([]DiagnosticJob, 0, len(items))
	for _, item := range items {
		if diagnosticContains(query, item.ID, item.Kind, item.Status, item.LastError) {
			out = append(out, item)
		}
	}
	return out
}

func filterDiagnosticAudit(items []DiagnosticAuditRecord, query string) []DiagnosticAuditRecord {
	out := make([]DiagnosticAuditRecord, 0, len(items))
	for _, item := range items {
		if diagnosticContains(query, item.ID, item.Action, item.ResourceID, item.TraceID, item.RequestID, item.Method, item.Path) {
			out = append(out, item)
		}
	}
	return out
}

func filterDiagnosticErrors(items []DiagnosticError, query string) []DiagnosticError {
	out := make([]DiagnosticError, 0, len(items))
	for _, item := range items {
		correlation := item.Correlation
		if diagnosticContains(query, item.ID, item.Category, item.Summary, correlation.ProcessID, correlation.RouteID, correlation.JobID, correlation.AuditID, correlation.RequestID, correlation.TraceID) {
			out = append(out, item)
		}
	}
	return out
}

func diagnosticContains(query string, values ...string) bool {
	for _, value := range values {
		if strings.Contains(strings.ToLower(value), query) {
			return true
		}
	}
	return false
}

func diagnosticLimit(value int) (int, error) {
	if value == 0 {
		return diagnosticDefaultLimit, nil
	}
	if value < 1 || value > diagnosticMaxLimit {
		return 0, fmt.Errorf("diagnostic limit must be between 1 and %d", diagnosticMaxLimit)
	}
	return value, nil
}

func validDiagnosticJobStatus(status jobsvc.Status) bool {
	switch status {
	case jobsvc.StatusScheduled, jobsvc.StatusRunning, jobsvc.StatusRetryWait, jobsvc.StatusSucceeded, jobsvc.StatusDeadLetter:
		return true
	default:
		return false
	}
}

func pluginJobNamespace(pluginID string) string   { return "plugin." + pluginID }
func pluginJobID(pluginID, localID string) string { return "plugin:" + pluginID + ":" + localID }

func cloneDiagnosticMap(values map[string]any) map[string]any {
	out := make(map[string]any, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}
