CREATE TABLE {{table:records}} (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    status TEXT NOT NULL,
    quantity INTEGER NOT NULL,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    owner_id TEXT NOT NULL,
    version INTEGER NOT NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);

CREATE INDEX idx_datastore_e2e_records_name ON {{table:records}} (name);
CREATE INDEX idx_datastore_e2e_records_status ON {{table:records}} (status);

CREATE TABLE {{table:ledger_entries}} (
    id TEXT PRIMARY KEY,
    record_id TEXT NOT NULL,
    tenant_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    owner_id TEXT NOT NULL,
    version INTEGER NOT NULL,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);

CREATE INDEX idx_datastore_e2e_ledger_record ON {{table:ledger_entries}} (record_id);
