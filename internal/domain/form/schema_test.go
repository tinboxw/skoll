package form

import (
	"testing"
	"time"

	"github.com/tinboxw/skoll/internal/domain/shared"
)

func TestPharmaOAFormFixturesAreExpressible(t *testing.T) {
	for _, fixture := range []struct {
		name       string
		input      SchemaInput
		wantFields []string
	}{
		{name: "leave", input: leaveFormFixture(), wantFields: []string{"employee", "leave_type", "period"}},
		{name: "reimbursement", input: reimbursementFormFixture(), wantFields: []string{"applicant", "expense_items", "attachments"}},
		{name: "purchase request", input: purchaseRequestFormFixture(), wantFields: []string{"supplier", "purchase_items", "expected_date"}},
		{name: "inbound", input: inboundFormFixture(), wantFields: []string{"warehouse", "inbound_items", "temperature_record"}},
		{name: "customer qualification", input: customerQualificationFormFixture(), wantFields: []string{"customer_name", "qualification_type", "qualification_files"}},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			schema, err := NewSchema(fixture.input)
			if err != nil {
				t.Fatalf("NewSchema error: %v", err)
			}
			for _, key := range fixture.wantFields {
				if _, ok := schema.FieldByKey(key); !ok {
					t.Fatalf("expected field %q in schema %+v", key, schema.Fields)
				}
			}
		})
	}
}

func TestFormSchemaValidationRejectsInvalidShapes(t *testing.T) {
	valid := leaveFormFixture()
	valid.Fields = append(valid.Fields, valid.Fields[0])
	if _, err := NewSchema(valid); err == nil {
		t.Fatal("expected duplicate field key error")
	}

	invalidAttachment := leaveFormFixture()
	invalidAttachment.Fields = append(invalidAttachment.Fields, Field{
		Key:        "broken_attachment",
		Label:      "Broken Attachment",
		Type:       FieldAttachment,
		Attachment: &AttachmentConfig{},
	})
	if _, err := NewSchema(invalidAttachment); err == nil {
		t.Fatal("expected invalid attachment config error")
	}

	invalidDetail := reimbursementFormFixture()
	invalidDetail.Fields[1].DetailTable.Columns = append(invalidDetail.Fields[1].DetailTable.Columns, Field{
		Key:         "nested",
		Label:       "Nested",
		Type:        FieldDetailTable,
		DetailTable: &DetailTableConfig{Columns: []Field{{Key: "name", Label: "Name", Type: FieldString}}},
	})
	if _, err := NewSchema(invalidDetail); err == nil {
		t.Fatal("expected nested detail table error")
	}
}

func leaveFormFixture() SchemaInput {
	return SchemaInput{
		ID:           shared.ID("form-leave"),
		Key:          "oa.leave",
		Name:         "Leave Request",
		Version:      1,
		BusinessType: "oa.leave",
		Fields: []Field{
			userField("employee", "Employee"),
			{Key: "leave_type", Label: "Leave Type", Type: FieldDictionary, Required: true, Dictionary: "oa.leave_type"},
			{Key: "period", Label: "Leave Period", Type: FieldDate, Required: true, Validation: []ValidationRule{{Type: ValidationRequired}}},
			{Key: "reason", Label: "Reason", Type: FieldTextarea, Required: true, Validation: []ValidationRule{{Type: ValidationMinLength, Value: "4"}}},
		},
		Now: fixtureNow(),
	}
}

func reimbursementFormFixture() SchemaInput {
	return SchemaInput{
		ID:           shared.ID("form-reimbursement"),
		Key:          "oa.reimbursement",
		Name:         "Reimbursement",
		Version:      1,
		BusinessType: "oa.reimbursement",
		Fields: []Field{
			userField("applicant", "Applicant"),
			{
				Key:      "expense_items",
				Label:    "Expense Items",
				Type:     FieldDetailTable,
				Required: true,
				DetailTable: &DetailTableConfig{MinRows: 1, Columns: []Field{
					{Key: "category", Label: "Category", Type: FieldDictionary, Dictionary: "oa.expense_category", Required: true},
					{Key: "amount", Label: "Amount", Type: FieldNumber, Required: true, Validation: []ValidationRule{{Type: ValidationMin, Value: "0.01"}}},
					{Key: "occurred_at", Label: "Occurred At", Type: FieldDate, Required: true},
				}},
			},
			attachmentField("attachments", "Invoices"),
		},
		Now: fixtureNow(),
	}
}

