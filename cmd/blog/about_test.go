package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/jonathanschwarzhaupt/home-blog/internal/assert"
	"github.com/jonathanschwarzhaupt/home-blog/internal/database"
	"github.com/jonathanschwarzhaupt/home-blog/internal/database/mocks"
)

func TestAbout_RendersLatestRevisionAndSkillGroups(t *testing.T) {
	mockDB := &mocks.MockQuerier{
		GetLatestAboutRevisionFunc: func(ctx context.Context) (database.AboutRevision, error) {
			return database.AboutRevision{
				ID:   2,
				Body: "## My Journey\n\nSome text. I [wrote it up](/posts/gizmosql-in-kubernetes).",
			}, nil
		},
		ListSkillsByOrderFunc: func(ctx context.Context) ([]database.Skill, error) {
			return []database.Skill{
				{ID: 1, Category: "Languages", Name: "Go", OrderKey: 1},
				{ID: 2, Category: "Languages", Name: "Python", OrderKey: 2},
				{ID: 3, Category: "Cloud", Name: "AWS", OrderKey: 3},
			}, nil
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	rs, err := http.Get(ts.URL + "/about")
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	body, err := io.ReadAll(rs.Body)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, rs.StatusCode, http.StatusOK)

	html := string(body)
	assert.StringContains(t, html, "<img")
	assert.StringContains(t, html, "My Journey")
	assert.StringContains(t, html, `href="/posts/gizmosql-in-kubernetes"`)
	assert.StringContains(t, html, "Technical Skills")
	assert.StringContains(t, html, "Languages")
	assert.StringContains(t, html, "Go")
	assert.StringContains(t, html, "Python")
	assert.StringContains(t, html, "Cloud")
	assert.StringContains(t, html, "AWS")
}

func TestAbout_NoSkillsHidesSkillsSection(t *testing.T) {
	mockDB := &mocks.MockQuerier{
		GetLatestAboutRevisionFunc: func(ctx context.Context) (database.AboutRevision, error) {
			return database.AboutRevision{ID: 1, Body: "Just prose, no skills yet."}, nil
		},
		ListSkillsByOrderFunc: func(ctx context.Context) ([]database.Skill, error) {
			return []database.Skill{}, nil
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	rs, err := http.Get(ts.URL + "/about")
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	body, err := io.ReadAll(rs.Body)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, rs.StatusCode, http.StatusOK)
	html := string(body)
	assert.StringContains(t, html, "Just prose, no skills yet.")
	assert.False(t, strings.Contains(html, "Technical Skills"))
}

func TestAbout_ServerErrorWhenRevisionFetchFails(t *testing.T) {
	mockDB := &mocks.MockQuerier{
		GetLatestAboutRevisionFunc: func(ctx context.Context) (database.AboutRevision, error) {
			return database.AboutRevision{}, pgx.ErrNoRows
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	rs, err := http.Get(ts.URL + "/about")
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	assert.Equal(t, rs.StatusCode, http.StatusInternalServerError)
}
