package bootstrap

import (
	"bytes"
	"context"
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
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/plugin"
	"github.com/tinboxw/skoll/internal/plugin/datastore"
	"github.com/tinboxw/skoll/internal/plugin/hostservice"
	storesql "github.com/tinboxw/skoll/internal/store/sql"
	"github.com/tinboxw/skoll/internal/store/sql/gormrepo"
	"github.com/tinboxw/skoll/pkg/pluginsdk"
	"github.com/tinboxw/skoll/pkg/security"
)

func TestIndependentPluginDataStoreProcessLifecycleE2E(t *testing.T) {
	const pluginID = "datastore_e2e"
	const jwtSecret = "datastore-e2e-jwt-secret"
	repositoryRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	pluginSource := filepath.Join(t.TempDir(), pluginID)
	copyDataStoreE2EDirectory(t, filepath.Join(repositoryRoot, "plugins", pluginID), pluginSource)
	address := reserveDataStoreE2EAddress(t)
	rewriteDataStoreE2EAddress(t, filepath.Join(pluginSource, "plugin.yaml"), address)
	buildDataStoreE2EBackend(t, repositoryRoot, pluginSource, pluginID)

	packageResult, err := plugin.BuildPackage(pluginSource, filepath.Join(t.TempDir(), "dist"), plugin.NewFileLoader())
	if err != nil {
		t.Fatalf("build independent plugin package: %v", err)
	}
	installedDir, packageInfo, err := plugin.InstallPackage(packageResult.ArtifactPath, packageResult.ChecksumPath, filepath.Join(t.TempDir(), "plugins"), plugin.NewFileLoader())
	if err != nil {
		t.Fatalf("install independent plugin package: %v", err)
	}
	for _, required := range []string{"plugin.yaml", datastore.SchemaManifestName, "migrations/001_records.up.sql", dataStoreE2EBackendRelativePath(pluginID)} {
		if _, statErr := os.Stat(filepath.Join(installedDir, filepath.FromSlash(required))); statErr != nil {
			t.Fatalf("installed package missing %s: %v", required, statErr)
		}
	}

	db := gormrepo.TestDB(t)
	unitOfWork := storesql.NewUnitOfWorkWithDB(db)
	registry := datastore.NewSchemaRegistry()
	dataLifecycle, err := datastore.NewLifecycle(db, registry)
	if err != nil {
		t.Fatal(err)
	}
	dataDirectories, err := plugin.NewPluginDataDirectories(filepath.Join(t.TempDir(), "plugin-data"))
	if err != nil {
		t.Fatal(err)
	}
	transactions, err := hostservice.NewTransactionService(unitOfWork)
	if err != nil {
		t.Fatal(err)
	}
	scopes := dataStoreE2EScopes{}
	audit := &dataStoreE2EAudit{}
	configService := &dataStoreE2EConfig{}
	secretService := &dataStoreE2ESecrets{}
	gateway, err := plugin.NewHostGateway(func(requestedPluginID string) (pluginsdk.HostServices, error) {
		if requestedPluginID != pluginID {
			return pluginsdk.HostServices{}, errors.New("unexpected plugin identity")
		}
		store, storeErr := datastore.NewService(db, unitOfWork, registry, scopes, audit, datastore.DialectSQLite, requestedPluginID)
		if storeErr != nil {
			return pluginsdk.HostServices{}, storeErr
		}
		return pluginsdk.HostServices{
			PluginID: requestedPluginID, Transactions: transactions, DataScopes: scopes, DataStore: store,
			Events:          dataStoreE2EEvents{},
			DocumentNumbers: dataStoreE2EDocumentNumbers{}, Documents: dataStoreE2EDocuments{}, Files: dataStoreE2EFiles{}, Audit: audit, Config: configService, Secrets: secretService,
			Workflows: dataStoreE2EWorkflows{}, Jobs: dataStoreE2EJobs{},
		}, nil
	}, jwtSecret, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = gateway.Close() })

	runtimeManager := plugin.NewRuntimeManager(plugin.NewFileLoader(), plugin.NewTopologicalResolver())
	manager := &pluginManagerWithExtensions{
		Manager: runtimeManager, builtinInfos: map[string]plugin.Info{}, extensions: map[string]plugin.RegistrySnapshot{},
		routeHandlers: map[string]http.HandlerFunc{}, routePermissions: mustEmptyRoutePermissionRegistry(),
		migrationHook: plugin.NewPluginMigrationHook(gormrepo.NewPluginMigrationStore(db), nil), dataLifecycle: dataLifecycle,
		dataDirectories: dataDirectories, hostGateway: gateway, eventSubscriptions: map[string][]func(){},
	}
	manager.serviceSupervisor = plugin.NewServiceSupervisor(
		plugin.NewManagedProcessLauncher(plugin.NewHTTPHealthChecker(time.Second), 25*time.Millisecond, gateway, dataDirectories),
		nil, 8*time.Second, 3*time.Second,
	)
	t.Cleanup(func() { _ = manager.Close() })

	installed, err := manager.Install(installedDir)
	if err != nil {
		t.Fatal(err)
	}
	if installed.ID != packageInfo.ID || installed.State != plugin.StateInstalled {
		t.Fatalf("unexpected installed plugin: %+v package=%+v", installed, packageInfo)
	}
	if err = manager.Enable(pluginID); err != nil {
		t.Fatalf("enable independent plugin: %v", err)
	}

	token := signDataStoreE2EUser(t, jwtSecret, "employee-1", "tenant-a", "org-a")
	baseURL := "http://" + address + "/v1/plugins/" + pluginID + "/api"
	createdPayload := map[string]any{"id": "record-1", "name": "Stable record", "status": "active", "quantity": 10}
	created := dataStoreE2ERequest(t, http.MethodPost, baseURL+"/records", token, createdPayload)
	if created.status != http.StatusCreated || !bytes.Contains(created.body, []byte("Stable record")) {
		t.Fatalf("create status=%d body=%s", created.status, created.body)
	}
	replayed := dataStoreE2ERequest(t, http.MethodPost, baseURL+"/records", token, createdPayload)
	if replayed.status != http.StatusCreated || !bytes.Equal(replayed.body, created.body) {
		t.Fatalf("idempotent replay status=%d body=%s first=%s", replayed.status, replayed.body, created.body)
	}
	inactive := dataStoreE2ERequest(t, http.MethodPost, baseURL+"/records", token, map[string]any{
		"id": "record-2", "name": "Inactive record", "status": "inactive", "quantity": 3,
	})
	if inactive.status != http.StatusCreated {
		t.Fatalf("create inactive status=%d body=%s", inactive.status, inactive.body)
	}

	const adjustments = 8
	adjustmentErrors := make(chan error, adjustments)
	var adjustmentsDone sync.WaitGroup
	for index := 0; index < adjustments; index++ {
		adjustmentsDone.Add(1)
		go func(index int) {
			defer adjustmentsDone.Done()
			response, requestErr := dataStoreE2EDo(http.MethodPost, baseURL+"/records/adjust", token, map[string]any{
				"id": "record-1", "delta": 1, "idempotencyKey": fmt.Sprintf("record-1.adjust.%d", index),
			})
			if requestErr != nil {
				adjustmentErrors <- requestErr
				return
			}
			if response.status != http.StatusOK {
				adjustmentErrors <- fmt.Errorf("adjust %d status=%d body=%s", index, response.status, response.body)
			}
		}(index)
	}
	adjustmentsDone.Wait()
	close(adjustmentErrors)
	for adjustmentErr := range adjustmentErrors {
		t.Fatal(adjustmentErr)
	}
	aggregate := dataStoreE2ERequest(t, http.MethodGet, baseURL+"/records/aggregate", token, nil)
	assertDataStoreE2EAggregate(t, aggregate, map[string][]string{
		"active":   {"1", "18"},
		"inactive": {"1", "3"},
	})

	ledger := dataStoreE2ERequest(t, http.MethodPost, baseURL+"/ledger", token, map[string]string{"id": "ledger-1", "recordId": "record-1"})
	if ledger.status != http.StatusCreated {
		t.Fatalf("append ledger status=%d body=%s", ledger.status, ledger.body)
	}
	deniedDelete := dataStoreE2ERequest(t, http.MethodDelete, baseURL+"/ledger", token, map[string]string{"id": "ledger-1"})
	if deniedDelete.status != http.StatusUnprocessableEntity || !bytes.Contains(deniedDelete.body, []byte("append-only")) {
		t.Fatalf("append-only denial status=%d body=%s", deniedDelete.status, deniedDelete.body)
	}
	rollback := dataStoreE2ERequest(t, http.MethodPost, baseURL+"/records/rollback", token, map[string]string{"id": "rolled-back", "name": "Must disappear"})
	if rollback.status != http.StatusInternalServerError {
		t.Fatalf("rollback probe status=%d body=%s", rollback.status, rollback.body)
	}
	afterRollback := dataStoreE2ERequest(t, http.MethodGet, baseURL+"/records", token, nil)
	if bytes.Contains(afterRollback.body, []byte("rolled-back")) {
		t.Fatalf("remote rollback retained row: %s", afterRollback.body)
	}
	forged := dataStoreE2ERequest(t, http.MethodGet, baseURL+"/records/forged", token, nil)
	if forged.status != http.StatusForbidden || !bytes.Contains(forged.body, []byte("forbidden")) {
		t.Fatalf("forged scope status=%d body=%s", forged.status, forged.body)
	}

	if err = manager.Disable(pluginID); err != nil {
		t.Fatalf("disable independent plugin: %v", err)
	}
	waitDataStoreE2EUnavailable(t, baseURL+"/records")
	if err = manager.Enable(pluginID); err != nil {
		t.Fatalf("restart independent plugin: %v", err)
	}
	listed := dataStoreE2ERequest(t, http.MethodGet, baseURL+"/records", token, nil)
	if listed.status != http.StatusOK || !bytes.Contains(listed.body, []byte("Stable record")) {
		t.Fatalf("restart list status=%d body=%s", listed.status, listed.body)
	}
	restartedAggregate := dataStoreE2ERequest(t, http.MethodGet, baseURL+"/records/aggregate", token, nil)
	assertDataStoreE2EAggregate(t, restartedAggregate, map[string][]string{
		"active":   {"1", "18"},
		"inactive": {"1", "3"},
	})

	recordsTable := dataStoreE2EPhysicalTable(t, registry, pluginID, "records")
	ledgerTable := dataStoreE2EPhysicalTable(t, registry, pluginID, "ledger_entries")
	if err = manager.Disable(pluginID); err != nil {
		t.Fatal(err)
	}
	if err = manager.RollbackPluginData(pluginID, 1); err != nil {
		t.Fatalf("explicit datastore rollback: %v", err)
	}
	if db.Migrator().HasTable(recordsTable) || db.Migrator().HasTable(ledgerTable) {
		t.Fatal("explicit rollback retained datastore tables")
	}
	if err = manager.Enable(pluginID); err != nil {
		t.Fatalf("enable after explicit rollback: %v", err)
	}
	empty := dataStoreE2ERequest(t, http.MethodGet, baseURL+"/records", token, nil)
	if empty.status != http.StatusOK || bytes.Contains(empty.body, []byte("Stable record")) {
		t.Fatalf("rollback did not recreate a clean datastore: status=%d body=%s", empty.status, empty.body)
	}
	second := dataStoreE2ERequest(t, http.MethodPost, baseURL+"/records", token, map[string]string{"id": "record-after-rollback", "name": "Disposable record"})
	if second.status != http.StatusCreated {
		t.Fatalf("create after rollback status=%d body=%s", second.status, second.body)
	}

	if err = manager.Uninstall(pluginID); err != nil {
		t.Fatalf("uninstall independent plugin: %v", err)
	}
	waitDataStoreE2EUnavailable(t, baseURL+"/records")
	if db.Migrator().HasTable(recordsTable) || db.Migrator().HasTable(ledgerTable) {
		t.Fatal("drop uninstall retained physical datastore tables")
	}
	if _, exists := registry.Snapshot(pluginID); exists {
		t.Fatal("drop uninstall retained schema registry entry")
	}
	var mutationRows int64
	if err = db.Model(&gormrepo.PluginDataMutationModel{}).Where("plugin_id = ?", pluginID).Count(&mutationRows).Error; err != nil || mutationRows != 0 {
		t.Fatalf("drop uninstall idempotency rows=%d err=%v", mutationRows, err)
	}
	if !audit.hasAction("datastore.insert") {
		t.Fatalf("datastore mutation was not audited: %+v", audit.actions())
	}
}

