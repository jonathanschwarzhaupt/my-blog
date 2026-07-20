package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/jonathanschwarzhaupt/home-blog/internal/assert"
	"github.com/jonathanschwarzhaupt/home-blog/internal/database"
	"github.com/jonathanschwarzhaupt/home-blog/internal/database/mocks"
)

func TestAboutHistory_ListsRevisionsMarksCurrent(t *testing.T) {
	newer := pgtype.Timestamptz{Time: time.Date(2026, time.July, 20, 12, 0, 0, 0, time.UTC), Valid: true}
	older := pgtype.Timestamptz{Time: time.Date(2026, time.June, 1, 9, 0, 0, 0, time.UTC), Valid: true}

	mockDB := &mocks.MockQuerier{
		ListAboutRevisionsFunc: func(ctx context.Context) ([]database.AboutRevision, error) {
			return []database.AboutRevision{
				{ID: 2, Body: "Newest body", CreatedAt: newer},
				{ID: 1, Body: "Oldest body", CreatedAt: older},
			}, nil
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	rs, err := http.Get(ts.URL + "/admin/about/history")
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
	assert.StringContains(t, html, "Newest body")
	assert.StringContains(t, html, "Oldest body")
	assert.StringContains(t, html, "(current)")
	assert.StringContains(t, html, `action="/admin/about/history/1/restore"`)
	// The current (newest) revision shouldn't offer to restore itself.
	assert.Equal(t, strings.Count(html, "/restore\""), 1)
}

func TestAboutRestore_InsertsCopyOfOldRevisionRedirects(t *testing.T) {
	var gotID int64
	var gotBody string

	mockDB := &mocks.MockQuerier{
		GetAboutRevisionFunc: func(ctx context.Context, id int64) (database.AboutRevision, error) {
			gotID = id
			return database.AboutRevision{ID: id, Body: "Old body to restore"}, nil
		},
		InsertAboutRevisionFunc: func(ctx context.Context, body string) (database.AboutRevision, error) {
			gotBody = body
			return database.AboutRevision{ID: 99, Body: body}, nil
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	client := ts.Client()
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	rs, err := client.Post(ts.URL+"/admin/about/history/1/restore", "application/x-www-form-urlencoded", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	assert.Equal(t, rs.StatusCode, http.StatusSeeOther)
	assert.Equal(t, rs.Header.Get("Location"), "/admin/about/edit")
	assert.Equal(t, gotID, int64(1))
	assert.Equal(t, gotBody, "Old body to restore")
}

func TestAboutRestore_NotFound_WhenRevisionDoesNotExist(t *testing.T) {
	mockDB := &mocks.MockQuerier{
		GetAboutRevisionFunc: func(ctx context.Context, id int64) (database.AboutRevision, error) {
			return database.AboutRevision{}, pgx.ErrNoRows
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	rs, err := http.Post(ts.URL+"/admin/about/history/999/restore", "application/x-www-form-urlencoded", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	assert.Equal(t, rs.StatusCode, http.StatusNotFound)
}

func TestAboutRestore_NotFound_WhenIDNotNumeric(t *testing.T) {
	app := newTestApplicationWithDB(&mocks.MockQuerier{})

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	rs, err := http.Post(ts.URL+"/admin/about/history/not-a-number/restore", "application/x-www-form-urlencoded", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	assert.Equal(t, rs.StatusCode, http.StatusNotFound)
}
