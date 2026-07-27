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
	"sync"
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
	domainworkflow "github.com/tinboxw/skoll/internal/domain/workflow"
	"github.com/tinboxw/skoll/internal/plugin"
	"github.com/tinboxw/skoll/internal/plugin/hostservice"
	jobsvc "github.com/tinboxw/skoll/internal/service/job"
	workflowsvc "github.com/tinboxw/skoll/internal/service/workflow"
	storesql "github.com/tinboxw/skoll/internal/store/sql"
	"github.com/tinboxw/skoll/internal/store/sql/gormrepo"
	"github.com/tinboxw/skoll/pkg/pluginsdk"
	"github.com/tinboxw/skoll/pkg/security"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	workflowE2EPluginID  = "workflow_e2e"
	workflowE2EJWTSecret = "workflow-e2e-jwt-secret"
	workflowE2EMeaning   = "I authorize the regulated quality decision"
)

type workflowE2ECall struct {
	Operation  string                              `json:"operation"`
	ID         string                              `json:"id,omitempty"`
	Definition pluginsdk.WorkflowDefinitionInput   `json:"definition,omitempty"`
	Start      pluginsdk.WorkflowStartInput        `json:"start,omitempty"`
	Task       pluginsdk.WorkflowTaskActionInput   `json:"task,omitempty"`
	Target     pluginsdk.WorkflowTargetActionInput `json:"target,omitempty"`
}

type workflowE2ERuntime struct {
	t         *testing.T
	dbPath    string
	pluginDir string
	address   string
	data      *plugin.PluginDataDirectories
	proofs    *security.ReverificationProofService
	clock     *workflowE2EClock
	audit     *dataStoreE2EAudit
	db        *gorm.DB
	service   workflowsvc.Service
	gateway   *plugin.HostGateway
	handle    plugin.ServiceHandle
}

type workflowE2EClock struct {
	mu  sync.RWMutex
	now time.Time
}

func (c *workflowE2EClock) Now() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.now
}

func (c *workflowE2EClock) Advance(duration time.Duration) {
	c.mu.Lock()
	c.now = c.now.Add(duration)
	c.mu.Unlock()
}

type workflowE2EEvidence struct{}

func (workflowE2EEvidence) ResolveWorkflowEvidence(
	_ context.Context,
	_ domainworkflow.Actor,
	fileIDs []shared.ID,
) ([]domainworkflow.EvidenceReference, error) {
	items := make([]domainworkflow.EvidenceReference, 0, len(fileIDs))
	for _, fileID := range fileIDs {
		items = append(items, domainworkflow.EvidenceReference{
			FileID: fileID, Name: "quality-certificate.pdf", Hash: "sha256:regulated-evidence",
			Size: 2048, MIME: "application/pdf",
		})
	}
	return items, nil
}

type workflowE2EDataStore struct{}

func (workflowE2EDataStore) Query(context.Context, pluginsdk.DataQuery) (pluginsdk.DataPage, error) {
	return pluginsdk.DataPage{}, errors.New("datastore is outside workflow acceptance")
}
func (workflowE2EDataStore) Mutate(context.Context, pluginsdk.DataMutation) (pluginsdk.DataMutationResult, error) {
	return pluginsdk.DataMutationResult{}, errors.New("datastore is outside workflow acceptance")
}
func (workflowE2EDataStore) Aggregate(context.Context, pluginsdk.DataAggregateQuery) (pluginsdk.DataAggregatePage, error) {
	return pluginsdk.DataAggregatePage{}, errors.New("datastore is outside workflow acceptance")
}

