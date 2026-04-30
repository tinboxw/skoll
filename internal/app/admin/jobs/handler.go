package jobs

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/module/audit"
	"github.com/tinboxw/skoll/internal/module/jobscheduler"
)

// APIRegistry keeps route registration in sync with API discovery/permissions.
type APIRegistry interface {
	RegisterMany(entries []string)
}

type JobService interface {
	Create(name, schedule string) jobscheduler.Job
	Get(id int64) (jobscheduler.Job, error)
	List() []jobscheduler.Job
	Run(jobID int64) (jobscheduler.Execution, error)
	History(jobID int64, limit int) []jobscheduler.Execution
	ClaimRun(jobID int64, executionKey, instanceID string, now time.Time) (jobscheduler.DispatchClaim, error)
	RenewClaimLease(executionKey, instanceID string, leaseTTLSeconds int64, now time.Time) (jobscheduler.DispatchClaim, error)
	ClaimStatus(executionKey string) jobscheduler.DispatchClaim
	SetRetryPolicy(jobID int64, policy jobscheduler.RetryPolicy) (jobscheduler.RetryPolicy, error)
	GetRetryPolicy(jobID int64) (jobscheduler.RetryPolicy, error)
	ScheduleRetry(jobID int64, executionKey string, attempt int, now time.Time) (jobscheduler.RetrySchedule, error)
	MarkDeadLetter(jobID int64, executionKey, reason string, retryCount int, now time.Time) (jobscheduler.DeadLetter, error)
	ListDeadLetters(limit int) []jobscheduler.DeadLetter
	ReplayDeadLetter(executionKey, operator string, now time.Time) (jobscheduler.DeadLetter, error)
	ReliabilitySnapshot(now time.Time) jobscheduler.ReliabilityMetrics
}

type AuditService interface {
	Append(actor, action, target string) audit.Record
}

type Handler struct {
	jobs  JobService
	audit AuditService
}

func NewHandler(jobs JobService, audit AuditService) *Handler {
	return &Handler{jobs: jobs, audit: audit}
}

func (h *Handler) Register(mux *http.ServeMux, wrapper func(http.Handler) http.Handler, apis APIRegistry) {
	if mux == nil || h == nil || h.jobs == nil {
		return
	}

	handle := func(pattern string, next http.HandlerFunc) {
		if apis != nil {
			apis.RegisterMany([]string{pattern})
		}
		h := http.Handler(next)
		if wrapper != nil {
			h = wrapper(h)
		}
		mux.Handle(pattern, h)
	}

	handle("POST /admin/v1/jobs", h.createJob)
	handle("GET /admin/v1/jobs", h.listJobs)
	handle("POST /admin/v1/jobs/{id}/run", h.runJob)
	handle("GET /admin/v1/jobs/{id}/history", h.listJobHistory)
	handle("POST /admin/v1/jobs/{id}/dispatch-claim", h.claimJobDispatch)
	handle("POST /admin/v1/job-dispatch-claims/{execution_key}/renew", h.renewJobDispatchClaim)
	handle("GET /admin/v1/job-dispatch-claims/{execution_key}", h.getJobDispatchClaim)
	handle("PUT /admin/v1/jobs/{id}/retry-policy", h.setJobRetryPolicy)
	handle("GET /admin/v1/jobs/{id}/retry-policy", h.getJobRetryPolicy)
	handle("POST /admin/v1/jobs/{id}/retries/schedule", h.scheduleJobRetry)
	handle("POST /admin/v1/jobs/{id}/dead-letters", h.markJobDeadLetter)
	handle("GET /admin/v1/jobs/dead-letters", h.listJobDeadLetters)
	handle("POST /admin/v1/jobs/dead-letters/{execution_key}/replay", h.replayJobDeadLetter)
	handle("GET /admin/v1/jobs/reliability/metrics", h.jobReliabilityMetrics)
}

func (h *Handler) createJob(w http.ResponseWriter, r *http.Request) {
	var req createJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Schedule = strings.TrimSpace(req.Schedule)
	if req.Name == "" || req.Schedule == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "name and schedule are required"})
		return
	}
	respondJSON(w, http.StatusCreated, h.jobs.Create(req.Name, req.Schedule))
}

func (h *Handler) listJobs(w http.ResponseWriter, _ *http.Request) {
	respondJSON(w, http.StatusOK, h.jobs.List())
}

