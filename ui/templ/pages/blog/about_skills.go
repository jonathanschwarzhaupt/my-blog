package blog

import "github.com/jonathanschwarzhaupt/home-blog/internal/models"

// SkillGroup is every Skill sharing a Category, in the order that Category
// is first encountered.
type SkillGroup struct {
	Category string
	Skills   []models.Skill
}

// GroupSkills groups skills (already ordered by OrderKey ascending) by
// Category, preserving the order each Category is first encountered — all
// Skills sharing a Category render together as one group, even if they
// aren't contiguous in the ordered input. See
// docs/adr/0009-about-page-db-backed-with-revision-history.md.
func GroupSkills(skills []models.Skill) []SkillGroup {
	var groups []SkillGroup
	index := make(map[string]int, len(skills))

	for _, s := range skills {
		i, ok := index[s.Category]
		if !ok {
			i = len(groups)
			index[s.Category] = i
			groups = append(groups, SkillGroup{Category: s.Category})
		}
		groups[i].Skills = append(groups[i].Skills, s)
	}

	return groups
}
