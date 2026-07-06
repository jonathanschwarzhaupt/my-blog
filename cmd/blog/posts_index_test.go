package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jonathanschwarzhaupt/my-blog/internal/assert"
	"github.com/jonathanschwarzhaupt/my-blog/internal/database"
	"github.com/jonathanschwarzhaupt/my-blog/internal/database/mocks"
)

func TestPostsIndex_ListsAllPosts(t *testing.T) {
	mockDB := &mocks.MockQuerier{
		ListPostsFilteredFunc: func(ctx context.Context, arg database.ListPostsFilteredParams) ([]database.ListPostsFilteredRow, error) {
			return []database.ListPostsFilteredRow{
				{ID: 1, Title: "First Post", Slug: "first-post", SoWhat: "It's the first one.", TotalCount: 2},
				{ID: 2, Title: "Second Post", Slug: "second-post", SoWhat: "It's the second one.", TotalCount: 2},
			}, nil
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

	assert.Equal(t, rs.StatusCode, http.StatusOK)

	html := string(body)
	assert.StringContains(t, html, "First Post")
	assert.StringContains(t, html, "Second Post")
	assert.StringContains(t, html, `href="/posts/first-post"`)
	assert.StringContains(t, html, `href="/posts/second-post"`)
}

func TestPostsIndex_NoPosts(t *testing.T) {
	mockDB := &mocks.MockQuerier{
		ListPostsFilteredFunc: func(ctx context.Context, arg database.ListPostsFilteredParams) ([]database.ListPostsFilteredRow, error) {
			return []database.ListPostsFilteredRow{}, nil
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
