CREATE TABLE {{table:purchase_inbounds}} (
    id VARCHAR(64) PRIMARY KEY,
    number VARCHAR(64) NOT NULL,
    purchase_order_id VARCHAR(64) NOT NULL,
    purchase_order_number VARCHAR(64) NOT NULL,
    warehouse_id VARCHAR(64) NOT NULL,
    area_id VARCHAR(64) NOT NULL,
    location_id VARCHAR(64) NOT NULL,
    lines TEXT NOT NULL,
    attachments TEXT NOT NULL,
    status VARCHAR(32) NOT NULL,
    received_by VARCHAR(128) NOT NULL,
    received_at TIMESTAMP NOT NULL,
    last_operation_key VARCHAR(128) NOT NULL,
    tenant_id VARCHAR(128) NOT NULL,
    organization_id VARCHAR(128) NOT NULL,
    owner_id VARCHAR(128) NOT NULL,
    version BIGINT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);
CREATE UNIQUE INDEX uq_pharma_oa_purchase_inbounds_number ON {{table:purchase_inbounds}} (tenant_id, number);
CREATE UNIQUE INDEX uq_pharma_oa_purchase_inbounds_operation ON {{table:purchase_inbounds}} (tenant_id, last_operation_key);
CREATE INDEX idx_pharma_oa_purchase_inbounds_scope ON {{table:purchase_inbounds}} (tenant_id, organization_id, owner_id, received_at);
CREATE INDEX idx_pharma_oa_purchase_inbounds_order ON {{table:purchase_inbounds}} (tenant_id, purchase_order_id, received_at);
