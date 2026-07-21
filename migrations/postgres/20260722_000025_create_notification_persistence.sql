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
	due_at TIMESTAMPTZ NULL,
	created_at TIMESTAMPTZ NOT NULL,
	updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_notification_actor_status_updated ON sk_notification_items (actor_id, category, status, updated_at);
CREATE INDEX IF NOT EXISTS idx_notification_category_status ON sk_notification_items (category, status);
CREATE INDEX IF NOT EXISTS idx_notification_target ON sk_notification_items (target_type, target_id);

CREATE TABLE IF NOT EXISTS sk_notification_reminder_rules (
	id VARCHAR(128) PRIMARY KEY,
	name VARCHAR(255) NOT NULL,
	actor_id VARCHAR(64) NOT NULL,
	title VARCHAR(255) NOT NULL,
	body TEXT NOT NULL,
	target_type VARCHAR(64) NOT NULL,
	target_id VARCHAR(128) NOT NULL,
	target_path VARCHAR(512) NOT NULL,
	due_at TIMESTAMPTZ NULL,
	interval_nanos BIGINT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL,
	updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_notification_rule_actor_due ON sk_notification_reminder_rules (actor_id, due_at);

CREATE TABLE IF NOT EXISTS sk_notification_delivery_attempts (
	id VARCHAR(128) PRIMARY KEY,
	notification_id VARCHAR(128) NOT NULL,
	channel VARCHAR(64) NOT NULL,
	idempotency_key VARCHAR(128) NOT NULL,
	status VARCHAR(32) NOT NULL,
	attempt INTEGER NOT NULL,
	error TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL,
	CONSTRAINT idx_notification_delivery_idempotency UNIQUE (notification_id, channel, idempotency_key),
	CONSTRAINT fk_notification_delivery_item FOREIGN KEY (notification_id) REFERENCES sk_notification_items(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_notification_delivery_stream ON sk_notification_delivery_attempts (notification_id, channel, attempt);
CREATE INDEX IF NOT EXISTS idx_notification_delivery_status ON sk_notification_delivery_attempts (notification_id, status);
