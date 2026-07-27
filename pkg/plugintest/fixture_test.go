package plugintest

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	pluginruntime "github.com/tinboxw/skoll/internal/plugin"
	"github.com/tinboxw/skoll/pkg/pluginsdk"
	"github.com/tinboxw/skoll/pkg/security"
)

func TestServicesExposeDeterministicCurrentHostContract(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	clock := NewClock(now)
	identity := Identity{
		Subject: "user-42", TenantID: "tenant-7", OrganizationID: "org-3",
		Role: "operator", Roles: []string{"operator", "approver"},
	}
	services, err := NewServices(ServicesOptions{Clock: clock, Identity: identity})
	if err != nil {
		t.Fatal(err)
	}
	host, err := services.Host("fixture-plugin")
	if err != nil {
		t.Fatal(err)
	}
	if err := host.Validate(); err != nil {
		t.Fatalf("host contract: %v", err)
	}
	token, err := identity.Token(DefaultJWTSecret, time.Hour, now)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := security.ParseJWT(DefaultJWTSecret, token)
	if err != nil || claims.Subject != identity.Subject {
		t.Fatalf("identity token subject=%q error=%v", claims.Subject, err)
	}
	scope, err := host.DataScopes.Resolve(context.Background(), pluginsdk.Permission{Resource: "fixture.records", Action: "read"})
	if err != nil || !scope.Allows(pluginsdk.ScopedRecord{TenantID: identity.TenantID, OwnerID: identity.Subject, OrganizationID: identity.OrganizationID}) {
		t.Fatalf("identity scope=%+v error=%v", scope, err)
	}

	injected := errors.New("injected transaction failure")
	services.Failures.FailNext(OperationTransactionWithin, injected)
	if err := host.Transactions.Within(context.Background(), func(pluginsdk.Transaction) error { return nil }); !errors.Is(err, injected) {
		t.Fatalf("transaction failure=%v", err)
	}
	called := false
	if err := host.Transactions.Within(context.Background(), func(transaction pluginsdk.Transaction) error {
		called = transaction.Context() != nil
		return nil
	}); err != nil || !called {
		t.Fatalf("transaction recovery called=%v error=%v", called, err)
	}

	event, err := host.Events.Publish(context.Background(), pluginsdk.EventPublication{
		IdempotencyKey: "event-key", Name: "fixture.record.created", SchemaVersion: 1,
		Scope: pluginsdk.EventScope{TenantID: identity.TenantID}, CorrelationID: "correlation-1",
		Payload: pluginsdk.EventPayload{"id": {Type: pluginsdk.DataValueString, Value: "record-1"}},
	})
	if err != nil || event.ID != "fixture-event-1" || !event.OccurredAt.Equal(now) {
		t.Fatalf("event=%+v error=%v", event, err)
	}
	rule := pluginsdk.DocumentNumberRule{
		DocumentType: "purchase_request", Prefix: "PR", Separator: "-", Period: pluginsdk.DocumentNumberPeriodYear,
		Width: 4, Start: 1, GapPolicy: pluginsdk.DocumentNumberGapTransactional,
	}
	numberInput := pluginsdk.DocumentNumberInput{
		Rule: rule, TenantID: identity.TenantID, Permission: pluginsdk.Permission{Resource: "purchase_request", Action: "create"},
		OccurredAt: now, IdempotencyKey: "number-key",
	}
	first, err := host.DocumentNumbers.Issue(context.Background(), numberInput)
	if err != nil {
		t.Fatal(err)
	}
	duplicate, err := host.DocumentNumbers.Issue(context.Background(), numberInput)
	if err != nil || !duplicate.Duplicate || duplicate.Number != first.Number {
		t.Fatalf("duplicate number=%+v first=%+v error=%v", duplicate, first, err)
	}
	file, err := host.Files.Store(context.Background(), pluginsdk.FileWrite{
		Key: "evidence-1", Name: "evidence.txt", Content: []byte("proof"), Visibility: pluginsdk.FileVisibilityPrivate,
	})
	if err != nil || file.Hash == "" {
		t.Fatalf("file=%+v error=%v", file, err)
	}
	content, err := services.Files.Content(file.ID)
	if err != nil || string(content) != "proof" {
		t.Fatalf("file content=%q error=%v", content, err)
	}
	receipt, err := host.Audit.Record(context.Background(), pluginsdk.AuditEntry{Action: "fixture.create", Resource: "record", ResourceID: "record-1"})
	if err != nil || receipt.ID != "fixture-audit-1" || !receipt.OccurredAt.Equal(now) {
		t.Fatalf("audit=%+v error=%v", receipt, err)
	}

	definition, err := host.Workflows.CreateDefinition(context.Background(), pluginsdk.WorkflowDefinitionInput{
		ID: "approval-v1", Key: "approval", Name: "Approval", Version: 1,
		Nodes: []pluginsdk.WorkflowNode{
			{ID: "start", Key: "start", Name: "Start", Type: pluginsdk.WorkflowNodeStart},
			{ID: "approve", Key: "approve", Name: "Approve", Type: pluginsdk.WorkflowNodeApproval, AssigneeIDs: []string{"approver-1"}},
			{ID: "end", Key: "end", Name: "End", Type: pluginsdk.WorkflowNodeEnd},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := host.Workflows.PublishDefinition(context.Background(), definition.ID); err != nil {
		t.Fatal(err)
	}
	instance, err := host.Workflows.Start(context.Background(), pluginsdk.WorkflowStartInput{
		ID: "instance-1", DefinitionID: definition.ID, BusinessType: "request", BusinessID: "request-1", Title: "Request",
	})
	if err != nil || len(instance.Tasks) != 1 || instance.Tasks[0].Status != pluginsdk.WorkflowTaskPending {
		t.Fatalf("workflow instance=%+v error=%v", instance, err)
	}
	approved, err := host.Workflows.Approve(context.Background(), pluginsdk.WorkflowTaskActionInput{InstanceID: instance.ID, TaskID: instance.Tasks[0].ID})
	if err != nil || approved.Status != pluginsdk.WorkflowInstanceApproved {
		t.Fatalf("approved workflow=%+v error=%v", approved, err)
	}

	job, err := host.Jobs.Schedule(context.Background(), pluginsdk.JobScheduleInput{
		ID: "job-1", Kind: "fixture.scan", IdempotencyKey: "job-key", RunAt: now, MaxAttempts: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	leased, err := host.Jobs.LeaseDue(context.Background(), pluginsdk.JobLeaseInput{WorkerID: "worker-1", Limit: 1, LeaseDuration: time.Minute})
	if err != nil || len(leased) != 1 || leased[0].ID != job.ID {
		t.Fatalf("leased jobs=%+v error=%v", leased, err)
	}
	completed, err := host.Jobs.Complete(context.Background(), pluginsdk.JobCompleteInput{JobID: job.ID, LeaseToken: leased[0].LeaseToken})
	if err != nil || completed.Status != pluginsdk.JobStatusSucceeded {
		t.Fatalf("completed job=%+v error=%v", completed, err)
	}
	clock.Advance(time.Minute)
	if !services.Clock.Now().Equal(now.Add(time.Minute)) {
		t.Fatalf("clock=%s", services.Clock.Now())
	}
}

func TestPackageRunnerAndRuntimeLifecycle(t *testing.T) {
	const pluginID = "fixture-process"
	address, err := ReserveLoopbackAddress()
	if err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(t.TempDir(), pluginID)
	entry := filepath.Join(source, "backend", "bin", pluginID+"-server")
	if runtime.GOOS == "windows" {
		entry += ".exe"
	}
	if err := os.MkdirAll(filepath.Dir(entry), 0o755); err != nil {
		t.Fatal(err)
	}
	build := exec.Command("go", "build", "-o", entry, "./testdata/process")
	build.Dir = "."
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build fixture process: %v\n%s", err, output)
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(entry, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(source, "migrations"), 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := fmt.Sprintf(`id: %s
name: Fixture Process
version: 1.0.0
ui_mode: backend_only
service_base_url: http://%s
service_health_url: http://%s/health
data:
  namespace: fixture_process
  migration_version: v1.0.0
  migration_directory: migrations
  uninstall_policy: drop
  rollback_policy: automatic
`, pluginID, address, address)
	writeFixtureFile(t, filepath.Join(source, "plugin.yaml"), manifest)
	writeFixtureFile(t, filepath.Join(source, "migrations", "001_create.up.sql"), "fixture-create")
	writeFixtureFile(t, filepath.Join(source, "migrations", "001_create.down.sql"), "fixture-drop")

	installed, err := NewPackageRunner().BuildAndInstall(source, t.TempDir(), filepath.Join(t.TempDir(), "plugins"))
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateInstalledPackage(installed); err != nil {
		t.Fatal(err)
	}
	services, err := NewServices(ServicesOptions{
		Clock: NewClock(time.Now().UTC()), Identity: Identity{Subject: "runtime-user", TenantID: "runtime-tenant", OrganizationID: "runtime-org"},
	})
	if err != nil {
		t.Fatal(err)
	}
	fixtureRuntime, err := NewRuntime(RuntimeOptions{
		PluginID: pluginID, Services: services, DataRoot: filepath.Join(t.TempDir(), "data"),
		StartTimeout: 5 * time.Second, StopTimeout: 3 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = fixtureRuntime.Close(context.Background()) })
	if _, err := fixtureRuntime.Install(installed.Directory); err != nil {
		t.Fatal(err)
	}
	if err := fixtureRuntime.Enable(context.Background()); err != nil {
		t.Fatal(err)
	}
	assertFixtureProcess(t, "http://"+address+"/probe", pluginID, true)
	if snapshot, ok := fixtureRuntime.Service(); !ok || snapshot.State != pluginruntime.ServiceStateReady {
		t.Fatalf("enabled service snapshot=%+v ok=%v", snapshot, ok)
	}
	if err := fixtureRuntime.Restart(context.Background()); err != nil {
		t.Fatal(err)
	}
	assertFixtureProcess(t, "http://"+address+"/probe", pluginID, true)
	if err := fixtureRuntime.Disable(context.Background()); err != nil {
		t.Fatal(err)
	}
	assertFixtureProcess(t, "http://"+address+"/probe", pluginID, false)
	if err := fixtureRuntime.Enable(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := fixtureRuntime.Uninstall(context.Background()); err != nil {
		t.Fatal(err)
	}
	info, err := fixtureRuntime.Info()
	if err != nil || info.State != pluginruntime.StateUninstalled {
		t.Fatalf("uninstalled plugin state=%s error=%v", info.State, err)
	}
	assertFixtureProcess(t, "http://"+address+"/probe", pluginID, false)
}

func TestConcurrentAndPropertyHarnessesProveIdempotencyAndRecovery(t *testing.T) {
	const workers = 24
	now := time.Now().UTC().Truncate(time.Second)
	services, err := NewServices(ServicesOptions{
		Clock: NewClock(now),
		Identity: Identity{
			Subject: "property-user", TenantID: "property-tenant", OrganizationID: "property-org",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	host, err := services.Host("property-plugin")
	if err != nil {
		t.Fatal(err)
	}
	report, err := RunConcurrent(context.Background(), workers, func(ctx context.Context, _ int) (pluginsdk.Job, error) {
		return host.Jobs.Schedule(ctx, pluginsdk.JobScheduleInput{
			ID: "job-contention", Kind: "property.scan", IdempotencyKey: "shared-key", RunAt: now,
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := report.RequireSuccess(); err != nil {
		t.Fatal(err)
	}
	for _, job := range report.Values() {
		if job.ID != "job-contention" || job.IdempotencyKey != "shared-key" {
			t.Fatalf("contention produced divergent job: %+v", job)
		}
	}
	jobs, err := host.Jobs.List(context.Background(), pluginsdk.JobQuery{})
	if err != nil || len(jobs) != 1 {
		t.Fatalf("contention persisted jobs=%+v error=%v", jobs, err)
	}

	cases, err := GeneratePropertyCases(0x5c011, 100, func(generator *PropertyGenerator) pluginsdk.JobScheduleInput {
		return pluginsdk.JobScheduleInput{
			ID: generator.ID("job"), Kind: generator.Pick("qualification.scan", "inventory.reconcile", "approval.remind"),
			IdempotencyKey: generator.ID("key"), RunAt: now.Add(time.Duration(generator.IntN(60)) * time.Second),
			MaxAttempts: 1 + generator.IntN(5),
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := CheckProperty(cases, func(item PropertyCase[pluginsdk.JobScheduleInput]) error {
		first, scheduleErr := host.Jobs.Schedule(context.Background(), item.Value)
		if scheduleErr != nil {
			return scheduleErr
		}
		duplicate := item.Value
		duplicate.ID = duplicate.ID + "-duplicate"
		second, scheduleErr := host.Jobs.Schedule(context.Background(), duplicate)
		if scheduleErr != nil {
			return scheduleErr
		}
		if first.ID != second.ID || first.IdempotencyKey != second.IdempotencyKey {
			return fmt.Errorf("idempotent schedule diverged: first=%+v second=%+v", first, second)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	services.Failures.FailNext(OperationJobLeaseDue, errors.New("temporary lease failure"))
	if _, err := host.Jobs.LeaseDue(context.Background(), pluginsdk.JobLeaseInput{WorkerID: "worker-1", Limit: 10, LeaseDuration: time.Minute}); err == nil {
		t.Fatal("expected injected lease failure")
	}
	leased, err := host.Jobs.LeaseDue(context.Background(), pluginsdk.JobLeaseInput{WorkerID: "worker-1", Limit: 10, LeaseDuration: time.Minute})
	if err != nil || len(leased) == 0 {
		t.Fatalf("lease did not recover: jobs=%+v error=%v", leased, err)
	}
}

func TestTransactionalValueAndEventuallyProveRollbackAndReconciliation(t *testing.T) {
	var unconfigured TransactionalValue[map[string]int]
	if _, err := unconfigured.Snapshot(); err == nil {
		t.Fatal("unconfigured transaction snapshot succeeded")
	}

	state, err := NewTransactionalValue(map[string]int{}, func(input map[string]int) map[string]int {
		output := make(map[string]int, len(input))
		for key, value := range input {
			output[key] = value
		}
		return output
	})
	if err != nil {
		t.Fatal(err)
	}
	rejected := errors.New("reject mutation")
	if err := state.Within(context.Background(), func(value *map[string]int) error {
		(*value)["stock"] = 10
		return rejected
	}); !errors.Is(err, rejected) {
		t.Fatalf("rollback error=%v", err)
	}
	snapshot, err := state.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := snapshot["stock"]; exists {
		t.Fatal("failed transaction committed fixture state")
	}
	if err := state.Within(context.Background(), func(value *map[string]int) error {
		(*value)["stock"] = 10
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	attempts := 0
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := Eventually(ctx, time.Millisecond, func() (bool, error) {
		attempts++
		if attempts == 3 {
			snapshot, err := state.Snapshot()
			return snapshot["stock"] == 10, err
		}
		return false, nil
	}); err != nil {
		t.Fatal(err)
	}
	if attempts != 3 {
		t.Fatalf("reconciliation attempts=%d", attempts)
	}
}

func writeFixtureFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertFixtureProcess(t *testing.T, target, pluginID string, available bool) {
	t.Helper()
	client := &http.Client{Timeout: 200 * time.Millisecond}
	var lastErr error
	for attempt := 0; attempt < 30; attempt++ {
		response, err := client.Get(target)
		if err == nil {
			body := make([]byte, 256)
			count, _ := response.Body.Read(body)
			_ = response.Body.Close()
			if available && response.StatusCode == http.StatusOK && strings.Contains(string(body[:count]), pluginID) {
				return
			}
			if !available {
				lastErr = fmt.Errorf("service returned status %d", response.StatusCode)
			}
		} else {
			lastErr = err
			if !available {
				return
			}
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("fixture process available=%v: %v", available, lastErr)
}
