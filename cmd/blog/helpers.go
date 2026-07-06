package main

import (
	"net/http"
	"time"

	"github.com/a-h/templ"
	"github.com/jackc/pgx/v5/pgtype"
)

func (app *application) serverError(w http.ResponseWriter, r *http.Request, err error) {
	app.logger.Error(err.Error(), "request_id", requestIDFromContext(r.Context()), "method", r.Method, "uri", r.URL.RequestURI())
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

func (app *application) clientError(w http.ResponseWriter, status int) {
	http.Error(w, http.StatusText(status), status)
}

// notFound is Let's Go's own helper shape (helpers.go, chapter 03.04) —
// clientError(w, 404) under a semantic name, so handlers signal "not found"
// the same way they already signal any other client error, rather than
// calling net/http's http.NotFound directly. It doesn't know anything about
// the styled page; styleNotFound (notfound.go) upgrades the resulting 404
// response, decoupled from whichever code path produced it.
func (app *application) notFound(w http.ResponseWriter) {
	app.clientError(w, http.StatusNotFound)
}

func (app *application) render(w http.ResponseWriter, r *http.Request, status int, component templ.Component) {
	w.WriteHeader(status)
	if err := component.Render(r.Context(), w); err != nil {
		app.logger.Error(err.Error(), "request_id", requestIDFromContext(r.Context()), "method", r.Method, "uri", r.URL.RequestURI())
	}
}

// parseOptionalDate parses a "YYYY-MM-DD" date-input value into a nullable
// SQL param. A blank string is a valid "not provided" case (Valid: false,
// letting COALESCE(..., now()) apply at the SQL layer) rather than an error;
// a non-blank string that fails to parse is the only failure case.
func parseOptionalDate(s string) (pgtype.Timestamptz, bool) {
	if s == "" {
		return pgtype.Timestamptz{}, true
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return pgtype.Timestamptz{}, false
	}
	return pgtype.Timestamptz{Time: t, Valid: true}, true
}
