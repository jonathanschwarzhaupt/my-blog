package main

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jonathanschwarzhaupt/home-blog/internal/markdown"
	"github.com/jonathanschwarzhaupt/home-blog/internal/models"
	"github.com/jonathanschwarzhaupt/home-blog/ui/templ/pages/blog"
)

func (app *application) postView(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

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

	post := models.PostFromDatabase(dbPost)

	bodyHTML, err := markdown.Render(post.Body)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	app.render(w, r, http.StatusOK, blog.PostView(post, bodyHTML))
}
