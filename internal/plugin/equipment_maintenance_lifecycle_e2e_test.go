package plugin

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tinboxw/skoll/pkg/pluginsdk"
	"github.com/tinboxw/skoll/pkg/security"
)

func TestEquipmentMaintenancePackagedLifecycleE2E(t *testing.T) {
	if testing.Short() || os.Getenv("SKOLL_EQUIPMENT_PLUGIN_E2E") != "1" {
		t.Skip("set SKOLL_EQUIPMENT_PLUGIN_E2E=1 to run the packaged equipment-maintenance lifecycle E2E")
	}
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	webModules := filepath.Join(repoRoot, "web", "node_modules")
	if info, statErr := os.Stat(webModules); statErr != nil || !info.IsDir() {
		t.Skip("web/node_modules is required for equipment-maintenance lifecycle E2E")
	}

	address := reserveEquipmentAddress(t)
	source := filepath.Join(t.TempDir(), "equipment_maintenance")
	copyEquipmentSource(t, filepath.Join(repoRoot, "plugins", "equipment_maintenance"), source)
	rewriteEquipmentAddress(t, filepath.Join(source, "plugin.yaml"), address)
	linkEquipmentNodeModules(t, webModules, filepath.Join(source, "web", "node_modules"))
	before := equipmentSourceHashes(t, source)
	buildEquipmentPackageSurface(t, repoRoot, source)
	after := equipmentSourceHashes(t, source)
	if strings.Join(before, "\n") != strings.Join(after, "\n") {
		t.Fatalf("build changed plugin source\nbefore=%v\nafter=%v", before, after)
	}

	result, err := BuildPackage(source, filepath.Join(t.TempDir(), "dist"), NewFileLoader())
	if err != nil {
		t.Fatalf("build package: %v", err)
	}
	if verified, verifyErr := VerifyPackage(result.ArtifactPath, result.ChecksumPath); verifyErr != nil || verified != result.SHA256 {
		t.Fatalf("verify package digest=%q want=%q err=%v", verified, result.SHA256, verifyErr)
	}
	installedDir, packageInfo, err := InstallPackage(result.ArtifactPath, result.ChecksumPath, filepath.Join(t.TempDir(), "plugins"), NewFileLoader())
	if err != nil {
		t.Fatalf("install package: %v", err)
	}
	for _, required := range []string{"plugin.yaml", "web/dist/index.html", managedBackendRelativePath("equipment_maintenance"), "migrations/001_initial.up.sql", "migrations/001_initial.down.sql"} {
		if _, statErr := os.Stat(filepath.Join(installedDir, filepath.FromSlash(required))); statErr != nil {
			t.Fatalf("installed package missing %s: %v", required, statErr)
		}
	}

	const jwtSecret = "equipment-maintenance-lifecycle-secret"
	transactions := &gatewayTransactions{}
	audit := &equipmentGatewayAudit{}
	host := pluginsdk.HostServices{
		PluginID: "equipment_maintenance", Transactions: transactions, DataScopes: gatewayScopes{}, Files: gatewayFiles{},
		DataStore: gatewayDataStore{}, Audit: audit, Config: gatewayConfig{}, Secrets: &gatewaySecrets{}, Workflows: gatewayWorkflows{}, Jobs: gatewayJobs{},
	}
	gateway, err := NewHostGateway(func(pluginID string) (pluginsdk.HostServices, error) {
		if pluginID != host.PluginID {
			return pluginsdk.HostServices{}, errors.New("unexpected plugin identity")
		}
		return host, nil
	}, jwtSecret, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = gateway.Close() })

	dataRoot := filepath.Join(t.TempDir(), "data")
	dataDirectories, err := NewPluginDataDirectories(dataRoot)
	if err != nil {
		t.Fatal(err)
	}
	manager := NewRuntimeManager(NewFileLoader(), NewTopologicalResolver())
	installed, err := manager.Install(installedDir)
	if err != nil {
		t.Fatalf("runtime install: %v", err)
	}
	if installed.ID != packageInfo.ID || installed.State != StateInstalled {
		t.Fatalf("unexpected installed state: %+v package=%+v", installed, packageInfo)
	}
	migrationStore := newTestMigrationStore()
	migrationHook := NewPluginMigrationHook(migrationStore, nil)
	if _, err := migrationHook.Run(context.Background(), PluginMigrationHookInput{
		PluginID: installed.ID, PluginDir: installedDir, MigrationDirectory: installed.DataManifest.MigrationDirectory,
		Action: PluginMigrationInstall, ToVersion: installed.DataManifest.MigrationVersion,
	}); err != nil {
		t.Fatalf("install migration: %v", err)
	}
	if len(migrationStore.records) != 1 {
		t.Fatalf("install migration ledger=%+v", migrationStore.records)
	}

	launcher := NewManagedProcessLauncher(NewHTTPHealthChecker(time.Second), 50*time.Millisecond, gateway, dataDirectories)
	supervisor := NewServiceSupervisor(launcher, nil, 5*time.Second, 2*time.Second)
	t.Cleanup(func() { _ = supervisor.Shutdown(context.Background()) })
	if err := manager.Enable(installed.ID); err != nil {
		t.Fatalf("enable plugin: %v", err)
	}
	enabled := mustEquipmentInfo(t, manager)
	if err := supervisor.Start(context.Background(), enabled); err != nil {
		t.Fatalf("start packaged backend: %v", err)
	}

	userToken, err := security.SignJWT(jwtSecret, security.JWTIdentity{Subject: "engineer-1", Role: "operator", Roles: []string{"operator"}}, time.Minute, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	baseURL := "http://" + address + "/v1/plugins/equipment_maintenance/api"
	created := equipmentRequest(t, http.MethodPost, baseURL+"/assets", userToken, map[string]any{
		"tenantId": "tenant-1", "organizationId": "plant-1", "code": "EQ-LIFE-001", "name": "Filling line",
	})
	if created.Status != http.StatusCreated || !bytes.Contains(created.Body, []byte("EQ-LIFE-001")) {
		t.Fatalf("create asset status=%d body=%s", created.Status, created.Body)
	}
	job := equipmentRequest(t, http.MethodPost, baseURL+"/jobs/maintenance-due-scan", userToken, map[string]any{
		"tenantId": "tenant-1", "organizationId": "plant-1",
	})
	if job.Status != http.StatusAccepted || !bytes.Contains(job.Body, []byte("job-1")) {
		t.Fatalf("schedule job status=%d body=%s", job.Status, job.Body)
	}
	if transactions.commits < 1 || !audit.hasAction("equipment_maintenance.asset.create") || !audit.hasAction("equipment_maintenance.job.maintenance_scan") {
		t.Fatalf("host services were not exercised: transactions=%+v audit=%+v", transactions, audit.actions())
	}

	dataDir, err := dataDirectories.Prepare(installed.ID)
	if err != nil {
		t.Fatal(err)
	}
	statePath := filepath.Join(dataDir, "equipment-maintenance.json")
	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("plugin state was not persisted: %v", err)
	}
	if err := manager.Disable(installed.ID); err != nil {
		t.Fatalf("disable plugin: %v", err)
	}
	if err := supervisor.Stop(context.Background(), installed.ID); err != nil {
		t.Fatalf("stop plugin: %v", err)
	}
	if snapshot, ok := supervisor.Snapshot(installed.ID); !ok || snapshot.State != ServiceStateStopped {
		t.Fatalf("disabled service snapshot=%+v exists=%v", snapshot, ok)
	}
	if err := equipmentEndpointClosed(baseURL + "/assets"); err != nil {
		t.Fatal(err)
	}

	if err := manager.Enable(installed.ID); err != nil {
		t.Fatalf("re-enable plugin: %v", err)
	}
	if err := supervisor.Start(context.Background(), mustEquipmentInfo(t, manager)); err != nil {
		t.Fatalf("restart packaged backend: %v", err)
	}
	listed := equipmentRequest(t, http.MethodGet, baseURL+"/assets?limit=20", userToken, nil)
	if listed.Status != http.StatusOK || !bytes.Contains(listed.Body, []byte("EQ-LIFE-001")) {
		t.Fatalf("restart did not retain data: status=%d body=%s", listed.Status, listed.Body)
	}

	if err := manager.Disable(installed.ID); err != nil {
		t.Fatalf("disable before uninstall: %v", err)
	}
	if err := supervisor.Stop(context.Background(), installed.ID); err != nil {
		t.Fatalf("stop before uninstall: %v", err)
	}
	if _, err := migrationHook.Run(context.Background(), PluginMigrationHookInput{
		PluginID: installed.ID, PluginDir: installedDir, MigrationDirectory: installed.DataManifest.MigrationDirectory,
		Action: PluginMigrationUninstall, FromVersion: installed.DataManifest.MigrationVersion, UninstallPolicy: installed.DataManifest.UninstallPolicy,
	}); err != nil {
		t.Fatalf("uninstall migration: %v", err)
	}
	if err := dataDirectories.Uninstall(installed.ID, installed.DataManifest.UninstallPolicy); err != nil {
		t.Fatalf("drop plugin data: %v", err)
	}
	if err := manager.Uninstall(installed.ID); err != nil {
		t.Fatalf("uninstall plugin: %v", err)
	}
	final, err := manager.Get(installed.ID)
	if err != nil || final.State != StateUninstalled || len(migrationStore.records) != 0 {
		t.Fatalf("final state=%+v ledger=%+v err=%v", final, migrationStore.records, err)
	}
	if _, err := os.Lstat(dataDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("drop uninstall retained plugin data: %v", err)
	}
	if err := equipmentEndpointClosed(baseURL + "/assets"); err != nil {
		t.Fatal(err)
	}
}

