package audit

import (
	"fmt"
	"strings"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
)

type LoginResult string

const (
	LoginResultSuccess LoginResult = "success"
	LoginResultFailure LoginResult = "failure"
)

type LoginLogInput struct {
	ID            shared.ID
	Account       string
	ActorID       shared.ID
	Result        LoginResult
	IP            string
	UserAgent     string
	FailureReason string
	SessionID     string
	Trace         TraceContext
	Metadata      map[string]any
	OccurredAt    time.Time
}

type LoginLog struct {
	ID            shared.ID
	Account       string
	ActorID       shared.ID
	Result        LoginResult
	IP            string
	UserAgent     string
	FailureReason string
	SessionID     string
	Trace         TraceContext
	Metadata      map[string]any
	OccurredAt    time.Time
}

func NewLoginLog(in LoginLogInput) (*LoginLog, error) {
	in = normalizeLoginLogInput(in)
	if err := validateLoginLogInput(in); err != nil {
		return nil, err
	}
	return &LoginLog{
		ID:            in.ID,
		Account:       in.Account,
		ActorID:       in.ActorID,
		Result:        in.Result,
		IP:            in.IP,
		UserAgent:     in.UserAgent,
		FailureReason: in.FailureReason,
		SessionID:     in.SessionID,
		Trace:         in.Trace,
		Metadata:      copyMetadata(in.Metadata),
		OccurredAt:    in.OccurredAt,
	}, nil
}

func (r LoginResult) Validate() error {
	switch r {
	case LoginResultSuccess, LoginResultFailure:
		return nil
	default:
		return fmt.Errorf("login result is invalid")
	}
}

func normalizeLoginLogInput(in LoginLogInput) LoginLogInput {
	in.Account = strings.TrimSpace(strings.ToLower(in.Account))
	in.ActorID = shared.ID(strings.TrimSpace(in.ActorID.String()))
	in.IP = strings.TrimSpace(in.IP)
	in.UserAgent = strings.TrimSpace(in.UserAgent)
	in.FailureReason = strings.TrimSpace(in.FailureReason)
	in.SessionID = strings.TrimSpace(in.SessionID)
	in.Trace = normalizeTrace(in.Trace)
	return in
}

func validateLoginLogInput(in LoginLogInput) error {
	if in.ID.IsZero() {
		return fmt.Errorf("login log id is required")
	}
	if in.Account == "" {
		return fmt.Errorf("login account is required")
	}
	if err := in.Result.Validate(); err != nil {
		return err
	}
	if in.Result == LoginResultSuccess && in.SessionID == "" {
		return fmt.Errorf("login session id is required for success")
	}
	if in.Result == LoginResultFailure && in.FailureReason == "" {
		return fmt.Errorf("login failure reason is required")
	}
	if in.OccurredAt.IsZero() {
		return fmt.Errorf("login log occurred time is required")
	}
	return nil
}
