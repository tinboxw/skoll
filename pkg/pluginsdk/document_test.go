package pluginsdk

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestDocumentSchemaAndRecordValidateAllBusinessValues(t *testing.T) {
	schema := validDocumentSchema()
	if err := schema.Validate(); err != nil {
		t.Fatalf("validate schema: %v", err)
	}
	record := validDocumentRecord()
	if err := schema.ValidateRecord(record); err != nil {
		t.Fatalf("validate record: %v", err)
	}
}

func TestDocumentSchemaRejectsInvalidDefinitions(t *testing.T) {
	tests := []struct {
		name  string
		field string
		alter func(*DocumentSchema)
	}{
		{name: "invalid key", field: "key", alter: func(s *DocumentSchema) { s.Key = "PurchaseOrder" }},
		{name: "duplicate header", field: "header[1].key", alter: func(s *DocumentSchema) { s.Header[1].Key = s.Header[0].Key }},
		{name: "invalid reference type", field: "header[9].referenceType", alter: func(s *DocumentSchema) { s.Header[9].ReferenceType = "Supplier" }},
		{name: "numeric rule on text", field: "header[0].rules[0].type", alter: func(s *DocumentSchema) {
			s.Header[0].Rules = []DocumentValidationRule{{Type: DocumentValidationMin, Value: "1"}}
		}},
		{name: "inverted numeric range", field: "header[2].rules", alter: func(s *DocumentSchema) {
			s.Header[2].Rules = []DocumentValidationRule{{Type: DocumentValidationMin, Value: "2"}, {Type: DocumentValidationMax, Value: "1"}}
		}},
		{name: "bad line limits", field: "lines[0]", alter: func(s *DocumentSchema) { s.Lines[0].MinItems = 101 }},
		{name: "terminal initial state", field: "initialState", alter: func(s *DocumentSchema) { s.InitialState = "approved" }},
		{name: "unknown action target", field: "actions[0].to", alter: func(s *DocumentSchema) { s.Actions[0].To = "missing" }},
		{name: "terminal action source", field: "actions[0].from[0]", alter: func(s *DocumentSchema) { s.Actions[0].From[0] = "approved" }},
		{name: "self transition", field: "actions[0].from[0]", alter: func(s *DocumentSchema) { s.Actions[0].To = "draft" }},
		{name: "non terminal dead end", field: "states.rejected", alter: func(s *DocumentSchema) { s.Actions[0].From = []string{"draft"} }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			schema := validDocumentSchema()
			test.alter(&schema)
			assertDocumentContractField(t, schema.Validate(), test.field)
		})
	}
}

func TestDocumentValuesRejectAmbiguousRepresentations(t *testing.T) {
	tests := []struct {
		name  string
		value DocumentValue
		field string
	}{
		{name: "integer leading zero", value: DocumentValue{Type: DocumentFieldInteger, Value: "01"}, field: "value.value"},
		{name: "decimal exponent", value: DocumentValue{Type: DocumentFieldDecimal, Value: "1e3"}, field: "value.value"},
		{name: "money lowercase currency", value: DocumentValue{Type: DocumentFieldMoney, Value: "12.30", Currency: "cny"}, field: "value.currency"},
		{name: "quantity missing unit", value: DocumentValue{Type: DocumentFieldQuantity, Value: "10"}, field: "value.unit"},
		{name: "boolean number", value: DocumentValue{Type: DocumentFieldBoolean, Value: "1"}, field: "value.value"},
		{name: "invalid calendar date", value: DocumentValue{Type: DocumentFieldDate, Value: "2026-02-30"}, field: "value.value"},
		{name: "non UTC datetime", value: DocumentValue{Type: DocumentFieldDateTime, Value: "2026-07-22T16:00:00+08:00"}, field: "value.value"},
		{name: "missing reference", value: DocumentValue{Type: DocumentFieldReference}, field: "value"},
		{name: "invalid json", value: DocumentValue{Type: DocumentFieldJSON, Value: "{"}, field: "value.value"},
		{name: "unrelated attribute", value: DocumentValue{Type: DocumentFieldString, Value: "ok", Currency: "CNY"}, field: "value"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertDocumentContractField(t, test.value.Validate(), test.field)
		})
	}
}