func TestIndependentPluginGovernedWorkflowProcessE2E(t *testing.T) {
	runtime := newWorkflowE2ERuntime(t)
	defer runtime.close()
	runtime.start()

	tokens := map[string]string{}
	for _, actorID := range []string{"starter-1", "quality-1", "quality-2", "quality-3", "delegate-1", "director-1", "intruder-1"} {
		tokens[actorID] = workflowE2EUserToken(t, actorID)
	}

	quorum := workflowE2EDefinition("quorum", []string{"quality-1", "quality-2", "quality-3"}, pluginsdk.WorkflowDecisionQuorum, 2, nil, nil)
	runtime.createAndPublish(tokens["starter-1"], quorum)
	quorumInstance := runtime.call(tokens["starter-1"], workflowE2ECall{
		Operation: "start",
		Start: pluginsdk.WorkflowStartInput{
			ID: "quorum-instance", DefinitionID: quorum.ID, BusinessType: "quality-release",
			BusinessID: "batch-quorum", Title: "Quorum quality release",
		},
	}, http.StatusOK)
	qualityTasks := workflowE2ETasksByActor(t, quorumInstance)

	const duplicateWorkers = 32
	var duplicateGroup sync.WaitGroup
	duplicateErrors := make(chan error, duplicateWorkers)
	for worker := 0; worker < duplicateWorkers; worker++ {
		actorID := "quality-1"
		if worker%2 == 1 {
			actorID = "quality-2"
		}
		duplicateGroup.Add(1)
		go func(actorID string) {
			defer duplicateGroup.Done()
			status, body, err := runtime.do(tokens[actorID], workflowE2ECall{
				Operation: "approve",
				Task: pluginsdk.WorkflowTaskActionInput{
					InstanceID: "quorum-instance", TaskID: qualityTasks[actorID], Comment: "parallel quorum approval",
				},
			})
			if err != nil {
				duplicateErrors <- err
				return
			}
			if status != http.StatusOK {
				duplicateErrors <- fmt.Errorf("parallel approval status=%d body=%s", status, body)
			}
		}(actorID)
	}
	duplicateGroup.Wait()
	close(duplicateErrors)
	for err := range duplicateErrors {
		t.Fatal(err)
	}
	quorumResult := runtime.get(tokens["starter-1"], "quorum-instance", http.StatusOK)
	workflowE2EAssertStatusAndActions(t, quorumResult, pluginsdk.WorkflowInstanceApproved, pluginsdk.WorkflowActionApprove, 2)
	runtime.call(tokens["quality-3"], workflowE2ECall{
		Operation: "approve",
		Task: pluginsdk.WorkflowTaskActionInput{
			InstanceID: "quorum-instance", TaskID: qualityTasks["quality-3"], Comment: "stale third decision",
		},
	}, http.StatusUnprocessableEntity)

	any := workflowE2EDefinition("delegation", []string{"quality-1"}, pluginsdk.WorkflowDecisionAny, 1, nil, nil)
	runtime.createAndPublish(tokens["starter-1"], any)
	delegation := runtime.call(tokens["starter-1"], workflowE2ECall{
		Operation: "start",
		Start: pluginsdk.WorkflowStartInput{
			ID: "delegation-instance", DefinitionID: any.ID, BusinessType: "controlled-change",
			BusinessID: "change-1", Title: "Controlled change",
		},
	}, http.StatusOK)
	originalTask := workflowE2ETasksByActor(t, delegation)["quality-1"]
	runtime.call(tokens["intruder-1"], workflowE2ECall{
		Operation: "approve",
		Task: pluginsdk.WorkflowTaskActionInput{
			InstanceID: "delegation-instance", TaskID: originalTask, Comment: "unauthorized approval",
		},
	}, http.StatusUnprocessableEntity)
	delegated := runtime.call(tokens["quality-1"], workflowE2ECall{
		Operation: "delegate",
		Target: pluginsdk.WorkflowTargetActionInput{
			InstanceID: "delegation-instance", TaskID: originalTask,
			Target:  pluginsdk.WorkflowActor{ID: "delegate-1", Name: "Delegated quality approver"},
			Comment: "planned delegation",
		},
	}, http.StatusOK)
	delegatedTask := workflowE2ETasksByActor(t, delegated)["delegate-1"]
	delegationResult := runtime.call(tokens["delegate-1"], workflowE2ECall{
		Operation: "approve",
		Task: pluginsdk.WorkflowTaskActionInput{
			InstanceID: "delegation-instance", TaskID: delegatedTask, Comment: "delegated approval",
		},
	}, http.StatusOK)
	workflowE2EAssertStatusAndActions(t, delegationResult, pluginsdk.WorkflowInstanceApproved, pluginsdk.WorkflowActionDelegate, 1)

	signature := workflowE2EDefinition(
		"signature", []string{"quality-1"}, pluginsdk.WorkflowDecisionAny, 1, nil,
		&pluginsdk.WorkflowSignaturePolicy{Meaning: workflowE2EMeaning, RequireEvidence: true},
	)
	runtime.createAndPublish(tokens["starter-1"], signature)
	signedStart := runtime.call(tokens["starter-1"], workflowE2ECall{
		Operation: "start",
		Start: pluginsdk.WorkflowStartInput{
			ID: "signed-instance", DefinitionID: signature.ID, BusinessType: "batch-release",
			BusinessID: "batch-signed", Title: "Signed batch release",
		},
	}, http.StatusOK)
	signedTask := workflowE2ETasksByActor(t, signedStart)["quality-1"]
	proof, _, err := runtime.proofs.IssueReverificationProof(
		"quality-1", "plugin:"+workflowE2EPluginID, security.ReverificationPurposeWorkflowSignature,
		"password", time.Now().UTC(), 2*time.Minute,
	)
	if err != nil {
		t.Fatal(err)
	}
	signed := runtime.call(tokens["quality-1"], workflowE2ECall{
		Operation: "approve",
		Task: pluginsdk.WorkflowTaskActionInput{
			InstanceID: "signed-instance", TaskID: signedTask, Comment: "release approved",
			Signature: &pluginsdk.WorkflowDecisionSignature{
				Proof: proof, Meaning: workflowE2EMeaning, EvidenceIDs: []string{"certificate-1"},
			},
		},
	}, http.StatusOK)
	receipts := workflowE2EArray(t, signed, "Receipts")
	if len(receipts) != 1 {
		t.Fatalf("signed workflow receipt count=%d body=%v", len(receipts), signed)
	}
	receipt := workflowE2EMap(t, receipts[0])
	if workflowE2EString(t, receipt, "EvidenceDigest") == "" ||
		workflowE2EString(t, receipt, "AuditCorrelationID") != workflowE2EString(t, receipt, "ID") {
		t.Fatalf("signature receipt is incomplete: %v", receipt)
	}
	reusedStart := runtime.call(tokens["starter-1"], workflowE2ECall{
		Operation: "start",
		Start: pluginsdk.WorkflowStartInput{
			ID: "reused-proof-instance", DefinitionID: signature.ID, BusinessType: "batch-release",
			BusinessID: "batch-reused", Title: "Reused proof probe",
		},
	}, http.StatusOK)
	runtime.call(tokens["quality-1"], workflowE2ECall{
		Operation: "approve",
		Task: pluginsdk.WorkflowTaskActionInput{
			InstanceID: "reused-proof-instance", TaskID: workflowE2ETasksByActor(t, reusedStart)["quality-1"],
			Signature: &pluginsdk.WorkflowDecisionSignature{
				Proof: proof, Meaning: workflowE2EMeaning, EvidenceIDs: []string{"certificate-1"},
			},
		},
	}, http.StatusUnprocessableEntity)

	escalationRule := &pluginsdk.WorkflowEscalationRule{
		AfterSeconds: 60, Target: pluginsdk.WorkflowActor{ID: "director-1", Name: "Quality director"},
	}
	escalation := workflowE2EDefinition("escalation", []string{"quality-1"}, pluginsdk.WorkflowDecisionAny, 1, escalationRule, nil)
	runtime.createAndPublish(tokens["starter-1"], escalation)
	escalationStart := runtime.call(tokens["starter-1"], workflowE2ECall{
		Operation: "start",
		Start: pluginsdk.WorkflowStartInput{
			ID: "escalation-instance", DefinitionID: escalation.ID, BusinessType: "deviation",
			BusinessID: "deviation-1", Title: "Timed deviation approval",
		},
	}, http.StatusOK)
	expiredTask := workflowE2ETasksByActor(t, escalationStart)["quality-1"]
	runtime.clock.Advance(2 * time.Minute)
	var timerRow gormrepo.JobModel
	if err := runtime.db.Where("namespace = ? AND kind = ?", "system.workflow", "approval-escalation").Order("created_at DESC").First(&timerRow).Error; err != nil {
		t.Fatalf("load persisted escalation timer: %v", err)
	}
	if timerRow.RunAt.After(runtime.clock.Now()) {
		t.Fatalf("escalation timer is not due: runAt=%s clock=%s", timerRow.RunAt, runtime.clock.Now())
	}
	var timerGroup sync.WaitGroup
	timerErrors := make(chan error, 16)
	timerCompleted := make(chan int, 16)
	for worker := 0; worker < 16; worker++ {
		timerGroup.Add(1)
		go func(worker int) {
			defer timerGroup.Done()
			completed, processErr := runtime.service.ProcessDueTimers(context.Background(), fmt.Sprintf("workflow-e2e-%d", worker), 10, time.Minute)
			if processErr != nil {
				timerErrors <- processErr
			}
			timerCompleted <- completed
		}(worker)
	}
	timerGroup.Wait()
	close(timerErrors)
	close(timerCompleted)
	for err := range timerErrors {
		t.Fatal(err)
	}
	completedTotal := 0
	for completed := range timerCompleted {
		completedTotal += completed
	}
	if completedTotal != 1 {
		t.Fatalf("escalation timer completed=%d want=1 row=%+v", completedTotal, timerRow)
	}
	escalated := runtime.get(tokens["starter-1"], "escalation-instance", http.StatusOK)
	workflowE2EAssertStatusAndActions(t, escalated, pluginsdk.WorkflowInstanceRunning, pluginsdk.WorkflowActionEscalate, 1)
	runtime.call(tokens["quality-1"], workflowE2ECall{
		Operation: "approve",
		Task: pluginsdk.WorkflowTaskActionInput{
			InstanceID: "escalation-instance", TaskID: expiredTask, Comment: "expired approval",
		},
	}, http.StatusUnprocessableEntity)
	escalatedTask := workflowE2ETasksByActor(t, escalated)["director-1"]
	runtime.call(tokens["director-1"], workflowE2ECall{
		Operation: "approve",
		Task: pluginsdk.WorkflowTaskActionInput{
			InstanceID: "escalation-instance", TaskID: escalatedTask, Comment: "director approval",
		},
	}, http.StatusOK)

	runtime.restart()
	restarted := runtime.get(tokens["starter-1"], "signed-instance", http.StatusOK)
	if len(workflowE2EArray(t, restarted, "Receipts")) != 1 {
		t.Fatalf("restart lost immutable signature evidence: %v", restarted)
	}
	if err := runtime.db.Model(&gormrepo.WorkflowActionModel{}).
		Where("instance_id = ? AND action_type = ?", "plugin:"+workflowE2EPluginID+":signed-instance", string(domainworkflow.ActionApprove)).
		Update("comment", "tampered after signing").Error; err != nil {
		t.Fatal(err)
	}
	runtime.get(tokens["starter-1"], "signed-instance", http.StatusUnprocessableEntity)
}

