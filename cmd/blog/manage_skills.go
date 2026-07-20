package main

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jonathanschwarzhaupt/home-blog/internal/database"
	"github.com/jonathanschwarzhaupt/home-blog/internal/models"
	"github.com/jonathanschwarzhaupt/home-blog/internal/validator"
	"github.com/jonathanschwarzhaupt/home-blog/ui/templ/pages/admin"
)

func (app *application) manageSkills(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	skills, ok := app.listSkillsOrServerError(ctx, w, r)
	if !ok {
		return
	}

	rows := make([]admin.SkillRow, 0, len(skills)+admin.BlankSkillRows)
	for _, s := range skills {
		rows = append(rows, admin.SkillRow{Category: s.Category, Name: s.Name, Order: formatOrderKey(s.OrderKey)})
	}
	for range admin.BlankSkillRows {
		rows = append(rows, admin.SkillRow{})
	}

	flash := app.sessionManager.PopString(r.Context(), "flash")
	app.render(w, r, http.StatusOK, admin.ManageSkills(rows, validator.Validator{}, flash))
}

// manageSkillsPost fully replaces the skill set on every save — the same
// clear-and-resync pattern syncPostProjects (post_edit.go) already uses,
// rather than diffing rows against what's currently stored. A row with a
// blank Name is treated as "no skill here" and skipped entirely, which is
// how a row gets removed: just clear its Name and save.
func (app *application) manageSkillsPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	rowCount, err := strconv.Atoi(r.PostForm.Get(admin.SkillRowCountFieldName))
	if err != nil || rowCount < 0 {
		app.clientError(w, http.StatusBadRequest)
		return
	}

	var v validator.Validator
	rows := make([]admin.SkillRow, rowCount)
	type parsedSkill struct {
		category string
		name     string
		orderKey float64
	}
	var toInsert []parsedSkill

	for i := range rowCount {
		category := r.PostForm.Get(admin.SkillCategoryFieldName(i))
		name := r.PostForm.Get(admin.SkillNameFieldName(i))
		orderStr := r.PostForm.Get(admin.SkillOrderFieldName(i))
		rows[i] = admin.SkillRow{Category: category, Name: name, Order: orderStr}

		if !validator.NotBlank(name) {
			continue // blank name = no skill in this row, silently skipped
		}

		if !validator.NotBlank(category) {
			v.AddFieldError(admin.SkillRowKey(i), "Category cannot be blank")
			continue
		}

		orderKey, ok := parseOrderKey(orderStr)
		if !ok {
			v.AddFieldError(admin.SkillRowKey(i), "Order must be a number")
			continue
		}

		toInsert = append(toInsert, parsedSkill{category: category, name: name, orderKey: orderKey})
	}

	if !v.Valid() {
		app.render(w, r, http.StatusUnprocessableEntity, admin.ManageSkills(rows, v, ""))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Not wrapped in a transaction — same reasoning as syncPostProjects and
	// manageOrderPost: a single-admin tool doesn't warrant transaction
	// plumbing through the Querier interface for the residual risk of a
	// mid-loop failure, which is rare and recoverable by resubmitting.
	if err := app.db.DeleteAllSkills(ctx); err != nil {
		app.serverError(w, r, models.WrapDBError(err))
		return
	}
	for _, s := range toInsert {
		if _, err := app.db.InsertSkill(ctx, database.InsertSkillParams{
			Category: strings.TrimSpace(s.category),
			Name:     strings.TrimSpace(s.name),
			OrderKey: s.orderKey,
		}); err != nil {
			app.serverError(w, r, models.WrapDBError(err))
			return
		}
	}

	app.sessionManager.Put(r.Context(), "flash", "Skills updated")
	http.Redirect(w, r, "/admin/skills", http.StatusSeeOther)
}
