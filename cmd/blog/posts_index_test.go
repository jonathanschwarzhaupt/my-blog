package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/jonathanschwarzhaupt/home-blog/internal/assert"
	"github.com/jonathanschwarzhaupt/home-blog/internal/database"
	"github.com/jonathanschwarzhaupt/home-blog/internal/database/mocks"
)

func TestPostsIndex_ListsAllPosts(t *testing.T) {
	mockDB := &mocks.MockQuerier{
		ListPostsFilteredFunc: func(ctx context.Context, arg database.ListPostsFilteredParams) ([]database.ListPostsFilteredRow, error) {
			return []database.ListPostsFilteredRow{
				{ID: 1, Title: "First Post", Slug: "first-post", SoWhat: "It's the first one.", TotalCount: 2},
				{ID: 2, Title: "Second Post", Slug: "second-post", SoWhat: "It's the second one.", TotalCount: 2},
			}, nil
		},
		ListDistinctTagsFunc: func(ctx context.Context) ([]string, error) { return nil, nil },
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	rs, err := http.Get(ts.URL + "/posts")
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
	assert.StringContains(t, html, "First Post")
	assert.StringContains(t, html, "Second Post")
	assert.StringContains(t, html, `href="/posts/first-post"`)
	assert.StringContains(t, html, `href="/posts/second-post"`)
}

func TestPostsIndex_ShowsPublishedDateOnEachCard(t *testing.T) {
	mockDB := &mocks.MockQuerier{
		ListPostsFilteredFunc: func(ctx context.Context, arg database.ListPostsFilteredParams) ([]database.ListPostsFilteredRow, error) {
			return []database.ListPostsFilteredRow{
				{
					ID:          1,
					Title:       "Tagged Post",
					Slug:        "tagged-post",
					SoWhat:      "It's got tags.",
					Tags:        []string{"go"},
					PublishedAt: pgtype.Timestamptz{Time: time.Date(2026, time.January, 22, 0, 0, 0, 0, time.UTC), Valid: true},
					TotalCount:  2,
				},
				{
					ID:          2,
					Title:       "Untagged Post",
					Slug:        "untagged-post",
					SoWhat:      "No tags here.",
					PublishedAt: pgtype.Timestamptz{Time: time.Date(2026, time.February, 3, 0, 0, 0, 0, time.UTC), Valid: true},
					TotalCount:  2,
				},
			}, nil
		},
		ListDistinctTagsFunc: func(ctx context.Context) ([]string, error) { return nil, nil },
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	rs, err := http.Get(ts.URL + "/posts")
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	body, err := io.ReadAll(rs.Body)
	if err != nil {
		t.Fatal(err)
	}

	html := string(body)
	assert.StringContains(t, html, "2026-01-22")
	assert.StringContains(t, html, "2026-02-03")
}

func TestPostsIndex_NoPosts(t *testing.T) {
	mockDB := &mocks.MockQuerier{
		ListPostsFilteredFunc: func(ctx context.Context, arg database.ListPostsFilteredParams) ([]database.ListPostsFilteredRow, error) {
			return []database.ListPostsFilteredRow{}, nil
		},
		ListDistinctTagsFunc: func(ctx context.Context) ([]string, error) { return nil, nil },
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	rs, err := http.Get(ts.URL + "/posts")
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	assert.Equal(t, rs.StatusCode, http.StatusOK)
}

func TestPostsIndex_SinglePostSlugStillWorks(t *testing.T) {
	mockDB := &mocks.MockQuerier{
		GetPostFunc: func(ctx context.Context, slug string) (database.Post, error) {
			return database.Post{ID: 1, Title: "A Post", Slug: slug}, nil
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	rs, err := http.Get(ts.URL + "/posts/a-post")
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	assert.Equal(t, rs.StatusCode, http.StatusOK)
}

func TestPostsIndex_PassesFiltersThroughToTheQuery(t *testing.T) {
	var gotParams database.ListPostsFilteredParams

	mockDB := &mocks.MockQuerier{
		ListPostsFilteredFunc: func(ctx context.Context, arg database.ListPostsFilteredParams) ([]database.ListPostsFilteredRow, error) {
			gotParams = arg
			return []database.ListPostsFilteredRow{}, nil
		},
		ListDistinctTagsFunc: func(ctx context.Context) ([]string, error) { return nil, nil },
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	rs, err := http.Get(ts.URL + "/posts?sort=oldest&from=2020-01-01&to=2020-12-31&page=3")
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	assert.Equal(t, rs.StatusCode, http.StatusOK)
	assert.Equal(t, gotParams.SortOldest, true)
	assert.Equal(t, gotParams.FromDate.Valid, true)
	assert.Equal(t, gotParams.FromDate.Time.Format("2006-01-02"), "2020-01-01")
	assert.Equal(t, gotParams.ToDate.Valid, true)
	assert.Equal(t, gotParams.ToDate.Time.Format("2006-01-02"), "2020-12-31")
	assert.Equal(t, gotParams.PageLimit, int32(7))
	assert.Equal(t, gotParams.PageOffset, int32(14)) // (page 3 - 1) * 7
}

func TestPostsIndex_InvalidQueryParamsFallBackToDefaults(t *testing.T) {
	var gotParams database.ListPostsFilteredParams

	mockDB := &mocks.MockQuerier{
		ListPostsFilteredFunc: func(ctx context.Context, arg database.ListPostsFilteredParams) ([]database.ListPostsFilteredRow, error) {
			gotParams = arg
			return []database.ListPostsFilteredRow{}, nil
		},
		ListDistinctTagsFunc: func(ctx context.Context) ([]string, error) { return nil, nil },
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	rs, err := http.Get(ts.URL + "/posts?sort=sideways&from=not-a-date&page=-5")
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	assert.Equal(t, rs.StatusCode, http.StatusOK)
	assert.Equal(t, gotParams.SortOldest, false) // invalid sort falls back to "newest"
	assert.Equal(t, gotParams.FromDate.Valid, false)
	assert.Equal(t, gotParams.PageOffset, int32(0)) // invalid page falls back to 1
}

func TestPostsIndex_PageBeyondLastPageReturnsEmptyNotError(t *testing.T) {
	mockDB := &mocks.MockQuerier{
		ListPostsFilteredFunc: func(ctx context.Context, arg database.ListPostsFilteredParams) ([]database.ListPostsFilteredRow, error) {
			return []database.ListPostsFilteredRow{}, nil
		},
		ListDistinctTagsFunc: func(ctx context.Context) ([]string, error) { return nil, nil },
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	rs, err := http.Get(ts.URL + "/posts?page=999")
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	body, err := io.ReadAll(rs.Body)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, rs.StatusCode, http.StatusOK)
	assert.StringContains(t, string(body), "No posts match these filters.")
}

func TestPostsIndex_TagQueryParamPassedThrough(t *testing.T) {
	var gotParams database.ListPostsFilteredParams

	mockDB := &mocks.MockQuerier{
		ListPostsFilteredFunc: func(ctx context.Context, arg database.ListPostsFilteredParams) ([]database.ListPostsFilteredRow, error) {
			gotParams = arg
			return []database.ListPostsFilteredRow{}, nil
		},
		ListDistinctTagsFunc: func(ctx context.Context) ([]string, error) { return nil, nil },
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	rs, err := http.Get(ts.URL + "/posts?tag=go")
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	assert.Equal(t, rs.StatusCode, http.StatusOK)
	assert.Equal(t, gotParams.Tag.Valid, true)
	assert.Equal(t, gotParams.Tag.String, "go")
}

func TestPostsIndex_NoTagQueryParamLeavesTagUnset(t *testing.T) {
	var gotParams database.ListPostsFilteredParams

	mockDB := &mocks.MockQuerier{
		ListPostsFilteredFunc: func(ctx context.Context, arg database.ListPostsFilteredParams) ([]database.ListPostsFilteredRow, error) {
			gotParams = arg
			return []database.ListPostsFilteredRow{}, nil
		},
		ListDistinctTagsFunc: func(ctx context.Context) ([]string, error) { return nil, nil },
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	rs, err := http.Get(ts.URL + "/posts")
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	assert.Equal(t, rs.StatusCode, http.StatusOK)
	assert.Equal(t, gotParams.Tag.Valid, false)
}

func TestPostsIndex_ShowsAllDistinctTagsAsLinks(t *testing.T) {
	mockDB := &mocks.MockQuerier{
		ListPostsFilteredFunc: func(ctx context.Context, arg database.ListPostsFilteredParams) ([]database.ListPostsFilteredRow, error) {
			return []database.ListPostsFilteredRow{}, nil
		},
		ListDistinctTagsFunc: func(ctx context.Context) ([]string, error) {
			return []string{"go", "homelab", "postgres"}, nil
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	rs, err := http.Get(ts.URL + "/posts")
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	body, err := io.ReadAll(rs.Body)
	if err != nil {
		t.Fatal(err)
	}

	html := string(body)
	assert.StringContains(t, html, `href="/posts?tag=go"`)
	assert.StringContains(t, html, `href="/posts?tag=homelab"`)
	assert.StringContains(t, html, `href="/posts?tag=postgres"`)
}
