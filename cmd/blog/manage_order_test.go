package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/jonathanschwarzhaupt/home-blog/internal/assert"
	"github.com/jonathanschwarzhaupt/home-blog/internal/database"
	"github.com/jonathanschwarzhaupt/home-blog/internal/database/mocks"
)

func TestManageOrder_RendersProjectsInOrder(t *testing.T) {
	mockDB := &mocks.MockQuerier{
		ListProjectsByOrderFunc: func(ctx context.Context) ([]database.Project, error) {
			return []database.Project{
				{ID: 1, Name: "Homelab", Slug: "homelab", OrderKey: 1},
				{ID: 2, Name: "Blog Rebuild", Slug: "blog-rebuild", OrderKey: 2.5},
			}, nil
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	rs, err := http.Get(ts.URL + "/admin/order")
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
	assert.StringContains(t, html, "Homelab")
	assert.StringContains(t, html, "Blog Rebuild")
	assert.StringContains(t, html, `name="order_key_1"`)
	assert.StringContains(t, html, `value="1"`)
	assert.StringContains(t, html, `name="order_key_2"`)
	assert.StringContains(t, html, `value="2.5"`)

	firstIdx := strings.Index(html, "Homelab")
	secondIdx := strings.Index(html, "Blog Rebuild")
	assert.True(t, firstIdx >= 0 && secondIdx >= 0 && firstIdx < secondIdx)
}

func TestManageOrderPost_UpdatesAllProjects(t *testing.T) {
	var gotUpdates []database.UpdateProjectOrderKeyParams

	mockDB := &mocks.MockQuerier{
		ListProjectsByOrderFunc: func(ctx context.Context) ([]database.Project, error) {
			return []database.Project{
				{ID: 1, Name: "Homelab", Slug: "homelab", OrderKey: 1},
				{ID: 2, Name: "Blog Rebuild", Slug: "blog-rebuild", OrderKey: 2},
			}, nil
		},
		UpdateProjectOrderKeyFunc: func(ctx context.Context, arg database.UpdateProjectOrderKeyParams) error {
			gotUpdates = append(gotUpdates, arg)
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
	form.Set("order_key_1", "0.5")
	form.Set("order_key_2", "1")

	rs, err := client.PostForm(ts.URL+"/admin/order", form)
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	assert.Equal(t, rs.StatusCode, http.StatusSeeOther)
	assert.Equal(t, rs.Header.Get("Location"), "/admin/order")
	assert.Equal(t, len(gotUpdates), 2)
	assert.Equal(t, gotUpdates[0].ID, int64(1))
	assert.Equal(t, gotUpdates[0].OrderKey, 0.5)
	assert.Equal(t, gotUpdates[1].ID, int64(2))
	assert.Equal(t, gotUpdates[1].OrderKey, float64(1))
}

func TestManageOrderPost_InvalidValueReRendersForm(t *testing.T) {
	updateCallCount := 0

	mockDB := &mocks.MockQuerier{
		ListProjectsByOrderFunc: func(ctx context.Context) ([]database.Project, error) {
			return []database.Project{
				{ID: 1, Name: "Homelab", Slug: "homelab", OrderKey: 1},
				{ID: 2, Name: "Blog Rebuild", Slug: "blog-rebuild", OrderKey: 2},
			}, nil
		},
		UpdateProjectOrderKeyFunc: func(ctx context.Context, arg database.UpdateProjectOrderKeyParams) error {
			updateCallCount++
			return nil
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	form := url.Values{}
	form.Set("order_key_1", "not-a-number")
	form.Set("order_key_2", "1")

	rs, err := http.PostForm(ts.URL+"/admin/order", form)
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	body, err := io.ReadAll(rs.Body)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, rs.StatusCode, http.StatusUnprocessableEntity)
	assert.Equal(t, updateCallCount, 0)
	html := string(body)
	assert.StringContains(t, html, "Must be a number")
	// The invalid submission is preserved so the admin can see and fix
	// exactly what they typed, not silently reset to the old DB value.
	assert.StringContains(t, html, `value="not-a-number"`)
}
