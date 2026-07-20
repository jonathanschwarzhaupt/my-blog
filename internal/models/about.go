package models

import (
	"time"

	"github.com/jonathanschwarzhaupt/home-blog/internal/database"
)

// AboutRevision is a single historical snapshot of the About page's
// content — see CONTEXT.md's Revision entry for how this differs from a
// Post/Project's Version field. Rows are never updated, only inserted.
type AboutRevision struct {
	ID        int64
	Body      string
	CreatedAt time.Time
}

func AboutRevisionFromDatabase(r database.AboutRevision) AboutRevision {
	return AboutRevision{
		ID:        r.ID,
		Body:      r.Body,
		CreatedAt: r.CreatedAt.Time,
	}
}
