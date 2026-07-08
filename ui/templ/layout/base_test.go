package layout

import (
	"bytes"
	"context"
	"html"
	"strings"
	"testing"
)

func TestBase_ComposesTheFooterEasterEggQuip(t *testing.T) {
	var buf bytes.Buffer

	if err := Base("Test", "").Render(context.Background(), &buf); err != nil {
		t.Fatal(err)
	}

	rendered := buf.String()

	// The quip itself is hidden by default (opacity-0, revealed only on
	// hover/focus — see easter_egg_test.go for that behavior); this test
	// only confirms Base() actually composes EasterEggQuip() in, i.e. some
	// known quip is present in the DOM at all, regardless of visibility.
	//
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
