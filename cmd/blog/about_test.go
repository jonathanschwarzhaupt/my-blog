package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jonathanschwarzhaupt/my-blog/internal/assert"
)

func TestAbout_RendersExpectedSections(t *testing.T) {
	app := newTestApplication()

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	rs, err := http.Get(ts.URL + "/about")
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
	assert.StringContains(t, html, "<img")
	assert.StringContains(t, html, "My Journey")
	assert.StringContains(t, html, "What I Do")
	assert.StringContains(t, html, "Technical Skills")
	assert.StringContains(t, html, "Beyond Engineering")
	assert.StringContains(t, html, `href="/projects"`)
	assert.StringContains(t, html, `href="/posts"`)
	assert.StringContains(t, html, `href="/posts/gizmosql-in-kubernetes"`)
}
