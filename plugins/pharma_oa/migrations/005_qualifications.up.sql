CREATE TABLE {{table:qualification_types}} (
    id VARCHAR(64) PRIMARY KEY,
    code VARCHAR(64) NOT NULL,
    name VARCHAR(255) NOT NULL,
    subject_type VARCHAR(32) NOT NULL,
    business_gate VARCHAR(32) NOT NULL,
    description VARCHAR(512) NOT NULL,
    validity_days BIGINT NOT NULL,
    alert_days BIGINT NOT NULL,
    evidence_required BOOLEAN NOT NULL,
    business_required BOOLEAN NOT NULL,
    status VARCHAR(32) NOT NULL,
    disable_reason VARCHAR(512) NULL,
    tenant_id VARCHAR(128) NOT NULL,
    organization_id VARCHAR(128) NOT NULL,
    owner_id VARCHAR(128) NOT NULL,
    version BIGINT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    CONSTRAINT uq_pharma_oa_qualification_types_tenant_code UNIQUE (tenant_id, code)
);
CREATE INDEX idx_pharma_oa_qualification_types_scope ON {{table:qualification_types}} (tenant_id, organization_id, subject_type, business_gate, status);

CREATE TABLE {{table:qualifications}} (
    id VARCHAR(64) PRIMARY KEY,
    type_id VARCHAR(64) NOT NULL,
    subject_type VARCHAR(32) NOT NULL,
    subject_id VARCHAR(64) NOT NULL,
    certificate_number VARCHAR(128) NOT NULL,
    issuer VARCHAR(255) NOT NULL,
    valid_from TIMESTAMP NOT NULL,
    valid_to TIMESTAMP NOT NULL,
    status VARCHAR(32) NOT NULL,
    evidence_file_id VARCHAR(128) NULL,
    evidence_file_name VARCHAR(255) NULL,
    evidence_file_hash VARCHAR(128) NULL,
    review_comment VARCHAR(1000) NULL,
    submitted_at TIMESTAMP NULL,
    reviewed_at TIMESTAMP NULL,
    reviewed_by VARCHAR(128) NULL,
    revoked_at TIMESTAMP NULL,
    last_alert_key VARCHAR(255) NULL,
    tenant_id VARCHAR(128) NOT NULL,
    organization_id VARCHAR(128) NOT NULL,
    owner_id VARCHAR(128) NOT NULL,
    version BIGINT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    CONSTRAINT uq_pharma_oa_qualifications_tenant_type_subject_certificate UNIQUE (tenant_id, type_id, subject_id, certificate_number),
    CONSTRAINT fk_pharma_oa_qualifications_type FOREIGN KEY (type_id) REFERENCES {{table:qualification_types}} (id)
);
CREATE INDEX idx_pharma_oa_qualifications_scope ON {{table:qualifications}} (tenant_id, organization_id, subject_type, subject_id, status);
CREATE INDEX idx_pharma_oa_qualifications_expiry ON {{table:qualifications}} (tenant_id, status, valid_to);
