package layout

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestFooterLinks_RendersEachLinkWithCorrectHrefAndNewTabBehavior(t *testing.T) {
	var buf bytes.Buffer

	if err := FooterLinks().Render(context.Background(), &buf); err != nil {
		t.Fatal(err)
	}

	rendered := buf.String()

	for _, l := range footerLinks {
		t.Run(l.label, func(t *testing.T) {
			href := `href="` + l.href + `"`
			idx := strings.Index(rendered, href)
			if idx == -1 {
				t.Fatalf("rendered footer doesn't contain link %q", href)
			}

			// target/rel are only emitted (right after href, per templ
			// codegen's fixed attribute order) when newTab is true — check
			// the tag's own attribute span, not the whole rendered string,
			// so this can't accidentally match a neighboring link's tag.
			tagEnd := strings.Index(rendered[idx:], ">")
			tag := rendered[idx : idx+tagEnd]

			hasNewTab := strings.Contains(tag, `target="_blank"`) && strings.Contains(tag, `rel="noopener noreferrer"`)
			if hasNewTab != l.newTab {
				t.Fatalf("link %q: target=_blank+rel=noopener present = %v, want %v", l.href, hasNewTab, l.newTab)
			}
		})
	}
}
