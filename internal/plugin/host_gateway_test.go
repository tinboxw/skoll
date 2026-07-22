package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tinboxw/skoll/pkg/pluginclient"
	"github.com/tinboxw/skoll/pkg/pluginsdk"
	"github.com/tinboxw/skoll/pkg/security"
)

type gatewayTxMarker struct{}

type gatewayTransactions struct {
	mu        sync.Mutex
	commits   int
	rollbacks int
}

func (s *gatewayTransactions) Within(ctx context.Context, fn func(pluginsdk.Transaction) error) error {
	err := fn(gatewayTransaction{ctx: context.WithValue(ctx, gatewayTxMarker{}, true)})
	s.mu.Lock()
	defer s.mu.Unlock()
	if err != nil {
		s.rollbacks++
	} else {
		s.commits++
	}
	return err
}

type gatewayTransaction struct{ ctx context.Context }

func (t gatewayTransaction) Context() context.Context { return t.ctx }

type gatewayScopes struct{}

func (gatewayScopes) Resolve(ctx context.Context, _ pluginsdk.Permission) (pluginsdk.ScopePredicate, error) {
	claims, ok := security.JWTClaimsFromContext(ctx)
	if !ok {
		return pluginsdk.ScopePredicate{}, errors.New("user claims required")
	}
	return pluginsdk.NewScopePredicate(pluginsdk.TrustedScope{SubjectID: claims.Subject, AllTenants: true, AllOwners: true, AllOrganizations: true})
}

type gatewayFiles struct{}

func (gatewayFiles) Store(context.Context, pluginsdk.FileWrite) (pluginsdk.FileObject, error) {
	return pluginsdk.FileObject{ID: "file-1"}, nil
}
func (gatewayFiles) List(context.Context, pluginsdk.FileQuery) ([]pluginsdk.FileObject, error) {
	return []pluginsdk.FileObject{{ID: "file-1"}}, nil
}
func (gatewayFiles) Get(context.Context, string) (pluginsdk.FileObject, error) {
	return pluginsdk.FileObject{ID: "file-1"}, nil
}
func (gatewayFiles) Download(context.Context, string) (pluginsdk.FileDownload, error) {
	return pluginsdk.FileDownload{ID: "file-1"}, nil
}
func (gatewayFiles) Delete(context.Context, string) error { return nil }

type gatewayAudit struct{}

func (gatewayAudit) Record(context.Context, pluginsdk.AuditEntry) (pluginsdk.AuditReceipt, error) {
	return pluginsdk.AuditReceipt{ID: "audit-1"}, nil
}

type gatewayConfig struct{}

func (gatewayConfig) Get(ctx context.Context) (map[string]any, error) {
	return map[string]any{"transaction": ctx.Value(gatewayTxMarker{}) == true}, nil
}
func (gatewayConfig) Replace(context.Context, map[string]any) (map[string]any, error) {
	return map[string]any{"saved": true}, nil
}

type gatewaySecrets struct {
	mu    sync.Mutex
	value string
}

func (s *gatewaySecrets) Get(context.Context, string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.value, nil
}
func (s *gatewaySecrets) Set(_ context.Context, _ string, value string) error {
	s.mu.Lock()
	s.value = value
	s.mu.Unlock()
	return nil
}

type gatewayWorkflows struct{}