func (h *Handler) runJob(w http.ResponseWriter, r *http.Request) {
	id, err := parsePathInt64(r, "id")
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	exec, err := h.jobs.Run(id)
	if err != nil {
		if err == jobscheduler.ErrJobNotFound {
			respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	respondJSON(w, http.StatusOK, exec)
}

func (h *Handler) listJobHistory(w http.ResponseWriter, r *http.Request) {
	id, err := parsePathInt64(r, "id")
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if _, err := h.jobs.Get(id); err != nil {
		if err == jobscheduler.ErrJobNotFound {
			respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	limit := 20
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, convErr := strconv.Atoi(raw)
		if convErr != nil || parsed <= 0 {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "limit must be a positive integer"})
			return
		}
		if parsed > 200 {
			parsed = 200
		}
		limit = parsed
	}
	respondJSON(w, http.StatusOK, h.jobs.History(id, limit))
}

func (h *Handler) claimJobDispatch(w http.ResponseWriter, r *http.Request) {
	jobID, err := parsePathInt64(r, "id")
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	var req claimJobDispatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	req.ExecutionKey = strings.TrimSpace(req.ExecutionKey)
	req.InstanceID = strings.TrimSpace(req.InstanceID)
	if req.ExecutionKey == "" || req.InstanceID == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "execution_key and instance_id are required"})
		return
	}

	out, err := h.jobs.ClaimRun(jobID, req.ExecutionKey, req.InstanceID, time.Now().UTC())
	if err != nil {
		if err == jobscheduler.ErrJobNotFound {
			respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	action := "job_dispatch_claim"
	if out.DuplicateBlocked {
		action = "job_dispatch_duplicate_blocked"
	}
	h.appendAudit("consistency", action, req.ExecutionKey)
	respondJSON(w, http.StatusOK, out)
}

func (h *Handler) getJobDispatchClaim(w http.ResponseWriter, r *http.Request) {
	executionKey, err := parsePathString(r, "execution_key")
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	respondJSON(w, http.StatusOK, h.jobs.ClaimStatus(executionKey))
}

func (h *Handler) renewJobDispatchClaim(w http.ResponseWriter, r *http.Request) {
	executionKey, err := parsePathString(r, "execution_key")
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	var req renewJobDispatchClaimRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	req.InstanceID = strings.TrimSpace(req.InstanceID)
	if req.InstanceID == "" || req.LeaseTTLSecond <= 0 {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "instance_id and lease_ttl_sec > 0 are required"})
		return
	}

	out, err := h.jobs.RenewClaimLease(executionKey, req.InstanceID, req.LeaseTTLSecond, time.Now().UTC())
	if err != nil {
		switch err {
		case jobscheduler.ErrClaimNotFound:
			respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		case jobscheduler.ErrClaimLeaseOwnerMismatch:
			respondJSON(w, http.StatusForbidden, map[string]string{"error": err.Error()})
			return
		default:
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
	}
	h.appendAudit("consistency", "job_dispatch_claim_renew", executionKey)
	respondJSON(w, http.StatusOK, out)
}

