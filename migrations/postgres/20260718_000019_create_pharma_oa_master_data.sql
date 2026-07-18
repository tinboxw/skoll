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
	certificates_json TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
	created_by VARCHAR(64) NOT NULL DEFAULT '',
	updated_by VARCHAR(64) NOT NULL DEFAULT ''
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_pharma_employees_code ON pharma_oa_employees (code);
CREATE INDEX IF NOT EXISTS idx_pharma_employees_org_status ON pharma_oa_employees (department_id, status);
CREATE INDEX IF NOT EXISTS idx_pharma_employees_name ON pharma_oa_employees (name);

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
	temperature_json TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
	created_by VARCHAR(64) NOT NULL DEFAULT '',
	updated_by VARCHAR(64) NOT NULL DEFAULT ''
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_pharma_products_code ON pharma_oa_products (code);
CREATE UNIQUE INDEX IF NOT EXISTS uk_pharma_products_approval ON pharma_oa_products (approval_number);
CREATE INDEX IF NOT EXISTS idx_pharma_products_status_name ON pharma_oa_products (status, name);

CREATE TABLE IF NOT EXISTS pharma_oa_suppliers (
	id VARCHAR(64) PRIMARY KEY,
	code VARCHAR(128) NOT NULL,
	name VARCHAR(255) NOT NULL,
	rating INTEGER NOT NULL DEFAULT 3,
	status VARCHAR(32) NOT NULL,
	disable_reason VARCHAR(512) NOT NULL DEFAULT '',
	contacts_json TEXT NOT NULL,
	qualifications_json TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
	created_by VARCHAR(64) NOT NULL DEFAULT '',
	updated_by VARCHAR(64) NOT NULL DEFAULT ''
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_pharma_suppliers_code ON pharma_oa_suppliers (code);
CREATE INDEX IF NOT EXISTS idx_pharma_suppliers_status_name ON pharma_oa_suppliers (status, name);

CREATE TABLE IF NOT EXISTS pharma_oa_customers (
	id VARCHAR(64) PRIMARY KEY,
	code VARCHAR(128) NOT NULL,
	name VARCHAR(255) NOT NULL,
	region VARCHAR(128) NOT NULL DEFAULT '',
	organization_id VARCHAR(64) NOT NULL DEFAULT '',
	owner_id VARCHAR(64) NOT NULL DEFAULT '',
	rating INTEGER NOT NULL DEFAULT 3,
	status VARCHAR(32) NOT NULL,
	disable_reason VARCHAR(512) NOT NULL DEFAULT '',
	contacts_json TEXT NOT NULL,
	qualifications_json TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
	created_by VARCHAR(64) NOT NULL DEFAULT '',
	updated_by VARCHAR(64) NOT NULL DEFAULT ''
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_pharma_customers_code ON pharma_oa_customers (code);
CREATE INDEX IF NOT EXISTS idx_pharma_customers_scope_status ON pharma_oa_customers (organization_id, owner_id, status);
CREATE INDEX IF NOT EXISTS idx_pharma_customers_name ON pharma_oa_customers (name);

CREATE TABLE IF NOT EXISTS pharma_oa_warehouses (
	id VARCHAR(64) PRIMARY KEY,
	code VARCHAR(128) NOT NULL,
	name VARCHAR(255) NOT NULL,
	region VARCHAR(128) NOT NULL DEFAULT '',
	status VARCHAR(32) NOT NULL,
	disable_reason VARCHAR(512) NOT NULL DEFAULT '',
	temperature_json TEXT NOT NULL,
	areas_json TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
	created_by VARCHAR(64) NOT NULL DEFAULT '',
	updated_by VARCHAR(64) NOT NULL DEFAULT ''
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_pharma_warehouses_code ON pharma_oa_warehouses (code);
CREATE INDEX IF NOT EXISTS idx_pharma_warehouses_status_name ON pharma_oa_warehouses (status, name);
