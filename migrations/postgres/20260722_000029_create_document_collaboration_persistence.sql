CREATE TABLE IF NOT EXISTS sk_document_attachments (
	plugin_id VARCHAR(64) NOT NULL,
	tenant_id VARCHAR(128) NOT NULL,
	document_id VARCHAR(128) NOT NULL,
	attachment_id VARCHAR(128) NOT NULL,
	file_id VARCHAR(128) NOT NULL,
	file_key VARCHAR(512) NOT NULL,
	file_name VARCHAR(255) NOT NULL,
	file_size BIGINT NOT NULL,
	file_mime VARCHAR(255) NOT NULL,
	file_hash VARCHAR(128) NOT NULL,
	file_visibility VARCHAR(32) NOT NULL,
	file_status VARCHAR(32) NOT NULL,
	file_metadata_json TEXT NOT NULL,
	file_created_at TIMESTAMPTZ NOT NULL,
	file_updated_at TIMESTAMPTZ NOT NULL,
	added_by_id VARCHAR(512) NOT NULL,
	added_by_name VARCHAR(256) NOT NULL,
	added_at TIMESTAMPTZ NOT NULL,
	added_sequence BIGINT NOT NULL,
	removed_by_id VARCHAR(512) NULL,
	removed_by_name VARCHAR(256) NULL,
	removed_at TIMESTAMPTZ NULL,
	removed_sequence BIGINT NULL,
	PRIMARY KEY (plugin_id, tenant_id, document_id, attachment_id),
	CONSTRAINT fk_document_attachments_binding FOREIGN KEY (plugin_id, tenant_id, document_id)
		REFERENCES sk_document_workflow_bindings (plugin_id, tenant_id, document_id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_document_attachment_file ON sk_document_attachments (file_id);
CREATE INDEX IF NOT EXISTS idx_document_attachment_active ON sk_document_attachments (plugin_id, tenant_id, document_id, removed_at);

CREATE TABLE IF NOT EXISTS sk_document_comments (
	plugin_id VARCHAR(64) NOT NULL,
	tenant_id VARCHAR(128) NOT NULL,
	document_id VARCHAR(128) NOT NULL,
	comment_id VARCHAR(128) NOT NULL,
	body TEXT NOT NULL,
	author_id VARCHAR(512) NOT NULL,
	author_name VARCHAR(256) NOT NULL,
	created_at TIMESTAMPTZ NOT NULL,
	sequence BIGINT NOT NULL,
	PRIMARY KEY (plugin_id, tenant_id, document_id, comment_id),
	CONSTRAINT fk_document_comments_binding FOREIGN KEY (plugin_id, tenant_id, document_id)
		REFERENCES sk_document_workflow_bindings (plugin_id, tenant_id, document_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS sk_document_timeline_events (
	plugin_id VARCHAR(64) NOT NULL,
	tenant_id VARCHAR(128) NOT NULL,
	document_id VARCHAR(128) NOT NULL,
	event_id VARCHAR(192) NOT NULL,
	sequence BIGINT NOT NULL,
	kind VARCHAR(32) NOT NULL,
	action VARCHAR(32) NULL,
	attachment_id VARCHAR(128) NULL,
	file_id VARCHAR(128) NULL,
	comment_id VARCHAR(128) NULL,
	actor_id VARCHAR(512) NOT NULL,
	actor_name VARCHAR(256) NOT NULL,
	occurred_at TIMESTAMPTZ NOT NULL,
	PRIMARY KEY (plugin_id, tenant_id, document_id, event_id),
	CONSTRAINT idx_document_timeline_sequence UNIQUE (plugin_id, tenant_id, document_id, sequence),
	CONSTRAINT fk_document_timeline_binding FOREIGN KEY (plugin_id, tenant_id, document_id)
		REFERENCES sk_document_workflow_bindings (plugin_id, tenant_id, document_id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_document_timeline_occurred ON sk_document_timeline_events (plugin_id, tenant_id, document_id, occurred_at);
