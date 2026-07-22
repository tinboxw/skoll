package gormrepo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	documentworkflowsvc "github.com/tinboxw/skoll/internal/service/documentworkflow"
	storesql "github.com/tinboxw/skoll/internal/store/sql"
	"github.com/tinboxw/skoll/pkg/pluginsdk"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type DocumentWorkflowStore struct {
	db *gorm.DB
}

func NewDocumentWorkflowStore(db *gorm.DB) *DocumentWorkflowStore {
	if db == nil {
		panic("document workflow database is required")
	}
	return &DocumentWorkflowStore{db: db}
}

func (s *DocumentWorkflowStore) Get(ctx context.Context, key documentworkflowsvc.Key, forUpdate bool) (documentworkflowsvc.Binding, error) {
	db := storesql.ResolveDB(ctx, s.db)
	if forUpdate {
		if storesql.DBFromContext(ctx) == nil {
			return documentworkflowsvc.Binding{}, documentworkflowsvc.ErrTransactionRequired
		}
		db = db.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var row DocumentWorkflowBindingModel
	if err := db.Where(documentWorkflowKey(key)).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return documentworkflowsvc.Binding{}, documentworkflowsvc.ErrNotFound
		}
		return documentworkflowsvc.Binding{}, err
	}
	return documentWorkflowBinding(row)
}

func (s *DocumentWorkflowStore) FindAction(ctx context.Context, key documentworkflowsvc.Key, idempotencyKey string) (documentworkflowsvc.ActionRecord, bool, error) {
	db := storesql.DBFromContext(ctx)
	if db == nil {
		return documentworkflowsvc.ActionRecord{}, false, documentworkflowsvc.ErrTransactionRequired
	}
	var row DocumentWorkflowActionModel
	err := db.Where(documentWorkflowKey(key)).Where("idempotency_key = ?", idempotencyKey).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return documentworkflowsvc.ActionRecord{}, false, nil
	}
	if err != nil {
		return documentworkflowsvc.ActionRecord{}, false, err
	}
	return documentworkflowsvc.ActionRecord{
		Key: key, IdempotencyKey: row.IdempotencyKey, Action: row.Action,
		RequestHash: row.RequestHash, ResultJSON: []byte(row.ResultJSON), CreatedAt: row.CreatedAt,
	}, true, nil
}

func (s *DocumentWorkflowStore) Create(ctx context.Context, binding documentworkflowsvc.Binding, action documentworkflowsvc.ActionRecord, event pluginsdk.DocumentTimelineEvent) error {
	db := storesql.DBFromContext(ctx)
	if db == nil {
		return documentworkflowsvc.ErrTransactionRequired
	}
	row, err := documentWorkflowBindingRow(binding)
	if err != nil {
		return err
	}
	if err := db.Create(&row).Error; err != nil {
		if isDocumentWorkflowDuplicate(err) {
			return documentworkflowsvc.ErrConflict
		}
		return err
	}
	if err := createDocumentWorkflowAction(db, action); err != nil {
		return err
	}
	return createDocumentTimelineEvent(db, binding.Key, &event)
}

