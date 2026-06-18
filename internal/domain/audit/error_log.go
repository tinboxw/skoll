package audit

import (
	"fmt"
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
)

type ErrorLevel string

const (
	ErrorLevelInfo     ErrorLevel = "info"
	ErrorLevelWarn     ErrorLevel = "warn"
	ErrorLevelError    ErrorLevel = "error"
	ErrorLevelCritical ErrorLevel = "critical"
)

type RequestContext struct {
	Method     string
	Path       string
	Query      string
	StatusCode int
}

type ErrorLogInput struct {
	ID         shared.ID
	Trace      TraceContext
	Request    RequestContext
	ErrorCode  string
	Level      ErrorLevel
	Summary    string
	Message    string
	Metadata   map[string]any
	OccurredAt time.Time
}

type ErrorLog struct {
	ID         shared.ID
	Trace      TraceContext
	Request    RequestContext
	ErrorCode  string
	Level      ErrorLevel
	Summary    string
	Message    string
	Metadata   map[string]any
	OccurredAt time.Time
}

func NewErrorLog(in ErrorLogInput) (*ErrorLog, error) {
	in = normalizeErrorLogInput(in)
	if err := validateErrorLogInput(in); err != nil {
		return nil, err
	}
	return &ErrorLog{
		ID:         in.ID,
		Trace:      in.Trace,
		Request:    in.Request,
		ErrorCode:  in.ErrorCode,
		Level:      in.Level,
		Summary:    in.Summary,
		Message:    in.Message,
		Metadata:   copyMetadata(in.Metadata),
		OccurredAt: in.OccurredAt,
	}, nil
}

func (l ErrorLevel) Validate() error {
	switch l {
	case ErrorLevelInfo, ErrorLevelWarn, ErrorLevelError, ErrorLevelCritical:
		return nil
	default:
		return fmt.Errorf("error log level is invalid")
	}
}

func normalizeErrorLogInput(in ErrorLogInput) ErrorLogInput {
	in.Trace = normalizeTrace(in.Trace)
	in.Request = normalizeRequest(in.Request)
	in.ErrorCode = strings.TrimSpace(strings.ToLower(in.ErrorCode))
	in.Summary = strings.TrimSpace(in.Summary)
	in.Message = strings.TrimSpace(in.Message)
	return in
}

func normalizeRequest(request RequestContext) RequestContext {
	return RequestContext{
		Method:     strings.TrimSpace(strings.ToUpper(request.Method)),
		Path:       strings.TrimSpace(request.Path),
		Query:      strings.TrimSpace(request.Query),
		StatusCode: request.StatusCode,
	}
}

func validateErrorLogInput(in ErrorLogInput) error {
	if in.ID.IsZero() {
		return fmt.Errorf("error log id is required")
	}
	if in.Trace.TraceID == "" && in.Trace.RequestID == "" {
		return fmt.Errorf("error log trace or request id is required")
	}
	if in.Request.Method == "" || in.Request.Path == "" {
		return fmt.Errorf("error log request method and path are required")
	}
	if in.ErrorCode == "" {
		return fmt.Errorf("error code is required")
	}
	if err := in.Level.Validate(); err != nil {
		return err
	}
	if in.Summary == "" {
		return fmt.Errorf("error summary is required")
	}
	if in.OccurredAt.IsZero() {
		return fmt.Errorf("error log occurred time is required")
	}
	return nil
}
