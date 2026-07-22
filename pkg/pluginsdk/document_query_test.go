package pluginsdk

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestDocumentQueryContractsValidateSearchPrintAndExport(t *testing.T) {
	permission := Permission{Resource: "medical_oa.approval_request", Action: "read"}
	sensitive := Permission{Resource: "medical_oa.approval_request_sensitive", Action: "read"}
	from := time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC)
	to := from.Add(24 * time.Hour)
	search := DocumentSearchInput{
		TenantID: "tenant-a", Permission: permission, Types: []string{"approval_request"}, States: []string{"pending"}, Text: "Supplier",
		CreatedBy: []string{"user-1"}, CreatedFrom: &from, CreatedTo: &to, SortField: DocumentSearchSortUpdatedAt, Direction: DocumentSearchDescending, Limit: MaxDocumentSearchPage,
	}
	if err := search.Validate(); err != nil {
		t.Fatalf("valid document search rejected: %v", err)
	}
	printInput := DocumentPrintInput{TenantID: "tenant-a", Permission: permission, DocumentID: "document-1", IncludeSensitive: true, SensitivePermission: sensitive}
	if err := printInput.Validate(); err != nil {
		t.Fatalf("valid print input rejected: %v", err)
	}
	export := DocumentExportInput{
		JobID: "export-1", IdempotencyKey: "export-1", Search: search, Format: DocumentExportCSV, MaxRows: MaxDocumentExportRows,
		IncludeSensitive: true, SensitivePermission: sensitive,
	}
	if err := export.Validate(); err != nil {
		t.Fatalf("valid export input rejected: %v", err)
	}
}

func TestDocumentQueryContractsRejectUnboundedAndAmbiguousRequests(t *testing.T) {
	permission := Permission{Resource: "medical_oa.approval_request", Action: "read"}
	base := DocumentSearchInput{TenantID: "tenant-a", Permission: permission}
	tests := []struct {
		name  string
		input interface{ Validate() error }
		field string
	}{
		{name: "duplicate type", input: DocumentSearchInput{TenantID: "tenant-a", Permission: permission, Types: []string{"request", "request"}}, field: "types"},
		{name: "long text", input: DocumentSearchInput{TenantID: "tenant-a", Permission: permission, Text: strings.Repeat("x", MaxDocumentSearchText+1)}, field: "text"},
		{name: "large page", input: DocumentSearchInput{TenantID: "tenant-a", Permission: permission, Limit: MaxDocumentSearchPage + 1}, field: "limit"},
		{name: "long cursor", input: DocumentSearchInput{TenantID: "tenant-a", Permission: permission, Cursor: strings.Repeat("x", MaxDocumentSearchCursor+1)}, field: "cursor"},
		{name: "sensitive flag missing permission", input: DocumentPrintInput{TenantID: "tenant-a", Permission: permission, DocumentID: "document-1", IncludeSensitive: true}, field: "sensitivePermission"},
		{name: "unexpected sensitive permission", input: DocumentPrintInput{TenantID: "tenant-a", Permission: permission, DocumentID: "document-1", SensitivePermission: permission}, field: "sensitivePermission"},
		{name: "export starts from cursor", input: DocumentExportInput{JobID: "export-1", IdempotencyKey: "export-1", Search: func() DocumentSearchInput { value := base; value.Cursor = "cursor"; return value }(), Format: DocumentExportCSV, MaxRows: 100}, field: "search.cursor"},
		{name: "export row limit", input: DocumentExportInput{JobID: "export-1", IdempotencyKey: "export-1", Search: base, Format: DocumentExportCSV, MaxRows: MaxDocumentExportRows + 1}, field: "maxRows"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var contractErr *DocumentWorkflowError
			err := test.input.Validate()
			if !errors.As(err, &contractErr) || contractErr.Code != DocumentWorkflowErrorInvalidRequest || contractErr.Field != test.field {
				t.Fatalf("validation error=%v", err)
			}
		})
	}
}

func TestDocumentQueryResponsesRejectDuplicatesAndInvalidContinuation(t *testing.T) {
	now := time.Date(2026, time.July, 22, 8, 0, 0, 0, time.UTC)
	summary := DocumentSummary{
		ID: "document-1", Type: "approval_request", Number: "OA-001", Title: "Approval", State: "pending", Version: 1,
		CreatedAt: now, UpdatedAt: now, CreatedBy: "user-1", UpdatedBy: "user-1",
	}
	if err := (DocumentSearchPage{Items: []DocumentSummary{summary}, HasMore: true}).Validate(); err == nil {
		t.Fatal("continuation without cursor was accepted")
	}
	if err := (DocumentSearchPage{Items: []DocumentSummary{summary, summary}}).Validate(); err == nil {
		t.Fatal("duplicate summaries were accepted")
	}
}
