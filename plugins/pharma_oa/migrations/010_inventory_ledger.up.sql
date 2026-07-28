ALTER TABLE {{table:purchase_inbounds}} ADD COLUMN request_hash VARCHAR(64) NOT NULL DEFAULT '';

CREATE TABLE {{table:inventory_lots}} (
    id VARCHAR(64) PRIMARY KEY,
    product_id VARCHAR(64) NOT NULL,
    batch_no VARCHAR(128) NOT NULL,
    production_date TIMESTAMP NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    tenant_id VARCHAR(128) NOT NULL,
    organization_id VARCHAR(128) NOT NULL,
    owner_id VARCHAR(128) NOT NULL,
    version BIGINT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);
CREATE UNIQUE INDEX uq_pharma_oa_inventory_lots_product_batch ON {{table:inventory_lots}} (tenant_id, organization_id, product_id, batch_no);
CREATE INDEX idx_pharma_oa_inventory_lots_product ON {{table:inventory_lots}} (tenant_id, organization_id, product_id, expires_at);
CREATE INDEX idx_pharma_oa_inventory_lots_batch ON {{table:inventory_lots}} (tenant_id, organization_id, batch_no, expires_at);

CREATE TABLE {{table:stock_ledger}} (
    id VARCHAR(64) PRIMARY KEY,
    entry_type VARCHAR(32) NOT NULL,
    product_id VARCHAR(64) NOT NULL,
    lot_id VARCHAR(64) NOT NULL,
    batch_no VARCHAR(128) NOT NULL,
    warehouse_id VARCHAR(64) NOT NULL,
    area_id VARCHAR(64) NOT NULL,
    location_id VARCHAR(64) NOT NULL,
    quantity_micros BIGINT NOT NULL,
    source_document_type VARCHAR(64) NOT NULL,
    source_document_id VARCHAR(64) NOT NULL,
    source_document_number VARCHAR(64) NOT NULL,
    source_document_line_id VARCHAR(128) NOT NULL,
    occurred_at TIMESTAMP NOT NULL,
    tenant_id VARCHAR(128) NOT NULL,
    organization_id VARCHAR(128) NOT NULL,
    owner_id VARCHAR(128) NOT NULL,
    version BIGINT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);
CREATE UNIQUE INDEX uq_pharma_oa_stock_ledger_source_line ON {{table:stock_ledger}} (tenant_id, organization_id, source_document_type, source_document_id, source_document_line_id);
CREATE INDEX idx_pharma_oa_stock_ledger_product ON {{table:stock_ledger}} (tenant_id, organization_id, product_id, occurred_at);
CREATE INDEX idx_pharma_oa_stock_ledger_batch ON {{table:stock_ledger}} (tenant_id, organization_id, batch_no, occurred_at);
CREATE INDEX idx_pharma_oa_stock_ledger_warehouse ON {{table:stock_ledger}} (tenant_id, organization_id, warehouse_id, occurred_at);
CREATE INDEX idx_pharma_oa_stock_ledger_location ON {{table:stock_ledger}} (tenant_id, organization_id, location_id, occurred_at);
CREATE INDEX idx_pharma_oa_stock_ledger_source ON {{table:stock_ledger}} (tenant_id, organization_id, source_document_type, source_document_number, occurred_at);

CREATE TABLE {{table:stock_balances}} (
    id VARCHAR(64) PRIMARY KEY,
    product_id VARCHAR(64) NOT NULL,
    lot_id VARCHAR(64) NOT NULL,
    batch_no VARCHAR(128) NOT NULL,
    warehouse_id VARCHAR(64) NOT NULL,
    area_id VARCHAR(64) NOT NULL,
    location_id VARCHAR(64) NOT NULL,
    quantity_micros BIGINT NOT NULL,
    tenant_id VARCHAR(128) NOT NULL,
    organization_id VARCHAR(128) NOT NULL,
    owner_id VARCHAR(128) NOT NULL,
    version BIGINT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);
CREATE UNIQUE INDEX uq_pharma_oa_stock_balances_projection ON {{table:stock_balances}} (tenant_id, organization_id, product_id, lot_id, location_id);
CREATE INDEX idx_pharma_oa_stock_balances_product_batch ON {{table:stock_balances}} (tenant_id, organization_id, product_id, batch_no);
CREATE INDEX idx_pharma_oa_stock_balances_warehouse ON {{table:stock_balances}} (tenant_id, organization_id, warehouse_id, product_id);
CREATE INDEX idx_pharma_oa_stock_balances_location ON {{table:stock_balances}} (tenant_id, organization_id, location_id, product_id);