type dataStoreE2EDocumentNumbers struct{}

func (dataStoreE2EDocumentNumbers) Preview(_ context.Context, input pluginsdk.DocumentNumberInput) (pluginsdk.DocumentNumberResult, error) {
	return pluginsdk.DocumentNumberResult{Number: input.Rule.Prefix + "-000001", Sequence: 1}, nil
}

type dataStoreE2EDocuments struct{}

func (dataStoreE2EDocuments) Submit(context.Context, pluginsdk.DocumentWorkflowSubmitInput) (pluginsdk.DocumentWorkflowResult, error) {
	return pluginsdk.DocumentWorkflowResult{}, nil
}
func (dataStoreE2EDocuments) Act(context.Context, pluginsdk.DocumentWorkflowActionInput) (pluginsdk.DocumentWorkflowResult, error) {
	return pluginsdk.DocumentWorkflowResult{}, nil
}
func (dataStoreE2EDocuments) Get(context.Context, pluginsdk.DocumentWorkflowGetInput) (pluginsdk.DocumentWorkflowResult, error) {
	return pluginsdk.DocumentWorkflowResult{}, nil
}
func (dataStoreE2EDocuments) AddAttachment(context.Context, pluginsdk.DocumentAttachmentAddInput) (pluginsdk.DocumentAttachmentResult, error) {
	return pluginsdk.DocumentAttachmentResult{}, nil
}
func (dataStoreE2EDocuments) RemoveAttachment(context.Context, pluginsdk.DocumentAttachmentRemoveInput) (pluginsdk.DocumentAttachmentResult, error) {
	return pluginsdk.DocumentAttachmentResult{}, nil
}
func (dataStoreE2EDocuments) ListAttachments(context.Context, pluginsdk.DocumentCollaborationQueryInput) ([]pluginsdk.DocumentAttachment, error) {
	return nil, nil
}
func (dataStoreE2EDocuments) AddComment(context.Context, pluginsdk.DocumentCommentAddInput) (pluginsdk.DocumentCommentResult, error) {
	return pluginsdk.DocumentCommentResult{}, nil
}
func (dataStoreE2EDocuments) ListComments(context.Context, pluginsdk.DocumentCollaborationQueryInput) ([]pluginsdk.DocumentComment, error) {
	return nil, nil
}
func (dataStoreE2EDocuments) Timeline(context.Context, pluginsdk.DocumentTimelineQueryInput) (pluginsdk.DocumentTimelinePage, error) {
	return pluginsdk.DocumentTimelinePage{}, nil
}
func (dataStoreE2EDocuments) Search(context.Context, pluginsdk.DocumentSearchInput) (pluginsdk.DocumentSearchPage, error) {
	return pluginsdk.DocumentSearchPage{}, nil
}
func (dataStoreE2EDocuments) Print(context.Context, pluginsdk.DocumentPrintInput) (pluginsdk.DocumentPrintPayload, error) {
	return pluginsdk.DocumentPrintPayload{}, nil
}
func (dataStoreE2EDocuments) Export(context.Context, pluginsdk.DocumentExportInput) (pluginsdk.Job, error) {
	return pluginsdk.Job{}, nil
}

