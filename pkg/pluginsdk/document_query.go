package pluginsdk

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	MaxDocumentSearchTerms  = 20
	MaxDocumentSearchText   = 128
	MaxDocumentSearchPage   = 200
	MaxDocumentSearchCursor = 2048
	MaxDocumentExportRows   = 50000
	DocumentExportJobKind   = "document_export"
)

type DocumentSearchSortField string

const (
	DocumentSearchSortUpdatedAt DocumentSearchSortField = "updatedAt"
	DocumentSearchSortCreatedAt DocumentSearchSortField = "createdAt"
	DocumentSearchSortNumber    DocumentSearchSortField = "number"
)

type DocumentSearchDirection string

const (
	DocumentSearchAscending  DocumentSearchDirection = "asc"
	DocumentSearchDescending DocumentSearchDirection = "desc"
)

type DocumentSearchInput struct {
	TenantID    string                  `json:"tenantId"`
	Permission  Permission              `json:"permission"`
	Types       []string                `json:"types,omitempty"`
	States      []string                `json:"states,omitempty"`
	Text        string                  `json:"text,omitempty"`
	CreatedBy   []string                `json:"createdBy,omitempty"`
	CreatedFrom *time.Time              `json:"createdFrom,omitempty"`
	CreatedTo   *time.Time              `json:"createdTo,omitempty"`
	SortField   DocumentSearchSortField `json:"sortField,omitempty"`
	Direction   DocumentSearchDirection `json:"direction,omitempty"`
	Cursor      string                  `json:"cursor,omitempty"`
	Limit       int                     `json:"limit,omitempty"`
}

