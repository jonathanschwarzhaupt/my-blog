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

	"github.com/jonathanschwarzhaupt/my-blog/internal/assert"
	"github.com/jonathanschwarzhaupt/my-blog/internal/database"
	"github.com/jonathanschwarzhaupt/my-blog/internal/database/mocks"
)

func TestPostEdit_LoadsExistingPost(t *testing.T) {
	var gotSlug string

	mockDB := &mocks.MockQuerier{
		GetPostFunc: func(ctx context.Context, slug string) (database.Post, error) {
			gotSlug = slug
			return database.Post{
				ID:          42,
				Title:       "Original Title",
				Slug:        "original-title",
				Body:        "Original body",
				SoWhat:      "Original so what",
				Tags:        []string{"go", "homelab"},
				Version:     3,
				PublishedAt: pgtype.Timestamptz{Time: time.Date(2020, time.June, 15, 0, 0, 0, 0, time.UTC), Valid: true},
			}, nil
		},
		ListProjectsFunc: func(ctx context.Context) ([]database.Project, error) {
			return nil, nil
		},
		GetProjectsForPostFunc: func(ctx context.Context, postID int64) ([]database.Project, error) {
			return nil, nil
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	rs, err := http.Get(ts.URL + "/posts/original-title/edit")
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	body, err := io.ReadAll(rs.Body)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, rs.StatusCode, http.StatusOK)
	assert.Equal(t, gotSlug, "original-title")

	html := string(body)
	assert.StringContains(t, html, "Original Title")
	assert.StringContains(t, html, "Original body")
	assert.StringContains(t, html, "Original so what")
	assert.StringContains(t, html, "go, homelab")
	assert.StringContains(t, html, `value="3"`)
	assert.StringContains(t, html, `value="2020-06-15"`)
}

func TestPostUpdate_Valid(t *testing.T) {
	var gotParams database.UpdatePostParams

	mockDB := &mocks.MockQuerier{
		GetPostFunc: func(ctx context.Context, slug string) (database.Post, error) {
			return database.Post{ID: 42, Slug: slug, Version: 3}, nil
		},
		UpdatePostFunc: func(ctx context.Context, arg database.UpdatePostParams) (database.Post, error) {
			gotParams = arg
			return database.Post{
				ID:      arg.ID,
				Title:   arg.Title,
				Slug:    "original-title",
				Body:    arg.Body,
				SoWhat:  arg.SoWhat,
				Tags:    arg.Tags,
				Version: arg.Version + 1,
			}, nil
		},
		DeletePostProjectsFunc: func(ctx context.Context, postID int64) error {
			return nil
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
	form.Set("version", "3")
	form.Set("title", "Updated Title")
	form.Set("body", "Updated body")
	form.Set("so_what", "Updated so what")
	form.Set("tags", "go, updated")
	form.Set("published_at", "2026-01-01")

	rs, err := client.PostForm(ts.URL+"/posts/original-title/edit", form)
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	assert.Equal(t, rs.StatusCode, http.StatusSeeOther)
	assert.Equal(t, rs.Header.Get("Location"), "/posts/original-title/edit")
	assert.Equal(t, gotParams.ID, int64(42))
	assert.Equal(t, gotParams.Version, int32(3))
	assert.Equal(t, gotParams.Title, "Updated Title")
	assert.Equal(t, gotParams.Body, "Updated body")
	assert.Equal(t, gotParams.SoWhat, "Updated so what")
	assert.Equal(t, len(gotParams.Tags), 2)
	assert.Equal(t, gotParams.Tags[0], "go")
	assert.Equal(t, gotParams.Tags[1], "updated")
	assert.Equal(t, gotParams.PublishedAt.Valid, true)
	assert.Equal(t, gotParams.PublishedAt.Time.Format("2006-01-02"), "2026-01-01")
}

func TestPostUpdate_StaleVersionConflict(t *testing.T) {
	updateCallCount := 0

	mockDB := &mocks.MockQuerier{
		GetPostFunc: func(ctx context.Context, slug string) (database.Post, error) {
			return database.Post{ID: 42, Slug: slug, Version: 5}, nil
		},
		UpdatePostFunc: func(ctx context.Context, arg database.UpdatePostParams) (database.Post, error) {
			updateCallCount++
			return database.Post{}, pgx.ErrNoRows
		},
		ListProjectsFunc: func(ctx context.Context) ([]database.Project, error) {
			return nil, nil
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	form := url.Values{}
	form.Set("version", "1") // stale — someone else already bumped it
	form.Set("title", "Updated Title")
	form.Set("body", "Updated body")
	form.Set("so_what", "Updated so what")
	form.Set("published_at", "2026-01-01")

	rs, err := http.PostForm(ts.URL+"/posts/original-title/edit", form)
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	body, err := io.ReadAll(rs.Body)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, rs.StatusCode, http.StatusConflict)
	assert.Equal(t, updateCallCount, 1)
	assert.StringContains(t, string(body), "edited elsewhere")
	// The re-rendered form should carry the refreshed (current) version, not
	// the stale one that was submitted, so a bare retry doesn't loop forever.
	assert.StringContains(t, string(body), `value="5"`)
}
