package main

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jonathanschwarzhaupt/my-blog/internal/models"
)

func (app *application) postDelete(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Look up the post by the URL's slug to get its authoritative ID — same
	// reasoning as postUpdate: never trust a client-submitted id for which
	// row to delete, and this also means an already-deleted slug 404s here
	// rather than surfacing as a confusing zero-rows-affected delete below.
	dbPost, err := app.db.GetPost(ctx, slug)
	if err != nil {
		err = models.WrapDBError(err)
		if errors.Is(err, models.ErrNoRecord) {
			app.notFound(w)
			return
		}
		app.serverError(w, r, err)
		return
	}

	rowsAffected, err := app.db.DeletePost(ctx, dbPost.ID)
	if err != nil {
		app.serverError(w, r, models.WrapDBError(err))
		return
	}
	if rowsAffected == 0 {
		// Someone else deleted this row between our GetPost and this
		// DeletePost — treat the race the same as a not-found.
		app.notFound(w)
		return
	}

	// Redirect to /admin, not /posts: GET /posts is registered outside the
	// sessionManager.LoadAndSave middleware chain (it's shared, unwrapped,
	// between both binaries — see routes.go), so popping a flash there would
	// panic. /admin is already dynamic-gated (session-wrapped) — same reason
	// postCreatePost and postUpdate redirect back to a session-wrapped page
	// rather than a shared public one.
	app.sessionManager.Put(r.Context(), "flash", "Post deleted")
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}
