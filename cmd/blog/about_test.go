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
	assert.StringContains(t, html, "Skills")
	assert.StringContains(t, html, `href="/projects"`)

	// The Projects CTA (not the nav's own /projects link) must be a styled
	// button (templui's Button component), not a plain inline <a> —
	// "inline-flex" is one of Button's base classes, present regardless of
	// variant/size/theme, so this doesn't couple to exact color/accent
	// choices while still proving it's button-rendered.
	linkIdx := lastIndexOf(html, `href="/projects"`)
	assert.True(t, linkIdx >= 0)
	tagStart := lastIndexOf(html[:linkIdx], "<a")
	assert.True(t, tagStart >= 0)
	tagEnd := indexOf(html[tagStart:], ">")
	assert.True(t, tagEnd >= 0)
	assert.StringContains(t, html[tagStart:tagStart+tagEnd], "inline-flex")
}

func lastIndexOf(s, substr string) int {
	last := -1
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			last = i
		}
	}
	return last
}
