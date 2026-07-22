CREATE TABLE IF NOT EXISTS sk_document_workflow_bindings (
	plugin_id VARCHAR(64) NOT NULL,
	tenant_id VARCHAR(128) NOT NULL,
	document_id VARCHAR(128) NOT NULL,
	document_type VARCHAR(63) NOT NULL,
	definition_id VARCHAR(512) NOT NULL,
	workflow_instance_id VARCHAR(512) NOT NULL,
	state VARCHAR(63) NOT NULL,
	version BIGINT NOT NULL,
	schema_json LONGTEXT NOT NULL,
	document_json LONGTEXT NOT NULL,
	created_at DATETIME(3) NOT NULL,
	updated_at DATETIME(3) NOT NULL,
	PRIMARY KEY (plugin_id, tenant_id, document_id),
	UNIQUE KEY idx_document_workflow_instance (plugin_id, workflow_instance_id)
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
