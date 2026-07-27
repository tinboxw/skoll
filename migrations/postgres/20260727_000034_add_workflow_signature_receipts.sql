ALTER TABLE sk_workflow_nodes
	ADD COLUMN signature_meaning TEXT NOT NULL DEFAULT '',
	ADD COLUMN signature_evidence BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE sk_workflow_actions
	ADD COLUMN receipt_id VARCHAR(128) NOT NULL DEFAULT '';

CREATE INDEX idx_workflow_action_receipt ON sk_workflow_actions (receipt_id);

CREATE TABLE sk_workflow_signature_receipts (
	id VARCHAR(128) PRIMARY KEY,
	action_id VARCHAR(128) NOT NULL,
	instance_id VARCHAR(64) NOT NULL,
	definition_id VARCHAR(64) NOT NULL,
	definition_key VARCHAR(128) NOT NULL,
	business_type VARCHAR(128) NOT NULL,
	business_id VARCHAR(128) NOT NULL,
	task_id VARCHAR(128) NOT NULL,
	node_id VARCHAR(64) NOT NULL,
	action_type VARCHAR(32) NOT NULL,
	actor_id VARCHAR(64) NOT NULL,
	actor_name VARCHAR(255) NOT NULL,
	meaning TEXT NOT NULL,
	verification_id VARCHAR(128) NOT NULL,
	verification_method VARCHAR(32) NOT NULL,
	verification_at TIMESTAMPTZ NOT NULL,
	audience VARCHAR(128) NOT NULL,
	evidence_json TEXT NOT NULL,
	comment_digest VARCHAR(64) NOT NULL,
	evidence_digest VARCHAR(64) NOT NULL,
	audit_correlation_id VARCHAR(128) NOT NULL,
	signed_at TIMESTAMPTZ NOT NULL,
	CONSTRAINT uk_workflow_signature_receipt_action UNIQUE (action_id),
	CONSTRAINT uk_workflow_signature_receipt_verification UNIQUE (verification_id),
	CONSTRAINT uk_workflow_signature_receipt_audit UNIQUE (audit_correlation_id),
	CONSTRAINT fk_workflow_signature_receipt_action FOREIGN KEY (action_id) REFERENCES sk_workflow_actions(id),
	CONSTRAINT fk_workflow_signature_receipt_instance FOREIGN KEY (instance_id) REFERENCES sk_workflow_instances(id)
);

CREATE INDEX idx_workflow_receipt_instance_signed
	ON sk_workflow_signature_receipts (instance_id, signed_at);
CREATE INDEX idx_workflow_signature_receipt_actor
	ON sk_workflow_signature_receipts (actor_id);