func (dataStoreE2EDocumentNumbers) Issue(_ context.Context, input pluginsdk.DocumentNumberInput) (pluginsdk.DocumentNumberResult, error) {
	return pluginsdk.DocumentNumberResult{Number: input.Rule.Prefix + "-000001", Sequence: 1}, nil
}

type dataStoreE2EScopes struct{}

func (dataStoreE2EScopes) Resolve(ctx context.Context, _ pluginsdk.Permission) (pluginsdk.ScopePredicate, error) {
	claims, ok := security.JWTClaimsFromContext(ctx)
	if !ok || strings.TrimSpace(claims.Subject) == "" || len(claims.OrganizationPath) == 0 {
		return pluginsdk.ScopePredicate{}, errors.New("trusted user scope is required")
	}
	return pluginsdk.NewScopePredicate(pluginsdk.TrustedScope{
		SubjectID: claims.Subject, TenantIDs: []string{claims.OrganizationPath[0]},
		OrganizationIDs: []string{claims.OrganizationID}, OwnerIDs: []string{claims.Subject},
	})
}

type dataStoreE2EAudit struct {
	mu      sync.Mutex
	entries []pluginsdk.AuditEntry
}

func (a *dataStoreE2EAudit) Record(_ context.Context, entry pluginsdk.AuditEntry) (pluginsdk.AuditReceipt, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.entries = append(a.entries, entry)
	return pluginsdk.AuditReceipt{ID: "audit-e2e", OccurredAt: time.Now().UTC()}, nil
}

