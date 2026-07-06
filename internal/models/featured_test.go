package models_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jonathanschwarzhaupt/my-blog/internal/assert"
	"github.com/jonathanschwarzhaupt/my-blog/internal/database"
	"github.com/jonathanschwarzhaupt/my-blog/internal/models"
)

func realTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dsn := os.Getenv("BLOG_DB_DSN")
	if dsn == "" {
		t.Skip("BLOG_DB_DSN not set")
	}

	pool, err := models.OpenPool(context.Background(), dsn, models.PoolConfig{
		MaxConns: 5, MinConns: 1, MaxConnLifetime: time.Hour, MaxConnIdleTime: time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	return pool
}

func pgErrCode(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code
	}
	return ""
}

func TestFeaturedRank_ChecksRange(t *testing.T) {
	pool := realTestPool(t)
	ctx := t.Context()

	var postID int64
	err := pool.QueryRow(ctx, "SELECT id FROM posts LIMIT 1").Scan(&postID)
	if err != nil {
		t.Skip("no posts in the database to test against")
	}

	_, err = pool.Exec(ctx, "UPDATE posts SET featured_rank = 5 WHERE id = $1", postID)
	t.Cleanup(func() { pool.Exec(context.Background(), "UPDATE posts SET featured_rank = NULL WHERE id = $1", postID) })

	assert.Equal(t, pgErrCode(err), "23514") // check_violation
}

func TestFeaturedRank_EnforcesUniqueness(t *testing.T) {
	pool := realTestPool(t)
	ctx := t.Context()

	rows, err := pool.Query(ctx, "SELECT id FROM posts LIMIT 2")
	if err != nil {
		t.Fatal(err)
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	rows.Close()
	if len(ids) < 2 {
		t.Skip("need at least 2 posts in the database to test uniqueness")
	}

	t.Cleanup(func() {
		pool.Exec(context.Background(), "UPDATE posts SET featured_rank = NULL WHERE id = ANY($1)", ids)
	})

	_, err = pool.Exec(ctx, "UPDATE posts SET featured_rank = 1 WHERE id = $1", ids[0])
	if err != nil {
		t.Fatal(err)
	}

	_, err = pool.Exec(ctx, "UPDATE posts SET featured_rank = 1 WHERE id = $1", ids[1])

	assert.Equal(t, pgErrCode(err), "23505") // unique_violation
}

func TestListFeaturedPosts_OrdersByRankAndExcludesUnfeatured(t *testing.T) {
	pool := realTestPool(t)
	ctx := t.Context()
	q := database.New(pool)

	rows, err := pool.Query(ctx, "SELECT id FROM posts ORDER BY id LIMIT 2")
	if err != nil {
		t.Fatal(err)
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	rows.Close()
	if len(ids) < 2 {
		t.Skip("need at least 2 posts in the database to test ordering")
	}

	t.Cleanup(func() {
		q.ClearFeaturedPosts(context.Background())
	})

	// Deliberately set them in reverse order (ids[1] -> rank 1, ids[0] -> rank
	// 2) to prove ListFeaturedPosts orders by featured_rank, not insertion
	// order or id.
	if err := q.SetFeaturedPost(ctx, database.SetFeaturedPostParams{FeaturedRank: pgtype.Int4{Int32: 1, Valid: true}, ID: ids[1]}); err != nil {
		t.Fatal(err)
	}
	if err := q.SetFeaturedPost(ctx, database.SetFeaturedPostParams{FeaturedRank: pgtype.Int4{Int32: 2, Valid: true}, ID: ids[0]}); err != nil {
		t.Fatal(err)
	}

	featured, err := q.ListFeaturedPosts(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if len(featured) != 2 {
		t.Fatalf("got %d featured posts; want 2", len(featured))
	}
	assert.Equal(t, featured[0].ID, ids[1])
	assert.Equal(t, featured[1].ID, ids[0])
}

func TestListProjects_CreatedAtIsPopulated(t *testing.T) {
	pool := realTestPool(t)
	ctx := t.Context()
	q := database.New(pool)

	projects, err := q.ListProjects(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(projects) == 0 {
		t.Skip("no projects in the database to test against")
	}

	for _, p := range projects {
		if !p.CreatedAt.Valid {
			t.Fatalf("project %d (%s) has an invalid CreatedAt", p.ID, p.Name)
		}
		if p.CreatedAt.Time.IsZero() {
			t.Fatalf("project %d (%s) has a zero CreatedAt", p.ID, p.Name)
		}
	}
}

func TestInsertPost_ExplicitPublishedAtIsStoredAsGiven(t *testing.T) {
	pool := realTestPool(t)
	ctx := t.Context()
	q := database.New(pool)

	explicit := time.Date(2019, time.March, 4, 0, 0, 0, 0, time.UTC)

	post, err := q.InsertPost(ctx, database.InsertPostParams{
		Title:       "Backdating integration test post",
		Slug:        "backdating-integration-test-post",
		Body:        "body",
		SoWhat:      "it matters",
		Tags:        []string{},
		PublishedAt: pgtype.Timestamptz{Time: explicit, Valid: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { pool.Exec(context.Background(), "DELETE FROM posts WHERE id = $1", post.ID) })

	assert.Equal(t, post.PublishedAt.Time.Format("2006-01-02"), "2019-03-04")
}

func TestInsertPost_OmittedPublishedAtDefaultsToNow(t *testing.T) {
	pool := realTestPool(t)
	ctx := t.Context()
	q := database.New(pool)

	before := time.Now().Add(-time.Minute)

	post, err := q.InsertPost(ctx, database.InsertPostParams{
		Title:  "Backdating integration test post (default)",
		Slug:   "backdating-integration-test-post-default",
		Body:   "body",
		SoWhat: "it matters",
		Tags:   []string{},
		// PublishedAt deliberately left as the zero value (Valid: false)
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { pool.Exec(context.Background(), "DELETE FROM posts WHERE id = $1", post.ID) })

	if post.PublishedAt.Time.Before(before) {
		t.Fatalf("got PublishedAt %v; want it close to now (after %v)", post.PublishedAt.Time, before)
	}
}
