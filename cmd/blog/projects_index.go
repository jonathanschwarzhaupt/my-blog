package main

import (
	"context"
	"net/http"
	"time"

	"github.com/jonathanschwarzhaupt/my-blog/internal/models"
	"github.com/jonathanschwarzhaupt/my-blog/ui/templ/pages/blog"
)

func (app *application) projectsIndex(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	dbProjects, err := app.db.ListProjects(ctx)
	if err != nil {
		app.serverError(w, r, models.WrapDBError(err))
		return
	}

	projects := make([]models.Project, len(dbProjects))
	for i, p := range dbProjects {
		projects[i] = models.ProjectFromDatabase(p)
	}

	app.render(w, r, http.StatusOK, blog.ProjectsIndex(projects))
}
