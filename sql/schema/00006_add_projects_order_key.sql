-- +goose Up
ALTER TABLE projects ADD COLUMN order_key double precision NOT NULL DEFAULT 0;

-- Backfill: preserve existing creation order as the initial curated order.
UPDATE projects SET order_key = id;

CREATE INDEX projects_order_key_idx ON projects (order_key ASC, id ASC);

-- +goose Down
DROP INDEX IF EXISTS projects_order_key_idx;
ALTER TABLE projects DROP COLUMN IF EXISTS order_key;
