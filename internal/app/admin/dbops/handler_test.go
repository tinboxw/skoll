package dbops

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tinboxw/skoll/internal/module/audit"
)

type testAPIRegistry struct {
	entries []string
}

func (r *testAPIRegistry) RegisterMany(entries []string) {
	r.entries = append(r.entries, entries...)
}

func newDBOpsMux() *http.ServeMux {
	mux := http.NewServeMux()
	NewHandler(audit.NewService()).Register(mux, nil, &testAPIRegistry{})
	return mux
}

func TestDatabaseOpsGovernanceRoutes(t *testing.T) {
	mux := newDBOpsMux()

	planReq := httptest.NewRequest(http.MethodPost, "/admin/v1/db/migrations/plan", strings.NewReader(`{"from_version":"2026.04","to_version":"2026.05","steps":["add_table_users","add_index_users_email"]}`))
	planReq.Header.Set("Content-Type", "application/json")
	planRR := httptest.NewRecorder()
	mux.ServeHTTP(planRR, planReq)
	if planRR.Code != http.StatusOK {
		t.Fatalf("expected migration plan status 200, got %d", planRR.Code)
	}

	driftReq := httptest.NewRequest(http.MethodPost, "/admin/v1/db/migrations/drift-detect", strings.NewReader(`{"from_version":"2026.04","to_version":"2026.05","expected_steps":["add_table_users","add_index_users_email"],"applied_steps":["add_table_users","hotfix_sessions_index"]}`))
	driftReq.Header.Set("Content-Type", "application/json")
	driftRR := httptest.NewRecorder()
	mux.ServeHTTP(driftRR, driftReq)
	if driftRR.Code != http.StatusOK {
		t.Fatalf("expected drift detect status 200, got %d body=%s", driftRR.Code, driftRR.Body.String())
	}
	if !strings.Contains(driftRR.Body.String(), `"drift_detected":true`) || !strings.Contains(driftRR.Body.String(), `"impact_grade":"high"`) {
		t.Fatalf("expected drift detected payload with impact grade, got %s", driftRR.Body.String())
	}

	driftBadReq := httptest.NewRequest(http.MethodPost, "/admin/v1/db/migrations/drift-detect", strings.NewReader(`{"from_version":"2026.04","to_version":"2026.05"}`))
	driftBadReq.Header.Set("Content-Type", "application/json")
	driftBadRR := httptest.NewRecorder()
	mux.ServeHTTP(driftBadRR, driftBadReq)
	if driftBadRR.Code != http.StatusBadRequest {
		t.Fatalf("expected drift detect validation status 400, got %d", driftBadRR.Code)
	}

	backupReq := httptest.NewRequest(http.MethodPost, "/admin/v1/db/backup", strings.NewReader(`{"backup_id":"bk-001","reason":"pre-release"}`))
	backupReq.Header.Set("Content-Type", "application/json")
	backupRR := httptest.NewRecorder()
	mux.ServeHTTP(backupRR, backupReq)
	if backupRR.Code != http.StatusCreated {
		t.Fatalf("expected backup status 201, got %d", backupRR.Code)
	}

	catalogReq := httptest.NewRequest(http.MethodGet, "/admin/v1/db/backups/catalog?limit=10", nil)
	catalogRR := httptest.NewRecorder()
	mux.ServeHTTP(catalogRR, catalogReq)
	if catalogRR.Code != http.StatusOK {
		t.Fatalf("expected backup catalog status 200, got %d body=%s", catalogRR.Code, catalogRR.Body.String())
	}
	if !strings.Contains(catalogRR.Body.String(), `"backup_id":"bk-001"`) {
		t.Fatalf("expected backup catalog item, got %s", catalogRR.Body.String())
	}

	restoreDrillForbiddenReq := httptest.NewRequest(http.MethodPost, "/admin/v1/db/restore/drills", strings.NewReader(`{"backup_id":"bk-001","confirm_token":"","expected_max_rto_ms":500,"expected_max_rpo_ms":300}`))
	restoreDrillForbiddenReq.Header.Set("Content-Type", "application/json")
	restoreDrillForbiddenRR := httptest.NewRecorder()
	mux.ServeHTTP(restoreDrillForbiddenRR, restoreDrillForbiddenReq)
	if restoreDrillForbiddenRR.Code != http.StatusForbidden {
		t.Fatalf("expected restore drill forbidden status 403, got %d", restoreDrillForbiddenRR.Code)
	}

	restoreDrillReq := httptest.NewRequest(http.MethodPost, "/admin/v1/db/restore/drills", strings.NewReader(`{"backup_id":"bk-001","confirm_token":"I_UNDERSTAND","expected_max_rto_ms":500,"expected_max_rpo_ms":300}`))
	restoreDrillReq.Header.Set("Content-Type", "application/json")
	restoreDrillRR := httptest.NewRecorder()
	mux.ServeHTTP(restoreDrillRR, restoreDrillReq)
	if restoreDrillRR.Code != http.StatusOK {
		t.Fatalf("expected restore drill status 200, got %d body=%s", restoreDrillRR.Code, restoreDrillRR.Body.String())
	}
	if !strings.Contains(restoreDrillRR.Body.String(), `"rto_compliant":true`) || !strings.Contains(restoreDrillRR.Body.String(), `"rpo_compliant":true`) || !strings.Contains(restoreDrillRR.Body.String(), `"data_check_passed":true`) {
		t.Fatalf("expected restore drill RTO/RPO/data-check payload, got %s", restoreDrillRR.Body.String())
	}

	restoreDrillListReq := httptest.NewRequest(http.MethodGet, "/admin/v1/db/restore/drills?limit=10", nil)
	restoreDrillListRR := httptest.NewRecorder()
	mux.ServeHTTP(restoreDrillListRR, restoreDrillListReq)
	if restoreDrillListRR.Code != http.StatusOK {
		t.Fatalf("expected restore drill list status 200, got %d body=%s", restoreDrillListRR.Code, restoreDrillListRR.Body.String())
	}
	if !strings.Contains(restoreDrillListRR.Body.String(), `"drill_id":"drill-`) {
		t.Fatalf("expected drill evidence list payload, got %s", restoreDrillListRR.Body.String())
	}

	restoreForbiddenReq := httptest.NewRequest(http.MethodPost, "/admin/v1/db/restore", strings.NewReader(`{"backup_id":"bk-001","confirm_token":""}`))
	restoreForbiddenReq.Header.Set("Content-Type", "application/json")
	restoreForbiddenRR := httptest.NewRecorder()
	mux.ServeHTTP(restoreForbiddenRR, restoreForbiddenReq)
	if restoreForbiddenRR.Code != http.StatusForbidden {
		t.Fatalf("expected restore forbidden status 403, got %d", restoreForbiddenRR.Code)
	}

	restoreReq := httptest.NewRequest(http.MethodPost, "/admin/v1/db/restore", strings.NewReader(`{"backup_id":"bk-001","confirm_token":"I_UNDERSTAND"}`))
	restoreReq.Header.Set("Content-Type", "application/json")
	restoreRR := httptest.NewRecorder()
	mux.ServeHTTP(restoreRR, restoreReq)
	if restoreRR.Code != http.StatusOK {
		t.Fatalf("expected restore status 200, got %d", restoreRR.Code)
	}

	sqlBlockedReq := httptest.NewRequest(http.MethodPost, "/admin/v1/db/sql/execute", strings.NewReader(`{"sql":"DROP TABLE users"}`))
	sqlBlockedReq.Header.Set("Content-Type", "application/json")
	sqlBlockedRR := httptest.NewRecorder()
	mux.ServeHTTP(sqlBlockedRR, sqlBlockedReq)
	if sqlBlockedRR.Code != http.StatusForbidden {
		t.Fatalf("expected dangerous sql blocked status 403, got %d", sqlBlockedRR.Code)
	}

	readOnlyReq := httptest.NewRequest(http.MethodPost, "/admin/v1/db/sql/execute", strings.NewReader(`{"sql":"SELECT 1","sql_class":"read_only"}`))
	readOnlyReq.Header.Set("Content-Type", "application/json")
	readOnlyRR := httptest.NewRecorder()
	mux.ServeHTTP(readOnlyRR, readOnlyReq)
	if readOnlyRR.Code != http.StatusOK || !strings.Contains(readOnlyRR.Body.String(), `"sql_class":"read_only"`) {
		t.Fatalf("expected read-only sql approved, got status=%d body=%s", readOnlyRR.Code, readOnlyRR.Body.String())
	}

	writeGuardMissingConfirmReq := httptest.NewRequest(http.MethodPost, "/admin/v1/db/sql/execute", strings.NewReader(`{"sql":"UPDATE users SET status='ok'","sql_class":"write_guarded"}`))
	writeGuardMissingConfirmReq.Header.Set("Content-Type", "application/json")
	writeGuardMissingConfirmRR := httptest.NewRecorder()
	mux.ServeHTTP(writeGuardMissingConfirmRR, writeGuardMissingConfirmReq)
	if writeGuardMissingConfirmRR.Code != http.StatusForbidden {
		t.Fatalf("expected write-guarded confirmation required status 403, got %d", writeGuardMissingConfirmRR.Code)
	}

	writeGuardReq := httptest.NewRequest(http.MethodPost, "/admin/v1/db/sql/execute", strings.NewReader(`{"sql":"UPDATE users SET status='ok'","sql_class":"write_guarded","confirm_token":"I_UNDERSTAND"}`))
	writeGuardReq.Header.Set("Content-Type", "application/json")
	writeGuardRR := httptest.NewRecorder()
	mux.ServeHTTP(writeGuardRR, writeGuardReq)
	if writeGuardRR.Code != http.StatusOK || !strings.Contains(writeGuardRR.Body.String(), `"sql_class":"write_guarded"`) {
		t.Fatalf("expected write-guarded sql approved, got status=%d body=%s", writeGuardRR.Code, writeGuardRR.Body.String())
	}

	sqlAllowedReq := httptest.NewRequest(http.MethodPost, "/admin/v1/db/sql/execute", strings.NewReader(`{"sql":"DELETE FROM sessions WHERE expired=1","sql_class":"destructive_confirmed","allow_dangerous":true,"confirm_token":"I_UNDERSTAND","confirm_token_dual":"CONFIRM_DESTRUCTIVE_SQL"}`))
	sqlAllowedReq.Header.Set("Content-Type", "application/json")
	sqlAllowedRR := httptest.NewRecorder()
	mux.ServeHTTP(sqlAllowedRR, sqlAllowedReq)
	if sqlAllowedRR.Code != http.StatusOK {
		t.Fatalf("expected controlled sql status 200, got %d", sqlAllowedRR.Code)
	}
	if !strings.Contains(sqlAllowedRR.Body.String(), `"allowed":true`) || !strings.Contains(sqlAllowedRR.Body.String(), `"sql_class":"destructive_confirmed"`) {
		t.Fatalf("expected allowed sql response, got %s", sqlAllowedRR.Body.String())
	}
}
