package hardening

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
)

type AuditService interface {
	Append(actor, action, target string) audit.Record
}

type APIRegistry interface {
	RegisterMany(entries []string)
}

// PostureSnapshot is returned by Snapshot() and consumed by the dashboard aggregate.
type PostureSnapshot struct {
	EndpointGuardrailsEnabled int   `json:"endpoint_guardrails_enabled"`
	AlertProfilesEnabled      int   `json:"alert_profiles_enabled"`
	IncidentRunbookProfiles   int   `json:"incident_runbook_profiles"`
	FaultDrillsRecorded       int   `json:"fault_drills_recorded"`
	LatestDrillRecordedAtSec  int64 `json:"latest_drill_recorded_at_unix_sec"`
}

type Handler struct {
	audit AuditService
	state hardeningState
}

type hardeningState struct {
	mu            sync.Mutex
	profiles      map[string]endpointGuardrailProfile
	alertProfiles map[string]alertProfile
	runbooks      map[string]incidentRunbookProfile
	drills        []faultDrillRecord
	nextDrillID   int64
}

func NewHandler(auditSvc AuditService) *Handler {
	return &Handler{
		audit: auditSvc,
		state: hardeningState{
			profiles:      make(map[string]endpointGuardrailProfile),
			alertProfiles: make(map[string]alertProfile),
			runbooks:      make(map[string]incidentRunbookProfile),
			nextDrillID:   1,
		},
	}
}

func (h *Handler) Register(mux *http.ServeMux, wrapper func(http.Handler) http.Handler, apis APIRegistry) {
	if mux == nil || h == nil || h.audit == nil {
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

	handle("PUT /admin/v1/system/hardening/endpoint-guardrails", h.setEndpointGuardrails)
	handle("GET /admin/v1/system/hardening/endpoint-guardrails", h.listEndpointGuardrails)
	handle("PUT /admin/v1/system/hardening/alert-profiles", h.setAlertProfiles)
	handle("GET /admin/v1/system/hardening/alert-profiles", h.listAlertProfiles)
	handle("PUT /admin/v1/system/hardening/incident-runbooks", h.setIncidentRunbooks)
	handle("GET /admin/v1/system/hardening/incident-runbooks", h.listIncidentRunbooks)
	handle("POST /admin/v1/system/hardening/fault-drills", h.createFaultDrill)
	handle("GET /admin/v1/system/hardening/fault-drills", h.listFaultDrills)
}

// Snapshot returns the current hardening posture for use by the dashboard aggregate.
func (h *Handler) Snapshot() PostureSnapshot {
	h.state.mu.Lock()
	defer h.state.mu.Unlock()

	endpointEnabled := 0
	for _, item := range h.state.profiles {
		if item.Enabled {
			endpointEnabled++
		}
	}
	alertEnabled := 0
	for _, item := range h.state.alertProfiles {
		if item.Enabled {
			alertEnabled++
		}
	}
	latest := int64(0)
	for _, item := range h.state.drills {
		if item.RecordedAtUnixSec > latest {
			latest = item.RecordedAtUnixSec
		}
	}
	return PostureSnapshot{
		EndpointGuardrailsEnabled: endpointEnabled,
		AlertProfilesEnabled:      alertEnabled,
		IncidentRunbookProfiles:   len(h.state.runbooks),
		FaultDrillsRecorded:       len(h.state.drills),
		LatestDrillRecordedAtSec:  latest,
	}
}

func (h *Handler) setEndpointGuardrails(w http.ResponseWriter, r *http.Request) {
	var req setEndpointGuardrailsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	if len(req.Profiles) == 0 {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "profiles must not be empty"})
		return
	}

	now := time.Now().UTC().Unix()
	normalized := make(map[string]endpointGuardrailProfile, len(req.Profiles))
	for _, item := range req.Profiles {
		item.Endpoint = strings.TrimSpace(item.Endpoint)
		if item.Endpoint == "" {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "endpoint is required"})
			return
		}
		if item.RateLimitRPM <= 0 {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "rate_limit_rpm must be > 0"})
			return
		}
		if item.TimeoutMillis <= 0 {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "timeout_millis must be > 0"})
			return
		}
		if item.CircuitErrorThreshold <= 0 {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "circuit_error_threshold must be > 0"})
			return
		}
		if item.CircuitOpenWindowSec <= 0 {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "circuit_open_window_sec must be > 0"})
			return
		}
		item.UpdatedAtUnixSec = now
		normalized[item.Endpoint] = item
	}

	h.state.mu.Lock()
	h.state.profiles = normalized
	h.state.mu.Unlock()

	h.audit.Append("system", "endpoint_guardrails_updated", strconv.Itoa(len(normalized)))
	respondJSON(w, http.StatusOK, map[string]any{"items": h.endpointGuardrailsSnapshot()})
}

