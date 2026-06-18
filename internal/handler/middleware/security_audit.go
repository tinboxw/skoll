package middleware

import (
	"net/http"
	"strings"
	"time"

	domainaudit "github.com/tinboxw/skoll/internal/domain/audit"
	"github.com/tinboxw/skoll/internal/domain/shared"
)

type PermissionDeniedAuditInput struct {
	ID         shared.ID
	ActorID    string
	ActorName  string
	Resource   string
	Action     string
	Reason     string
	Request    *http.Request
	OccurredAt time.Time
}

func NewPermissionDeniedAuditEvent(in PermissionDeniedAuditInput) (*domainaudit.Event, error) {
	resource := strings.TrimSpace(strings.ToLower(in.Resource))
	if resource == "" {
		resource = "permission"
	}
	action := strings.TrimSpace(strings.ToLower(in.Action))
	if action == "" {
		action = "access"
	}
	reason := strings.TrimSpace(in.Reason)
	if reason == "" {
		reason = "permission_denied"
	}
	actorID := strings.TrimSpace(in.ActorID)
	if actorID == "" {
		actorID = "anonymous"
	}
	requestPath := ""
	requestMethod := ""
	requestID := ""
	traceID := ""
	userAgent := ""
	ip := ""
	if in.Request != nil {
		requestPath = in.Request.URL.Path
		requestMethod = in.Request.Method
		requestID = strings.TrimSpace(in.Request.Header.Get("X-Request-Id"))
		traceID = strings.TrimSpace(in.Request.Header.Get("X-Trace-Id"))
		userAgent = in.Request.UserAgent()
		ip = requestIP(in.Request)
	}
	return domainaudit.NewEvent(domainaudit.EventInput{
		ID:     in.ID,
		Type:   domainaudit.EventTypeSecurity,
		Action: domainaudit.AuditAction("system.security.deny"),
		Actor: domainaudit.ActorRef{
			Type: "user",
			ID:   shared.ID(actorID),
			Name: strings.TrimSpace(in.ActorName),
		},
		Resource: domainaudit.ResourceRef{
			Type: resource,
			ID:   action,
			Name: resource + "." + action,
		},
		Result: domainaudit.EventResultDenied,
		Trace: domainaudit.TraceContext{
			TraceID:   traceID,
			RequestID: requestID,
			Method:    requestMethod,
			Path:      requestPath,
			IP:        ip,
			UserAgent: userAgent,
		},
		Risk: domainaudit.EventRiskHigh,
		Metadata: map[string]any{
			"resource": resource,
			"action":   action,
			"reason":   reason,
		},
		SourceData: map[string]any{
			"kind":      "permission_denied",
			"resource":  resource,
			"action":    action,
			"reason":    reason,
			"method":    requestMethod,
			"path":      requestPath,
			"requestId": requestID,
		},
		OccurredAt: in.OccurredAt,
	})
}
