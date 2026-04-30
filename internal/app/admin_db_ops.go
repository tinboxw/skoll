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
)

type migrationPlanRequest struct {
	FromVersion string   `json:"from_version"`
	ToVersion   string   `json:"to_version"`
	Steps       []string `json:"steps"`
}

type migrationPlanResponse struct {
	PlanID       string   `json:"plan_id"`
	FromVersion  string   `json:"from_version"`
	ToVersion    string   `json:"to_version"`
	Steps        []string `json:"steps"`
	CreatedAtSec int64    `json:"created_at_unix_sec"`
}

type migrationDriftDetectRequest struct {
	FromVersion   string   `json:"from_version"`
	ToVersion     string   `json:"to_version"`
	ExpectedSteps []string `json:"expected_steps"`
	AppliedSteps  []string `json:"applied_steps"`
}

type migrationDriftDetectResponse struct {
	ReportID           string   `json:"report_id"`
	FromVersion        string   `json:"from_version"`
	ToVersion          string   `json:"to_version"`
	DriftDetected      bool     `json:"drift_detected"`
	MissingExpected    []string `json:"missing_expected,omitempty"`
	UnexpectedApplied  []string `json:"unexpected_applied,omitempty"`
	ImpactGrade        string   `json:"impact_grade"`
	Recommendations    []string `json:"recommendations,omitempty"`
	GeneratedAtUnixSec int64    `json:"generated_at_unix_sec"`
}

type backupRequest struct {
	BackupID string `json:"backup_id"`
	Reason   string `json:"reason"`
}

type backupResponse struct {
	BackupID     string `json:"backup_id"`
	Status       string `json:"status"`
	CreatedAtSec int64  `json:"created_at_unix_sec"`
}

type backupCatalogItem struct {
	BackupID     string `json:"backup_id"`
	Reason       string `json:"reason,omitempty"`
	Status       string `json:"status"`
	CreatedAtSec int64  `json:"created_at_unix_sec"`
}

type restoreDrillRequest struct {
	BackupID         string `json:"backup_id"`
	ConfirmToken     string `json:"confirm_token"`
	ExpectedMaxRTOms int64  `json:"expected_max_rto_ms"`
	ExpectedMaxRPOms int64  `json:"expected_max_rpo_ms"`
}

type restoreDrillResponse struct {
	DrillID          string `json:"drill_id"`
	BackupID         string `json:"backup_id"`
	Succeeded        bool   `json:"succeeded"`
	DurationMs       int64  `json:"duration_ms"`
	ExpectedMaxRTOms int64  `json:"expected_max_rto_ms"`
	RTOCompliant     bool   `json:"rto_compliant"`
	ReplicaLagMs     int64  `json:"replica_lag_ms"`
	ExpectedMaxRPOms int64  `json:"expected_max_rpo_ms"`
	RPOCompliant     bool   `json:"rpo_compliant"`
	DataCheckPassed  bool   `json:"data_check_passed"`
	PerformedAtSec   int64  `json:"performed_at_unix_sec"`
}

type restoreRequest struct {
	BackupID     string `json:"backup_id"`
	ConfirmToken string `json:"confirm_token"`
}

type controlledSQLRequest struct {
	SQL              string `json:"sql"`
	SQLClass         string `json:"sql_class"`
	AllowDangerous   bool   `json:"allow_dangerous"`
	ConfirmToken     string `json:"confirm_token"`
	ConfirmTokenDual string `json:"confirm_token_dual"`
}

type controlledSQLResponse struct {
	Allowed      bool   `json:"allowed"`
	SQLClass     string `json:"sql_class"`
	ExecutionID  string `json:"execution_id,omitempty"`
	SafetyResult string `json:"safety_result"`
}

type dbOpsState struct {
	mu      sync.Mutex
	nextID  int64
	backups map[string]backupCatalogItem
	drills  []restoreDrillResponse
}

var adminDBOpsState = dbOpsState{nextID: 1, backups: make(map[string]backupCatalogItem)}

const dbDangerousConfirmToken = "I_UNDERSTAND"
const dbDangerousDualConfirmToken = "CONFIRM_DESTRUCTIVE_SQL"

const sqlClassReadOnly = "read_only"
const sqlClassWriteGuarded = "write_guarded"
const sqlClassDestructiveConfirmed = "destructive_confirmed"