func (h *Handler) listEndpointGuardrails(w http.ResponseWriter, _ *http.Request) {
	respondJSON(w, http.StatusOK, map[string]any{"items": h.endpointGuardrailsSnapshot()})
}

func (h *Handler) endpointGuardrailsSnapshot() []endpointGuardrailProfile {
	h.state.mu.Lock()
	defer h.state.mu.Unlock()

	items := make([]endpointGuardrailProfile, 0, len(h.state.profiles))
	for _, item := range h.state.profiles {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Endpoint < items[j].Endpoint })
	return items
}

func (h *Handler) setAlertProfiles(w http.ResponseWriter, r *http.Request) {
	var req setAlertProfilesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	if len(req.Profiles) == 0 {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "profiles must not be empty"})
		return
	}

	now := time.Now().UTC().Unix()
	normalized := make(map[string]alertProfile, len(req.Profiles))
	for _, item := range req.Profiles {
		item.Metric = strings.TrimSpace(item.Metric)
		item.Runbook = strings.TrimSpace(item.Runbook)
		item.Owner = strings.TrimSpace(item.Owner)
		if item.Metric == "" {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "metric is required"})
			return
		}
		if item.WarnThreshold <= 0 {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "warn_threshold must be > 0"})
			return
		}
		if item.CriticalThreshold <= 0 {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "critical_threshold must be > 0"})
			return
		}
		if item.CriticalThreshold < item.WarnThreshold {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "critical_threshold must be >= warn_threshold"})
			return
		}
		if item.WindowSeconds <= 0 {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "window_seconds must be > 0"})
			return
		}
		if item.Runbook == "" {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "runbook is required"})
			return
		}
		if item.Owner == "" {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "owner is required"})
			return
		}
		item.UpdatedAtUnixSec = now
		normalized[item.Metric] = item
	}

	h.state.mu.Lock()
	h.state.alertProfiles = normalized
	h.state.mu.Unlock()

	h.audit.Append("system", "alert_profiles_updated", strconv.Itoa(len(normalized)))
	respondJSON(w, http.StatusOK, map[string]any{"items": h.alertProfilesSnapshot()})
}

func (h *Handler) listAlertProfiles(w http.ResponseWriter, _ *http.Request) {
	respondJSON(w, http.StatusOK, map[string]any{"items": h.alertProfilesSnapshot()})
}

func (h *Handler) alertProfilesSnapshot() []alertProfile {
	h.state.mu.Lock()
	defer h.state.mu.Unlock()

	items := make([]alertProfile, 0, len(h.state.alertProfiles))
	for _, item := range h.state.alertProfiles {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Metric < items[j].Metric })
	return items
}

func (h *Handler) setIncidentRunbooks(w http.ResponseWriter, r *http.Request) {
	var req setIncidentRunbooksRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	if len(req.Profiles) == 0 {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "profiles must not be empty"})
		return
	}

	now := time.Now().UTC().Unix()
	normalized := make(map[string]incidentRunbookProfile, len(req.Profiles))
	for _, item := range req.Profiles {
		item.Domain = strings.TrimSpace(item.Domain)
		item.Severity = strings.TrimSpace(item.Severity)
		item.Runbook = strings.TrimSpace(item.Runbook)
		item.Owner = strings.TrimSpace(item.Owner)
		item.Escalation = strings.TrimSpace(item.Escalation)
		if item.Domain == "" {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "domain is required"})
			return
		}
		if item.Severity == "" {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "severity is required"})
			return
		}
		if item.Runbook == "" {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "runbook is required"})
			return
		}
		if item.Owner == "" {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "owner is required"})
			return
		}
		if item.Escalation == "" {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "escalation is required"})
			return
		}
		if item.MitigationSLASeconds <= 0 {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "mitigation_sla_seconds must be > 0"})
			return
		}
		item.UpdatedAtUnixSec = now
		normalized[item.Domain+":"+item.Severity] = item
	}

	h.state.mu.Lock()
	h.state.runbooks = normalized
	h.state.mu.Unlock()

	h.audit.Append("system", "incident_runbooks_updated", strconv.Itoa(len(normalized)))
	respondJSON(w, http.StatusOK, map[string]any{"items": h.incidentRunbooksSnapshot()})
}

func (h *Handler) listIncidentRunbooks(w http.ResponseWriter, _ *http.Request) {
	respondJSON(w, http.StatusOK, map[string]any{"items": h.incidentRunbooksSnapshot()})
}

