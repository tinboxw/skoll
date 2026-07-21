CREATE TABLE IF NOT EXISTS sk_workflow_definitions (
	id VARCHAR(64) PRIMARY KEY,
	definition_key VARCHAR(128) NOT NULL,
	name VARCHAR(255) NOT NULL,
	version INT NOT NULL,
	status VARCHAR(32) NOT NULL,
	created_at DATETIME(3) NOT NULL,
	updated_at DATETIME(3) NOT NULL,
	UNIQUE KEY idx_workflow_definition_key_version (definition_key, version),
	KEY idx_workflow_definition_status_updated (status, updated_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS sk_workflow_nodes (
	definition_id VARCHAR(64) NOT NULL,
	node_id VARCHAR(64) NOT NULL,
	node_key VARCHAR(128) NOT NULL,
	name VARCHAR(255) NOT NULL,
	node_type VARCHAR(32) NOT NULL,
	position INT NOT NULL,
	PRIMARY KEY (definition_id, node_id),
	KEY idx_workflow_node_definition_position (definition_id, position),
	CONSTRAINT fk_workflow_node_definition FOREIGN KEY (definition_id) REFERENCES sk_workflow_definitions(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS sk_workflow_node_assignees (
	definition_id VARCHAR(64) NOT NULL,
	node_id VARCHAR(64) NOT NULL,
	position INT NOT NULL,
	assignee_id VARCHAR(64) NOT NULL,
	PRIMARY KEY (definition_id, node_id, position),
	KEY idx_workflow_node_assignee (assignee_id),
	CONSTRAINT fk_workflow_assignee_node FOREIGN KEY (definition_id, node_id) REFERENCES sk_workflow_nodes(definition_id, node_id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS sk_workflow_transitions (
	definition_id VARCHAR(64) NOT NULL,
	position INT NOT NULL,
	from_node_id VARCHAR(64) NOT NULL,
	to_node_id VARCHAR(64) NOT NULL,
	PRIMARY KEY (definition_id, position),
	KEY idx_workflow_transition_from (definition_id, from_node_id),
	KEY idx_workflow_transition_to (definition_id, to_node_id),
	CONSTRAINT fk_workflow_transition_definition FOREIGN KEY (definition_id) REFERENCES sk_workflow_definitions(id) ON DELETE CASCADE,
	CONSTRAINT fk_workflow_transition_from FOREIGN KEY (definition_id, from_node_id) REFERENCES sk_workflow_nodes(definition_id, node_id) ON DELETE CASCADE,
	CONSTRAINT fk_workflow_transition_to FOREIGN KEY (definition_id, to_node_id) REFERENCES sk_workflow_nodes(definition_id, node_id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS sk_workflow_instances (
	id VARCHAR(64) PRIMARY KEY,
	definition_id VARCHAR(64) NOT NULL,
	definition_key VARCHAR(128) NOT NULL,
	business_type VARCHAR(128) NOT NULL,
	business_id VARCHAR(128) NOT NULL,
	title VARCHAR(255) NOT NULL,
	status VARCHAR(32) NOT NULL,
	starter_id VARCHAR(64) NOT NULL,
	starter_name VARCHAR(255) NOT NULL,
	current_node_id VARCHAR(64) NOT NULL,
	created_at DATETIME(3) NOT NULL,
	updated_at DATETIME(3) NOT NULL,
	KEY idx_workflow_instance_definition_status (definition_id, status),
	KEY idx_workflow_instance_business (business_type, business_id),
	KEY idx_workflow_instance_starter_status (starter_id, status),
	KEY idx_workflow_instance_status_updated (status, updated_at),
	CONSTRAINT fk_workflow_instance_definition FOREIGN KEY (definition_id) REFERENCES sk_workflow_definitions(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS sk_workflow_tasks (
	id VARCHAR(128) PRIMARY KEY,
	instance_id VARCHAR(64) NOT NULL,
	position INT NOT NULL,
	node_id VARCHAR(64) NOT NULL,
	assignee_id VARCHAR(64) NOT NULL,
	assignee_name VARCHAR(255) NOT NULL,
	status VARCHAR(32) NOT NULL,
	created_at DATETIME(3) NOT NULL,
	completed_at DATETIME(3) NULL,
	KEY idx_workflow_task_instance_position (instance_id, position),
	KEY idx_workflow_task_instance_status (instance_id, status),
	KEY idx_workflow_task_assignee_status (assignee_id, status),
	CONSTRAINT fk_workflow_task_instance FOREIGN KEY (instance_id) REFERENCES sk_workflow_instances(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS sk_workflow_actions (
	id VARCHAR(128) PRIMARY KEY,
	instance_id VARCHAR(64) NOT NULL,
	position INT NOT NULL,
	action_type VARCHAR(32) NOT NULL,
	task_id VARCHAR(128) NOT NULL,
	node_id VARCHAR(64) NOT NULL,
	actor_id VARCHAR(64) NOT NULL,
	actor_name VARCHAR(255) NOT NULL,
	target_id VARCHAR(64) NOT NULL,
	target_name VARCHAR(255) NOT NULL,
	comment TEXT NOT NULL,
	created_at DATETIME(3) NOT NULL,
	KEY idx_workflow_action_instance_position (instance_id, position),
	KEY idx_workflow_action_instance_created (instance_id, created_at),
	KEY idx_workflow_action_actor_created (actor_id, created_at),
	KEY idx_workflow_action_task (task_id),
	CONSTRAINT fk_workflow_action_instance FOREIGN KEY (instance_id) REFERENCES sk_workflow_instances(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
