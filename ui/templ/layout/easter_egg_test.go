package layout

import (
	"bytes"
	"context"
	"html"
	"strings"
	"testing"
)

func TestEasterEggQuip_TriggerIsAFocusableButtonWithAnAccessibleLabel(t *testing.T) {
	var buf bytes.Buffer

	if err := EasterEggQuip().Render(context.Background(), &buf); err != nil {
		t.Fatal(err)
	}

	rendered := buf.String()

	if !strings.Contains(rendered, "<button") {
		t.Fatal("trigger isn't a <button> — keyboard users can't Tab to a plain <div>/<span>")
	}
	if !strings.Contains(rendered, `aria-label="`) {
		t.Fatal("trigger has no aria-label — its only visible content is a bare icon")
	}
}

func TestEasterEggQuip_RevealsOnHoverAndFocusWithoutJS(t *testing.T) {
	var buf bytes.Buffer

	if err := EasterEggQuip().Render(context.Background(), &buf); err != nil {
		t.Fatal(err)
	}

	rendered := buf.String()

	// CSS-only reveal per the issue (no JS): Tailwind's group-hover/
	// group-focus-within variants toggle the quip's visibility off the
	// trigger's :hover/:focus-within state directly, no script involved.
	if !strings.Contains(rendered, "group-hover/quip:opacity-100") {
		t.Fatal("quip isn't wired to reveal on hover")
	}
	if !strings.Contains(rendered, "group-focus-within/quip:opacity-100") {
		t.Fatal("quip isn't wired to reveal on keyboard focus")
	}
	if !strings.Contains(rendered, "opacity-0") {
		t.Fatal("quip has no hidden-by-default state to reveal from")
	}
}

func TestEasterEggQuip_RendersOneOfTheKnownQuips(t *testing.T) {
	var buf bytes.Buffer

	if err := EasterEggQuip().Render(context.Background(), &buf); err != nil {
		t.Fatal(err)
	}

	rendered := buf.String()

	found := false
	for _, q := range footerQuips {
		if strings.Contains(rendered, html.EscapeString(q)) {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("rendered component doesn't contain any known footer quip")
	}
}
