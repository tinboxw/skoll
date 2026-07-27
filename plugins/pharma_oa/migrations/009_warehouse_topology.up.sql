CREATE TABLE {{table:warehouses}} (
    id VARCHAR(64) PRIMARY KEY,
    code VARCHAR(64) NOT NULL,
    name VARCHAR(160) NOT NULL,
    address VARCHAR(500) NOT NULL,
    contact_name VARCHAR(120) NOT NULL,
    contact_phone VARCHAR(64) NOT NULL,
    status VARCHAR(32) NOT NULL,
    disable_reason VARCHAR(500),
    tenant_id VARCHAR(128) NOT NULL,
    organization_id VARCHAR(128) NOT NULL,
    owner_id VARCHAR(128) NOT NULL,
    version BIGINT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);
CREATE UNIQUE INDEX uq_pharma_oa_warehouses_code ON {{table:warehouses}} (tenant_id, organization_id, code);
CREATE INDEX idx_pharma_oa_warehouses_scope ON {{table:warehouses}} (tenant_id, organization_id, status, updated_at);

CREATE TABLE {{table:warehouse_areas}} (
    id VARCHAR(64) PRIMARY KEY,
    warehouse_id VARCHAR(64) NOT NULL,
    code VARCHAR(64) NOT NULL,
    name VARCHAR(160) NOT NULL,
    temperature_min DECIMAL(8,3),
    temperature_max DECIMAL(8,3),
    status VARCHAR(32) NOT NULL,
    disable_reason VARCHAR(500),
    tenant_id VARCHAR(128) NOT NULL,
    organization_id VARCHAR(128) NOT NULL,
    owner_id VARCHAR(128) NOT NULL,
    version BIGINT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);
CREATE UNIQUE INDEX uq_pharma_oa_warehouse_areas_code ON {{table:warehouse_areas}} (tenant_id, warehouse_id, code);
CREATE INDEX idx_pharma_oa_warehouse_areas_scope ON {{table:warehouse_areas}} (tenant_id, organization_id, warehouse_id, status, updated_at);

CREATE TABLE {{table:warehouse_locations}} (
    id VARCHAR(64) PRIMARY KEY,
    warehouse_id VARCHAR(64) NOT NULL,
    area_id VARCHAR(64) NOT NULL,
    code VARCHAR(64) NOT NULL,
    name VARCHAR(160) NOT NULL,
    location_type VARCHAR(32) NOT NULL,
    status VARCHAR(32) NOT NULL,
    disable_reason VARCHAR(500),
    tenant_id VARCHAR(128) NOT NULL,
    organization_id VARCHAR(128) NOT NULL,
    owner_id VARCHAR(128) NOT NULL,
    version BIGINT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);
CREATE UNIQUE INDEX uq_pharma_oa_warehouse_locations_code ON {{table:warehouse_locations}} (tenant_id, area_id, code);
CREATE INDEX idx_pharma_oa_warehouse_locations_scope ON {{table:warehouse_locations}} (tenant_id, organization_id, warehouse_id, area_id, status, updated_at);
