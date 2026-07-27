ALTER TABLE sk_workflow_nodes
	ADD COLUMN escalation_after_seconds BIGINT NULL,
	ADD COLUMN escalation_target_id VARCHAR(64) NULL,
	ADD COLUMN escalation_target_name VARCHAR(255) NULL;

ALTER TABLE sk_workflow_tasks
	ADD COLUMN original_assignee_id VARCHAR(64) NOT NULL,
	ADD COLUMN original_assignee_name VARCHAR(255) NOT NULL,
	ADD COLUMN assignment VARCHAR(32) NOT NULL,
	ADD COLUMN authorized_by_id VARCHAR(64) NULL,
	ADD COLUMN authorized_by_name VARCHAR(255) NULL,
	ADD COLUMN authorization_id VARCHAR(128) NULL;

CREATE TABLE sk_workflow_substitutions (
	id VARCHAR(128) PRIMARY KEY,
	principal_id VARCHAR(64) NOT NULL,
	principal_name VARCHAR(255) NOT NULL,
	substitute_id VARCHAR(64) NOT NULL,
	substitute_name VARCHAR(255) NOT NULL,
	starts_at DATETIME(3) NOT NULL,
	ends_at DATETIME(3) NOT NULL,
	created_by_id VARCHAR(64) NOT NULL,
	created_by_name VARCHAR(255) NOT NULL,
	reason TEXT NOT NULL,
	created_at DATETIME(3) NOT NULL,
	revoked_at DATETIME(3) NULL,
	KEY idx_workflow_substitution_principal_window (principal_id, starts_at, ends_at),
	KEY idx_workflow_substitution_substitute (substitute_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
