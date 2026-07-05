-- name: InsertPost :one
INSERT INTO posts (title, slug, body, so_what, tags)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetPost :one
SELECT * FROM posts WHERE slug = $1;

-- name: ListPosts :many
SELECT * FROM posts ORDER BY published_at DESC, id ASC;

-- name: UpdatePost :one
UPDATE posts
SET title = $1, body = $2, so_what = $3, tags = $4, version = version + 1
WHERE id = $5 AND version = $6
RETURNING *;

-- name: ListFeaturedPosts :many
SELECT * FROM posts WHERE featured_rank IS NOT NULL ORDER BY featured_rank ASC;

-- name: ClearFeaturedPosts :exec
UPDATE posts SET featured_rank = NULL WHERE featured_rank IS NOT NULL;

-- name: SetFeaturedPost :exec
UPDATE posts SET featured_rank = $1 WHERE id = $2;
