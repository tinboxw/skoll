package pluginsdk

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	MaxDocumentNumberWidth          = 18
	MaxDocumentNumberIdempotencyKey = 128
)

var (
	documentNumberPrefixPattern = regexp.MustCompile(`^[A-Z][A-Z0-9]{0,15}$`)
	documentNumberTenantPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)
	documentNumberPeriodPattern = regexp.MustCompile(`^(?:[0-9]{4}|[0-9]{6}|[0-9]{8})?$`)
)

type DocumentNumberPeriod string

const (
	DocumentNumberPeriodNone  DocumentNumberPeriod = "none"
	DocumentNumberPeriodYear  DocumentNumberPeriod = "year"
	DocumentNumberPeriodMonth DocumentNumberPeriod = "month"
	DocumentNumberPeriodDay   DocumentNumberPeriod = "day"
)

type DocumentNumberGapPolicy string

const (
	// DocumentNumberGapTransactional reserves a number in the caller's transaction.
	// Rollback removes both the increment and its idempotency record.
	DocumentNumberGapTransactional DocumentNumberGapPolicy = "transactional"
)

type DocumentNumberRule struct {
	DocumentType string                  `json:"documentType"`
	Prefix       string                  `json:"prefix"`
	Separator    string                  `json:"separator"`
	Period       DocumentNumberPeriod    `json:"period"`
	Width        int                     `json:"width"`
	Start        int64                   `json:"start"`
	GapPolicy    DocumentNumberGapPolicy `json:"gapPolicy"`
}

type DocumentNumberInput struct {
	Rule           DocumentNumberRule `json:"rule"`
	TenantID       string             `json:"tenantId"`
	Permission     Permission         `json:"permission"`
	OccurredAt     time.Time          `json:"occurredAt"`
	IdempotencyKey string             `json:"idempotencyKey,omitempty"`
}

type DocumentNumberResult struct {
	Number    string `json:"number"`
	PeriodKey string `json:"periodKey"`
	Sequence  int64  `json:"sequence"`
	Duplicate bool   `json:"duplicate"`
}

type DocumentNumberService interface {
	Preview(ctx context.Context, input DocumentNumberInput) (DocumentNumberResult, error)
	Issue(ctx context.Context, input DocumentNumberInput) (DocumentNumberResult, error)
}

type DocumentNumberErrorCode string

const (
	DocumentNumberErrorInvalidRequest      DocumentNumberErrorCode = "invalid_request"
	DocumentNumberErrorForbidden           DocumentNumberErrorCode = "forbidden"
	DocumentNumberErrorConflict            DocumentNumberErrorCode = "conflict"
	DocumentNumberErrorTransactionRequired DocumentNumberErrorCode = "transaction_required"
	DocumentNumberErrorExhausted           DocumentNumberErrorCode = "exhausted"
	DocumentNumberErrorUnavailable         DocumentNumberErrorCode = "unavailable"
)

type DocumentNumberError struct {
	Code      DocumentNumberErrorCode `json:"code"`
	Field     string                  `json:"field,omitempty"`
	Message   string                  `json:"message"`
	Retryable bool                    `json:"retryable"`
}

func (e *DocumentNumberError) Error() string {
	if e == nil {
		return "document number operation failed"
	}
	if e.Field == "" {
		return fmt.Sprintf("document number operation failed: %s: %s", e.Code, e.Message)
	}
	return fmt.Sprintf("document number operation failed: %s: %s: %s", e.Code, e.Field, e.Message)
}

func NewDocumentNumberError(code DocumentNumberErrorCode, field, message string, retryable bool) *DocumentNumberError {
	return &DocumentNumberError{Code: code, Field: strings.TrimSpace(field), Message: strings.TrimSpace(message), Retryable: retryable}
}

func (r DocumentNumberRule) Validate() error {
	if !documentKeyPattern.MatchString(r.DocumentType) {
		return invalidDocumentNumber("rule.documentType", "document type must be a lowercase document key")
	}
	if !documentNumberPrefixPattern.MatchString(r.Prefix) {
		return invalidDocumentNumber("rule.prefix", "prefix must be bounded uppercase letters and digits")
	}
	if r.Separator != "-" && r.Separator != "/" && r.Separator != "_" {
		return invalidDocumentNumber("rule.separator", "separator must be one of -, /, or _")
	}
	switch r.Period {
	case DocumentNumberPeriodNone, DocumentNumberPeriodYear, DocumentNumberPeriodMonth, DocumentNumberPeriodDay:
	default:
		return invalidDocumentNumber("rule.period", "period is unsupported")
	}
	if r.Width < 1 || r.Width > MaxDocumentNumberWidth {
		return invalidDocumentNumber("rule.width", "sequence width is outside the supported range")
	}
	if r.Start < 1 || len(strconv.FormatInt(r.Start, 10)) > r.Width {
		return invalidDocumentNumber("rule.start", "sequence start does not fit the declared width")
	}
	if r.GapPolicy != DocumentNumberGapTransactional {
		return invalidDocumentNumber("rule.gapPolicy", "gap policy must be transactional")
	}
	return nil
}

func (in DocumentNumberInput) Validate(requireIdempotency bool) error {
	if err := in.Rule.Validate(); err != nil {
		return err
	}
	if !documentNumberTenantPattern.MatchString(in.TenantID) {
		return invalidDocumentNumber("tenantId", "tenant id must be a bounded identifier")
	}
	if err := (DataScopeIntent{Permission: in.Permission}).Validate(); err != nil {
		return invalidDocumentNumber("permission", "permission is invalid")
	}
	if in.OccurredAt.IsZero() {
		return invalidDocumentNumber("occurredAt", "business timestamp is required")
	}
	if _, offset := in.OccurredAt.Zone(); offset != 0 {
		return invalidDocumentNumber("occurredAt", "business timestamp must use UTC")
	}
	key := strings.TrimSpace(in.IdempotencyKey)
	if requireIdempotency && key == "" {
		return invalidDocumentNumber("idempotencyKey", "idempotency key is required for issuance")
	}
	if key != in.IdempotencyKey || len(key) > MaxDocumentNumberIdempotencyKey || strings.ContainsAny(key, "\r\n\t") {
		return invalidDocumentNumber("idempotencyKey", "idempotency key is invalid")
	}
	return nil
}

func (r DocumentNumberResult) Validate() error {
	if strings.TrimSpace(r.Number) == "" || len(r.Number) > 128 {
		return invalidDocumentNumber("number", "document number is invalid")
	}
	if !documentNumberPeriodPattern.MatchString(r.PeriodKey) {
		return invalidDocumentNumber("periodKey", "period key is invalid")
	}
	if r.Sequence < 1 {
		return invalidDocumentNumber("sequence", "sequence must be positive")
	}
	return nil
}

func invalidDocumentNumber(field, message string) error {
	return NewDocumentNumberError(DocumentNumberErrorInvalidRequest, field, message, false)
}