func (h *Handler) setJobRetryPolicy(w http.ResponseWriter, r *http.Request) {
	jobID, err := parsePathInt64(r, "id")
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	var req setJobRetryPolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	policy, err := h.jobs.SetRetryPolicy(jobID, jobscheduler.RetryPolicy{
		MaxRetries:        req.MaxRetries,
		BackoffBaseMillis: req.BackoffBaseMillis,
		BackoffMaxMillis:  req.BackoffMaxMillis,
		JitterPercent:     req.JitterPercent,
	})
	if err != nil {
		if err == jobscheduler.ErrJobNotFound {
			respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	h.appendAudit("scheduler", "retry_policy_set", fmt.Sprintf("job:%d", jobID))
	respondJSON(w, http.StatusOK, policy)
}

func (h *Handler) getJobRetryPolicy(w http.ResponseWriter, r *http.Request) {
	jobID, err := parsePathInt64(r, "id")
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	policy, err := h.jobs.GetRetryPolicy(jobID)
	if err != nil {
		if err == jobscheduler.ErrJobNotFound {
			respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	respondJSON(w, http.StatusOK, policy)
}

func (h *Handler) scheduleJobRetry(w http.ResponseWriter, r *http.Request) {
	jobID, err := parsePathInt64(r, "id")
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	var req scheduleJobRetryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	req.ExecutionKey = strings.TrimSpace(req.ExecutionKey)
	if req.ExecutionKey == "" || req.Attempt <= 0 {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "execution_key and attempt > 0 are required"})
		return
	}
	scheduled, err := h.jobs.ScheduleRetry(jobID, req.ExecutionKey, req.Attempt, time.Now().UTC())
	if err != nil {
		if err == jobscheduler.ErrJobNotFound {
			respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	h.appendAudit("scheduler", "retry_scheduled", req.ExecutionKey)
	respondJSON(w, http.StatusOK, scheduled)
}

func (h *Handler) markJobDeadLetter(w http.ResponseWriter, r *http.Request) {
	jobID, err := parsePathInt64(r, "id")
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	var req markJobDeadLetterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	req.ExecutionKey = strings.TrimSpace(req.ExecutionKey)
	req.Reason = strings.TrimSpace(req.Reason)
	if req.ExecutionKey == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "execution_key is required"})
		return
	}
	item, err := h.jobs.MarkDeadLetter(jobID, req.ExecutionKey, req.Reason, req.RetryCount, time.Now().UTC())
	if err != nil {
		if err == jobscheduler.ErrJobNotFound {
			respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	h.appendAudit("scheduler", "dead_letter_marked", req.ExecutionKey)
	respondJSON(w, http.StatusOK, item)
}

func (h *Handler) listJobDeadLetters(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "limit must be a positive integer"})
			return
		}
		limit = parsed
	}
	respondJSON(w, http.StatusOK, map[string]any{"items": h.jobs.ListDeadLetters(limit)})
}

func (h *Handler) replayJobDeadLetter(w http.ResponseWriter, r *http.Request) {
	executionKey, err := parsePathString(r, "execution_key")
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	var req replayJobDeadLetterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	item, err := h.jobs.ReplayDeadLetter(executionKey, strings.TrimSpace(req.Operator), time.Now().UTC())
	if err != nil {
		if err == jobscheduler.ErrDeadLetterNotFound {
			respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	h.appendAudit("scheduler", "dead_letter_replayed", executionKey)
	respondJSON(w, http.StatusOK, item)
}

func (h *Handler) jobReliabilityMetrics(w http.ResponseWriter, _ *http.Request) {
	respondJSON(w, http.StatusOK, h.jobs.ReliabilitySnapshot(time.Now().UTC()))
}

func (h *Handler) appendAudit(actor, action, target string) {
	if h.audit == nil {
		return
	}
	h.audit.Append(actor, action, target)
}

func parsePathInt64(r *http.Request, key string) (int64, error) {
	raw := strings.TrimSpace(r.PathValue(key))
	if raw == "" {
		return 0, fmt.Errorf("%s is required", key)
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", key)
	}
	return id, nil
}

func parsePathString(r *http.Request, key string) (string, error) {
	raw := strings.TrimSpace(r.PathValue(key))
	if raw == "" {
		return "", fmt.Errorf("%s is required", key)
	}
	return raw, nil
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

type createJobRequest struct {
	Name     string `json:"name"`
	Schedule string `json:"schedule"`
}

type claimJobDispatchRequest struct {
	ExecutionKey string `json:"execution_key"`
	InstanceID   string `json:"instance_id"`
}

type renewJobDispatchClaimRequest struct {
	InstanceID     string `json:"instance_id"`
	LeaseTTLSecond int64  `json:"lease_ttl_sec"`
}

type setJobRetryPolicyRequest struct {
	MaxRetries        int `json:"max_retries"`
	BackoffBaseMillis int `json:"backoff_base_millis"`
	BackoffMaxMillis  int `json:"backoff_max_millis"`
	JitterPercent     int `json:"jitter_percent"`
}

type scheduleJobRetryRequest struct {
	ExecutionKey string `json:"execution_key"`
	Attempt      int    `json:"attempt"`
}

type markJobDeadLetterRequest struct {
	ExecutionKey string `json:"execution_key"`
	Reason       string `json:"reason"`
	RetryCount   int    `json:"retry_count"`
}

type replayJobDeadLetterRequest struct {
	Operator string `json:"operator"`
}