func (a *dataStoreE2EAudit) hasAction(action string) bool {
	for _, current := range a.actions() {
		if current == action {
			return true
		}
	}
	return false
}

func (a *dataStoreE2EAudit) actions() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]string, 0, len(a.entries))
	for _, entry := range a.entries {
		out = append(out, entry.Action)
	}
	return out
}

type dataStoreE2EFiles struct{}

func (dataStoreE2EFiles) Store(context.Context, pluginsdk.FileWrite) (pluginsdk.FileObject, error) {
	return pluginsdk.FileObject{ID: "file-e2e"}, nil
}
func (dataStoreE2EFiles) List(context.Context, pluginsdk.FileQuery) ([]pluginsdk.FileObject, error) {
	return []pluginsdk.FileObject{}, nil
}
func (dataStoreE2EFiles) Get(context.Context, string) (pluginsdk.FileObject, error) {
	return pluginsdk.FileObject{ID: "file-e2e"}, nil
}
func (dataStoreE2EFiles) Download(context.Context, string) (pluginsdk.FileDownload, error) {
	return pluginsdk.FileDownload{ID: "file-e2e"}, nil
}
func (dataStoreE2EFiles) Delete(context.Context, string) error { return nil }

type dataStoreE2EConfig struct {
	mu     sync.Mutex
	config map[string]any
}

