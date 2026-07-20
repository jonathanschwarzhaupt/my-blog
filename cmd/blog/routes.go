package main

import (
	"net/http"

	"github.com/justinas/alice"

	"github.com/jonathanschwarzhaupt/home-blog/ui"
	"github.com/jonathanschwarzhaupt/home-blog/ui/templ/layout"
)

func (app *application) routes() http.Handler {
	mux := http.NewServeMux()

	mux.Handle("GET /static/", http.FileServerFS(ui.Files))
	mux.HandleFunc("GET /health", app.healthcheck)
	mux.HandleFunc("GET /{$}", app.home)
	mux.HandleFunc("GET /posts", app.postsIndex)
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
		mux.Handle("POST /posts/{slug}/delete", dynamic.ThenFunc(app.postDelete))
		mux.Handle("GET /projects/new", dynamic.ThenFunc(app.projectCreate))
		mux.Handle("POST /projects", dynamic.ThenFunc(app.projectCreatePost))
		mux.Handle("GET /projects/{slug}/edit", dynamic.ThenFunc(app.projectEdit))
		mux.Handle("POST /projects/{slug}/edit", dynamic.ThenFunc(app.projectUpdate))
		mux.Handle("POST /projects/{slug}/delete", dynamic.ThenFunc(app.projectDelete))
		mux.Handle("GET /admin", dynamic.ThenFunc(app.adminDashboard))
		mux.Handle("GET /admin/featured", dynamic.ThenFunc(app.manageFeatured))
		mux.Handle("POST /admin/featured", dynamic.ThenFunc(app.manageFeaturedPost))
		mux.Handle("GET /admin/order", dynamic.ThenFunc(app.manageOrder))
		mux.Handle("POST /admin/order", dynamic.ThenFunc(app.manageOrderPost))
		mux.Handle("GET /admin/stats", dynamic.ThenFunc(app.stats))
	}

	middlewares := []alice.Constructor{requestID, app.recoverPanic, app.metrics.middleware}
	if !layout.Features.Admin {
		// Tailscale reachability is already the access control for the
		// admin deployment, so rate limiting is skipped there entirely
		// rather than just relaxed.
		middlewares = append(middlewares, app.limiter.middleware)
	}
	middlewares = append(middlewares, app.logRequest, commonHeaders, app.styleNotFound)

	standard := alice.New(middlewares...)

	return standard.Then(mux)
}
