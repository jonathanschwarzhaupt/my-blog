package main

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jonathanschwarzhaupt/my-blog/internal/database"
	"github.com/jonathanschwarzhaupt/my-blog/internal/models"
	"github.com/jonathanschwarzhaupt/my-blog/ui/templ/pages/admin"
)

func (app *application) postCreate(w http.ResponseWriter, r *http.Request) {
	form := admin.ComposeForm{}
	flash := app.sessionManager.PopString(r.Context(), "flash")
	app.render(w, r, http.StatusOK, admin.Compose(form, flash))
}

func (app *application) postCreatePost(w http.ResponseWriter, r *http.Request) {
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

	slug := slugify(form.Title)
	form.CheckField(slug != "", "title", "Title must contain at least one letter or number")

	if !form.Valid() {
		app.render(w, r, http.StatusUnprocessableEntity, admin.Compose(form, ""))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := app.db.InsertPost(ctx, database.InsertPostParams{
		Title:  form.Title,
		Slug:   slug,
		Body:   form.Body,
		SoWhat: form.SoWhat,
		Tags:   splitTags(form.Tags),
	})
	if err != nil {
		err = models.WrapDBError(err)
		if errors.Is(err, models.ErrDuplicateSlug) {
			form.AddNonFieldError("A post with this title (or a very similar one) already exists — try a different title.")
			app.render(w, r, http.StatusUnprocessableEntity, admin.Compose(form, ""))
			return
		}
		app.serverError(w, r, err)
		return
	}

	app.sessionManager.Put(r.Context(), "flash", "Post published")

	http.Redirect(w, r, "/posts/new", http.StatusSeeOther)
}
