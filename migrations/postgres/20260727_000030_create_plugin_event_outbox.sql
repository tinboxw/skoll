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
    occurred_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT uq_plugin_event_idempotency UNIQUE (publisher, idempotency_key)
);

CREATE INDEX IF NOT EXISTS idx_plugin_event_pending
    ON sk_plugin_event_outbox (status, publisher, tenant_id, created_at);

CREATE INDEX IF NOT EXISTS idx_plugin_event_correlation
    ON sk_plugin_event_outbox (correlation_id);
