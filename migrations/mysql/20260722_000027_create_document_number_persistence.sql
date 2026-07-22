CREATE TABLE IF NOT EXISTS sk_document_number_sequences (
	plugin_id VARCHAR(64) NOT NULL,
	tenant_id VARCHAR(128) NOT NULL,
	document_type VARCHAR(63) NOT NULL,
	period_key VARCHAR(8) NOT NULL,
	rule_hash CHAR(64) NOT NULL,
	last_value BIGINT NOT NULL,
	updated_at DATETIME(3) NOT NULL,
	PRIMARY KEY (plugin_id, tenant_id, document_type, period_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS sk_document_number_issues (
	plugin_id VARCHAR(64) NOT NULL,
	tenant_id VARCHAR(128) NOT NULL,
	document_type VARCHAR(63) NOT NULL,
	idempotency_key VARCHAR(128) NOT NULL,
	period_key VARCHAR(8) NOT NULL,
	sequence_value BIGINT NOT NULL,
	number VARCHAR(128) NOT NULL,
	request_hash CHAR(64) NOT NULL,
	issued_at DATETIME(3) NOT NULL,
	PRIMARY KEY (plugin_id, tenant_id, document_type, idempotency_key),
	UNIQUE KEY idx_document_number_value (plugin_id, tenant_id, document_type, period_key, sequence_value)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
