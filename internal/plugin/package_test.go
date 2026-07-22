package plugin

import (
	"archive/zip"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestBuildVerifyAndInstallPackage(t *testing.T) {
	source := writePackageTestPlugin(t)
	dist := filepath.Join(t.TempDir(), "dist")
	loader := NewFileLoader()

	first, err := BuildPackage(source, dist, loader)
	if err != nil {
		t.Fatalf("BuildPackage(first) error = %v", err)
	}
	second, err := BuildPackage(source, dist, loader)
	if err != nil {
		t.Fatalf("BuildPackage(second) error = %v", err)
	}
	if first.SHA256 != second.SHA256 {
		t.Fatalf("package is not deterministic: first=%s second=%s", first.SHA256, second.SHA256)
	}
	verified, err := VerifyPackage(second.ArtifactPath, second.ChecksumPath)
	if err != nil || verified != second.SHA256 {
		t.Fatalf("VerifyPackage() = %q, %v", verified, err)
	}
	assertPackageEntries(t, second.ArtifactPath, []string{"frontend/dist/app.js", "plugin.yaml"}, []string{"frontend/node_modules/ignored.js", ".git/config"})

	pluginsRoot := filepath.Join(t.TempDir(), "plugins")
	installedDir, info, err := InstallPackage(second.ArtifactPath, second.ChecksumPath, pluginsRoot, loader)
	if err != nil {
		t.Fatalf("InstallPackage() error = %v", err)
	}
	if info.ID != "package-demo" || info.Version != "1.2.3" || installedDir != filepath.Join(pluginsRoot, "package-demo") {
		t.Fatalf("unexpected install result: dir=%s info=%+v", installedDir, info)
	}
	manager := NewRuntimeManager(loader, NewTopologicalResolver())
	runtimeInfo, err := manager.Install(installedDir)
	if err != nil {
		t.Fatalf("runtime Install() error = %v", err)
	}
	if runtimeInfo.ID != info.ID || runtimeInfo.Version != info.Version {
		t.Fatalf("runtime contract differs: extracted=%+v runtime=%+v", info, runtimeInfo)
	}
}

func TestVerifyPackageRejectsTampering(t *testing.T) {
	result, err := BuildPackage(writePackageTestPlugin(t), t.TempDir(), NewFileLoader())
	if err != nil {
		t.Fatalf("BuildPackage() error = %v", err)
	}
	file, err := os.OpenFile(result.ArtifactPath, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatalf("open artifact: %v", err)
	}
	if _, err := file.WriteString("tampered"); err != nil {
		t.Fatalf("tamper artifact: %v", err)
	}
	_ = file.Close()
	if _, err := VerifyPackage(result.ArtifactPath, result.ChecksumPath); err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("VerifyPackage() error = %v, want checksum mismatch", err)
	}
}

func TestBuildPackageRequiresManagedBackendAndPreservesExecutableMode(t *testing.T) {
	source := writePackageTestPlugin(t)
	manifest := "id: package-demo\nname: Package Demo\nversion: 1.2.3\nui_mode: separated\nservice_base_url: http://127.0.0.1:19090\nservice_health_url: http://127.0.0.1:19090/health\n"
	if err := os.WriteFile(filepath.Join(source, "plugin.yaml"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := BuildPackage(source, t.TempDir(), NewFileLoader()); err == nil || !strings.Contains(err.Error(), "backend entry") {
		t.Fatalf("missing managed backend error = %v", err)
	}

	entry := filepath.Join(source, filepath.FromSlash(managedBackendRelativePath("package-demo")))
	if err := os.MkdirAll(filepath.Dir(entry), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(entry, []byte("managed backend"), 0o755); err != nil {
		t.Fatal(err)
	}
	dist := t.TempDir()
	result, err := BuildPackage(source, dist, NewFileLoader())
	if err != nil {
		t.Fatal(err)
	}
	reader, err := zip.OpenReader(result.ArtifactPath)
	if err != nil {
		t.Fatal(err)
	}
	wantEntry := managedBackendRelativePath("package-demo")
	archiveExecutable := false
	for _, entry := range reader.File {
		if entry.Name == wantEntry {
			archiveExecutable = entry.Mode().Perm()&0o111 != 0
		}
	}
	_ = reader.Close()
	if !archiveExecutable {
		t.Fatalf("managed backend %s is not executable in package", wantEntry)
	}
	installed, _, err := InstallPackage(result.ArtifactPath, result.ChecksumPath, t.TempDir(), NewFileLoader())
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(installed, filepath.FromSlash(managedBackendRelativePath("package-demo"))))
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o111 == 0 {
		t.Fatalf("installed managed backend mode = %v", info.Mode())
	}
}

func TestInstallPackageRejectsTraversal(t *testing.T) {
	dir := t.TempDir()
	artifact := filepath.Join(dir, "bad.zip")
	file, err := os.Create(artifact)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	entry, err := writer.Create("../outside.txt")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = entry.Write([]byte("outside"))
	_ = writer.Close()
	_ = file.Close()
	digest, err := filePackageSHA256(artifact)
	if err != nil {
		t.Fatal(err)
	}
	checksum := artifact + packageChecksumSuffix
	if err := os.WriteFile(checksum, []byte(digest+"  "+filepath.Base(artifact)+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := InstallPackage(artifact, checksum, filepath.Join(dir, "plugins"), NewFileLoader()); err == nil || !strings.Contains(err.Error(), "invalid plugin package entry") {
		t.Fatalf("InstallPackage() error = %v, want invalid entry", err)
	}
}

func writePackageTestPlugin(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "package-demo")
	files := map[string]string{
		"plugin.yaml":                      "id: package-demo\nname: Package Demo\nversion: 1.2.3\nui_mode: separated\n",
		"backend/main.go":                  "package main\n",
		"frontend/dist/app.js":             "export default {};\n",
		"frontend/node_modules/ignored.js": "ignored\n",
		".git/config":                      "ignored\n",
	}
	for name, content := range files {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func assertPackageEntries(t *testing.T, artifact string, included, excluded []string) {
	t.Helper()
	reader, err := zip.OpenReader(artifact)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	entries := make(map[string]bool, len(reader.File))
	for _, entry := range reader.File {
		entries[entry.Name] = true
	}
	for _, name := range included {
		if !entries[name] {
			t.Errorf("package missing %s", name)
		}
	}
	for _, name := range excluded {
		if entries[name] {
			t.Errorf("package unexpectedly contains %s", name)
		}
	}
}