func (a *dataStoreE2EConfig) Get(context.Context) (map[string]any, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return cloneDataStoreE2EMap(a.config), nil
}
func (a *dataStoreE2EConfig) Replace(_ context.Context, values map[string]any) (map[string]any, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.config = cloneDataStoreE2EMap(values)
	return cloneDataStoreE2EMap(a.config), nil
}

type dataStoreE2ESecrets struct {
	mu     sync.Mutex
	secret string
}

func (a *dataStoreE2ESecrets) Get(context.Context, string) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.secret, nil
}
func (a *dataStoreE2ESecrets) Set(_ context.Context, _ string, value string) error {
	a.mu.Lock()
	a.secret = value
	a.mu.Unlock()
	return nil
}

type dataStoreE2EEvents struct{}

func (dataStoreE2EEvents) Publish(context.Context, pluginsdk.EventPublication) (pluginsdk.EventEnvelope, error) {
	return pluginsdk.EventEnvelope{}, nil
}

type dataStoreE2EWorkflows struct{}

func (dataStoreE2EWorkflows) CreateDefinition(context.Context, pluginsdk.WorkflowDefinitionInput) (pluginsdk.WorkflowDefinition, error) {
	return pluginsdk.WorkflowDefinition{ID: "definition-e2e"}, nil
}
func (dataStoreE2EWorkflows) GetDefinition(context.Context, string) (pluginsdk.WorkflowDefinition, error) {
	return pluginsdk.WorkflowDefinition{ID: "definition-e2e"}, nil
}
func (dataStoreE2EWorkflows) PublishDefinition(context.Context, string) (pluginsdk.WorkflowDefinition, error) {
	return pluginsdk.WorkflowDefinition{ID: "definition-e2e", Status: pluginsdk.WorkflowDefinitionPublished}, nil
}
func (dataStoreE2EWorkflows) Start(context.Context, pluginsdk.WorkflowStartInput) (pluginsdk.WorkflowInstance, error) {
	return pluginsdk.WorkflowInstance{ID: "instance-e2e"}, nil
}
func (dataStoreE2EWorkflows) GetInstance(context.Context, string) (pluginsdk.WorkflowInstance, error) {
	return pluginsdk.WorkflowInstance{ID: "instance-e2e"}, nil
}
func (dataStoreE2EWorkflows) Approve(context.Context, pluginsdk.WorkflowTaskActionInput) (pluginsdk.WorkflowInstance, error) {
	return pluginsdk.WorkflowInstance{ID: "instance-e2e", Status: pluginsdk.WorkflowInstanceApproved}, nil
}
func (dataStoreE2EWorkflows) Reject(context.Context, pluginsdk.WorkflowTaskActionInput) (pluginsdk.WorkflowInstance, error) {
	return pluginsdk.WorkflowInstance{ID: "instance-e2e", Status: pluginsdk.WorkflowInstanceRejected}, nil
}
func (dataStoreE2EWorkflows) Withdraw(context.Context, pluginsdk.WorkflowInstanceActionInput) (pluginsdk.WorkflowInstance, error) {
	return pluginsdk.WorkflowInstance{ID: "instance-e2e", Status: pluginsdk.WorkflowInstanceWithdrawn}, nil
}
func (dataStoreE2EWorkflows) Cancel(context.Context, pluginsdk.WorkflowInstanceActionInput) (pluginsdk.WorkflowInstance, error) {
	return pluginsdk.WorkflowInstance{ID: "instance-e2e", Status: pluginsdk.WorkflowInstanceCanceled}, nil
}
func (dataStoreE2EWorkflows) Transfer(context.Context, pluginsdk.WorkflowTargetActionInput) (pluginsdk.WorkflowInstance, error) {
	return pluginsdk.WorkflowInstance{ID: "instance-e2e"}, nil
}
func (dataStoreE2EWorkflows) Copy(context.Context, pluginsdk.WorkflowTargetActionInput) (pluginsdk.WorkflowInstance, error) {
	return pluginsdk.WorkflowInstance{ID: "instance-e2e"}, nil
}

