ALTER TABLE sk_workflow_nodes
	ADD COLUMN escalation_after_seconds BIGINT,
	ADD COLUMN escalation_target_id VARCHAR(64),
	ADD COLUMN escalation_target_name VARCHAR(255);

ALTER TABLE sk_workflow_tasks
	ADD COLUMN original_assignee_id VARCHAR(64) NOT NULL,
	ADD COLUMN original_assignee_name VARCHAR(255) NOT NULL,
	ADD COLUMN assignment VARCHAR(32) NOT NULL,
	ADD COLUMN authorized_by_id VARCHAR(64),
	ADD COLUMN authorized_by_name VARCHAR(255),
	ADD COLUMN authorization_id VARCHAR(128);

CREATE TABLE sk_workflow_substitutions (
	id VARCHAR(128) PRIMARY KEY,
	principal_id VARCHAR(64) NOT NULL,
	principal_name VARCHAR(255) NOT NULL,
	substitute_id VARCHAR(64) NOT NULL,
	substitute_name VARCHAR(255) NOT NULL,
	starts_at TIMESTAMPTZ NOT NULL,
	ends_at TIMESTAMPTZ NOT NULL,
	created_by_id VARCHAR(64) NOT NULL,
	created_by_name VARCHAR(255) NOT NULL,
	reason TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL,
	revoked_at TIMESTAMPTZ
);

CREATE INDEX idx_workflow_substitution_principal_window
	ON sk_workflow_substitutions (principal_id, starts_at, ends_at);
CREATE INDEX idx_workflow_substitution_substitute
	ON sk_workflow_substitutions (substitute_id);
