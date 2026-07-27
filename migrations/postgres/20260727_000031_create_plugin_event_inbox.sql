CREATE TABLE IF NOT EXISTS sk_plugin_event_inbox (
    delivery_id VARCHAR(128) PRIMARY KEY,
    event_id VARCHAR(128) NOT NULL,
    subscriber VARCHAR(64) NOT NULL,
    handler VARCHAR(128) NOT NULL,
    publisher VARCHAR(64) NOT NULL,
    event_name VARCHAR(128) NOT NULL,
    schema_version BIGINT NOT NULL CHECK (schema_version > 0),
    tenant_id VARCHAR(128) NOT NULL DEFAULT '',
    delivery_json TEXT NOT NULL,
    status VARCHAR(24) NOT NULL,
    attempt_count INTEGER NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
    lease_owner VARCHAR(128) NOT NULL DEFAULT '',
    lease_token VARCHAR(64) NOT NULL DEFAULT '',
    lease_expires_at TIMESTAMPTZ NULL,
    last_error TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    processed_at TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS idx_plugin_event_inbox_event
    ON sk_plugin_event_inbox (event_id);

CREATE INDEX IF NOT EXISTS idx_plugin_event_inbox_status
    ON sk_plugin_event_inbox (status, subscriber);

CREATE INDEX IF NOT EXISTS idx_plugin_event_inbox_lease
    ON sk_plugin_event_inbox (lease_expires_at);
