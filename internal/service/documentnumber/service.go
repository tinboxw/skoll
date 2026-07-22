package documentnumber

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

var (
	ErrConflict            = errors.New("document number idempotency conflict")
	ErrRuleConflict        = errors.New("document number rule conflicts with the active sequence")
	ErrTransactionRequired = errors.New("document number transaction is required")
	ErrExhausted           = errors.New("document number sequence is exhausted")
)

type SequenceKey struct {
	PluginID     string
	TenantID     string
	DocumentType string
	PeriodKey    string
}

type Reservation struct {
	Sequence    int64
	Number      string
	PeriodKey   string
	Duplicate   bool
	RequestHash string
	IssuedAt    time.Time
}

type ReservationRequest struct {
	Key            SequenceKey
	Rule           pluginsdk.DocumentNumberRule
	RuleHash       string
	IdempotencyKey string
	RequestHash    string
	OccurredAt     time.Time
}

type Sequence struct {
	LastValue int64
	RuleHash  string
}

type Repository interface {
	Last(ctx context.Context, key SequenceKey) (Sequence, bool, error)
	Reserve(ctx context.Context, request ReservationRequest) (Reservation, error)
}

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	if repository == nil {
		panic("document number repository is required")
	}
	return &Service{repository: repository}
}

func (s *Service) Preview(ctx context.Context, pluginID string, input pluginsdk.DocumentNumberInput) (pluginsdk.DocumentNumberResult, error) {
	if err := input.Validate(false); err != nil {
		return pluginsdk.DocumentNumberResult{}, err
	}
	key, err := sequenceKey(pluginID, input)
	if err != nil {
		return pluginsdk.DocumentNumberResult{}, err
	}
	sequence, exists, err := s.repository.Last(ctx, key)
	if err != nil {
		return pluginsdk.DocumentNumberResult{}, err
	}
	next := input.Rule.Start
	if exists {
		if sequence.RuleHash != ruleHash(input.Rule) {
			return pluginsdk.DocumentNumberResult{}, ErrRuleConflict
		}
		if sequence.LastValue == maxSequence(input.Rule.Width) {
			return pluginsdk.DocumentNumberResult{}, ErrExhausted
		}
		next = sequence.LastValue + 1
	}
	number, err := Format(input.Rule, key.PeriodKey, next)
	if err != nil {
		return pluginsdk.DocumentNumberResult{}, err
	}
	return pluginsdk.DocumentNumberResult{Number: number, PeriodKey: key.PeriodKey, Sequence: next}, nil
}

func (s *Service) Issue(ctx context.Context, pluginID string, input pluginsdk.DocumentNumberInput) (pluginsdk.DocumentNumberResult, error) {
	if err := input.Validate(true); err != nil {
		return pluginsdk.DocumentNumberResult{}, err
	}
	key, err := sequenceKey(pluginID, input)
	if err != nil {
		return pluginsdk.DocumentNumberResult{}, err
	}
	hash, err := requestHash(input, key.PeriodKey)
	if err != nil {
		return pluginsdk.DocumentNumberResult{}, err
	}
	reservation, err := s.repository.Reserve(ctx, ReservationRequest{
		Key: key, Rule: input.Rule, RuleHash: ruleHash(input.Rule), IdempotencyKey: input.IdempotencyKey,
		RequestHash: hash, OccurredAt: input.OccurredAt.UTC(),
	})
	if err != nil {
		return pluginsdk.DocumentNumberResult{}, err
	}
	return pluginsdk.DocumentNumberResult{
		Number: reservation.Number, PeriodKey: reservation.PeriodKey,
		Sequence: reservation.Sequence, Duplicate: reservation.Duplicate,
	}, nil
}

func ruleHash(rule pluginsdk.DocumentNumberRule) string {
	raw, _ := json.Marshal(rule)
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:])
}

func Format(rule pluginsdk.DocumentNumberRule, periodKey string, sequence int64) (string, error) {
	if err := rule.Validate(); err != nil {
		return "", err
	}
	if sequence < rule.Start || sequence > maxSequence(rule.Width) {
		return "", ErrExhausted
	}
	parts := []string{rule.Prefix}
	if periodKey != "" {
		parts = append(parts, periodKey)
	}
	parts = append(parts, fmt.Sprintf("%0*d", rule.Width, sequence))
	return strings.Join(parts, rule.Separator), nil
}

func PeriodKey(period pluginsdk.DocumentNumberPeriod, occurredAt time.Time) (string, error) {
	if occurredAt.IsZero() {
		return "", fmt.Errorf("document number business timestamp is required")
	}
	occurredAt = occurredAt.UTC()
	switch period {
	case pluginsdk.DocumentNumberPeriodNone:
		return "", nil
	case pluginsdk.DocumentNumberPeriodYear:
		return occurredAt.Format("2006"), nil
	case pluginsdk.DocumentNumberPeriodMonth:
		return occurredAt.Format("200601"), nil
	case pluginsdk.DocumentNumberPeriodDay:
		return occurredAt.Format("20060102"), nil
	default:
		return "", fmt.Errorf("document number period is unsupported")
	}
}

func sequenceKey(pluginID string, input pluginsdk.DocumentNumberInput) (SequenceKey, error) {
	pluginID = strings.ToLower(strings.TrimSpace(pluginID))
	if pluginID == "" || len(pluginID) > 64 {
		return SequenceKey{}, fmt.Errorf("document number plugin identity is invalid")
	}
	periodKey, err := PeriodKey(input.Rule.Period, input.OccurredAt)
	if err != nil {
		return SequenceKey{}, err
	}
	return SequenceKey{PluginID: pluginID, TenantID: input.TenantID, DocumentType: input.Rule.DocumentType, PeriodKey: periodKey}, nil
}

func requestHash(input pluginsdk.DocumentNumberInput, periodKey string) (string, error) {
	payload := struct {
		Rule       pluginsdk.DocumentNumberRule `json:"rule"`
		TenantID   string                       `json:"tenantId"`
		Permission pluginsdk.Permission         `json:"permission"`
		PeriodKey  string                       `json:"periodKey"`
	}{Rule: input.Rule, TenantID: input.TenantID, Permission: input.Permission, PeriodKey: periodKey}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:]), nil
}

func maxSequence(width int) int64 {
	value := int64(1)
	for range width {
		value *= 10
	}
	return value - 1
}
