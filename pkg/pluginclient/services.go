package pluginclient

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

type transaction struct{ ctx context.Context }

func (t transaction) Context() context.Context { return t.ctx }

type transactionService struct{ client *Client }

func (s transactionService) Within(ctx context.Context, fn func(pluginsdk.Transaction) error) error {
	if fn == nil {
		return errors.New("plugin transaction callback is required")
	}
	var started TransactionStartResponse
	if err := s.client.call(ctx, "transactions", "start", struct{}{}, &started); err != nil {
		return err
	}
	if strings.TrimSpace(started.ID) == "" {
		return errors.New("plugin host returned an empty transaction identity")
	}
	txCtx := context.WithValue(contextOrBackground(ctx), transactionContextKey{}, started.ID)
	callbackErr := fn(transaction{ctx: txCtx})
	finishBase, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	finishCtx := context.WithValue(finishBase, transactionContextKey{}, started.ID)
	finishErr := s.client.call(finishCtx, "transactions", "finish", TransactionFinishRequest{Commit: callbackErr == nil}, nil)
	return errors.Join(callbackErr, finishErr)
}

type dataScopeService struct{ client *Client }

func (s dataScopeService) Resolve(ctx context.Context, permission pluginsdk.Permission) (pluginsdk.ScopePredicate, error) {
	var snapshot ScopeSnapshot
	if err := s.client.call(ctx, "scopes", "resolve", permission, &snapshot); err != nil {
		return pluginsdk.ScopePredicate{}, err
	}
	if snapshot.Denied {
		return pluginsdk.NewDeniedScopePredicate(snapshot.SubjectID), nil
	}
	return pluginsdk.NewScopePredicate(pluginsdk.TrustedScope{
		SubjectID: snapshot.SubjectID, TenantIDs: snapshot.TenantIDs, OwnerIDs: snapshot.OwnerIDs,
		OrganizationIDs: snapshot.OrganizationIDs, AllTenants: snapshot.AllTenants,
		AllOwners: snapshot.AllOwners, AllOrganizations: snapshot.AllOrganizations,
	})
}

type dataStoreService struct{ client *Client }

func (s dataStoreService) Query(ctx context.Context, query pluginsdk.DataQuery) (out pluginsdk.DataPage, err error) {
	if err = query.Validate(); err != nil {
		return out, err
	}
	err = s.client.call(ctx, "datastore", "query", query, &out)
	if err == nil {
		if validateErr := validateDataPageResponse(query, out); validateErr != nil {
			err = pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorUnavailable, "response", "plugin host returned an invalid data page", true)
		}
	}
	return
}

func (s dataStoreService) Mutate(ctx context.Context, mutation pluginsdk.DataMutation) (out pluginsdk.DataMutationResult, err error) {
	if err = mutation.Validate(); err != nil {
		return out, err
	}
	err = s.client.call(ctx, "datastore", "mutate", mutation, &out)
	if err == nil {
		if validateErr := validateMutationResponse(mutation, out); validateErr != nil {
			err = pluginsdk.NewDataStoreError(pluginsdk.DataStoreErrorUnavailable, "response", "plugin host returned an invalid mutation result", true)
		}
	}
	return
}

func validateDataPageResponse(query pluginsdk.DataQuery, page pluginsdk.DataPage) error {
	if err := page.Validate(); err != nil {
		return err
	}
	expected := identifierSet(query.Fields)
	for _, record := range page.Records {
		if !sameIdentifiers(expected, record.Values) {
			return errors.New("data page record fields do not match the query")
		}
	}
	return nil
}

func validateMutationResponse(mutation pluginsdk.DataMutation, result pluginsdk.DataMutationResult) error {
	if err := result.Validate(); err != nil {
		return err
	}
	if len(mutation.Returning) == 0 {
		if result.Record != nil {
			return errors.New("mutation returned an undeclared record")
		}
		return nil
	}
	if result.RowsAffected == 1 && (result.Record == nil || !sameIdentifiers(identifierSet(mutation.Returning), result.Record.Values)) {
		return errors.New("mutation record fields do not match returning")
	}
	return nil
}

func identifierSet(fields []string) map[string]struct{} {
	set := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		set[field] = struct{}{}
	}
	return set
}

func sameIdentifiers(expected map[string]struct{}, values map[string]pluginsdk.DataValue) bool {
	if len(expected) != len(values) {
		return false
	}
	for field := range values {
		if _, exists := expected[field]; !exists {
			return false
		}
	}
	return true
}

type fileService struct{ client *Client }

func (s fileService) Store(ctx context.Context, in pluginsdk.FileWrite) (out pluginsdk.FileObject, err error) {
	err = s.client.call(ctx, "files", "store", in, &out)
	return
}
func (s fileService) List(ctx context.Context, in pluginsdk.FileQuery) (out []pluginsdk.FileObject, err error) {
	err = s.client.call(ctx, "files", "list", in, &out)
	return
}
func (s fileService) Get(ctx context.Context, id string) (out pluginsdk.FileObject, err error) {
	err = s.client.call(ctx, "files", "get", idRequest{ID: id}, &out)
	return
}
func (s fileService) Download(ctx context.Context, id string) (out pluginsdk.FileDownload, err error) {
	err = s.client.call(ctx, "files", "download", idRequest{ID: id}, &out)
	return
}
func (s fileService) Delete(ctx context.Context, id string) error {
	return s.client.call(ctx, "files", "delete", idRequest{ID: id}, nil)
}

type auditService struct{ client *Client }