func newWorkflowE2ERuntime(t *testing.T) *workflowE2ERuntime {
	t.Helper()
	repositoryRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	pluginDir := filepath.Join(t.TempDir(), workflowE2EPluginID)
	if err := os.MkdirAll(filepath.Join(pluginDir, "backend", "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	entry := filepath.Join(pluginDir, "backend", "bin", workflowE2EPluginID+"-server")
	if runtime.GOOS == "windows" {
		entry += ".exe"
	}
	command := exec.Command("go", "build", "-o", entry, "./internal/bootstrap/testdata/workflow_e2e")
	command.Dir = repositoryRoot
	if output, buildErr := command.CombinedOutput(); buildErr != nil {
		t.Fatalf("build workflow acceptance process: %v\n%s", buildErr, output)
	}
	data, err := plugin.NewPluginDataDirectories(filepath.Join(t.TempDir(), "plugin-data"))
	if err != nil {
		t.Fatal(err)
	}
	proofs, err := security.NewReverificationProofService("workflow-e2e-proof-secret")
	if err != nil {
		t.Fatal(err)
	}
	return &workflowE2ERuntime{
		t: t, dbPath: filepath.ToSlash(filepath.Join(t.TempDir(), "workflow-e2e.db")),
		pluginDir: pluginDir, address: workflowE2EReserveAddress(t), data: data,
		proofs: proofs, clock: &workflowE2EClock{now: time.Now().UTC()}, audit: &dataStoreE2EAudit{},
	}
}

func (r *workflowE2ERuntime) start() {
	r.t.Helper()
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?_busy_timeout=5000&_journal_mode=WAL", r.dbPath)), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		r.t.Fatal(err)
	}
	if err = db.AutoMigrate(gormrepo.AllModels()...); err != nil {
		r.t.Fatal(err)
	}
	r.db = db
	jobs := jobsvc.NewService(gormrepo.NewJobStore(db), r.clock.Now)
	r.service = workflowsvc.NewService(gormrepo.NewWorkflowStore(db), workflowsvc.Options{
		Jobs: jobs, UnitOfWork: storesql.NewUnitOfWorkWithDB(db), Now: r.clock.Now,
		Proofs: r.proofs, Evidence: workflowE2EEvidence{},
	})
	transactions, err := hostservice.NewTransactionService(storesql.NewUnitOfWorkWithDB(db))
	if err != nil {
		r.t.Fatal(err)
	}
	r.gateway, err = plugin.NewHostGateway(func(pluginID string) (pluginsdk.HostServices, error) {
		if pluginID != workflowE2EPluginID {
			return pluginsdk.HostServices{}, errors.New("unexpected workflow acceptance plugin identity")
		}
		workflows, workflowErr := hostservice.NewWorkflowService(pluginID, r.service, r.audit)
		if workflowErr != nil {
			return pluginsdk.HostServices{}, workflowErr
		}
		return pluginsdk.HostServices{
			PluginID: pluginID, Transactions: transactions, DataScopes: dataStoreE2EScopes{},
			DataStore: workflowE2EDataStore{}, Events: dataStoreE2EEvents{},
			DocumentNumbers: dataStoreE2EDocumentNumbers{}, Documents: dataStoreE2EDocuments{},
			Files: dataStoreE2EFiles{}, Audit: r.audit, Config: &dataStoreE2EConfig{},
			Secrets: &dataStoreE2ESecrets{}, Workflows: workflows, Jobs: dataStoreE2EJobs{},
		}, nil
	}, workflowE2EJWTSecret, 5*time.Second)
	if err != nil {
		r.t.Fatal(err)
	}
	launcher := plugin.NewManagedProcessLauncher(plugin.NewHTTPHealthChecker(time.Second), 25*time.Millisecond, r.gateway, r.data)
	r.handle, err = launcher.Start(context.Background(), plugin.Info{
		ID: workflowE2EPluginID, Name: "Workflow E2E", Version: "1.0.0", State: plugin.StateEnabled,
		Source: r.pluginDir, ServiceBaseURL: "http://" + r.address,
		ServiceHealthURL: "http://" + r.address + "/health",
		DataManifest: &plugin.DataManifest{
			Namespace: workflowE2EPluginID, UninstallPolicy: plugin.DataUninstallRetain, RollbackPolicy: plugin.DataRollbackManual,
		},
	})
	if err != nil {
		r.t.Fatal(err)
	}
}

func (r *workflowE2ERuntime) stop() {
	r.t.Helper()
	if r.handle != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := r.handle.Stop(ctx); err != nil {
			cancel()
			r.t.Fatal(err)
		}
		cancel()
		r.handle = nil
	}
	if r.gateway != nil {
		if err := r.gateway.Close(); err != nil {
			r.t.Fatal(err)
		}
		r.gateway = nil
	}
	if r.db != nil {
		sqlDB, err := r.db.DB()
		if err != nil {
			r.t.Fatal(err)
		}
		if err = sqlDB.Close(); err != nil {
			r.t.Fatal(err)
		}
		r.db = nil
	}
}

