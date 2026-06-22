CREATE TABLE IF NOT EXISTS sk_dictionary_types (
	id VARCHAR(64) PRIMARY KEY,
	code VARCHAR(128) NOT NULL,
	name VARCHAR(255) NOT NULL,
	description VARCHAR(512) NOT NULL DEFAULT '',
	status VARCHAR(32) NOT NULL DEFAULT 'enabled',
	sort INT NOT NULL DEFAULT 0,
	builtin BOOL NOT NULL DEFAULT FALSE,
	created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
	updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
	UNIQUE KEY uk_dictionary_types_code (code),
	KEY idx_dictionary_types_status_sort (status, sort)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS sk_dictionary_items (
	id VARCHAR(64) PRIMARY KEY,
	type_code VARCHAR(128) NOT NULL,
	label VARCHAR(255) NOT NULL,
	value VARCHAR(255) NOT NULL,
	status VARCHAR(32) NOT NULL DEFAULT 'enabled',
	sort INT NOT NULL DEFAULT 0,
	builtin BOOL NOT NULL DEFAULT FALSE,
	created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
	updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
	UNIQUE KEY uk_dictionary_items_type_value (type_code, value),
	KEY idx_dictionary_items_type_sort (type_code, sort),
	KEY idx_dictionary_items_status_sort (status, sort),
	CONSTRAINT fk_dictionary_items_type_code
		FOREIGN KEY (type_code) REFERENCES sk_dictionary_types(code)
		ON UPDATE CASCADE
		ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
