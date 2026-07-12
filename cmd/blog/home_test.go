package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/jonathanschwarzhaupt/home-blog/internal/assert"
	"github.com/jonathanschwarzhaupt/home-blog/internal/database"
	"github.com/jonathanschwarzhaupt/home-blog/internal/database/mocks"
)

func TestHome_ShowsFeaturedPostsAndProjectsInRankOrder(t *testing.T) {
	mockDB := &mocks.MockQuerier{
		ListFeaturedPostsFunc: func(ctx context.Context) ([]database.Post, error) {
			return []database.Post{
				{ID: 1, Title: "First Slot Post", Slug: "first-slot-post", SoWhat: "one", FeaturedRank: pgtype.Int4{Int32: 1, Valid: true}},
				{ID: 2, Title: "Second Slot Post", Slug: "second-slot-post", SoWhat: "two", FeaturedRank: pgtype.Int4{Int32: 2, Valid: true}},
			}, nil
		},
		ListFeaturedProjectsFunc: func(ctx context.Context) ([]database.Project, error) {
			return []database.Project{
				{ID: 1, Name: "Featured Project", Slug: "featured-project", Description: "desc", FeaturedRank: pgtype.Int4{Int32: 1, Valid: true}},
			}, nil
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	rs, err := http.Get(ts.URL + "/")
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
	firstIdx := strings.Index(html, "First Slot Post")
	secondIdx := strings.Index(html, "Second Slot Post")
	assert.True(t, firstIdx >= 0)
	assert.True(t, secondIdx >= 0)
	assert.True(t, firstIdx < secondIdx)

	assert.StringContains(t, html, "Featured Project")
	assert.StringContains(t, html, `href="/posts"`)
	assert.StringContains(t, html, `href="/projects"`)
	assert.StringContains(t, html, `href="/about"`)
}

func TestHome_RendersWithNoFeaturedContent(t *testing.T) {
	mockDB := &mocks.MockQuerier{
		ListFeaturedPostsFunc: func(ctx context.Context) ([]database.Post, error) {
			return nil, nil
		},
		ListFeaturedProjectsFunc: func(ctx context.Context) ([]database.Project, error) {
			return nil, nil
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	rs, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	assert.Equal(t, rs.StatusCode, http.StatusOK)
}
