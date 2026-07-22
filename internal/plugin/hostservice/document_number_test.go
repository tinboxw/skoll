package hostservice

import (
	"context"
	"errors"
	"testing"
	"time"

	documentnumbersvc "github.com/tinboxw/skoll/internal/service/documentnumber"
	"github.com/tinboxw/skoll/pkg/pluginsdk"
)

type documentNumberTestBackend struct {
	issueError error
	issues     int
}

func (b *documentNumberTestBackend) Preview(context.Context, string, pluginsdk.DocumentNumberInput) (pluginsdk.DocumentNumberResult, error) {
	return pluginsdk.DocumentNumberResult{Number: "PO-202607-000001", PeriodKey: "202607", Sequence: 1}, nil
}

func (b *documentNumberTestBackend) Issue(context.Context, string, pluginsdk.DocumentNumberInput) (pluginsdk.DocumentNumberResult, error) {
	if b.issueError != nil {
		return pluginsdk.DocumentNumberResult{}, b.issueError
	}
	b.issues++
	return pluginsdk.DocumentNumberResult{
		Number: "PO-202607-000001", PeriodKey: "202607", Sequence: 1, Duplicate: b.issues > 1,
	}, nil
}

type documentNumberTestScopes struct {
	tenantID string
}

func (s documentNumberTestScopes) Resolve(context.Context, pluginsdk.Permission) (pluginsdk.ScopePredicate, error) {
	return pluginsdk.NewScopePredicate(pluginsdk.TrustedScope{
		SubjectID: "user-1", TenantIDs: []string{s.tenantID}, AllOwners: true, AllOrganizations: true,
	})
}

type documentNumberTestAudit struct {
	entries []pluginsdk.AuditEntry
}

func (a *documentNumberTestAudit) Record(_ context.Context, entry pluginsdk.AuditEntry) (pluginsdk.AuditReceipt, error) {
	a.entries = append(a.entries, entry)
	return pluginsdk.AuditReceipt{ID: "audit-1"}, nil
}

func TestDocumentNumberHostEnforcesTenantScopeAndAuditsFirstIssue(t *testing.T) {
	backend := &documentNumberTestBackend{}
	audit := &documentNumberTestAudit{}
	service, err := NewDocumentNumberService("pharma_oa", backend, documentNumberTestScopes{tenantID: "tenant-a"}, audit)
	if err != nil {
		t.Fatal(err)
	}
	input := documentNumberHostTestInput()
	if preview, previewErr := service.Preview(context.Background(), input); previewErr != nil || preview.Sequence != 1 {
		t.Fatalf("preview=%+v err=%v", preview, previewErr)
	}
	first, err := service.Issue(context.Background(), input)
	if err != nil || first.Duplicate {
		t.Fatalf("first=%+v err=%v", first, err)
	}
	duplicate, err := service.Issue(context.Background(), input)
	if err != nil || !duplicate.Duplicate {
		t.Fatalf("duplicate=%+v err=%v", duplicate, err)
	}
	if len(audit.entries) != 1 || audit.entries[0].Action != "document_number.issue" || audit.entries[0].ResourceID != first.Number {
		t.Fatalf("audit entries=%+v", audit.entries)
	}

	denied := input
	denied.TenantID = "tenant-b"
	_, err = service.Issue(context.Background(), denied)
	var numberErr *pluginsdk.DocumentNumberError
	if !errors.As(err, &numberErr) || numberErr.Code != pluginsdk.DocumentNumberErrorForbidden || numberErr.Field != "tenantId" {
		t.Fatalf("denied error=%v", err)
	}
	if backend.issues != 2 {
		t.Fatalf("denied issue reached backend: issues=%d", backend.issues)
	}
}

func TestDocumentNumberHostMapsStableContractErrors(t *testing.T) {
	tests := []struct {
		name  string
		cause error
		code  pluginsdk.DocumentNumberErrorCode
		field string
	}{
		{name: "transaction", cause: documentnumbersvc.ErrTransactionRequired, code: pluginsdk.DocumentNumberErrorTransactionRequired, field: "transaction"},
		{name: "idempotency", cause: documentnumbersvc.ErrConflict, code: pluginsdk.DocumentNumberErrorConflict, field: "idempotencyKey"},
		{name: "rule", cause: documentnumbersvc.ErrRuleConflict, code: pluginsdk.DocumentNumberErrorConflict, field: "rule"},
		{name: "exhausted", cause: documentnumbersvc.ErrExhausted, code: pluginsdk.DocumentNumberErrorExhausted, field: "rule.width"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			backend := &documentNumberTestBackend{issueError: test.cause}
			service, err := NewDocumentNumberService("pharma_oa", backend, documentNumberTestScopes{tenantID: "tenant-a"}, &documentNumberTestAudit{})
			if err != nil {
				t.Fatal(err)
			}
			_, err = service.Issue(context.Background(), documentNumberHostTestInput())
			var numberErr *pluginsdk.DocumentNumberError
			if !errors.As(err, &numberErr) || numberErr.Code != test.code || numberErr.Field != test.field {
				t.Fatalf("error=%v", err)
			}
		})
	}
}

func documentNumberHostTestInput() pluginsdk.DocumentNumberInput {
	return pluginsdk.DocumentNumberInput{
		Rule: pluginsdk.DocumentNumberRule{
			DocumentType: "purchase_order", Prefix: "PO", Separator: "-", Period: pluginsdk.DocumentNumberPeriodMonth,
			Width: 6, Start: 1, GapPolicy: pluginsdk.DocumentNumberGapTransactional,
		},
		TenantID: "tenant-a", Permission: pluginsdk.Permission{Resource: "pharma_oa.purchase_order", Action: "issue"},
		OccurredAt: time.Date(2026, 7, 22, 0, 0, 0, 0, time.UTC), IdempotencyKey: "purchase-order-1",
	}
}
