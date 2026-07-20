package main

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/jonathanschwarzhaupt/home-blog/internal/models"
	"github.com/jonathanschwarzhaupt/home-blog/ui/templ/pages/admin"
)

func (app *application) aboutHistory(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	dbRevisions, err := app.db.ListAboutRevisions(ctx)
	if err != nil {
		app.serverError(w, r, models.WrapDBError(err))
		return
	}

	revisions := make([]models.AboutRevision, len(dbRevisions))
	for i, r := range dbRevisions {
		revisions[i] = models.AboutRevisionFromDatabase(r)
	}

	flash := app.sessionManager.PopString(r.Context(), "flash")
	app.render(w, r, http.StatusOK, admin.AboutHistory(revisions, flash))
}

// aboutRestore never rewrites or deletes anything — it inserts a new
// revision that copies an old one's body, so restoring is itself just
// another save and the log stays append-only. See
// docs/adr/0009-about-page-db-backed-with-revision-history.md.
func (app *application) aboutRestore(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		app.notFound(w)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	dbRevision, err := app.db.GetAboutRevision(ctx, id)
	if err != nil {
		err = models.WrapDBError(err)
		if errors.Is(err, models.ErrNoRecord) {
			app.notFound(w)
			return
		}
		app.serverError(w, r, err)
		return
	}

	if _, err := app.db.InsertAboutRevision(ctx, dbRevision.Body); err != nil {
		app.serverError(w, r, models.WrapDBError(err))
		return
	}

	app.sessionManager.Put(r.Context(), "flash", "Restored an earlier revision")
	http.Redirect(w, r, "/admin/about/edit", http.StatusSeeOther)
}
