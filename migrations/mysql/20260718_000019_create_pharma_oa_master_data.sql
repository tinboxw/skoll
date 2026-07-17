CREATE TABLE IF NOT EXISTS pharma_oa_employees (
	id VARCHAR(64) PRIMARY KEY,
	code VARCHAR(128) NOT NULL,
	name VARCHAR(255) NOT NULL,
	department_id VARCHAR(64) NOT NULL DEFAULT '',
	position_id VARCHAR(64) NOT NULL DEFAULT '',
	phone VARCHAR(64) NOT NULL DEFAULT '',
	email VARCHAR(255) NOT NULL DEFAULT '',
	status VARCHAR(32) NOT NULL,
	leave_reason VARCHAR(512) NOT NULL DEFAULT '',
	certificates_json LONGTEXT NOT NULL,
	created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
	updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
	created_by VARCHAR(64) NOT NULL DEFAULT '',
	updated_by VARCHAR(64) NOT NULL DEFAULT '',
	UNIQUE KEY uk_pharma_employees_code (code),
	KEY idx_pharma_employees_org_status (department_id, status),
	KEY idx_pharma_employees_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS pharma_oa_products (
	id VARCHAR(64) PRIMARY KEY,
	code VARCHAR(128) NOT NULL,
	name VARCHAR(255) NOT NULL,
	specification VARCHAR(255) NOT NULL DEFAULT '',
	dosage_form VARCHAR(128) NOT NULL DEFAULT '',
	manufacturer VARCHAR(255) NOT NULL DEFAULT '',
	approval_number VARCHAR(191) NOT NULL,
	status VARCHAR(32) NOT NULL,
	disable_reason VARCHAR(512) NOT NULL DEFAULT '',
	temperature_json LONGTEXT NOT NULL,
	created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
	updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
	created_by VARCHAR(64) NOT NULL DEFAULT '',
	updated_by VARCHAR(64) NOT NULL DEFAULT '',
	UNIQUE KEY uk_pharma_products_code (code),
	UNIQUE KEY uk_pharma_products_approval (approval_number),
	KEY idx_pharma_products_status_name (status, name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS pharma_oa_suppliers (
	id VARCHAR(64) PRIMARY KEY,
	code VARCHAR(128) NOT NULL,
	name VARCHAR(255) NOT NULL,
	rating INT NOT NULL DEFAULT 3,
	status VARCHAR(32) NOT NULL,
	disable_reason VARCHAR(512) NOT NULL DEFAULT '',
	contacts_json LONGTEXT NOT NULL,
	qualifications_json LONGTEXT NOT NULL,
	created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
	updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
	created_by VARCHAR(64) NOT NULL DEFAULT '',
	updated_by VARCHAR(64) NOT NULL DEFAULT '',
	UNIQUE KEY uk_pharma_suppliers_code (code),
	KEY idx_pharma_suppliers_status_name (status, name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS pharma_oa_customers (
	id VARCHAR(64) PRIMARY KEY,
	code VARCHAR(128) NOT NULL,
	name VARCHAR(255) NOT NULL,
	region VARCHAR(128) NOT NULL DEFAULT '',
	organization_id VARCHAR(64) NOT NULL DEFAULT '',
	owner_id VARCHAR(64) NOT NULL DEFAULT '',
	rating INT NOT NULL DEFAULT 3,
	status VARCHAR(32) NOT NULL,
	disable_reason VARCHAR(512) NOT NULL DEFAULT '',
	contacts_json LONGTEXT NOT NULL,
	qualifications_json LONGTEXT NOT NULL,
	created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
	updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
	created_by VARCHAR(64) NOT NULL DEFAULT '',
	updated_by VARCHAR(64) NOT NULL DEFAULT '',
	UNIQUE KEY uk_pharma_customers_code (code),
	KEY idx_pharma_customers_scope_status (organization_id, owner_id, status),
	KEY idx_pharma_customers_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS pharma_oa_warehouses (
	id VARCHAR(64) PRIMARY KEY,
	code VARCHAR(128) NOT NULL,
	name VARCHAR(255) NOT NULL,
	region VARCHAR(128) NOT NULL DEFAULT '',
	status VARCHAR(32) NOT NULL,
	disable_reason VARCHAR(512) NOT NULL DEFAULT '',
	temperature_json LONGTEXT NOT NULL,
	areas_json LONGTEXT NOT NULL,
	created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
	updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
	created_by VARCHAR(64) NOT NULL DEFAULT '',
	updated_by VARCHAR(64) NOT NULL DEFAULT '',
	UNIQUE KEY uk_pharma_warehouses_code (code),
	KEY idx_pharma_warehouses_status_name (status, name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
