package models

import "github.com/jonathanschwarzhaupt/home-blog/internal/database"

// Skill is a technology/tool shown on the About page, grouped under a
// free-text Category — see CONTEXT.md's Skill entry. Category isn't a
// separate curated entity: the About page groups Skills by whatever
// distinct Category values exist.
type Skill struct {
	ID       int64
	Category string
	Name     string
	OrderKey float64
}

func SkillFromDatabase(s database.Skill) Skill {
	return Skill{
		ID:       s.ID,
		Category: s.Category,
		Name:     s.Name,
		OrderKey: s.OrderKey,
	}
}
