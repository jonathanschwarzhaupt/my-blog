package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jonathanschwarzhaupt/home-blog/internal/assert"
	"github.com/jonathanschwarzhaupt/home-blog/internal/database"
	"github.com/jonathanschwarzhaupt/home-blog/internal/database/mocks"
)

func TestProjectsIndex_ListsAllProjects(t *testing.T) {
	mockDB := &mocks.MockQuerier{
		ListProjectsFilteredFunc: func(ctx context.Context, arg database.ListProjectsFilteredParams) ([]database.ListProjectsFilteredRow, error) {
			return []database.ListProjectsFilteredRow{
				{ID: 1, Name: "Blog Rebuild", Slug: "blog-rebuild", Description: "Rebuilding this very blog.", TotalCount: 2},
				{ID: 2, Name: "Homelab", Slug: "homelab", Description: "Running my own infra.", TotalCount: 2},
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
		ListProjectsFilteredFunc: func(ctx context.Context, arg database.ListProjectsFilteredParams) ([]database.ListProjectsFilteredRow, error) {
			return []database.ListProjectsFilteredRow{
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
		ListProjectsFilteredFunc: func(ctx context.Context, arg database.ListProjectsFilteredParams) ([]database.ListProjectsFilteredRow, error) {
			return []database.ListProjectsFilteredRow{}, nil
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

func TestProjectsIndex_PassesFiltersThroughToTheQuery(t *testing.T) {
	var gotParams database.ListProjectsFilteredParams

	mockDB := &mocks.MockQuerier{
		ListProjectsFilteredFunc: func(ctx context.Context, arg database.ListProjectsFilteredParams) ([]database.ListProjectsFilteredRow, error) {
			gotParams = arg
			return []database.ListProjectsFilteredRow{}, nil
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	rs, err := http.Get(ts.URL + "/projects?sort=oldest&from=2020-01-01&to=2020-12-31&page=3")
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	assert.Equal(t, rs.StatusCode, http.StatusOK)
	assert.Equal(t, gotParams.SortOldest, true)
	assert.Equal(t, gotParams.FromDate.Valid, true)
	assert.Equal(t, gotParams.FromDate.Time.Format("2006-01-02"), "2020-01-01")
	assert.Equal(t, gotParams.ToDate.Valid, true)
	assert.Equal(t, gotParams.ToDate.Time.Format("2006-01-02"), "2020-12-31")
	assert.Equal(t, gotParams.PageLimit, int32(7))
	assert.Equal(t, gotParams.PageOffset, int32(14)) // (page 3 - 1) * 7
}

func TestProjectsIndex_InvalidQueryParamsFallBackToDefaults(t *testing.T) {
	var gotParams database.ListProjectsFilteredParams

	mockDB := &mocks.MockQuerier{
		ListProjectsFilteredFunc: func(ctx context.Context, arg database.ListProjectsFilteredParams) ([]database.ListProjectsFilteredRow, error) {
			gotParams = arg
			return []database.ListProjectsFilteredRow{}, nil
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	rs, err := http.Get(ts.URL + "/projects?sort=sideways&from=not-a-date&page=-5")
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	assert.Equal(t, rs.StatusCode, http.StatusOK)
	assert.Equal(t, gotParams.SortOldest, false)
	assert.Equal(t, gotParams.FromDate.Valid, false)
	assert.Equal(t, gotParams.PageOffset, int32(0))
}

func TestProjectsIndex_PageBeyondLastPageReturnsEmptyNotError(t *testing.T) {
	mockDB := &mocks.MockQuerier{
		ListProjectsFilteredFunc: func(ctx context.Context, arg database.ListProjectsFilteredParams) ([]database.ListProjectsFilteredRow, error) {
			return []database.ListProjectsFilteredRow{}, nil
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	rs, err := http.Get(ts.URL + "/projects?page=999")
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	body, err := io.ReadAll(rs.Body)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, rs.StatusCode, http.StatusOK)
	assert.StringContains(t, string(body), "No projects match these filters.")
}
