package gormrepo

import "time"

type DocumentWorkflowBindingModel struct {
	PluginID           string    `gorm:"column:plugin_id;type:varchar(64);primaryKey;uniqueIndex:idx_document_workflow_instance,priority:1;index:idx_document_search_updated,priority:1;index:idx_document_search_created,priority:1;index:idx_document_search_number,priority:1;index:idx_document_search_type_state,priority:1"`
	TenantID           string    `gorm:"column:tenant_id;type:varchar(128);primaryKey;index:idx_document_search_updated,priority:2;index:idx_document_search_created,priority:2;index:idx_document_search_number,priority:2;index:idx_document_search_type_state,priority:2"`
	DocumentID         string    `gorm:"column:document_id;type:varchar(128);primaryKey;index:idx_document_search_updated,priority:4;index:idx_document_search_created,priority:4;index:idx_document_search_number,priority:4;index:idx_document_search_type_state,priority:6"`
	DocumentType       string    `gorm:"column:document_type;type:varchar(63);not null;index:idx_document_search_type_state,priority:3"`
	Number             string    `gorm:"column:number;type:varchar(128);not null;index:idx_document_search_number,priority:3"`
	Title              string    `gorm:"column:title;type:varchar(256);not null"`
	DefinitionID       string    `gorm:"column:definition_id;type:varchar(512);not null"`
	WorkflowInstanceID string    `gorm:"column:workflow_instance_id;type:varchar(512);not null;uniqueIndex:idx_document_workflow_instance,priority:2"`
	State              string    `gorm:"column:state;type:varchar(63);not null;index:idx_document_search_type_state,priority:4"`
	Version            int64     `gorm:"column:version;not null"`
	SchemaJSON         string    `gorm:"column:schema_json;type:longtext;not null"`
	DocumentJSON       string    `gorm:"column:document_json;type:longtext;not null"`
	CreatedBy          string    `gorm:"column:created_by;type:varchar(512);not null"`
	UpdatedBy          string    `gorm:"column:updated_by;type:varchar(512);not null"`
	CreatedAt          time.Time `gorm:"column:created_at;not null;index:idx_document_search_created,priority:3"`
	UpdatedAt          time.Time `gorm:"column:updated_at;not null;index:idx_document_search_updated,priority:3;index:idx_document_search_type_state,priority:5"`
}

func (DocumentWorkflowBindingModel) TableName() string { return "sk_document_workflow_bindings" }

type DocumentWorkflowActionModel struct {
	PluginID       string    `gorm:"column:plugin_id;type:varchar(64);primaryKey"`
	TenantID       string    `gorm:"column:tenant_id;type:varchar(128);primaryKey"`
	DocumentID     string    `gorm:"column:document_id;type:varchar(128);primaryKey"`
	IdempotencyKey string    `gorm:"column:idempotency_key;type:varchar(128);primaryKey"`
	Action         string    `gorm:"column:action;type:varchar(32);not null"`
	RequestHash    string    `gorm:"column:request_hash;type:char(64);not null"`
	ResultJSON     string    `gorm:"column:result_json;type:longtext;not null"`
	CreatedAt      time.Time `gorm:"column:created_at;not null"`
}

func (DocumentWorkflowActionModel) TableName() string { return "sk_document_workflow_actions" }
