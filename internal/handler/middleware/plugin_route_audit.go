package middleware

import (
	"net/http"
	"strings"
	"time"

	domainaudit "github.com/tinboxw/skoll/internal/domain/audit"
	"github.com/tinboxw/skoll/internal/domain/shared"
)

type PluginRouteAuditInput struct {
	ID          shared.ID
	ActorID     string
	ActorName   string
	Source      string
	Permission  string
	AuditAction string
	StatusCode  int
	Request     *http.Request
	OccurredAt  time.Time
}

func NewPluginRouteAuditEvent(in PluginRouteAuditInput) (*domainaudit.Event, error) {
	action, err := domainaudit.ParseAuditAction(in.AuditAction)
	if err != nil {
		return nil, err
	}
	actorID := strings.TrimSpace(in.ActorID)
	if actorID == "" {
		actorID = "anonymous"
	}
	source := strings.TrimSpace(strings.ToLower(in.Source))
	if source == "" {
		source = "plugin"
	}
	permission := strings.TrimSpace(strings.ToLower(in.Permission))
	statusCode := in.StatusCode
	if statusCode == 0 {
		statusCode = http.StatusOK
	}
	result, risk := pluginRouteResultAndRisk(statusCode)

	method := ""
	path := ""
	requestID := ""
	traceID := ""
	userAgent := ""
	ip := ""
	if in.Request != nil {
		method = strings.ToUpper(strings.TrimSpace(in.Request.Method))
		path = strings.TrimSpace(in.Request.URL.Path)
		requestID = strings.TrimSpace(in.Request.Header.Get("X-Request-Id"))
		traceID = strings.TrimSpace(in.Request.Header.Get("X-Trace-Id"))
		userAgent = in.Request.UserAgent()
		ip = requestIP(in.Request)
	}

	return domainaudit.NewEvent(domainaudit.EventInput{
		ID:     in.ID,
		Type:   domainaudit.EventTypePlugin,
		Action: action,
		Actor: domainaudit.ActorRef{
			Type: "user",
			ID:   shared.ID(actorID),
			Name: strings.TrimSpace(in.ActorName),
		},
		Resource: domainaudit.ResourceRef{
			Type: "plugin_route",
			ID:   method + " " + path,
			Name: source,
		},
		Result: result,
		Trace: domainaudit.TraceContext{
			TraceID:   traceID,
			RequestID: requestID,
			Method:    method,
			Path:      path,
			IP:        ip,
			UserAgent: userAgent,
		},
		Risk: risk,
		Metadata: map[string]any{
			"source":     source,
			"permission": permission,
			"status":     statusCode,
		},
		SourceData: map[string]any{
			"kind":       "plugin_route",
			"source":     source,
			"permission": permission,
			"method":     method,
			"path":       path,
			"status":     statusCode,
			"requestId":  requestID,
		},
		OccurredAt: in.OccurredAt,
	})
}

func pluginRouteResultAndRisk(statusCode int) (domainaudit.EventResult, domainaudit.EventRisk) {
	if statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden {
		return domainaudit.EventResultDenied, domainaudit.EventRiskHigh
	}
	if statusCode >= http.StatusBadRequest {
		return domainaudit.EventResultFailure, domainaudit.EventRiskMedium
	}
	return domainaudit.EventResultSuccess, domainaudit.EventRiskLow
}
