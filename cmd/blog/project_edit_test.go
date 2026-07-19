package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/jonathanschwarzhaupt/home-blog/internal/assert"
	"github.com/jonathanschwarzhaupt/home-blog/internal/database"
	"github.com/jonathanschwarzhaupt/home-blog/internal/database/mocks"
)

func TestProjectEdit_LoadsExistingProject(t *testing.T) {
	var gotSlug string

	mockDB := &mocks.MockQuerier{
		GetProjectBySlugFunc: func(ctx context.Context, slug string) (database.Project, error) {
			gotSlug = slug
			return database.Project{
				ID:          7,
				Name:        "Homelab",
				Slug:        slug,
				Description: "Running my own infra.",
				OrderKey:    2.5,
				CreatedAt:   pgtype.Timestamptz{Time: time.Date(2020, time.June, 15, 0, 0, 0, 0, time.UTC), Valid: true},
			}, nil
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	rs, err := http.Get(ts.URL + "/projects/homelab/edit")
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	body, err := io.ReadAll(rs.Body)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, rs.StatusCode, http.StatusOK)
	assert.Equal(t, gotSlug, "homelab")

	html := string(body)
	assert.StringContains(t, html, "Homelab")
	assert.StringContains(t, html, "Running my own infra.")
	assert.StringContains(t, html, `value="2.5"`)
	assert.StringContains(t, html, `value="2020-06-15"`)
}

func TestProjectEdit_NotFound_WhenProjectDoesNotExist(t *testing.T) {
	mockDB := &mocks.MockQuerier{
		GetProjectBySlugFunc: func(ctx context.Context, slug string) (database.Project, error) {
			return database.Project{}, pgx.ErrNoRows
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	rs, err := http.Get(ts.URL + "/projects/does-not-exist/edit")
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	assert.Equal(t, rs.StatusCode, http.StatusNotFound)
}

func TestProjectUpdate_Valid(t *testing.T) {
	var gotParams database.UpdateProjectParams

	mockDB := &mocks.MockQuerier{
		GetProjectBySlugFunc: func(ctx context.Context, slug string) (database.Project, error) {
			return database.Project{ID: 7, Name: "Homelab", Slug: slug}, nil
		},
		UpdateProjectFunc: func(ctx context.Context, arg database.UpdateProjectParams) (database.Project, error) {
			gotParams = arg
			return database.Project{ID: arg.ID, Name: "Homelab", Slug: "homelab", Description: arg.Description, OrderKey: arg.OrderKey, CreatedAt: arg.CreatedAt}, nil
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	client := ts.Client()
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	form := url.Values{}
	form.Set("description", "Updated description")
	form.Set("order_key", "1.75")
	form.Set("created_at", "2026-01-01")

	rs, err := client.PostForm(ts.URL+"/projects/homelab/edit", form)
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	assert.Equal(t, rs.StatusCode, http.StatusSeeOther)
	assert.Equal(t, rs.Header.Get("Location"), "/projects/homelab/edit")
	assert.Equal(t, gotParams.ID, int64(7))
	assert.Equal(t, gotParams.Description, "Updated description")
	assert.Equal(t, gotParams.OrderKey, 1.75)
	assert.Equal(t, gotParams.CreatedAt.Valid, true)
	assert.Equal(t, gotParams.CreatedAt.Time.Format("2006-01-02"), "2026-01-01")
}

func TestProjectUpdate_InvalidOrderKeyReRendersForm(t *testing.T) {
	updateCallCount := 0

	mockDB := &mocks.MockQuerier{
		GetProjectBySlugFunc: func(ctx context.Context, slug string) (database.Project, error) {
			return database.Project{ID: 7, Name: "Homelab", Slug: slug}, nil
		},
		UpdateProjectFunc: func(ctx context.Context, arg database.UpdateProjectParams) (database.Project, error) {
			updateCallCount++
			return database.Project{}, nil
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	form := url.Values{}
	form.Set("description", "Updated description")
	form.Set("order_key", "not-a-number")
	form.Set("created_at", "2026-01-01")

	rs, err := http.PostForm(ts.URL+"/projects/homelab/edit", form)
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	body, err := io.ReadAll(rs.Body)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, rs.StatusCode, http.StatusUnprocessableEntity)
	assert.Equal(t, updateCallCount, 0)
	assert.StringContains(t, string(body), "Must be a number")
}

func TestProjectUpdate_BlankCreatedAtReRendersForm(t *testing.T) {
	updateCallCount := 0

	mockDB := &mocks.MockQuerier{
		GetProjectBySlugFunc: func(ctx context.Context, slug string) (database.Project, error) {
			return database.Project{ID: 7, Name: "Homelab", Slug: slug}, nil
		},
		UpdateProjectFunc: func(ctx context.Context, arg database.UpdateProjectParams) (database.Project, error) {
			updateCallCount++
			return database.Project{}, nil
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	form := url.Values{}
	form.Set("description", "Updated description")
	form.Set("order_key", "1")
	form.Set("created_at", "")

	rs, err := http.PostForm(ts.URL+"/projects/homelab/edit", form)
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	assert.Equal(t, rs.StatusCode, http.StatusUnprocessableEntity)
	assert.Equal(t, updateCallCount, 0)
}

func TestProjectUpdate_NotFound_WhenProjectDoesNotExist(t *testing.T) {
	mockDB := &mocks.MockQuerier{
		GetProjectBySlugFunc: func(ctx context.Context, slug string) (database.Project, error) {
			return database.Project{}, pgx.ErrNoRows
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	form := url.Values{}
	form.Set("description", "Updated description")
	form.Set("order_key", "1")
	form.Set("created_at", "2026-01-01")

	rs, err := http.PostForm(ts.URL+"/projects/does-not-exist/edit", form)
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	assert.Equal(t, rs.StatusCode, http.StatusNotFound)
}
