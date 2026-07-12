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

func (app *application) postCreate(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	allProjects, ok := app.listProjectsOrServerError(ctx, w, r)
	if !ok {
		return
	}

	form := admin.PostForm{}
	flash := app.sessionManager.PopString(r.Context(), "flash")
	app.render(w, r, http.StatusOK, admin.PostCreate(form, allProjects, flash))
}

func (app *application) postCreatePost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	var form admin.PostForm
	if err := app.formDecoder.Decode(&form, r.PostForm); err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form.Validate()

	slug := slugify(form.Title)
	form.CheckField(slug != "", "title", "Title must contain at least one letter or number")

	publishedAt, ok := parseOptionalDate(form.PublishedAt)
	form.CheckField(ok, "published_at", "Must be a valid date")

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
		app.render(w, r, http.StatusUnprocessableEntity, admin.PostCreate(form, allProjects, ""))
		return
	}

	post, err := app.db.InsertPost(ctx, database.InsertPostParams{
		Title:       form.Title,
		Slug:        slug,
		Body:        form.Body,
		SoWhat:      form.SoWhat,
		Tags:        splitTags(form.Tags),
		PublishedAt: publishedAt,
	})
	if err != nil {
		err = models.WrapDBError(err)
		if errors.Is(err, models.ErrDuplicateSlug) {
			form.AddNonFieldError("A post with this title (or a very similar one) already exists — try a different title.")
			allProjects, ok := app.listProjectsOrServerError(ctx, w, r)
			if !ok {
				return
			}
			app.render(w, r, http.StatusUnprocessableEntity, admin.PostCreate(form, allProjects, ""))
			return
		}
		app.serverError(w, r, err)
		return
	}

	if err := app.syncPostProjects(ctx, post.ID, form.ProjectIDs); err != nil {
		err = models.WrapDBError(err)
		if errors.Is(err, models.ErrInvalidProject) {
			form.AddNonFieldError("The post was saved, but one or more selected projects no longer exist — please re-select and try again.")
			allProjects, ok := app.listProjectsOrServerError(ctx, w, r)
			if !ok {
				return
			}
			app.render(w, r, http.StatusUnprocessableEntity, admin.PostCreate(form, allProjects, ""))
			return
		}
		app.serverError(w, r, err)
		return
	}

	app.sessionManager.Put(r.Context(), "flash", "Post published")

	http.Redirect(w, r, "/posts/new", http.StatusSeeOther)
}
