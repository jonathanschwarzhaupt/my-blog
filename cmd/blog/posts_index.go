package main

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/jonathanschwarzhaupt/home-blog/internal/database"
	"github.com/jonathanschwarzhaupt/home-blog/internal/models"
	"github.com/jonathanschwarzhaupt/home-blog/ui/templ/pages/blog"
)

func (app *application) postsIndex(w http.ResponseWriter, r *http.Request) {
	filters := blog.ParsePostFilters(r.URL.Query())

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// An unparseable from/to is treated the same as "not provided" rather
	// than a validation error — this is a read-only browsing page, there's
	// no form field to report an error against.
	fromDate, _ := parseOptionalDate(filters.From)
	toDate, _ := parseOptionalDate(filters.To)

	var tag pgtype.Text
	if filters.Tag != "" {
		tag = pgtype.Text{String: filters.Tag, Valid: true}
	}

	rows, err := app.db.ListPostsFiltered(ctx, database.ListPostsFilteredParams{
		FromDate:   fromDate,
		ToDate:     toDate,
		Tag:        tag,
		SortOldest: filters.Sort == "oldest",
		PageLimit:  blog.PostsPerPage,
		PageOffset: int32((filters.Page - 1) * blog.PostsPerPage),
	})
	if err != nil {
		app.serverError(w, r, models.WrapDBError(err))
		return
	}

	allTags, err := app.db.ListDistinctTags(ctx)
	if err != nil {
		app.serverError(w, r, models.WrapDBError(err))
		return
	}

	var totalCount int64
	posts := make([]models.Post, len(rows))
	for i, row := range rows {
		totalCount = row.TotalCount
		posts[i] = models.PostFromDatabase(database.Post{
			ID:          row.ID,
			Title:       row.Title,
			Slug:        row.Slug,
			Body:        row.Body,
			SoWhat:      row.SoWhat,
			Tags:        row.Tags,
			Version:     row.Version,
			PublishedAt: row.PublishedAt,
		})
	}

	totalPages := int((totalCount + int64(blog.PostsPerPage) - 1) / int64(blog.PostsPerPage))
	if totalPages < 1 {
		totalPages = 1
	}

	app.render(w, r, http.StatusOK, blog.PostsIndex(posts, filters, totalPages, allTags))
}
