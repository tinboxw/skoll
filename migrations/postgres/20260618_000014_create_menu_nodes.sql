CREATE TABLE IF NOT EXISTS sk_menu_nodes (
	id BIGSERIAL PRIMARY KEY,
	menu_key VARCHAR(128) NOT NULL,
	parent_key VARCHAR(128) NOT NULL DEFAULT '',
	source VARCHAR(64) NOT NULL,
	name VARCHAR(128) NOT NULL,
	path VARCHAR(256) NOT NULL,
	component VARCHAR(256) NOT NULL DEFAULT '',
	icon VARCHAR(128) NOT NULL DEFAULT '',
	sort INTEGER NOT NULL DEFAULT 0,
	visible BOOLEAN NOT NULL DEFAULT TRUE,
	required_roles_json TEXT,
	required_permissions_json TEXT,
	created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_menu_nodes_key
	ON sk_menu_nodes (menu_key);

CREATE INDEX IF NOT EXISTS idx_menu_nodes_parent_sort
	ON sk_menu_nodes (parent_key, sort, id);

CREATE INDEX IF NOT EXISTS idx_menu_nodes_source
	ON sk_menu_nodes (source);

CREATE INDEX IF NOT EXISTS idx_menu_nodes_path
	ON sk_menu_nodes (path);

CREATE INDEX IF NOT EXISTS idx_menu_nodes_sort
	ON sk_menu_nodes (sort);

CREATE INDEX IF NOT EXISTS idx_menu_nodes_visible
	ON sk_menu_nodes (visible);
