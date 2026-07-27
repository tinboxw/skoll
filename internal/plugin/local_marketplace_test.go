package plugin

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestLocalMarketplaceServiceListsPluginsAndPackages(t *testing.T) {
	root := t.TempDir()
	pluginDir := filepath.Join(root, "demo")
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		t.Fatalf("mkdir plugin: %v", err)
	}
	manifest := []byte(`id: demo
name: Demo Plugin
version: 0.2.0
api_version: v1
migration_version: v0.2.0
ui_mode: frontend_only
sign_algo: RSA-SHA256
sign_timestamp: 2026-06-29T00:00:00Z
sign_value: signed
i18n_locales:
  - zh-CN
permissions:
  - key: demo.export
    type: api
    module: demo
    name: Export demo
    risk: high
data:
  namespace: demo
  migration_version: v0.2.0
  uninstall_policy: drop
  rollback_policy: manual
  tables:
    - name: exports
      primary_key: id
      columns: id, code
      indexes: idx_demo_exports_code(code)
api:
  routes:
    - method: GET
      path: /v1/plugins/demo/api/exports
      summary: List demo exports
      permission: demo.exports.read
      audit_action: demo.exports.read
`)
	if err := os.WriteFile(filepath.Join(pluginDir, "plugin.yaml"), manifest, 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	dist := filepath.Join(root, "_dist")
	if err := os.MkdirAll(dist, 0o755); err != nil {
		t.Fatalf("mkdir dist: %v", err)
	}
	writeZip(t, filepath.Join(dist, "demo-0.2.0.zip"))

	catalog, err := NewLocalMarketplaceService(nil).List(root)
	if err != nil {
		t.Fatalf("list local marketplace: %v", err)
	}
	if catalog.PluginsRoot != filepath.Clean(root) {
		t.Fatalf("unexpected root: %s", catalog.PluginsRoot)
	}
	if len(catalog.Items) != 1 {
		t.Fatalf("expected one item, got %+v", catalog.Items)
	}
	item := catalog.Items[0]
	if item.ID != "demo" || item.Version != "0.2.0" || !item.Installable {
		t.Fatalf("unexpected item identity: %+v", item)
	}
	if item.PackageDigest == "" || item.PackageSizeBytes == 0 {
		t.Fatalf("expected package digest and size: %+v", item)
	}
	if item.Signature.Status != "signed" || item.Signature.Algorithm != "RSA-SHA256" {
		t.Fatalf("expected signature summary, got %+v", item.Signature)
	}
	if item.Risk.Level != "critical" {
		t.Fatalf("expected critical risk from destructive data uninstall policy, got %+v", item.Risk)
	}
	if len(item.Risk.Data) == 0 {
		t.Fatalf("expected data risk summary, got %+v", item.Risk)
	}
	if len(item.Risk.APIs) == 0 || len(item.Risk.Audit) == 0 {
		t.Fatalf("expected api and audit risk summary, got %+v", item.Risk)
	}
}

func TestLocalMarketplaceServiceListsPackageWithoutManifest(t *testing.T) {
	root := t.TempDir()
	dist := filepath.Join(root, "_dist")
	if err := os.MkdirAll(dist, 0o755); err != nil {
		t.Fatalf("mkdir dist: %v", err)
	}
	writeZip(t, filepath.Join(dist, "orphan-1.0.0.zip"))

	catalog, err := NewLocalMarketplaceService(nil).List(root)
	if err != nil {
		t.Fatalf("list local marketplace: %v", err)
	}
	if len(catalog.Items) != 1 {
		t.Fatalf("expected orphan package item, got %+v", catalog.Items)
	}
	item := catalog.Items[0]
	if item.ID != "orphan" || item.Version != "1.0.0" || item.Signature.Status != "unknown" || !item.Installable {
		t.Fatalf("unexpected orphan package item: %+v", item)
	}
}

func writeZip(t *testing.T, path string) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("create zip: %v", err)
	}
	defer file.Close()
	writer := zip.NewWriter(file)
	defer writer.Close()
	entry, err := writer.Create("plugin.yaml")
	if err != nil {
		t.Fatalf("create zip entry: %v", err)
	}
	if _, err := entry.Write([]byte("id: demo\n")); err != nil {
		t.Fatalf("write zip entry: %v", err)
	}
}
