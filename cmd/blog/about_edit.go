package main

import (
	"context"
	"net/http"
	"time"

	"github.com/jonathanschwarzhaupt/home-blog/internal/models"
	"github.com/jonathanschwarzhaupt/home-blog/internal/validator"
	"github.com/jonathanschwarzhaupt/home-blog/ui/templ/pages/admin"
)

func (app *application) aboutEdit(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	revision, ok := app.latestAboutRevisionOrServerError(ctx, w, r)
	if !ok {
		return
	}

	form := admin.AboutEditForm{Body: revision.Body}

	flash := app.sessionManager.PopString(r.Context(), "flash")
	app.render(w, r, http.StatusOK, admin.AboutEdit(form, flash))
}

func (app *application) aboutUpdate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	var form admin.AboutEditForm
	if err := app.formDecoder.Decode(&form, r.PostForm); err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form.CheckField(validator.NotBlank(form.Body), "body", "This field cannot be blank")

	if !form.Valid() {
		app.render(w, r, http.StatusUnprocessableEntity, admin.AboutEdit(form, ""))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Insert-only: this never overwrites the previous revision, only ever
	// supersedes it — see docs/adr/0009-about-page-db-backed-with-revision-history.md.
	if _, err := app.db.InsertAboutRevision(ctx, form.Body); err != nil {
		app.serverError(w, r, models.WrapDBError(err))
		return
	}

	app.sessionManager.Put(r.Context(), "flash", "About page updated")
	http.Redirect(w, r, "/admin/about/edit", http.StatusSeeOther)
}