type equipmentGatewayAudit struct {
	mu      sync.Mutex
	entries []pluginsdk.AuditEntry
}

func (a *equipmentGatewayAudit) Record(_ context.Context, entry pluginsdk.AuditEntry) (pluginsdk.AuditReceipt, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.entries = append(a.entries, entry)
	return pluginsdk.AuditReceipt{ID: fmt.Sprintf("audit-%d", len(a.entries))}, nil
}

func (a *equipmentGatewayAudit) hasAction(action string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, entry := range a.entries {
		if entry.Action == action {
			return true
		}
	}
	return false
}

func (a *equipmentGatewayAudit) actions() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]string, 0, len(a.entries))
	for _, entry := range a.entries {
		out = append(out, entry.Action)
	}
	return out
}

type equipmentResponse struct {
	Status int
	Body   []byte
}

func equipmentRequest(t *testing.T, method, target, token string, payload map[string]any) equipmentResponse {
	t.Helper()
	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
		body = bytes.NewReader(raw)
	}
	request, err := http.NewRequest(method, target, body)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	response, err := (&http.Client{Timeout: 3 * time.Second}).Do(request)
	if err != nil {
		t.Fatalf("request %s %s: %v", method, target, err)
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return equipmentResponse{Status: response.StatusCode, Body: raw}
}

