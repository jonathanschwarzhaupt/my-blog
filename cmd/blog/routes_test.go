package main

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jonathanschwarzhaupt/my-blog/internal/assert"
	"github.com/jonathanschwarzhaupt/my-blog/internal/database"
	"github.com/jonathanschwarzhaupt/my-blog/internal/database/mocks"
)

func newTestApplication() *application {
	return newTestApplicationWithDB(&mocks.MockQuerier{})
}

func newTestApplicationWithDB(db database.Querier) *application {
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
