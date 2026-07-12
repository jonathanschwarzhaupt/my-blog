package main

import (
	"net/http"

	"github.com/jonathanschwarzhaupt/home-blog/internal/metrics"
	"github.com/jonathanschwarzhaupt/home-blog/ui/templ/pages/admin"
)

func (app *application) stats(w http.ResponseWriter, r *http.Request) {
	stats, err := metrics.Gather(app.metricsRegistry, app.startedAt)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	app.render(w, r, http.StatusOK, admin.Stats(stats))
}
