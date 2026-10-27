-- +goose NO TRANSACTION
-- +goose Up
-- See the PostgreSQL migration of the same name.
CREATE TABLE IF NOT EXISTS metrics_service_index (
  name TEXT NOT NULL,
  service TEXT NOT NULL,
  job TEXT NOT NULL,
  updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
  PRIMARY KEY (name, service, job)
);

CREATE INDEX IF NOT EXISTS idx_metrics_service_index_service_name ON metrics_service_index(service, name);

-- +goose Down
DROP TABLE IF EXISTS metrics_service_index;
