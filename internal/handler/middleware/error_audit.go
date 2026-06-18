package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	domainaudit "github.com/tinboxw/skoll/internal/domain/audit"
	"github.com/tinboxw/skoll/internal/domain/shared"
)

type ErrorAuditInput struct {
	ID         shared.ID
	LogID      shared.ID
	Request    *http.Request
	StatusCode int
	ErrorCode  string
	Summary    string
	Message    string
	Panic      any
	OccurredAt time.Time
}

type ErrorAuditOption func(*errorAuditOptions)

type errorAuditOptions struct {
	nowFn AuditClock
	idFn  AuditIDFunc
}

func ErrorAudit(sink AuditEventSink, opts ...ErrorAuditOption) func(http.Handler) http.Handler {
	cfg := errorAuditOptions{
		nowFn: func() time.Time { return time.Now().UTC() },
		idFn: func() shared.ID {
			return shared.ID("audit-event-" + strconv.FormatInt(time.Now().UTC().UnixNano(), 10))
		},
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			recorder := &auditResponseRecorder{ResponseWriter: w, statusCode: http.StatusOK}
			defer func() {
				if rec := recover(); rec != nil {
					appendErrorAuditEvent(r, sink, ErrorAuditInput{
						ID:         cfg.idFn(),
						LogID:      cfg.idFn(),
						Request:    r,
						StatusCode: http.StatusInternalServerError,
						ErrorCode:  "panic",
						Summary:    "panic recovered",
						Message:    fmt.Sprint(rec),
						Panic:      rec,
						OccurredAt: cfg.nowFn(),
					})
					http.Error(recorder, "internal server error", http.StatusInternalServerError)
				}
			}()

			next.ServeHTTP(recorder, r)
			if recorder.statusCode >= http.StatusInternalServerError {
				appendErrorAuditEvent(r, sink, ErrorAuditInput{
					ID:         cfg.idFn(),
					LogID:      cfg.idFn(),
					Request:    r,
					StatusCode: recorder.statusCode,
					ErrorCode:  "handler_error",
					Summary:    "handler returned error status",
					Message:    http.StatusText(recorder.statusCode),
					OccurredAt: cfg.nowFn(),
				})
			}
		})
	}
}

func WithErrorAuditClock(nowFn AuditClock) ErrorAuditOption {
	return func(cfg *errorAuditOptions) {
		if nowFn != nil {
			cfg.nowFn = nowFn
		}
	}
}

func WithErrorAuditID(idFn AuditIDFunc) ErrorAuditOption {
	return func(cfg *errorAuditOptions) {
		if idFn != nil {
			cfg.idFn = idFn
		}
	}
}

func NewErrorAuditEvent(in ErrorAuditInput) (*domainaudit.Event, error) {
	statusCode := in.StatusCode
	if statusCode == 0 {
		statusCode = http.StatusInternalServerError
	}
	errorCode := strings.TrimSpace(strings.ToLower(in.ErrorCode))
	if errorCode == "" {
		errorCode = "handler_error"
	}
	summary := strings.TrimSpace(in.Summary)
	if summary == "" {
		summary = "handler returned error status"
	}
	level := domainaudit.ErrorLevelError
	action := domainaudit.AuditAction("error.request.handled")
	risk := domainaudit.EventRiskMedium
	if in.Panic != nil || errorCode == "panic" {
		level = domainaudit.ErrorLevelCritical
		action = domainaudit.AuditAction("error.request.panic")
		risk = domainaudit.EventRiskCritical
	}
	trace := errorTrace(in.Request, in.ID)
	request := domainaudit.RequestContext{
		Method:     requestMethod(in.Request),
		Path:       requestPath(in.Request),
		Query:      requestQuery(in.Request),
		StatusCode: statusCode,
	}
	errorLog, err := domainaudit.NewErrorLog(domainaudit.ErrorLogInput{
		ID:         in.LogID,
		Trace:      trace,
		Request:    request,
		ErrorCode:  errorCode,
		Level:      level,
		Summary:    summary,
		Message:    strings.TrimSpace(in.Message),
		Metadata:   map[string]any{"status": statusCode},
		OccurredAt: in.OccurredAt,
	})
	if err != nil {
		return nil, err
	}
	return domainaudit.NewEvent(domainaudit.EventInput{
		ID:       in.ID,
		Type:     domainaudit.EventTypeError,
		Action:   action,
		Actor:    requestActor(errorRequestContext(in.Request)),
		Resource: domainaudit.ResourceRef{Type: "http_request", ID: request.Method + " " + request.Path},
		Result:   domainaudit.EventResultFailure,
		Trace:    trace,
		Risk:     risk,
		Metadata: map[string]any{
			"status":    statusCode,
			"errorCode": errorCode,
			"summary":   summary,
		},
		SourceData: map[string]any{
			"kind":       "error_log",
			"id":         errorLog.ID.String(),
			"traceId":    errorLog.Trace.TraceID,
			"requestId":  errorLog.Trace.RequestID,
			"method":     errorLog.Request.Method,
			"path":       errorLog.Request.Path,
			"query":      errorLog.Request.Query,
			"status":     errorLog.Request.StatusCode,
			"errorCode":  errorLog.ErrorCode,
			"level":      string(errorLog.Level),
			"summary":    errorLog.Summary,
			"message":    errorLog.Message,
			"occurredAt": errorLog.OccurredAt.Format(time.RFC3339Nano),
		},
		OccurredAt: in.OccurredAt,
	})
}

func appendErrorAuditEvent(_ *http.Request, sink AuditEventSink, in ErrorAuditInput) {
	if sink == nil {
		return
	}
	event, err := NewErrorAuditEvent(in)
	if err != nil {
		return
	}
	_ = sink.AppendEvent(errorRequestContext(in.Request), event)
}

func errorTrace(r *http.Request, fallbackID shared.ID) domainaudit.TraceContext {
	trace := domainaudit.TraceContext{
		Method:    requestMethod(r),
		Path:      requestPath(r),
		IP:        requestIP(r),
		UserAgent: "",
	}
	if r != nil {
		trace.TraceID = strings.TrimSpace(r.Header.Get("X-Trace-Id"))
		trace.RequestID = strings.TrimSpace(r.Header.Get("X-Request-Id"))
		trace.UserAgent = r.UserAgent()
	}
	if trace.TraceID == "" && trace.RequestID == "" {
		trace.RequestID = fallbackID.String()
	}
	return trace
}

func errorRequestContext(r *http.Request) context.Context {
	if r == nil {
		return context.Background()
	}
	return r.Context()
}

func requestMethod(r *http.Request) string {
	if r == nil {
		return ""
	}
	return r.Method
}

func requestPath(r *http.Request) string {
	if r == nil || r.URL == nil {
		return ""
	}
	return r.URL.Path
}

func requestQuery(r *http.Request) string {
	if r == nil || r.URL == nil {
		return ""
	}
	return r.URL.RawQuery
}