func TestDocumentRecordValidationIsStrictAndDeterministic(t *testing.T) {
	schema := validDocumentSchema()

	record := validDocumentRecord()
	record.Header["z_unknown"] = DocumentValue{Type: DocumentFieldString, Value: "z"}
	record.Header["a_unknown"] = DocumentValue{Type: DocumentFieldString, Value: "a"}
	assertDocumentContractField(t, schema.ValidateRecord(record), "header.a_unknown")

	tests := []struct {
		name  string
		field string
		alter func(*DocumentRecord)
	}{
		{name: "schema mismatch", field: "schemaVersion", alter: func(r *DocumentRecord) { r.SchemaVersion++ }},
		{name: "missing required header", field: "header.code", alter: func(r *DocumentRecord) { delete(r.Header, "code") }},
		{name: "wrong value type", field: "header.total.type", alter: func(r *DocumentRecord) { r.Header["total"] = DocumentValue{Type: DocumentFieldDecimal, Value: "12.30"} }},
		{name: "reference mismatch", field: "header.supplier.reference.type", alter: func(r *DocumentRecord) {
			r.Header["supplier"] = DocumentValue{Type: DocumentFieldReference, Reference: &DocumentReference{Type: "customer", ID: "customer-1"}}
		}},
		{name: "numeric rule", field: "header.item_count", alter: func(r *DocumentRecord) {
			r.Header["item_count"] = DocumentValue{Type: DocumentFieldInteger, Value: "-1"}
		}},
		{name: "missing required lines", field: "lines.items", alter: func(r *DocumentRecord) { r.Lines["items"] = nil }},
		{name: "duplicate line id", field: "lines.items[1].id", alter: func(r *DocumentRecord) { r.Lines["items"] = append(r.Lines["items"], r.Lines["items"][0]) }},
		{name: "unknown line group", field: "lines.extra", alter: func(r *DocumentRecord) { r.Lines["extra"] = []DocumentLine{} }},
		{name: "invalid optimistic version", field: "version", alter: func(r *DocumentRecord) { r.Version = 0 }},
		{name: "non UTC metadata", field: "metadata.updatedAt", alter: func(r *DocumentRecord) {
			r.Metadata.UpdatedAt = time.Date(2026, 7, 22, 16, 0, 0, 0, time.FixedZone("CST", 8*60*60))
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			record := validDocumentRecord()
			test.alter(&record)
			assertDocumentContractField(t, schema.ValidateRecord(record), test.field)
		})
	}
}

func TestDocumentStateTransitionsAndActionVersion(t *testing.T) {
	schema := validDocumentSchema()
	next, err := schema.NextState("draft", "submit", "")
	if err != nil || next != "pending" {
		t.Fatalf("submit next=%q err=%v", next, err)
	}
	assertDocumentContractField(t, actionError(schema, "pending", "reject", ""), "actions[2].comment")
	next, err = schema.NextState("pending", "reject", "Missing qualification")
	if err != nil || next != "rejected" {
		t.Fatalf("reject next=%q err=%v", next, err)
	}
	assertDocumentContractField(t, actionError(schema, "approved", "submit", ""), "action")

	input := DocumentActionInput{DocumentID: "purchase-1", Action: "approve", ExpectedVersion: 4, Comment: "Approved"}
	if err = input.Validate(); err != nil {
		t.Fatalf("validate action input: %v", err)
	}
	input.ExpectedVersion = 0
	assertDocumentContractField(t, input.Validate(), "expectedVersion")
}

func TestDocumentContractWireUsesCamelCaseAndExplicitValues(t *testing.T) {
	raw, err := json.Marshal(validDocumentRecord())
	if err != nil {
		t.Fatal(err)
	}
	wire := string(raw)
	for _, expected := range []string{`"schemaVersion":1`, `"createdAt":`, `"createdBy":`, `"reference":{"type":"supplier"`, `"currency":"CNY"`, `"unit":"box"`} {
		if !strings.Contains(wire, expected) {
			t.Fatalf("wire=%s missing=%s", wire, expected)
		}
	}
	for _, forbidden := range []string{`"SchemaVersion"`, `"CreatedAt"`, `"ReferenceType"`} {
		if strings.Contains(wire, forbidden) {
			t.Fatalf("wire=%s contains=%s", wire, forbidden)
		}
	}
}

