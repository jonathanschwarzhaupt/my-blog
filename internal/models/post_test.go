package models

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/jonathanschwarzhaupt/home-blog/internal/assert"
	"github.com/jonathanschwarzhaupt/home-blog/internal/database"
)

func TestPostFromDatabase(t *testing.T) {
	published := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)

	dbPost := database.Post{
		ID:          1,
		Title:       "Hello",
		Slug:        "hello",
		Body:        "World",
		SoWhat:      "It matters",
		Tags:        []string{"go", "blog"},
		Version:     1,
		PublishedAt: pgtype.Timestamptz{Time: published, Valid: true},
	}

	got := PostFromDatabase(dbPost)

	assert.Equal(t, got.ID, int64(1))
	assert.Equal(t, got.Title, "Hello")
	assert.Equal(t, got.Slug, "hello")
	assert.Equal(t, got.Body, "World")
	assert.Equal(t, got.SoWhat, "It matters")
	assert.Equal(t, got.Version, int32(1))
	assert.True(t, got.PublishedAt.Equal(published))
}
