-- +goose NO TRANSACTION
-- +goose Up
-- See the PostgreSQL migration of the same name.
CREATE TABLE IF NOT EXISTS metrics_cardinality (
  name TEXT NOT NULL,
  source TEXT NOT NULL,
  series INTEGER NOT NULL,
  updated_at DATETIME NOT NULL DEFAULT (datetime('now')),
  PRIMARY KEY (name, source)
);

CREATE INDEX IF NOT EXISTS idx_metrics_cardinality_source ON metrics_cardinality(source);

-- +goose Down
DROP TABLE IF EXISTS metrics_cardinality;