func planDatabaseMigrationHandler(auditSvc AuditService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req migrationPlanRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}
		req.FromVersion = strings.TrimSpace(req.FromVersion)
		req.ToVersion = strings.TrimSpace(req.ToVersion)
		if req.FromVersion == "" || req.ToVersion == "" || len(req.Steps) == 0 {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "from_version, to_version and steps are required"})
			return
		}

		adminDBOpsState.mu.Lock()
		planID := fmt.Sprintf("mig-%06d", adminDBOpsState.nextID)
		adminDBOpsState.nextID++
		adminDBOpsState.mu.Unlock()

		auditSvc.Append("dbops", "migration_plan", planID)
		respondJSON(w, http.StatusOK, migrationPlanResponse{PlanID: planID, FromVersion: req.FromVersion, ToVersion: req.ToVersion, Steps: req.Steps, CreatedAtSec: time.Now().UTC().Unix()})
	}
}

func detectDatabaseMigrationDriftHandler(auditSvc AuditService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req migrationDriftDetectRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}
		req.FromVersion = strings.TrimSpace(req.FromVersion)
		req.ToVersion = strings.TrimSpace(req.ToVersion)
		if req.FromVersion == "" || req.ToVersion == "" || len(req.ExpectedSteps) == 0 {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "from_version, to_version and expected_steps are required"})
			return
		}

		expected := uniqueNonEmptyStrings(req.ExpectedSteps)
		applied := uniqueNonEmptyStrings(req.AppliedSteps)

		appliedSet := make(map[string]struct{}, len(applied))
		for _, step := range applied {
			appliedSet[step] = struct{}{}
		}
		expectedSet := make(map[string]struct{}, len(expected))
		for _, step := range expected {
			expectedSet[step] = struct{}{}
		}

		missing := make([]string, 0)
		for _, step := range expected {
			if _, ok := appliedSet[step]; !ok {
				missing = append(missing, step)
			}
		}

		unexpected := make([]string, 0)
		for _, step := range applied {
			if _, ok := expectedSet[step]; !ok {
				unexpected = append(unexpected, step)
			}
		}

		sort.Strings(missing)
		sort.Strings(unexpected)

		impact := "none"
		if len(missing) > 0 {
			impact = "high"
		} else if len(unexpected) > 0 {
			impact = "medium"
		}

		recommendations := make([]string, 0, 2)
		if len(missing) > 0 {
			recommendations = append(recommendations, "apply missing expected migration steps before release")
		}
		if len(unexpected) > 0 {
			recommendations = append(recommendations, "verify unexpected applied steps against approved migration inventory")
		}

		now := time.Now().UTC()
		adminDBOpsState.mu.Lock()
		reportID := fmt.Sprintf("drift-%06d", adminDBOpsState.nextID)
		adminDBOpsState.nextID++
		adminDBOpsState.mu.Unlock()

		auditSvc.Append("dbops", "migration_drift_detect", reportID)
		respondJSON(w, http.StatusOK, migrationDriftDetectResponse{
			ReportID:           reportID,
			FromVersion:        req.FromVersion,
			ToVersion:          req.ToVersion,
			DriftDetected:      len(missing) > 0 || len(unexpected) > 0,
			MissingExpected:    missing,
			UnexpectedApplied:  unexpected,
			ImpactGrade:        impact,
			Recommendations:    recommendations,
			GeneratedAtUnixSec: now.Unix(),
		})
	}
}

func backupDatabaseHandler(auditSvc AuditService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req backupRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}
		req.BackupID = strings.TrimSpace(req.BackupID)
		if req.BackupID == "" {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "backup_id is required"})
			return
		}
		now := time.Now().UTC()
		adminDBOpsState.mu.Lock()
		adminDBOpsState.backups[req.BackupID] = backupCatalogItem{BackupID: req.BackupID, Reason: strings.TrimSpace(req.Reason), Status: "ready", CreatedAtSec: now.Unix()}
		adminDBOpsState.mu.Unlock()
		auditSvc.Append("dbops", "backup", req.BackupID)
		respondJSON(w, http.StatusCreated, backupResponse{BackupID: req.BackupID, Status: "ready", CreatedAtSec: now.Unix()})
	}
}