func purchaseRequestFormFixture() SchemaInput {
	return SchemaInput{
		ID:           shared.ID("form-purchase-request"),
		Key:          "pharma.purchase_request",
		Name:         "Purchase Request",
		Version:      1,
		BusinessType: "pharma.purchase_request",
		Fields: []Field{
			{Key: "supplier", Label: "Supplier", Type: FieldString, Required: true},
			{Key: "expected_date", Label: "Expected Date", Type: FieldDate, Required: true},
			{
				Key:      "purchase_items",
				Label:    "Purchase Items",
				Type:     FieldDetailTable,
				Required: true,
				DetailTable: &DetailTableConfig{MinRows: 1, Columns: []Field{
					{Key: "product", Label: "Product", Type: FieldString, Required: true},
					{Key: "quantity", Label: "Quantity", Type: FieldNumber, Required: true, Validation: []ValidationRule{{Type: ValidationMin, Value: "1"}}},
					{Key: "unit_price", Label: "Unit Price", Type: FieldNumber, Required: true, Validation: []ValidationRule{{Type: ValidationMin, Value: "0"}}},
				}},
			},
		},
		Now: fixtureNow(),
	}
}

func inboundFormFixture() SchemaInput {
	return SchemaInput{
		ID:           shared.ID("form-inbound"),
		Key:          "pharma.inbound",
		Name:         "Purchase Inbound",
		Version:      1,
		BusinessType: "pharma.inbound",
		Fields: []Field{
			{Key: "warehouse", Label: "Warehouse", Type: FieldString, Required: true},
			{Key: "temperature_record", Label: "Temperature Record", Type: FieldNumber, Required: true, Validation: []ValidationRule{{Type: ValidationMin, Value: "-30"}, {Type: ValidationMax, Value: "30"}}},
			{
				Key:      "inbound_items",
				Label:    "Inbound Items",
				Type:     FieldDetailTable,
				Required: true,
				DetailTable: &DetailTableConfig{MinRows: 1, Columns: []Field{
					{Key: "product", Label: "Product", Type: FieldString, Required: true},
					{Key: "batch_no", Label: "Batch No", Type: FieldString, Required: true},
					{Key: "expiry_date", Label: "Expiry Date", Type: FieldDate, Required: true},
					{Key: "quantity", Label: "Quantity", Type: FieldNumber, Required: true},
				}},
			},
			attachmentField("inspection_files", "Inspection Files"),
		},
		Now: fixtureNow(),
	}
}

func customerQualificationFormFixture() SchemaInput {
	return SchemaInput{
		ID:           shared.ID("form-customer-qualification"),
		Key:          "pharma.customer_qualification",
		Name:         "Customer Qualification",
		Version:      1,
		BusinessType: "pharma.customer_qualification",
		Fields: []Field{
			{Key: "customer_name", Label: "Customer Name", Type: FieldString, Required: true},
			{Key: "qualification_type", Label: "Qualification Type", Type: FieldSelect, Required: true, Options: []Option{{Label: "Business License", Value: "business_license"}, {Label: "GSP Certificate", Value: "gsp_certificate"}}},
			{Key: "valid_to", Label: "Valid To", Type: FieldDate, Required: true},
			attachmentField("qualification_files", "Qualification Files"),
		},
		Now: fixtureNow(),
	}
}

func userField(key, label string) Field {
	return Field{Key: key, Label: label, Type: FieldUser, Required: true}
}

func attachmentField(key, label string) Field {
	return Field{
		Key:      key,
		Label:    label,
		Type:     FieldAttachment,
		Required: true,
		Attachment: &AttachmentConfig{
			MaxFiles:  5,
			MaxSizeMB: 20,
			Accept:    []string{"pdf", "jpg", "png"},
			Required:  true,
		},
	}
}

func fixtureNow() time.Time {
	return time.Date(2026, time.July, 4, 9, 0, 0, 0, time.UTC)
}
