package models

import (
	"github.com/jonathanschwarzhaupt/my-blog/internal/database"
)

type Project struct {
	ID          int64
	Name        string
	Slug        string
	Description string
}

func ProjectFromDatabase(p database.Project) Project {
	return Project{
		ID:          p.ID,
		Name:        p.Name,
		Slug:        p.Slug,
		Description: p.Description,
	}
}
