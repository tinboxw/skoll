CREATE TABLE IF NOT EXISTS sk_permission_resources (
	id BIGSERIAL PRIMARY KEY,
	permission_key VARCHAR(128) NOT NULL,
	resource_type VARCHAR(32) NOT NULL,
	module VARCHAR(64) NOT NULL,
	source VARCHAR(64) NOT NULL,
	name VARCHAR(128) NOT NULL,
	risk VARCHAR(32) NOT NULL DEFAULT 'low',
	metadata_json TEXT,
	enabled BOOLEAN NOT NULL DEFAULT TRUE,
	created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_permission_resources_key
	ON sk_permission_resources (permission_key);

CREATE INDEX IF NOT EXISTS idx_permission_resources_type
	ON sk_permission_resources (resource_type);

CREATE INDEX IF NOT EXISTS idx_permission_resources_module_type
	ON sk_permission_resources (module, resource_type);

CREATE INDEX IF NOT EXISTS idx_permission_resources_source_type
	ON sk_permission_resources (source, resource_type);

CREATE INDEX IF NOT EXISTS idx_permission_resources_risk
	ON sk_permission_resources (risk);

CREATE INDEX IF NOT EXISTS idx_permission_resources_enabled
	ON sk_permission_resources (enabled);
