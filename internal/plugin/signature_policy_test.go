package plugin

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestCheckSignatureCoverageRequiresManifestBackendAndFrontendAssets(t *testing.T) {
	dir := setupSignatureCoveragePlugin(t)
	writeSignatureAssetManifest(t, dir, []string{
		"plugin.yaml",
		"backend/main.go",
		"frontend/dist/index.html",
		"frontend/dist/assets/app.js",
	})

	report, err := CheckSignatureCoverage(dir, Info{
		ID:     "signed-demo",
		UIMode: UIModeSeparated,
	}, SignatureCoveragePolicy{})
	if err != nil {
		t.Fatalf("signature coverage: %v", err)
	}
	if report.ManifestDigest == "" || report.AssetManifestDigest == "" {
		t.Fatalf("expected manifest digests, got %+v", report)
	}
	if report.BackendEntry != "backend/main.go" {
		t.Fatalf("expected backend entry, got %+v", report)
	}
	if len(report.FrontendAssets) != 2 {
		t.Fatalf("expected frontend assets, got %+v", report.FrontendAssets)
	}
}

func TestCheckSignatureCoverageBlocksMissingFrontendAsset(t *testing.T) {
	dir := setupSignatureCoveragePlugin(t)
	writeSignatureAssetManifest(t, dir, []string{
		"plugin.yaml",
		"backend/main.go",
		"frontend/dist/index.html",
	})

	report, err := CheckSignatureCoverage(dir, Info{
		ID:     "signed-demo",
		UIMode: UIModeSeparated,
	}, SignatureCoveragePolicy{})
	if !errors.Is(err, ErrSignatureCoverage) {
		t.Fatalf("expected signature coverage error, got %v", err)
	}
	if len(report.Missing) != 1 || report.Missing[0] != "frontend/dist/assets/app.js" {
		t.Fatalf("expected missing frontend asset, got %+v", report)
	}
}

func TestCheckSignatureCoverageBlocksTamperedDigest(t *testing.T) {
	dir := setupSignatureCoveragePlugin(t)
	writeSignatureAssetManifest(t, dir, []string{
		"plugin.yaml",
		"backend/main.go",
		"frontend/dist/index.html",
		"frontend/dist/assets/app.js",
	})
	if err := os.WriteFile(filepath.Join(dir, "frontend", "dist", "assets", "app.js"), []byte("tampered"), 0o644); err != nil {
		t.Fatalf("tamper asset: %v", err)
	}

	report, err := CheckSignatureCoverage(dir, Info{
		ID:     "signed-demo",
		UIMode: UIModeSeparated,
	}, SignatureCoveragePolicy{})
	if !errors.Is(err, ErrSignatureCoverage) {
		t.Fatalf("expected signature coverage error, got %v", err)
	}
	if len(report.Mismatched) != 1 || report.Mismatched[0] != "frontend/dist/assets/app.js" {
		t.Fatalf("expected mismatched frontend asset, got %+v", report)
	}
}

func setupSignatureCoveragePlugin(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "plugin.yaml"), "id: signed-demo\nname: Signed Demo\nversion: 1.0.0\n")
	writeFile(t, filepath.Join(dir, "backend", "main.go"), "package main\nfunc main() {}\n")
	writeFile(t, filepath.Join(dir, "frontend", "dist", "index.html"), "<div id=\"app\"></div>\n")
	writeFile(t, filepath.Join(dir, "frontend", "dist", "assets", "app.js"), "console.log('ok')\n")
	return dir
}

func writeSignatureAssetManifest(t *testing.T, dir string, files []string) {
	t.Helper()
	entries := make([]SignatureAssetEntry, 0, len(files))
	for _, rel := range files {
		digest, err := fileSHA256(filepath.Join(dir, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("digest %s: %v", rel, err)
		}
		entries = append(entries, SignatureAssetEntry{Path: rel, SHA256: digest})
	}
	raw, err := json.MarshalIndent(SignatureAssetManifest{
		SchemaVersion: "v1",
		Files:         entries,
	}, "", "  ")
	if err != nil {
		t.Fatalf("marshal asset manifest: %v", err)
	}
	writeFile(t, filepath.Join(dir, ".skoll", "signature-assets.json"), string(raw)+"\n")
}

func writeFile(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
