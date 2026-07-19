package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/jonathanschwarzhaupt/home-blog/internal/assert"
	"github.com/jonathanschwarzhaupt/home-blog/internal/database"
	"github.com/jonathanschwarzhaupt/home-blog/internal/database/mocks"
)

func TestProjectView_ListsPostsOldestFirst(t *testing.T) {
	older := pgtype.Timestamptz{Time: time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC), Valid: true}
	newer := pgtype.Timestamptz{Time: time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC), Valid: true}

	mockDB := &mocks.MockQuerier{
		GetProjectBySlugFunc: func(ctx context.Context, slug string) (database.Project, error) {
			return database.Project{ID: 1, Name: "Homelab", Slug: slug, Description: "Running my own infra."}, nil
		},
		ListPostsByProjectSlugFunc: func(ctx context.Context, slug string) ([]database.Post, error) {
			return []database.Post{
				{ID: 1, Title: "Part One", Slug: "part-one", SoWhat: "the beginning", Tags: []string{"go", "homelab"}, PublishedAt: older},
				{ID: 2, Title: "Part Two", Slug: "part-two", SoWhat: "the sequel", PublishedAt: newer},
			}, nil
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	rs, err := http.Get(ts.URL + "/projects/homelab")
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	body, err := io.ReadAll(rs.Body)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, rs.StatusCode, http.StatusOK)

	html := string(body)
	assert.StringContains(t, html, "Homelab")
	assert.StringContains(t, html, "Running my own infra.")

	firstIdx := strings.Index(html, "Part One")
	secondIdx := strings.Index(html, "Part Two")
	assert.True(t, firstIdx >= 0 && secondIdx >= 0 && firstIdx < secondIdx)

	// Exercises PostCard's tag-badge rendering path through the Project
	// view specifically, not just Home (which shares the same component).
	assert.StringContains(t, html, "go")
	assert.StringContains(t, html, "homelab")
}

func TestProjectView_UnknownSlugNotFound(t *testing.T) {
	mockDB := &mocks.MockQuerier{
		GetProjectBySlugFunc: func(ctx context.Context, slug string) (database.Project, error) {
			return database.Project{}, pgx.ErrNoRows
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	rs, err := http.Get(ts.URL + "/projects/nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	assert.Equal(t, rs.StatusCode, http.StatusNotFound)
}

func TestProjectView_ShowsAdminMenuWhenEnabled(t *testing.T) {
	mockDB := &mocks.MockQuerier{
		GetProjectBySlugFunc: func(ctx context.Context, slug string) (database.Project, error) {
			return database.Project{ID: 1, Name: "Homelab", Slug: slug}, nil
		},
		ListPostsByProjectSlugFunc: func(ctx context.Context, slug string) ([]database.Post, error) {
			return []database.Post{{ID: 1, Slug: "part-one"}, {ID: 2, Slug: "part-two"}}, nil
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	rs, err := http.Get(ts.URL + "/projects/homelab")
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	body, err := io.ReadAll(rs.Body)
	if err != nil {
		t.Fatal(err)
	}

	html := string(body)
	assert.Equal(t, rs.StatusCode, http.StatusOK)
	assert.StringContains(t, html, `href="/projects/homelab/edit"`)
	assert.StringContains(t, html, `action="/projects/homelab/delete"`)
	// The delete confirm names the actual post count (2 here) — it must not
	// be a generic "are you sure" that leaves the not-deleting-posts
	// guarantee implicit.
	assert.StringContains(t, html, "(event,2)")
}

func TestProjectView_HidesAdminMenuWhenDisabled(t *testing.T) {
	mockDB := &mocks.MockQuerier{
		GetProjectBySlugFunc: func(ctx context.Context, slug string) (database.Project, error) {
			return database.Project{ID: 1, Name: "Homelab", Slug: slug}, nil
		},
		ListPostsByProjectSlugFunc: func(ctx context.Context, slug string) ([]database.Post, error) {
			return []database.Post{}, nil
		},
	}

	app := newTestPublicApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	rs, err := http.Get(ts.URL + "/projects/homelab")
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	body, err := io.ReadAll(rs.Body)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, rs.StatusCode, http.StatusOK)
	assert.False(t, strings.Contains(string(body), `href="/projects/homelab/edit"`))
}

func TestProjectView_NoPostsYet(t *testing.T) {
	mockDB := &mocks.MockQuerier{
		GetProjectBySlugFunc: func(ctx context.Context, slug string) (database.Project, error) {
			return database.Project{ID: 1, Name: "Homelab", Slug: slug}, nil
		},
		ListPostsByProjectSlugFunc: func(ctx context.Context, slug string) ([]database.Post, error) {
			return []database.Post{}, nil
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	rs, err := http.Get(ts.URL + "/projects/homelab")
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	body, err := io.ReadAll(rs.Body)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, rs.StatusCode, http.StatusOK)
	assert.StringContains(t, string(body), "Homelab")
}
