package generator

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	domaingenerator "github.com/tinboxw/skoll/internal/domain/generator"
	pluginruntime "github.com/tinboxw/skoll/internal/plugin"
)

func TestGeneratedPluginBuildPackageAndInstallWithoutSourceEdits(t *testing.T) {
	if testing.Short() || os.Getenv("SKOLL_GENERATOR_PLUGIN_E2E") != "1" {
		t.Skip("set SKOLL_GENERATOR_PLUGIN_E2E=1 to run the standalone generated-plugin E2E")
	}
	repoRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	webModules := filepath.Join(repoRoot, "web", "node_modules")
	if stat, err := os.Stat(webModules); err != nil || !stat.IsDir() {
		t.Skip("web/node_modules is required for generated plugin build E2E")
	}
	address := reserveGeneratedServiceAddress(t)
	baseURL := "http://" + address
	spec := generatedPluginE2ESpec(t, baseURL)
	result, err := NewService().DryRun(context.Background(), DryRunInput{
		Spec: spec, BatchID: "pr3-generated-plugin-e2e", ActorID: "generator-test", MigrationTimestamp: "20260722_010203",
	})
	if err != nil {
		t.Fatalf("DryRun() error = %v", err)
	}
	pluginDir := materializeGeneratedPlugin(t, result, "pharma-oa")
	linkNodeModules(t, webModules, filepath.Join(pluginDir, "web", "node_modules"))
	before := generatedSourceHashes(t, pluginDir)
	runGeneratedGoTests(t, pluginDir)

	dist := filepath.Join(pluginDir, "dist")
	command := generatedPackageCommand(t, pluginDir, dist)
	command.Env = append(command.Environ(), "SKOLL_REPO_ROOT="+repoRoot)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("generated package command failed: %v\n%s", err, output)
	}
	t.Logf("generated package output:\n%s", output)
	after := generatedSourceHashes(t, pluginDir)
	if strings.Join(before, "\n") != strings.Join(after, "\n") {
		t.Fatalf("generated source changed during build\nbefore=%v\nafter=%v", before, after)
	}

	artifact := filepath.Join(dist, "pharma-oa-1.0.0.zip")
	checksum := artifact + ".sha256"
	if _, err := os.Stat(checksum); err != nil {
		t.Fatalf("generated checksum not found at %s: %v; package files=%v\n%s", checksum, err, findGeneratedPackageFiles(pluginDir), output)
	}
	if _, err := pluginruntime.VerifyPackage(artifact, checksum); err != nil {
		t.Fatalf("verify generated package: %v", err)
	}
	installedDir, info, err := pluginruntime.InstallPackage(artifact, checksum, filepath.Join(t.TempDir(), "plugins"), pluginruntime.NewFileLoader())
	if err != nil {
		t.Fatalf("install generated package: %v", err)
	}
	if info.ID != "pharma-oa" {
		t.Fatalf("installed plugin = %+v", info)
	}
	for _, required := range []string{"plugin.yaml", "web/dist/index.html", generatedBackendBinaryPath("pharma-oa")} {
		if _, err := os.Stat(filepath.Join(installedDir, filepath.FromSlash(required))); err != nil {
			t.Errorf("installed package missing %s: %v", required, err)
		}
	}
	runGeneratedPluginLifecycle(t, installedDir, spec)
}

func generatedPluginE2ESpec(t *testing.T, baseURL string) *domaingenerator.GeneratorSpec {
	t.Helper()
	in := validServiceGeneratorSpecInput()
	in.Plugin = domaingenerator.PluginSpec{
		Enabled: true, ID: "pharma-oa", Name: "Pharma OA", Description: "Generated pharma OA business plugin",
		DataNamespace: "pharma-oa", UninstallPolicy: "drop", RollbackPolicy: "automatic",
		ServiceBaseURL: baseURL, ServiceHealthURL: baseURL + "/health",
	}
	in.Table.Name = "pharma_oa_products"
	in.Indexes[0].Name = "idx_pharma_oa_products_name"
	namespaceServicePluginInput(&in, "pharma_oa")
	spec, err := domaingenerator.NewGeneratorSpec(in)
	if err != nil {
		t.Fatalf("NewGeneratorSpec() error = %v", err)
	}
	return spec
}

func reserveGeneratedServiceAddress(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	_ = listener.Close()
	return address
}

