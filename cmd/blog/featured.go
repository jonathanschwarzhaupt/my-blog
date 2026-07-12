package main

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/jonathanschwarzhaupt/home-blog/internal/database"
	"github.com/jonathanschwarzhaupt/home-blog/internal/models"
	"github.com/jonathanschwarzhaupt/home-blog/ui/templ/pages/admin"
)

func (app *application) manageFeatured(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	posts, ok := app.listPostsOrServerError(ctx, w, r)
	if !ok {
		return
	}

	projects, ok := app.listProjectsOrServerError(ctx, w, r)
	if !ok {
		return
	}

	form, ok := app.currentFeaturedForm(ctx, w, r)
	if !ok {
		return
	}

	flash := app.sessionManager.PopString(r.Context(), "flash")
	app.render(w, r, http.StatusOK, admin.ManageFeatured(form, posts, projects, flash))
}

func (app *application) manageFeaturedPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	var form admin.FeaturedForm
	if err := app.formDecoder.Decode(&form, r.PostForm); err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	if hasDuplicateNonZero(form.PostSlot1, form.PostSlot2, form.PostSlot3) {
		form.AddNonFieldError("The same post can't be selected in more than one slot.")
	}
	if hasDuplicateNonZero(form.ProjectSlot1, form.ProjectSlot2, form.ProjectSlot3) {
		form.AddNonFieldError("The same project can't be selected in more than one slot.")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if !form.Valid() {
		posts, ok := app.listPostsOrServerError(ctx, w, r)
		if !ok {
			return
		}
		projects, ok := app.listProjectsOrServerError(ctx, w, r)
		if !ok {
			return
		}
		app.render(w, r, http.StatusUnprocessableEntity, admin.ManageFeatured(form, posts, projects, ""))
		return
	}

	if err := app.db.ClearFeaturedPosts(ctx); err != nil {
		app.serverError(w, r, models.WrapDBError(err))
		return
	}
	postSlots := []int64{form.PostSlot1, form.PostSlot2, form.PostSlot3}
	for i, id := range postSlots {
		if id == 0 {
			continue
		}
		if err := app.db.SetFeaturedPost(ctx, database.SetFeaturedPostParams{
			FeaturedRank: pgtype.Int4{Int32: int32(i + 1), Valid: true},
			ID:           id,
		}); err != nil {
			app.serverError(w, r, models.WrapDBError(err))
			return
		}
	}

	if err := app.db.ClearFeaturedProjects(ctx); err != nil {
		app.serverError(w, r, models.WrapDBError(err))
		return
	}
	projectSlots := []int64{form.ProjectSlot1, form.ProjectSlot2, form.ProjectSlot3}
	for i, id := range projectSlots {
		if id == 0 {
			continue
		}
		if err := app.db.SetFeaturedProject(ctx, database.SetFeaturedProjectParams{
			FeaturedRank: pgtype.Int4{Int32: int32(i + 1), Valid: true},
			ID:           id,
		}); err != nil {
			app.serverError(w, r, models.WrapDBError(err))
			return
		}
	}

	app.sessionManager.Put(r.Context(), "flash", "Featured content updated")
	http.Redirect(w, r, "/admin/featured", http.StatusSeeOther)
}

// currentFeaturedForm pre-populates a FeaturedForm from whichever posts/
// projects are currently featured, so the Manage Featured page reflects
// today's selection rather than opening blank every time.
func (app *application) currentFeaturedForm(ctx context.Context, w http.ResponseWriter, r *http.Request) (admin.FeaturedForm, bool) {
	var form admin.FeaturedForm

	featuredPosts, err := app.db.ListFeaturedPosts(ctx)
	if err != nil {
		app.serverError(w, r, models.WrapDBError(err))
		return form, false
	}
	for _, p := range featuredPosts {
		switch p.FeaturedRank.Int32 {
		case 1:
			form.PostSlot1 = p.ID
		case 2:
			form.PostSlot2 = p.ID
		case 3:
			form.PostSlot3 = p.ID
		}
	}

	featuredProjects, err := app.db.ListFeaturedProjects(ctx)
	if err != nil {
		app.serverError(w, r, models.WrapDBError(err))
		return form, false
	}
	for _, p := range featuredProjects {
		switch p.FeaturedRank.Int32 {
		case 1:
			form.ProjectSlot1 = p.ID
		case 2:
			form.ProjectSlot2 = p.ID
		case 3:
			form.ProjectSlot3 = p.ID
		}
	}

	return form, true
}

// listPostsOrServerError mirrors listProjectsOrServerError (project.go) for
// posts — no such helper existed yet since home.go was the only caller of
// ListPosts and inlined the mapping directly.
func (app *application) listPostsOrServerError(ctx context.Context, w http.ResponseWriter, r *http.Request) (posts []models.Post, ok bool) {
	dbPosts, err := app.db.ListPosts(ctx)
	if err != nil {
		app.serverError(w, r, models.WrapDBError(err))
		return nil, false
	}
	return postsFromDatabase(dbPosts), true
}

func postsFromDatabase(dbPosts []database.Post) []models.Post {
	posts := make([]models.Post, len(dbPosts))
	for i, p := range dbPosts {
		posts[i] = models.PostFromDatabase(p)
	}
	return posts
}

func hasDuplicateNonZero(ids ...int64) bool {
	seen := make(map[int64]bool, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if seen[id] {
			return true
		}
		seen[id] = true
	}
	return false
}
