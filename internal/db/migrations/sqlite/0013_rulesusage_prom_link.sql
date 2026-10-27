-- +goose Up
-- See the PostgreSQL migration of the same name.
ALTER TABLE RulesUsage ADD COLUMN prom_link TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE RulesUsage DROP COLUMN prom_link;
