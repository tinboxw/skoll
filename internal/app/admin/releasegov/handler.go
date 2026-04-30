package releasegov

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/tinboxw/skoll/internal/module/audit"
	"github.com/tinboxw/skoll/internal/module/releasegov"
)

type ReleaseService interface {
	SubmitEvidence(input releasegov.EvidenceInput, now time.Time) (releasegov.Evidence, error)
	Scorecard(milestone string, allowedRegression float64) releasegov.Scorecard
}

type AuditService interface {
	Append(actor, action, target string) audit.Record
}

type APIRegistry interface {
	RegisterMany(entries []string)
}

type Handler struct {
	svc   ReleaseService
	audit AuditService
	state releaseClosureState
}

type releaseClosureState struct {
	mu          sync.Mutex
	checkpoints map[string]parityClosureCheckpoint
	policy      releaseBlockingPolicy
}

func NewHandler(svc ReleaseService, auditSvc AuditService) *Handler {
	return &Handler{
		svc:   svc,
		audit: auditSvc,
		state: releaseClosureState{
			checkpoints: make(map[string]parityClosureCheckpoint),
			policy: releaseBlockingPolicy{
				AllowedRegression:      0.10,
				BlockOnGoTestFailure:   true,
				BlockOnGoRaceFailure:   true,
				BlockOnReadmeNotSynced: true,
				BlockOnMissingEvidence: true,
			},
		},
	}
}

func (h *Handler) Register(mux *http.ServeMux, wrapper func(http.Handler) http.Handler, apis APIRegistry) {
	if mux == nil || h.svc == nil {
		return
	}

	handle := func(pattern string, next http.HandlerFunc) {
		if apis != nil {
			apis.RegisterMany([]string{pattern})
		}
		hd := http.Handler(next)
		if wrapper != nil {
			hd = wrapper(hd)
		}
		mux.Handle(pattern, hd)
	}

	handle("POST /admin/v1/release-governance/evidence", h.submitReleaseEvidence)
	handle("GET /admin/v1/release-governance/scorecard/{milestone}", h.getReleaseScorecard)
	handle("PUT /admin/v1/release-governance/blocking-policy", h.setReleaseBlockingPolicy)
	handle("GET /admin/v1/release-governance/blocking-policy", h.getReleaseBlockingPolicy)
	handle("GET /admin/v1/release-governance/block-decision/{milestone}", h.getReleaseBlockDecision)
	handle("PUT /admin/v1/release-governance/parity-closure/checkpoints", h.setParityClosureCheckpoints)
	handle("GET /admin/v1/release-governance/parity-closure/checkpoints", h.listParityClosureCheckpoints)
	handle("GET /admin/v1/release-governance/parity-closure/report", h.getParityClosureReport)
}

func (h *Handler) submitReleaseEvidence(w http.ResponseWriter, r *http.Request) {
	var req submitReleaseEvidenceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}

	out, err := h.svc.SubmitEvidence(releasegov.EvidenceInput{
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
	h.audit.Append("release-governance", "evidence_submit", out.Milestone)
	respondJSON(w, http.StatusCreated, out)
}

func (h *Handler) getReleaseScorecard(w http.ResponseWriter, r *http.Request) {
	milestone, err := parsePathString(r, "milestone")
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	allowedRegression := 0.10
	if raw := strings.TrimSpace(r.URL.Query().Get("allowed_regression")); raw != "" {
		parsed, parseErr := strconv.ParseFloat(raw, 64)
		if parseErr != nil || parsed <= 0 {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "allowed_regression must be a positive float"})
			return
		}
		allowedRegression = parsed
	}
	respondJSON(w, http.StatusOK, h.svc.Scorecard(milestone, allowedRegression))
}

func (h *Handler) setReleaseBlockingPolicy(w http.ResponseWriter, r *http.Request) {
	var req setReleaseBlockingPolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	if req.AllowedRegression <= 0 {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "allowed_regression_ratio must be > 0"})
		return
	}

	h.state.mu.Lock()
	h.state.policy = releaseBlockingPolicy{
		AllowedRegression:      req.AllowedRegression,
		BlockOnGoTestFailure:   req.BlockOnGoTestFailure,
		BlockOnGoRaceFailure:   req.BlockOnGoRaceFailure,
		BlockOnReadmeNotSynced: req.BlockOnReadmeNotSynced,
		BlockOnMissingEvidence: req.BlockOnMissingEvidence,
		UpdatedAtUnixSec:       time.Now().UTC().Unix(),
	}
	policy := h.state.policy
	h.state.mu.Unlock()

	h.audit.Append("release-governance", "blocking_policy_updated", fmt.Sprintf("%.4f", policy.AllowedRegression))
	respondJSON(w, http.StatusOK, policy)
}

func (h *Handler) getReleaseBlockingPolicy(w http.ResponseWriter, _ *http.Request) {
	h.state.mu.Lock()
	policy := h.state.policy
	h.state.mu.Unlock()
	respondJSON(w, http.StatusOK, policy)
}

func (h *Handler) getReleaseBlockDecision(w http.ResponseWriter, r *http.Request) {
	milestone, err := parsePathString(r, "milestone")
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	h.state.mu.Lock()
	policy := h.state.policy
	h.state.mu.Unlock()

	scorecard := h.svc.Scorecard(milestone, policy.AllowedRegression)
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

func (h *Handler) setParityClosureCheckpoints(w http.ResponseWriter, r *http.Request) {
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

	h.state.mu.Lock()
	h.state.checkpoints = normalized
	h.state.mu.Unlock()

	h.audit.Append("release-governance", "parity_closure_checkpoints_updated", strconv.Itoa(len(normalized)))
	respondJSON(w, http.StatusOK, map[string]any{"items": h.parityClosureCheckpointsSnapshot()})
}

func (h *Handler) listParityClosureCheckpoints(w http.ResponseWriter, _ *http.Request) {
	respondJSON(w, http.StatusOK, map[string]any{"items": h.parityClosureCheckpointsSnapshot()})
}

func (h *Handler) getParityClosureReport(w http.ResponseWriter, _ *http.Request) {
	items := h.parityClosureCheckpointsSnapshot()
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

func (h *Handler) parityClosureCheckpointsSnapshot() []parityClosureCheckpoint {
	h.state.mu.Lock()
	defer h.state.mu.Unlock()

	items := make([]parityClosureCheckpoint, 0, len(h.state.checkpoints))
	for _, item := range h.state.checkpoints {
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

func parsePathString(r *http.Request, param string) (string, error) {
	v := strings.TrimSpace(r.PathValue(param))
	if v == "" {
		return "", fmt.Errorf("%s path parameter is required", param)
	}
	return v, nil
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// --- DTOs ---

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
