package main

import (
	"net/http"

	"github.com/justinas/alice"

	"github.com/jonathanschwarzhaupt/my-blog/ui"
	"github.com/jonathanschwarzhaupt/my-blog/ui/templ/layout"
)

func (app *application) routes() http.Handler {
	mux := http.NewServeMux()

	mux.Handle("GET /static/", http.FileServerFS(ui.Files))
	mux.HandleFunc("GET /health", app.healthcheck)
	mux.HandleFunc("GET /{$}", app.home)
	mux.HandleFunc("GET /posts/{slug}", app.postView)
	mux.HandleFunc("GET /projects", app.projectsIndex)
	mux.HandleFunc("GET /projects/{slug}", app.projectView)
	mux.HandleFunc("GET /feed.xml", app.feed)
	mux.HandleFunc("GET /about", app.about)

	if layout.Features.Admin {
		dynamic := alice.New(preventCSRF, app.sessionManager.LoadAndSave)

		mux.Handle("GET /posts/new", dynamic.ThenFunc(app.postCreate))
		mux.Handle("POST /posts", dynamic.ThenFunc(app.postCreatePost))
		mux.Handle("GET /posts/{slug}/edit", dynamic.ThenFunc(app.postEdit))
		mux.Handle("POST /posts/{slug}/edit", dynamic.ThenFunc(app.postUpdate))
		mux.Handle("GET /projects/new", dynamic.ThenFunc(app.projectCreate))
		mux.Handle("POST /projects", dynamic.ThenFunc(app.projectCreatePost))
	}

	middlewares := []alice.Constructor{app.recoverPanic}
	if !layout.Features.Admin {
		// Tailscale reachability is already the access control for the
		// admin deployment, so rate limiting is skipped there entirely
		// rather than just relaxed.
		middlewares = append(middlewares, app.limiter.middleware)
	}
	middlewares = append(middlewares, app.logRequest, commonHeaders)

	standard := alice.New(middlewares...)

	return standard.Then(mux)
}