func (r *workflowE2ERuntime) restart() {
	r.stop()
	r.start()
}

func (r *workflowE2ERuntime) close() {
	if r.handle != nil || r.gateway != nil || r.db != nil {
		r.stop()
	}
}

func (r *workflowE2ERuntime) createAndPublish(token string, definition pluginsdk.WorkflowDefinitionInput) {
	r.call(token, workflowE2ECall{Operation: "create-definition", Definition: definition}, http.StatusOK)
	r.call(token, workflowE2ECall{Operation: "publish-definition", ID: definition.ID}, http.StatusOK)
}

func (r *workflowE2ERuntime) get(token, id string, wantStatus int) map[string]any {
	return r.call(token, workflowE2ECall{Operation: "get-instance", ID: id}, wantStatus)
}

func (r *workflowE2ERuntime) call(token string, call workflowE2ECall, wantStatus int) map[string]any {
	r.t.Helper()
	status, body, err := r.do(token, call)
	if err != nil {
		r.t.Fatal(err)
	}
	if status != wantStatus {
		r.t.Fatalf("workflow operation %s status=%d want=%d body=%s", call.Operation, status, wantStatus, body)
	}
	var result map[string]any
	if err = json.Unmarshal(body, &result); err != nil {
		r.t.Fatalf("decode workflow operation %s: %v body=%s", call.Operation, err, body)
	}
	return result
}

