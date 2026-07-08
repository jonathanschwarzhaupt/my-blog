package layout

import (
	"bytes"
	"context"
	"html"
	"strings"
	"testing"
)

func TestStayConnected_ControlIsAriaDisabledNotNativelyDisabled(t *testing.T) {
	var buf bytes.Buffer

	if err := StayConnected().Render(context.Background(), &buf); err != nil {
		t.Fatal(err)
	}

	rendered := buf.String()

	if !strings.Contains(rendered, `aria-disabled="true"`) {
		t.Fatal("control isn't marked aria-disabled=\"true\"")
	}
	if strings.Contains(rendered, " disabled") {
		t.Fatal("control uses the native disabled attribute, which would suppress hover/focus")
	}
	if strings.Contains(rendered, "href=") {
		t.Fatal("control has an href, but it must not navigate anywhere real yet")
	}
}

func TestStayConnected_PopoverRevealsOnHoverAndFocus(t *testing.T) {
	var buf bytes.Buffer

	if err := StayConnected().Render(context.Background(), &buf); err != nil {
		t.Fatal(err)
	}

	rendered := buf.String()

	// popover.js (already vendored) reveals data-tui-popover-type="hover"
	// triggers on both mouseenter/mouseleave AND focusin/focusout — hover
	// wiring is the seam that gets both interactions for free.
	if !strings.Contains(rendered, `data-tui-popover-type="hover"`) {
		t.Fatal("control isn't wired as a hover-revealing popover trigger")
	}
}

func TestStayConnected_RendersComingSoonCopy(t *testing.T) {
	var buf bytes.Buffer

	if err := StayConnected().Render(context.Background(), &buf); err != nil {
		t.Fatal(err)
	}

	rendered := buf.String()

	// templ HTML-escapes text content (e.g. an apostrophe becomes &#39;),
	// so compare against the escaped form — same reasoning as the footer
	// quip test in base_test.go.
	if !strings.Contains(rendered, html.EscapeString(comingSoonMessage)) {
		t.Fatal("rendered popover doesn't contain the coming-soon copy")
	}
}
