CREATE TABLE IF NOT EXISTS sk_permission_resources (
	id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
	permission_key VARCHAR(128) NOT NULL,
	resource_type VARCHAR(32) NOT NULL,
	module VARCHAR(64) NOT NULL,
	source VARCHAR(64) NOT NULL,
	name VARCHAR(128) NOT NULL,
	risk VARCHAR(32) NOT NULL DEFAULT 'low',
	metadata_json TEXT NULL,
	enabled TINYINT(1) NOT NULL DEFAULT 1,
	created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
	updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
	UNIQUE KEY uk_permission_resources_key (permission_key),
	KEY idx_permission_resources_type (resource_type),
	KEY idx_permission_resources_module_type (module, resource_type),
	KEY idx_permission_resources_source_type (source, resource_type),
	KEY idx_permission_resources_risk (risk),
	KEY idx_permission_resources_enabled (enabled)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
