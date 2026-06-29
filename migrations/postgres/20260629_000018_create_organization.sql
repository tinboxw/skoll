CREATE TABLE IF NOT EXISTS sk_departments (
	id VARCHAR(64) PRIMARY KEY,
	parent_id VARCHAR(64) NOT NULL DEFAULT '',
	code VARCHAR(128) NOT NULL,
	name VARCHAR(255) NOT NULL,
	leader_user_id VARCHAR(64) NOT NULL DEFAULT '',
	status VARCHAR(32) NOT NULL DEFAULT 'enabled',
	sort INTEGER NOT NULL DEFAULT 0,
	created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_departments_code
	ON sk_departments (code);

CREATE INDEX IF NOT EXISTS idx_departments_parent_sort
	ON sk_departments (parent_id, sort);

CREATE INDEX IF NOT EXISTS idx_departments_status_sort
	ON sk_departments (status, sort);

CREATE INDEX IF NOT EXISTS idx_departments_leader_user_id
	ON sk_departments (leader_user_id);

CREATE TABLE IF NOT EXISTS sk_positions (
	id VARCHAR(64) PRIMARY KEY,
	code VARCHAR(128) NOT NULL,
	name VARCHAR(255) NOT NULL,
	description VARCHAR(512) NOT NULL DEFAULT '',
	status VARCHAR(32) NOT NULL DEFAULT 'enabled',
	sort INTEGER NOT NULL DEFAULT 0,
	created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_positions_code
	ON sk_positions (code);

CREATE INDEX IF NOT EXISTS idx_positions_status_sort
	ON sk_positions (status, sort);

CREATE TABLE IF NOT EXISTS sk_user_organization_assignments (
	user_id VARCHAR(64) PRIMARY KEY,
	department_id VARCHAR(64) NOT NULL,
	position_id VARCHAR(64) NOT NULL DEFAULT '',
	primary_assignment BOOLEAN NOT NULL DEFAULT TRUE,
	created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_user_assignments_department
	ON sk_user_organization_assignments (department_id);

CREATE INDEX IF NOT EXISTS idx_user_assignments_position
	ON sk_user_organization_assignments (position_id);
