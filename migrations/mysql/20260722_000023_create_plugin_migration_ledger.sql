CREATE TABLE IF NOT EXISTS sk_plugin_migrations (
	id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
	plugin_id VARCHAR(128) NOT NULL,
	version BIGINT NOT NULL,
	name VARCHAR(255) NOT NULL,
	checksum VARCHAR(64) NOT NULL,
	applied_at DATETIME(3) NOT NULL,
	UNIQUE KEY idx_plugin_migration_version (plugin_id, version),
	KEY idx_sk_plugin_migrations_applied_at (applied_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
