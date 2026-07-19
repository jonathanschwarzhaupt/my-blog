package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/jonathanschwarzhaupt/home-blog/internal/assert"
	"github.com/jonathanschwarzhaupt/home-blog/internal/database"
	"github.com/jonathanschwarzhaupt/home-blog/internal/database/mocks"
)

func TestProjectDelete_Valid(t *testing.T) {
	var gotID int64
	var gotSlug string

	mockDB := &mocks.MockQuerier{
		GetProjectBySlugFunc: func(ctx context.Context, slug string) (database.Project, error) {
			gotSlug = slug
			return database.Project{ID: 42, Slug: slug}, nil
		},
		DeleteProjectFunc: func(ctx context.Context, id int64) (int64, error) {
			gotID = id
			return 1, nil
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	client := ts.Client()
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	rs, err := client.Post(ts.URL+"/projects/homelab/delete", "application/x-www-form-urlencoded", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	assert.Equal(t, rs.StatusCode, http.StatusSeeOther)
	assert.Equal(t, rs.Header.Get("Location"), "/admin")
	assert.Equal(t, gotSlug, "homelab")
	assert.Equal(t, gotID, int64(42))
}

func TestProjectDelete_NotFound_WhenProjectDoesNotExist(t *testing.T) {
	deleteCallCount := 0

	mockDB := &mocks.MockQuerier{
		GetProjectBySlugFunc: func(ctx context.Context, slug string) (database.Project, error) {
			return database.Project{}, pgx.ErrNoRows
		},
		DeleteProjectFunc: func(ctx context.Context, id int64) (int64, error) {
			deleteCallCount++
			return 0, nil
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	rs, err := http.Post(ts.URL+"/projects/does-not-exist/delete", "application/x-www-form-urlencoded", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	assert.Equal(t, rs.StatusCode, http.StatusNotFound)
	assert.Equal(t, deleteCallCount, 0)
}

func TestProjectDelete_NotFound_WhenAlreadyDeletedConcurrently(t *testing.T) {
	mockDB := &mocks.MockQuerier{
		GetProjectBySlugFunc: func(ctx context.Context, slug string) (database.Project, error) {
			return database.Project{ID: 42, Slug: slug}, nil
		},
		DeleteProjectFunc: func(ctx context.Context, id int64) (int64, error) {
			// Someone else already deleted this row between our
			// GetProjectBySlug and this DeleteProject — zero rows affected,
			// no error.
			return 0, nil
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	rs, err := http.Post(ts.URL+"/projects/homelab/delete", "application/x-www-form-urlencoded", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	assert.Equal(t, rs.StatusCode, http.StatusNotFound)
}

// TestProjectDelete_DoesNotDeletePosts documents the contract this feature
// exists to guarantee: deleting a project never calls anything that deletes
// posts. post_projects rows for the project cascade-delete at the database
// level (ON DELETE CASCADE on post_projects.project_id) — DeleteProject only
// ever targets the projects table.
func TestProjectDelete_DoesNotDeletePosts(t *testing.T) {
	deletePostCalled := false

	mockDB := &mocks.MockQuerier{
		GetProjectBySlugFunc: func(ctx context.Context, slug string) (database.Project, error) {
			return database.Project{ID: 42, Slug: slug}, nil
		},
		DeleteProjectFunc: func(ctx context.Context, id int64) (int64, error) {
			return 1, nil
		},
		DeletePostFunc: func(ctx context.Context, id int64) (int64, error) {
			deletePostCalled = true
			return 1, nil
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	client := ts.Client()
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	rs, err := client.Post(ts.URL+"/projects/homelab/delete", "application/x-www-form-urlencoded", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	assert.Equal(t, rs.StatusCode, http.StatusSeeOther)
	assert.False(t, deletePostCalled)
}
