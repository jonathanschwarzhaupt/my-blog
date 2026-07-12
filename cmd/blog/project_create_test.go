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

	"github.com/jonathanschwarzhaupt/home-blog/internal/assert"
	"github.com/jonathanschwarzhaupt/home-blog/internal/database"
	"github.com/jonathanschwarzhaupt/home-blog/internal/database/mocks"
)

func TestProjectCreatePost_Valid(t *testing.T) {
	var gotParams database.InsertProjectParams

	mockDB := &mocks.MockQuerier{
		InsertProjectFunc: func(ctx context.Context, arg database.InsertProjectParams) (database.Project, error) {
			gotParams = arg
			return database.Project{ID: 1, Name: arg.Name, Slug: arg.Slug, Description: arg.Description}, nil
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
	form.Set("name", "Homelab")
	form.Set("description", "Everything about running my own infrastructure.")

	rs, err := client.PostForm(ts.URL+"/projects", form)
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	assert.Equal(t, rs.StatusCode, http.StatusSeeOther)
	assert.Equal(t, rs.Header.Get("Location"), "/projects/new")
	assert.Equal(t, gotParams.Name, "Homelab")
	assert.Equal(t, gotParams.Slug, "homelab")
	assert.Equal(t, gotParams.Description, "Everything about running my own infrastructure.")
}

func TestProjectCreatePost_ExplicitCreatedAt(t *testing.T) {
	var gotParams database.InsertProjectParams

	mockDB := &mocks.MockQuerier{
		InsertProjectFunc: func(ctx context.Context, arg database.InsertProjectParams) (database.Project, error) {
			gotParams = arg
			return database.Project{ID: 1, Name: arg.Name, Slug: arg.Slug}, nil
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
	form.Set("name", "Old Project")
	form.Set("created_at", "2018-09-01")

	rs, err := client.PostForm(ts.URL+"/projects", form)
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	assert.Equal(t, rs.StatusCode, http.StatusSeeOther)

	if !gotParams.CreatedAt.Valid {
		t.Fatal("got CreatedAt.Valid = false; want true (explicit date was provided)")
	}
	assert.Equal(t, gotParams.CreatedAt.Time.Format("2006-01-02"), "2018-09-01")
}

func TestProjectCreatePost_BlankCreatedAtLeavesDateUnset(t *testing.T) {
	var gotParams database.InsertProjectParams

	mockDB := &mocks.MockQuerier{
		InsertProjectFunc: func(ctx context.Context, arg database.InsertProjectParams) (database.Project, error) {
			gotParams = arg
			return database.Project{ID: 1, Name: arg.Name, Slug: arg.Slug}, nil
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
	form.Set("name", "Fresh Project")
	// created_at deliberately omitted

	rs, err := client.PostForm(ts.URL+"/projects", form)
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	assert.Equal(t, rs.StatusCode, http.StatusSeeOther)

	if gotParams.CreatedAt.Valid {
		t.Fatal("got CreatedAt.Valid = true; want false (no date was provided, so the DB default should apply)")
	}
}

func TestProjectCreatePost_InvalidCreatedAtRejected(t *testing.T) {
	insertCallCount := 0

	mockDB := &mocks.MockQuerier{
		InsertProjectFunc: func(ctx context.Context, arg database.InsertProjectParams) (database.Project, error) {
			insertCallCount++
			return database.Project{}, nil
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	form := url.Values{}
	form.Set("name", "Homelab")
	form.Set("created_at", "not-a-date")

	rs, err := http.PostForm(ts.URL+"/projects", form)
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

func TestProjectCreatePost_BlankName(t *testing.T) {
	insertCalled := false

	mockDB := &mocks.MockQuerier{
		InsertProjectFunc: func(ctx context.Context, arg database.InsertProjectParams) (database.Project, error) {
			insertCalled = true
			return database.Project{}, nil
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	form := url.Values{}
	form.Set("name", "")
	form.Set("description", "x")

	rs, err := http.PostForm(ts.URL+"/projects", form)
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

func TestProjectCreatePost_DuplicateSlug(t *testing.T) {
	insertCallCount := 0

	mockDB := &mocks.MockQuerier{
		InsertProjectFunc: func(ctx context.Context, arg database.InsertProjectParams) (database.Project, error) {
			insertCallCount++
			return database.Project{}, &pgconn.PgError{Code: "23505"}
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	form := url.Values{}
	form.Set("name", "Homelab")
	form.Set("description", "x")

	rs, err := http.PostForm(ts.URL+"/projects", form)
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

func TestProjectCreatePost_EmptyNameSlug(t *testing.T) {
	insertCalled := false

	mockDB := &mocks.MockQuerier{
		InsertProjectFunc: func(ctx context.Context, arg database.InsertProjectParams) (database.Project, error) {
			insertCalled = true
			return database.Project{}, nil
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	form := url.Values{}
	form.Set("name", "???")
	form.Set("description", "x")

	rs, err := http.PostForm(ts.URL+"/projects", form)
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
	assert.StringContains(t, string(body), "Name must contain at least one letter or number")
}

func TestPostCreatePost_AssignsValidProjects(t *testing.T) {
	var gotPostID int64
	var gotAssociations []database.InsertPostProjectParams

	mockDB := &mocks.MockQuerier{
		ListProjectsFunc: func(ctx context.Context) ([]database.Project, error) {
			return nil, nil
		},
		GetProjectsByIDsFunc: func(ctx context.Context, ids []int64) ([]database.Project, error) {
			assert.Equal(t, len(ids), 2)
			return []database.Project{{ID: 1}, {ID: 2}}, nil
		},
		InsertPostFunc: func(ctx context.Context, arg database.InsertPostParams) (database.Post, error) {
			return database.Post{ID: 7, Title: arg.Title, Slug: arg.Slug, SoWhat: arg.SoWhat}, nil
		},
		DeletePostProjectsFunc: func(ctx context.Context, postID int64) error {
			gotPostID = postID
			return nil
		},
		InsertPostProjectFunc: func(ctx context.Context, arg database.InsertPostProjectParams) error {
			gotAssociations = append(gotAssociations, arg)
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
	form.Set("body", "body")
	form.Set("so_what", "it matters")
	form.Add("project_ids", "1")
	form.Add("project_ids", "2")

	rs, err := client.PostForm(ts.URL+"/posts", form)
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	assert.Equal(t, rs.StatusCode, http.StatusSeeOther)
	assert.Equal(t, gotPostID, int64(7))
	assert.Equal(t, len(gotAssociations), 2)
	assert.Equal(t, gotAssociations[0].PostID, int64(7))
	assert.Equal(t, gotAssociations[0].ProjectID, int64(1))
	assert.Equal(t, gotAssociations[1].ProjectID, int64(2))
}

func TestPostCreatePost_RejectsNonexistentProject(t *testing.T) {
	insertPostCalled := false

	mockDB := &mocks.MockQuerier{
		ListProjectsFunc: func(ctx context.Context) ([]database.Project, error) {
			return nil, nil
		},
		GetProjectsByIDsFunc: func(ctx context.Context, ids []int64) ([]database.Project, error) {
			return []database.Project{}, nil // none of the requested IDs exist
		},
		InsertPostFunc: func(ctx context.Context, arg database.InsertPostParams) (database.Post, error) {
			insertPostCalled = true
			return database.Post{}, nil
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	form := url.Values{}
	form.Set("title", "Hello World")
	form.Set("body", "body")
	form.Set("so_what", "it matters")
	form.Set("project_ids", "999")

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
	assert.False(t, insertPostCalled)
	assert.StringContains(t, string(body), "selected projects don")
}

func TestPostEdit_PreChecksExistingProjects(t *testing.T) {
	mockDB := &mocks.MockQuerier{
		GetPostFunc: func(ctx context.Context, slug string) (database.Post, error) {
			return database.Post{ID: 7, Slug: slug, Version: 1}, nil
		},
		ListProjectsFunc: func(ctx context.Context) ([]database.Project, error) {
			return []database.Project{{ID: 1, Name: "Homelab"}, {ID: 2, Name: "Blog Rebuild"}}, nil
		},
		GetProjectsForPostFunc: func(ctx context.Context, postID int64) ([]database.Project, error) {
			return []database.Project{{ID: 2, Name: "Blog Rebuild"}}, nil
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	rs, err := http.Get(ts.URL + "/posts/some-post/edit")
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	body, err := io.ReadAll(rs.Body)
	if err != nil {
		t.Fatal(err)
	}

	html := string(body)
	assert.StringContains(t, html, "Homelab")
	assert.StringContains(t, html, "Blog Rebuild")

	// The checkbox for project id=2 (the post's current association) should
	// carry the checked attribute; found by locating its own <input> tag
	// rather than assuming a fixed attribute order, since that's an
	// implementation detail of the checkbox component, not our markup.
	valueIdx := strings.Index(html, `value="2"`)
	assert.True(t, valueIdx >= 0)
	tagStart := strings.LastIndex(html[:valueIdx], "<input")
	assert.True(t, tagStart >= 0)
	tagEnd := strings.Index(html[tagStart:], ">")
	assert.True(t, tagEnd >= 0)
	assert.StringContains(t, html[tagStart:tagStart+tagEnd], "checked")
}

func TestPostUpdate_RejectsNonexistentProject(t *testing.T) {
	updateCalled := false

	mockDB := &mocks.MockQuerier{
		GetPostFunc: func(ctx context.Context, slug string) (database.Post, error) {
			return database.Post{ID: 7, Slug: slug, Version: 1}, nil
		},
		ListProjectsFunc: func(ctx context.Context) ([]database.Project, error) {
			return nil, nil
		},
		GetProjectsByIDsFunc: func(ctx context.Context, ids []int64) ([]database.Project, error) {
			return []database.Project{}, nil // none of the requested IDs exist
		},
		UpdatePostFunc: func(ctx context.Context, arg database.UpdatePostParams) (database.Post, error) {
			updateCalled = true
			return database.Post{}, nil
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	form := url.Values{}
	form.Set("version", "1")
	form.Set("title", "Updated Title")
	form.Set("body", "body")
	form.Set("so_what", "it matters")
	form.Set("project_ids", "999")
	form.Set("published_at", "2026-01-01")

	rs, err := http.PostForm(ts.URL+"/posts/some-post/edit", form)
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	body, err := io.ReadAll(rs.Body)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, rs.StatusCode, http.StatusUnprocessableEntity)
	assert.False(t, updateCalled)
	assert.StringContains(t, string(body), "selected projects don")
}

func TestPostUpdate_RemovesAllProjectAssociations(t *testing.T) {
	deleteCalled := false
	insertPostProjectCalled := false

	mockDB := &mocks.MockQuerier{
		GetPostFunc: func(ctx context.Context, slug string) (database.Post, error) {
			return database.Post{ID: 7, Slug: slug, Version: 1}, nil
		},
		UpdatePostFunc: func(ctx context.Context, arg database.UpdatePostParams) (database.Post, error) {
			return database.Post{ID: arg.ID, Version: arg.Version + 1}, nil
		},
		DeletePostProjectsFunc: func(ctx context.Context, postID int64) error {
			deleteCalled = true
			assert.Equal(t, postID, int64(7))
			return nil
		},
		InsertPostProjectFunc: func(ctx context.Context, arg database.InsertPostProjectParams) error {
			insertPostProjectCalled = true
			return nil
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	// No project_ids submitted at all — simulates unchecking every box.
	form := url.Values{}
	form.Set("version", "1")
	form.Set("title", "Updated Title")
	form.Set("body", "body")
	form.Set("so_what", "it matters")
	form.Set("published_at", "2026-01-01")

	client := ts.Client()
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	rs, err := client.PostForm(ts.URL+"/posts/some-post/edit", form)
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	assert.Equal(t, rs.StatusCode, http.StatusSeeOther)
	assert.True(t, deleteCalled)
	assert.False(t, insertPostProjectCalled)
}
