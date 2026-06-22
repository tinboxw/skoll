CREATE TABLE IF NOT EXISTS sk_dictionary_types (
	id VARCHAR(64) PRIMARY KEY,
	code VARCHAR(128) NOT NULL,
	name VARCHAR(255) NOT NULL,
	description VARCHAR(512) NOT NULL DEFAULT '',
	status VARCHAR(32) NOT NULL DEFAULT 'enabled',
	sort INTEGER NOT NULL DEFAULT 0,
	builtin BOOLEAN NOT NULL DEFAULT FALSE,
	created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_dictionary_types_code
	ON sk_dictionary_types (code);

CREATE INDEX IF NOT EXISTS idx_dictionary_types_status_sort
	ON sk_dictionary_types (status, sort);

CREATE TABLE IF NOT EXISTS sk_dictionary_items (
	id VARCHAR(64) PRIMARY KEY,
	type_code VARCHAR(128) NOT NULL,
	label VARCHAR(255) NOT NULL,
	value VARCHAR(255) NOT NULL,
	status VARCHAR(32) NOT NULL DEFAULT 'enabled',
	sort INTEGER NOT NULL DEFAULT 0,
	builtin BOOLEAN NOT NULL DEFAULT FALSE,
	created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
	CONSTRAINT fk_dictionary_items_type_code
		FOREIGN KEY (type_code) REFERENCES sk_dictionary_types(code)
		ON UPDATE CASCADE
		ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_dictionary_items_type_value
	ON sk_dictionary_items (type_code, value);

CREATE INDEX IF NOT EXISTS idx_dictionary_items_type_sort
	ON sk_dictionary_items (type_code, sort);

CREATE INDEX IF NOT EXISTS idx_dictionary_items_status_sort
	ON sk_dictionary_items (status, sort);
