package hostservice

import (
	"context"
	"errors"
	"fmt"
	"strings"

	documentnumbersvc "github.com/tinboxw/skoll/internal/service/documentnumber"
	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

type documentNumberBackend interface {
	Preview(context.Context, string, pluginsdk.DocumentNumberInput) (pluginsdk.DocumentNumberResult, error)
	Issue(context.Context, string, pluginsdk.DocumentNumberInput) (pluginsdk.DocumentNumberResult, error)
}

type documentNumberService struct {
	pluginID string
	backend  documentNumberBackend
	scopes   pluginsdk.DataScopeService
	audit    pluginsdk.AuditService
}

func NewDocumentNumberService(pluginID string, backend documentNumberBackend, scopes pluginsdk.DataScopeService, audit pluginsdk.AuditService) (pluginsdk.DocumentNumberService, error) {
	pluginID = strings.ToLower(strings.TrimSpace(pluginID))
	if pluginID == "" || backend == nil || scopes == nil || audit == nil {
		return nil, fmt.Errorf("document number host dependencies are required")
	}
	return &documentNumberService{pluginID: pluginID, backend: backend, scopes: scopes, audit: audit}, nil
}

func (s *documentNumberService) Preview(ctx context.Context, input pluginsdk.DocumentNumberInput) (pluginsdk.DocumentNumberResult, error) {
	if err := input.Validate(false); err != nil {
		return pluginsdk.DocumentNumberResult{}, err
	}
	if err := s.authorizeTenant(ctx, input); err != nil {
		return pluginsdk.DocumentNumberResult{}, err
	}
	result, err := s.backend.Preview(ctx, s.pluginID, input)
	return result, documentNumberHostError(err)
}

func (s *documentNumberService) Issue(ctx context.Context, input pluginsdk.DocumentNumberInput) (pluginsdk.DocumentNumberResult, error) {
	if err := input.Validate(true); err != nil {
		return pluginsdk.DocumentNumberResult{}, err
	}
	if err := s.authorizeTenant(ctx, input); err != nil {
		return pluginsdk.DocumentNumberResult{}, err
	}
	result, err := s.backend.Issue(ctx, s.pluginID, input)
	if err != nil {
		return pluginsdk.DocumentNumberResult{}, documentNumberHostError(err)
	}
	if !result.Duplicate {
		_, err = s.audit.Record(ctx, pluginsdk.AuditEntry{
			Action: "document_number.issue", Resource: input.Rule.DocumentType, ResourceID: result.Number,
			Detail: map[string]any{"tenantId": input.TenantID, "periodKey": result.PeriodKey, "sequence": result.Sequence},
		})
		if err != nil {
			return pluginsdk.DocumentNumberResult{}, documentNumberHostError(err)
		}
	}
	return result, nil
}

func (s *documentNumberService) authorizeTenant(ctx context.Context, input pluginsdk.DocumentNumberInput) error {
	predicate, err := s.scopes.Resolve(ctx, input.Permission)
	if err != nil {
		return pluginsdk.NewDocumentNumberError(pluginsdk.DocumentNumberErrorForbidden, "permission", "document number permission was denied", false)
	}
	constrained := predicate.Constrain(pluginsdk.ScopeFilter{TenantIDs: []string{input.TenantID}})
	if constrained.Denied() || constrained.AllTenants() || len(constrained.TenantIDs()) != 1 || constrained.TenantIDs()[0] != input.TenantID {
		return pluginsdk.NewDocumentNumberError(pluginsdk.DocumentNumberErrorForbidden, "tenantId", "tenant is outside the trusted scope", false)
	}
	return nil
}

func documentNumberHostError(err error) error {
	if err == nil {
		return nil
	}
	var numberErr *pluginsdk.DocumentNumberError
	if errors.As(err, &numberErr) {
		return err
	}
	switch {
	case errors.Is(err, documentnumbersvc.ErrConflict):
		return pluginsdk.NewDocumentNumberError(pluginsdk.DocumentNumberErrorConflict, "idempotencyKey", "idempotency key was reused with different numbering input", false)
	case errors.Is(err, documentnumbersvc.ErrRuleConflict):
		return pluginsdk.NewDocumentNumberError(pluginsdk.DocumentNumberErrorConflict, "rule", "numbering rule cannot change within an active tenant, document type, and period sequence", false)
	case errors.Is(err, documentnumbersvc.ErrTransactionRequired):
		return pluginsdk.NewDocumentNumberError(pluginsdk.DocumentNumberErrorTransactionRequired, "transaction", "formal issuance requires an active host transaction", false)
	case errors.Is(err, documentnumbersvc.ErrExhausted):
		return pluginsdk.NewDocumentNumberError(pluginsdk.DocumentNumberErrorExhausted, "rule.width", "document number sequence is exhausted", false)
	default:
		return pluginsdk.NewDocumentNumberError(pluginsdk.DocumentNumberErrorUnavailable, "", "document number service is unavailable", true)
	}
}

var _ pluginsdk.DocumentNumberService = (*documentNumberService)(nil)
