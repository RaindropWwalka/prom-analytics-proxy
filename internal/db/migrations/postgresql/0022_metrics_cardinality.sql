-- +goose Up
-- How many head series each metadata source holds for a metric, from its
-- /api/v1/status/tsdb top-N (inventory.cardinality_limit). Each catalog sync
-- replaces a source's rows wholesale: a metric that left its top-N drops out.
CREATE TABLE IF NOT EXISTS metrics_cardinality (
  name TEXT NOT NULL,
  source TEXT NOT NULL,
  series BIGINT NOT NULL,
  updated_at TIMESTAMP NOT NULL DEFAULT now(),
  PRIMARY KEY (name, source)
);

CREATE INDEX IF NOT EXISTS idx_metrics_cardinality_source ON metrics_cardinality(source);

-- +goose Down
DROP TABLE IF EXISTS metrics_cardinality;
