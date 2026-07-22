CREATE TABLE equipment_maintenance_assets (
    id VARCHAR(64) PRIMARY KEY,
    tenant_id VARCHAR(128) NOT NULL,
    organization_id VARCHAR(128) NOT NULL,
    owner_id VARCHAR(128) NOT NULL,
    code VARCHAR(128) NOT NULL,
    name VARCHAR(255) NOT NULL,
    category VARCHAR(128) NOT NULL,
    model VARCHAR(128) NOT NULL,
    serial_number VARCHAR(128) NOT NULL,
    location VARCHAR(255) NOT NULL,
    status VARCHAR(32) NOT NULL,
    next_maintenance_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    CONSTRAINT uq_equipment_assets_tenant_code UNIQUE (tenant_id, code)
);

CREATE INDEX idx_equipment_assets_scope ON equipment_maintenance_assets (tenant_id, organization_id, status);
CREATE INDEX idx_equipment_assets_due ON equipment_maintenance_assets (tenant_id, next_maintenance_at);

CREATE TABLE equipment_maintenance_work_orders (
    id VARCHAR(64) PRIMARY KEY,
    tenant_id VARCHAR(128) NOT NULL,
    organization_id VARCHAR(128) NOT NULL,
    owner_id VARCHAR(128) NOT NULL,
    asset_id VARCHAR(64) NOT NULL,
    number VARCHAR(128) NOT NULL,
    title VARCHAR(255) NOT NULL,
    priority VARCHAR(32) NOT NULL,
    status VARCHAR(32) NOT NULL,
    assignee_id VARCHAR(128) NOT NULL,
    workflow_instance_id VARCHAR(128) NOT NULL,
    estimated_cost DECIMAL(18, 2) NOT NULL,
    started_at TIMESTAMP NULL,
    completed_at TIMESTAMP NULL,
    closed_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    CONSTRAINT uq_equipment_work_orders_tenant_number UNIQUE (tenant_id, number)
);

CREATE INDEX idx_equipment_work_orders_scope ON equipment_maintenance_work_orders (tenant_id, organization_id, status);
CREATE INDEX idx_equipment_work_orders_asset ON equipment_maintenance_work_orders (asset_id, status);

CREATE TABLE equipment_maintenance_plans (
    id VARCHAR(64) PRIMARY KEY,
    tenant_id VARCHAR(128) NOT NULL,
    organization_id VARCHAR(128) NOT NULL,
    owner_id VARCHAR(128) NOT NULL,
    asset_id VARCHAR(64) NOT NULL,
    name VARCHAR(255) NOT NULL,
    interval_days INTEGER NOT NULL,
    next_run_at TIMESTAMP NOT NULL,
    status VARCHAR(32) NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

CREATE INDEX idx_equipment_plans_scope ON equipment_maintenance_plans (tenant_id, organization_id, status);
CREATE INDEX idx_equipment_plans_due ON equipment_maintenance_plans (tenant_id, next_run_at);

CREATE TABLE equipment_maintenance_inspections (
    id VARCHAR(64) PRIMARY KEY,
    tenant_id VARCHAR(128) NOT NULL,
    organization_id VARCHAR(128) NOT NULL,
    owner_id VARCHAR(128) NOT NULL,
    plan_id VARCHAR(64) NOT NULL,
    asset_id VARCHAR(64) NOT NULL,
    work_order_id VARCHAR(64) NOT NULL,
    status VARCHAR(32) NOT NULL,
    result VARCHAR(64) NOT NULL,
    finding TEXT NOT NULL,
    attachment_ids TEXT NOT NULL,
    inspected_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

CREATE INDEX idx_equipment_inspections_scope ON equipment_maintenance_inspections (tenant_id, organization_id, inspected_at);
CREATE INDEX idx_equipment_inspections_asset ON equipment_maintenance_inspections (asset_id, inspected_at);

CREATE TABLE equipment_maintenance_spare_parts (
    id VARCHAR(64) PRIMARY KEY,
    tenant_id VARCHAR(128) NOT NULL,
    organization_id VARCHAR(128) NOT NULL,
    owner_id VARCHAR(128) NOT NULL,
    sku VARCHAR(128) NOT NULL,
    name VARCHAR(255) NOT NULL,
    unit VARCHAR(32) NOT NULL,
    quantity DECIMAL(18, 4) NOT NULL,
    minimum_quantity DECIMAL(18, 4) NOT NULL,
    status VARCHAR(32) NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    CONSTRAINT uq_equipment_spares_tenant_sku UNIQUE (tenant_id, sku)
);

CREATE INDEX idx_equipment_spares_scope ON equipment_maintenance_spare_parts (tenant_id, organization_id, status);

CREATE TABLE equipment_maintenance_spare_movements (
    id VARCHAR(64) PRIMARY KEY,
    tenant_id VARCHAR(128) NOT NULL,
    organization_id VARCHAR(128) NOT NULL,
    owner_id VARCHAR(128) NOT NULL,
    spare_part_id VARCHAR(64) NOT NULL,
    work_order_id VARCHAR(64) NOT NULL,
    movement_type VARCHAR(32) NOT NULL,
    quantity DECIMAL(18, 4) NOT NULL,
    idempotency_key VARCHAR(128) NOT NULL,
    occurred_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL,
    CONSTRAINT uq_equipment_movements_idempotency UNIQUE (tenant_id, idempotency_key)
);

CREATE INDEX idx_equipment_movements_part ON equipment_maintenance_spare_movements (spare_part_id, occurred_at);
CREATE INDEX idx_equipment_movements_work_order ON equipment_maintenance_spare_movements (work_order_id, occurred_at);
