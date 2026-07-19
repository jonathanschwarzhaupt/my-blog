package main

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jonathanschwarzhaupt/home-blog/internal/models"
)

func (app *application) projectDelete(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Look up the project by the URL's slug to get its authoritative ID —
	// same reasoning as postDelete: never trust a client-submitted id for
	// which row to delete, and an already-deleted slug 404s here rather than
	// surfacing as a confusing zero-rows-affected delete below.
	dbProject, err := app.db.GetProjectBySlug(ctx, slug)
	if err != nil {
		err = models.WrapDBError(err)
		if errors.Is(err, models.ErrNoRecord) {
			app.notFound(w)
			return
		}
		app.serverError(w, r, err)
		return
	}

	// post_projects rows cascade-delete (ON DELETE CASCADE on
	// post_projects.project_id) — the posts themselves have no FK to
	// projects at all, so they're untouched, only unlinked.
	rowsAffected, err := app.db.DeleteProject(ctx, dbProject.ID)
	if err != nil {
		app.serverError(w, r, models.WrapDBError(err))
		return
	}
	if rowsAffected == 0 {
		// Someone else deleted this row between our GetProjectBySlug and this
		// DeleteProject — treat the race the same as a not-found.
		app.notFound(w)
		return
	}

	app.sessionManager.Put(r.Context(), "flash", "Project deleted")
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}
