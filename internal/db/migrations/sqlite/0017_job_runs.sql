-- +goose Up
-- See the PostgreSQL migration of the same name.
CREATE TABLE IF NOT EXISTS job_runs (
  name TEXT PRIMARY KEY,
  finished_at_unix INTEGER NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS job_runs;
