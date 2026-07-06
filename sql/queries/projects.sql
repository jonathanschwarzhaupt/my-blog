-- name: InsertProject :one
INSERT INTO projects (name, slug, description, created_at)
VALUES ($1, $2, $3, COALESCE(sqlc.narg('created_at')::timestamptz, now()))
RETURNING *;

-- name: GetProjectBySlug :one
SELECT * FROM projects WHERE slug = $1;

-- name: ListProjects :many
SELECT * FROM projects ORDER BY name ASC;

-- name: ListProjectsFiltered :many
SELECT *, count(*) OVER() AS total_count FROM projects
WHERE (sqlc.narg('from_date')::timestamptz IS NULL OR created_at >= sqlc.narg('from_date'))
  AND (sqlc.narg('to_date')::timestamptz IS NULL OR created_at <= sqlc.narg('to_date'))
ORDER BY
  CASE WHEN sqlc.arg('sort_oldest')::bool THEN created_at END ASC,
  CASE WHEN NOT sqlc.arg('sort_oldest')::bool THEN created_at END DESC,
  id ASC
LIMIT sqlc.arg('page_limit') OFFSET sqlc.arg('page_offset');

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
