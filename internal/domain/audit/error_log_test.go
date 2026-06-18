package audit

import (
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
)

func TestNewErrorLogBuildsCompleteLog(t *testing.T) {
	now := time.Date(2026, time.June, 19, 12, 0, 0, 0, time.UTC)
	metadata := map[string]any{" module ": "audit"}

	log, err := NewErrorLog(ErrorLogInput{
		ID: shared.ID("error-1"),
		Trace: TraceContext{
			TraceID:   " trace-1 ",
			RequestID: " req-1 ",
			IP:        " 127.0.0.1 ",
			UserAgent: " test-agent ",
		},
		Request: RequestContext{
			Method:     " post ",
			Path:       " /skoll/v1/users ",
			Query:      " page=1 ",
			StatusCode: 500,
		},
		ErrorCode:  " USER_CREATE_FAILED ",
		Level:      ErrorLevelError,
		Summary:    " create user failed ",
		Message:    " database unavailable ",
		Metadata:   metadata,
		OccurredAt: now,
	})
	if err != nil {
		t.Fatalf("NewErrorLog() error = %v", err)
	}

	if log.Trace.TraceID != "trace-1" || log.Trace.RequestID != "req-1" || log.Trace.IP != "127.0.0.1" {
		t.Fatalf("unexpected trace: %+v", log.Trace)
	}
	if log.Request.Method != "POST" || log.Request.Path != "/skoll/v1/users" || log.Request.Query != "page=1" || log.Request.StatusCode != 500 {
		t.Fatalf("unexpected request: %+v", log.Request)
	}
	if log.ErrorCode != "user_create_failed" || log.Level != ErrorLevelError || log.Summary != "create user failed" {
		t.Fatalf("unexpected error fields: %+v", log)
	}
	if log.Metadata["module"] != "audit" {
		t.Fatalf("unexpected metadata: %+v", log.Metadata)
	}
	metadata["module"] = "changed"
	if log.Metadata["module"] != "audit" {
		t.Fatalf("metadata should be copied: %+v", log.Metadata)
	}
}

func TestNewErrorLogAcceptsRequestIDWithoutTraceID(t *testing.T) {
	log, err := NewErrorLog(validErrorLogInput(func(in *ErrorLogInput) {
		in.Trace.TraceID = ""
		in.Trace.RequestID = "req-1"
		in.Level = ErrorLevelCritical
	}))
	if err != nil {
		t.Fatalf("NewErrorLog() error = %v", err)
	}
	if log.Trace.RequestID != "req-1" || log.Level != ErrorLevelCritical {
		t.Fatalf("unexpected log: %+v", log)
	}
}

func TestNewErrorLogRejectsInvalidInput(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*ErrorLogInput)
	}{
		{name: "missing id", mutate: func(in *ErrorLogInput) { in.ID = "" }},
		{name: "missing trace and request id", mutate: func(in *ErrorLogInput) {
			in.Trace.TraceID = ""
			in.Trace.RequestID = ""
		}},
		{name: "missing method", mutate: func(in *ErrorLogInput) { in.Request.Method = " " }},
		{name: "missing path", mutate: func(in *ErrorLogInput) { in.Request.Path = " " }},
		{name: "missing error code", mutate: func(in *ErrorLogInput) { in.ErrorCode = " " }},
		{name: "invalid level", mutate: func(in *ErrorLogInput) { in.Level = ErrorLevel("fatal") }},
		{name: "missing summary", mutate: func(in *ErrorLogInput) { in.Summary = " " }},
		{name: "missing time", mutate: func(in *ErrorLogInput) { in.OccurredAt = time.Time{} }},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewErrorLog(validErrorLogInput(tt.mutate))
			if err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestErrorLevelValidate(t *testing.T) {
	for _, level := range []ErrorLevel{ErrorLevelInfo, ErrorLevelWarn, ErrorLevelError, ErrorLevelCritical} {
		if err := level.Validate(); err != nil {
			t.Fatalf("level %q should be valid: %v", level, err)
		}
	}
	if err := ErrorLevel("fatal").Validate(); err == nil {
		t.Fatal("expected invalid error level")
	}
}

func validErrorLogInput(mutate func(*ErrorLogInput)) ErrorLogInput {
	in := ErrorLogInput{
		ID: shared.ID("error-1"),
		Trace: TraceContext{
			TraceID: "trace-1",
		},
		Request: RequestContext{
			Method:     "GET",
			Path:       "/skoll/v1/users",
			StatusCode: 500,
		},
		ErrorCode:  "server_error",
		Level:      ErrorLevelError,
		Summary:    "server error",
		OccurredAt: time.Date(2026, time.June, 19, 12, 0, 0, 0, time.UTC),
	}
	if mutate != nil {
		mutate(&in)
	}
	return in
}
