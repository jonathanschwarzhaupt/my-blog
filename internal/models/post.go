package models

import (
	"time"

	"github.com/jonathanschwarzhaupt/home-blog/internal/database"
)

type Post struct {
	ID          int64
	Title       string
	Slug        string
	Body        string
	SoWhat      string
	Tags        []string
	Version     int32
	PublishedAt time.Time
}

func PostFromDatabase(p database.Post) Post {
	return Post{
		ID:          p.ID,
		Title:       p.Title,
		Slug:        p.Slug,
		Body:        p.Body,
		SoWhat:      p.SoWhat,
		Tags:        p.Tags,
		Version:     p.Version,
		PublishedAt: p.PublishedAt.Time,
	}
}
