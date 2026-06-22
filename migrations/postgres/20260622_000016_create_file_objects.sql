CREATE TABLE IF NOT EXISTS sk_file_objects (
	id VARCHAR(64) PRIMARY KEY,
	object_key VARCHAR(256) NOT NULL,
	name VARCHAR(255) NOT NULL,
	size_bytes BIGINT NOT NULL DEFAULT 0,
	mime VARCHAR(128) NOT NULL,
	hash VARCHAR(256) NOT NULL,
	owner_type VARCHAR(64) NOT NULL,
	owner_id VARCHAR(64) NOT NULL,
	visibility VARCHAR(32) NOT NULL DEFAULT 'private',
	storage_driver VARCHAR(64) NOT NULL,
	status VARCHAR(32) NOT NULL DEFAULT 'pending',
	source_module VARCHAR(64) NOT NULL,
	source_plugin_id VARCHAR(64) NOT NULL DEFAULT '',
	metadata_json TEXT,
	created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_file_objects_key
	ON sk_file_objects (object_key);

CREATE INDEX IF NOT EXISTS idx_file_objects_owner
	ON sk_file_objects (owner_type, owner_id);

CREATE INDEX IF NOT EXISTS idx_file_objects_visibility_status
	ON sk_file_objects (visibility, status);

CREATE INDEX IF NOT EXISTS idx_file_objects_storage_status
	ON sk_file_objects (storage_driver, status);

CREATE INDEX IF NOT EXISTS idx_file_objects_source
	ON sk_file_objects (source_module, source_plugin_id);

CREATE INDEX IF NOT EXISTS idx_file_objects_status_updated
	ON sk_file_objects (status, updated_at);

CREATE INDEX IF NOT EXISTS idx_file_objects_hash
	ON sk_file_objects (hash);
