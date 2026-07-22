package pluginsdk

import (
	"errors"
	"testing"
	"time"
)

func TestDocumentNumberContracts(t *testing.T) {
	input := validDocumentNumberInput()
	if err := input.Validate(true); err != nil {
		t.Fatalf("validate issue input: %v", err)
	}
	if err := input.Validate(false); err != nil {
		t.Fatalf("validate preview input: %v", err)
	}
	result := DocumentNumberResult{Number: "PO-202607-000001", PeriodKey: "202607", Sequence: 1}
	if err := result.Validate(); err != nil {
		t.Fatalf("validate result: %v", err)
	}
}

func TestDocumentNumberContractsRejectInvalidRulesAndInputs(t *testing.T) {
	tests := []struct {
		name  string
		field string
		alter func(*DocumentNumberInput)
	}{
		{name: "document type", field: "rule.documentType", alter: func(in *DocumentNumberInput) { in.Rule.DocumentType = "Purchase" }},
		{name: "prefix", field: "rule.prefix", alter: func(in *DocumentNumberInput) { in.Rule.Prefix = "po" }},
		{name: "separator", field: "rule.separator", alter: func(in *DocumentNumberInput) { in.Rule.Separator = "." }},
		{name: "period", field: "rule.period", alter: func(in *DocumentNumberInput) { in.Rule.Period = "quarter" }},
		{name: "width", field: "rule.width", alter: func(in *DocumentNumberInput) { in.Rule.Width = 19 }},
		{name: "start", field: "rule.start", alter: func(in *DocumentNumberInput) { in.Rule.Start = 1000000 }},
		{name: "gap policy", field: "rule.gapPolicy", alter: func(in *DocumentNumberInput) { in.Rule.GapPolicy = "consume" }},
		{name: "tenant", field: "tenantId", alter: func(in *DocumentNumberInput) { in.TenantID = " tenant-a" }},
		{name: "permission", field: "permission", alter: func(in *DocumentNumberInput) { in.Permission = Permission{} }},
		{name: "timestamp", field: "occurredAt", alter: func(in *DocumentNumberInput) { in.OccurredAt = time.Time{} }},
		{name: "timezone", field: "occurredAt", alter: func(in *DocumentNumberInput) {
			in.OccurredAt = time.Date(2026, 7, 22, 16, 0, 0, 0, time.FixedZone("CST", 8*60*60))
		}},
		{name: "idempotency", field: "idempotencyKey", alter: func(in *DocumentNumberInput) { in.IdempotencyKey = "" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := validDocumentNumberInput()
			test.alter(&input)
			var numberErr *DocumentNumberError
			if err := input.Validate(true); !errors.As(err, &numberErr) || numberErr.Field != test.field || numberErr.Code != DocumentNumberErrorInvalidRequest {
				t.Fatalf("error=%v field=%v", err, numberErr)
			}
		})
	}
}

func TestDocumentNumberResultRejectsInvalidPeriodKey(t *testing.T) {
	for _, periodKey := range []string{"202", "2026-07", "abcdefgh", "202607220"} {
		result := DocumentNumberResult{Number: "PO-000001", PeriodKey: periodKey, Sequence: 1}
		var numberErr *DocumentNumberError
		if err := result.Validate(); !errors.As(err, &numberErr) || numberErr.Field != "periodKey" {
			t.Fatalf("periodKey=%q error=%v", periodKey, err)
		}
	}
}

func validDocumentNumberInput() DocumentNumberInput {
	return DocumentNumberInput{
		Rule: DocumentNumberRule{
			DocumentType: "purchase_order", Prefix: "PO", Separator: "-", Period: DocumentNumberPeriodMonth,
			Width: 6, Start: 1, GapPolicy: DocumentNumberGapTransactional,
		},
		TenantID: "tenant-a", Permission: Permission{Resource: "medical_oa.purchase_order", Action: "issue"},
		OccurredAt: time.Date(2026, 7, 22, 8, 0, 0, 0, time.UTC), IdempotencyKey: "purchase-1.issue",
	}
}
