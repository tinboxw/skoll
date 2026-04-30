package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/tinboxw/skoll/internal/module/releasegov"
)

type submitReleaseEvidenceRequest struct {
	Milestone           string  `json:"milestone"`
	GoTestPassed        bool    `json:"go_test_passed"`
	GoRacePassed        bool    `json:"go_race_passed"`
	ReadmeSynced        bool    `json:"readme_synced"`
	BenchmarkNsPerOp    float64 `json:"benchmark_ns_per_op"`
	BaselineNsPerOp     float64 `json:"baseline_ns_per_op"`
	BenchmarkCommand    string  `json:"benchmark_command"`
	EvidenceDescription string  `json:"evidence_description"`
}

type parityClosureCheckpoint struct {
	ReferenceProject string   `json:"reference_project"`
	Capability       string   `json:"capability"`
	Status           string   `json:"status"`
	EvidenceLinks    []string `json:"evidence_links"`
	KnownGap         string   `json:"known_gap,omitempty"`
	Owner            string   `json:"owner"`
	UpdatedAtUnixSec int64    `json:"updated_at_unix_sec"`
}

type setParityClosureCheckpointsRequest struct {
	Items []parityClosureCheckpoint `json:"items"`
}

type parityClosureReport struct {
	ReferenceProjects         []string `json:"reference_projects"`
	TotalCheckpoints          int      `json:"total_checkpoints"`
	CompletedCheckpoints      int      `json:"completed_checkpoints"`
	KnownGapCheckpoints       int      `json:"known_gap_checkpoints"`
	EvidenceLinkedCheckpoints int      `json:"evidence_linked_checkpoints"`
	CompatibilityStatement    string   `json:"compatibility_statement"`
	KnownGapSummary           []string `json:"known_gap_summary"`
	GeneratedAtUnixSec        int64    `json:"generated_at_unix_sec"`
}

type releaseBlockingPolicy struct {
	AllowedRegression      float64 `json:"allowed_regression_ratio"`
	BlockOnGoTestFailure   bool    `json:"block_on_go_test_failure"`
	BlockOnGoRaceFailure   bool    `json:"block_on_go_race_failure"`
	BlockOnReadmeNotSynced bool    `json:"block_on_readme_not_synced"`
	BlockOnMissingEvidence bool    `json:"block_on_missing_evidence"`
	UpdatedAtUnixSec       int64   `json:"updated_at_unix_sec"`
}

type setReleaseBlockingPolicyRequest struct {
	AllowedRegression      float64 `json:"allowed_regression_ratio"`
	BlockOnGoTestFailure   bool    `json:"block_on_go_test_failure"`
	BlockOnGoRaceFailure   bool    `json:"block_on_go_race_failure"`
	BlockOnReadmeNotSynced bool    `json:"block_on_readme_not_synced"`
	BlockOnMissingEvidence bool    `json:"block_on_missing_evidence"`
}

type releaseBlockDecision struct {
	Milestone          string                `json:"milestone"`
	Blocked            bool                  `json:"blocked"`
	Reasons            []string              `json:"reasons,omitempty"`
	Scorecard          releasegov.Scorecard  `json:"scorecard"`
	Policy             releaseBlockingPolicy `json:"policy"`
	EvaluatedAtUnixSec int64                 `json:"evaluated_at_unix_sec"`
}

type releaseClosureState struct {
	mu          sync.Mutex
	checkpoints map[string]parityClosureCheckpoint
	policy      releaseBlockingPolicy
}

var adminReleaseClosureState = releaseClosureState{
	checkpoints: make(map[string]parityClosureCheckpoint),
	policy: releaseBlockingPolicy{
		AllowedRegression:      0.10,
		BlockOnGoTestFailure:   true,
		BlockOnGoRaceFailure:   true,
		BlockOnReadmeNotSynced: true,
		BlockOnMissingEvidence: true,
	},
}

