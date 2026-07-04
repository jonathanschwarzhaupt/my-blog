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

	form := admin.ComposeForm{
		Version: dbPost.Version,
		Title:   dbPost.Title,
		Body:    dbPost.Body,
		SoWhat:  dbPost.SoWhat,
		Tags:    strings.Join(dbPost.Tags, ", "),
	}

	flash := app.sessionManager.PopString(r.Context(), "flash")
	app.render(w, r, http.StatusOK, admin.Edit(form, slug, flash))
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

	if !form.Valid() {
		app.render(w, r, http.StatusUnprocessableEntity, admin.Edit(form, slug, ""))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

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
			app.render(w, r, http.StatusConflict, admin.Edit(form, slug, ""))
			return
		}
		app.serverError(w, r, err)
		return
	}

	app.sessionManager.Put(r.Context(), "flash", "Post updated")

	http.Redirect(w, r, "/posts/"+slug+"/edit", http.StatusSeeOther)
}
