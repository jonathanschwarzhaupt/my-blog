package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jonathanschwarzhaupt/home-blog/internal/assert"
)

func TestAdminDashboard_LinksToEveryAdminAction(t *testing.T) {
	app := newTestApplication()

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	rs, err := http.Get(ts.URL + "/admin")
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
	assert.StringContains(t, html, `href="/posts/new"`)
	assert.StringContains(t, html, `href="/projects/new"`)
	assert.StringContains(t, html, `href="/admin/featured"`)
	assert.StringContains(t, html, `href="/admin/stats"`)
}

func TestAdminDashboard_NotFoundInPublicMode(t *testing.T) {
	app := newTestPublicApplicationWithDB(nil)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	rs, err := http.Get(ts.URL + "/admin")
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	assert.Equal(t, rs.StatusCode, http.StatusNotFound)
}
