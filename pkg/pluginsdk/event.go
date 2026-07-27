package pluginsdk

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	MaxEventPayloadFields   = 64
	MaxEventPayloadBytes    = 64 << 10
	MaxEventSchemaVersions  = 16
	MaxEventCorrelationSize = 128
)

var (
	eventNamePattern        = regexp.MustCompile(`^[a-z][a-z0-9_.-]{2,127}$`)
	eventPublisherPattern   = regexp.MustCompile(`^[a-z][a-z0-9_-]{1,62}$`)
	eventPayloadTypePattern = regexp.MustCompile(`^[a-z][a-z0-9_.-]{2,127}$`)
	eventIdentityPattern    = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)
)

type EventService interface {
	Publish(ctx context.Context, publication EventPublication) (EventEnvelope, error)
}

type EventScopeMode string

const (
	EventScopeTenant EventScopeMode = "tenant"
	EventScopeGlobal EventScopeMode = "global"
)

type EventScope struct {
	TenantID       string `json:"tenantId,omitempty"`
	OrganizationID string `json:"organizationId,omitempty"`
	OwnerID        string `json:"ownerId,omitempty"`
}

func (s EventScope) Validate(mode EventScopeMode) error {
	switch mode {
	case EventScopeTenant:
		if !eventIdentityPattern.MatchString(strings.TrimSpace(s.TenantID)) {
			return NewEventError(EventErrorScopeMismatch, "scope.tenantId", "tenant-scoped event requires a valid tenant identity", false)
		}
	case EventScopeGlobal:
		if strings.TrimSpace(s.TenantID) != "" || strings.TrimSpace(s.OrganizationID) != "" || strings.TrimSpace(s.OwnerID) != "" {
			return NewEventError(EventErrorScopeMismatch, "scope", "global event cannot carry tenant scope", false)
		}
	default:
		return invalidEventContract("scopeMode", "event scope mode is unsupported")
	}
	for field, value := range map[string]string{
		"scope.organizationId": s.OrganizationID,
		"scope.ownerId":        s.OwnerID,
	} {
		if value != "" && !eventIdentityPattern.MatchString(strings.TrimSpace(value)) {
			return invalidEventContract(field, "event scope identity is invalid")
		}
	}
	return nil
}

type EventSubject struct {
	Type string `json:"type,omitempty"`
	ID   string `json:"id,omitempty"`
}

func (s EventSubject) Validate() error {
	if (strings.TrimSpace(s.Type) == "") != (strings.TrimSpace(s.ID) == "") {
		return invalidEventContract("subject", "event subject type and identity must be supplied together")
	}
	if s.Type != "" && !eventPayloadTypePattern.MatchString(strings.TrimSpace(s.Type)) {
		return invalidEventContract("subject.type", "event subject type is invalid")
	}
	if s.ID != "" && !eventIdentityPattern.MatchString(strings.TrimSpace(s.ID)) {
		return invalidEventContract("subject.id", "event subject identity is invalid")
	}
	return nil
}

// EventPayload uses explicit DataValue wire types so values remain lossless and
// deterministic across plugin processes.
type EventPayload map[string]DataValue

func (p EventPayload) Validate() error {
	if len(p) > MaxEventPayloadFields {
		return NewEventError(EventErrorPayloadLimit, "payload", "event payload field limit exceeded", false)
	}
	keys := make([]string, 0, len(p))
	for key := range p {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if !dataIdentifierPattern.MatchString(key) {
			return invalidEventContract("payload."+key, "event payload field is invalid")
		}
		value := p[key]
		if err := value.Validate(); err != nil {
			return invalidEventContract("payload."+key, err.Error())
		}
	}
	encoded, err := json.Marshal(p)
	if err != nil {
		return invalidEventContract("payload", "event payload cannot be encoded")
	}
	if len(encoded) > MaxEventPayloadBytes {
		return NewEventError(EventErrorPayloadLimit, "payload", "event payload size limit exceeded", false)
	}
	return nil
}

type EventPublicationDeclaration struct {
	Name          string         `json:"name"`
	SchemaVersion uint32         `json:"schemaVersion"`
	PayloadType   string         `json:"payloadType"`
	Scope         EventScopeMode `json:"scope"`
}

func (d EventPublicationDeclaration) Validate() error {
	if !eventNamePattern.MatchString(strings.TrimSpace(d.Name)) {
		return invalidEventContract("publication.name", "event name is invalid")
	}
	if d.SchemaVersion == 0 {
		return invalidEventContract("publication.schemaVersion", "event schema version must be positive")
	}
	if !eventPayloadTypePattern.MatchString(strings.TrimSpace(d.PayloadType)) {
		return invalidEventContract("publication.payloadType", "event payload type is invalid")
	}
	if d.Scope != EventScopeTenant && d.Scope != EventScopeGlobal {
		return invalidEventContract("publication.scope", "event scope mode is unsupported")
	}
	return nil
}

type EventSubscriptionDeclaration struct {
	Publisher      string   `json:"publisher"`
	Name           string   `json:"name"`
	SchemaVersions []uint32 `json:"schemaVersions"`
	Handler        string   `json:"handler"`
	RetryPolicy    string   `json:"retryPolicy"`
}

