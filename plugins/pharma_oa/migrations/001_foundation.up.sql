CREATE TABLE pharma_oa_module_registry (
    id VARCHAR(64) PRIMARY KEY,
    tenant_id VARCHAR(128) NOT NULL,
    organization_id VARCHAR(128) NOT NULL,
    owner_id VARCHAR(128) NOT NULL,
    module_key VARCHAR(128) NOT NULL,
    status VARCHAR(32) NOT NULL,
    contract_version VARCHAR(32) NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    CONSTRAINT uq_pharma_oa_modules_tenant_key UNIQUE (tenant_id, module_key)
);

CREATE INDEX idx_pharma_oa_modules_scope ON pharma_oa_module_registry (tenant_id, organization_id, status);

CREATE TABLE pharma_oa_document_type_registry (
    id VARCHAR(64) PRIMARY KEY,
    tenant_id VARCHAR(128) NOT NULL,
    organization_id VARCHAR(128) NOT NULL,
    owner_id VARCHAR(128) NOT NULL,
    document_type VARCHAR(128) NOT NULL,
    module_key VARCHAR(128) NOT NULL,
    workflow_key VARCHAR(128) NOT NULL,
    status VARCHAR(32) NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    CONSTRAINT uq_pharma_oa_documents_tenant_type UNIQUE (tenant_id, document_type)
);

CREATE INDEX idx_pharma_oa_documents_scope ON pharma_oa_document_type_registry (tenant_id, organization_id, status);
