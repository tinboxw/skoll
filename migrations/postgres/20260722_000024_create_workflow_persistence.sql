CREATE TABLE IF NOT EXISTS sk_workflow_definitions (
	id VARCHAR(64) PRIMARY KEY,
	definition_key VARCHAR(128) NOT NULL,
	name VARCHAR(255) NOT NULL,
	version INTEGER NOT NULL,
	status VARCHAR(32) NOT NULL,
	created_at TIMESTAMPTZ NOT NULL,
	updated_at TIMESTAMPTZ NOT NULL,
	CONSTRAINT idx_workflow_definition_key_version UNIQUE (definition_key, version)
);

CREATE INDEX IF NOT EXISTS idx_workflow_definition_status_updated ON sk_workflow_definitions (status, updated_at);

CREATE TABLE IF NOT EXISTS sk_workflow_nodes (
	definition_id VARCHAR(64) NOT NULL REFERENCES sk_workflow_definitions(id) ON DELETE CASCADE,
	node_id VARCHAR(64) NOT NULL,
	node_key VARCHAR(128) NOT NULL,
	name VARCHAR(255) NOT NULL,
	node_type VARCHAR(32) NOT NULL,
	position INTEGER NOT NULL,
	PRIMARY KEY (definition_id, node_id)
);

CREATE INDEX IF NOT EXISTS idx_workflow_node_definition_position ON sk_workflow_nodes (definition_id, position);

CREATE TABLE IF NOT EXISTS sk_workflow_node_assignees (
	definition_id VARCHAR(64) NOT NULL,
	node_id VARCHAR(64) NOT NULL,
	position INTEGER NOT NULL,
	assignee_id VARCHAR(64) NOT NULL,
	PRIMARY KEY (definition_id, node_id, position),
	CONSTRAINT fk_workflow_assignee_node FOREIGN KEY (definition_id, node_id) REFERENCES sk_workflow_nodes(definition_id, node_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_workflow_node_assignee ON sk_workflow_node_assignees (assignee_id);

CREATE TABLE IF NOT EXISTS sk_workflow_transitions (
	definition_id VARCHAR(64) NOT NULL REFERENCES sk_workflow_definitions(id) ON DELETE CASCADE,
	position INTEGER NOT NULL,
	from_node_id VARCHAR(64) NOT NULL,
	to_node_id VARCHAR(64) NOT NULL,
	PRIMARY KEY (definition_id, position),
	CONSTRAINT fk_workflow_transition_from FOREIGN KEY (definition_id, from_node_id) REFERENCES sk_workflow_nodes(definition_id, node_id) ON DELETE CASCADE,
	CONSTRAINT fk_workflow_transition_to FOREIGN KEY (definition_id, to_node_id) REFERENCES sk_workflow_nodes(definition_id, node_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_workflow_transition_from ON sk_workflow_transitions (definition_id, from_node_id);
CREATE INDEX IF NOT EXISTS idx_workflow_transition_to ON sk_workflow_transitions (definition_id, to_node_id);

CREATE TABLE IF NOT EXISTS sk_workflow_instances (
	id VARCHAR(64) PRIMARY KEY,
	definition_id VARCHAR(64) NOT NULL REFERENCES sk_workflow_definitions(id),
	definition_key VARCHAR(128) NOT NULL,
	business_type VARCHAR(128) NOT NULL,
	business_id VARCHAR(128) NOT NULL,
	title VARCHAR(255) NOT NULL,
	status VARCHAR(32) NOT NULL,
	starter_id VARCHAR(64) NOT NULL,
	starter_name VARCHAR(255) NOT NULL,
	current_node_id VARCHAR(64) NOT NULL,
	created_at TIMESTAMPTZ NOT NULL,
	updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_workflow_instance_definition_status ON sk_workflow_instances (definition_id, status);
CREATE INDEX IF NOT EXISTS idx_workflow_instance_business ON sk_workflow_instances (business_type, business_id);
CREATE INDEX IF NOT EXISTS idx_workflow_instance_starter_status ON sk_workflow_instances (starter_id, status);
CREATE INDEX IF NOT EXISTS idx_workflow_instance_status_updated ON sk_workflow_instances (status, updated_at);

CREATE TABLE IF NOT EXISTS sk_workflow_tasks (
	id VARCHAR(128) PRIMARY KEY,
	instance_id VARCHAR(64) NOT NULL REFERENCES sk_workflow_instances(id) ON DELETE CASCADE,
	position INTEGER NOT NULL,
	node_id VARCHAR(64) NOT NULL,
	assignee_id VARCHAR(64) NOT NULL,
	assignee_name VARCHAR(255) NOT NULL,
	status VARCHAR(32) NOT NULL,
	created_at TIMESTAMPTZ NOT NULL,
	completed_at TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS idx_workflow_task_instance_position ON sk_workflow_tasks (instance_id, position);
CREATE INDEX IF NOT EXISTS idx_workflow_task_instance_status ON sk_workflow_tasks (instance_id, status);
CREATE INDEX IF NOT EXISTS idx_workflow_task_assignee_status ON sk_workflow_tasks (assignee_id, status);

CREATE TABLE IF NOT EXISTS sk_workflow_actions (
	id VARCHAR(128) PRIMARY KEY,
	instance_id VARCHAR(64) NOT NULL REFERENCES sk_workflow_instances(id) ON DELETE CASCADE,
	position INTEGER NOT NULL,
	action_type VARCHAR(32) NOT NULL,
	task_id VARCHAR(128) NOT NULL,
	node_id VARCHAR(64) NOT NULL,
	actor_id VARCHAR(64) NOT NULL,
	actor_name VARCHAR(255) NOT NULL,
	target_id VARCHAR(64) NOT NULL,
	target_name VARCHAR(255) NOT NULL,
	comment TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_workflow_action_instance_position ON sk_workflow_actions (instance_id, position);
CREATE INDEX IF NOT EXISTS idx_workflow_action_instance_created ON sk_workflow_actions (instance_id, created_at);
CREATE INDEX IF NOT EXISTS idx_workflow_action_actor_created ON sk_workflow_actions (actor_id, created_at);
CREATE INDEX IF NOT EXISTS idx_workflow_action_task ON sk_workflow_actions (task_id);
