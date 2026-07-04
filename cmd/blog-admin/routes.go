package main

import (
	"net/http"

	"github.com/justinas/alice"

	"github.com/jonathanschwarzhaupt/my-blog/ui"
)

func (app *application) routes() http.Handler {
	mux := http.NewServeMux()

	mux.Handle("GET /static/", http.FileServerFS(ui.Files))
	mux.HandleFunc("GET /health", app.healthcheck)

	dynamic := alice.New(preventCSRF, app.sessionManager.LoadAndSave)

	mux.Handle("GET /posts/new", dynamic.ThenFunc(app.postCreate))
	mux.Handle("POST /posts", dynamic.ThenFunc(app.postCreatePost))
	mux.Handle("GET /posts/{slug}/edit", dynamic.ThenFunc(app.postEdit))
	mux.Handle("POST /posts/{slug}/edit", dynamic.ThenFunc(app.postUpdate))
	mux.Handle("GET /projects/new", dynamic.ThenFunc(app.projectCreate))
	mux.Handle("POST /projects", dynamic.ThenFunc(app.projectCreatePost))

	standard := alice.New(app.recoverPanic, app.logRequest, commonHeaders)

	return standard.Then(mux)
}