func (s *DocumentWorkflowStore) Update(ctx context.Context, binding documentworkflowsvc.Binding, previousVersion int64, action documentworkflowsvc.ActionRecord, event pluginsdk.DocumentTimelineEvent) error {
	db := storesql.DBFromContext(ctx)
	if db == nil {
		return documentworkflowsvc.ErrTransactionRequired
	}
	row, err := documentWorkflowBindingRow(binding)
	if err != nil {
		return err
	}
	result := db.Model(&DocumentWorkflowBindingModel{}).Where(documentWorkflowKey(binding.Key)).Where("version = ?", previousVersion).Updates(map[string]any{
		"state": row.State, "version": row.Version, "document_json": row.DocumentJSON, "updated_at": row.UpdatedAt,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return documentworkflowsvc.ErrConflict
	}
	if err := createDocumentWorkflowAction(db, action); err != nil {
		return err
	}
	return createDocumentTimelineEvent(db, binding.Key, &event)
}

func (s *DocumentWorkflowStore) AddAttachment(ctx context.Context, key documentworkflowsvc.Key, attachment pluginsdk.DocumentAttachment, event pluginsdk.DocumentTimelineEvent) (pluginsdk.DocumentAttachmentResult, error) {
	db := storesql.DBFromContext(ctx)
	if db == nil {
		return pluginsdk.DocumentAttachmentResult{}, documentworkflowsvc.ErrTransactionRequired
	}
	var count int64
	if err := db.Model(&DocumentAttachmentModel{}).Where(documentWorkflowKey(key)).Where("removed_at IS NULL").Count(&count).Error; err != nil {
		return pluginsdk.DocumentAttachmentResult{}, err
	}
	if count >= pluginsdk.MaxDocumentAttachments {
		return pluginsdk.DocumentAttachmentResult{}, documentworkflowsvc.ErrConflict
	}
	sequence, err := nextDocumentTimelineSequence(db, key)
	if err != nil {
		return pluginsdk.DocumentAttachmentResult{}, err
	}
	attachment.AddedSequence = sequence
	event.Sequence = sequence
	row, err := documentAttachmentRow(key, attachment)
	if err != nil {
		return pluginsdk.DocumentAttachmentResult{}, err
	}
	if err = db.Create(&row).Error; err != nil {
		if isDocumentWorkflowDuplicate(err) {
			return pluginsdk.DocumentAttachmentResult{}, documentworkflowsvc.ErrConflict
		}
		return pluginsdk.DocumentAttachmentResult{}, err
	}
	if err = insertDocumentTimelineEvent(db, key, event); err != nil {
		return pluginsdk.DocumentAttachmentResult{}, err
	}
	return pluginsdk.DocumentAttachmentResult{Attachment: attachment, Event: event}, nil
}

func (s *DocumentWorkflowStore) RemoveAttachment(ctx context.Context, key documentworkflowsvc.Key, attachmentID string, actor pluginsdk.WorkflowActor, now time.Time, event pluginsdk.DocumentTimelineEvent) (pluginsdk.DocumentAttachmentResult, error) {
	db := storesql.DBFromContext(ctx)
	if db == nil {
		return pluginsdk.DocumentAttachmentResult{}, documentworkflowsvc.ErrTransactionRequired
	}
	var row DocumentAttachmentModel
	if err := db.Clauses(clause.Locking{Strength: "UPDATE"}).Where(documentWorkflowKey(key)).Where("attachment_id = ?", attachmentID).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return pluginsdk.DocumentAttachmentResult{}, documentworkflowsvc.ErrNotFound
		}
		return pluginsdk.DocumentAttachmentResult{}, err
	}
	if row.RemovedAt != nil {
		return pluginsdk.DocumentAttachmentResult{}, documentworkflowsvc.ErrConflict
	}
	sequence, err := nextDocumentTimelineSequence(db, key)
	if err != nil {
		return pluginsdk.DocumentAttachmentResult{}, err
	}
	event.Sequence, event.FileID = sequence, row.FileID
	result := db.Model(&DocumentAttachmentModel{}).Where(documentWorkflowKey(key)).Where("attachment_id = ? AND removed_at IS NULL", attachmentID).Updates(map[string]any{
		"removed_by_id": actor.ID, "removed_by_name": actor.Name, "removed_at": now, "removed_sequence": sequence,
	})
	if result.Error != nil {
		return pluginsdk.DocumentAttachmentResult{}, result.Error
	}
	if result.RowsAffected != 1 {
		return pluginsdk.DocumentAttachmentResult{}, documentworkflowsvc.ErrConflict
	}
	row.RemovedByID, row.RemovedByName, row.RemovedAt, row.RemovedSequence = actor.ID, actor.Name, &now, &sequence
	if err = insertDocumentTimelineEvent(db, key, event); err != nil {
		return pluginsdk.DocumentAttachmentResult{}, err
	}
	attachment, err := documentAttachment(row)
	if err != nil {
		return pluginsdk.DocumentAttachmentResult{}, err
	}
	return pluginsdk.DocumentAttachmentResult{Attachment: attachment, Event: event}, nil
}

