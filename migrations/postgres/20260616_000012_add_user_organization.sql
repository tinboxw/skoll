ALTER TABLE sk_users
  ADD COLUMN department_id VARCHAR(128) DEFAULT '',
  ADD COLUMN position_id VARCHAR(128) DEFAULT '';

CREATE INDEX idx_sk_users_department_id ON sk_users (department_id);
CREATE INDEX idx_sk_users_position_id ON sk_users (position_id);
