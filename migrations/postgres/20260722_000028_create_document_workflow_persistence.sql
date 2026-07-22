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
	schema_json TEXT NOT NULL,
	document_json TEXT NOT NULL,
	created_by VARCHAR(512) NOT NULL,
	updated_by VARCHAR(512) NOT NULL,
	created_at TIMESTAMPTZ NOT NULL,
	updated_at TIMESTAMPTZ NOT NULL,
	PRIMARY KEY (plugin_id, tenant_id, document_id),
	CONSTRAINT idx_document_workflow_instance UNIQUE (plugin_id, workflow_instance_id)
);
CREATE INDEX IF NOT EXISTS idx_document_search_updated ON sk_document_workflow_bindings (plugin_id, tenant_id, updated_at, document_id);
CREATE INDEX IF NOT EXISTS idx_document_search_created ON sk_document_workflow_bindings (plugin_id, tenant_id, created_at, document_id);
CREATE INDEX IF NOT EXISTS idx_document_search_number ON sk_document_workflow_bindings (plugin_id, tenant_id, number, document_id);
CREATE INDEX IF NOT EXISTS idx_document_search_type_state ON sk_document_workflow_bindings (plugin_id, tenant_id, document_type, state, updated_at, document_id);

CREATE TABLE IF NOT EXISTS sk_document_workflow_actions (
	plugin_id VARCHAR(64) NOT NULL,
	tenant_id VARCHAR(128) NOT NULL,
	document_id VARCHAR(128) NOT NULL,
	idempotency_key VARCHAR(128) NOT NULL,
	action VARCHAR(32) NOT NULL,
	request_hash CHAR(64) NOT NULL,
	result_json TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL,
	PRIMARY KEY (plugin_id, tenant_id, document_id, idempotency_key),
	CONSTRAINT fk_document_workflow_actions_binding FOREIGN KEY (plugin_id, tenant_id, document_id)
		REFERENCES sk_document_workflow_bindings (plugin_id, tenant_id, document_id) ON DELETE CASCADE
);
