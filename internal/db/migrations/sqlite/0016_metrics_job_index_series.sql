-- +goose Up
-- See the PostgreSQL migration of the same name.
ALTER TABLE metrics_job_index ADD COLUMN series INTEGER;

-- +goose Down
ALTER TABLE metrics_job_index DROP COLUMN series;
