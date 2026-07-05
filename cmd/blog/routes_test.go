package main

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/go-playground/form/v4"
	"github.com/jackc/pgx/v5"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/jonathanschwarzhaupt/my-blog/internal/assert"
	"github.com/jonathanschwarzhaupt/my-blog/internal/database"
	"github.com/jonathanschwarzhaupt/my-blog/internal/database/mocks"
	"github.com/jonathanschwarzhaupt/my-blog/ui/templ/layout"
)

func newTestApplication() *application {
	return newTestApplicationWithDB(&mocks.MockQuerier{})
}

// newTestApplicationWithDB builds an application configured for admin mode
// (layout.Features.Admin = true), matching what main.go actually constructs
// in that mode: sessionManager/formDecoder non-nil, limiter nil. Most tests
// exercise admin routes/behavior, so this is the default constructor.
func newTestApplicationWithDB(db database.Querier) *application {
	layout.Features.Admin = true

	sessionManager := scs.New()
	sessionManager.Lifetime = 12 * time.Hour

	return &application{
		logger:         slog.New(slog.NewTextHandler(io.Discard, nil)),
		db:             db,
		baseURL:        "http://example.com",
		formDecoder:    form.NewDecoder(),
		sessionManager: sessionManager,
		metrics:        newHTTPMetrics(prometheus.NewRegistry()),
	}
}

// newTestPublicApplicationWithDB builds an application configured for public
// mode (layout.Features.Admin = false), matching what main.go actually
// constructs there: limiter non-nil, sessionManager/formDecoder nil. Using
// this (rather than newTestApplicationWithDB plus manually flipping the
// flag) means a test genuinely exercises the same nil-field invariant
// production relies on, instead of a fixture that has every field populated
// regardless of mode.
func newTestPublicApplicationWithDB(db database.Querier) *application {
	layout.Features.Admin = false

	return &application{
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		db:      db,
		baseURL: "http://example.com",
		limiter: newRateLimiter(0, 0, false), // disabled — rate limiting isn't under test here
		metrics: newHTTPMetrics(prometheus.NewRegistry()),
	}
}

func TestRoutes_RequestIDFlowsThroughToLogRecord(t *testing.T) {
	rec := &recordingHandler{}

	app := newTestApplication()
	app.logger = slog.New(rec)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	rs, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	headerID := rs.Header.Get("X-Request-Id")
	assert.NotEqual(t, headerID, "")

	if len(rec.records) != 1 {
		t.Fatalf("got %d records; want 1", len(rec.records))
	}

	attrs := recordAttrs(rec.records[0])
	assert.Equal(t, attrs["request_id"], any(headerID))
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

func TestBase_HidesAdminNav(t *testing.T) {
	mockDB := &mocks.MockQuerier{
		ListPostsFunc: func(ctx context.Context) ([]database.Post, error) {
			return nil, nil
		},
	}

	app := newTestPublicApplicationWithDB(mockDB)

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

func TestRoutes_AdminRoutesNotFoundWhenDisabled(t *testing.T) {
	mockDB := &mocks.MockQuerier{
		GetPostFunc: func(ctx context.Context, slug string) (database.Post, error) {
			return database.Post{}, pgx.ErrNoRows
		},
		GetProjectBySlugFunc: func(ctx context.Context, slug string) (database.Project, error) {
			return database.Project{}, pgx.ErrNoRows
		},
	}

	app := newTestPublicApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	rs, err := http.Get(ts.URL + "/posts/new")
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()
	assert.Equal(t, rs.StatusCode, http.StatusNotFound)

	// 405, not 404: GET /posts (the posts index) is registered
	// unconditionally in both modes, so this path now has a registered
	// method — POST just isn't one of them in public mode. Still no way to
	// create a post; net/http returns 405 rather than 404 when a path
	// matches but the method doesn't.
	rs2, err := http.PostForm(ts.URL+"/posts", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer rs2.Body.Close()
	assert.Equal(t, rs2.StatusCode, http.StatusMethodNotAllowed)

	rs3, err := http.Get(ts.URL + "/projects/new")
	if err != nil {
		t.Fatal(err)
	}
	defer rs3.Body.Close()
	assert.Equal(t, rs3.StatusCode, http.StatusNotFound)
}
