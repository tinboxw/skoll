package sdkconformance

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

type Report struct {
	Transaction bool
	Scope       bool
	File        bool
	Audit       bool
	Config      bool
	Secret      bool
	Workflow    bool
	Job         bool
}

func Run(ctx context.Context, host pluginsdk.HostServices) (Report, error) {
	if err := host.Validate(); err != nil {
		return Report{}, err
	}
	report := Report{}
	if err := host.Transactions.Within(ctx, func(tx pluginsdk.Transaction) error {
		if tx.Context() == nil {
			return fmt.Errorf("transaction context is missing")
		}
		report.Transaction = true
		return nil
	}); err != nil {
		return report, fmt.Errorf("transaction contract: %w", err)
	}
	scope, err := host.DataScopes.Resolve(ctx, pluginsdk.Permission{Resource: "conformance.record", Action: "read"})
	if err != nil || scope.Denied() {
		return report, fmt.Errorf("scope contract: %w", err)
	}
	report.Scope = true
	file, err := host.Files.Store(ctx, pluginsdk.FileWrite{Key: "evidence/contract.txt", Name: "contract.txt", Content: []byte("sdk-conformance")})
	if err != nil {
		return report, fmt.Errorf("file store contract: %w", err)
	}
	if _, err = host.Files.Get(ctx, file.ID); err != nil {
		return report, fmt.Errorf("file get contract: %w", err)
	}
	report.File = true
	if _, err = host.Audit.Record(ctx, pluginsdk.AuditEntry{Action: "conformance.run", Resource: "contract", ResourceID: "sdk", Detail: map[string]any{"token": "must-redact"}}); err != nil {
		return report, fmt.Errorf("audit contract: %w", err)
	}
	report.Audit = true
	if _, err = host.Config.Replace(ctx, map[string]any{"mode": "strict"}); err != nil {
		return report, fmt.Errorf("config contract: %w", err)
	}
	config, err := host.Config.Get(ctx)
	if err != nil || config["mode"] != "strict" {
		return report, fmt.Errorf("config read contract: %w", err)
	}
	report.Config = true
	if err = host.Secrets.Set(ctx, "remote.api_key", "conformance-secret"); err != nil {
		return report, fmt.Errorf("secret set contract: %w", err)
	}
	secret, err := host.Secrets.Get(ctx, "remote.api_key")
	if err != nil || secret != "conformance-secret" {
		return report, fmt.Errorf("secret get contract: %w", err)
	}
	report.Secret = true
	definition, err := host.Workflows.CreateDefinition(ctx, pluginsdk.WorkflowDefinitionInput{
		ID: "approval", Key: "approval", Name: "SDK approval", Version: 1,
		Nodes: []pluginsdk.WorkflowNode{
			{ID: "start", Key: "start", Name: "Start", Type: pluginsdk.WorkflowNodeStart},
			{ID: "review", Key: "review", Name: "Review", Type: pluginsdk.WorkflowNodeApproval, AssigneeIDs: []string{"conformance-user"}},
			{ID: "end", Key: "end", Name: "End", Type: pluginsdk.WorkflowNodeEnd},
		},
		Transitions: []pluginsdk.WorkflowTransition{{From: "start", To: "review"}, {From: "review", To: "end"}},
	})
	if err != nil {
		return report, fmt.Errorf("workflow definition contract: %w", err)
	}
	if _, err = host.Workflows.PublishDefinition(ctx, definition.ID); err != nil {
		return report, fmt.Errorf("workflow publish contract: %w", err)
	}
	instance, err := host.Workflows.Start(ctx, pluginsdk.WorkflowStartInput{
		ID: "request", DefinitionID: definition.ID, BusinessType: "request", BusinessID: "request-1", Title: "SDK request",
	})
	if err != nil || len(instance.Tasks) != 1 {
		return report, fmt.Errorf("workflow start contract: %w", err)
	}
	instance, err = host.Workflows.Approve(ctx, pluginsdk.WorkflowTaskActionInput{InstanceID: instance.ID, TaskID: instance.Tasks[0].ID})
	if err != nil || instance.Status != pluginsdk.WorkflowInstanceApproved {
		return report, fmt.Errorf("workflow approve contract: %w", err)
	}
	report.Workflow = true
	job, err := host.Jobs.Schedule(ctx, pluginsdk.JobScheduleInput{
		ID: "sync", Kind: "sync", IdempotencyKey: "sync-1", Payload: json.RawMessage(`{"mode":"full"}`), MaxAttempts: 1,
	})
	if err != nil {
		return report, fmt.Errorf("job schedule contract: %w", err)
	}
	leased, err := host.Jobs.LeaseDue(ctx, pluginsdk.JobLeaseInput{WorkerID: "worker", Limit: 1, LeaseDuration: time.Minute})
	if err != nil || len(leased) != 1 || leased[0].ID != job.ID {
		return report, fmt.Errorf("job lease contract: %w", err)
	}
	completed, err := host.Jobs.Complete(ctx, pluginsdk.JobCompleteInput{JobID: job.ID, LeaseToken: leased[0].LeaseToken, Result: json.RawMessage(`{"ok":true}`)})
	if err != nil || completed.Status != pluginsdk.JobStatusSucceeded {
		return report, fmt.Errorf("job complete contract: %w", err)
	}
	report.Job = true
	return report, nil
}
