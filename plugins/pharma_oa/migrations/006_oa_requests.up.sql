CREATE TABLE {{table:oa_requests}} (
    id VARCHAR(64) PRIMARY KEY,
    request_type VARCHAR(32) NOT NULL,
    title VARCHAR(200) NOT NULL,
    description VARCHAR(5000) NULL,
    form_data TEXT NOT NULL,
    status VARCHAR(32) NOT NULL,
    approver_id VARCHAR(128) NOT NULL,
    approver_name VARCHAR(200) NOT NULL,
    workflow_definition_id VARCHAR(128) NULL,
    workflow_instance_id VARCHAR(128) NULL,
    submitted_at TIMESTAMP NULL,
    last_operation_key VARCHAR(128) NOT NULL,
    tenant_id VARCHAR(128) NOT NULL,
    organization_id VARCHAR(128) NOT NULL,
    owner_id VARCHAR(128) NOT NULL,
    version BIGINT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);
CREATE INDEX idx_pharma_oa_requests_scope ON {{table:oa_requests}} (tenant_id, organization_id, owner_id, status);
CREATE INDEX idx_pharma_oa_requests_type ON {{table:oa_requests}} (tenant_id, request_type, updated_at);
CREATE INDEX idx_pharma_oa_requests_workflow ON {{table:oa_requests}} (workflow_instance_id);
