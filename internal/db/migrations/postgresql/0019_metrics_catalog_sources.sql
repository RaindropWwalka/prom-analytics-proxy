-- +goose Up
-- Which metadata source (inventory.metadata_sources) reported each metric.
-- One row per (metric, source): a metric is usually reported by several.
-- last_seen_at is UTC, like metrics_catalog.last_synced_at, and is only
-- refreshed once a row has gone unconfirmed for a while (see
-- UpsertMetricsCatalogSources), so a steady state rewrites almost nothing.

CREATE TABLE IF NOT EXISTS metrics_catalog_sources (
  name TEXT NOT NULL,
  source TEXT NOT NULL,
  last_seen_at TIMESTAMP NOT NULL,
  PRIMARY KEY (name, source)
);

-- Stale-row cleanup scans by age.
CREATE INDEX IF NOT EXISTS idx_metrics_catalog_sources_last_seen_at ON metrics_catalog_sources(last_seen_at);

-- +goose Down
DROP TABLE IF EXISTS metrics_catalog_sources;
