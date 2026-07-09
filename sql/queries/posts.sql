-- name: InsertPost :one
INSERT INTO posts (title, slug, body, so_what, tags, published_at)
VALUES ($1, $2, $3, $4, $5, COALESCE(sqlc.narg('published_at')::timestamptz, now()))
RETURNING *;

-- name: GetPost :one
SELECT * FROM posts WHERE slug = $1;

-- name: ListPosts :many
SELECT * FROM posts ORDER BY published_at DESC, id ASC;

-- name: ListPostsFiltered :many
SELECT *, count(*) OVER() AS total_count FROM posts
WHERE (sqlc.narg('from_date')::timestamptz IS NULL OR published_at >= sqlc.narg('from_date'))
  AND (sqlc.narg('to_date')::timestamptz IS NULL OR published_at <= sqlc.narg('to_date'))
  AND (sqlc.narg('tag')::text IS NULL OR sqlc.narg('tag')::text = ANY(tags))
ORDER BY
  CASE WHEN sqlc.arg('sort_oldest')::bool THEN published_at END ASC,
  CASE WHEN NOT sqlc.arg('sort_oldest')::bool THEN published_at END DESC,
  id ASC
LIMIT sqlc.arg('page_limit') OFFSET sqlc.arg('page_offset');

-- name: ListDistinctTags :many
SELECT DISTINCT unnest(tags)::text AS tag FROM posts ORDER BY 1;

-- name: UpdatePost :one
UPDATE posts
SET title = $1, body = $2, so_what = $3, tags = $4, published_at = $5, version = version + 1
WHERE id = $6 AND version = $7
RETURNING *;

-- name: DeletePost :execrows
DELETE FROM posts WHERE id = $1;

-- name: ListFeaturedPosts :many
SELECT * FROM posts WHERE featured_rank IS NOT NULL ORDER BY featured_rank ASC;

-- name: ClearFeaturedPosts :exec
UPDATE posts SET featured_rank = NULL WHERE featured_rank IS NOT NULL;

-- name: SetFeaturedPost :exec
UPDATE posts SET featured_rank = $1 WHERE id = $2;
