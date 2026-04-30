package app

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/module/jobscheduler"
)

func createJobHandler(svc JobService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		respondJSON(w, http.StatusCreated, svc.Create(req.Name, req.Schedule))
	}
}

func listJobsHandler(svc JobService) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		respondJSON(w, http.StatusOK, svc.List())
	}
}

func runJobHandler(svc JobService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parsePathInt64(r, "id")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		exec, err := svc.Run(id)
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
}

func listJobHistoryHandler(svc JobService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parsePathInt64(r, "id")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if _, err := svc.Get(id); err != nil {
			if err == jobscheduler.ErrJobNotFound {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}
		limit := 20
		if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
			parsed, err := strconv.Atoi(raw)
			if err != nil || parsed <= 0 {
				respondJSON(w, http.StatusBadRequest, map[string]string{"error": "limit must be a positive integer"})
				return
			}
			if parsed > 200 {
				parsed = 200
			}
			limit = parsed
		}
		respondJSON(w, http.StatusOK, svc.History(id, limit))
	}
}

func claimJobDispatchHandler(svc JobService, auditSvc AuditService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		out, err := svc.ClaimRun(jobID, req.ExecutionKey, req.InstanceID, time.Now().UTC())
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
		auditSvc.Append("consistency", action, req.ExecutionKey)
		respondJSON(w, http.StatusOK, out)
	}
}

func getJobDispatchClaimHandler(svc JobService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		executionKey, err := parsePathString(r, "execution_key")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		respondJSON(w, http.StatusOK, svc.ClaimStatus(executionKey))
	}
}

func renewJobDispatchClaimHandler(svc JobService, auditSvc AuditService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		out, err := svc.RenewClaimLease(executionKey, req.InstanceID, req.LeaseTTLSecond, time.Now().UTC())
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
		auditSvc.Append("consistency", "job_dispatch_claim_renew", executionKey)
		respondJSON(w, http.StatusOK, out)
	}
}

func setJobRetryPolicyHandler(svc JobService, auditSvc AuditService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		policy, err := svc.SetRetryPolicy(jobID, jobscheduler.RetryPolicy{
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
		auditSvc.Append("scheduler", "retry_policy_set", fmt.Sprintf("job:%d", jobID))
		respondJSON(w, http.StatusOK, policy)
	}
}

func getJobRetryPolicyHandler(svc JobService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jobID, err := parsePathInt64(r, "id")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		policy, err := svc.GetRetryPolicy(jobID)
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
}

func scheduleJobRetryHandler(svc JobService, auditSvc AuditService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		scheduled, err := svc.ScheduleRetry(jobID, req.ExecutionKey, req.Attempt, time.Now().UTC())
		if err != nil {
			if err == jobscheduler.ErrJobNotFound {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		auditSvc.Append("scheduler", "retry_scheduled", req.ExecutionKey)
		respondJSON(w, http.StatusOK, scheduled)
	}
}

func markJobDeadLetterHandler(svc JobService, auditSvc AuditService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		item, err := svc.MarkDeadLetter(jobID, req.ExecutionKey, req.Reason, req.RetryCount, time.Now().UTC())
		if err != nil {
			if err == jobscheduler.ErrJobNotFound {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		auditSvc.Append("scheduler", "dead_letter_marked", req.ExecutionKey)
		respondJSON(w, http.StatusOK, item)
	}
}

func listJobDeadLettersHandler(svc JobService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit := 20
		if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
			parsed, err := strconv.Atoi(raw)
			if err != nil || parsed <= 0 {
				respondJSON(w, http.StatusBadRequest, map[string]string{"error": "limit must be a positive integer"})
				return
			}
			limit = parsed
		}
		respondJSON(w, http.StatusOK, map[string]any{"items": svc.ListDeadLetters(limit)})
	}
}

func replayJobDeadLetterHandler(svc JobService, auditSvc AuditService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		item, err := svc.ReplayDeadLetter(executionKey, strings.TrimSpace(req.Operator), time.Now().UTC())
		if err != nil {
			if err == jobscheduler.ErrDeadLetterNotFound {
				respondJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
				return
			}
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		auditSvc.Append("scheduler", "dead_letter_replayed", executionKey)
		respondJSON(w, http.StatusOK, item)
	}
}

func jobReliabilityMetricsHandler(svc JobService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		respondJSON(w, http.StatusOK, svc.ReliabilitySnapshot(time.Now().UTC()))
	}
}
