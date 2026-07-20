package main

import (
	"context"
	"net/http"
	"time"

	"github.com/jonathanschwarzhaupt/home-blog/internal/database"
	"github.com/jonathanschwarzhaupt/home-blog/internal/models"
	"github.com/jonathanschwarzhaupt/home-blog/internal/validator"
	"github.com/jonathanschwarzhaupt/home-blog/ui/templ/pages/admin"
)

func (app *application) manageOrder(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	projects, ok := app.listProjectsByOrderOrServerError(ctx, w, r)
	if !ok {
		return
	}

	values := make(map[int64]string, len(projects))
	for _, p := range projects {
		values[p.ID] = formatOrderKey(p.OrderKey)
	}

	flash := app.sessionManager.PopString(r.Context(), "flash")
	app.render(w, r, http.StatusOK, admin.ManageOrder(projects, values, validator.Validator{}, flash))
}

func (app *application) manageOrderPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Fetched fresh rather than trusting the submitted set of project ids —
	// this is also what defines the authoritative row order for a
	// re-rendered form on a validation failure.
	projects, ok := app.listProjectsByOrderOrServerError(ctx, w, r)
	if !ok {
		return
	}

	var v validator.Validator
	values := make(map[int64]string, len(projects))
	parsed := make(map[int64]float64, len(projects))
	for _, p := range projects {
		field := admin.OrderKeyFieldName(p.ID)
		submitted := r.PostForm.Get(field)
		values[p.ID] = submitted

		orderKey, ok := parseOrderKey(submitted)
		v.CheckField(ok, field, "Must be a number")
		if ok {
			parsed[p.ID] = orderKey
		}
	}

	if !v.Valid() {
		app.render(w, r, http.StatusUnprocessableEntity, admin.ManageOrder(projects, values, v, ""))
		return
	}

	// Always writes every row, like manageFeaturedPost does for its slots —
	// simpler than diffing against what actually changed, and cheap for a
	// "few in number by design" collection.
	for _, p := range projects {
		if err := app.db.UpdateProjectOrderKey(ctx, database.UpdateProjectOrderKeyParams{
			OrderKey: parsed[p.ID],
			ID:       p.ID,
		}); err != nil {
			app.serverError(w, r, models.WrapDBError(err))
			return
		}
	}

	app.sessionManager.Put(r.Context(), "flash", "Project order updated")
	http.Redirect(w, r, "/admin/order", http.StatusSeeOther)
}

// listProjectsByOrderOrServerError fetches every Project ordered by its
// current order_key — mirrors listProjectsOrServerError (project.go), which
// orders alphabetically instead for populating a post's project-checkbox
// list; this page's whole point is showing/editing the curated order, so it
// needs the projects pre-sorted by it.
func (app *application) listProjectsByOrderOrServerError(ctx context.Context, w http.ResponseWriter, r *http.Request) (projects []models.Project, ok bool) {
	dbProjects, err := app.db.ListProjectsByOrder(ctx)
	if err != nil {
		app.serverError(w, r, models.WrapDBError(err))
		return nil, false
	}
	return projectsFromDatabase(dbProjects), true
}
