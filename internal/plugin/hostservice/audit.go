package hostservice

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	auditsvc "github.com/tinboxw/skoll/internal/service/audit"
	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

var auditNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_.-]{0,127}$`)

type auditService struct {
	pluginID string
	audit    auditsvc.Service
}

func NewAuditService(pluginID string, audit auditsvc.Service) (pluginsdk.AuditService, error) {
	pluginID = strings.ToLower(strings.TrimSpace(pluginID))
	if pluginID == "" {
		return nil, fmt.Errorf("plugin host identity is required")
	}
	if audit == nil {
		return nil, fmt.Errorf("plugin host audit backend is required")
	}
	return &auditService{pluginID: pluginID, audit: audit}, nil
}

func (s *auditService) Record(ctx context.Context, entry pluginsdk.AuditEntry) (pluginsdk.AuditReceipt, error) {
	action := strings.ToLower(strings.TrimSpace(entry.Action))
	resource := strings.ToLower(strings.TrimSpace(entry.Resource))
	if !auditNamePattern.MatchString(action) || !auditNamePattern.MatchString(resource) {
		return pluginsdk.AuditReceipt{}, fmt.Errorf("plugin audit action and resource are invalid")
	}
	result, err := normalizeAuditResult(entry.Result)
	if err != nil {
		return pluginsdk.AuditReceipt{}, err
	}
	risk, err := normalizeAuditRisk(entry.Risk)
	if err != nil {
		return pluginsdk.AuditReceipt{}, err
	}
	detail := redactMap(entry.Detail)
	detail["pluginId"] = s.pluginID
	detail["result"] = result
	detail["risk"] = risk
	if operation, ok := pluginsdk.OperationContextFromContext(ctx); ok {
		detail["correlationId"] = operation.CorrelationID
		if operation.RequestID != "" {
			detail["requestId"] = operation.RequestID
		}
		if operation.TraceID != "" {
			detail["traceId"] = operation.TraceID
		}
	}
	if err := validateMetadataSize(detail); err != nil {
		return pluginsdk.AuditReceipt{}, err
	}
	actor := trustedHostActor(ctx, s.pluginID)
	record, err := s.audit.Append(ctx, actor.id, "plugin."+s.pluginID+"."+action, "plugin:"+s.pluginID+":"+resource, strings.TrimSpace(entry.ResourceID), detail)
	if err != nil {
		return pluginsdk.AuditReceipt{}, err
	}
	return pluginsdk.AuditReceipt{ID: record.ID.String(), OccurredAt: record.OccurredAt}, nil
}

func normalizeAuditResult(value pluginsdk.AuditResult) (string, error) {
	if value == "" {
		return string(pluginsdk.AuditResultSuccess), nil
	}
	switch value {
	case pluginsdk.AuditResultSuccess, pluginsdk.AuditResultFailure, pluginsdk.AuditResultDenied:
		return string(value), nil
	default:
		return "", fmt.Errorf("plugin audit result is invalid")
	}
}

func normalizeAuditRisk(value pluginsdk.AuditRisk) (string, error) {
	if value == "" {
		return string(pluginsdk.AuditRiskLow), nil
	}
	switch value {
	case pluginsdk.AuditRiskLow, pluginsdk.AuditRiskMedium, pluginsdk.AuditRiskHigh, pluginsdk.AuditRiskCritical:
		return string(value), nil
	default:
		return "", fmt.Errorf("plugin audit risk is invalid")
	}
}

var _ pluginsdk.AuditService = (*auditService)(nil)
