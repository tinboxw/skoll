package gormrepo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDocumentWorkflowMigrationsMatchCurrentModels(t *testing.T) {
	required := []string{
		"sk_document_workflow_bindings", "sk_document_workflow_actions",
		"plugin_id", "tenant_id", "document_id", "workflow_instance_id", "number", "title", "created_by", "updated_by",
		"schema_json", "document_json", "idempotency_key", "request_hash", "result_json",
		"idx_document_workflow_instance", "idx_document_search_updated", "idx_document_search_created",
		"idx_document_search_number", "idx_document_search_type_state", "fk_document_workflow_actions_binding", "foreign key",
	}
	root := filepath.Join("..", "..", "..", "..", "migrations")
	for _, dialect := range []string{"mysql", "postgres"} {
		path := filepath.Join(root, dialect, "20260722_000028_create_document_workflow_persistence.sql")
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s migration: %v", dialect, err)
		}
		text := strings.ToLower(string(body))
		for _, token := range required {
			if !strings.Contains(text, token) {
				t.Fatalf("%s document workflow migration missing %q", dialect, token)
			}
		}
		if strings.Contains(text, "drop table") || strings.Contains(text, "legacy") {
			t.Fatalf("%s migration contains a destructive or compatibility path", dialect)
		}
	}
}

func TestAllModelsIncludesDocumentWorkflowPersistence(t *testing.T) {
	models := AllModels()
	var binding, action bool
	for _, model := range models {
		switch model.(type) {
		case *DocumentWorkflowBindingModel:
			binding = true
		case *DocumentWorkflowActionModel:
			action = true
		}
	}
	if !binding || !action {
		t.Fatalf("document workflow models missing: binding=%t action=%t", binding, action)
	}
}