func (r *workflowE2ERuntime) do(token string, call workflowE2ECall) (int, []byte, error) {
	raw, err := json.Marshal(call)
	if err != nil {
		return 0, nil, err
	}
	request, err := http.NewRequest(http.MethodPost, "http://"+r.address+"/v1/workflows", bytes.NewReader(raw))
	if err != nil {
		return 0, nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+token)
	response, err := (&http.Client{Timeout: 10 * time.Second}).Do(request)
	if err != nil {
		return 0, nil, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	return response.StatusCode, body, err
}

func workflowE2EDefinition(
	id string,
	assignees []string,
	strategy pluginsdk.WorkflowDecisionStrategy,
	quorum int,
	escalation *pluginsdk.WorkflowEscalationRule,
	signature *pluginsdk.WorkflowSignaturePolicy,
) pluginsdk.WorkflowDefinitionInput {
	return pluginsdk.WorkflowDefinitionInput{
		ID: id, Key: id, Name: id, Version: 1,
		Nodes: []pluginsdk.WorkflowNode{
			{ID: "start", Key: "start", Name: "Start", Type: pluginsdk.WorkflowNodeStart},
			{
				ID: "approval", Key: "approval", Name: "Approval", Type: pluginsdk.WorkflowNodeApproval,
				AssigneeIDs: assignees, Decision: &pluginsdk.WorkflowDecisionRule{Strategy: strategy, Quorum: quorum},
				Escalation: escalation, Signature: signature,
			},
			{ID: "end", Key: "end", Name: "End", Type: pluginsdk.WorkflowNodeEnd},
		},
		Transitions: []pluginsdk.WorkflowTransition{{From: "start", To: "approval"}, {From: "approval", To: "end"}},
	}
}

func workflowE2ETasksByActor(t *testing.T, instance map[string]any) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, raw := range workflowE2EArray(t, instance, "Tasks") {
		task := workflowE2EMap(t, raw)
		if workflowE2EString(t, task, "Status") != string(pluginsdk.WorkflowTaskPending) {
			continue
		}
		actor := workflowE2EMap(t, task["Assignee"])
		out[workflowE2EString(t, actor, "ID")] = workflowE2EString(t, task, "ID")
	}
	return out
}

