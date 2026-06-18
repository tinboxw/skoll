CREATE TABLE IF NOT EXISTS sk_audit_events (
	id VARCHAR(64) PRIMARY KEY,
	event_type VARCHAR(32) NOT NULL,
	action VARCHAR(192) NOT NULL,
	actor_type VARCHAR(64) NOT NULL,
	actor_id VARCHAR(64) NOT NULL DEFAULT '',
	actor_name VARCHAR(128) NOT NULL DEFAULT '',
	resource_type VARCHAR(64) NOT NULL,
	resource_id VARCHAR(128) NOT NULL DEFAULT '',
	resource_name VARCHAR(128) NOT NULL DEFAULT '',
	result VARCHAR(32) NOT NULL,
	risk VARCHAR(32) NOT NULL DEFAULT 'low',
	trace_id VARCHAR(128) NOT NULL DEFAULT '',
	request_id VARCHAR(128) NOT NULL DEFAULT '',
	request_method VARCHAR(16) NOT NULL DEFAULT '',
	request_path VARCHAR(512) NOT NULL DEFAULT '',
	request_ip VARCHAR(64) NOT NULL DEFAULT '',
	user_agent VARCHAR(512) NOT NULL DEFAULT '',
	metadata_json TEXT,
	source_json TEXT,
	occurred_at TIMESTAMPTZ NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_audit_events_type_time
	ON sk_audit_events (event_type, occurred_at);

CREATE INDEX IF NOT EXISTS idx_audit_events_actor_time
	ON sk_audit_events (actor_id, occurred_at);

CREATE INDEX IF NOT EXISTS idx_audit_events_action_time
	ON sk_audit_events (action, occurred_at);

CREATE INDEX IF NOT EXISTS idx_audit_events_resource_time
	ON sk_audit_events (resource_type, resource_id, occurred_at);

CREATE INDEX IF NOT EXISTS idx_audit_events_result_time
	ON sk_audit_events (result, occurred_at);

CREATE INDEX IF NOT EXISTS idx_audit_events_risk_time
	ON sk_audit_events (risk, occurred_at);

CREATE INDEX IF NOT EXISTS idx_audit_events_trace
	ON sk_audit_events (trace_id);

CREATE INDEX IF NOT EXISTS idx_audit_events_request
	ON sk_audit_events (request_id);

CREATE INDEX IF NOT EXISTS idx_audit_events_occurred
	ON sk_audit_events (occurred_at);
