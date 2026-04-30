package pluginmgr

import (
	"testing"
	"time"
)

func TestServiceInstallEnableDisable(t *testing.T) {
	svc := NewService()
	installed := svc.Install("audit-ext", "1.0.0", []string{"on_boot", "on_shutdown"})
	if !installed.Enabled {
		t.Fatalf("expected plugin enabled on install")
	}

	disabled, err := svc.Disable("audit-ext")
	if err != nil {
		t.Fatalf("disable plugin failed: %v", err)
	}
	if disabled.Enabled {
		t.Fatalf("expected plugin disabled")
	}

	enabled, err := svc.Enable("audit-ext")
	if err != nil {
		t.Fatalf("enable plugin failed: %v", err)
	}
	if !enabled.Enabled {
		t.Fatalf("expected plugin enabled")
	}
}

func TestServiceInstallPackageAndVersionCheck(t *testing.T) {
	svc := NewService()

	installed, err := svc.InstallPackageVerified("billing-ext", "1.2.3", "https://example.com/plugins/billing-ext-1.2.3.tgz", "sha256:abc123", "sig:sha256:abc123", nil, []string{"on_boot"})
	if err != nil {
		t.Fatalf("install package failed: %v", err)
	}
	if installed.PackageURL == "" || installed.PackageHash == "" {
		t.Fatalf("expected package metadata to be recorded")
	}

	result, err := svc.CheckVersion("billing-ext", "1.3.0")
	if err != nil {
		t.Fatalf("check version failed: %v", err)
	}
	if !result.UpdateAvailable {
		t.Fatalf("expected update to be available")
	}

	upToDate, err := svc.CheckVersion("billing-ext", "1.2.3")
	if err != nil {
		t.Fatalf("check version failed: %v", err)
	}
	if upToDate.UpdateAvailable {
		t.Fatalf("expected no update for same version")
	}
}

func TestServiceInstallPackageVerified_InvalidSignature(t *testing.T) {
	svc := NewService()
	if _, err := svc.InstallPackageVerified("billing-ext", "1.2.3", "https://example.com/plugins/billing-ext-1.2.3.tgz", "sha256:abc123", "bad-signature", nil, []string{"on_boot"}); err != ErrPluginSignatureInvalid {
		t.Fatalf("expected ErrPluginSignatureInvalid, got %v", err)
	}
}

func TestServiceUpgradePackage_DependencyAndRollback(t *testing.T) {
	svc := NewService()
	_, err := svc.InstallPackageVerified("core-ext", "1.0.0", "https://example.com/plugins/core-ext-1.0.0.tgz", "sha256:core100", "sig:sha256:core100", nil, []string{"on_boot"})
	if err != nil {
		t.Fatalf("install core failed: %v", err)
	}
	_, err = svc.InstallPackageVerified("feature-ext", "1.0.0", "https://example.com/plugins/feature-ext-1.0.0.tgz", "sha256:feat100", "sig:sha256:feat100", []Dependency{{Name: "core-ext", MinVersion: "1.0.0"}}, []string{"on_boot"})
	if err != nil {
		t.Fatalf("install feature failed: %v", err)
	}

	result, err := svc.UpgradePackage("feature-ext", "1.1.0", "https://example.com/plugins/feature-ext-1.1.0.tgz", "sha256:feat110", "bad-signature", nil, []string{"on_boot"})
	if err != nil {
		t.Fatalf("upgrade should return rollback result, got error %v", err)
	}
	if result.Succeeded || !result.RolledBack {
		t.Fatalf("expected failed upgrade with rollback, got %+v", result)
	}

	current, err := svc.Get("feature-ext")
	if err != nil {
		t.Fatalf("get plugin failed: %v", err)
	}
	if current.Version != "1.0.0" {
		t.Fatalf("expected rollback to previous version 1.0.0, got %s", current.Version)
	}

	result, err = svc.UpgradePackage("feature-ext", "1.1.0", "https://example.com/plugins/feature-ext-1.1.0.tgz", "sha256:feat110", "sig:sha256:feat110", []Dependency{{Name: "core-ext", MinVersion: "1.0.0"}}, []string{"on_boot"})
	if err != nil {
		t.Fatalf("upgrade failed: %v", err)
	}
	if !result.Succeeded || result.RolledBack {
		t.Fatalf("expected successful upgrade result, got %+v", result)
	}
}

