package hostservice

import (
	"context"
	"errors"
	"fmt"
	"strings"

	documentworkflowsvc "github.com/tinboxw/skoll/internal/service/documentworkflow"
	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

type documentWorkflowBackend interface {
	Submit(context.Context, string, pluginsdk.WorkflowActor, pluginsdk.DocumentWorkflowSubmitInput) (pluginsdk.DocumentWorkflowResult, error)
	Act(context.Context, string, pluginsdk.WorkflowActor, pluginsdk.DocumentWorkflowActionInput) (pluginsdk.DocumentWorkflowResult, error)
	Get(context.Context, string, pluginsdk.DocumentWorkflowGetInput) (pluginsdk.DocumentWorkflowResult, error)
}

type documentWorkflowService struct {
	pluginID string
	backend  documentWorkflowBackend
	scopes   pluginsdk.DataScopeService
	audit    pluginsdk.AuditService
}

func NewDocumentWorkflowService(pluginID string, backend documentWorkflowBackend, scopes pluginsdk.DataScopeService, audit pluginsdk.AuditService) (pluginsdk.DocumentWorkflowService, error) {
	pluginID = strings.ToLower(strings.TrimSpace(pluginID))
	if pluginID == "" || backend == nil || scopes == nil || audit == nil {
		return nil, fmt.Errorf("document workflow host dependencies are required")
	}
	return &documentWorkflowService{pluginID: pluginID, backend: backend, scopes: scopes, audit: audit}, nil
}

func (s *documentWorkflowService) Submit(ctx context.Context, input pluginsdk.DocumentWorkflowSubmitInput) (pluginsdk.DocumentWorkflowResult, error) {
	if err := input.Validate(); err != nil {
		return pluginsdk.DocumentWorkflowResult{}, documentWorkflowHostError(err)
	}
	if err := s.authorizeTenant(ctx, input.TenantID, input.Permission); err != nil {
		return pluginsdk.DocumentWorkflowResult{}, err
	}
	result, err := s.backend.Submit(ctx, s.pluginID, s.actor(ctx), input)
	if err != nil {
		return pluginsdk.DocumentWorkflowResult{}, documentWorkflowHostError(err)
	}
	if !result.Duplicate {
		if err = s.record(ctx, "document.submit", input.Draft.ID, result); err != nil {
			return pluginsdk.DocumentWorkflowResult{}, documentWorkflowHostError(err)
		}
	}
	return result, nil
}

func (s *documentWorkflowService) Act(ctx context.Context, input pluginsdk.DocumentWorkflowActionInput) (pluginsdk.DocumentWorkflowResult, error) {
	if err := input.Validate(); err != nil {
		return pluginsdk.DocumentWorkflowResult{}, documentWorkflowHostError(err)
	}
	if err := s.authorizeTenant(ctx, input.TenantID, input.Permission); err != nil {
		return pluginsdk.DocumentWorkflowResult{}, err
	}
	result, err := s.backend.Act(ctx, s.pluginID, s.actor(ctx), input)
	if err != nil {
		return pluginsdk.DocumentWorkflowResult{}, documentWorkflowHostError(err)
	}
	if !result.Duplicate {
		if err = s.record(ctx, "document."+string(input.Action), input.DocumentID, result); err != nil {
			return pluginsdk.DocumentWorkflowResult{}, documentWorkflowHostError(err)
		}
	}
	return result, nil
}

func (s *documentWorkflowService) Get(ctx context.Context, input pluginsdk.DocumentWorkflowGetInput) (pluginsdk.DocumentWorkflowResult, error) {
	if err := input.Validate(); err != nil {
		return pluginsdk.DocumentWorkflowResult{}, documentWorkflowHostError(err)
	}
	if err := s.authorizeTenant(ctx, input.TenantID, input.Permission); err != nil {
		return pluginsdk.DocumentWorkflowResult{}, err
	}
	result, err := s.backend.Get(ctx, s.pluginID, input)
	return result, documentWorkflowHostError(err)
}

func (s *documentWorkflowService) authorizeTenant(ctx context.Context, tenantID string, permission pluginsdk.Permission) error {
	predicate, err := s.scopes.Resolve(ctx, permission)
	if err != nil {
		return pluginsdk.NewDocumentWorkflowError(pluginsdk.DocumentWorkflowErrorForbidden, "permission", "document permission was denied", false)
	}
	constrained := predicate.Constrain(pluginsdk.ScopeFilter{TenantIDs: []string{tenantID}})
	if constrained.Denied() || constrained.AllTenants() || len(constrained.TenantIDs()) != 1 || constrained.TenantIDs()[0] != tenantID {
		return pluginsdk.NewDocumentWorkflowError(pluginsdk.DocumentWorkflowErrorForbidden, "tenantId", "tenant is outside the trusted scope", false)
	}
	return nil
}

func (s *documentWorkflowService) actor(ctx context.Context) pluginsdk.WorkflowActor {
	actor := trustedHostActor(ctx, s.pluginID)
	return pluginsdk.WorkflowActor{ID: actor.id, Name: actor.name}
}

func (s *documentWorkflowService) record(ctx context.Context, action, documentID string, result pluginsdk.DocumentWorkflowResult) error {
	_, err := s.audit.Record(ctx, pluginsdk.AuditEntry{
		Action: action, Resource: result.Document.Type, ResourceID: documentID, Risk: pluginsdk.AuditRiskMedium,
		Detail: map[string]any{"state": result.Document.State, "version": result.Document.Version, "workflowInstanceId": result.Workflow.ID},
	})
	return err
}

func documentWorkflowHostError(err error) error {
	if err == nil {
		return nil
	}
	var publicErr *pluginsdk.DocumentWorkflowError
	if errors.As(err, &publicErr) {
		return err
	}
	var contractErr *pluginsdk.DocumentContractError
	if errors.As(err, &contractErr) {
		return pluginsdk.NewDocumentWorkflowError(pluginsdk.DocumentWorkflowErrorInvalidRequest, contractErr.Field, contractErr.Message, false)
	}
	switch {
	case errors.Is(err, documentworkflowsvc.ErrNotFound):
		return pluginsdk.NewDocumentWorkflowError(pluginsdk.DocumentWorkflowErrorNotFound, "documentId", "document workflow was not found", false)
	case errors.Is(err, documentworkflowsvc.ErrConflict):
		return pluginsdk.NewDocumentWorkflowError(pluginsdk.DocumentWorkflowErrorConflict, "expectedVersion", "document workflow action conflicts with current state or idempotency input", false)
	case errors.Is(err, documentworkflowsvc.ErrTransactionRequired):
		return pluginsdk.NewDocumentWorkflowError(pluginsdk.DocumentWorkflowErrorTransactionRequired, "transaction", "document workflow writes require an active host transaction", false)
	default:
		return pluginsdk.NewDocumentWorkflowError(pluginsdk.DocumentWorkflowErrorUnavailable, "", "document workflow service is unavailable", true)
	}
}

var _ pluginsdk.DocumentWorkflowService = (*documentWorkflowService)(nil)