func submitReleaseEvidenceHandler(svc ReleaseService, auditSvc AuditService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req submitReleaseEvidenceRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}

		out, err := svc.SubmitEvidence(releasegov.EvidenceInput{
			Milestone:           req.Milestone,
			GoTestPassed:        req.GoTestPassed,
			GoRacePassed:        req.GoRacePassed,
			ReadmeSynced:        req.ReadmeSynced,
			BenchmarkNsPerOp:    req.BenchmarkNsPerOp,
			BaselineNsPerOp:     req.BaselineNsPerOp,
			BenchmarkCommand:    req.BenchmarkCommand,
			EvidenceDescription: req.EvidenceDescription,
		}, time.Now().UTC())
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		auditSvc.Append("release-governance", "evidence_submit", out.Milestone)
		respondJSON(w, http.StatusCreated, out)
	}
}

func getReleaseScorecardHandler(svc ReleaseService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		milestone, err := parsePathString(r, "milestone")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		allowedRegression := 0.10
		if raw := strings.TrimSpace(r.URL.Query().Get("allowed_regression")); raw != "" {
			parsed, err := strconv.ParseFloat(raw, 64)
			if err != nil || parsed <= 0 {
				respondJSON(w, http.StatusBadRequest, map[string]string{"error": "allowed_regression must be a positive float"})
				return
			}
			allowedRegression = parsed
		}
		respondJSON(w, http.StatusOK, svc.Scorecard(milestone, allowedRegression))
	}
}

func setReleaseBlockingPolicyHandler(auditSvc AuditService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req setReleaseBlockingPolicyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}
		if req.AllowedRegression <= 0 {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "allowed_regression_ratio must be > 0"})
			return
		}

		adminReleaseClosureState.mu.Lock()
		adminReleaseClosureState.policy = releaseBlockingPolicy{
			AllowedRegression:      req.AllowedRegression,
			BlockOnGoTestFailure:   req.BlockOnGoTestFailure,
			BlockOnGoRaceFailure:   req.BlockOnGoRaceFailure,
			BlockOnReadmeNotSynced: req.BlockOnReadmeNotSynced,
			BlockOnMissingEvidence: req.BlockOnMissingEvidence,
			UpdatedAtUnixSec:       time.Now().UTC().Unix(),
		}
		policy := adminReleaseClosureState.policy
		adminReleaseClosureState.mu.Unlock()

		auditSvc.Append("release-governance", "blocking_policy_updated", fmt.Sprintf("%.4f", policy.AllowedRegression))
		respondJSON(w, http.StatusOK, policy)
	}
}

func getReleaseBlockingPolicyHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		adminReleaseClosureState.mu.Lock()
		policy := adminReleaseClosureState.policy
		adminReleaseClosureState.mu.Unlock()
		respondJSON(w, http.StatusOK, policy)
	}
}

func getReleaseBlockDecisionHandler(svc ReleaseService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		milestone, err := parsePathString(r, "milestone")
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		adminReleaseClosureState.mu.Lock()
		policy := adminReleaseClosureState.policy
		adminReleaseClosureState.mu.Unlock()

		scorecard := svc.Scorecard(milestone, policy.AllowedRegression)
		reasons := make([]string, 0, len(scorecard.FailedChecks))
		for _, check := range scorecard.FailedChecks {
			switch check {
			case "missing_evidence":
				if policy.BlockOnMissingEvidence {
					reasons = append(reasons, check)
				}
			case "go_test_failed":
				if policy.BlockOnGoTestFailure {
					reasons = append(reasons, check)
				}
			case "go_race_failed":
				if policy.BlockOnGoRaceFailure {
					reasons = append(reasons, check)
				}
			case "readme_not_synced":
				if policy.BlockOnReadmeNotSynced {
					reasons = append(reasons, check)
				}
			default:
				reasons = append(reasons, check)
			}
		}

		respondJSON(w, http.StatusOK, releaseBlockDecision{
			Milestone:          strings.TrimSpace(strings.ToLower(milestone)),
			Blocked:            len(reasons) > 0,
			Reasons:            reasons,
			Scorecard:          scorecard,
			Policy:             policy,
			EvaluatedAtUnixSec: time.Now().UTC().Unix(),
		})
	}
}

