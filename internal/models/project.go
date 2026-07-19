package models

import (
	"time"

	"github.com/jonathanschwarzhaupt/home-blog/internal/database"
)

type Project struct {
	ID          int64
	Name        string
	Slug        string
	Description string
	CreatedAt   time.Time
	OrderKey    float64
}

func ProjectFromDatabase(p database.Project) Project {
	return Project{
		ID:          p.ID,
		Name:        p.Name,
		Slug:        p.Slug,
		Description: p.Description,
		CreatedAt:   p.CreatedAt.Time,
		OrderKey:    p.OrderKey,
	}
}
