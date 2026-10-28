-- +goose Up
-- When each periodic job (or step of one) last finished successfully, so a
-- replica that takes over leadership - after a restart or a deploy - waits
-- out the rest of the interval instead of starting a full cycle at once.
-- Unix seconds, free of the column's time zone handling.
CREATE TABLE IF NOT EXISTS job_runs (
  name TEXT PRIMARY KEY,
  finished_at_unix BIGINT NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS job_runs;