func (s *DocumentWorkflowStore) ListAttachments(ctx context.Context, key documentworkflowsvc.Key, offset, limit int, includeRemoved bool) ([]pluginsdk.DocumentAttachment, error) {
	db := storesql.ResolveDB(ctx, s.db).Where(documentWorkflowKey(key))
	if !includeRemoved {
		db = db.Where("removed_at IS NULL")
	}
	var rows []DocumentAttachmentModel
	if err := db.Order("added_sequence ASC").Offset(offset).Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]pluginsdk.DocumentAttachment, 0, len(rows))
	for _, row := range rows {
		item, err := documentAttachment(row)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, nil
}

func (s *DocumentWorkflowStore) AddComment(ctx context.Context, key documentworkflowsvc.Key, comment pluginsdk.DocumentComment, event pluginsdk.DocumentTimelineEvent) (pluginsdk.DocumentCommentResult, error) {
	db := storesql.DBFromContext(ctx)
	if db == nil {
		return pluginsdk.DocumentCommentResult{}, documentworkflowsvc.ErrTransactionRequired
	}
	var count int64
	if err := db.Model(&DocumentCommentModel{}).Where(documentWorkflowKey(key)).Count(&count).Error; err != nil {
		return pluginsdk.DocumentCommentResult{}, err
	}
	if count >= pluginsdk.MaxDocumentComments {
		return pluginsdk.DocumentCommentResult{}, documentworkflowsvc.ErrConflict
	}
	sequence, err := nextDocumentTimelineSequence(db, key)
	if err != nil {
		return pluginsdk.DocumentCommentResult{}, err
	}
	comment.Sequence, event.Sequence = sequence, sequence
	row := DocumentCommentModel{
		PluginID: key.PluginID, TenantID: key.TenantID, DocumentID: key.DocumentID, CommentID: comment.ID,
		Body: comment.Body, AuthorID: comment.Author.ID, AuthorName: comment.Author.Name, CreatedAt: comment.CreatedAt, Sequence: sequence,
	}
	if err = db.Create(&row).Error; err != nil {
		if isDocumentWorkflowDuplicate(err) {
			return pluginsdk.DocumentCommentResult{}, documentworkflowsvc.ErrConflict
		}
		return pluginsdk.DocumentCommentResult{}, err
	}
	if err = insertDocumentTimelineEvent(db, key, event); err != nil {
		return pluginsdk.DocumentCommentResult{}, err
	}
	return pluginsdk.DocumentCommentResult{Comment: comment, Event: event}, nil
}

func (s *DocumentWorkflowStore) ListComments(ctx context.Context, key documentworkflowsvc.Key, offset, limit int) ([]pluginsdk.DocumentComment, error) {
	var rows []DocumentCommentModel
	if err := storesql.ResolveDB(ctx, s.db).Where(documentWorkflowKey(key)).Order("sequence ASC").Offset(offset).Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]pluginsdk.DocumentComment, 0, len(rows))
	for _, row := range rows {
		out = append(out, pluginsdk.DocumentComment{
			ID: row.CommentID, DocumentID: row.DocumentID, Body: row.Body,
			Author: pluginsdk.WorkflowActor{ID: row.AuthorID, Name: row.AuthorName}, CreatedAt: row.CreatedAt, Sequence: row.Sequence,
		})
	}
	return out, nil
}

func (s *DocumentWorkflowStore) Timeline(ctx context.Context, key documentworkflowsvc.Key, afterSequence int64, limit int) (pluginsdk.DocumentTimelinePage, error) {
	var rows []DocumentTimelineEventModel
	if err := storesql.ResolveDB(ctx, s.db).Where(documentWorkflowKey(key)).Where("sequence > ?", afterSequence).Order("sequence ASC").Limit(limit + 1).Find(&rows).Error; err != nil {
		return pluginsdk.DocumentTimelinePage{}, err
	}
	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}
	events := make([]pluginsdk.DocumentTimelineEvent, 0, len(rows))
	for _, row := range rows {
		events = append(events, documentTimelineEvent(row))
	}
	next := afterSequence
	if len(events) > 0 {
		next = events[len(events)-1].Sequence
	}
	return pluginsdk.DocumentTimelinePage{Events: events, NextSequence: next, HasMore: hasMore}, nil
}

