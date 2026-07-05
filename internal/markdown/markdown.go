// Package markdown renders post body markdown source to HTML at display
// time — the database and compose/edit forms only ever see raw markdown
// text; this is the one place it gets converted.
package markdown

import (
	"bytes"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

// md is safe to share across goroutines: goldmark's Markdown value holds no
// per-call state, only immutable parser/renderer configuration.
var md = goldmark.New(goldmark.WithExtensions(extension.GFM))

// Render converts raw markdown source to HTML.
//
// Raw HTML embedded in the source (block-level or inline) is dropped
// entirely — replaced with an "<!-- raw HTML omitted -->" comment rather
// than rendered or even escaped — since html.WithUnsafe() is deliberately
// not enabled. This is a single-trusted-author blog, so passthrough
// wouldn't really be "unsafe" in the usual multi-user sense — but there's
// no concrete need for raw HTML in post content, and leaving it off closes
// the question before it's ever asked.
func Render(raw string) (string, error) {
	var buf bytes.Buffer
	if err := md.Convert([]byte(raw), &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}
