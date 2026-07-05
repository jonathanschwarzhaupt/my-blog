package main

import (
	"context"
	"net/http"
	"time"

	"github.com/jonathanschwarzhaupt/my-blog/internal/models"
	"github.com/jonathanschwarzhaupt/my-blog/ui/templ/pages/blog"
)

func (app *application) home(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	dbPosts, err := app.db.ListFeaturedPosts(ctx)
	if err != nil {
		app.serverError(w, r, models.WrapDBError(err))
		return
	}

	dbProjects, err := app.db.ListFeaturedProjects(ctx)
	if err != nil {
		app.serverError(w, r, models.WrapDBError(err))
		return
	}

	app.render(w, r, http.StatusOK, blog.Home(postsFromDatabase(dbPosts), projectsFromDatabase(dbProjects)))
}
