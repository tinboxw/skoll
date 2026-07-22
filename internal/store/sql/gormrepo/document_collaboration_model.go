package gormrepo

import "time"

type DocumentAttachmentModel struct {
	PluginID        string     `gorm:"column:plugin_id;type:varchar(64);primaryKey;index:idx_document_attachment_active,priority:1"`
	TenantID        string     `gorm:"column:tenant_id;type:varchar(128);primaryKey;index:idx_document_attachment_active,priority:2"`
	DocumentID      string     `gorm:"column:document_id;type:varchar(128);primaryKey;index:idx_document_attachment_active,priority:3"`
	AttachmentID    string     `gorm:"column:attachment_id;type:varchar(128);primaryKey"`
	FileID          string     `gorm:"column:file_id;type:varchar(128);not null;index:idx_document_attachment_file"`
	FileKey         string     `gorm:"column:file_key;type:varchar(512);not null"`
	FileName        string     `gorm:"column:file_name;type:varchar(255);not null"`
	FileSize        int64      `gorm:"column:file_size;not null"`
	FileMIME        string     `gorm:"column:file_mime;type:varchar(255);not null"`
	FileHash        string     `gorm:"column:file_hash;type:varchar(128);not null"`
	FileVisibility  string     `gorm:"column:file_visibility;type:varchar(32);not null"`
	FileStatus      string     `gorm:"column:file_status;type:varchar(32);not null"`
	FileMetadata    string     `gorm:"column:file_metadata_json;type:longtext;not null"`
	FileCreatedAt   time.Time  `gorm:"column:file_created_at;not null"`
	FileUpdatedAt   time.Time  `gorm:"column:file_updated_at;not null"`
	AddedByID       string     `gorm:"column:added_by_id;type:varchar(512);not null"`
	AddedByName     string     `gorm:"column:added_by_name;type:varchar(256);not null"`
	AddedAt         time.Time  `gorm:"column:added_at;not null"`
	AddedSequence   int64      `gorm:"column:added_sequence;not null"`
	RemovedByID     string     `gorm:"column:removed_by_id;type:varchar(512)"`
	RemovedByName   string     `gorm:"column:removed_by_name;type:varchar(256)"`
	RemovedAt       *time.Time `gorm:"column:removed_at;index:idx_document_attachment_active,priority:4"`
	RemovedSequence *int64     `gorm:"column:removed_sequence"`
}

func (DocumentAttachmentModel) TableName() string { return "sk_document_attachments" }

type DocumentCommentModel struct {
	PluginID   string    `gorm:"column:plugin_id;type:varchar(64);primaryKey"`
	TenantID   string    `gorm:"column:tenant_id;type:varchar(128);primaryKey"`
	DocumentID string    `gorm:"column:document_id;type:varchar(128);primaryKey"`
	CommentID  string    `gorm:"column:comment_id;type:varchar(128);primaryKey"`
	Body       string    `gorm:"column:body;type:text;not null"`
	AuthorID   string    `gorm:"column:author_id;type:varchar(512);not null"`
	AuthorName string    `gorm:"column:author_name;type:varchar(256);not null"`
	CreatedAt  time.Time `gorm:"column:created_at;not null"`
	Sequence   int64     `gorm:"column:sequence;not null"`
}

func (DocumentCommentModel) TableName() string { return "sk_document_comments" }

type DocumentTimelineEventModel struct {
	PluginID     string    `gorm:"column:plugin_id;type:varchar(64);primaryKey;uniqueIndex:idx_document_timeline_sequence,priority:1;index:idx_document_timeline_occurred,priority:1"`
	TenantID     string    `gorm:"column:tenant_id;type:varchar(128);primaryKey;uniqueIndex:idx_document_timeline_sequence,priority:2;index:idx_document_timeline_occurred,priority:2"`
	DocumentID   string    `gorm:"column:document_id;type:varchar(128);primaryKey;uniqueIndex:idx_document_timeline_sequence,priority:3;index:idx_document_timeline_occurred,priority:3"`
	EventID      string    `gorm:"column:event_id;type:varchar(192);primaryKey"`
	Sequence     int64     `gorm:"column:sequence;not null;uniqueIndex:idx_document_timeline_sequence,priority:4"`
	Kind         string    `gorm:"column:kind;type:varchar(32);not null"`
	Action       string    `gorm:"column:action;type:varchar(32)"`
	AttachmentID string    `gorm:"column:attachment_id;type:varchar(128)"`
	FileID       string    `gorm:"column:file_id;type:varchar(128)"`
	CommentID    string    `gorm:"column:comment_id;type:varchar(128)"`
	ActorID      string    `gorm:"column:actor_id;type:varchar(512);not null"`
	ActorName    string    `gorm:"column:actor_name;type:varchar(256);not null"`
	OccurredAt   time.Time `gorm:"column:occurred_at;not null;index:idx_document_timeline_occurred,priority:4"`
}

func (DocumentTimelineEventModel) TableName() string { return "sk_document_timeline_events" }