func (d EventSubscriptionDeclaration) Validate() error {
	if !eventPublisherPattern.MatchString(strings.TrimSpace(d.Publisher)) {
		return invalidEventContract("subscription.publisher", "event publisher is invalid")
	}
	if !eventNamePattern.MatchString(strings.TrimSpace(d.Name)) {
		return invalidEventContract("subscription.name", "event name is invalid")
	}
	if strings.TrimSpace(d.Handler) == "" {
		return invalidEventContract("subscription.handler", "event handler is required")
	}
	if len(d.SchemaVersions) == 0 || len(d.SchemaVersions) > MaxEventSchemaVersions {
		return invalidEventContract("subscription.schemaVersions", "event schema versions are outside the supported range")
	}
	seen := map[uint32]struct{}{}
	for _, version := range d.SchemaVersions {
		if version == 0 {
			return invalidEventContract("subscription.schemaVersions", "event schema version must be positive")
		}
		if _, ok := seen[version]; ok {
			return invalidEventContract("subscription.schemaVersions", "event schema version is duplicated")
		}
		seen[version] = struct{}{}
	}
	switch strings.TrimSpace(strings.ToLower(d.RetryPolicy)) {
	case "none", "standard", "aggressive":
	default:
		return invalidEventContract("subscription.retryPolicy", "event retry policy is unsupported")
	}
	return nil
}

type EventPublication struct {
	IdempotencyKey string       `json:"idempotencyKey"`
	Name           string       `json:"name"`
	SchemaVersion  uint32       `json:"schemaVersion"`
	Scope          EventScope   `json:"scope"`
	CorrelationID  string       `json:"correlationId"`
	CausationID    string       `json:"causationId,omitempty"`
	Subject        EventSubject `json:"subject,omitempty"`
	Payload        EventPayload `json:"payload"`
}

func (p EventPublication) Validate(declaration EventPublicationDeclaration) error {
	if err := declaration.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(p.Name) != strings.TrimSpace(declaration.Name) {
		return NewEventError(EventErrorUndeclaredPublication, "name", "event publication is not declared", false)
	}
	if p.SchemaVersion != declaration.SchemaVersion {
		return NewEventError(EventErrorUnsupportedSchema, "schemaVersion", "event schema version is not declared", false)
	}
	if !eventIdentityPattern.MatchString(strings.TrimSpace(p.IdempotencyKey)) {
		return invalidEventContract("idempotencyKey", "event idempotency key is invalid")
	}
	if !eventIdentityPattern.MatchString(strings.TrimSpace(p.CorrelationID)) {
		return invalidEventContract("correlationId", "event correlation identity is invalid")
	}
	if len(p.CorrelationID) > MaxEventCorrelationSize || len(p.CausationID) > MaxEventCorrelationSize {
		return invalidEventContract("correlationId", "event correlation identity exceeds size limit")
	}
	if p.CausationID != "" && !eventIdentityPattern.MatchString(strings.TrimSpace(p.CausationID)) {
		return invalidEventContract("causationId", "event causation identity is invalid")
	}
	if err := p.Scope.Validate(declaration.Scope); err != nil {
		return err
	}
	if err := p.Subject.Validate(); err != nil {
		return err
	}
	return p.Payload.Validate()
}

type EventEnvelope struct {
	ID            string       `json:"id"`
	Publisher     string       `json:"publisher"`
	Name          string       `json:"name"`
	SchemaVersion uint32       `json:"schemaVersion"`
	PayloadType   string       `json:"payloadType"`
	Scope         EventScope   `json:"scope"`
	CorrelationID string       `json:"correlationId"`
	CausationID   string       `json:"causationId,omitempty"`
	Subject       EventSubject `json:"subject,omitempty"`
	Payload       EventPayload `json:"payload"`
	OccurredAt    time.Time    `json:"occurredAt"`
}

func (e EventEnvelope) Validate(declaration EventPublicationDeclaration) error {
	if !eventIdentityPattern.MatchString(strings.TrimSpace(e.ID)) {
		return invalidEventContract("id", "event identity is invalid")
	}
	if !eventPublisherPattern.MatchString(strings.TrimSpace(e.Publisher)) {
		return invalidEventContract("publisher", "event publisher is invalid")
	}
	if strings.TrimSpace(e.PayloadType) != strings.TrimSpace(declaration.PayloadType) {
		return NewEventError(EventErrorUnsupportedSchema, "payloadType", "event payload type does not match declaration", false)
	}
	if e.OccurredAt.IsZero() {
		return invalidEventContract("occurredAt", "event occurrence time is required")
	}
	return (EventPublication{
		IdempotencyKey: e.ID,
		Name:           e.Name, SchemaVersion: e.SchemaVersion, Scope: e.Scope,
		CorrelationID: e.CorrelationID, CausationID: e.CausationID,
		Subject: e.Subject, Payload: e.Payload,
	}).Validate(declaration)
}

type EventErrorCode string

const (
	EventErrorInvalidRequest         EventErrorCode = "invalid_request"
	EventErrorUndeclaredPublication  EventErrorCode = "undeclared_publication"
	EventErrorUndeclaredSubscription EventErrorCode = "undeclared_subscription"
	EventErrorUnsupportedSchema      EventErrorCode = "unsupported_schema_version"
	EventErrorScopeMismatch          EventErrorCode = "scope_mismatch"
	EventErrorPayloadLimit           EventErrorCode = "payload_limit_exceeded"
	EventErrorConflict               EventErrorCode = "conflict"
	EventErrorTransactionRequired    EventErrorCode = "transaction_required"
	EventErrorUnavailable            EventErrorCode = "unavailable"
)

type EventError struct {
	Code      EventErrorCode `json:"code"`
	Field     string         `json:"field,omitempty"`
	Message   string         `json:"message"`
	Retryable bool           `json:"retryable"`
}

func (e *EventError) Error() string {
	if e == nil {
		return ""
	}
	if e.Field == "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}
	return fmt.Sprintf("%s: %s: %s", e.Code, e.Field, e.Message)
}

func NewEventError(code EventErrorCode, field, message string, retryable bool) *EventError {
	return &EventError{Code: code, Field: field, Message: message, Retryable: retryable}
}

func invalidEventContract(field, message string) error {
	return NewEventError(EventErrorInvalidRequest, field, message, false)
}
