package main

import (
	"context"
	"net/http"
	"time"

	"github.com/jonathanschwarzhaupt/home-blog/internal/database"
	"github.com/jonathanschwarzhaupt/home-blog/internal/models"
	"github.com/jonathanschwarzhaupt/home-blog/ui/templ/pages/blog"
)

func (app *application) projectsIndex(w http.ResponseWriter, r *http.Request) {
	filters := blog.ParseProjectFilters(r.URL.Query())

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// An unparseable from/to is treated the same as "not provided" rather
	// than a validation error — this is a read-only browsing page, there's
	// no form field to report an error against.
	fromDate, _ := parseOptionalDate(filters.From)
	toDate, _ := parseOptionalDate(filters.To)

	rows, err := app.db.ListProjectsFiltered(ctx, database.ListProjectsFilteredParams{
		FromDate:   fromDate,
		ToDate:     toDate,
		SortOldest: filters.Sort == "oldest",
		PageLimit:  blog.ProjectsPerPage,
		PageOffset: int32((filters.Page - 1) * blog.ProjectsPerPage),
	})
	if err != nil {
		app.serverError(w, r, models.WrapDBError(err))
		return
	}

	var totalCount int64
	projects := make([]models.Project, len(rows))
	for i, row := range rows {
		totalCount = row.TotalCount
		projects[i] = models.ProjectFromDatabase(database.Project{
			ID:          row.ID,
			Name:        row.Name,
			Slug:        row.Slug,
			Description: row.Description,
			CreatedAt:   row.CreatedAt,
		})
	}

	totalPages := int((totalCount + int64(blog.ProjectsPerPage) - 1) / int64(blog.ProjectsPerPage))
	if totalPages < 1 {
		totalPages = 1
	}

	app.render(w, r, http.StatusOK, blog.ProjectsIndex(projects, filters, totalPages))
}
