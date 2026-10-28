-- +goose NO TRANSACTION
-- +goose Up
-- Which metadata source (inventory.metadata_sources) reported each metric.
-- See the PostgreSQL migration of the same name.

CREATE TABLE IF NOT EXISTS metrics_catalog_sources (
  name TEXT NOT NULL,
  source TEXT NOT NULL,
  last_seen_at DATETIME NOT NULL,
  PRIMARY KEY (name, source)
);

CREATE INDEX IF NOT EXISTS idx_metrics_catalog_sources_last_seen_at ON metrics_catalog_sources(last_seen_at);

-- +goose Down
DROP TABLE IF EXISTS metrics_catalog_sources;
