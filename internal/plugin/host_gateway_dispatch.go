package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/tinboxw/skoll/pkg/pluginclient"
	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

type hostIDRequest struct {
	ID string `json:"id"`
}
type hostKeyRequest struct {
	Key string `json:"key"`
}
type hostSecretSetRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
type hostValueResponse struct {
	Value string `json:"value"`
}

func dispatchHostCall(ctx context.Context, host pluginsdk.HostServices, capability, operation string, decoder *json.Decoder) (any, error) {
	switch capability + "." + operation {
	case "datastore.query":
		var in pluginsdk.DataQuery
		if err := decodeHostInput(decoder, &in); err != nil {
			return nil, err
		}
		if err := in.Validate(); err != nil {
			return nil, err
		}
		return host.DataStore.Query(ctx, in)
	case "datastore.mutate":
		var in pluginsdk.DataMutation
		if err := decodeHostInput(decoder, &in); err != nil {
			return nil, err
		}
		if err := in.Validate(); err != nil {
			return nil, err
		}
		return host.DataStore.Mutate(ctx, in)
	case "document-numbers.preview":
		var in pluginsdk.DocumentNumberInput
		if err := decodeHostInput(decoder, &in); err != nil {
			return nil, err
		}
		if err := in.Validate(false); err != nil {
			return nil, err
		}
		return host.DocumentNumbers.Preview(ctx, in)
	case "document-numbers.issue":
		var in pluginsdk.DocumentNumberInput
		if err := decodeHostInput(decoder, &in); err != nil {
			return nil, err
		}
		if err := in.Validate(true); err != nil {
			return nil, err
		}
		return host.DocumentNumbers.Issue(ctx, in)
	case "documents.submit":
		var in pluginsdk.DocumentWorkflowSubmitInput
		if err := decodeHostInput(decoder, &in); err != nil {
			return nil, err
		}
		return host.Documents.Submit(ctx, in)
	case "documents.act":
		var in pluginsdk.DocumentWorkflowActionInput
		if err := decodeHostInput(decoder, &in); err != nil {
			return nil, err
		}
		return host.Documents.Act(ctx, in)
	case "documents.get":
		var in pluginsdk.DocumentWorkflowGetInput
		if err := decodeHostInput(decoder, &in); err != nil {
			return nil, err
		}
		return host.Documents.Get(ctx, in)
	case "scopes.resolve":
		var in pluginsdk.Permission
		if err := decodeHostInput(decoder, &in); err != nil {
			return nil, err
		}
		item, err := host.DataScopes.Resolve(ctx, in)
		if err != nil {
			return nil, err
		}
		return pluginclient.ScopeSnapshot{SubjectID: item.SubjectID(), TenantIDs: item.TenantIDs(), OwnerIDs: item.OwnerIDs(), OrganizationIDs: item.OrganizationIDs(), AllTenants: item.AllTenants(), AllOwners: item.AllOwners(), AllOrganizations: item.AllOrganizations(), Denied: item.Denied()}, nil
	case "files.store":
		var in pluginsdk.FileWrite
		if err := decodeHostInput(decoder, &in); err != nil {
			return nil, err
		}
		return host.Files.Store(ctx, in)
	case "files.list":
		var in pluginsdk.FileQuery
		if err := decodeHostInput(decoder, &in); err != nil {
			return nil, err
		}
		return host.Files.List(ctx, in)
	case "files.get":
		var in hostIDRequest
		if err := decodeHostInput(decoder, &in); err != nil {
			return nil, err
		}
		return host.Files.Get(ctx, in.ID)
	case "files.download":
		var in hostIDRequest
		if err := decodeHostInput(decoder, &in); err != nil {
			return nil, err
		}
		return host.Files.Download(ctx, in.ID)
	case "files.delete":
		var in hostIDRequest
		if err := decodeHostInput(decoder, &in); err != nil {
			return nil, err
		}
		return nil, host.Files.Delete(ctx, in.ID)
	case "audit.record":
		var in pluginsdk.AuditEntry
		if err := decodeHostInput(decoder, &in); err != nil {
			return nil, err
		}
		return host.Audit.Record(ctx, in)
	case "config.get":
		var in struct{}
		if err := decodeHostInput(decoder, &in); err != nil {
			return nil, err
		}
		return host.Config.Get(ctx)
	case "config.replace":
		var in map[string]any
		if err := decodeHostInput(decoder, &in); err != nil {
			return nil, err
		}
		return host.Config.Replace(ctx, in)
	case "secrets.get":
		var in hostKeyRequest
		if err := decodeHostInput(decoder, &in); err != nil {
			return nil, err
		}
		value, err := host.Secrets.Get(ctx, in.Key)
		return hostValueResponse{Value: value}, err
	case "secrets.set":
		var in hostSecretSetRequest
		if err := decodeHostInput(decoder, &in); err != nil {
			return nil, err
		}
		return nil, host.Secrets.Set(ctx, in.Key, in.Value)
	case "workflows.create-definition":
		var in pluginsdk.WorkflowDefinitionInput
		if err := decodeHostInput(decoder, &in); err != nil {
			return nil, err
		}
		return host.Workflows.CreateDefinition(ctx, in)
	case "workflows.get-definition":
		var in hostIDRequest
		if err := decodeHostInput(decoder, &in); err != nil {
			return nil, err
		}
		return host.Workflows.GetDefinition(ctx, in.ID)
	case "workflows.publish-definition":
		var in hostIDRequest
		if err := decodeHostInput(decoder, &in); err != nil {
			return nil, err
		}
		return host.Workflows.PublishDefinition(ctx, in.ID)
	case "workflows.start":
		var in pluginsdk.WorkflowStartInput
		if err := decodeHostInput(decoder, &in); err != nil {
			return nil, err
		}
		return host.Workflows.Start(ctx, in)
	case "workflows.get-instance":
		var in hostIDRequest
		if err := decodeHostInput(decoder, &in); err != nil {
			return nil, err
		}
		return host.Workflows.GetInstance(ctx, in.ID)
	case "workflows.approve":
		var in pluginsdk.WorkflowTaskActionInput
		if err := decodeHostInput(decoder, &in); err != nil {
			return nil, err
		}
		return host.Workflows.Approve(ctx, in)
	case "workflows.reject":
		var in pluginsdk.WorkflowTaskActionInput
		if err := decodeHostInput(decoder, &in); err != nil {
			return nil, err
		}
		return host.Workflows.Reject(ctx, in)
	case "workflows.withdraw":
		var in pluginsdk.WorkflowInstanceActionInput
		if err := decodeHostInput(decoder, &in); err != nil {
			return nil, err
		}
		return host.Workflows.Withdraw(ctx, in)
	case "workflows.cancel":
		var in pluginsdk.WorkflowInstanceActionInput
		if err := decodeHostInput(decoder, &in); err != nil {
			return nil, err
		}
		return host.Workflows.Cancel(ctx, in)
	case "workflows.transfer":
		var in pluginsdk.WorkflowTargetActionInput
		if err := decodeHostInput(decoder, &in); err != nil {
			return nil, err
		}
		return host.Workflows.Transfer(ctx, in)
	case "workflows.copy":
		var in pluginsdk.WorkflowTargetActionInput
		if err := decodeHostInput(decoder, &in); err != nil {
			return nil, err
		}
		return host.Workflows.Copy(ctx, in)
	case "jobs.schedule":
		var in pluginsdk.JobScheduleInput
		if err := decodeHostInput(decoder, &in); err != nil {
			return nil, err
		}
		return host.Jobs.Schedule(ctx, in)
	case "jobs.lease-due":
		var in pluginsdk.JobLeaseInput
		if err := decodeHostInput(decoder, &in); err != nil {
			return nil, err
		}
		return host.Jobs.LeaseDue(ctx, in)
	case "jobs.complete":
		var in pluginsdk.JobCompleteInput
		if err := decodeHostInput(decoder, &in); err != nil {
			return nil, err
		}
		return host.Jobs.Complete(ctx, in)
	case "jobs.fail":
		var in pluginsdk.JobFailInput
		if err := decodeHostInput(decoder, &in); err != nil {
			return nil, err
		}
		return host.Jobs.Fail(ctx, in)
	case "jobs.get":
		var in hostIDRequest
		if err := decodeHostInput(decoder, &in); err != nil {
			return nil, err
		}
		return host.Jobs.Get(ctx, in.ID)
	case "jobs.list":
		var in pluginsdk.JobQuery
		if err := decodeHostInput(decoder, &in); err != nil {
			return nil, err
		}
		return host.Jobs.List(ctx, in)
	default:
		return nil, fmt.Errorf("plugin host operation is not declared")
	}
}

func decodeHostInput(decoder *json.Decoder, target any) error {
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode plugin host request: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err == nil {
		return fmt.Errorf("decode plugin host request: trailing JSON value")
	} else if !errors.Is(err, io.EOF) {
		return fmt.Errorf("decode plugin host request: %w", err)
	}
	return nil
}
