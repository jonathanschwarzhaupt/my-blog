package main

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jonathanschwarzhaupt/my-blog/internal/models"
	"github.com/jonathanschwarzhaupt/my-blog/ui/templ/pages/blog"
)

func (app *application) projectView(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	dbProject, err := app.db.GetProjectBySlug(ctx, slug)
	if err != nil {
		err = models.WrapDBError(err)
		if errors.Is(err, models.ErrNoRecord) {
			http.NotFound(w, r)
			return
		}
		app.serverError(w, r, err)
		return
	}

	dbPosts, err := app.db.ListPostsByProjectSlug(ctx, slug)
	if err != nil {
		app.serverError(w, r, models.WrapDBError(err))
		return
	}

	posts := make([]models.Post, len(dbPosts))
	for i, p := range dbPosts {
		posts[i] = models.PostFromDatabase(p)
	}

	app.render(w, r, http.StatusOK, blog.ProjectView(models.ProjectFromDatabase(dbProject), posts))
}
