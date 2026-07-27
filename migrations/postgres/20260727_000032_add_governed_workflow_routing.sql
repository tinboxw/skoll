ALTER TABLE sk_workflow_nodes
	ADD COLUMN decision_strategy VARCHAR(32) NOT NULL,
	ADD COLUMN decision_quorum INTEGER NOT NULL;

ALTER TABLE sk_workflow_transitions
	ADD COLUMN condition_json TEXT NOT NULL;

ALTER TABLE sk_workflow_instances
	ADD COLUMN active_node_ids_json TEXT NOT NULL,
	ADD COLUMN variables_json TEXT NOT NULL;
