CREATE TABLE {{table:employees}} (
    id VARCHAR(64) PRIMARY KEY,
    code VARCHAR(64) NOT NULL,
    name VARCHAR(255) NOT NULL,
    department_id VARCHAR(128) NOT NULL,
    position_id VARCHAR(128) NOT NULL,
    employment_status VARCHAR(32) NOT NULL,
    phone VARCHAR(64) NOT NULL,
    email VARCHAR(255) NOT NULL,
    hire_date TIMESTAMP NOT NULL,
    left_at TIMESTAMP NULL,
    leave_reason VARCHAR(512) NULL,
    attachments TEXT NOT NULL,
    certificates TEXT NOT NULL,
    tenant_id VARCHAR(128) NOT NULL,
    organization_id VARCHAR(128) NOT NULL,
    owner_id VARCHAR(128) NOT NULL,
    version BIGINT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    CONSTRAINT uq_pharma_oa_employees_tenant_code UNIQUE (tenant_id, code)
);

CREATE INDEX idx_pharma_oa_employees_scope ON {{table:employees}} (tenant_id, organization_id, employment_status);
CREATE INDEX idx_pharma_oa_employees_name ON {{table:employees}} (tenant_id, name);
CREATE INDEX idx_pharma_oa_employees_assignment ON {{table:employees}} (tenant_id, department_id, position_id);