func listBackupCatalogHandler() http.HandlerFunc {
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
		adminDBOpsState.mu.Lock()
		items := make([]backupCatalogItem, 0, len(adminDBOpsState.backups))
		for _, item := range adminDBOpsState.backups {
			items = append(items, item)
		}
		adminDBOpsState.mu.Unlock()
		sort.Slice(items, func(i, j int) bool { return items[i].CreatedAtSec > items[j].CreatedAtSec })
		if limit < len(items) {
			items = items[:limit]
		}
		respondJSON(w, http.StatusOK, map[string]any{"items": items})
	}
}

func restoreDatabaseHandler(auditSvc AuditService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req restoreRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}
		req.BackupID = strings.TrimSpace(req.BackupID)
		req.ConfirmToken = strings.TrimSpace(req.ConfirmToken)
		if req.BackupID == "" {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "backup_id is required"})
			return
		}
		if req.ConfirmToken != dbDangerousConfirmToken {
			respondJSON(w, http.StatusForbidden, map[string]string{"error": "confirm_token required for restore"})
			return
		}
		adminDBOpsState.mu.Lock()
		_, ok := adminDBOpsState.backups[req.BackupID]
		adminDBOpsState.mu.Unlock()
		if !ok {
			respondJSON(w, http.StatusNotFound, map[string]string{"error": "backup_id not found"})
			return
		}
		auditSvc.Append("dbops", "restore", req.BackupID)
		respondJSON(w, http.StatusOK, map[string]any{"backup_id": req.BackupID, "status": "restored", "restored_at_unix_sec": time.Now().UTC().Unix()})
	}
}

func executeRestoreDrillHandler(auditSvc AuditService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req restoreDrillRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}
		req.BackupID = strings.TrimSpace(req.BackupID)
		req.ConfirmToken = strings.TrimSpace(req.ConfirmToken)
		if req.BackupID == "" {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "backup_id is required"})
			return
		}
		if req.ConfirmToken != dbDangerousConfirmToken {
			respondJSON(w, http.StatusForbidden, map[string]string{"error": "confirm_token required for restore drill"})
			return
		}
		if req.ExpectedMaxRTOms <= 0 {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "expected_max_rto_ms must be > 0"})
			return
		}
		if req.ExpectedMaxRPOms <= 0 {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "expected_max_rpo_ms must be > 0"})
			return
		}

		now := time.Now().UTC()
		simulatedDurationMs := int64(350)
		simulatedReplicaLagMs := int64(120)
		adminDBOpsState.mu.Lock()
		if _, ok := adminDBOpsState.backups[req.BackupID]; !ok {
			adminDBOpsState.mu.Unlock()
			respondJSON(w, http.StatusNotFound, map[string]string{"error": "backup_id not found"})
			return
		}
		drillID := fmt.Sprintf("drill-%06d", adminDBOpsState.nextID)
		adminDBOpsState.nextID++
		result := restoreDrillResponse{
			DrillID:          drillID,
			BackupID:         req.BackupID,
			Succeeded:        true,
			DurationMs:       simulatedDurationMs,
			ExpectedMaxRTOms: req.ExpectedMaxRTOms,
			RTOCompliant:     simulatedDurationMs <= req.ExpectedMaxRTOms,
			ReplicaLagMs:     simulatedReplicaLagMs,
			ExpectedMaxRPOms: req.ExpectedMaxRPOms,
			RPOCompliant:     simulatedReplicaLagMs <= req.ExpectedMaxRPOms,
			DataCheckPassed:  true,
			PerformedAtSec:   now.Unix(),
		}
		adminDBOpsState.drills = append(adminDBOpsState.drills, result)
		adminDBOpsState.mu.Unlock()

		auditSvc.Append("dbops", "restore_drill", result.DrillID)
		respondJSON(w, http.StatusOK, result)
	}
}

func listRestoreDrillEvidenceHandler() http.HandlerFunc {
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
		adminDBOpsState.mu.Lock()
		items := make([]restoreDrillResponse, len(adminDBOpsState.drills))
		copy(items, adminDBOpsState.drills)
		adminDBOpsState.mu.Unlock()
		sort.Slice(items, func(i, j int) bool { return items[i].PerformedAtSec > items[j].PerformedAtSec })
		if limit < len(items) {
			items = items[:limit]
		}
		respondJSON(w, http.StatusOK, map[string]any{"items": items})
	}
}