type dataStoreE2EJobs struct{}

func (dataStoreE2EJobs) Schedule(context.Context, pluginsdk.JobScheduleInput) (pluginsdk.Job, error) {
	return pluginsdk.Job{ID: "job-e2e"}, nil
}
func (dataStoreE2EJobs) LeaseDue(context.Context, pluginsdk.JobLeaseInput) ([]pluginsdk.Job, error) {
	return []pluginsdk.Job{}, nil
}
func (dataStoreE2EJobs) Complete(context.Context, pluginsdk.JobCompleteInput) (pluginsdk.Job, error) {
	return pluginsdk.Job{ID: "job-e2e", Status: pluginsdk.JobStatusSucceeded}, nil
}
func (dataStoreE2EJobs) Fail(context.Context, pluginsdk.JobFailInput) (pluginsdk.Job, error) {
	return pluginsdk.Job{ID: "job-e2e", Status: pluginsdk.JobStatusRetryWait}, nil
}
func (dataStoreE2EJobs) Get(context.Context, string) (pluginsdk.Job, error) {
	return pluginsdk.Job{ID: "job-e2e"}, nil
}
func (dataStoreE2EJobs) List(context.Context, pluginsdk.JobQuery) ([]pluginsdk.Job, error) {
	return []pluginsdk.Job{}, nil
}

type dataStoreE2EResponse struct {
	status int
	body   []byte
}

func dataStoreE2ERequest(t *testing.T, method, target, token string, payload any) dataStoreE2EResponse {
	t.Helper()
	response, err := dataStoreE2EDo(method, target, token, payload)
	if err != nil {
		t.Fatalf("request %s %s: %v", method, target, err)
	}
	return response
}

func dataStoreE2EDo(method, target, token string, payload any) (dataStoreE2EResponse, error) {
	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return dataStoreE2EResponse{}, err
		}
		body = bytes.NewReader(raw)
	}
	request, err := http.NewRequest(method, target, body)
	if err != nil {
		return dataStoreE2EResponse{}, err
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	response, err := (&http.Client{Timeout: 3 * time.Second}).Do(request)
	if err != nil {
		return dataStoreE2EResponse{}, err
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(response.Body)
	if err != nil {
		return dataStoreE2EResponse{}, err
	}
	return dataStoreE2EResponse{status: response.StatusCode, body: raw}, nil
}

func buildDataStoreE2EBackend(t *testing.T, repositoryRoot, pluginDir, pluginID string) {
	t.Helper()
	entry := filepath.Join(pluginDir, filepath.FromSlash(dataStoreE2EBackendRelativePath(pluginID)))
	if err := os.MkdirAll(filepath.Dir(entry), 0o755); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "build", "-o", entry, "./plugins/"+pluginID+"/backend")
	command.Dir = repositoryRoot
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("build independent datastore backend: %v\n%s", err, output)
	}
}

