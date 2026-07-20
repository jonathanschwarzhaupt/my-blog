package main

import (
	"context"
	"net/http"
	"time"

	"github.com/jonathanschwarzhaupt/home-blog/internal/markdown"
	"github.com/jonathanschwarzhaupt/home-blog/internal/models"
	"github.com/jonathanschwarzhaupt/home-blog/ui/templ/pages/blog"
)

func (app *application) about(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	revision, ok := app.latestAboutRevisionOrServerError(ctx, w, r)
	if !ok {
		return
	}

	bodyHTML, err := markdown.Render(revision.Body)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	skills, ok := app.listSkillsOrServerError(ctx, w, r)
	if !ok {
		return
	}

	app.render(w, r, http.StatusOK, blog.About(bodyHTML, blog.GroupSkills(skills)))
}

// latestAboutRevisionOrServerError fetches the current About content,
// replying with a 500 and returning ok = false on failure so the caller can
// return immediately. An empty about_revision table isn't a routine
// "not found" case the way a missing post/project slug is — the seed
// migration guarantees at least one row, and revisions are never pruned —
// so a genuinely empty table is a data-integrity anomaly worth a 500 and a
// logged error, not a silently-handled 404.
func (app *application) latestAboutRevisionOrServerError(ctx context.Context, w http.ResponseWriter, r *http.Request) (models.AboutRevision, bool) {
	dbRevision, err := app.db.GetLatestAboutRevision(ctx)
	if err != nil {
		app.serverError(w, r, models.WrapDBError(err))
		return models.AboutRevision{}, false
	}
	return models.AboutRevisionFromDatabase(dbRevision), true
}

func (app *application) listSkillsOrServerError(ctx context.Context, w http.ResponseWriter, r *http.Request) ([]models.Skill, bool) {
	dbSkills, err := app.db.ListSkillsByOrder(ctx)
	if err != nil {
		app.serverError(w, r, models.WrapDBError(err))
		return nil, false
	}
	skills := make([]models.Skill, len(dbSkills))
	for i, s := range dbSkills {
		skills[i] = models.SkillFromDatabase(s)
	}
	return skills, true
}
