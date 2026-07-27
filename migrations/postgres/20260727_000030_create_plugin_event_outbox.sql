CREATE TABLE IF NOT EXISTS sk_plugin_event_outbox (
    id VARCHAR(128) PRIMARY KEY,
    publisher VARCHAR(64) NOT NULL,
    idempotency_key VARCHAR(128) NOT NULL,
    event_name VARCHAR(128) NOT NULL,
    schema_version BIGINT NOT NULL CHECK (schema_version > 0),
    payload_type VARCHAR(128) NOT NULL,
    tenant_id VARCHAR(128) NOT NULL DEFAULT '',
    organization_id VARCHAR(128) NOT NULL DEFAULT '',
    owner_id VARCHAR(128) NOT NULL DEFAULT '',
    correlation_id VARCHAR(128) NOT NULL,
    causation_id VARCHAR(128) NOT NULL DEFAULT '',
    subject_type VARCHAR(128) NOT NULL DEFAULT '',
    subject_id VARCHAR(128) NOT NULL DEFAULT '',
    payload_json TEXT NOT NULL,
    request_hash VARCHAR(64) NOT NULL,
    status VARCHAR(24) NOT NULL,
    attempt_count INTEGER NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
    max_attempts INTEGER NOT NULL DEFAULT 5 CHECK (max_attempts > 0),
    next_attempt_at TIMESTAMPTZ NOT NULL,
    lease_owner VARCHAR(128) NOT NULL DEFAULT '',
    lease_token VARCHAR(64) NOT NULL DEFAULT '',
    lease_expires_at TIMESTAMPTZ NULL,
    last_error TEXT NOT NULL DEFAULT '',
    occurred_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    delivered_at TIMESTAMPTZ NULL,
    dead_lettered_at TIMESTAMPTZ NULL,
    CONSTRAINT uq_plugin_event_idempotency UNIQUE (publisher, idempotency_key)
);

CREATE INDEX IF NOT EXISTS idx_plugin_event_dispatch
    ON sk_plugin_event_outbox (status, next_attempt_at, publisher);

CREATE INDEX IF NOT EXISTS idx_plugin_event_lease_expiry
    ON sk_plugin_event_outbox (lease_expires_at);

CREATE INDEX IF NOT EXISTS idx_plugin_event_correlation
    ON sk_plugin_event_outbox (correlation_id);
