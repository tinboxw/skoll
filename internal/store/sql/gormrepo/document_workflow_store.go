package gormrepo

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

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

func (s *DocumentWorkflowStore) Create(ctx context.Context, binding documentworkflowsvc.Binding, action documentworkflowsvc.ActionRecord) error {
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
	return createDocumentWorkflowAction(db, action)
}

func (s *DocumentWorkflowStore) Update(ctx context.Context, binding documentworkflowsvc.Binding, previousVersion int64, action documentworkflowsvc.ActionRecord) error {
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
	return createDocumentWorkflowAction(db, action)
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

func isDocumentWorkflowDuplicate(err error) bool {
	message := strings.ToLower(err.Error())
	return errors.Is(err, gorm.ErrDuplicatedKey) || strings.Contains(message, "unique") || strings.Contains(message, "duplicate")
}

func documentWorkflowKey(key documentworkflowsvc.Key) map[string]any {
	return map[string]any{"plugin_id": key.PluginID, "tenant_id": key.TenantID, "document_id": key.DocumentID}
}

var _ documentworkflowsvc.Repository = (*DocumentWorkflowStore)(nil)