func dataStoreE2EBackendRelativePath(pluginID string) string {
	name := pluginID + "-server"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.ToSlash(filepath.Join("backend", "bin", name))
}

func reserveDataStoreE2EAddress(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	_ = listener.Close()
	return address
}

func rewriteDataStoreE2EAddress(t *testing.T, path, address string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	updated := strings.ReplaceAll(string(raw), "127.0.0.1:18093", address)
	if updated == string(raw) {
		t.Fatal("datastore E2E service address was not rewritten")
	}
	if err = os.WriteFile(path, []byte(updated), 0o600); err != nil {
		t.Fatal(err)
	}
}

func copyDataStoreE2EDirectory(t *testing.T, source, destination string) {
	t.Helper()
	err := filepath.WalkDir(source, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, raw, 0o600)
	})
	if err != nil {
		t.Fatal(err)
	}
}

func signDataStoreE2EUser(t *testing.T, secret, subject, tenantID, organizationID string) string {
	t.Helper()
	token, err := security.SignJWT(secret, security.JWTIdentity{
		Subject: subject, OrganizationID: organizationID, OrganizationPath: []string{tenantID, organizationID},
		Role: "operator", Roles: []string{"operator"},
	}, time.Minute, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func waitDataStoreE2EUnavailable(t *testing.T, target string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		response, err := (&http.Client{Timeout: 100 * time.Millisecond}).Get(target)
		if response != nil {
			_ = response.Body.Close()
		}
		if err != nil {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("disabled plugin endpoint remained reachable: %s", target)
}

func dataStoreE2EPhysicalTable(t *testing.T, registry *datastore.SchemaRegistry, pluginID, logicalName string) string {
	t.Helper()
	snapshot, exists := registry.Snapshot(pluginID)
	if !exists {
		t.Fatalf("plugin schema is not active: %+v exists=%v", snapshot, exists)
	}
	for _, table := range snapshot.Tables {
		if table.LogicalName == logicalName {
			return table.PhysicalName
		}
	}
	t.Fatalf("plugin table %q is not active: %+v", logicalName, snapshot)
	return ""
}

func assertDataStoreE2EAggregate(t *testing.T, response dataStoreE2EResponse, expected map[string][]string) {
	t.Helper()
	if response.status != http.StatusOK {
		t.Fatalf("aggregate status=%d body=%s", response.status, response.body)
	}
	var page pluginsdk.DataAggregatePage
	if err := json.Unmarshal(response.body, &page); err != nil {
		t.Fatal(err)
	}
	if len(page.Rows) != len(expected) {
		t.Fatalf("aggregate rows=%+v", page.Rows)
	}
	for _, row := range page.Rows {
		status := row.Group["status"].Value
		values, exists := expected[status]
		if !exists || len(row.Values) != len(values) {
			t.Fatalf("unexpected aggregate row=%+v", row)
		}
		for index, value := range values {
			if row.Values[index].Value != value {
				t.Fatalf("aggregate row=%+v expected=%v", row, values)
			}
		}
	}
}

func cloneDataStoreE2EMap(values map[string]any) map[string]any {
	copy := make(map[string]any, len(values))
	for key, value := range values {
		copy[key] = value
	}
	return copy
}

var _ pluginsdk.FileService = dataStoreE2EFiles{}
var _ pluginsdk.ConfigService = (*dataStoreE2EConfig)(nil)
var _ pluginsdk.SecretService = (*dataStoreE2ESecrets)(nil)
var _ pluginsdk.WorkflowService = dataStoreE2EWorkflows{}
var _ pluginsdk.DocumentWorkflowService = dataStoreE2EDocuments{}
var _ pluginsdk.JobService = dataStoreE2EJobs{}