func workflowE2EAssertStatusAndActions(
	t *testing.T,
	instance map[string]any,
	status pluginsdk.WorkflowInstanceStatus,
	action pluginsdk.WorkflowActionType,
	want int,
) {
	t.Helper()
	if workflowE2EString(t, instance, "Status") != string(status) {
		t.Fatalf("workflow status=%v want=%s", instance["Status"], status)
	}
	count := 0
	for _, raw := range workflowE2EArray(t, instance, "Timeline") {
		if workflowE2EString(t, workflowE2EMap(t, raw), "Type") == string(action) {
			count++
		}
	}
	if count != want {
		t.Fatalf("workflow action %s count=%d want=%d timeline=%v", action, count, want, instance["Timeline"])
	}
}

func workflowE2EMap(t *testing.T, value any) map[string]any {
	t.Helper()
	out, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("value is not an object: %T %v", value, value)
	}
	return out
}

func workflowE2EArray(t *testing.T, value map[string]any, field string) []any {
	t.Helper()
	out, ok := value[field].([]any)
	if !ok {
		t.Fatalf("%s is not an array: %T %v", field, value[field], value[field])
	}
	return out
}

func workflowE2EString(t *testing.T, value map[string]any, field string) string {
	t.Helper()
	out, ok := value[field].(string)
	if !ok {
		t.Fatalf("%s is not a string: %T %v", field, value[field], value[field])
	}
	return out
}

func workflowE2EUserToken(t *testing.T, subject string) string {
	t.Helper()
	token, err := security.SignJWT(workflowE2EJWTSecret, security.JWTIdentity{
		Subject: subject, OrganizationID: "org-a", OrganizationPath: []string{"tenant-a", "org-a"},
		Role: "workflow-operator", Roles: []string{"workflow-operator"},
	}, 30*time.Minute, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func workflowE2EReserveAddress(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	if err = listener.Close(); err != nil {
		t.Fatal(err)
	}
	return address
}