func (gatewayWorkflows) CreateDefinition(context.Context, pluginsdk.WorkflowDefinitionInput) (pluginsdk.WorkflowDefinition, error) {
	return pluginsdk.WorkflowDefinition{ID: "definition-1"}, nil
}
func (gatewayWorkflows) GetDefinition(context.Context, string) (pluginsdk.WorkflowDefinition, error) {
	return pluginsdk.WorkflowDefinition{ID: "definition-1"}, nil
}
func (gatewayWorkflows) PublishDefinition(context.Context, string) (pluginsdk.WorkflowDefinition, error) {
	return pluginsdk.WorkflowDefinition{ID: "definition-1", Status: pluginsdk.WorkflowDefinitionPublished}, nil
}
func (gatewayWorkflows) Start(context.Context, pluginsdk.WorkflowStartInput) (pluginsdk.WorkflowInstance, error) {
	return pluginsdk.WorkflowInstance{ID: "instance-1"}, nil
}
func (gatewayWorkflows) GetInstance(context.Context, string) (pluginsdk.WorkflowInstance, error) {
	return pluginsdk.WorkflowInstance{ID: "instance-1"}, nil
}
func (gatewayWorkflows) Approve(context.Context, pluginsdk.WorkflowTaskActionInput) (pluginsdk.WorkflowInstance, error) {
	return pluginsdk.WorkflowInstance{ID: "instance-1", Status: pluginsdk.WorkflowInstanceApproved}, nil
}
func (gatewayWorkflows) Reject(context.Context, pluginsdk.WorkflowTaskActionInput) (pluginsdk.WorkflowInstance, error) {
	return pluginsdk.WorkflowInstance{ID: "instance-1", Status: pluginsdk.WorkflowInstanceRejected}, nil
}
func (gatewayWorkflows) Withdraw(context.Context, pluginsdk.WorkflowInstanceActionInput) (pluginsdk.WorkflowInstance, error) {
	return pluginsdk.WorkflowInstance{ID: "instance-1", Status: pluginsdk.WorkflowInstanceWithdrawn}, nil
}
func (gatewayWorkflows) Transfer(context.Context, pluginsdk.WorkflowTargetActionInput) (pluginsdk.WorkflowInstance, error) {
	return pluginsdk.WorkflowInstance{ID: "instance-1"}, nil
}
func (gatewayWorkflows) Copy(context.Context, pluginsdk.WorkflowTargetActionInput) (pluginsdk.WorkflowInstance, error) {
	return pluginsdk.WorkflowInstance{ID: "instance-1"}, nil
}

type gatewayJobs struct{}

func (gatewayJobs) Schedule(context.Context, pluginsdk.JobScheduleInput) (pluginsdk.Job, error) {
	return pluginsdk.Job{ID: "job-1"}, nil
}
func (gatewayJobs) LeaseDue(context.Context, pluginsdk.JobLeaseInput) ([]pluginsdk.Job, error) {
	return []pluginsdk.Job{{ID: "job-1"}}, nil
}
func (gatewayJobs) Complete(context.Context, pluginsdk.JobCompleteInput) (pluginsdk.Job, error) {
	return pluginsdk.Job{ID: "job-1", Status: pluginsdk.JobStatusSucceeded}, nil
}
func (gatewayJobs) Fail(context.Context, pluginsdk.JobFailInput) (pluginsdk.Job, error) {
	return pluginsdk.Job{ID: "job-1", Status: pluginsdk.JobStatusRetryWait}, nil
}
func (gatewayJobs) Get(context.Context, string) (pluginsdk.Job, error) {
	return pluginsdk.Job{ID: "job-1"}, nil
}
func (gatewayJobs) List(context.Context, pluginsdk.JobQuery) ([]pluginsdk.Job, error) {
	return []pluginsdk.Job{{ID: "job-1"}}, nil
}

