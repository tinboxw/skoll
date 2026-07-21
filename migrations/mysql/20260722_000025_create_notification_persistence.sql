CREATE TABLE IF NOT EXISTS sk_notification_items (
	id VARCHAR(128) PRIMARY KEY,
	category VARCHAR(32) NOT NULL,
	status VARCHAR(32) NOT NULL,
	title VARCHAR(255) NOT NULL,
	body TEXT NOT NULL,
	actor_id VARCHAR(64) NOT NULL,
	target_type VARCHAR(64) NOT NULL,
	target_id VARCHAR(128) NOT NULL,
	target_path VARCHAR(512) NOT NULL,
	due_at DATETIME(3) NULL,
	created_at DATETIME(3) NOT NULL,
	updated_at DATETIME(3) NOT NULL,
	KEY idx_notification_actor_status_updated (actor_id, category, status, updated_at),
	KEY idx_notification_category_status (category, status),
	KEY idx_notification_target (target_type, target_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS sk_notification_reminder_rules (
	id VARCHAR(128) PRIMARY KEY,
	name VARCHAR(255) NOT NULL,
	actor_id VARCHAR(64) NOT NULL,
	title VARCHAR(255) NOT NULL,
	body TEXT NOT NULL,
	target_type VARCHAR(64) NOT NULL,
	target_id VARCHAR(128) NOT NULL,
	target_path VARCHAR(512) NOT NULL,
	due_at DATETIME(3) NULL,
	interval_nanos BIGINT NOT NULL,
	created_at DATETIME(3) NOT NULL,
	updated_at DATETIME(3) NOT NULL,
	KEY idx_notification_rule_actor_due (actor_id, due_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS sk_notification_delivery_attempts (
	id VARCHAR(128) PRIMARY KEY,
	notification_id VARCHAR(128) NOT NULL,
	channel VARCHAR(64) NOT NULL,
	idempotency_key VARCHAR(128) NOT NULL,
	status VARCHAR(32) NOT NULL,
	attempt INT NOT NULL,
	error TEXT NOT NULL,
	created_at DATETIME(3) NOT NULL,
	UNIQUE KEY idx_notification_delivery_idempotency (notification_id, channel, idempotency_key),
	KEY idx_notification_delivery_stream (notification_id, channel, attempt),
	KEY idx_notification_delivery_status (notification_id, status),
	CONSTRAINT fk_notification_delivery_item FOREIGN KEY (notification_id) REFERENCES sk_notification_items(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