func equipmentEndpointClosed(target string) error {
	request, _ := http.NewRequest(http.MethodGet, target, nil)
	response, err := (&http.Client{Timeout: 500 * time.Millisecond}).Do(request)
	if err != nil {
		return nil
	}
	defer response.Body.Close()
	return fmt.Errorf("disabled equipment endpoint remained reachable with status %d", response.StatusCode)
}

func mustEquipmentInfo(t *testing.T, manager *RuntimeManager) Info {
	t.Helper()
	info, err := manager.Get("equipment_maintenance")
	if err != nil {
		t.Fatal(err)
	}
	return info
}

func reserveEquipmentAddress(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	_ = listener.Close()
	return address
}

func rewriteEquipmentAddress(t *testing.T, manifestPath, address string) {
	t.Helper()
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	updated := strings.ReplaceAll(string(raw), "127.0.0.1:18092", address)
	if updated == string(raw) {
		t.Fatal("equipment manifest service address was not rewritten")
	}
	if err := os.WriteFile(manifestPath, []byte(updated), 0o600); err != nil {
		t.Fatal(err)
	}
}

func buildEquipmentPackageSurface(t *testing.T, repoRoot, source string) {
	t.Helper()
	binary := filepath.Join(source, filepath.FromSlash(managedBackendRelativePath("equipment_maintenance")))
	if err := os.MkdirAll(filepath.Dir(binary), 0o755); err != nil {
		t.Fatal(err)
	}
	goBuild := exec.Command("go", "build", "-o", binary, "./plugins/equipment_maintenance/backend")
	goBuild.Dir = repoRoot
	if output, err := goBuild.CombinedOutput(); err != nil {
		t.Fatalf("build equipment backend: %v\n%s", err, output)
	}
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skipf("node is required: %v", err)
	}
	webRoot := filepath.Join(source, "web")
	vueTSC := exec.Command(node, filepath.Join(repoRoot, "web", "node_modules", "vue-tsc", "index.js"), "--noEmit", "-p", filepath.Join(webRoot, "tsconfig.json"))
	vueTSC.Dir = webRoot
	if output, err := vueTSC.CombinedOutput(); err != nil {
		t.Fatalf("typecheck equipment frontend: %v\n%s", err, output)
	}
	vite := exec.Command(node, filepath.Join(repoRoot, "web", "node_modules", "vite", "bin", "vite.js"), "build", "--config", filepath.Join(webRoot, "vite.config.ts"))
	vite.Dir = webRoot
	if output, err := vite.CombinedOutput(); err != nil {
		t.Fatalf("build equipment frontend: %v\n%s", err, output)
	}
}

func copyEquipmentSource(t *testing.T, source, destination string) {
	t.Helper()
	err := filepath.WalkDir(source, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		name := entry.Name()
		if entry.IsDir() && (name == "node_modules" || name == "dist" || name == "bin" || name == ".skoll-dev") {
			return filepath.SkipDir
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, raw, info.Mode().Perm())
	})
	if err != nil {
		t.Fatalf("copy equipment plugin source: %v", err)
	}
}

func linkEquipmentNodeModules(t *testing.T, source, target string) {
	t.Helper()
	if err := os.Symlink(source, target); err == nil {
		return
	} else if runtime.GOOS != "windows" {
		t.Fatalf("link frontend dependencies: %v", err)
	}
	command := exec.Command("cmd", "/c", "mklink", "/J", target, source)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("link frontend dependencies: %v\n%s", err, output)
	}
}

func equipmentSourceHashes(t *testing.T, root string) []string {
	t.Helper()
	items := make([]string, 0)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		name := entry.Name()
		if name == "node_modules" {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() && (name == "node_modules" || name == "dist" || name == "bin") {
			return filepath.SkipDir
		}
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		relative, _ := filepath.Rel(root, path)
		sum := sha256.Sum256(raw)
		items = append(items, filepath.ToSlash(relative)+"="+hex.EncodeToString(sum[:]))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(items)
	return items
}