func validDocumentSchema() DocumentSchema {
	return DocumentSchema{
		Key: "purchase_order", Name: "Purchase order", Version: 1, InitialState: "draft",
		Header: []DocumentFieldSchema{
			{Key: "code", Label: "Code", Type: DocumentFieldString, Required: true, Rules: []DocumentValidationRule{{Type: DocumentValidationMinLength, Value: "3"}, {Type: DocumentValidationPattern, Value: `^[A-Z0-9-]+$`}}},
			{Key: "description", Label: "Description", Type: DocumentFieldText, Rules: []DocumentValidationRule{{Type: DocumentValidationMaxLength, Value: "500"}}},
			{Key: "item_count", Label: "Item count", Type: DocumentFieldInteger, Required: true, Rules: []DocumentValidationRule{{Type: DocumentValidationMin, Value: "0"}}},
			{Key: "discount", Label: "Discount", Type: DocumentFieldDecimal, Rules: []DocumentValidationRule{{Type: DocumentValidationMin, Value: "0"}, {Type: DocumentValidationMax, Value: "1"}}},
			{Key: "total", Label: "Total", Type: DocumentFieldMoney, Required: true, Rules: []DocumentValidationRule{{Type: DocumentValidationMin, Value: "0"}}},
			{Key: "weight", Label: "Weight", Type: DocumentFieldQuantity, Rules: []DocumentValidationRule{{Type: DocumentValidationMin, Value: "0"}}},
			{Key: "urgent", Label: "Urgent", Type: DocumentFieldBoolean, Required: true},
			{Key: "issued_on", Label: "Issued on", Type: DocumentFieldDate, Required: true},
			{Key: "needed_at", Label: "Needed at", Type: DocumentFieldDateTime, Required: true},
			{Key: "supplier", Label: "Supplier", Type: DocumentFieldReference, Required: true, ReferenceType: "supplier"},
			{Key: "custom", Label: "Custom", Type: DocumentFieldJSON},
		},
		Lines: []DocumentLineSchema{{
			Key: "items", Name: "Items", MinItems: 1, MaxItems: 100,
			Fields: []DocumentFieldSchema{
				{Key: "product", Label: "Product", Type: DocumentFieldReference, Required: true, ReferenceType: "product"},
				{Key: "quantity", Label: "Quantity", Type: DocumentFieldQuantity, Required: true, Rules: []DocumentValidationRule{{Type: DocumentValidationMin, Value: "0.001"}}},
				{Key: "unit_price", Label: "Unit price", Type: DocumentFieldMoney, Required: true, Rules: []DocumentValidationRule{{Type: DocumentValidationMin, Value: "0"}}},
				{Key: "expires_on", Label: "Expires on", Type: DocumentFieldDate},
			},
		}},
		States: []DocumentStateSchema{
			{Key: "draft", Name: "Draft"}, {Key: "pending", Name: "Pending"},
			{Key: "approved", Name: "Approved", Terminal: true}, {Key: "rejected", Name: "Rejected"},
		},
		Actions: []DocumentActionSchema{
			{Key: "submit", Name: "Submit", From: []string{"draft", "rejected"}, To: "pending"},
			{Key: "approve", Name: "Approve", From: []string{"pending"}, To: "approved"},
			{Key: "reject", Name: "Reject", From: []string{"pending"}, To: "rejected", RequiresComment: true},
		},
	}
}

func validDocumentRecord() DocumentRecord {
	now := time.Date(2026, 7, 22, 8, 0, 0, 0, time.UTC)
	return DocumentRecord{
		ID: "purchase-1", Type: "purchase_order", SchemaVersion: 1, Number: "PO-20260722-0001",
		Title: "Hospital replenishment", State: "draft", Version: 1,
		Header: map[string]DocumentValue{
			"code":        {Type: DocumentFieldString, Value: "PO-001"},
			"description": {Type: DocumentFieldText, Value: "Routine purchase"},
			"item_count":  {Type: DocumentFieldInteger, Value: "1"},
			"discount":    {Type: DocumentFieldDecimal, Value: "0.05"},
			"total":       {Type: DocumentFieldMoney, Value: "1250.50", Currency: "CNY"},
			"weight":      {Type: DocumentFieldQuantity, Value: "12.5", Unit: "kg"},
			"urgent":      {Type: DocumentFieldBoolean, Value: "false"},
			"issued_on":   {Type: DocumentFieldDate, Value: "2026-07-22"},
			"needed_at":   {Type: DocumentFieldDateTime, Value: "2026-07-23T08:30:00Z"},
			"supplier":    {Type: DocumentFieldReference, Reference: &DocumentReference{Type: "supplier", ID: "supplier-1", Label: "Northwind Pharma"}},
			"custom":      {Type: DocumentFieldJSON, Value: `{"temperature":"ambient"}`},
		},
		Lines: map[string][]DocumentLine{
			"items": {{ID: "line-1", Values: map[string]DocumentValue{
				"product":    {Type: DocumentFieldReference, Reference: &DocumentReference{Type: "product", ID: "product-1", Label: "Tablet"}},
				"quantity":   {Type: DocumentFieldQuantity, Value: "10", Unit: "box"},
				"unit_price": {Type: DocumentFieldMoney, Value: "125.05", Currency: "CNY"},
				"expires_on": {Type: DocumentFieldDate, Value: "2028-12-31"},
			}}},
		},
		Metadata: DocumentMetadata{CreatedAt: now, UpdatedAt: now.Add(time.Minute), CreatedBy: "employee-1", UpdatedBy: "employee-1", Tags: []string{"purchase", "hospital"}},
	}
}

func actionError(schema DocumentSchema, state, action, comment string) error {
	_, err := schema.NextState(state, action, comment)
	return err
}

func assertDocumentContractField(t *testing.T, err error, field string) {
	t.Helper()
	var contractErr *DocumentContractError
	if !errors.As(err, &contractErr) {
		t.Fatalf("error=%v is not DocumentContractError", err)
	}
	if contractErr.Field != field {
		t.Fatalf("field=%q want=%q error=%v", contractErr.Field, field, err)
	}
}
