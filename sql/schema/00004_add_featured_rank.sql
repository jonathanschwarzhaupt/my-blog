-- +goose Up
ALTER TABLE posts ADD COLUMN featured_rank integer;
ALTER TABLE posts ADD CONSTRAINT posts_featured_rank_range CHECK (featured_rank IS NULL OR featured_rank BETWEEN 1 AND 3);
CREATE UNIQUE INDEX posts_featured_rank_idx ON posts (featured_rank) WHERE featured_rank IS NOT NULL;

ALTER TABLE projects ADD COLUMN featured_rank integer;
ALTER TABLE projects ADD CONSTRAINT projects_featured_rank_range CHECK (featured_rank IS NULL OR featured_rank BETWEEN 1 AND 3);
CREATE UNIQUE INDEX projects_featured_rank_idx ON projects (featured_rank) WHERE featured_rank IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS projects_featured_rank_idx;
ALTER TABLE projects DROP CONSTRAINT IF EXISTS projects_featured_rank_range;
ALTER TABLE projects DROP COLUMN IF EXISTS featured_rank;

DROP INDEX IF EXISTS posts_featured_rank_idx;
ALTER TABLE posts DROP CONSTRAINT IF EXISTS posts_featured_rank_range;
ALTER TABLE posts DROP COLUMN IF EXISTS featured_rank;
