package models

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/jonathanschwarzhaupt/home-blog/internal/assert"
	"github.com/jonathanschwarzhaupt/home-blog/internal/database"
)

func TestProjectFromDatabase(t *testing.T) {
	createdAt := time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC)

	dbProject := database.Project{
		ID:          1,
		Name:        "Homelab",
		Slug:        "homelab",
		Description: "Everything about running my own infrastructure.",
		CreatedAt:   pgtype.Timestamptz{Time: createdAt, Valid: true},
	}

	got := ProjectFromDatabase(dbProject)

	assert.Equal(t, got.ID, int64(1))
	assert.Equal(t, got.Name, "Homelab")
	assert.Equal(t, got.Slug, "homelab")
	assert.Equal(t, got.Description, "Everything about running my own infrastructure.")
	assert.Equal(t, got.CreatedAt, createdAt)
}
