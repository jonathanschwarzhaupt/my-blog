package main

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jonathanschwarzhaupt/my-blog/internal/assert"
	"github.com/jonathanschwarzhaupt/my-blog/internal/database"
	"github.com/jonathanschwarzhaupt/my-blog/internal/database/mocks"
	"github.com/jonathanschwarzhaupt/my-blog/ui/templ/layout"
)

func newTestApplication() *application {
	return newTestApplicationWithDB(&mocks.MockQuerier{})
}

func newTestApplicationWithDB(db database.Querier) *application {
	// Mirrors blog's main(), so tests exercise the same nav that actually
	// ships in production rather than the shared package's zero-value
	// default (which happens to also be false here, but this is explicit).
	layout.Features.Admin = false

	return &application{
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		db:      db,
		limiter: newRateLimiter(0, 0, false), // disabled — rate limiting isn't under test here
		baseURL: "http://example.com",
	}
}

func TestHealthcheck(t *testing.T) {
	app := newTestApplication()

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	rs, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	assert.Equal(t, rs.StatusCode, http.StatusOK)
	assert.Equal(t, rs.Header.Get("X-Content-Type-Options"), "nosniff")
	assert.Equal(t, rs.Header.Get("X-Frame-Options"), "deny")
}

func TestBase_HidesAdminNav(t *testing.T) {
	mockDB := &mocks.MockQuerier{
		ListPostsFunc: func(ctx context.Context) ([]database.Post, error) {
			return nil, nil
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	rs, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	body, err := io.ReadAll(rs.Body)
	if err != nil {
		t.Fatal(err)
	}

	html := string(body)
	assert.False(t, strings.Contains(html, `href="/posts/new"`))
	assert.False(t, strings.Contains(html, `href="/projects/new"`))
}