func documentWorkflowBindingRow(binding documentworkflowsvc.Binding) (DocumentWorkflowBindingModel, error) {
	schemaJSON, err := json.Marshal(binding.Schema)
	if err != nil {
		return DocumentWorkflowBindingModel{}, err
	}
	documentJSON, err := json.Marshal(binding.Document)
	if err != nil {
		return DocumentWorkflowBindingModel{}, err
	}
	return DocumentWorkflowBindingModel{
		PluginID: binding.Key.PluginID, TenantID: binding.Key.TenantID, DocumentID: binding.Key.DocumentID,
		DocumentType: binding.Document.Type, DefinitionID: binding.DefinitionID, WorkflowInstanceID: binding.WorkflowInstanceID,
		State: binding.Document.State, Version: binding.Document.Version, SchemaJSON: string(schemaJSON), DocumentJSON: string(documentJSON),
		CreatedAt: binding.CreatedAt, UpdatedAt: binding.UpdatedAt,
	}, nil
}

func documentWorkflowBinding(row DocumentWorkflowBindingModel) (documentworkflowsvc.Binding, error) {
	var schema pluginsdk.DocumentSchema
	if err := json.Unmarshal([]byte(row.SchemaJSON), &schema); err != nil {
		return documentworkflowsvc.Binding{}, err
	}
	var document pluginsdk.DocumentRecord
	if err := json.Unmarshal([]byte(row.DocumentJSON), &document); err != nil {
		return documentworkflowsvc.Binding{}, err
	}
	if err := schema.ValidateRecord(document); err != nil {
		return documentworkflowsvc.Binding{}, err
	}
	if row.DocumentType != document.Type || row.State != document.State || row.Version != document.Version {
		return documentworkflowsvc.Binding{}, errors.New("document workflow binding columns do not match the stored document")
	}
	return documentworkflowsvc.Binding{
		Key:    documentworkflowsvc.Key{PluginID: row.PluginID, TenantID: row.TenantID, DocumentID: row.DocumentID},
		Schema: schema, Document: document, DefinitionID: row.DefinitionID, WorkflowInstanceID: row.WorkflowInstanceID,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}, nil
}

func createDocumentWorkflowAction(db *gorm.DB, action documentworkflowsvc.ActionRecord) error {
	row := DocumentWorkflowActionModel{
		PluginID: action.Key.PluginID, TenantID: action.Key.TenantID, DocumentID: action.Key.DocumentID,
		IdempotencyKey: action.IdempotencyKey, Action: action.Action, RequestHash: action.RequestHash,
		ResultJSON: string(action.ResultJSON), CreatedAt: action.CreatedAt,
	}
	if err := db.Create(&row).Error; err != nil {
		if isDocumentWorkflowDuplicate(err) {
			return documentworkflowsvc.ErrConflict
		}
		return err
	}
	return nil
}

func createDocumentTimelineEvent(db *gorm.DB, key documentworkflowsvc.Key, event *pluginsdk.DocumentTimelineEvent) error {
	sequence, err := nextDocumentTimelineSequence(db, key)
	if err != nil {
		return err
	}
	event.Sequence = sequence
	return insertDocumentTimelineEvent(db, key, *event)
}

func nextDocumentTimelineSequence(db *gorm.DB, key documentworkflowsvc.Key) (int64, error) {
	var last int64
	err := db.Model(&DocumentTimelineEventModel{}).Where(documentWorkflowKey(key)).Select("COALESCE(MAX(sequence), 0)").Scan(&last).Error
	if err != nil {
		return 0, err
	}
	if last == int64(^uint64(0)>>1) {
		return 0, fmt.Errorf("document timeline sequence exhausted")
	}
	return last + 1, nil
}