func (h *Handler) incidentRunbooksSnapshot() []incidentRunbookProfile {
	h.state.mu.Lock()
	defer h.state.mu.Unlock()

	items := make([]incidentRunbookProfile, 0, len(h.state.runbooks))
	for _, item := range h.state.runbooks {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Domain == items[j].Domain {
			return items[i].Severity < items[j].Severity
		}
		return items[i].Domain < items[j].Domain
	})
	return items
}

func (h *Handler) createFaultDrill(w http.ResponseWriter, r *http.Request) {
	var req createFaultDrillRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	req.Scenario = strings.TrimSpace(req.Scenario)
	req.Domain = strings.TrimSpace(req.Domain)
	req.Injector = strings.TrimSpace(req.Injector)
	req.MitigationEvidence = strings.TrimSpace(req.MitigationEvidence)
	req.ResidualRisk = strings.TrimSpace(req.ResidualRisk)
	if req.Scenario == "" || req.Domain == "" || req.Injector == "" || req.MitigationEvidence == "" || req.ResidualRisk == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "scenario, domain, injector, mitigation_evidence and residual_risk are required"})
		return
	}

	h.state.mu.Lock()
	record := faultDrillRecord{
		DrillID:            fmt.Sprintf("drill-%d", h.state.nextDrillID),
		Scenario:           req.Scenario,
		Domain:             req.Domain,
		Injector:           req.Injector,
		MitigationEvidence: req.MitigationEvidence,
		ResidualRisk:       req.ResidualRisk,
		RecordedAtUnixSec:  time.Now().UTC().Unix(),
	}
	h.state.nextDrillID++
	h.state.drills = append(h.state.drills, record)
	h.state.mu.Unlock()

	h.audit.Append("system", "fault_drill_recorded", record.DrillID)
	respondJSON(w, http.StatusCreated, record)
}

func (h *Handler) listFaultDrills(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "limit must be a positive integer"})
			return
		}
		limit = parsed
	}

	h.state.mu.Lock()
	items := make([]faultDrillRecord, len(h.state.drills))
	copy(items, h.state.drills)
	h.state.mu.Unlock()

	sort.Slice(items, func(i, j int) bool { return items[i].RecordedAtUnixSec > items[j].RecordedAtUnixSec })
	if limit < len(items) {
		items = items[:limit]
	}
	respondJSON(w, http.StatusOK, map[string]any{"items": items})
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// --- DTOs ---

type endpointGuardrailProfile struct {
	Endpoint              string `json:"endpoint"`
	RateLimitRPM          int    `json:"rate_limit_rpm"`
	TimeoutMillis         int    `json:"timeout_millis"`
	CircuitErrorThreshold int    `json:"circuit_error_threshold"`
	CircuitOpenWindowSec  int    `json:"circuit_open_window_sec"`
	Enabled               bool   `json:"enabled"`
	UpdatedAtUnixSec      int64  `json:"updated_at_unix_sec"`
}

type setEndpointGuardrailsRequest struct {
	Profiles []endpointGuardrailProfile `json:"profiles"`
}

type alertProfile struct {
	Metric            string  `json:"metric"`
	WarnThreshold     float64 `json:"warn_threshold"`
	CriticalThreshold float64 `json:"critical_threshold"`
	WindowSeconds     int     `json:"window_seconds"`
	Runbook           string  `json:"runbook"`
	Owner             string  `json:"owner"`
	Enabled           bool    `json:"enabled"`
	UpdatedAtUnixSec  int64   `json:"updated_at_unix_sec"`
}

type setAlertProfilesRequest struct {
	Profiles []alertProfile `json:"profiles"`
}

type incidentRunbookProfile struct {
	Domain               string `json:"domain"`
	Severity             string `json:"severity"`
	Runbook              string `json:"runbook"`
	Owner                string `json:"owner"`
	Escalation           string `json:"escalation"`
	MitigationSLASeconds int    `json:"mitigation_sla_seconds"`
	UpdatedAtUnixSec     int64  `json:"updated_at_unix_sec"`
}

type setIncidentRunbooksRequest struct {
	Profiles []incidentRunbookProfile `json:"profiles"`
}

type createFaultDrillRequest struct {
	Scenario           string `json:"scenario"`
	Domain             string `json:"domain"`
	Injector           string `json:"injector"`
	MitigationEvidence string `json:"mitigation_evidence"`
	ResidualRisk       string `json:"residual_risk"`
}

type faultDrillRecord struct {
	DrillID            string `json:"drill_id"`
	Scenario           string `json:"scenario"`
	Domain             string `json:"domain"`
	Injector           string `json:"injector"`
	MitigationEvidence string `json:"mitigation_evidence"`
	ResidualRisk       string `json:"residual_risk"`
	RecordedAtUnixSec  int64  `json:"recorded_at_unix_sec"`
}
