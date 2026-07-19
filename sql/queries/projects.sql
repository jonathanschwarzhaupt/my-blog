-- name: InsertProject :one
-- order_key is never a caller-supplied value: it's computed here as one past
-- the current maximum, so a new project always lands at the end of the
-- curated order (see docs/adr/0006-project-ordering-via-editable-order-key.md).
INSERT INTO projects (name, slug, description, created_at, order_key)
VALUES (
  $1, $2, $3,
  COALESCE(sqlc.narg('created_at')::timestamptz, now()),
  COALESCE((SELECT MAX(order_key) FROM projects), 0) + 1
)
RETURNING *;

-- name: GetProjectBySlug :one
SELECT * FROM projects WHERE slug = $1;

-- name: ListProjects :many
SELECT * FROM projects ORDER BY name ASC;

-- name: ListProjectsFiltered :many
-- sort_mode is one of "curated" (order_key ASC, the default), "newest", or
-- "oldest" (both by created_at) — exactly one CASE branch is non-NULL for any
-- given sort_mode, so it alone determines the effective order; id ASC is the
-- final tie-break in all three modes.
SELECT *, count(*) OVER() AS total_count FROM projects
WHERE (sqlc.narg('from_date')::timestamptz IS NULL OR created_at >= sqlc.narg('from_date'))
  AND (sqlc.narg('to_date')::timestamptz IS NULL OR created_at <= sqlc.narg('to_date'))
ORDER BY
  CASE WHEN sqlc.arg('sort_mode')::text = 'curated' THEN order_key END ASC,
  CASE WHEN sqlc.arg('sort_mode')::text = 'oldest' THEN created_at END ASC,
  CASE WHEN sqlc.arg('sort_mode')::text = 'newest' THEN created_at END DESC,
  id ASC
LIMIT sqlc.arg('page_limit') OFFSET sqlc.arg('page_offset');

-- name: UpdateProject :one
UPDATE projects
SET description = $1, order_key = $2, created_at = $3
WHERE id = $4
RETURNING *;

-- name: DeleteProject :execrows
-- post_projects rows for this project cascade-delete (ON DELETE CASCADE on
-- post_projects.project_id, see 00003_create_projects_tables.sql) — the
-- posts themselves are untouched, only unlinked.
DELETE FROM projects WHERE id = $1;

-- name: GetProjectsByIDs :many
SELECT * FROM projects WHERE id = ANY(sqlc.arg(ids)::bigint[]);

-- name: GetProjectsForPost :many
SELECT projects.* FROM projects
JOIN post_projects ON post_projects.project_id = projects.id
WHERE post_projects.post_id = $1
ORDER BY projects.name ASC;

-- name: ListPostsByProjectSlug :many
SELECT posts.* FROM posts
JOIN post_projects ON post_projects.post_id = posts.id
JOIN projects ON post_projects.project_id = projects.id
WHERE projects.slug = $1
ORDER BY posts.published_at ASC, posts.id ASC;

-- name: DeletePostProjects :exec
DELETE FROM post_projects WHERE post_id = $1;

-- name: InsertPostProject :exec
INSERT INTO post_projects (post_id, project_id) VALUES ($1, $2);

-- name: ListFeaturedProjects :many
SELECT * FROM projects WHERE featured_rank IS NOT NULL ORDER BY featured_rank ASC;

-- name: ClearFeaturedProjects :exec
UPDATE projects SET featured_rank = NULL WHERE featured_rank IS NOT NULL;

-- name: SetFeaturedProject :exec
UPDATE projects SET featured_rank = $1 WHERE id = $2;
