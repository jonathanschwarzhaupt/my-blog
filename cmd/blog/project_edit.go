package main

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jonathanschwarzhaupt/home-blog/internal/database"
	"github.com/jonathanschwarzhaupt/home-blog/internal/models"
	"github.com/jonathanschwarzhaupt/home-blog/ui/templ/pages/admin"
)

func (app *application) projectEdit(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

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

	form := admin.ProjectEditForm{
		Description: dbProject.Description,
		OrderKey:    formatOrderKey(dbProject.OrderKey),
		CreatedAt:   dbProject.CreatedAt.Time.Format("2006-01-02"),
	}

	flash := app.sessionManager.PopString(r.Context(), "flash")
	app.render(w, r, http.StatusOK, admin.ProjectEdit(form, slug, dbProject.Name, flash))
}

func (app *application) projectUpdate(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")

	if err := r.ParseForm(); err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	var form admin.ProjectEditForm
	if err := app.formDecoder.Decode(&form, r.PostForm); err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	orderKey, ok := parseOrderKey(form.OrderKey)
	form.CheckField(ok, "order_key", "Must be a number")

	createdAt, ok := parseOptionalDate(form.CreatedAt)
	form.CheckField(ok, "created_at", "Must be a valid date")
	form.CheckField(form.CreatedAt != "", "created_at", "This field cannot be blank")

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Look up the project by the URL's slug to get both its authoritative ID
	// (never trust a client-submitted id) and its Name for re-rendering the
	// form on a validation failure — same reasoning as postUpdate.
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

	if !form.Valid() {
		app.render(w, r, http.StatusUnprocessableEntity, admin.ProjectEdit(form, slug, dbProject.Name, ""))
		return
	}

	_, err = app.db.UpdateProject(ctx, database.UpdateProjectParams{
		Description: form.Description,
		OrderKey:    orderKey,
		CreatedAt:   createdAt,
		ID:          dbProject.ID,
	})
	if err != nil {
		app.serverError(w, r, models.WrapDBError(err))
		return
	}

	app.sessionManager.Put(r.Context(), "flash", "Project updated")
	http.Redirect(w, r, "/projects/"+slug+"/edit", http.StatusSeeOther)
}
