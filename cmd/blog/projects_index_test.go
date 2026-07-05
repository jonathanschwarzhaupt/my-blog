package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jonathanschwarzhaupt/my-blog/internal/assert"
	"github.com/jonathanschwarzhaupt/my-blog/internal/database"
	"github.com/jonathanschwarzhaupt/my-blog/internal/database/mocks"
)

func TestProjectsIndex_ListsAllProjects(t *testing.T) {
	mockDB := &mocks.MockQuerier{
		ListProjectsFunc: func(ctx context.Context) ([]database.Project, error) {
			return []database.Project{
				{ID: 1, Name: "Blog Rebuild", Slug: "blog-rebuild", Description: "Rebuilding this very blog."},
				{ID: 2, Name: "Homelab", Slug: "homelab", Description: "Running my own infra."},
			}, nil
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	rs, err := http.Get(ts.URL + "/projects")
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
	assert.StringContains(t, html, "Blog Rebuild")
	assert.StringContains(t, html, "Homelab")
	assert.StringContains(t, html, `href="/projects/blog-rebuild"`)
	assert.StringContains(t, html, `href="/projects/homelab"`)
}

func TestProjectsIndex_EmptyDescription(t *testing.T) {
	mockDB := &mocks.MockQuerier{
		ListProjectsFunc: func(ctx context.Context) ([]database.Project, error) {
			return []database.Project{
				{ID: 1, Name: "New Project", Slug: "new-project", Description: ""},
			}, nil
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	rs, err := http.Get(ts.URL + "/projects")
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	body, err := io.ReadAll(rs.Body)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, rs.StatusCode, http.StatusOK)
	assert.StringContains(t, string(body), "New Project")
}

func TestProjectsIndex_NoProjects(t *testing.T) {
	mockDB := &mocks.MockQuerier{
		ListProjectsFunc: func(ctx context.Context) ([]database.Project, error) {
			return []database.Project{}, nil
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	rs, err := http.Get(ts.URL + "/projects")
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	assert.Equal(t, rs.StatusCode, http.StatusOK)
}
