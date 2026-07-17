package middleware

import (
	"net/http"
	"strings"
	"time"

	domainaudit "github.com/tinboxw/skoll/internal/domain/audit"
	"github.com/tinboxw/skoll/internal/domain/shared"
)

type PluginLifecycleAuditInput struct {
	ID         shared.ID
	ActorID    string
	ActorName  string
	Operation  string
	PluginID   string
	Succeeded  bool
	Error      string
	Request    *http.Request
	OccurredAt time.Time
}

func NewPluginLifecycleAuditEvent(in PluginLifecycleAuditInput) (*domainaudit.Event, error) {
	operation := strings.TrimSpace(strings.ToLower(in.Operation))
	action, err := domainaudit.ParseAuditAction("plugin.lifecycle." + operation)
	if err != nil {
		return nil, err
	}
	actorID := strings.TrimSpace(in.ActorID)
	if actorID == "" {
		actorID = "system"
	}
	pluginID := strings.TrimSpace(strings.ToLower(in.PluginID))
	if pluginID == "" {
		pluginID = "unknown"
	}
	result := domainaudit.EventResultFailure
	if in.Succeeded {
		result = domainaudit.EventResultSuccess
	}

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
	metadata := map[string]any{
		"operation": operation,
		"pluginId":  pluginID,
	}
	if message := strings.TrimSpace(in.Error); message != "" {
		metadata["error"] = message
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
			Type: "plugin",
			ID:   pluginID,
			Name: pluginID,
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
		Risk:       domainaudit.EventRiskHigh,
		Metadata:   metadata,
		SourceData: map[string]any{"kind": "plugin_lifecycle", "operation": operation, "pluginId": pluginID, "method": method, "path": path, "requestId": requestID},
		OccurredAt: in.OccurredAt,
	})
}
