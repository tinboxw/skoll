package plugins

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/module/audit"
	"github.com/tinboxw/skoll/internal/module/pluginmgr"
)

type testAPIRegistry struct {
	entries []string
}

func (r *testAPIRegistry) RegisterMany(entries []string) {
	r.entries = append(r.entries, entries...)
}

func newPluginsMux() *http.ServeMux {
	mux := http.NewServeMux()
	NewHandler(pluginmgr.NewService(), audit.NewService()).Register(mux, nil, &testAPIRegistry{})
	return mux
}

func testMarketplaceIndexSignature(source, signedBy string, expiresAtUnix int64, pkgDigests []string) string {
	raw := source + "|" + signedBy + "|" + fmt.Sprintf("%d", expiresAtUnix) + "|" + fmt.Sprintf("%d", len(pkgDigests))
	for _, digest := range pkgDigests {
		raw += "|" + digest
	}
	sum := sha256.Sum256([]byte(raw))
	return "idxsig:" + hex.EncodeToString(sum[:])
}

func TestPluginRoutes_InstallAndToggle(t *testing.T) {
	mux := newPluginsMux()

	installReq := httptest.NewRequest(http.MethodPost, "/admin/v1/plugins/manifests", strings.NewReader(`{"name":"audit-ext","version":"1.0.0","hooks":["on_boot"]}`))
	installReq.Header.Set("Content-Type", "application/json")
	installRR := httptest.NewRecorder()
	mux.ServeHTTP(installRR, installReq)
	if installRR.Code != http.StatusCreated {
		t.Fatalf("expected install status 201, got %d", installRR.Code)
	}

	disableReq := httptest.NewRequest(http.MethodPost, "/admin/v1/plugins/audit-ext/disable", nil)
	disableRR := httptest.NewRecorder()
	mux.ServeHTTP(disableRR, disableReq)
	if disableRR.Code != http.StatusOK {
		t.Fatalf("expected disable status 200, got %d", disableRR.Code)
	}
	if !strings.Contains(disableRR.Body.String(), `"enabled":false`) {
		t.Fatalf("expected plugin disabled, got %s", disableRR.Body.String())
	}

	enableReq := httptest.NewRequest(http.MethodPost, "/admin/v1/plugins/audit-ext/enable", nil)
	enableRR := httptest.NewRecorder()
	mux.ServeHTTP(enableRR, enableReq)
	if enableRR.Code != http.StatusOK {
		t.Fatalf("expected enable status 200, got %d", enableRR.Code)
	}
	if !strings.Contains(enableRR.Body.String(), `"enabled":true`) {
		t.Fatalf("expected plugin enabled, got %s", enableRR.Body.String())
	}

	packageReq := httptest.NewRequest(http.MethodPost, "/admin/v1/plugins/packages/install", strings.NewReader(`{"name":"audit-ext","version":"1.1.0","package_url":"https://example.com/plugins/audit-ext-1.1.0.tgz","package_hash":"sha256:abcd","signature":"sig:sha256:abcd","hooks":["on_boot"]}`))
	packageReq.Header.Set("Content-Type", "application/json")
	packageRR := httptest.NewRecorder()
	mux.ServeHTTP(packageRR, packageReq)
	if packageRR.Code != http.StatusCreated {
		t.Fatalf("expected package install status 201, got %d", packageRR.Code)
	}
	if !strings.Contains(packageRR.Body.String(), `"package_url":"https://example.com/plugins/audit-ext-1.1.0.tgz"`) {
		t.Fatalf("expected package metadata in response, got %s", packageRR.Body.String())
	}

	versionReq := httptest.NewRequest(http.MethodPost, "/admin/v1/plugins/audit-ext/version-check", strings.NewReader(`{"latest_version":"1.2.0"}`))
	versionReq.Header.Set("Content-Type", "application/json")
	versionRR := httptest.NewRecorder()
	mux.ServeHTTP(versionRR, versionReq)
	if versionRR.Code != http.StatusOK {
		t.Fatalf("expected version check status 200, got %d", versionRR.Code)
	}
	if !strings.Contains(versionRR.Body.String(), `"update_available":true`) {
		t.Fatalf("expected update availability in response, got %s", versionRR.Body.String())
	}

	badUpgradeReq := httptest.NewRequest(http.MethodPost, "/admin/v1/plugins/audit-ext/upgrade", strings.NewReader(`{"target_version":"1.2.0","package_url":"https://example.com/plugins/audit-ext-1.2.0.tgz","package_hash":"sha256:efgh","signature":"bad"}`))
	badUpgradeReq.Header.Set("Content-Type", "application/json")
	badUpgradeRR := httptest.NewRecorder()
	mux.ServeHTTP(badUpgradeRR, badUpgradeReq)
	if badUpgradeRR.Code != http.StatusOK {
		t.Fatalf("expected bad signature upgrade to return rollback result status 200, got %d", badUpgradeRR.Code)
	}
	if !strings.Contains(badUpgradeRR.Body.String(), `"rolled_back":true`) {
		t.Fatalf("expected rollback result for failed upgrade, got %s", badUpgradeRR.Body.String())
	}

	upgradeReq := httptest.NewRequest(http.MethodPost, "/admin/v1/plugins/audit-ext/upgrade", strings.NewReader(`{"target_version":"1.2.0","package_url":"https://example.com/plugins/audit-ext-1.2.0.tgz","package_hash":"sha256:efgh","signature":"sig:sha256:efgh"}`))
	upgradeReq.Header.Set("Content-Type", "application/json")
	upgradeRR := httptest.NewRecorder()
	mux.ServeHTTP(upgradeRR, upgradeReq)
	if upgradeRR.Code != http.StatusOK {
		t.Fatalf("expected upgrade status 200, got %d", upgradeRR.Code)
	}
	if !strings.Contains(upgradeRR.Body.String(), `"succeeded":true`) {
		t.Fatalf("expected successful upgrade result, got %s", upgradeRR.Body.String())
	}

	hookRegisterReq := httptest.NewRequest(http.MethodPost, "/admin/v1/plugins/hooks/register", strings.NewReader(`{"name":"on_user_created","namespace":"billing","version":"1.0.0","order":10,"timeout_millis":900,"retry_limit":2,"dead_letter":true}`))
	hookRegisterReq.Header.Set("Content-Type", "application/json")
	hookRegisterRR := httptest.NewRecorder()
	mux.ServeHTTP(hookRegisterRR, hookRegisterReq)
	if hookRegisterRR.Code != http.StatusCreated {
		t.Fatalf("expected hook register status 201, got %d body=%s", hookRegisterRR.Code, hookRegisterRR.Body.String())
	}

	hookListReq := httptest.NewRequest(http.MethodGet, "/admin/v1/plugins/hooks", nil)
	hookListRR := httptest.NewRecorder()
	mux.ServeHTTP(hookListRR, hookListReq)
	if hookListRR.Code != http.StatusOK {
		t.Fatalf("expected hook list status 200, got %d", hookListRR.Code)
	}
	if !strings.Contains(hookListRR.Body.String(), `"namespace":"billing"`) {
		t.Fatalf("expected hook namespace in list, got %s", hookListRR.Body.String())
	}

	hookOrderReq := httptest.NewRequest(http.MethodPost, "/admin/v1/plugins/hooks/billing/on_user_created/order", strings.NewReader(`{"order":30}`))
	hookOrderReq.Header.Set("Content-Type", "application/json")
	hookOrderRR := httptest.NewRecorder()
	mux.ServeHTTP(hookOrderRR, hookOrderReq)
	if hookOrderRR.Code != http.StatusOK {
		t.Fatalf("expected hook order status 200, got %d body=%s", hookOrderRR.Code, hookOrderRR.Body.String())
	}
	if !strings.Contains(hookOrderRR.Body.String(), `"order":30`) {
		t.Fatalf("expected updated hook order, got %s", hookOrderRR.Body.String())
	}

	hookRuntimeReq := httptest.NewRequest(http.MethodPost, "/admin/v1/plugins/hooks/billing/on_user_created/runtime", strings.NewReader(`{"timeout_millis":1200,"retry_limit":3,"dead_letter":false}`))
	hookRuntimeReq.Header.Set("Content-Type", "application/json")
	hookRuntimeRR := httptest.NewRecorder()
	mux.ServeHTTP(hookRuntimeRR, hookRuntimeReq)
	if hookRuntimeRR.Code != http.StatusOK {
		t.Fatalf("expected hook runtime status 200, got %d body=%s", hookRuntimeRR.Code, hookRuntimeRR.Body.String())
	}
	if !strings.Contains(hookRuntimeRR.Body.String(), `"timeout_millis":1200`) || !strings.Contains(hookRuntimeRR.Body.String(), `"dead_letter":false`) {
		t.Fatalf("expected updated runtime policy, got %s", hookRuntimeRR.Body.String())
	}

	hookDisableReq := httptest.NewRequest(http.MethodPost, "/admin/v1/plugins/hooks/billing/on_user_created/disable", nil)
	hookDisableRR := httptest.NewRecorder()
	mux.ServeHTTP(hookDisableRR, hookDisableReq)
	if hookDisableRR.Code != http.StatusOK {
		t.Fatalf("expected hook disable status 200, got %d", hookDisableRR.Code)
	}
	if !strings.Contains(hookDisableRR.Body.String(), `"enabled":false`) {
		t.Fatalf("expected hook disabled, got %s", hookDisableRR.Body.String())
	}

	hookEnableReq := httptest.NewRequest(http.MethodPost, "/admin/v1/plugins/hooks/billing/on_user_created/enable", nil)
	hookEnableRR := httptest.NewRecorder()
	mux.ServeHTTP(hookEnableRR, hookEnableReq)
	if hookEnableRR.Code != http.StatusOK {
		t.Fatalf("expected hook enable status 200, got %d", hookEnableRR.Code)
	}

	hookRuntimeDlqReq := httptest.NewRequest(http.MethodPost, "/admin/v1/plugins/hooks/billing/on_user_created/runtime", strings.NewReader(`{"timeout_millis":1200,"retry_limit":3,"dead_letter":true}`))
	hookRuntimeDlqReq.Header.Set("Content-Type", "application/json")
	hookRuntimeDlqRR := httptest.NewRecorder()
	mux.ServeHTTP(hookRuntimeDlqRR, hookRuntimeDlqReq)
	if hookRuntimeDlqRR.Code != http.StatusOK {
		t.Fatalf("expected hook runtime update status 200, got %d", hookRuntimeDlqRR.Code)
	}

	hookDiagReq := httptest.NewRequest(http.MethodPost, "/admin/v1/plugins/hooks/billing/on_user_created/execute-diagnostic", strings.NewReader(`{"fail_times":10}`))
	hookDiagReq.Header.Set("Content-Type", "application/json")
	hookDiagRR := httptest.NewRecorder()
	mux.ServeHTTP(hookDiagRR, hookDiagReq)
	if hookDiagRR.Code != http.StatusOK {
		t.Fatalf("expected hook diagnostic status 200, got %d body=%s", hookDiagRR.Code, hookDiagRR.Body.String())
	}
	if !strings.Contains(hookDiagRR.Body.String(), `"dead_lettered":true`) {
		t.Fatalf("expected dead-lettered diagnostic result, got %s", hookDiagRR.Body.String())
	}

	dlqReq := httptest.NewRequest(http.MethodGet, "/admin/v1/plugins/hooks/dead-letters", nil)
	dlqRR := httptest.NewRecorder()
	mux.ServeHTTP(dlqRR, dlqReq)
	if dlqRR.Code != http.StatusOK {
		t.Fatalf("expected dead-letter list status 200, got %d", dlqRR.Code)
	}
	if !strings.Contains(dlqRR.Body.String(), `"namespace":"billing"`) {
		t.Fatalf("expected dead-letter payload in response, got %s", dlqRR.Body.String())
	}

	compatReq := httptest.NewRequest(http.MethodPost, "/admin/v1/plugins/compatibility-check", strings.NewReader(`{"name":"audit-ext","version":"1.2.0","dependencies":[{"name":"audit-ext","min_version":"1.0.0"}]}`))
	compatReq.Header.Set("Content-Type", "application/json")
	compatRR := httptest.NewRecorder()
	mux.ServeHTTP(compatRR, compatReq)
	if compatRR.Code != http.StatusOK {
		t.Fatalf("expected compatibility status 200, got %d body=%s", compatRR.Code, compatRR.Body.String())
	}
	if !strings.Contains(compatRR.Body.String(), `"compatible":false`) {
		t.Fatalf("expected compatibility blockers for self dependency, got %s", compatRR.Body.String())
	}

	removeReq := httptest.NewRequest(http.MethodPost, "/admin/v1/plugins/audit-ext/remove", nil)
	removeRR := httptest.NewRecorder()
	mux.ServeHTTP(removeRR, removeReq)
	if removeRR.Code != http.StatusOK {
		t.Fatalf("expected remove status 200, got %d body=%s", removeRR.Code, removeRR.Body.String())
	}
	if !strings.Contains(removeRR.Body.String(), `"succeeded":true`) {
		t.Fatalf("expected successful remove result, got %s", removeRR.Body.String())
	}

	removeAgainReq := httptest.NewRequest(http.MethodPost, "/admin/v1/plugins/audit-ext/remove", nil)
	removeAgainRR := httptest.NewRecorder()
	mux.ServeHTTP(removeAgainRR, removeAgainReq)
	if removeAgainRR.Code != http.StatusOK {
		t.Fatalf("expected idempotent remove status 200, got %d", removeAgainRR.Code)
	}
	if !strings.Contains(removeAgainRR.Body.String(), `"idempotent":true`) {
		t.Fatalf("expected idempotent remove result, got %s", removeAgainRR.Body.String())
	}

	trustRootsReq := httptest.NewRequest(http.MethodPut, "/admin/v1/plugins/marketplace/trust-roots", strings.NewReader(`{"roots":["corp-root","backup-root","corp-root"]}`))
	trustRootsReq.Header.Set("Content-Type", "application/json")
	trustRootsRR := httptest.NewRecorder()
	mux.ServeHTTP(trustRootsRR, trustRootsReq)
	if trustRootsRR.Code != http.StatusOK {
		t.Fatalf("expected trust roots status 200, got %d body=%s", trustRootsRR.Code, trustRootsRR.Body.String())
	}

	expiresAt := time.Now().UTC().Add(15 * time.Minute)
	payload := `{"source":"official","signed_by":"corp-root","signature":"%s","expires_at_unix_sec":%d,"packages":[{"name":"audit-ext","version":"1.2.0","package_url":"https://example.com/audit-ext-1.2.0.tgz","package_hash":"sha256:abc"}]}`
	sig := testMarketplaceIndexSignature("official", "corp-root", expiresAt.Unix(), []string{"audit-ext@1.2.0#sha256:abc"})
	ingestReq := httptest.NewRequest(http.MethodPost, "/admin/v1/plugins/marketplace/index/ingest", strings.NewReader(fmt.Sprintf(payload, sig, expiresAt.Unix())))
	ingestReq.Header.Set("Content-Type", "application/json")
	ingestRR := httptest.NewRecorder()
	mux.ServeHTTP(ingestRR, ingestReq)
	if ingestRR.Code != http.StatusOK {
		t.Fatalf("expected marketplace ingest status 200, got %d body=%s", ingestRR.Code, ingestRR.Body.String())
	}
	if !strings.Contains(ingestRR.Body.String(), `"accepted":true`) {
		t.Fatalf("expected accepted ingest result, got %s", ingestRR.Body.String())
	}

	sourcesReq := httptest.NewRequest(http.MethodGet, "/admin/v1/plugins/marketplace/index/sources", nil)
	sourcesRR := httptest.NewRecorder()
	mux.ServeHTTP(sourcesRR, sourcesReq)
	if sourcesRR.Code != http.StatusOK {
		t.Fatalf("expected index sources status 200, got %d", sourcesRR.Code)
	}
	if !strings.Contains(sourcesRR.Body.String(), `"source":"official"`) {
		t.Fatalf("expected indexed source in response, got %s", sourcesRR.Body.String())
	}

	solverReq := httptest.NewRequest(http.MethodPost, "/admin/v1/plugins/dependency-solver/resolve", strings.NewReader(`{"items":[{"name":"billing-ext","version":"1.0.0","dependencies":[{"name":"audit-ext","min_version":"2.0.0"}]},{"name":"report-ext","version":"1.0.0","dependencies":[{"name":"missing-ext","min_version":"1.0.0"}]}]}`))
	solverReq.Header.Set("Content-Type", "application/json")
	solverRR := httptest.NewRecorder()
	mux.ServeHTTP(solverRR, solverReq)
	if solverRR.Code != http.StatusOK {
		t.Fatalf("expected dependency solver status 200, got %d body=%s", solverRR.Code, solverRR.Body.String())
	}
	if !strings.Contains(solverRR.Body.String(), `"deterministic":true`) || !strings.Contains(solverRR.Body.String(), `"conflicts"`) {
		t.Fatalf("expected dependency solver diagnostics payload, got %s", solverRR.Body.String())
	}

	reinstallReq := httptest.NewRequest(http.MethodPost, "/admin/v1/plugins/manifests", strings.NewReader(`{"name":"audit-ext","version":"1.2.0","hooks":["on_boot"]}`))
	reinstallReq.Header.Set("Content-Type", "application/json")
	reinstallRR := httptest.NewRecorder()
	mux.ServeHTTP(reinstallRR, reinstallReq)
	if reinstallRR.Code != http.StatusCreated {
		t.Fatalf("expected reinstall status 201, got %d body=%s", reinstallRR.Code, reinstallRR.Body.String())
	}

	txUpgradeReq := httptest.NewRequest(http.MethodPost, "/admin/v1/plugins/audit-ext/upgrade/transaction", strings.NewReader(`{"transaction_id":"tx-api-1","target_version":"1.3.0","package_url":"https://example.com/plugins/audit-ext-1.3.0.tgz","package_hash":"sha256:aa130","signature":"sig:sha256:aa130"}`))
	txUpgradeReq.Header.Set("Content-Type", "application/json")
	txUpgradeRR := httptest.NewRecorder()
	mux.ServeHTTP(txUpgradeRR, txUpgradeReq)
	if txUpgradeRR.Code != http.StatusOK {
		t.Fatalf("expected transactional upgrade status 200, got %d body=%s", txUpgradeRR.Code, txUpgradeRR.Body.String())
	}
	if !strings.Contains(txUpgradeRR.Body.String(), `"transaction_id":"tx-api-1"`) || !strings.Contains(txUpgradeRR.Body.String(), `"succeeded":true`) {
		t.Fatalf("expected transactional upgrade success payload, got %s", txUpgradeRR.Body.String())
	}

	provenanceReq := httptest.NewRequest(http.MethodGet, "/admin/v1/plugins/upgrade/provenance?limit=1", nil)
	provenanceRR := httptest.NewRecorder()
	mux.ServeHTTP(provenanceRR, provenanceReq)
	if provenanceRR.Code != http.StatusOK {
		t.Fatalf("expected provenance list status 200, got %d body=%s", provenanceRR.Code, provenanceRR.Body.String())
	}
	if !strings.Contains(provenanceRR.Body.String(), `"transaction_id":"tx-api-1"`) {
		t.Fatalf("expected provenance payload to include tx-api-1, got %s", provenanceRR.Body.String())
	}

	provenanceBadLimitReq := httptest.NewRequest(http.MethodGet, "/admin/v1/plugins/upgrade/provenance?limit=0", nil)
	provenanceBadLimitRR := httptest.NewRecorder()
	mux.ServeHTTP(provenanceBadLimitRR, provenanceBadLimitReq)
	if provenanceBadLimitRR.Code != http.StatusBadRequest {
		t.Fatalf("expected provenance bad limit status 400, got %d", provenanceBadLimitRR.Code)
	}
}