func insertDocumentTimelineEvent(db *gorm.DB, key documentworkflowsvc.Key, event pluginsdk.DocumentTimelineEvent) error {
	row := DocumentTimelineEventModel{
		PluginID: key.PluginID, TenantID: key.TenantID, DocumentID: key.DocumentID, EventID: event.ID,
		Sequence: event.Sequence, Kind: string(event.Kind), Action: event.Action, AttachmentID: event.AttachmentID,
		FileID: event.FileID, CommentID: event.CommentID, ActorID: event.Actor.ID, ActorName: event.Actor.Name, OccurredAt: event.OccurredAt,
	}
	if err := db.Create(&row).Error; err != nil {
		if isDocumentWorkflowDuplicate(err) {
			return documentworkflowsvc.ErrConflict
		}
		return err
	}
	return nil
}

func documentAttachmentRow(key documentworkflowsvc.Key, attachment pluginsdk.DocumentAttachment) (DocumentAttachmentModel, error) {
	metadata, err := json.Marshal(attachment.File.Metadata)
	if err != nil {
		return DocumentAttachmentModel{}, err
	}
	return DocumentAttachmentModel{
		PluginID: key.PluginID, TenantID: key.TenantID, DocumentID: key.DocumentID, AttachmentID: attachment.ID,
		FileID: attachment.File.ID, FileKey: attachment.File.Key, FileName: attachment.File.Name, FileSize: attachment.File.Size,
		FileMIME: attachment.File.MIME, FileHash: attachment.File.Hash, FileVisibility: string(attachment.File.Visibility),
		FileStatus: attachment.File.Status, FileMetadata: string(metadata), FileCreatedAt: attachment.File.CreatedAt, FileUpdatedAt: attachment.File.UpdatedAt,
		AddedByID: attachment.AddedBy.ID, AddedByName: attachment.AddedBy.Name, AddedAt: attachment.AddedAt, AddedSequence: attachment.AddedSequence,
	}, nil
}

func documentAttachment(row DocumentAttachmentModel) (pluginsdk.DocumentAttachment, error) {
	metadata := map[string]string{}
	if err := json.Unmarshal([]byte(row.FileMetadata), &metadata); err != nil {
		return pluginsdk.DocumentAttachment{}, err
	}
	return pluginsdk.DocumentAttachment{
		ID: row.AttachmentID, DocumentID: row.DocumentID,
		File: pluginsdk.FileObject{
			ID: row.FileID, Key: row.FileKey, Name: row.FileName, Size: row.FileSize, MIME: row.FileMIME, Hash: row.FileHash,
			Visibility: pluginsdk.FileVisibility(row.FileVisibility), Status: row.FileStatus, Metadata: metadata,
			CreatedAt: row.FileCreatedAt, UpdatedAt: row.FileUpdatedAt,
		},
		AddedBy: pluginsdk.WorkflowActor{ID: row.AddedByID, Name: row.AddedByName}, AddedAt: row.AddedAt, AddedSequence: row.AddedSequence,
		RemovedBy: pluginsdk.WorkflowActor{ID: row.RemovedByID, Name: row.RemovedByName}, RemovedAt: row.RemovedAt, RemovedSequence: row.RemovedSequence,
	}, nil
}

func documentTimelineEvent(row DocumentTimelineEventModel) pluginsdk.DocumentTimelineEvent {
	return pluginsdk.DocumentTimelineEvent{
		ID: row.EventID, Sequence: row.Sequence, DocumentID: row.DocumentID, Kind: pluginsdk.DocumentTimelineKind(row.Kind),
		Action: row.Action, AttachmentID: row.AttachmentID, FileID: row.FileID, CommentID: row.CommentID,
		Actor: pluginsdk.WorkflowActor{ID: row.ActorID, Name: row.ActorName}, OccurredAt: row.OccurredAt,
	}
}

func isDocumentWorkflowDuplicate(err error) bool {
	message := strings.ToLower(err.Error())
	return errors.Is(err, gorm.ErrDuplicatedKey) || strings.Contains(message, "unique") || strings.Contains(message, "duplicate")
}

func documentWorkflowKey(key documentworkflowsvc.Key) map[string]any {
	return map[string]any{"plugin_id": key.PluginID, "tenant_id": key.TenantID, "document_id": key.DocumentID}
}

var _ documentworkflowsvc.Repository = (*DocumentWorkflowStore)(nil)
