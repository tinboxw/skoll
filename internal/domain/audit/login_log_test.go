package audit

import (
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
)

func TestNewLoginLogBuildsSuccessLog(t *testing.T) {
	now := time.Date(2026, time.June, 19, 11, 0, 0, 0, time.UTC)
	metadata := map[string]any{" tenant ": "main"}

	log, err := NewLoginLog(LoginLogInput{
		ID:        shared.ID("login-1"),
		Account:   " Admin ",
		ActorID:   shared.ID(" user-1 "),
		Result:    LoginResultSuccess,
		IP:        " 127.0.0.1 ",
		UserAgent: " Mozilla/5.0 ",
		SessionID: " session-1 ",
		Trace: TraceContext{
			TraceID:   " trace-1 ",
			RequestID: " req-1 ",
			Method:    " post ",
			Path:      " /skoll/v1/auth/login ",
		},
		Metadata:   metadata,
		OccurredAt: now,
	})
	if err != nil {
		t.Fatalf("NewLoginLog() error = %v", err)
	}

	if log.Account != "admin" || log.ActorID != shared.ID("user-1") {
		t.Fatalf("unexpected actor fields: %+v", log)
	}
	if log.Result != LoginResultSuccess || log.SessionID != "session-1" || log.FailureReason != "" {
		t.Fatalf("unexpected result fields: %+v", log)
	}
	if log.IP != "127.0.0.1" || log.UserAgent != "Mozilla/5.0" {
		t.Fatalf("unexpected request fields: %+v", log)
	}
	if log.Trace.Method != "POST" || log.Trace.Path != "/skoll/v1/auth/login" {
		t.Fatalf("unexpected trace: %+v", log.Trace)
	}
	if log.Metadata["tenant"] != "main" {
		t.Fatalf("unexpected metadata: %+v", log.Metadata)
	}
	metadata["tenant"] = "changed"
	if log.Metadata["tenant"] != "main" {
		t.Fatalf("metadata should be copied: %+v", log.Metadata)
	}
}

func TestNewLoginLogBuildsFailureLog(t *testing.T) {
	log, err := NewLoginLog(validLoginLogInput(func(in *LoginLogInput) {
		in.Result = LoginResultFailure
		in.SessionID = ""
		in.FailureReason = "bad_password"
		in.ActorID = ""
	}))
	if err != nil {
		t.Fatalf("NewLoginLog() error = %v", err)
	}
	if log.Result != LoginResultFailure || log.FailureReason != "bad_password" || log.SessionID != "" {
		t.Fatalf("unexpected failure log: %+v", log)
	}
}

func TestNewLoginLogRejectsInvalidInput(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*LoginLogInput)
	}{
		{name: "missing id", mutate: func(in *LoginLogInput) { in.ID = "" }},
		{name: "missing account", mutate: func(in *LoginLogInput) { in.Account = " " }},
		{name: "invalid result", mutate: func(in *LoginLogInput) { in.Result = LoginResult("denied") }},
		{name: "success missing session", mutate: func(in *LoginLogInput) { in.SessionID = "" }},
		{name: "failure missing reason", mutate: func(in *LoginLogInput) {
			in.Result = LoginResultFailure
			in.SessionID = ""
			in.FailureReason = " "
		}},
		{name: "missing time", mutate: func(in *LoginLogInput) { in.OccurredAt = time.Time{} }},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewLoginLog(validLoginLogInput(tt.mutate))
			if err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestLoginResultValidate(t *testing.T) {
	for _, result := range []LoginResult{LoginResultSuccess, LoginResultFailure} {
		if err := result.Validate(); err != nil {
			t.Fatalf("result %q should be valid: %v", result, err)
		}
	}
	if err := LoginResult("denied").Validate(); err == nil {
		t.Fatal("expected invalid login result")
	}
}

func validLoginLogInput(mutate func(*LoginLogInput)) LoginLogInput {
	in := LoginLogInput{
		ID:         shared.ID("login-1"),
		Account:    "admin",
		ActorID:    shared.ID("user-1"),
		Result:     LoginResultSuccess,
		IP:         "127.0.0.1",
		UserAgent:  "test-agent",
		SessionID:  "session-1",
		OccurredAt: time.Date(2026, time.June, 19, 11, 0, 0, 0, time.UTC),
	}
	if mutate != nil {
		mutate(&in)
	}
	return in
}
