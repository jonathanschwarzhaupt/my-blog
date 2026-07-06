package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/jonathanschwarzhaupt/my-blog/internal/assert"
	"github.com/jonathanschwarzhaupt/my-blog/internal/database"
	"github.com/jonathanschwarzhaupt/my-blog/internal/database/mocks"
)

func TestPostCreatePost_Valid(t *testing.T) {
	var gotParams database.InsertPostParams

	mockDB := &mocks.MockQuerier{
		ListProjectsFunc: func(ctx context.Context) ([]database.Project, error) {
			return nil, nil
		},
		InsertPostFunc: func(ctx context.Context, arg database.InsertPostParams) (database.Post, error) {
			gotParams = arg
			return database.Post{
				ID:      1,
				Title:   arg.Title,
				Slug:    arg.Slug,
				Body:    arg.Body,
				SoWhat:  arg.SoWhat,
				Tags:    arg.Tags,
				Version: 1,
			}, nil
		},
		DeletePostProjectsFunc: func(ctx context.Context, postID int64) error {
			return nil
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	client := ts.Client()
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	form := url.Values{}
	form.Set("title", "Hello World")
	form.Set("body", "This is the body")
	form.Set("so_what", "It matters because reasons")
	form.Set("tags", "go, blog")

	rs, err := client.PostForm(ts.URL+"/posts", form)
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	assert.Equal(t, rs.StatusCode, http.StatusSeeOther)
	assert.Equal(t, rs.Header.Get("Location"), "/posts/new")
	assert.Equal(t, gotParams.Title, "Hello World")
	assert.Equal(t, gotParams.Slug, "hello-world")
	assert.Equal(t, gotParams.SoWhat, "It matters because reasons")
	assert.Equal(t, len(gotParams.Tags), 2)
	assert.Equal(t, gotParams.Tags[0], "go")
	assert.Equal(t, gotParams.Tags[1], "blog")
}

func TestPostCreatePost_ExplicitPublishedAt(t *testing.T) {
	var gotParams database.InsertPostParams

	mockDB := &mocks.MockQuerier{
		ListProjectsFunc: func(ctx context.Context) ([]database.Project, error) {
			return nil, nil
		},
		InsertPostFunc: func(ctx context.Context, arg database.InsertPostParams) (database.Post, error) {
			gotParams = arg
			return database.Post{ID: 1, Title: arg.Title, Slug: arg.Slug, SoWhat: arg.SoWhat, Version: 1}, nil
		},
		DeletePostProjectsFunc: func(ctx context.Context, postID int64) error {
			return nil
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	client := ts.Client()
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	form := url.Values{}
	form.Set("title", "Ported Post")
	form.Set("body", "Body")
	form.Set("so_what", "It matters")
	form.Set("published_at", "2020-06-15")

	rs, err := client.PostForm(ts.URL+"/posts", form)
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	assert.Equal(t, rs.StatusCode, http.StatusSeeOther)

	if !gotParams.PublishedAt.Valid {
		t.Fatal("got PublishedAt.Valid = false; want true (explicit date was provided)")
	}
	assert.Equal(t, gotParams.PublishedAt.Time.Format("2006-01-02"), "2020-06-15")
}

func TestPostCreatePost_BlankPublishedAtLeavesDateUnset(t *testing.T) {
	var gotParams database.InsertPostParams

	mockDB := &mocks.MockQuerier{
		ListProjectsFunc: func(ctx context.Context) ([]database.Project, error) {
			return nil, nil
		},
		InsertPostFunc: func(ctx context.Context, arg database.InsertPostParams) (database.Post, error) {
			gotParams = arg
			return database.Post{ID: 1, Title: arg.Title, Slug: arg.Slug, SoWhat: arg.SoWhat, Version: 1}, nil
		},
		DeletePostProjectsFunc: func(ctx context.Context, postID int64) error {
			return nil
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	client := ts.Client()
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	form := url.Values{}
	form.Set("title", "Fresh Post")
	form.Set("body", "Body")
	form.Set("so_what", "It matters")
	// published_at deliberately omitted

	rs, err := client.PostForm(ts.URL+"/posts", form)
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	assert.Equal(t, rs.StatusCode, http.StatusSeeOther)

	// Leaving PublishedAt.Valid = false is exactly what lets the SQL layer's
	// COALESCE(..., now()) fall through to the column default — this test
	// only needs to confirm the handler didn't invent a value on its own.
	if gotParams.PublishedAt.Valid {
		t.Fatal("got PublishedAt.Valid = true; want false (no date was provided, so the DB default should apply)")
	}
}

func TestPostCreatePost_InvalidPublishedAtRejected(t *testing.T) {
	insertCallCount := 0

	mockDB := &mocks.MockQuerier{
		ListProjectsFunc: func(ctx context.Context) ([]database.Project, error) {
			return nil, nil
		},
		InsertPostFunc: func(ctx context.Context, arg database.InsertPostParams) (database.Post, error) {
			insertCallCount++
			return database.Post{}, nil
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	form := url.Values{}
	form.Set("title", "Hello World")
	form.Set("body", "Body")
	form.Set("so_what", "It matters")
	form.Set("published_at", "not-a-date")

	rs, err := http.PostForm(ts.URL+"/posts", form)
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	body, err := io.ReadAll(rs.Body)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, rs.StatusCode, http.StatusUnprocessableEntity)
	assert.Equal(t, insertCallCount, 0)
	assert.StringContains(t, string(body), "Must be a valid date")
}

func TestPostCreatePost_BlankSoWhat(t *testing.T) {
	insertCalled := false

	mockDB := &mocks.MockQuerier{
		ListProjectsFunc: func(ctx context.Context) ([]database.Project, error) {
			return nil, nil
		},
		InsertPostFunc: func(ctx context.Context, arg database.InsertPostParams) (database.Post, error) {
			insertCalled = true
			return database.Post{}, nil
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	form := url.Values{}
	form.Set("title", "Hello World")
	form.Set("body", "This is the body")
	form.Set("so_what", "")

	rs, err := http.PostForm(ts.URL+"/posts", form)
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	body, err := io.ReadAll(rs.Body)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, rs.StatusCode, http.StatusUnprocessableEntity)
	assert.False(t, insertCalled)
	assert.StringContains(t, string(body), "This field cannot be blank")
}

func TestPostCreatePost_CrossOriginRejected(t *testing.T) {
	insertCalled := false

	mockDB := &mocks.MockQuerier{
		ListProjectsFunc: func(ctx context.Context) ([]database.Project, error) {
			return nil, nil
		},
		InsertPostFunc: func(ctx context.Context, arg database.InsertPostParams) (database.Post, error) {
			insertCalled = true
			return database.Post{}, nil
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	form := url.Values{}
	form.Set("title", "Hello World")
	form.Set("body", "This is the body")
	form.Set("so_what", "It matters")

	req, err := http.NewRequest(http.MethodPost, ts.URL+"/posts", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "https://evil.example.com")

	rs, err := ts.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	assert.Equal(t, rs.StatusCode, http.StatusForbidden)
	assert.False(t, insertCalled)
}

func TestPostCreatePost_EmptyTitleSlug(t *testing.T) {
	insertCalled := false

	mockDB := &mocks.MockQuerier{
		ListProjectsFunc: func(ctx context.Context) ([]database.Project, error) {
			return nil, nil
		},
		InsertPostFunc: func(ctx context.Context, arg database.InsertPostParams) (database.Post, error) {
			insertCalled = true
			return database.Post{}, nil
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	form := url.Values{}
	form.Set("title", "???")
	form.Set("body", "This is the body")
	form.Set("so_what", "It matters")

	rs, err := http.PostForm(ts.URL+"/posts", form)
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	body, err := io.ReadAll(rs.Body)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, rs.StatusCode, http.StatusUnprocessableEntity)
	assert.False(t, insertCalled)
	assert.StringContains(t, string(body), "Title must contain at least one letter or number")
}

func TestPostCreatePost_DuplicateSlug(t *testing.T) {
	insertCallCount := 0

	mockDB := &mocks.MockQuerier{
		ListProjectsFunc: func(ctx context.Context) ([]database.Project, error) {
			return nil, nil
		},
		InsertPostFunc: func(ctx context.Context, arg database.InsertPostParams) (database.Post, error) {
			insertCallCount++
			return database.Post{}, &pgconn.PgError{Code: "23505"}
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	form := url.Values{}
	form.Set("title", "Hello World")
	form.Set("body", "This is the body")
	form.Set("so_what", "It matters")

	rs, err := http.PostForm(ts.URL+"/posts", form)
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	body, err := io.ReadAll(rs.Body)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, rs.StatusCode, http.StatusUnprocessableEntity)
	assert.Equal(t, insertCallCount, 1)
	assert.StringContains(t, string(body), "already exists")
}