func (s auditService) Record(ctx context.Context, in pluginsdk.AuditEntry) (out pluginsdk.AuditReceipt, err error) {
	err = s.client.call(ctx, "audit", "record", in, &out)
	return
}

type configService struct{ client *Client }

func (s configService) Get(ctx context.Context) (out map[string]any, err error) {
	err = s.client.call(ctx, "config", "get", struct{}{}, &out)
	return
}
func (s configService) Replace(ctx context.Context, in map[string]any) (out map[string]any, err error) {
	err = s.client.call(ctx, "config", "replace", in, &out)
	return
}

type secretService struct{ client *Client }

func (s secretService) Get(ctx context.Context, key string) (out string, err error) {
	var r valueResponse
	err = s.client.call(ctx, "secrets", "get", keyRequest{Key: key}, &r)
	return r.Value, err
}
func (s secretService) Set(ctx context.Context, key, value string) error {
	return s.client.call(ctx, "secrets", "set", secretSetRequest{Key: key, Value: value}, nil)
}

type workflowService struct{ client *Client }

func (s workflowService) CreateDefinition(ctx context.Context, in pluginsdk.WorkflowDefinitionInput) (out pluginsdk.WorkflowDefinition, err error) {
	err = s.client.call(ctx, "workflows", "create-definition", in, &out)
	return
}
func (s workflowService) GetDefinition(ctx context.Context, id string) (out pluginsdk.WorkflowDefinition, err error) {
	err = s.client.call(ctx, "workflows", "get-definition", idRequest{ID: id}, &out)
	return
}
func (s workflowService) PublishDefinition(ctx context.Context, id string) (out pluginsdk.WorkflowDefinition, err error) {
	err = s.client.call(ctx, "workflows", "publish-definition", idRequest{ID: id}, &out)
	return
}
func (s workflowService) Start(ctx context.Context, in pluginsdk.WorkflowStartInput) (out pluginsdk.WorkflowInstance, err error) {
	err = s.client.call(ctx, "workflows", "start", in, &out)
	return
}
func (s workflowService) GetInstance(ctx context.Context, id string) (out pluginsdk.WorkflowInstance, err error) {
	err = s.client.call(ctx, "workflows", "get-instance", idRequest{ID: id}, &out)
	return
}
func (s workflowService) Approve(ctx context.Context, in pluginsdk.WorkflowTaskActionInput) (out pluginsdk.WorkflowInstance, err error) {
	err = s.client.call(ctx, "workflows", "approve", in, &out)
	return
}
func (s workflowService) Reject(ctx context.Context, in pluginsdk.WorkflowTaskActionInput) (out pluginsdk.WorkflowInstance, err error) {
	err = s.client.call(ctx, "workflows", "reject", in, &out)
	return
}
func (s workflowService) Withdraw(ctx context.Context, in pluginsdk.WorkflowInstanceActionInput) (out pluginsdk.WorkflowInstance, err error) {
	err = s.client.call(ctx, "workflows", "withdraw", in, &out)
	return
}
func (s workflowService) Transfer(ctx context.Context, in pluginsdk.WorkflowTargetActionInput) (out pluginsdk.WorkflowInstance, err error) {
	err = s.client.call(ctx, "workflows", "transfer", in, &out)
	return
}
func (s workflowService) Copy(ctx context.Context, in pluginsdk.WorkflowTargetActionInput) (out pluginsdk.WorkflowInstance, err error) {
	err = s.client.call(ctx, "workflows", "copy", in, &out)
	return
}

type jobService struct{ client *Client }

func (s jobService) Schedule(ctx context.Context, in pluginsdk.JobScheduleInput) (out pluginsdk.Job, err error) {
	err = s.client.call(ctx, "jobs", "schedule", in, &out)
	return
}
func (s jobService) LeaseDue(ctx context.Context, in pluginsdk.JobLeaseInput) (out []pluginsdk.Job, err error) {
	err = s.client.call(ctx, "jobs", "lease-due", in, &out)
	return
}
func (s jobService) Complete(ctx context.Context, in pluginsdk.JobCompleteInput) (out pluginsdk.Job, err error) {
	err = s.client.call(ctx, "jobs", "complete", in, &out)
	return
}
func (s jobService) Fail(ctx context.Context, in pluginsdk.JobFailInput) (out pluginsdk.Job, err error) {
	err = s.client.call(ctx, "jobs", "fail", in, &out)
	return
}
func (s jobService) Get(ctx context.Context, id string) (out pluginsdk.Job, err error) {
	err = s.client.call(ctx, "jobs", "get", idRequest{ID: id}, &out)
	return
}
func (s jobService) List(ctx context.Context, in pluginsdk.JobQuery) (out []pluginsdk.Job, err error) {
	err = s.client.call(ctx, "jobs", "list", in, &out)
	return
}

type idRequest struct {
	ID string `json:"id"`
}
type keyRequest struct {
	Key string `json:"key"`
}
type secretSetRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
type valueResponse struct {
	Value string `json:"value"`
}

var _ pluginsdk.TransactionService = transactionService{}
var _ pluginsdk.DataScopeService = dataScopeService{}
var _ pluginsdk.DataStoreService = dataStoreService{}
var _ pluginsdk.FileService = fileService{}
var _ pluginsdk.AuditService = auditService{}
var _ pluginsdk.ConfigService = configService{}
var _ pluginsdk.SecretService = secretService{}
var _ pluginsdk.WorkflowService = workflowService{}
var _ pluginsdk.JobService = jobService{}
