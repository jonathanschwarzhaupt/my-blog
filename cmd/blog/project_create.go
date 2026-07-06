package main

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jonathanschwarzhaupt/my-blog/internal/database"
	"github.com/jonathanschwarzhaupt/my-blog/internal/models"
	"github.com/jonathanschwarzhaupt/my-blog/internal/validator"
	"github.com/jonathanschwarzhaupt/my-blog/ui/templ/pages/admin"
)

func (app *application) projectCreate(w http.ResponseWriter, r *http.Request) {
	form := admin.ProjectForm{}
	flash := app.sessionManager.PopString(r.Context(), "flash")
	app.render(w, r, http.StatusOK, admin.ProjectCreate(form, flash))
}

func (app *application) projectCreatePost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	var form admin.ProjectForm
	if err := app.formDecoder.Decode(&form, r.PostForm); err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	form.CheckField(validator.NotBlank(form.Name), "name", "This field cannot be blank")

	slug := slugify(form.Name)
	form.CheckField(slug != "", "name", "Name must contain at least one letter or number")

	createdAt, ok := parseOptionalDate(form.CreatedAt)
	form.CheckField(ok, "created_at", "Must be a valid date")

	if !form.Valid() {
		app.render(w, r, http.StatusUnprocessableEntity, admin.ProjectCreate(form, ""))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := app.db.InsertProject(ctx, database.InsertProjectParams{
		Name:        form.Name,
		Slug:        slug,
		Description: form.Description,
		CreatedAt:   createdAt,
	})
	if err != nil {
		err = models.WrapDBError(err)
		if errors.Is(err, models.ErrDuplicateSlug) {
			form.AddNonFieldError("A project with this name (or a very similar one) already exists — try a different name.")
			app.render(w, r, http.StatusUnprocessableEntity, admin.ProjectCreate(form, ""))
			return
		}
		app.serverError(w, r, err)
		return
	}

	app.sessionManager.Put(r.Context(), "flash", "Project created")

	http.Redirect(w, r, "/projects/new", http.StatusSeeOther)
}

// validateProjectIDs confirms every id in form.ProjectIDs corresponds to a
// real Project, recording a non-field form error if not. An empty input is
// always valid — a Post may belong to zero Projects. The returned error is
// only non-nil for a genuine DB failure; an invalid-but-checkable selection
// is reported on the form itself, not via the return value.
func (app *application) validateProjectIDs(ctx context.Context, form *admin.PostForm) error {
	if len(form.ProjectIDs) == 0 {
		return nil
	}

	unique := dedupeInt64(form.ProjectIDs)

	found, err := app.db.GetProjectsByIDs(ctx, form.ProjectIDs)
	if err != nil {
		return err
	}

	if len(found) != len(unique) {
		form.AddNonFieldError("One or more selected projects don't exist.")
	}

	return nil
}

// syncPostProjects replaces postID's Project associations with projectIDs.
// Safe to call for a brand-new post with no existing associations yet.
//
// This isn't wrapped in a database transaction with the InsertPost/UpdatePost
// call that precedes it: validateProjectIDs already confirmed every id
// exists immediately before this runs, and there's currently no way to
// delete a Project through admin mode, so the only way this can still fail
// (a Project removed directly in the database between the check and this
// write, or a transient connection error) is rare and, if it does happen,
// recoverable — the Post itself is saved either way, and re-submitting the
// same edit retries the sync cleanly. A single-admin tool doesn't warrant
// the added complexity of transaction plumbing through the Querier
// interface for that residual risk.
func (app *application) syncPostProjects(ctx context.Context, postID int64, projectIDs []int64) error {
	if err := app.db.DeletePostProjects(ctx, postID); err != nil {
		return err
	}

	for _, id := range dedupeInt64(projectIDs) {
		if err := app.db.InsertPostProject(ctx, database.InsertPostProjectParams{PostID: postID, ProjectID: id}); err != nil {
			return err
		}
	}

	return nil
}

func dedupeInt64(ids []int64) []int64 {
	seen := make(map[int64]struct{}, len(ids))
	unique := make([]int64, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	return unique
}

// listProjectsOrServerError fetches every Project for populating a
// compose/edit form's checkbox list, replying with a 500 and returning ok =
// false on failure so the caller can return immediately.
func (app *application) listProjectsOrServerError(ctx context.Context, w http.ResponseWriter, r *http.Request) (projects []models.Project, ok bool) {
	dbProjects, err := app.db.ListProjects(ctx)
	if err != nil {
		app.serverError(w, r, models.WrapDBError(err))
		return nil, false
	}
	return projectsFromDatabase(dbProjects), true
}

func projectsFromDatabase(dbProjects []database.Project) []models.Project {
	projects := make([]models.Project, len(dbProjects))
	for i, p := range dbProjects {
		projects[i] = models.ProjectFromDatabase(p)
	}
	return projects
}