func runGeneratedGoTests(t *testing.T, pluginDir string) {
	t.Helper()
	cmd := exec.Command("go", "test", "./...", "-count=1")
	cmd.Dir = pluginDir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("generated plugin tests failed: %v\n%s", err, output)
	}
}

func runGeneratedPluginLifecycle(t *testing.T, pluginDir string, spec *domaingenerator.GeneratorSpec) {
	t.Helper()
	loader := pluginruntime.NewFileLoader()
	manager := pluginruntime.NewRuntimeManager(loader, pluginruntime.NewTopologicalResolver())
	info, err := manager.Install(pluginDir)
	if err != nil {
		t.Fatalf("runtime install generated plugin: %v", err)
	}
	migrationStore := newGeneratedMigrationStore()
	migrationHook := pluginruntime.NewPluginMigrationHook(migrationStore, nil)
	if _, err := migrationHook.Run(context.Background(), pluginruntime.PluginMigrationHookInput{
		PluginID: info.ID, PluginDir: pluginDir, MigrationDirectory: info.DataManifest.MigrationDirectory,
		Action: pluginruntime.PluginMigrationInstall, ToVersion: info.DataManifest.MigrationVersion,
	}); err != nil {
		t.Fatalf("install generated migration: %v", err)
	}
	if err := manager.Enable(info.ID); err != nil {
		t.Fatalf("enable generated plugin: %v", err)
	}
	dataDirectories, err := pluginruntime.NewPluginDataDirectories(filepath.Join(filepath.Dir(pluginDir), "plugin-data"))
	if err != nil {
		t.Fatal(err)
	}
	supervisor := pluginruntime.NewServiceSupervisor(pluginruntime.NewManagedProcessLauncher(pluginruntime.NewHTTPHealthChecker(time.Second), 100*time.Millisecond, generatedCredentialIssuer{}, dataDirectories), nil, 3*time.Second, time.Second)
	if err := supervisor.Start(context.Background(), mustGeneratedPluginInfo(t, manager, info.ID)); err != nil {
		t.Fatalf("start and supervise generated backend: %v", err)
	}

	create := generatedRuntimeRequest(t, manager, info.ID, http.MethodPost, spec.Plugin.ServiceBaseURL+pluginAPIBasePath(*spec), []byte(`{"name":"Aspirin"}`))
	if create.StatusCode != http.StatusCreated {
		t.Fatalf("generated create status=%d body=%s", create.StatusCode, create.Body)
	}
	var created struct {
		Data struct {
			Item map[string]any `json:"item"`
		} `json:"data"`
	}
	if err := json.Unmarshal(create.Body, &created); err != nil || strings.TrimSpace(fmt.Sprint(created.Data.Item["id"])) == "" {
		t.Fatalf("generated create response=%s error=%v", create.Body, err)
	}
	id := fmt.Sprint(created.Data.Item["id"])
	list := generatedRuntimeRequest(t, manager, info.ID, http.MethodGet, spec.Plugin.ServiceBaseURL+pluginAPIBasePath(*spec), nil)
	if list.StatusCode != http.StatusOK || !bytes.Contains(list.Body, []byte("Aspirin")) {
		t.Fatalf("generated list status=%d body=%s", list.StatusCode, list.Body)
	}
	detail := generatedRuntimeRequest(t, manager, info.ID, http.MethodGet, spec.Plugin.ServiceBaseURL+pluginAPIBasePath(*spec)+"/"+id, nil)
	if detail.StatusCode != http.StatusOK || !bytes.Contains(detail.Body, []byte(id)) {
		t.Fatalf("generated detail status=%d body=%s", detail.StatusCode, detail.Body)
	}

	if err := manager.Disable(info.ID); err != nil {
		t.Fatalf("disable generated plugin: %v", err)
	}
	if err := supervisor.Stop(context.Background(), info.ID); err != nil {
		t.Fatalf("stop generated service supervision: %v", err)
	}
	if response := generatedRuntimeRequest(t, manager, info.ID, http.MethodGet, spec.Plugin.ServiceBaseURL+pluginAPIBasePath(*spec), nil); response.StatusCode != http.StatusForbidden {
		t.Fatalf("disabled generated plugin remained executable: status=%d body=%s", response.StatusCode, response.Body)
	}
	if _, err := migrationHook.Run(context.Background(), pluginruntime.PluginMigrationHookInput{
		PluginID: info.ID, PluginDir: pluginDir, MigrationDirectory: info.DataManifest.MigrationDirectory,
		Action: pluginruntime.PluginMigrationUninstall, FromVersion: info.DataManifest.MigrationVersion, UninstallPolicy: info.DataManifest.UninstallPolicy,
	}); err != nil {
		t.Fatalf("uninstall generated migration: %v", err)
	}
	if err := manager.Uninstall(info.ID); err != nil {
		t.Fatalf("uninstall generated plugin: %v", err)
	}
	final, err := manager.Get(info.ID)
	if err != nil || final.State != pluginruntime.StateUninstalled || len(migrationStore.records) != 0 {
		t.Fatalf("generated uninstall final=%+v migrations=%+v error=%v", final, migrationStore.records, err)
	}
}