func BenchmarkServiceInstallPackageVerified(b *testing.B) {
	svc := NewService()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		name := "bench-plugin-" + string(rune('a'+(i%26)))
		_, err := svc.InstallPackageVerified(name, "1.0.0", "https://example.com/plugins/bench.tgz", "sha256:bench", "sig:sha256:bench", nil, []string{"on_boot"})
		if err != nil {
			b.Fatalf("install package verified failed: %v", err)
		}
	}
}

func TestServiceHookRegistryLifecycle(t *testing.T) {
	svc := NewService()

	hook, err := svc.RegisterHook("onUserCreated", "billing", "1.0.0", 10, 800, 2, true)
	if err != nil {
		t.Fatalf("register hook failed: %v", err)
	}
	if !hook.Enabled {
		t.Fatalf("expected hook enabled on register")
	}
	if hook.Namespace != "billing" || hook.Name != "onUserCreated" {
		t.Fatalf("unexpected hook identity: %+v", hook)
	}

	updated, err := svc.SetHookOrder("onUserCreated", "billing", 30)
	if err != nil {
		t.Fatalf("set hook order failed: %v", err)
	}
	if updated.Order != 30 {
		t.Fatalf("expected order=30, got %d", updated.Order)
	}

	runtimeUpdated, err := svc.SetHookRuntimePolicy("onUserCreated", "billing", 1200, 3, false)
	if err != nil {
		t.Fatalf("set runtime policy failed: %v", err)
	}
	if runtimeUpdated.TimeoutMillis != 1200 || runtimeUpdated.RetryLimit != 3 || runtimeUpdated.DeadLetter {
		t.Fatalf("unexpected runtime update: %+v", runtimeUpdated)
	}

	disabled, err := svc.SetHookEnabled("onUserCreated", "billing", false)
	if err != nil {
		t.Fatalf("disable hook failed: %v", err)
	}
	if disabled.Enabled {
		t.Fatalf("expected hook disabled")
	}
}

func TestServiceHookRegistryValidationAndIsolation(t *testing.T) {
	svc := NewService()

	if _, err := svc.RegisterHook("", "billing", "1.0.0", 0, 500, 1, true); err != ErrHookNameRequired {
		t.Fatalf("expected ErrHookNameRequired, got %v", err)
	}
	if _, err := svc.RegisterHook("onCreate", "", "1.0.0", 0, 500, 1, true); err != ErrHookNamespaceRequired {
		t.Fatalf("expected ErrHookNamespaceRequired, got %v", err)
	}

	_, err := svc.RegisterHook("onCreate", "moduleA", "1.0.0", 0, 500, 1, true)
	if err != nil {
		t.Fatalf("register moduleA hook failed: %v", err)
	}
	_, err = svc.RegisterHook("onCreate", "moduleB", "1.0.0", 0, 500, 1, true)
	if err != nil {
		t.Fatalf("register moduleB hook failed: %v", err)
	}

	if _, err := svc.SetHookEnabled("onCreate", "moduleC", false); err != ErrHookNotFound {
		t.Fatalf("expected ErrHookNotFound for unknown namespace, got %v", err)
	}

	all := svc.ListHooks()
	if len(all) != 2 {
		t.Fatalf("expected two isolated hook records, got %d", len(all))
	}
}

