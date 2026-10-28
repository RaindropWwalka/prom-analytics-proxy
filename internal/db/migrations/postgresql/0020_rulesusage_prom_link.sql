-- +goose Up
-- Link to the rule in the Prometheus that evaluates it, as metrics-usage
-- reports it: tells which Prometheus a rule lives in. Not part of a rule's
-- identity, so it stays out of the unique index and is refreshed on every push.
ALTER TABLE RulesUsage ADD COLUMN IF NOT EXISTS prom_link TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE RulesUsage DROP COLUMN IF EXISTS prom_link;
