package models

import (
	"testing"

	"github.com/jonathanschwarzhaupt/my-blog/internal/assert"
	"github.com/jonathanschwarzhaupt/my-blog/internal/database"
)

func TestProjectFromDatabase(t *testing.T) {
	dbProject := database.Project{
		ID:          1,
		Name:        "Homelab",
		Slug:        "homelab",
		Description: "Everything about running my own infrastructure.",
	}

	got := ProjectFromDatabase(dbProject)

	assert.Equal(t, got.ID, int64(1))
	assert.Equal(t, got.Name, "Homelab")
	assert.Equal(t, got.Slug, "homelab")
	assert.Equal(t, got.Description, "Everything about running my own infrastructure.")
}
