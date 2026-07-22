package gormrepo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDocumentCollaborationMigrationsMatchCurrentModels(t *testing.T) {
	required := []string{
		"sk_document_attachments", "sk_document_comments", "sk_document_timeline_events",
		"attachment_id", "file_metadata_json", "removed_by_id", "removed_sequence",
		"comment_id", "author_id", "sequence", "event_id", "occurred_at",
		"idx_document_attachment_active", "idx_document_timeline_sequence", "idx_document_timeline_occurred",
		"fk_document_attachments_binding", "fk_document_comments_binding", "fk_document_timeline_binding", "foreign key",
	}
	root := filepath.Join("..", "..", "..", "..", "migrations")
	for _, dialect := range []string{"mysql", "postgres"} {
		path := filepath.Join(root, dialect, "20260722_000029_create_document_collaboration_persistence.sql")
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s migration: %v", dialect, err)
		}
		text := strings.ToLower(string(body))
		for _, token := range required {
			if !strings.Contains(text, token) {
				t.Fatalf("%s document collaboration migration missing %q", dialect, token)
			}
		}
		if strings.Contains(text, "drop table") || strings.Contains(text, "legacy") {
			t.Fatalf("%s migration contains a destructive or compatibility path", dialect)
		}
		timeline := text[strings.Index(text, "create table if not exists sk_document_timeline_events"):]
		if strings.Contains(timeline, "updated_at") || strings.Contains(timeline, "deleted_at") {
			t.Fatalf("%s timeline persistence exposes mutable lifecycle columns", dialect)
		}
	}
}

func TestAllModelsIncludesDocumentCollaborationPersistence(t *testing.T) {
	models := AllModels()
	var attachment, comment, timeline bool
	for _, model := range models {
		switch model.(type) {
		case *DocumentAttachmentModel:
			attachment = true
		case *DocumentCommentModel:
			comment = true
		case *DocumentTimelineEventModel:
			timeline = true
		}
	}
	if !attachment || !comment || !timeline {
		t.Fatalf("document collaboration models missing: attachment=%t comment=%t timeline=%t", attachment, comment, timeline)
	}
}
