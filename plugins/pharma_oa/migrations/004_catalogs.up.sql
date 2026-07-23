CREATE TABLE {{table:catalogs}} (
    id VARCHAR(64) PRIMARY KEY,
    catalog_type VARCHAR(24) NOT NULL,
    code VARCHAR(64) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description VARCHAR(512) NOT NULL,
    parent_id VARCHAR(64) NULL,
    symbol VARCHAR(32) NULL,
    decimal_places BIGINT NOT NULL,
    unified_social_credit_code VARCHAR(32) NULL,
    license_number VARCHAR(128) NULL,
    status VARCHAR(32) NOT NULL,
    disable_reason VARCHAR(512) NULL,
    tenant_id VARCHAR(128) NOT NULL,
    organization_id VARCHAR(128) NOT NULL,
    owner_id VARCHAR(128) NOT NULL,
    version BIGINT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    CONSTRAINT uq_pharma_oa_catalogs_tenant_type_code UNIQUE (tenant_id, catalog_type, code),
    CONSTRAINT uq_pharma_oa_catalogs_tenant_type_credit UNIQUE (tenant_id, catalog_type, unified_social_credit_code)
);
CREATE INDEX idx_pharma_oa_catalogs_scope ON {{table:catalogs}} (tenant_id, organization_id, catalog_type, status);
CREATE INDEX idx_pharma_oa_catalogs_name ON {{table:catalogs}} (tenant_id, catalog_type, name);

CREATE TABLE {{table:products}} (
    id VARCHAR(64) PRIMARY KEY,
    code VARCHAR(64) NOT NULL,
    sku VARCHAR(64) NOT NULL,
    name VARCHAR(255) NOT NULL,
    generic_name VARCHAR(255) NOT NULL,
    category_id VARCHAR(64) NOT NULL,
    unit_id VARCHAR(64) NOT NULL,
    manufacturer_id VARCHAR(64) NOT NULL,
    dosage_form VARCHAR(128) NOT NULL,
    specification VARCHAR(255) NOT NULL,
    approval_number VARCHAR(128) NOT NULL,
    barcode VARCHAR(64) NULL,
    storage_condition VARCHAR(255) NOT NULL,
    temperature_min BIGINT NOT NULL,
    temperature_max BIGINT NOT NULL,
    status VARCHAR(32) NOT NULL,
    disable_reason VARCHAR(512) NULL,
    tenant_id VARCHAR(128) NOT NULL,
    organization_id VARCHAR(128) NOT NULL,
    owner_id VARCHAR(128) NOT NULL,
    version BIGINT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    CONSTRAINT uq_pharma_oa_products_tenant_code UNIQUE (tenant_id, code),
    CONSTRAINT uq_pharma_oa_products_tenant_sku UNIQUE (tenant_id, sku),
    CONSTRAINT uq_pharma_oa_products_tenant_approval UNIQUE (tenant_id, approval_number),
    CONSTRAINT fk_pharma_oa_products_category FOREIGN KEY (category_id) REFERENCES {{table:catalogs}} (id),
    CONSTRAINT fk_pharma_oa_products_unit FOREIGN KEY (unit_id) REFERENCES {{table:catalogs}} (id),
    CONSTRAINT fk_pharma_oa_products_manufacturer FOREIGN KEY (manufacturer_id) REFERENCES {{table:catalogs}} (id)
);
CREATE INDEX idx_pharma_oa_products_scope ON {{table:products}} (tenant_id, organization_id, status);
CREATE INDEX idx_pharma_oa_products_name ON {{table:products}} (tenant_id, name);
CREATE INDEX idx_pharma_oa_products_references ON {{table:products}} (tenant_id, category_id, unit_id, manufacturer_id);
