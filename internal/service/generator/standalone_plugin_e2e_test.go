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
	"sync"
	"testing"
	"time"

	domaingenerator "github.com/tinboxw/skoll/internal/domain/generator"
	pluginruntime "github.com/tinboxw/skoll/internal/plugin"
	"github.com/tinboxw/skoll/pkg/pluginsdk"
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
	linkDocumentUIPackage(t, repoRoot, pluginDir)
	linkNodeModules(t, webModules, filepath.Join(pluginDir, "web", "node_modules"))
	goWork := generatedGoWorkspace(t, repoRoot, pluginDir)
	before := generatedSourceHashes(t, pluginDir)
	runGeneratedGoTests(t, pluginDir, goWork)

	dist := filepath.Join(pluginDir, "dist")
	command := generatedPackageCommand(t, pluginDir, dist)
	command.Env = append(command.Environ(), "SKOLL_REPO_ROOT="+repoRoot, "GOWORK="+goWork)
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
	in.Document = &domaingenerator.DocumentSpec{
		Enabled: true, SchemaKey: "product_request", SchemaName: "Product Request",
		DefinitionID: "product-request-approval", NumberPrefix: "PR", TitleField: "name",
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

func runGeneratedGoTests(t *testing.T, pluginDir, goWork string) {
	t.Helper()
	cmd := exec.Command("go", "test", "./...", "-count=1")
	cmd.Dir = pluginDir
	cmd.Env = append(cmd.Environ(), "GOWORK="+goWork)
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
	documents := newGeneratedDocumentHost()
	gateway, err := pluginruntime.NewHostGateway(func(pluginID string) (pluginsdk.HostServices, error) {
		return generatedHostServices(pluginID, documents), nil
	}, "generated-plugin-e2e-secret", time.Minute)
	if err != nil {
		t.Fatalf("start generated host gateway: %v", err)
	}
	defer gateway.Close()
	supervisor := pluginruntime.NewServiceSupervisor(pluginruntime.NewManagedProcessLauncher(pluginruntime.NewHTTPHealthChecker(time.Second), 100*time.Millisecond, gateway, dataDirectories), nil, 3*time.Second, time.Second)
	if err := supervisor.Start(context.Background(), mustGeneratedPluginInfo(t, manager, info.ID)); err != nil {
		t.Fatalf("start and supervise generated backend: %v", err)
	}

	create := generatedRuntimeRequest(t, manager, info.ID, http.MethodPost, spec.Plugin.ServiceBaseURL+pluginAPIBasePath(*spec), []byte(`{"name":"Aspirin"}`))
	if create.StatusCode != http.StatusCreated {
		t.Fatalf("generated create status=%d body=%s", create.StatusCode, create.Body)
	}
	var created struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(create.Body, &created); err != nil || strings.TrimSpace(fmt.Sprint(created.Data["id"])) == "" {
		t.Fatalf("generated create response=%s error=%v", create.Body, err)
	}
	id := fmt.Sprint(created.Data["id"])
	submit := generatedRuntimeRequest(t, manager, info.ID, http.MethodPost, spec.Plugin.ServiceBaseURL+pluginAPIBasePath(*spec)+"/"+id+"/submit", nil)
	if submit.StatusCode != http.StatusOK || !bytes.Contains(submit.Body, []byte(`"state":"submitted"`)) {
		t.Fatalf("generated submit status=%d body=%s", submit.StatusCode, submit.Body)
	}
	list := generatedRuntimeRequest(t, manager, info.ID, http.MethodGet, spec.Plugin.ServiceBaseURL+pluginAPIBasePath(*spec), nil)
	if list.StatusCode != http.StatusOK || !bytes.Contains(list.Body, []byte("Aspirin")) {
		t.Fatalf("generated list status=%d body=%s", list.StatusCode, list.Body)
	}
	detail := generatedRuntimeRequest(t, manager, info.ID, http.MethodGet, spec.Plugin.ServiceBaseURL+pluginAPIBasePath(*spec)+"/"+id, nil)
	if detail.StatusCode != http.StatusOK || !bytes.Contains(detail.Body, []byte(id)) {
		t.Fatalf("generated detail status=%d body=%s", detail.StatusCode, detail.Body)
	}
	approve := generatedRuntimeRequest(t, manager, info.ID, http.MethodPost, spec.Plugin.ServiceBaseURL+pluginAPIBasePath(*spec)+"/"+id+"/approve", nil)
	if approve.StatusCode != http.StatusOK || !bytes.Contains(approve.Body, []byte(`"state":"approved"`)) {
		t.Fatalf("generated approve status=%d body=%s", approve.StatusCode, approve.Body)
	}
	export := generatedRuntimeRequest(t, manager, info.ID, http.MethodGet, spec.Plugin.ServiceBaseURL+pluginAPIBasePath(*spec)+"/export", nil)
	if export.StatusCode != http.StatusOK || !bytes.Contains(export.Body, []byte(pluginsdk.DocumentExportJobKind)) {
		t.Fatalf("generated export status=%d body=%s", export.StatusCode, export.Body)
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
	linkGeneratedDirectory(t, source, target)
}

func linkDocumentUIPackage(t *testing.T, repoRoot, pluginDir string) {
	t.Helper()
	workspace := filepath.Clean(filepath.Join(pluginDir, "..", "..", ".."))
	target := filepath.Join(workspace, "packages", "skoll-document-ui")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	linkGeneratedDirectory(t, filepath.Join(repoRoot, "packages", "skoll-document-ui"), target)
}

func generatedGoWorkspace(t *testing.T, repoRoot, pluginDir string) string {
	t.Helper()
	workspace := filepath.Clean(filepath.Join(pluginDir, "..", "..", ".."))
	for _, name := range []string{"go.mod", "go.sum"} {
		raw, err := os.ReadFile(filepath.Join(repoRoot, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(workspace, name), raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	linkGeneratedDirectory(t, filepath.Join(repoRoot, "pkg"), filepath.Join(workspace, "pkg"))
	return "off"
}

func linkGeneratedDirectory(t *testing.T, source, target string) {
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

type generatedTransaction struct{ ctx context.Context }

func (t generatedTransaction) Context() context.Context { return t.ctx }

type generatedTransactions struct{}

func (generatedTransactions) Within(ctx context.Context, fn func(pluginsdk.Transaction) error) error {
	return fn(generatedTransaction{ctx: ctx})
}

type generatedScopes struct{}

func (generatedScopes) Resolve(context.Context, pluginsdk.Permission) (pluginsdk.ScopePredicate, error) {
	return pluginsdk.NewScopePredicate(pluginsdk.TrustedScope{SubjectID: "generated-user", TenantIDs: []string{"tenant-demo"}, AllOwners: true, AllOrganizations: true})
}

type generatedDataStore struct{}

func (generatedDataStore) Query(context.Context, pluginsdk.DataQuery) (pluginsdk.DataPage, error) {
	return pluginsdk.DataPage{}, nil
}
func (generatedDataStore) Mutate(context.Context, pluginsdk.DataMutation) (pluginsdk.DataMutationResult, error) {
	return pluginsdk.DataMutationResult{}, nil
}
func (generatedDataStore) Aggregate(context.Context, pluginsdk.DataAggregateQuery) (pluginsdk.DataAggregatePage, error) {
	return pluginsdk.DataAggregatePage{}, nil
}

type generatedDocumentNumbers struct{}

func (generatedDocumentNumbers) Preview(context.Context, pluginsdk.DocumentNumberInput) (pluginsdk.DocumentNumberResult, error) {
	return pluginsdk.DocumentNumberResult{Number: "PR-000001", Sequence: 1}, nil
}
func (generatedDocumentNumbers) Issue(context.Context, pluginsdk.DocumentNumberInput) (pluginsdk.DocumentNumberResult, error) {
	return pluginsdk.DocumentNumberResult{Number: "PR-000001", Sequence: 1}, nil
}

type generatedDocumentBinding struct {
	schema pluginsdk.DocumentSchema
	result pluginsdk.DocumentWorkflowResult
}

type generatedDocumentHost struct {
	mu    sync.RWMutex
	items map[string]generatedDocumentBinding
}

func newGeneratedDocumentHost() *generatedDocumentHost {
	return &generatedDocumentHost{items: make(map[string]generatedDocumentBinding)}
}

func (h *generatedDocumentHost) Submit(_ context.Context, input pluginsdk.DocumentWorkflowSubmitInput) (pluginsdk.DocumentWorkflowResult, error) {
	if err := input.Validate(); err != nil {
		return pluginsdk.DocumentWorkflowResult{}, err
	}
	state, err := input.Schema.NextState(input.Schema.InitialState, "submit", input.Comment)
	if err != nil {
		return pluginsdk.DocumentWorkflowResult{}, err
	}
	now := time.Now().UTC()
	result := pluginsdk.DocumentWorkflowResult{
		Document: pluginsdk.DocumentRecord{
			ID: input.Draft.ID, Type: input.Draft.Type, SchemaVersion: input.Draft.SchemaVersion,
			Number: input.Draft.Number, Title: input.Draft.Title, State: state, Version: 1,
			Header: input.Draft.Header, Lines: input.Draft.Lines,
			Metadata: pluginsdk.DocumentMetadata{CreatedAt: now, UpdatedAt: now, CreatedBy: "generated-user", UpdatedBy: "generated-user"},
		},
		Workflow: pluginsdk.WorkflowInstance{
			ID: input.InstanceID, DefinitionID: input.DefinitionID, DefinitionKey: input.DefinitionID,
			BusinessType: input.Draft.Type, BusinessID: input.Draft.ID, Title: input.Draft.Title,
			Status: pluginsdk.WorkflowInstanceRunning, CurrentNode: "approval", Starter: pluginsdk.WorkflowActor{ID: "generated-user", Name: "Generated User"},
			Tasks:     []pluginsdk.WorkflowTask{{ID: input.Draft.ID + "-task", InstanceID: input.InstanceID, NodeID: "approval", Assignee: pluginsdk.WorkflowActor{ID: "approver", Name: "Approver"}, Status: pluginsdk.WorkflowTaskPending, CreatedAt: now}},
			CreatedAt: now, UpdatedAt: now,
		},
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if prior, exists := h.items[input.Draft.ID]; exists {
		prior.result.Duplicate = true
		return prior.result, nil
	}
	h.items[input.Draft.ID] = generatedDocumentBinding{schema: input.Schema, result: result}
	return result, nil
}

func (h *generatedDocumentHost) Act(_ context.Context, input pluginsdk.DocumentWorkflowActionInput) (pluginsdk.DocumentWorkflowResult, error) {
	if err := input.Validate(); err != nil {
		return pluginsdk.DocumentWorkflowResult{}, err
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	binding, exists := h.items[input.DocumentID]
	if !exists {
		return pluginsdk.DocumentWorkflowResult{}, fmt.Errorf("document not found")
	}
	if binding.result.Document.Version != input.ExpectedVersion {
		return pluginsdk.DocumentWorkflowResult{}, fmt.Errorf("document version conflict")
	}
	next, err := binding.schema.NextState(binding.result.Document.State, string(input.Action), input.Comment)
	if err != nil {
		return pluginsdk.DocumentWorkflowResult{}, err
	}
	now := time.Now().UTC()
	binding.result.Document.State = next
	binding.result.Document.Version++
	binding.result.Document.Metadata.UpdatedAt = now
	binding.result.Document.Metadata.UpdatedBy = "approver"
	binding.result.Workflow.Status = pluginsdk.WorkflowInstanceApproved
	binding.result.Workflow.CurrentNode = "end"
	binding.result.Workflow.UpdatedAt = now
	for index := range binding.result.Workflow.Tasks {
		if binding.result.Workflow.Tasks[index].ID == input.TaskID {
			binding.result.Workflow.Tasks[index].Status = pluginsdk.WorkflowTaskApproved
			binding.result.Workflow.Tasks[index].CompletedAt = &now
		}
	}
	h.items[input.DocumentID] = binding
	return binding.result, nil
}

func (h *generatedDocumentHost) Get(_ context.Context, input pluginsdk.DocumentWorkflowGetInput) (pluginsdk.DocumentWorkflowResult, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	binding, exists := h.items[input.DocumentID]
	if !exists {
		return pluginsdk.DocumentWorkflowResult{}, fmt.Errorf("document not found")
	}
	return binding.result, nil
}

func (h *generatedDocumentHost) Search(_ context.Context, input pluginsdk.DocumentSearchInput) (pluginsdk.DocumentSearchPage, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	page := pluginsdk.DocumentSearchPage{Items: make([]pluginsdk.DocumentSummary, 0, len(h.items))}
	for _, binding := range h.items {
		document := binding.result.Document
		if input.Text != "" && !strings.Contains(strings.ToLower(document.Title), strings.ToLower(input.Text)) {
			continue
		}
		if len(input.States) > 0 && document.State != input.States[0] {
			continue
		}
		page.Items = append(page.Items, pluginsdk.DocumentSummary{
			ID: document.ID, Type: document.Type, Number: document.Number, Title: document.Title, State: document.State, Version: document.Version,
			CreatedAt: document.Metadata.CreatedAt, UpdatedAt: document.Metadata.UpdatedAt, CreatedBy: document.Metadata.CreatedBy, UpdatedBy: document.Metadata.UpdatedBy,
		})
	}
	return page, nil
}

func (h *generatedDocumentHost) Print(_ context.Context, input pluginsdk.DocumentPrintInput) (pluginsdk.DocumentPrintPayload, error) {
	binding, err := h.binding(input.DocumentID)
	if err != nil {
		return pluginsdk.DocumentPrintPayload{}, err
	}
	return pluginsdk.DocumentPrintPayload{Schema: binding.schema, Document: binding.result.Document, Workflow: binding.result.Workflow, GeneratedAt: time.Now().UTC()}, nil
}

func (h *generatedDocumentHost) Export(_ context.Context, input pluginsdk.DocumentExportInput) (pluginsdk.Job, error) {
	payload, _ := json.Marshal(pluginsdk.DocumentExportPlan{Version: 1, Search: input.Search, Format: input.Format, MaxRows: input.MaxRows, Actor: pluginsdk.WorkflowActor{ID: "generated-user"}})
	now := time.Now().UTC()
	return pluginsdk.Job{ID: input.JobID, Kind: pluginsdk.DocumentExportJobKind, IdempotencyKey: input.IdempotencyKey, Payload: payload, Status: pluginsdk.JobStatusScheduled, RunAt: now, MaxAttempts: 3, CreatedAt: now, UpdatedAt: now}, nil
}

func (h *generatedDocumentHost) binding(id string) (generatedDocumentBinding, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	binding, exists := h.items[id]
	if !exists {
		return generatedDocumentBinding{}, fmt.Errorf("document not found")
	}
	return binding, nil
}

func (*generatedDocumentHost) AddAttachment(context.Context, pluginsdk.DocumentAttachmentAddInput) (pluginsdk.DocumentAttachmentResult, error) {
	return pluginsdk.DocumentAttachmentResult{}, nil
}
func (*generatedDocumentHost) RemoveAttachment(context.Context, pluginsdk.DocumentAttachmentRemoveInput) (pluginsdk.DocumentAttachmentResult, error) {
	return pluginsdk.DocumentAttachmentResult{}, nil
}
func (*generatedDocumentHost) ListAttachments(context.Context, pluginsdk.DocumentCollaborationQueryInput) ([]pluginsdk.DocumentAttachment, error) {
	return []pluginsdk.DocumentAttachment{}, nil
}
func (*generatedDocumentHost) AddComment(context.Context, pluginsdk.DocumentCommentAddInput) (pluginsdk.DocumentCommentResult, error) {
	return pluginsdk.DocumentCommentResult{}, nil
}
func (*generatedDocumentHost) ListComments(context.Context, pluginsdk.DocumentCollaborationQueryInput) ([]pluginsdk.DocumentComment, error) {
	return []pluginsdk.DocumentComment{}, nil
}
func (*generatedDocumentHost) Timeline(context.Context, pluginsdk.DocumentTimelineQueryInput) (pluginsdk.DocumentTimelinePage, error) {
	return pluginsdk.DocumentTimelinePage{}, nil
}

type generatedFiles struct{}

func (generatedFiles) Store(context.Context, pluginsdk.FileWrite) (pluginsdk.FileObject, error) {
	return pluginsdk.FileObject{}, nil
}
func (generatedFiles) List(context.Context, pluginsdk.FileQuery) ([]pluginsdk.FileObject, error) {
	return nil, nil
}
func (generatedFiles) Get(context.Context, string) (pluginsdk.FileObject, error) {
	return pluginsdk.FileObject{}, nil
}
func (generatedFiles) Download(context.Context, string) (pluginsdk.FileDownload, error) {
	return pluginsdk.FileDownload{}, nil
}
func (generatedFiles) Delete(context.Context, string) error { return nil }

type generatedAudit struct{}

func (generatedAudit) Record(context.Context, pluginsdk.AuditEntry) (pluginsdk.AuditReceipt, error) {
	return pluginsdk.AuditReceipt{}, nil
}

type generatedConfig struct{}

func (generatedConfig) Get(context.Context) (map[string]any, error) { return map[string]any{}, nil }
func (generatedConfig) Replace(_ context.Context, values map[string]any) (map[string]any, error) {
	return values, nil
}

type generatedSecrets struct{}

func (generatedSecrets) Get(context.Context, string) (string, error) { return "", nil }
func (generatedSecrets) Set(context.Context, string, string) error   { return nil }

type generatedWorkflows struct {
	mu          sync.Mutex
	definitions map[string]pluginsdk.WorkflowDefinition
}

func newGeneratedWorkflows() *generatedWorkflows {
	return &generatedWorkflows{definitions: make(map[string]pluginsdk.WorkflowDefinition)}
}
func (w *generatedWorkflows) CreateDefinition(_ context.Context, input pluginsdk.WorkflowDefinitionInput) (pluginsdk.WorkflowDefinition, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	now := time.Now().UTC()
	item := pluginsdk.WorkflowDefinition{ID: input.ID, Key: input.Key, Name: input.Name, Version: input.Version, Status: pluginsdk.WorkflowDefinitionDraft, Nodes: input.Nodes, Transitions: input.Transitions, CreatedAt: now, UpdatedAt: now}
	w.definitions[item.ID] = item
	return item, nil
}
func (w *generatedWorkflows) GetDefinition(_ context.Context, id string) (pluginsdk.WorkflowDefinition, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	item, ok := w.definitions[id]
	if !ok {
		return pluginsdk.WorkflowDefinition{}, fmt.Errorf("definition not found")
	}
	return item, nil
}
func (w *generatedWorkflows) PublishDefinition(_ context.Context, id string) (pluginsdk.WorkflowDefinition, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	item, ok := w.definitions[id]
	if !ok {
		return pluginsdk.WorkflowDefinition{}, fmt.Errorf("definition not found")
	}
	item.Status = pluginsdk.WorkflowDefinitionPublished
	item.UpdatedAt = time.Now().UTC()
	w.definitions[id] = item
	return item, nil
}
func (*generatedWorkflows) Start(context.Context, pluginsdk.WorkflowStartInput) (pluginsdk.WorkflowInstance, error) {
	return pluginsdk.WorkflowInstance{}, nil
}
func (*generatedWorkflows) GetInstance(context.Context, string) (pluginsdk.WorkflowInstance, error) {
	return pluginsdk.WorkflowInstance{}, nil
}
func (*generatedWorkflows) Approve(context.Context, pluginsdk.WorkflowTaskActionInput) (pluginsdk.WorkflowInstance, error) {
	return pluginsdk.WorkflowInstance{}, nil
}
func (*generatedWorkflows) Reject(context.Context, pluginsdk.WorkflowTaskActionInput) (pluginsdk.WorkflowInstance, error) {
	return pluginsdk.WorkflowInstance{}, nil
}
func (*generatedWorkflows) Withdraw(context.Context, pluginsdk.WorkflowInstanceActionInput) (pluginsdk.WorkflowInstance, error) {
	return pluginsdk.WorkflowInstance{}, nil
}
func (*generatedWorkflows) Cancel(context.Context, pluginsdk.WorkflowInstanceActionInput) (pluginsdk.WorkflowInstance, error) {
	return pluginsdk.WorkflowInstance{}, nil
}
func (*generatedWorkflows) Transfer(context.Context, pluginsdk.WorkflowTargetActionInput) (pluginsdk.WorkflowInstance, error) {
	return pluginsdk.WorkflowInstance{}, nil
}
func (*generatedWorkflows) Copy(context.Context, pluginsdk.WorkflowTargetActionInput) (pluginsdk.WorkflowInstance, error) {
	return pluginsdk.WorkflowInstance{}, nil
}

type generatedJobs struct{}

type generatedEvents struct{}

func (generatedEvents) Publish(context.Context, pluginsdk.EventPublication) (pluginsdk.EventEnvelope, error) {
	return pluginsdk.EventEnvelope{}, nil
}

func (generatedJobs) Schedule(context.Context, pluginsdk.JobScheduleInput) (pluginsdk.Job, error) {
	return pluginsdk.Job{}, nil
}
func (generatedJobs) LeaseDue(context.Context, pluginsdk.JobLeaseInput) ([]pluginsdk.Job, error) {
	return nil, nil
}
func (generatedJobs) Complete(context.Context, pluginsdk.JobCompleteInput) (pluginsdk.Job, error) {
	return pluginsdk.Job{}, nil
}
func (generatedJobs) Fail(context.Context, pluginsdk.JobFailInput) (pluginsdk.Job, error) {
	return pluginsdk.Job{}, nil
}
func (generatedJobs) Get(context.Context, string) (pluginsdk.Job, error) { return pluginsdk.Job{}, nil }
func (generatedJobs) List(context.Context, pluginsdk.JobQuery) ([]pluginsdk.Job, error) {
	return nil, nil
}

func generatedHostServices(pluginID string, documents *generatedDocumentHost) pluginsdk.HostServices {
	return pluginsdk.HostServices{
		PluginID: pluginID, Transactions: generatedTransactions{}, DataScopes: generatedScopes{}, DataStore: generatedDataStore{},
		Events:          generatedEvents{},
		DocumentNumbers: generatedDocumentNumbers{}, Documents: documents, Files: generatedFiles{}, Audit: generatedAudit{},
		Config: generatedConfig{}, Secrets: generatedSecrets{}, Workflows: newGeneratedWorkflows(), Jobs: generatedJobs{},
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
