-- +goose Up
-- Which services (inventory.service_label) of which job produce each metric,
-- for jobs such as a shared exporter that carry many services. Built by the
-- job index step from the same per-job query, so it costs no extra upstream
-- query.
CREATE TABLE IF NOT EXISTS metrics_service_index (
  name TEXT NOT NULL,
  service TEXT NOT NULL,
  job TEXT NOT NULL,
  updated_at TIMESTAMP NOT NULL DEFAULT now(),
  PRIMARY KEY (name, service, job)
);

-- Service-scoped catalog queries probe by service, then join by name.
CREATE INDEX IF NOT EXISTS idx_metrics_service_index_service_name ON metrics_service_index(service, name);

-- +goose Down
DROP TABLE IF EXISTS metrics_service_index;