func TestHostGatewayClientConformanceIdentityAndTransactions(t *testing.T) {
	transactions := &gatewayTransactions{}
	secrets := &gatewaySecrets{}
	host := pluginsdk.HostServices{PluginID: "equipment", Transactions: transactions, DataScopes: gatewayScopes{}, Files: gatewayFiles{}, Audit: gatewayAudit{}, Config: gatewayConfig{}, Secrets: secrets, Workflows: gatewayWorkflows{}, Jobs: gatewayJobs{}}
	gateway, err := NewHostGateway(func(id string) (pluginsdk.HostServices, error) {
		if id != "equipment" {
			return pluginsdk.HostServices{}, errors.New("wrong identity")
		}
		return host, nil
	}, "gateway-jwt-secret", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = gateway.Close() })
	credential, err := gateway.Issue("equipment")
	if err != nil {
		t.Fatal(err)
	}
	client, err := pluginclient.New(pluginclient.Options{PluginID: "equipment", HostURL: credential.HostURL, HostToken: credential.Token})
	if err != nil {
		t.Fatal(err)
	}
	services, err := client.HostServices()
	if err != nil {
		t.Fatal(err)
	}

	userToken, err := security.SignJWT("gateway-jwt-secret", security.JWTIdentity{Subject: "user-7", Role: "admin", Roles: []string{"admin"}}, time.Minute, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	ctx := pluginclient.WithUserToken(context.Background(), userToken)
	scope, err := services.DataScopes.Resolve(ctx, pluginsdk.Permission{Resource: "asset", Action: "read"})
	if err != nil || scope.SubjectID() != "user-7" {
		t.Fatalf("scope=%+v err=%v", scope, err)
	}
	if _, err := services.Files.Store(ctx, pluginsdk.FileWrite{Name: "proof.txt"}); err != nil {
		t.Fatal(err)
	}
	if _, err := services.Files.List(ctx, pluginsdk.FileQuery{}); err != nil {
		t.Fatal(err)
	}
	if _, err := services.Files.Get(ctx, "file-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := services.Files.Download(ctx, "file-1"); err != nil {
		t.Fatal(err)
	}
	if err := services.Files.Delete(ctx, "file-1"); err != nil {
		t.Fatal(err)
	}
	if receipt, err := services.Audit.Record(ctx, pluginsdk.AuditEntry{Action: "asset.read", Resource: "asset"}); err != nil || receipt.ID != "audit-1" {
		t.Fatalf("audit=%+v err=%v", receipt, err)
	}
	if err := services.Secrets.Set(ctx, "api.key", "secret"); err != nil {
		t.Fatal(err)
	}
	if value, err := services.Secrets.Get(ctx, "api.key"); err != nil || value != "secret" {
		t.Fatalf("secret=%q err=%v", value, err)
	}
	if values, err := services.Config.Replace(ctx, map[string]any{"enabled": true}); err != nil || values["saved"] != true {
		t.Fatalf("config=%v err=%v", values, err)
	}
	if item, err := services.Workflows.Start(ctx, pluginsdk.WorkflowStartInput{}); err != nil || item.ID != "instance-1" {
		t.Fatalf("workflow=%+v err=%v", item, err)
	}
	definition, err := services.Workflows.CreateDefinition(ctx, pluginsdk.WorkflowDefinitionInput{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = services.Workflows.GetDefinition(ctx, definition.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = services.Workflows.PublishDefinition(ctx, definition.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = services.Workflows.GetInstance(ctx, "instance-1"); err != nil {
		t.Fatal(err)
	}
	if _, err = services.Workflows.Approve(ctx, pluginsdk.WorkflowTaskActionInput{}); err != nil {
		t.Fatal(err)
	}
	if _, err = services.Workflows.Reject(ctx, pluginsdk.WorkflowTaskActionInput{}); err != nil {
		t.Fatal(err)
	}
	if _, err = services.Workflows.Withdraw(ctx, pluginsdk.WorkflowInstanceActionInput{}); err != nil {
		t.Fatal(err)
	}
	if _, err = services.Workflows.Transfer(ctx, pluginsdk.WorkflowTargetActionInput{}); err != nil {
		t.Fatal(err)
	}
	if _, err = services.Workflows.Copy(ctx, pluginsdk.WorkflowTargetActionInput{}); err != nil {
		t.Fatal(err)
	}
	if item, err := services.Jobs.Schedule(ctx, pluginsdk.JobScheduleInput{}); err != nil || item.ID != "job-1" {
		t.Fatalf("job=%+v err=%v", item, err)
	}
	if _, err = services.Jobs.LeaseDue(ctx, pluginsdk.JobLeaseInput{}); err != nil {
		t.Fatal(err)
	}
	if _, err = services.Jobs.Complete(ctx, pluginsdk.JobCompleteInput{}); err != nil {
		t.Fatal(err)
	}
	if _, err = services.Jobs.Fail(ctx, pluginsdk.JobFailInput{}); err != nil {
		t.Fatal(err)
	}
	if _, err = services.Jobs.Get(ctx, "job-1"); err != nil {
		t.Fatal(err)
	}
	if _, err = services.Jobs.List(ctx, pluginsdk.JobQuery{}); err != nil {
		t.Fatal(err)
	}
	if err := services.Transactions.Within(ctx, func(tx pluginsdk.Transaction) error {
		values, err := services.Config.Get(tx.Context())
		if err != nil {
			return err
		}
		if values["transaction"] != true {
			return errors.New("transaction context missing")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	rollback := errors.New("rollback proof")
	if err := services.Transactions.Within(ctx, func(pluginsdk.Transaction) error { return rollback }); !errors.Is(err, rollback) {
		t.Fatalf("rollback error=%v", err)
	}
	canceled, cancel := context.WithCancel(ctx)
	if err := services.Transactions.Within(canceled, func(pluginsdk.Transaction) error { cancel(); return context.Canceled }); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled transaction error=%v", err)
	}
	transactions.mu.Lock()
	commits, rollbacks := transactions.commits, transactions.rollbacks
	transactions.mu.Unlock()
	if commits != 1 || rollbacks != 2 {
		t.Fatalf("transactions commits=%d rollbacks=%d", commits, rollbacks)
	}

	gateway.Revoke(credential.Token)
	if _, err := services.Config.Get(ctx); err == nil {
		t.Fatal("revoked plugin credential retained host access")
	}
}

func TestHostGatewayRejectsMissingCredentialNonLoopbackAndInvalidUserToken(t *testing.T) {
	host := pluginsdk.HostServices{PluginID: "equipment", Transactions: &gatewayTransactions{}, DataScopes: gatewayScopes{}, Files: gatewayFiles{}, Audit: gatewayAudit{}, Config: gatewayConfig{}, Secrets: &gatewaySecrets{}, Workflows: gatewayWorkflows{}, Jobs: gatewayJobs{}}
	gateway, err := NewHostGateway(func(string) (pluginsdk.HostServices, error) { return host, nil }, "gateway-jwt-secret", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer gateway.Close()
	credential, err := gateway.Issue("equipment")
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/v1/config/get", strings.NewReader(`{}`))
	request.RemoteAddr = "10.0.0.8:1234"
	request.Header.Set("Authorization", "Bearer "+credential.Token)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	gateway.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("non-loopback status=%d", recorder.Code)
	}
	request = httptest.NewRequest(http.MethodPost, "/v1/config/get", strings.NewReader(`{}`))
	request.RemoteAddr = "127.0.0.1:1234"
	request.Header.Set("Content-Type", "application/json")
	recorder = httptest.NewRecorder()
	gateway.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("missing credential status=%d", recorder.Code)
	}
	request = httptest.NewRequest(http.MethodPost, "/v1/scopes/resolve", strings.NewReader(`{"Resource":"asset","Action":"read"}`))
	request.RemoteAddr = "127.0.0.1:1234"
	request.Header.Set("Authorization", "Bearer "+credential.Token)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(pluginclient.UserTokenHeader, "not-a-jwt")
	recorder = httptest.NewRecorder()
	gateway.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("invalid user token status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var body pluginclient.ErrorResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil || body.Code != "host_context_invalid" {
		t.Fatalf("error=%+v decode=%v", body, err)
	}
	request = httptest.NewRequest(http.MethodPost, "/v1/config/get?extra=true", strings.NewReader(`{"unknown":true}`))
	request.RemoteAddr = "127.0.0.1:1234"
	request.Header.Set("Authorization", "Bearer "+credential.Token)
	request.Header.Set("Content-Type", "application/json")
	recorder = httptest.NewRecorder()
	gateway.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("query request status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	request = httptest.NewRequest(http.MethodPost, "/v1/config/get", strings.NewReader(`{"unknown":true}`))
	request.RemoteAddr = "127.0.0.1:1234"
	request.Header.Set("Authorization", "Bearer "+credential.Token)
	request.Header.Set("Content-Type", "application/json")
	recorder = httptest.NewRecorder()
	gateway.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("unknown field status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}
