package markdown_test

import (
	"strings"
	"testing"

	"github.com/jonathanschwarzhaupt/home-blog/internal/assert"
	"github.com/jonathanschwarzhaupt/home-blog/internal/markdown"
)

func TestRender(t *testing.T) {
	tests := []struct {
		name       string
		raw        string
		wantSubstr string
	}{
		{name: "heading", raw: "# Title", wantSubstr: "<h1>Title</h1>"},
		{name: "bold", raw: "**bold**", wantSubstr: "<strong>bold</strong>"},
		{name: "italic", raw: "*italic*", wantSubstr: "<em>italic</em>"},
		{name: "link", raw: "[text](https://example.com)", wantSubstr: `<a href="https://example.com">text</a>`},
		{name: "list", raw: "- one\n- two", wantSubstr: "<li>one</li>"},
		{name: "fenced code block", raw: "```go\nfmt.Println(\"hi\")\n```", wantSubstr: "<pre><code"},
		{name: "gfm table", raw: "| A | B |\n|---|---|\n| 1 | 2 |", wantSubstr: "<table>"},
		{name: "gfm strikethrough", raw: "~~gone~~", wantSubstr: "<del>gone</del>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html, err := markdown.Render(tt.raw)
			if err != nil {
				t.Fatal(err)
			}

			assert.StringContains(t, html, tt.wantSubstr)
		})
	}
}

func TestRender_OmitsRawHTML(t *testing.T) {
	html, err := markdown.Render(`<script>alert("xss")</script>`)
	if err != nil {
		t.Fatal(err)
	}

	// goldmark's default (html.WithUnsafe() not enabled) drops raw HTML
	// blocks entirely rather than rendering or even escaping them.
	assert.False(t, strings.Contains(html, "<script>"))
	assert.False(t, strings.Contains(html, "alert("))
}

func TestRender_OmitsInlineRawHTML(t *testing.T) {
	html, err := markdown.Render("a paragraph mentioning <script> inline")
	if err != nil {
		t.Fatal(err)
	}

	// Consistent with block-level raw HTML: omitted entirely, not escaped
	// or rendered, whether it appears as its own block or inline in text.
	assert.False(t, strings.Contains(html, "<script>"))
	assert.StringContains(t, html, "a paragraph mentioning")
}
