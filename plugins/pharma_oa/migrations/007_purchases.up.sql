CREATE TABLE {{table:purchase_requests}} (
    id VARCHAR(64) PRIMARY KEY,
    number VARCHAR(64) NOT NULL,
    supplier_id VARCHAR(64) NOT NULL,
    supplier_code VARCHAR(64) NOT NULL,
    supplier_name VARCHAR(255) NOT NULL,
    requester_id VARCHAR(128) NOT NULL,
    approver_id VARCHAR(128) NOT NULL,
    approver_name VARCHAR(200) NOT NULL,
    reason VARCHAR(2000) NOT NULL,
    currency VARCHAR(3) NOT NULL,
    lines TEXT NOT NULL,
    total_amount DECIMAL(18,2) NOT NULL,
    status VARCHAR(32) NOT NULL,
    workflow_definition_id VARCHAR(128) NOT NULL,
    workflow_instance_id VARCHAR(128) NOT NULL,
    purchase_order_id VARCHAR(64) NULL,
    decision_comment VARCHAR(1000) NULL,
    decided_at TIMESTAMP NULL,
    last_operation_key VARCHAR(128) NOT NULL,
    tenant_id VARCHAR(128) NOT NULL,
    organization_id VARCHAR(128) NOT NULL,
    owner_id VARCHAR(128) NOT NULL,
    version BIGINT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);
CREATE UNIQUE INDEX uq_pharma_oa_purchase_requests_number ON {{table:purchase_requests}} (tenant_id, number);
CREATE UNIQUE INDEX uq_pharma_oa_purchase_requests_operation ON {{table:purchase_requests}} (tenant_id, last_operation_key);
CREATE INDEX idx_pharma_oa_purchase_requests_scope ON {{table:purchase_requests}} (tenant_id, organization_id, owner_id, status);
CREATE INDEX idx_pharma_oa_purchase_requests_supplier ON {{table:purchase_requests}} (tenant_id, supplier_id, updated_at);
CREATE INDEX idx_pharma_oa_purchase_requests_workflow ON {{table:purchase_requests}} (workflow_instance_id);

CREATE TABLE {{table:purchase_orders}} (
    id VARCHAR(64) PRIMARY KEY,
    number VARCHAR(64) NOT NULL,
    purchase_request_id VARCHAR(64) NOT NULL,
    supplier_id VARCHAR(64) NOT NULL,
    supplier_code VARCHAR(64) NOT NULL,
    supplier_name VARCHAR(255) NOT NULL,
    currency VARCHAR(3) NOT NULL,
    lines TEXT NOT NULL,
    total_amount DECIMAL(18,2) NOT NULL,
    status VARCHAR(32) NOT NULL,
    approved_by VARCHAR(128) NOT NULL,
    approved_at TIMESTAMP NOT NULL,
    last_operation_key VARCHAR(128) NOT NULL,
    tenant_id VARCHAR(128) NOT NULL,
    organization_id VARCHAR(128) NOT NULL,
    owner_id VARCHAR(128) NOT NULL,
    version BIGINT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);
CREATE UNIQUE INDEX uq_pharma_oa_purchase_orders_number ON {{table:purchase_orders}} (tenant_id, number);
CREATE UNIQUE INDEX uq_pharma_oa_purchase_orders_request ON {{table:purchase_orders}} (tenant_id, purchase_request_id);
CREATE INDEX idx_pharma_oa_purchase_orders_scope ON {{table:purchase_orders}} (tenant_id, organization_id, owner_id, status);
CREATE INDEX idx_pharma_oa_purchase_orders_supplier ON {{table:purchase_orders}} (tenant_id, supplier_id, updated_at);
