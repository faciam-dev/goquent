CREATE TABLE users (
  id BIGINT NOT NULL,
  tenant_id VARCHAR(64) NOT NULL,
  name VARCHAR(255) NOT NULL,
  email VARCHAR(255) NOT NULL,
  deleted_at TIMESTAMP NULL
);

CREATE INDEX idx_users_tenant_id ON users (tenant_id);
