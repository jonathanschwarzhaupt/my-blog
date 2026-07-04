-- name: InsertProject :one
INSERT INTO projects (name, slug, description)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetProjectBySlug :one
SELECT * FROM projects WHERE slug = $1;

-- name: ListProjects :many
SELECT * FROM projects ORDER BY name ASC;

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
