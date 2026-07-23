CREATE TABLE {{table:parties}} (
    id VARCHAR(64) PRIMARY KEY, party_type VARCHAR(16) NOT NULL, code VARCHAR(64) NOT NULL, name VARCHAR(255) NOT NULL,
    unified_social_credit_code VARCHAR(32) NOT NULL, region VARCHAR(128) NOT NULL, rating BIGINT NOT NULL, status VARCHAR(32) NOT NULL,
    disable_reason VARCHAR(512) NULL, contacts TEXT NOT NULL, addresses TEXT NOT NULL, settlement_terms TEXT NOT NULL,
    tenant_id VARCHAR(128) NOT NULL, organization_id VARCHAR(128) NOT NULL, owner_id VARCHAR(128) NOT NULL,
    version BIGINT NOT NULL, created_at TIMESTAMP NOT NULL, updated_at TIMESTAMP NOT NULL,
    CONSTRAINT uq_pharma_oa_parties_tenant_type_code UNIQUE (tenant_id, party_type, code),
    CONSTRAINT uq_pharma_oa_parties_tenant_type_credit UNIQUE (tenant_id, party_type, unified_social_credit_code)
);
CREATE INDEX idx_pharma_oa_parties_scope ON {{table:parties}} (tenant_id, organization_id, party_type, status);
CREATE INDEX idx_pharma_oa_parties_name ON {{table:parties}} (tenant_id, party_type, name);