type DocumentSummary struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Number    string    `json:"number"`
	Title     string    `json:"title"`
	State     string    `json:"state"`
	Version   int64     `json:"version"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	CreatedBy string    `json:"createdBy"`
	UpdatedBy string    `json:"updatedBy"`
	Tags      []string  `json:"tags,omitempty"`
}

type DocumentSearchPage struct {
	Items      []DocumentSummary `json:"items"`
	NextCursor string            `json:"nextCursor,omitempty"`
	HasMore    bool              `json:"hasMore"`
}

type DocumentPrintInput struct {
	TenantID            string     `json:"tenantId"`
	Permission          Permission `json:"permission"`
	DocumentID          string     `json:"documentId"`
	IncludeSensitive    bool       `json:"includeSensitive,omitempty"`
	SensitivePermission Permission `json:"sensitivePermission,omitempty"`
}

type DocumentPrintPayload struct {
	Schema         DocumentSchema   `json:"schema"`
	Document       DocumentRecord   `json:"document"`
	Workflow       WorkflowInstance `json:"workflow"`
	RedactedFields []string         `json:"redactedFields,omitempty"`
	GeneratedAt    time.Time        `json:"generatedAt"`
}

type DocumentExportFormat string

const DocumentExportCSV DocumentExportFormat = "csv"

type DocumentExportInput struct {
	JobID               string               `json:"jobId"`
	IdempotencyKey      string               `json:"idempotencyKey"`
	Search              DocumentSearchInput  `json:"search"`
	Format              DocumentExportFormat `json:"format"`
	MaxRows             int                  `json:"maxRows"`
	IncludeSensitive    bool                 `json:"includeSensitive,omitempty"`
	SensitivePermission Permission           `json:"sensitivePermission,omitempty"`
}

type DocumentExportPlan struct {
	Version             int                  `json:"version"`
	Search              DocumentSearchInput  `json:"search"`
	Format              DocumentExportFormat `json:"format"`
	MaxRows             int                  `json:"maxRows"`
	SensitiveAuthorized bool                 `json:"sensitiveAuthorized"`
	Actor               WorkflowActor        `json:"actor"`
}

type DocumentQueryService interface {
	Search(context.Context, DocumentSearchInput) (DocumentSearchPage, error)
	Print(context.Context, DocumentPrintInput) (DocumentPrintPayload, error)
	Export(context.Context, DocumentExportInput) (Job, error)
}

func (in DocumentSearchInput) Validate() error {
	if err := validateDocumentWorkflowScope(in.TenantID, in.Permission); err != nil {
		return err
	}
	if err := validateDocumentSearchTerms("types", in.Types, 63); err != nil {
		return err
	}
	if err := validateDocumentSearchTerms("states", in.States, 63); err != nil {
		return err
	}
	if err := validateDocumentSearchTerms("createdBy", in.CreatedBy, 512); err != nil {
		return err
	}
	if !utf8.ValidString(in.Text) || strings.TrimSpace(in.Text) != in.Text || len([]rune(in.Text)) > MaxDocumentSearchText {
		return NewDocumentWorkflowError(DocumentWorkflowErrorInvalidRequest, "text", "document search text is invalid", false)
	}
	if err := validateDocumentSearchTime("createdFrom", in.CreatedFrom); err != nil {
		return err
	}
	if err := validateDocumentSearchTime("createdTo", in.CreatedTo); err != nil {
		return err
	}
	if in.CreatedFrom != nil && in.CreatedTo != nil && in.CreatedFrom.After(*in.CreatedTo) {
		return NewDocumentWorkflowError(DocumentWorkflowErrorInvalidRequest, "createdTo", "document search time range is invalid", false)
	}
	if in.SortField != "" && in.SortField != DocumentSearchSortUpdatedAt && in.SortField != DocumentSearchSortCreatedAt && in.SortField != DocumentSearchSortNumber {
		return NewDocumentWorkflowError(DocumentWorkflowErrorInvalidRequest, "sortField", "document search sort field is invalid", false)
	}
	if in.Direction != "" && in.Direction != DocumentSearchAscending && in.Direction != DocumentSearchDescending {
		return NewDocumentWorkflowError(DocumentWorkflowErrorInvalidRequest, "direction", "document search direction is invalid", false)
	}
	if len(in.Cursor) > MaxDocumentSearchCursor {
		return NewDocumentWorkflowError(DocumentWorkflowErrorInvalidRequest, "cursor", "document search cursor is too long", false)
	}
	if in.Limit < 0 || in.Limit > MaxDocumentSearchPage {
		return NewDocumentWorkflowError(DocumentWorkflowErrorInvalidRequest, "limit", fmt.Sprintf("document search limit must be between 0 and %d", MaxDocumentSearchPage), false)
	}
	return nil
}

func (in DocumentPrintInput) Validate() error {
	if err := validateDocumentResource(in.TenantID, in.Permission, in.DocumentID); err != nil {
		return err
	}
	return validateSensitiveDocumentPermission(in.TenantID, in.IncludeSensitive, in.SensitivePermission)
}

func (in DocumentExportInput) Validate() error {
	if err := in.Search.Validate(); err != nil {
		return err
	}
	if err := validateDocumentID("jobId", in.JobID); err != nil {
		return documentWorkflowValidationError(err)
	}
	if err := validateDocumentID("idempotencyKey", in.IdempotencyKey); err != nil {
		return documentWorkflowValidationError(err)
	}
	if in.Format != DocumentExportCSV {
		return NewDocumentWorkflowError(DocumentWorkflowErrorInvalidRequest, "format", "document export format is invalid", false)
	}
	if in.MaxRows <= 0 || in.MaxRows > MaxDocumentExportRows {
		return NewDocumentWorkflowError(DocumentWorkflowErrorInvalidRequest, "maxRows", fmt.Sprintf("document export maxRows must be between 1 and %d", MaxDocumentExportRows), false)
	}
	if in.Search.Cursor != "" {
		return NewDocumentWorkflowError(DocumentWorkflowErrorInvalidRequest, "search.cursor", "document export search must start at the first page", false)
	}
	return validateSensitiveDocumentPermission(in.Search.TenantID, in.IncludeSensitive, in.SensitivePermission)
}

func (p DocumentSearchPage) Validate() error {
	if len(p.Items) > MaxDocumentSearchPage || p.HasMore != (p.NextCursor != "") {
		return NewDocumentWorkflowError(DocumentWorkflowErrorInvalidRequest, "response", "document search page is invalid", false)
	}
	seen := map[string]struct{}{}
	for _, item := range p.Items {
		if err := item.Validate(); err != nil {
			return err
		}
		if _, exists := seen[item.ID]; exists {
			return NewDocumentWorkflowError(DocumentWorkflowErrorInvalidRequest, "response.items", "document search page contains duplicate items", false)
		}
		seen[item.ID] = struct{}{}
	}
	if len(p.Items) == 0 && p.HasMore {
		return NewDocumentWorkflowError(DocumentWorkflowErrorInvalidRequest, "response", "empty document search page cannot continue", false)
	}
	return nil
}

func (s DocumentSummary) Validate() error {
	if err := validateDocumentID("summary.id", s.ID); err != nil {
		return documentWorkflowValidationError(err)
	}
	if !documentKeyPattern.MatchString(s.Type) || !documentKeyPattern.MatchString(s.State) || s.Version <= 0 {
		return NewDocumentWorkflowError(DocumentWorkflowErrorInvalidRequest, "summary", "document summary identity is invalid", false)
	}
	if err := validateDocumentBoundedText("summary.number", s.Number, 128, true); err != nil {
		return documentWorkflowValidationError(err)
	}
	if err := validateDocumentBoundedText("summary.title", s.Title, 256, true); err != nil {
		return documentWorkflowValidationError(err)
	}
	if !validDocumentActivityTime(s.CreatedAt) || !validDocumentActivityTime(s.UpdatedAt) || s.UpdatedAt.Before(s.CreatedAt) || strings.TrimSpace(s.CreatedBy) == "" || strings.TrimSpace(s.UpdatedBy) == "" {
		return NewDocumentWorkflowError(DocumentWorkflowErrorInvalidRequest, "summary.metadata", "document summary metadata is invalid", false)
	}
	return nil
}

func (p DocumentPrintPayload) Validate() error {
	if err := p.Schema.Validate(); err != nil {
		return documentWorkflowValidationError(err)
	}
	if p.Document.ID == "" || p.Document.Type != p.Schema.Key || p.Document.SchemaVersion != p.Schema.Version || p.Workflow.BusinessID != p.Document.ID || !validDocumentActivityTime(p.GeneratedAt) {
		return NewDocumentWorkflowError(DocumentWorkflowErrorInvalidRequest, "response", "document print payload is invalid", false)
	}
	seen := map[string]struct{}{}
	for _, field := range p.RedactedFields {
		if strings.TrimSpace(field) != field || field == "" {
			return NewDocumentWorkflowError(DocumentWorkflowErrorInvalidRequest, "response.redactedFields", "redacted field path is invalid", false)
		}
		if _, exists := seen[field]; exists {
			return NewDocumentWorkflowError(DocumentWorkflowErrorInvalidRequest, "response.redactedFields", "redacted field paths must be unique", false)
		}
		seen[field] = struct{}{}
	}
	return nil
}

func (p DocumentExportPlan) Validate() error {
	if p.Version != 1 || p.Format != DocumentExportCSV || p.MaxRows <= 0 || p.MaxRows > MaxDocumentExportRows || strings.TrimSpace(p.Actor.ID) == "" {
		return NewDocumentWorkflowError(DocumentWorkflowErrorInvalidRequest, "exportPlan", "document export plan is invalid", false)
	}
	return p.Search.Validate()
}

func (p DocumentExportPlan) JSON() (json.RawMessage, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(p)
}

func validateDocumentSearchTerms(field string, values []string, maximum int) error {
	if len(values) > MaxDocumentSearchTerms {
		return NewDocumentWorkflowError(DocumentWorkflowErrorInvalidRequest, field, "document search contains too many terms", false)
	}
	seen := map[string]struct{}{}
	for _, value := range values {
		if strings.TrimSpace(value) != value || value == "" || len(value) > maximum {
			return NewDocumentWorkflowError(DocumentWorkflowErrorInvalidRequest, field, "document search term is invalid", false)
		}
		if _, exists := seen[value]; exists {
			return NewDocumentWorkflowError(DocumentWorkflowErrorInvalidRequest, field, "document search terms must be unique", false)
		}
		seen[value] = struct{}{}
	}
	return nil
}

func validateDocumentSearchTime(field string, value *time.Time) error {
	if value != nil && !validDocumentActivityTime(*value) {
		return NewDocumentWorkflowError(DocumentWorkflowErrorInvalidRequest, field, "document search timestamp must be non-zero UTC", false)
	}
	return nil
}

func validateSensitiveDocumentPermission(tenantID string, include bool, permission Permission) error {
	empty := strings.TrimSpace(permission.Resource) == "" && strings.TrimSpace(permission.Action) == ""
	if !include {
		if !empty {
			return NewDocumentWorkflowError(DocumentWorkflowErrorInvalidRequest, "sensitivePermission", "sensitive permission requires includeSensitive", false)
		}
		return nil
	}
	if empty {
		return NewDocumentWorkflowError(DocumentWorkflowErrorInvalidRequest, "sensitivePermission", "sensitive permission is required", false)
	}
	return validateDocumentWorkflowScope(tenantID, permission)
}
