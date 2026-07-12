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

func TestPostDelete_Valid(t *testing.T) {
	var gotID int64
	var gotSlug string

	mockDB := &mocks.MockQuerier{
		GetPostFunc: func(ctx context.Context, slug string) (database.Post, error) {
			gotSlug = slug
			return database.Post{ID: 42, Slug: slug}, nil
		},
		DeletePostFunc: func(ctx context.Context, id int64) (int64, error) {
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

	rs, err := client.Post(ts.URL+"/posts/original-title/delete", "application/x-www-form-urlencoded", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	assert.Equal(t, rs.StatusCode, http.StatusSeeOther)
	assert.Equal(t, rs.Header.Get("Location"), "/admin")
	assert.Equal(t, gotSlug, "original-title")
	assert.Equal(t, gotID, int64(42))
}

func TestPostDelete_NotFound_WhenPostDoesNotExist(t *testing.T) {
	deleteCallCount := 0

	mockDB := &mocks.MockQuerier{
		GetPostFunc: func(ctx context.Context, slug string) (database.Post, error) {
			return database.Post{}, pgx.ErrNoRows
		},
		DeletePostFunc: func(ctx context.Context, id int64) (int64, error) {
			deleteCallCount++
			return 0, nil
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	rs, err := http.Post(ts.URL+"/posts/does-not-exist/delete", "application/x-www-form-urlencoded", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	assert.Equal(t, rs.StatusCode, http.StatusNotFound)
	assert.Equal(t, deleteCallCount, 0)
}

func TestPostDelete_NotFound_WhenAlreadyDeletedConcurrently(t *testing.T) {
	mockDB := &mocks.MockQuerier{
		GetPostFunc: func(ctx context.Context, slug string) (database.Post, error) {
			return database.Post{ID: 42, Slug: slug}, nil
		},
		DeletePostFunc: func(ctx context.Context, id int64) (int64, error) {
			// Someone else already deleted this row between our GetPost and
			// this DeletePost — zero rows affected, no error.
			return 0, nil
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	rs, err := http.Post(ts.URL+"/posts/original-title/delete", "application/x-www-form-urlencoded", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	assert.Equal(t, rs.StatusCode, http.StatusNotFound)
}
