package main

import (
	"net/http"

	"github.com/a-h/templ"
)

func (app *application) serverError(w http.ResponseWriter, r *http.Request, err error) {
	app.logger.Error(err.Error(), "request_id", requestIDFromContext(r.Context()), "method", r.Method, "uri", r.URL.RequestURI())
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

func (app *application) clientError(w http.ResponseWriter, status int) {
	http.Error(w, http.StatusText(status), status)
}

func (app *application) render(w http.ResponseWriter, r *http.Request, status int, component templ.Component) {
	w.WriteHeader(status)
	if err := component.Render(r.Context(), w); err != nil {
		app.logger.Error(err.Error(), "request_id", requestIDFromContext(r.Context()), "method", r.Method, "uri", r.URL.RequestURI())
	}
}
