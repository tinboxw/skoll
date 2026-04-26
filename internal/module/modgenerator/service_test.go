package modgenerator

import "testing"

func TestServiceGenerate(t *testing.T) {
	svc := NewService()
	result, err := svc.Generate("account")
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	if result.Module != "account" {
		t.Fatalf("unexpected module name: %s", result.Module)
	}
	if len(result.Artifacts) != 2 {
		t.Fatalf("expected 2 artifacts, got %d", len(result.Artifacts))
	}
}

func TestServiceGenerate_InvalidName(t *testing.T) {
	svc := NewService()
	if _, err := svc.Generate("Account-Module"); err == nil {
		t.Fatalf("expected invalid module name error")
	}
}

func TestServiceGenerateWithSchemaAndCompatibility(t *testing.T) {
	svc := NewService()
	result, err := svc.GenerateWithSchema("order", &FormSchema{
		Version: "v2",
		Fields: []FormField{
			{Name: "name", Type: "string", Required: true},
			{Name: "name", Type: "string", Required: true},
			{Name: "status", Type: "select", Required: false},
			{Name: "", Type: "string", Required: false},
		},
	}, "v2")
	if err != nil {
		t.Fatalf("generate with schema failed: %v", err)
	}
	if result.FormSchema == nil || len(result.FormSchema.Fields) != 2 {
		t.Fatalf("expected normalized schema fields, got %+v", result.FormSchema)
	}
	if result.Compatibility.TemplateVersion != "v2" {
		t.Fatalf("expected template version v2, got %+v", result.Compatibility)
	}
}

func TestServiceGenerateWithSchema_InvalidVersions(t *testing.T) {
	svc := NewService()
	if _, err := svc.GenerateWithSchema("order", &FormSchema{Version: "v3"}, "v1"); err == nil {
		t.Fatalf("expected invalid schema version error")
	}
	if _, err := svc.GenerateWithSchema("order", nil, "v9"); err == nil {
		t.Fatalf("expected invalid template version error")
	}
}

func BenchmarkServiceGenerateWithSchema(b *testing.B) {
	svc := NewService()
	schema := &FormSchema{
		Version: "v2",
		Fields: []FormField{
			{Name: "name", Type: "string", Required: true},
			{Name: "status", Type: "select", Required: false},
			{Name: "owner", Type: "string", Required: false},
		},
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := svc.GenerateWithSchema("billing", schema, "v2"); err != nil {
			b.Fatalf("generate with schema failed: %v", err)
		}
	}
}
