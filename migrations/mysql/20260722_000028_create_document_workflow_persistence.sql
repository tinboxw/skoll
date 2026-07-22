CREATE TABLE IF NOT EXISTS sk_document_workflow_bindings (
	plugin_id VARCHAR(64) NOT NULL,
	tenant_id VARCHAR(128) NOT NULL,
	document_id VARCHAR(128) NOT NULL,
	document_type VARCHAR(63) NOT NULL,
	number VARCHAR(128) NOT NULL,
	title VARCHAR(256) NOT NULL,
	definition_id VARCHAR(512) NOT NULL,
	workflow_instance_id VARCHAR(512) NOT NULL,
	state VARCHAR(63) NOT NULL,
	version BIGINT NOT NULL,
	schema_json LONGTEXT NOT NULL,
	document_json LONGTEXT NOT NULL,
	created_by VARCHAR(512) NOT NULL,
	updated_by VARCHAR(512) NOT NULL,
	created_at DATETIME(3) NOT NULL,
	updated_at DATETIME(3) NOT NULL,
	PRIMARY KEY (plugin_id, tenant_id, document_id),
	UNIQUE KEY idx_document_workflow_instance (plugin_id, workflow_instance_id),
	KEY idx_document_search_updated (plugin_id, tenant_id, updated_at, document_id),
	KEY idx_document_search_created (plugin_id, tenant_id, created_at, document_id),
	KEY idx_document_search_number (plugin_id, tenant_id, number, document_id),
	KEY idx_document_search_type_state (plugin_id, tenant_id, document_type, state, updated_at, document_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS sk_document_workflow_actions (
	plugin_id VARCHAR(64) NOT NULL,
	tenant_id VARCHAR(128) NOT NULL,
	document_id VARCHAR(128) NOT NULL,
	idempotency_key VARCHAR(128) NOT NULL,
	action VARCHAR(32) NOT NULL,
	request_hash CHAR(64) NOT NULL,
	result_json LONGTEXT NOT NULL,
	created_at DATETIME(3) NOT NULL,
	PRIMARY KEY (plugin_id, tenant_id, document_id, idempotency_key),
	CONSTRAINT fk_document_workflow_actions_binding FOREIGN KEY (plugin_id, tenant_id, document_id)
		REFERENCES sk_document_workflow_bindings (plugin_id, tenant_id, document_id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
