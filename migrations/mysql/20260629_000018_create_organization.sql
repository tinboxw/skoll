CREATE TABLE IF NOT EXISTS sk_departments (
	id VARCHAR(64) PRIMARY KEY,
	parent_id VARCHAR(64) NOT NULL DEFAULT '',
	code VARCHAR(128) NOT NULL,
	name VARCHAR(255) NOT NULL,
	leader_user_id VARCHAR(64) NOT NULL DEFAULT '',
	status VARCHAR(32) NOT NULL DEFAULT 'enabled',
	sort INT NOT NULL DEFAULT 0,
	created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
	updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
	UNIQUE KEY uk_departments_code (code),
	KEY idx_departments_parent_sort (parent_id, sort),
	KEY idx_departments_status_sort (status, sort),
	KEY idx_departments_leader_user_id (leader_user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS sk_positions (
	id VARCHAR(64) PRIMARY KEY,
	code VARCHAR(128) NOT NULL,
	name VARCHAR(255) NOT NULL,
	description VARCHAR(512) NOT NULL DEFAULT '',
	status VARCHAR(32) NOT NULL DEFAULT 'enabled',
	sort INT NOT NULL DEFAULT 0,
	created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
	updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
	UNIQUE KEY uk_positions_code (code),
	KEY idx_positions_status_sort (status, sort)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS sk_user_organization_assignments (
	user_id VARCHAR(64) PRIMARY KEY,
	department_id VARCHAR(64) NOT NULL,
	position_id VARCHAR(64) NOT NULL DEFAULT '',
	primary_assignment BOOL NOT NULL DEFAULT TRUE,
	created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
	updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
	KEY idx_user_assignments_department (department_id),
	KEY idx_user_assignments_position (position_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