func setParityClosureCheckpointsHandler(auditSvc AuditService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req setParityClosureCheckpointsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}
		if len(req.Items) == 0 {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "items must not be empty"})
			return
		}

		now := time.Now().UTC().Unix()
		normalized := make(map[string]parityClosureCheckpoint, len(req.Items))
		for _, item := range req.Items {
			item.ReferenceProject = strings.TrimSpace(strings.ToLower(item.ReferenceProject))
			item.Capability = strings.TrimSpace(item.Capability)
			item.Status = strings.TrimSpace(strings.ToLower(item.Status))
			item.KnownGap = strings.TrimSpace(item.KnownGap)
			item.Owner = strings.TrimSpace(item.Owner)
			if item.ReferenceProject == "" {
				respondJSON(w, http.StatusBadRequest, map[string]string{"error": "reference_project is required"})
				return
			}
			if item.Capability == "" {
				respondJSON(w, http.StatusBadRequest, map[string]string{"error": "capability is required"})
				return
			}
			if item.Status != "completed" && item.Status != "partial" && item.Status != "planned" {
				respondJSON(w, http.StatusBadRequest, map[string]string{"error": "status must be completed|partial|planned"})
				return
			}
			if len(item.EvidenceLinks) == 0 {
				respondJSON(w, http.StatusBadRequest, map[string]string{"error": "evidence_links must not be empty"})
				return
			}
			for i := range item.EvidenceLinks {
				item.EvidenceLinks[i] = strings.TrimSpace(item.EvidenceLinks[i])
				if item.EvidenceLinks[i] == "" {
					respondJSON(w, http.StatusBadRequest, map[string]string{"error": "evidence_links must not contain empty value"})
					return
				}
			}
			if item.Owner == "" {
				respondJSON(w, http.StatusBadRequest, map[string]string{"error": "owner is required"})
				return
			}
			item.UpdatedAtUnixSec = now
			normalized[item.ReferenceProject+":"+strings.ToLower(item.Capability)] = item
		}

		adminReleaseClosureState.mu.Lock()
		adminReleaseClosureState.checkpoints = normalized
		adminReleaseClosureState.mu.Unlock()

		auditSvc.Append("release-governance", "parity_closure_checkpoints_updated", strconv.Itoa(len(normalized)))
		respondJSON(w, http.StatusOK, map[string]any{"items": parityClosureCheckpointsSnapshot()})
	}
}

func listParityClosureCheckpointsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		respondJSON(w, http.StatusOK, map[string]any{"items": parityClosureCheckpointsSnapshot()})
	}
}

func getParityClosureReportHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		items := parityClosureCheckpointsSnapshot()
		projectsSet := make(map[string]struct{})
		knownGaps := make([]string, 0)
		completed := 0
		evidenceLinked := 0
		knownGapCount := 0
		for _, item := range items {
			projectsSet[item.ReferenceProject] = struct{}{}
			if item.Status == "completed" {
				completed++
			}
			if len(item.EvidenceLinks) > 0 {
				evidenceLinked++
			}
			if item.KnownGap != "" {
				knownGapCount++
				knownGaps = append(knownGaps, item.ReferenceProject+": "+item.Capability+" -> "+item.KnownGap)
			}
		}
		projects := make([]string, 0, len(projectsSet))
		for project := range projectsSet {
			projects = append(projects, project)
		}
		sort.Strings(projects)
		sort.Strings(knownGaps)

		statement := "parity closure in progress"
		if len(items) > 0 && completed == len(items) && knownGapCount == 0 {
			statement = "full parity achieved against tracked reference capabilities"
		} else if len(items) > 0 && completed > 0 {
			statement = "parity partially achieved with known gaps tracked"
		}

		respondJSON(w, http.StatusOK, parityClosureReport{
			ReferenceProjects:         projects,
			TotalCheckpoints:          len(items),
			CompletedCheckpoints:      completed,
			KnownGapCheckpoints:       knownGapCount,
			EvidenceLinkedCheckpoints: evidenceLinked,
			CompatibilityStatement:    statement,
			KnownGapSummary:           knownGaps,
			GeneratedAtUnixSec:        time.Now().UTC().Unix(),
		})
	}
}

func parityClosureCheckpointsSnapshot() []parityClosureCheckpoint {
	adminReleaseClosureState.mu.Lock()
	defer adminReleaseClosureState.mu.Unlock()

	items := make([]parityClosureCheckpoint, 0, len(adminReleaseClosureState.checkpoints))
	for _, item := range adminReleaseClosureState.checkpoints {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].ReferenceProject == items[j].ReferenceProject {
			return strings.ToLower(items[i].Capability) < strings.ToLower(items[j].Capability)
		}
		return items[i].ReferenceProject < items[j].ReferenceProject
	})
	return items
}
