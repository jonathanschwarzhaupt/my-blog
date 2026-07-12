package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/jonathanschwarzhaupt/home-blog/internal/assert"
	"github.com/jonathanschwarzhaupt/home-blog/internal/database"
	"github.com/jonathanschwarzhaupt/home-blog/internal/database/mocks"
)

func TestBackLink_PresentOnEveryAdminPage(t *testing.T) {
	mockDB := &mocks.MockQuerier{
		GetPostFunc: func(ctx context.Context, slug string) (database.Post, error) {
			return database.Post{ID: 1, Title: "A Post", Slug: slug}, nil
		},
		ListPostsFunc: func(ctx context.Context) ([]database.Post, error) {
			return nil, nil
		},
		ListProjectsFunc: func(ctx context.Context) ([]database.Project, error) {
			return nil, nil
		},
		GetProjectsForPostFunc: func(ctx context.Context, postID int64) ([]database.Project, error) {
			return nil, nil
		},
		ListFeaturedPostsFunc: func(ctx context.Context) ([]database.Post, error) {
			return nil, nil
		},
		ListFeaturedProjectsFunc: func(ctx context.Context) ([]database.Project, error) {
			return nil, nil
		},
	}

	app := newTestApplicationWithDB(mockDB)
	app.metricsRegistry = prometheus.NewRegistry()

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	tests := []struct {
		name     string
		path     string
		wantHref string
	}{
		{name: "dashboard", path: "/admin", wantHref: `href="/"`},
		{name: "compose", path: "/posts/new", wantHref: `href="/admin"`},
		{name: "edit", path: "/posts/a-post/edit", wantHref: `href="/posts/a-post"`},
		{name: "project create", path: "/projects/new", wantHref: `href="/admin"`},
		{name: "manage featured", path: "/admin/featured", wantHref: `href="/admin"`},
		{name: "stats", path: "/admin/stats", wantHref: `href="/admin"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rs, err := http.Get(ts.URL + tt.path)
			if err != nil {
				t.Fatal(err)
			}
			defer rs.Body.Close()

			body, err := io.ReadAll(rs.Body)
			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, rs.StatusCode, http.StatusOK)
			assert.StringContains(t, string(body), `aria-label="Back"`)
			assert.StringContains(t, string(body), tt.wantHref)
		})
	}
}

func TestBackLink_PresentOnPostAndProjectViewForVisitors(t *testing.T) {
	mockDB := &mocks.MockQuerier{
		GetPostFunc: func(ctx context.Context, slug string) (database.Post, error) {
			return database.Post{ID: 1, Title: "A Post", Slug: slug}, nil
		},
		GetProjectBySlugFunc: func(ctx context.Context, slug string) (database.Project, error) {
			return database.Project{ID: 1, Name: "A Project", Slug: slug}, nil
		},
		ListPostsByProjectSlugFunc: func(ctx context.Context, slug string) ([]database.Post, error) {
			return nil, nil
		},
	}

	app := newTestPublicApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	tests := []struct {
		name     string
		path     string
		wantHref string
	}{
		{name: "post view", path: "/posts/a-post", wantHref: `href="/posts"`},
		{name: "project view", path: "/projects/a-project", wantHref: `href="/projects"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rs, err := http.Get(ts.URL + tt.path)
			if err != nil {
				t.Fatal(err)
			}
			defer rs.Body.Close()

			body, err := io.ReadAll(rs.Body)
			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, rs.StatusCode, http.StatusOK)
			assert.StringContains(t, string(body), `aria-label="Back"`)
			assert.StringContains(t, string(body), tt.wantHref)
		})
	}
}
