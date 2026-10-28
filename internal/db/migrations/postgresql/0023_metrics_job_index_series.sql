-- +goose Up
-- How many series a job has of a metric, counted by the job index's query to
-- the upstream. Unlike metrics_cardinality's per-source top-N it covers every
-- metric a job produces; NULL until the next job index sync fills it.
ALTER TABLE metrics_job_index ADD COLUMN IF NOT EXISTS series BIGINT;

-- +goose Down
ALTER TABLE metrics_job_index DROP COLUMN IF EXISTS series;
