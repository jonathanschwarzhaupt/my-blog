package main

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jonathanschwarzhaupt/my-blog/internal/database"
	"github.com/jonathanschwarzhaupt/my-blog/internal/models"
	"github.com/jonathanschwarzhaupt/my-blog/ui/templ/pages/admin"
)

func (app *application) postEdit(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	dbPost, err := app.db.GetPost(ctx, slug)
	if err != nil {
		err = models.WrapDBError(err)
		if errors.Is(err, models.ErrNoRecord) {
			http.NotFound(w, r)
			return
		}
		app.serverError(w, r, err)
		return
	}

	allProjects, ok := app.listProjectsOrServerError(ctx, w, r)
	if !ok {
		return
	}

	currentProjects, err := app.db.GetProjectsForPost(ctx, dbPost.ID)
	if err != nil {
		app.serverError(w, r, models.WrapDBError(err))
		return
	}

	currentProjectIDs := make([]int64, len(currentProjects))
	for i, p := range currentProjects {
		currentProjectIDs[i] = p.ID
	}

	form := admin.ComposeForm{
		Version:    dbPost.Version,
		Title:      dbPost.Title,
		Body:       dbPost.Body,
		SoWhat:     dbPost.SoWhat,
		Tags:       strings.Join(dbPost.Tags, ", "),
		ProjectIDs: currentProjectIDs,
	}

	flash := app.sessionManager.PopString(r.Context(), "flash")
	app.render(w, r, http.StatusOK, admin.Edit(form, slug, allProjects, flash))
}

func (app *application) postUpdate(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")

	if err := r.ParseForm(); err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	var form admin.ComposeForm
	if err := app.formDecoder.Decode(&form, r.PostForm); err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form.Validate()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if form.Valid() {
		if err := app.validateProjectIDs(ctx, &form); err != nil {
			app.serverError(w, r, models.WrapDBError(err))
			return
		}
	}

	if !form.Valid() {
		allProjects, ok := app.listProjectsOrServerError(ctx, w, r)
		if !ok {
			return
		}
		app.render(w, r, http.StatusUnprocessableEntity, admin.Edit(form, slug, allProjects, ""))
		return
	}

	// Look up the post by the URL's slug to get its authoritative ID — never
	// trust a client-submitted id for which row to write to. This also means
	// a deleted post correctly 404s here instead of being reported as a
	// version conflict from UpdatePost.
	dbPost, err := app.db.GetPost(ctx, slug)
	if err != nil {
		err = models.WrapDBError(err)
		if errors.Is(err, models.ErrNoRecord) {
			http.NotFound(w, r)
			return
		}
		app.serverError(w, r, err)
		return
	}

	_, err = app.db.UpdatePost(ctx, database.UpdatePostParams{
		Title:   form.Title,
		Body:    form.Body,
		SoWhat:  form.SoWhat,
		Tags:    splitTags(form.Tags),
		ID:      dbPost.ID,
		Version: form.Version,
	})
	if err != nil {
		if errors.Is(models.WrapDBError(err), models.ErrNoRecord) {
			// We just confirmed the post exists (above), so a no-rows result
			// from UpdatePost's WHERE id = $x AND version = $y means someone
			// else changed it between our GetPost and this UpdatePost.
			form.Version = dbPost.Version // refresh so a retry submits the current version
			form.AddNonFieldError("This post was edited elsewhere since you loaded it — review the current version below and try again.")
			app.logger.Info(models.ErrEditConflict.Error(), "method", r.Method, "uri", r.URL.RequestURI())
			allProjects, ok := app.listProjectsOrServerError(ctx, w, r)
			if !ok {
				return
			}
			app.render(w, r, http.StatusConflict, admin.Edit(form, slug, allProjects, ""))
			return
		}
		app.serverError(w, r, err)
		return
	}

	if err := app.syncPostProjects(ctx, dbPost.ID, form.ProjectIDs); err != nil {
		err = models.WrapDBError(err)
		if errors.Is(err, models.ErrInvalidProject) {
			form.AddNonFieldError("The post was saved, but one or more selected projects no longer exist — please re-select and try again.")
			allProjects, ok := app.listProjectsOrServerError(ctx, w, r)
			if !ok {
				return
			}
			app.render(w, r, http.StatusUnprocessableEntity, admin.Edit(form, slug, allProjects, ""))
			return
		}
		app.serverError(w, r, err)
		return
	}

	app.sessionManager.Put(r.Context(), "flash", "Post updated")

	http.Redirect(w, r, "/posts/"+slug+"/edit", http.StatusSeeOther)
}
