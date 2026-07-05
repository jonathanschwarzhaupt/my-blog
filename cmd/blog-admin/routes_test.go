package main

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/go-playground/form/v4"

	"github.com/jonathanschwarzhaupt/my-blog/internal/assert"
	"github.com/jonathanschwarzhaupt/my-blog/internal/database"
	"github.com/jonathanschwarzhaupt/my-blog/internal/database/mocks"
	"github.com/jonathanschwarzhaupt/my-blog/ui/templ/layout"
)

func newTestApplication() *application {
	return newTestApplicationWithDB(&mocks.MockQuerier{})
}

func newTestApplicationWithDB(db database.Querier) *application {
	// Mirrors blog-admin's main(), so tests exercise the same nav that
	// actually ships in production rather than the shared package's
	// zero-value default.
	layout.Features.Admin = true

	sessionManager := scs.New()
	sessionManager.Lifetime = 12 * time.Hour

	return &application{
		logger:         slog.New(slog.NewTextHandler(io.Discard, nil)),
		db:             db,
		formDecoder:    form.NewDecoder(),
		sessionManager: sessionManager,
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

func TestBase_ShowsAdminNav(t *testing.T) {
	mockDB := &mocks.MockQuerier{
		ListProjectsFunc: func(ctx context.Context) ([]database.Project, error) {
			return nil, nil
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	rs, err := http.Get(ts.URL + "/posts/new")
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	body, err := io.ReadAll(rs.Body)
	if err != nil {
		t.Fatal(err)
	}

	html := string(body)
	assert.StringContains(t, html, `href="/posts/new"`)
	assert.StringContains(t, html, `href="/projects/new"`)
}