func executeControlledSQLHandler(auditSvc AuditService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req controlledSQLRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}
		sqlText := strings.TrimSpace(req.SQL)
		if sqlText == "" {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "sql is required"})
			return
		}
		resolvedClass, err := resolveSQLClass(strings.TrimSpace(req.SQLClass), sqlText)
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		destructive := isDestructiveSQL(sqlText)
		write := isWriteSQL(sqlText)

		safetyResult := "approved"
		switch resolvedClass {
		case sqlClassReadOnly:
			if write || destructive {
				respondJSON(w, http.StatusForbidden, controlledSQLResponse{Allowed: false, SQLClass: resolvedClass, SafetyResult: "class_violation_read_only"})
				return
			}
			safetyResult = "read_only_approved"
		case sqlClassWriteGuarded:
			if destructive {
				respondJSON(w, http.StatusForbidden, controlledSQLResponse{Allowed: false, SQLClass: resolvedClass, SafetyResult: "class_violation_use_destructive_confirmed"})
				return
			}
			if write && strings.TrimSpace(req.ConfirmToken) != dbDangerousConfirmToken {
				respondJSON(w, http.StatusForbidden, controlledSQLResponse{Allowed: false, SQLClass: resolvedClass, SafetyResult: "write_guard_confirmation_required"})
				return
			}
			safetyResult = "write_guarded_approved"
		case sqlClassDestructiveConfirmed:
			if !destructive {
				respondJSON(w, http.StatusBadRequest, controlledSQLResponse{Allowed: false, SQLClass: resolvedClass, SafetyResult: "no_destructive_operation_detected"})
				return
			}
			if !req.AllowDangerous || strings.TrimSpace(req.ConfirmToken) != dbDangerousConfirmToken || strings.TrimSpace(req.ConfirmTokenDual) != dbDangerousDualConfirmToken {
				respondJSON(w, http.StatusForbidden, controlledSQLResponse{Allowed: false, SQLClass: resolvedClass, SafetyResult: "dangerous_sql_dual_confirmation_required"})
				return
			}
			safetyResult = "destructive_confirmed_approved"
		default:
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "unsupported sql_class"})
			return
		}

		adminDBOpsState.mu.Lock()
		execID := fmt.Sprintf("sql-%06d", adminDBOpsState.nextID)
		adminDBOpsState.nextID++
		adminDBOpsState.mu.Unlock()

		auditSvc.Append("dbops", "sql_execute", execID)
		respondJSON(w, http.StatusOK, controlledSQLResponse{Allowed: true, SQLClass: resolvedClass, ExecutionID: execID, SafetyResult: safetyResult})
	}
}

func resolveSQLClass(requestedClass, sqlText string) (string, error) {
	if requestedClass != "" {
		switch requestedClass {
		case sqlClassReadOnly, sqlClassWriteGuarded, sqlClassDestructiveConfirmed:
			return requestedClass, nil
		default:
			return "", fmt.Errorf("sql_class must be one of %s|%s|%s", sqlClassReadOnly, sqlClassWriteGuarded, sqlClassDestructiveConfirmed)
		}
	}
	if isDestructiveSQL(sqlText) {
		return sqlClassDestructiveConfirmed, nil
	}
	if isWriteSQL(sqlText) {
		return sqlClassWriteGuarded, nil
	}
	return sqlClassReadOnly, nil
}

func isDestructiveSQL(sqlText string) bool {
	upper := strings.ToUpper(sqlText)
	return strings.Contains(upper, " DROP ") || strings.HasPrefix(upper, "DROP ") || strings.Contains(upper, " TRUNCATE ") || strings.HasPrefix(upper, "TRUNCATE ") || strings.Contains(upper, " DELETE ") || strings.HasPrefix(upper, "DELETE ")
}

func isWriteSQL(sqlText string) bool {
	upper := strings.ToUpper(sqlText)
	return strings.Contains(upper, " INSERT ") || strings.HasPrefix(upper, "INSERT ") || strings.Contains(upper, " UPDATE ") || strings.HasPrefix(upper, "UPDATE ") || strings.Contains(upper, " MERGE ") || strings.HasPrefix(upper, "MERGE ") || strings.Contains(upper, " UPSERT ") || strings.HasPrefix(upper, "UPSERT ") || strings.Contains(upper, " ALTER ") || strings.HasPrefix(upper, "ALTER ") || strings.Contains(upper, " CREATE ") || strings.HasPrefix(upper, "CREATE ")
}