type generatedCredentialIssuer struct{}

func (generatedCredentialIssuer) Issue(string) (pluginruntime.ProcessCredential, error) {
	return pluginruntime.ProcessCredential{HostURL: "http://127.0.0.1:1", Token: "generated-e2e-token"}, nil
}

func (generatedCredentialIssuer) Revoke(string) {}

type generatedHTTPResponse struct {
	StatusCode int
	Body       []byte
}

func generatedRuntimeRequest(t *testing.T, manager *pluginruntime.RuntimeManager, pluginID, method, target string, body []byte) generatedHTTPResponse {
	t.Helper()
	info := mustGeneratedPluginInfo(t, manager, pluginID)
	if info.State != pluginruntime.StateEnabled {
		return generatedHTTPResponse{StatusCode: http.StatusForbidden, Body: []byte("plugin_disabled")}
	}
	request, err := http.NewRequest(method, target, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("execute generated plugin request: %v", err)
	}
	defer response.Body.Close()
	var buffer bytes.Buffer
	_, _ = buffer.ReadFrom(response.Body)
	return generatedHTTPResponse{StatusCode: response.StatusCode, Body: buffer.Bytes()}
}

func mustGeneratedPluginInfo(t *testing.T, manager *pluginruntime.RuntimeManager, pluginID string) pluginruntime.Info {
	t.Helper()
	info, err := manager.Get(pluginID)
	if err != nil {
		t.Fatalf("get generated plugin: %v", err)
	}
	return info
}

func findGeneratedPackageFiles(root string) []string {
	files := make([]string, 0)
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry == nil {
			return nil
		}
		if !entry.IsDir() && (strings.HasSuffix(entry.Name(), ".zip") || strings.HasSuffix(entry.Name(), ".sha256")) {
			files = append(files, path)
		}
		return nil
	})
	return files
}

func generatedPackageCommand(t *testing.T, pluginDir, distDir string) *exec.Cmd {
	t.Helper()
	if runtime.GOOS == "windows" {
		powershell, err := exec.LookPath("powershell")
		if err != nil {
			t.Skipf("PowerShell is required: %v", err)
		}
		cmd := exec.Command(powershell, "-NoProfile", "-File", filepath.Join(pluginDir, "plugin.ps1"), "-Action", "package", "-DistDir", distDir)
		cmd.Dir = pluginDir
		return cmd
	}
	cmd := exec.Command("sh", filepath.Join(pluginDir, "plugin.sh"), "package")
	cmd.Dir = pluginDir
	cmd.Env = append(os.Environ(), "SKOLL_PLUGIN_DIST="+distDir)
	return cmd
}

func linkNodeModules(t *testing.T, source, target string) {
	t.Helper()
	linkErr := os.Symlink(source, target)
	if linkErr == nil {
		return
	}
	if runtime.GOOS != "windows" {
		t.Fatalf("link frontend dependencies: %v", linkErr)
	}
	cmd := exec.Command("cmd", "/c", "mklink", "/J", target, source)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("link frontend dependencies: %v\n%s", err, output)
	}
}

func generatedSourceHashes(t *testing.T, root string) []string {
	t.Helper()
	items := make([]string, 0)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		name := entry.Name()
		if name == "node_modules" || entry.Type()&os.ModeSymlink != 0 {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			if name == "node_modules" || name == "dist" || name == "bin" {
				return filepath.SkipDir
			}
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		sum := sha256.Sum256(raw)
		items = append(items, filepath.ToSlash(rel)+"="+hex.EncodeToString(sum[:]))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return items
}

func generatedBackendBinaryPath(pluginID string) string {
	name := pluginID + "-server"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.ToSlash(filepath.Join("backend", "bin", name))
}