func TestServiceLifecycleRemoveAndCompatibilityCheck(t *testing.T) {
	svc := NewService()
	_, err := svc.InstallPackageVerified("core-ext", "1.0.0", "https://example.com/plugins/core-ext-1.0.0.tgz", "sha256:core100", "sig:sha256:core100", nil, []string{"on_boot"})
	if err != nil {
		t.Fatalf("install core failed: %v", err)
	}

	compatible := svc.CheckCompatibility("billing-ext", "1.2.0", []Dependency{{Name: "core-ext", MinVersion: "1.0.0"}})
	if !compatible.Compatible {
		t.Fatalf("expected compatibility pass, got %+v", compatible)
	}

	incompatible := svc.CheckCompatibility("billing-ext", "1.2.0", []Dependency{{Name: "core-ext", MinVersion: "2.0.0"}})
	if incompatible.Compatible || len(incompatible.Blockers) == 0 {
		t.Fatalf("expected compatibility blockers, got %+v", incompatible)
	}

	removeExisting := svc.Remove("core-ext")
	if !removeExisting.Succeeded || removeExisting.Idempotent {
		t.Fatalf("expected successful non-idempotent remove, got %+v", removeExisting)
	}

	removeAgain := svc.Remove("core-ext")
	if !removeAgain.Succeeded || !removeAgain.Idempotent {
		t.Fatalf("expected idempotent remove for missing plugin, got %+v", removeAgain)
	}
}

func TestServiceHookExecutionDiagnosticAndDeadLetter(t *testing.T) {
	svc := NewService()
	_, err := svc.RegisterHook("on_order_paid", "billing", "1.0.0", 1, 500, 2, true)
	if err != nil {
		t.Fatalf("register hook failed: %v", err)
	}

	success, err := svc.ExecuteHookDiagnostic("on_order_paid", "billing", 1)
	if err != nil {
		t.Fatalf("execute diagnostic failed: %v", err)
	}
	if !success.Success || success.Attempts != 2 {
		t.Fatalf("expected success at second attempt, got %+v", success)
	}

	failed, err := svc.ExecuteHookDiagnostic("on_order_paid", "billing", 5)
	if err != nil {
		t.Fatalf("execute diagnostic failed: %v", err)
	}
	if failed.Success || !failed.DeadLettered {
		t.Fatalf("expected dead-lettered failed execution, got %+v", failed)
	}

	dlq := svc.ListHookDeadLetters()
	if len(dlq) == 0 {
		t.Fatalf("expected dead-letter records")
	}
	if dlq[len(dlq)-1].Name != "on_order_paid" {
		t.Fatalf("unexpected dead-letter record: %+v", dlq[len(dlq)-1])
	}
}

func TestServiceMarketplaceTrustAndSignedIndexIngest(t *testing.T) {
	svc := NewService()
	roots := svc.SetMarketplaceTrustRoots([]string{"corp-root", "corp-root", "backup-root"})
	if len(roots) != 2 {
		t.Fatalf("expected deduplicated trust roots, got %v", roots)
	}

	expiresAt := time.Now().UTC().Add(10 * time.Minute)
	pkgs := []MarketplaceIndexPackage{{Name: "audit-ext", Version: "1.2.0", PackageURL: "https://example.com/audit-ext.tgz", PackageHash: "sha256:abc"}}
	signature := signMarketplaceIndex("official", "corp-root", expiresAt, pkgs)

	result, err := svc.IngestMarketplaceIndex("official", "corp-root", signature, expiresAt, pkgs, time.Now().UTC())
	if err != nil {
		t.Fatalf("ingest marketplace index failed: %v", err)
	}
	if !result.Accepted || result.PackageCount != 1 {
		t.Fatalf("unexpected ingest result: %+v", result)
	}

	_, err = svc.IngestMarketplaceIndex("official", "unknown-root", signature, expiresAt, pkgs, time.Now().UTC())
	if err != ErrMarketplaceTrustRootNotFound {
		t.Fatalf("expected trust root error, got %v", err)
	}

	_, err = svc.IngestMarketplaceIndex("official", "corp-root", "bad-signature", expiresAt, pkgs, time.Now().UTC())
	if err != ErrMarketplaceIndexSignatureInvalid {
		t.Fatalf("expected signature invalid error, got %v", err)
	}
}
