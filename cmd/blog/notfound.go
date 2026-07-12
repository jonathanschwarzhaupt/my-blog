package main

import (
	"net/http"

	"github.com/jonathanschwarzhaupt/home-blog/ui/templ/pages/blog"
)

// styleNotFound intercepts any 404 response — whether from app.notFound
// (helpers.go, Let's Go's own clientError(w, 404) shape) or Go's stdlib
// default when no route pattern matches at all — and replaces it with the
// site's styled NotFound page (nav/footer included) instead of the
// plain-text default. Decoupled from whichever code path produced the 404,
// so handlers keep signaling "not found" the book's way without needing to
// know a styled page exists at all.
//
// This is deliberately a response-rewriting middleware rather than a
// catch-all "/" route: registering "/" was tried first and it broke net/http's
// own 405-for-wrong-method-on-a-registered-path behavior (e.g. POST /posts in
// public mode started returning 404 instead of 405, since the mux resolves a
// registered catch-all before falling back to its wrong-method check).
// Rewriting the response after the fact doesn't touch routing at all, so
// that behavior is untouched.
func (app *application) styleNotFound(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &notFoundInterceptor{ResponseWriter: w}
		next.ServeHTTP(rec, r)
		if rec.is404 {
			app.render(w, r, http.StatusNotFound, blog.NotFound())
		}
	})
}

// notFoundInterceptor suppresses a 404 response's status/body so the caller
// can substitute its own after the wrapped handler returns.
type notFoundInterceptor struct {
	http.ResponseWriter
	is404 bool
}

func (r *notFoundInterceptor) WriteHeader(status int) {
	if status == http.StatusNotFound {
		r.is404 = true
		return
	}
	r.ResponseWriter.WriteHeader(status)
}

func (r *notFoundInterceptor) Write(b []byte) (int, error) {
	if r.is404 {
		return len(b), nil
	}
	return r.ResponseWriter.Write(b)
}
