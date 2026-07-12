package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/jonathanschwarzhaupt/home-blog/internal/assert"
	"github.com/jonathanschwarzhaupt/home-blog/internal/database"
	"github.com/jonathanschwarzhaupt/home-blog/internal/database/mocks"
)

func TestPostView_RendersPost(t *testing.T) {
	var gotSlug string

	mockDB := &mocks.MockQuerier{
		GetPostFunc: func(ctx context.Context, slug string) (database.Post, error) {
			gotSlug = slug
			return database.Post{
				ID:          1,
				Title:       "Hello World",
				Slug:        "hello-world",
				Body:        "The body content",
				SoWhat:      "It matters because reasons",
				Tags:        []string{"go", "blog"},
				Version:     1,
				PublishedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
			}, nil
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	rs, err := http.Get(ts.URL + "/posts/hello-world")
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	body, err := io.ReadAll(rs.Body)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, rs.StatusCode, http.StatusOK)
	assert.Equal(t, gotSlug, "hello-world")

	html := string(body)
	assert.StringContains(t, html, "Hello World")
	assert.StringContains(t, html, "The body content")
	assert.StringContains(t, html, "It matters because reasons")
	assert.StringContains(t, html, "go")
	assert.StringContains(t, html, "blog")
}

func TestPostView_ShowsPublishedDateNextToTags(t *testing.T) {
	mockDB := &mocks.MockQuerier{
		GetPostFunc: func(ctx context.Context, slug string) (database.Post, error) {
			return database.Post{
				ID:          1,
				Title:       "Hello World",
				Slug:        "hello-world",
				Body:        "Body",
				SoWhat:      "So what",
				Tags:        []string{"go", "blog"},
				Version:     1,
				PublishedAt: pgtype.Timestamptz{Time: time.Date(2026, time.January, 22, 0, 0, 0, 0, time.UTC), Valid: true},
			}, nil
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	rs, err := http.Get(ts.URL + "/posts/hello-world")
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	body, err := io.ReadAll(rs.Body)
	if err != nil {
		t.Fatal(err)
	}

	assert.StringContains(t, string(body), "2026-01-22")
}

func TestPostView_ShowsPublishedDateEvenWithNoTags(t *testing.T) {
	mockDB := &mocks.MockQuerier{
		GetPostFunc: func(ctx context.Context, slug string) (database.Post, error) {
			return database.Post{
				ID:          1,
				Title:       "Hello World",
				Slug:        "hello-world",
				Body:        "Body",
				SoWhat:      "So what",
				Tags:        []string{},
				Version:     1,
				PublishedAt: pgtype.Timestamptz{Time: time.Date(2026, time.January, 22, 0, 0, 0, 0, time.UTC), Valid: true},
			}, nil
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	rs, err := http.Get(ts.URL + "/posts/hello-world")
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	body, err := io.ReadAll(rs.Body)
	if err != nil {
		t.Fatal(err)
	}

	assert.StringContains(t, string(body), "2026-01-22")
}

func TestPostView_RendersMarkdownBody(t *testing.T) {
	mockDB := &mocks.MockQuerier{
		GetPostFunc: func(ctx context.Context, slug string) (database.Post, error) {
			return database.Post{
				ID:          1,
				Title:       "Hello World",
				Slug:        "hello-world",
				Body:        "**bold** and a [link](https://example.com)",
				SoWhat:      "It matters because reasons",
				Version:     1,
				PublishedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
			}, nil
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	rs, err := http.Get(ts.URL + "/posts/hello-world")
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	body, err := io.ReadAll(rs.Body)
	if err != nil {
		t.Fatal(err)
	}

	html := string(body)
	assert.StringContains(t, html, "<strong>bold</strong>")
	assert.StringContains(t, html, `<a href="https://example.com">link</a>`)
}

func TestPostView_ShowsEditMenuInAdminMode(t *testing.T) {
	mockDB := &mocks.MockQuerier{
		GetPostFunc: func(ctx context.Context, slug string) (database.Post, error) {
			return database.Post{ID: 1, Title: "Hello World", Slug: "hello-world", Body: "Body", SoWhat: "So what"}, nil
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	rs, err := http.Get(ts.URL + "/posts/hello-world")
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	body, err := io.ReadAll(rs.Body)
	if err != nil {
		t.Fatal(err)
	}

	assert.StringContains(t, string(body), `href="/posts/hello-world/edit"`)
}

func TestPostView_HidesEditMenuInPublicMode(t *testing.T) {
	mockDB := &mocks.MockQuerier{
		GetPostFunc: func(ctx context.Context, slug string) (database.Post, error) {
			return database.Post{ID: 1, Title: "Hello World", Slug: "hello-world", Body: "Body", SoWhat: "So what"}, nil
		},
	}

	app := newTestPublicApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	rs, err := http.Get(ts.URL + "/posts/hello-world")
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	body, err := io.ReadAll(rs.Body)
	if err != nil {
		t.Fatal(err)
	}

	assert.False(t, strings.Contains(string(body), "/edit"))
}
