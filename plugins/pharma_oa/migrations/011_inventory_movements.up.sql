CREATE TABLE {{table:inventory_movements}} (
    id VARCHAR(64) PRIMARY KEY,
    number VARCHAR(64) NOT NULL,
    movement_type VARCHAR(32) NOT NULL,
    return_type VARCHAR(32) NULL,
    reason VARCHAR(2000) NOT NULL,
    source_warehouse_id VARCHAR(64) NULL,
    source_area_id VARCHAR(64) NULL,
    source_location_id VARCHAR(64) NULL,
    destination_warehouse_id VARCHAR(64) NULL,
    destination_area_id VARCHAR(64) NULL,
    destination_location_id VARCHAR(64) NULL,
    legs TEXT NOT NULL,
    requester_id VARCHAR(128) NOT NULL,
    requested_at TIMESTAMP NOT NULL,
    approver_id VARCHAR(128) NULL,
    approver_name VARCHAR(200) NULL,
    workflow_definition_id VARCHAR(128) NULL,
    workflow_instance_id VARCHAR(128) NULL,
    status VARCHAR(32) NOT NULL,
    create_operation_key VARCHAR(128) NOT NULL,
    request_hash VARCHAR(64) NOT NULL,
    last_operation_key VARCHAR(128) NOT NULL,
    last_operation_hash VARCHAR(64) NOT NULL,
    decision_comment VARCHAR(1000) NULL,
    decided_at TIMESTAMP NULL,
    posted_at TIMESTAMP NULL,
    tenant_id VARCHAR(128) NOT NULL,
    organization_id VARCHAR(128) NOT NULL,
    owner_id VARCHAR(128) NOT NULL,
    version BIGINT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);
CREATE UNIQUE INDEX uq_pharma_oa_inventory_movements_number ON {{table:inventory_movements}} (tenant_id, organization_id, number);
CREATE UNIQUE INDEX uq_pharma_oa_inventory_movements_create_operation ON {{table:inventory_movements}} (tenant_id, organization_id, create_operation_key);
CREATE INDEX idx_pharma_oa_inventory_movements_type_status ON {{table:inventory_movements}} (tenant_id, organization_id, movement_type, status, updated_at);
CREATE INDEX idx_pharma_oa_inventory_movements_source_location ON {{table:inventory_movements}} (tenant_id, organization_id, source_location_id, status, updated_at);
CREATE INDEX idx_pharma_oa_inventory_movements_destination_location ON {{table:inventory_movements}} (tenant_id, organization_id, destination_location_id, status, updated_at);
CREATE INDEX idx_pharma_oa_inventory_movements_workflow ON {{table:inventory_movements}} (workflow_instance_id);

CREATE TABLE {{table:stock_return_totals}} (
    id VARCHAR(64) PRIMARY KEY,
    reference_ledger_entry_id VARCHAR(64) NOT NULL,
    return_type VARCHAR(32) NOT NULL,
    quantity_micros BIGINT NOT NULL,
    tenant_id VARCHAR(128) NOT NULL,
    organization_id VARCHAR(128) NOT NULL,
    owner_id VARCHAR(128) NOT NULL,
    version BIGINT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);
CREATE UNIQUE INDEX uq_pharma_oa_stock_return_totals_reference ON {{table:stock_return_totals}} (tenant_id, organization_id, reference_ledger_entry_id, return_type);
CREATE INDEX idx_pharma_oa_stock_return_totals_type ON {{table:stock_return_totals}} (tenant_id, organization_id, return_type, updated_at);
