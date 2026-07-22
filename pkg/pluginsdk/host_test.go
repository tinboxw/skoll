package pluginsdk

import (
	"context"
	"testing"
)

type hostTestTransactions struct{}

func (hostTestTransactions) Within(_ context.Context, fn func(Transaction) error) error {
	return fn(hostTestTransaction{})
}

type hostTestTransaction struct{}

func (hostTestTransaction) Context() context.Context { return context.Background() }

type hostTestDataScopes struct{}

func (hostTestDataScopes) Resolve(_ context.Context, _ Permission) (ScopePredicate, error) {
	return NewScopePredicate(TrustedScope{
		SubjectID: "test", AllTenants: true, AllOwners: true, AllOrganizations: true,
	})
}

type hostTestDataStore struct{}

func (hostTestDataStore) Query(context.Context, DataQuery) (DataPage, error) {
	return DataPage{}, nil
}
func (hostTestDataStore) Mutate(context.Context, DataMutation) (DataMutationResult, error) {
	return DataMutationResult{}, nil
}

type hostTestFiles struct{}

func (hostTestFiles) Store(context.Context, FileWrite) (FileObject, error)  { return FileObject{}, nil }
func (hostTestFiles) List(context.Context, FileQuery) ([]FileObject, error) { return nil, nil }
func (hostTestFiles) Get(context.Context, string) (FileObject, error)       { return FileObject{}, nil }
func (hostTestFiles) Download(context.Context, string) (FileDownload, error) {
	return FileDownload{}, nil
}
func (hostTestFiles) Delete(context.Context, string) error { return nil }

type hostTestAudit struct{}

func (hostTestAudit) Record(context.Context, AuditEntry) (AuditReceipt, error) {
	return AuditReceipt{}, nil
}

type hostTestConfig struct{}

func (hostTestConfig) Get(context.Context) (map[string]any, error) { return map[string]any{}, nil }
func (hostTestConfig) Replace(_ context.Context, values map[string]any) (map[string]any, error) {
	return values, nil
}

type hostTestSecrets struct{}

func (hostTestSecrets) Get(context.Context, string) (string, error) { return "", nil }
func (hostTestSecrets) Set(context.Context, string, string) error   { return nil }

type hostTestWorkflows struct{}

func (hostTestWorkflows) CreateDefinition(context.Context, WorkflowDefinitionInput) (WorkflowDefinition, error) {
	return WorkflowDefinition{}, nil
}
func (hostTestWorkflows) GetDefinition(context.Context, string) (WorkflowDefinition, error) {
	return WorkflowDefinition{}, nil
}
func (hostTestWorkflows) PublishDefinition(context.Context, string) (WorkflowDefinition, error) {
	return WorkflowDefinition{}, nil
}
func (hostTestWorkflows) Start(context.Context, WorkflowStartInput) (WorkflowInstance, error) {
	return WorkflowInstance{}, nil
}
func (hostTestWorkflows) GetInstance(context.Context, string) (WorkflowInstance, error) {
	return WorkflowInstance{}, nil
}
func (hostTestWorkflows) Approve(context.Context, WorkflowTaskActionInput) (WorkflowInstance, error) {
	return WorkflowInstance{}, nil
}
func (hostTestWorkflows) Reject(context.Context, WorkflowTaskActionInput) (WorkflowInstance, error) {
	return WorkflowInstance{}, nil
}
func (hostTestWorkflows) Withdraw(context.Context, WorkflowInstanceActionInput) (WorkflowInstance, error) {
	return WorkflowInstance{}, nil
}
func (hostTestWorkflows) Transfer(context.Context, WorkflowTargetActionInput) (WorkflowInstance, error) {
	return WorkflowInstance{}, nil
}
func (hostTestWorkflows) Copy(context.Context, WorkflowTargetActionInput) (WorkflowInstance, error) {
	return WorkflowInstance{}, nil
}

type hostTestJobs struct{}

func (hostTestJobs) Schedule(context.Context, JobScheduleInput) (Job, error) { return Job{}, nil }
func (hostTestJobs) LeaseDue(context.Context, JobLeaseInput) ([]Job, error)  { return nil, nil }
func (hostTestJobs) Complete(context.Context, JobCompleteInput) (Job, error) { return Job{}, nil }
func (hostTestJobs) Fail(context.Context, JobFailInput) (Job, error)         { return Job{}, nil }
func (hostTestJobs) Get(context.Context, string) (Job, error)                { return Job{}, nil }
func (hostTestJobs) List(context.Context, JobQuery) ([]Job, error)           { return nil, nil }

func TestHostServicesValidateRequiresEveryPort(t *testing.T) {
	valid := HostServices{
		PluginID: "test", Transactions: hostTestTransactions{}, DataScopes: hostTestDataScopes{},
		DataStore: hostTestDataStore{},
		Files:     hostTestFiles{}, Audit: hostTestAudit{}, Config: hostTestConfig{}, Secrets: hostTestSecrets{},
		Workflows: hostTestWorkflows{}, Jobs: hostTestJobs{},
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid host services rejected: %v", err)
	}
	tests := []struct {
		name   string
		mutate func(*HostServices)
	}{
		{name: "plugin identity", mutate: func(host *HostServices) { host.PluginID = "" }},
		{name: "transactions", mutate: func(host *HostServices) { host.Transactions = nil }},
		{name: "data scopes", mutate: func(host *HostServices) { host.DataScopes = nil }},
		{name: "datastore", mutate: func(host *HostServices) { host.DataStore = nil }},
		{name: "files", mutate: func(host *HostServices) { host.Files = nil }},
		{name: "audit", mutate: func(host *HostServices) { host.Audit = nil }},
		{name: "config", mutate: func(host *HostServices) { host.Config = nil }},
		{name: "secrets", mutate: func(host *HostServices) { host.Secrets = nil }},
		{name: "workflows", mutate: func(host *HostServices) { host.Workflows = nil }},
		{name: "jobs", mutate: func(host *HostServices) { host.Jobs = nil }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			candidate := valid
			test.mutate(&candidate)
			if err := candidate.Validate(); err == nil {
				t.Fatalf("missing %s must be rejected", test.name)
			}
		})
	}
}
