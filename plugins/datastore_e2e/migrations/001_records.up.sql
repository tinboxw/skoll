CREATE TABLE {{table:records}} (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    owner_id TEXT NOT NULL,
    version INTEGER NOT NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);

CREATE INDEX idx_datastore_e2e_records_name ON {{table:records}} (name);
