package layout

import (
	"bytes"
	"context"
	"html"
	"strings"
	"testing"
)

func TestBase_RendersOneOfTheKnownQuipsInFooter(t *testing.T) {
	var buf bytes.Buffer

	if err := Base("Test", "").Render(context.Background(), &buf); err != nil {
		t.Fatal(err)
	}

	rendered := buf.String()

	// templ HTML-escapes text content (e.g. "it's" -> "it&#39;s"), so a
	// quip containing an apostrophe never matches a plain, unescaped
	// substring check — compare against the escaped form instead.
	found := false
	for _, q := range footerQuips {
		if strings.Contains(rendered, html.EscapeString(q)) {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("rendered page doesn't contain any known footer quip")
	}
}
