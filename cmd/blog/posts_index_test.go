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
		ListPostsFunc: func(ctx context.Context) ([]database.Post, error) {
			return []database.Post{
				{ID: 1, Title: "First Post", Slug: "first-post", SoWhat: "It's the first one."},
				{ID: 2, Title: "Second Post", Slug: "second-post", SoWhat: "It's the second one."},
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
		ListPostsFunc: func(ctx context.Context) ([]database.Post, error) {
			return []database.Post{}, nil
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
