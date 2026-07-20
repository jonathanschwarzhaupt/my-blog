-- name: InsertAboutRevision :one
-- Insert-only: the About page's content is never overwritten, only ever
-- superseded by a newer row. Restoring an old revision is just another
-- insert of that old body, not a separate operation — see
-- docs/adr/0009-about-page-db-backed-with-revision-history.md.
INSERT INTO about_revision (body) VALUES ($1) RETURNING *;

-- name: GetLatestAboutRevision :one
SELECT * FROM about_revision ORDER BY created_at DESC, id DESC LIMIT 1;

-- name: ListAboutRevisions :many
SELECT * FROM about_revision ORDER BY created_at DESC, id DESC;

-- name: GetAboutRevision :one
SELECT * FROM about_revision WHERE id = $1;
