CREATE TABLE IF NOT EXISTS sk_jobs (
	id VARCHAR(128) PRIMARY KEY,
	namespace VARCHAR(128) NOT NULL,
	kind VARCHAR(128) NOT NULL,
	idempotency_key VARCHAR(191) NULL,
	correlation_id VARCHAR(128) NOT NULL,
	payload_json TEXT NOT NULL,
	status VARCHAR(32) NOT NULL,
	run_at TIMESTAMPTZ NOT NULL,
	max_attempts INTEGER NOT NULL,
	attempt_count INTEGER NOT NULL,
	lease_owner VARCHAR(128) NOT NULL,
	lease_token VARCHAR(64) NOT NULL,
	lease_expires_at TIMESTAMPTZ NULL,
	last_error TEXT NOT NULL,
	result_json TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL,
	updated_at TIMESTAMPTZ NOT NULL,
	completed_at TIMESTAMPTZ NULL,
	dead_lettered_at TIMESTAMPTZ NULL,
	CONSTRAINT idx_job_idempotency UNIQUE (namespace, idempotency_key)
);

CREATE INDEX IF NOT EXISTS idx_job_namespace_status ON sk_jobs (namespace, status);
CREATE INDEX IF NOT EXISTS idx_job_kind_status ON sk_jobs (kind, status);
CREATE INDEX IF NOT EXISTS idx_job_correlation ON sk_jobs (correlation_id);
CREATE INDEX IF NOT EXISTS idx_job_due ON sk_jobs (status, run_at);
CREATE INDEX IF NOT EXISTS idx_job_lease_expiry ON sk_jobs (lease_expires_at);
CREATE INDEX IF NOT EXISTS idx_job_dead_lettered ON sk_jobs (dead_lettered_at);
