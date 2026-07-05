package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/jonathanschwarzhaupt/my-blog/internal/assert"
	"github.com/jonathanschwarzhaupt/my-blog/internal/database"
	"github.com/jonathanschwarzhaupt/my-blog/internal/database/mocks"
)

func TestManageFeatured_RendersSlotsAndOptions(t *testing.T) {
	mockDB := &mocks.MockQuerier{
		ListPostsFunc: func(ctx context.Context) ([]database.Post, error) {
			return []database.Post{{ID: 1, Title: "First Post", Slug: "first-post"}}, nil
		},
		ListProjectsFunc: func(ctx context.Context) ([]database.Project, error) {
			return []database.Project{{ID: 1, Name: "Homelab", Slug: "homelab"}}, nil
		},
		ListFeaturedPostsFunc: func(ctx context.Context) ([]database.Post, error) {
			return []database.Post{{ID: 1, Title: "First Post", Slug: "first-post", FeaturedRank: pgtype.Int4{Int32: 1, Valid: true}}}, nil
		},
		ListFeaturedProjectsFunc: func(ctx context.Context) ([]database.Project, error) {
			return nil, nil
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	rs, err := http.Get(ts.URL + "/admin/featured")
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
	assert.StringContains(t, html, "Homelab")
	assert.StringContains(t, html, `<option value="1" selected>First Post</option>`)
}

func TestManageFeaturedPost_ReplacesFeaturedSet(t *testing.T) {
	var clearedPosts, clearedProjects bool
	var setPosts []database.SetFeaturedPostParams
	var setProjects []database.SetFeaturedProjectParams

	mockDB := &mocks.MockQuerier{
		ClearFeaturedPostsFunc: func(ctx context.Context) error {
			clearedPosts = true
			return nil
		},
		SetFeaturedPostFunc: func(ctx context.Context, arg database.SetFeaturedPostParams) error {
			setPosts = append(setPosts, arg)
			return nil
		},
		ClearFeaturedProjectsFunc: func(ctx context.Context) error {
			clearedProjects = true
			return nil
		},
		SetFeaturedProjectFunc: func(ctx context.Context, arg database.SetFeaturedProjectParams) error {
			setProjects = append(setProjects, arg)
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
	form.Set("post_slot_1", "5")
	form.Set("post_slot_2", "0")
	form.Set("post_slot_3", "7")
	form.Set("project_slot_1", "0")
	form.Set("project_slot_2", "0")
	form.Set("project_slot_3", "0")

	rs, err := client.PostForm(ts.URL+"/admin/featured", form)
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	assert.Equal(t, rs.StatusCode, http.StatusSeeOther)
	assert.Equal(t, rs.Header.Get("Location"), "/admin/featured")

	assert.True(t, clearedPosts)
	assert.True(t, clearedProjects)

	if len(setPosts) != 2 {
		t.Fatalf("got %d SetFeaturedPost calls; want 2 (slot 2 was \"none\")", len(setPosts))
	}
	// Rank assignment doesn't depend on call order (the handler ranges over
	// a map, which Go deliberately randomizes), so compare as a set rather
	// than asserting a specific setPosts[0]/[1] order.
	gotRanks := map[int64]int32{setPosts[0].ID: setPosts[0].FeaturedRank.Int32, setPosts[1].ID: setPosts[1].FeaturedRank.Int32}
	assert.Equal(t, gotRanks, map[int64]int32{5: 1, 7: 3})

	if len(setProjects) != 0 {
		t.Fatalf("got %d SetFeaturedProject calls; want 0 (all slots were \"none\")", len(setProjects))
	}
}

func TestManageFeaturedPost_RejectsDuplicateSlotSelection(t *testing.T) {
	setCalled := false

	mockDB := &mocks.MockQuerier{
		ListPostsFunc: func(ctx context.Context) ([]database.Post, error) {
			return []database.Post{{ID: 5, Title: "Post Five", Slug: "post-five"}}, nil
		},
		ListProjectsFunc: func(ctx context.Context) ([]database.Project, error) {
			return nil, nil
		},
		SetFeaturedPostFunc: func(ctx context.Context, arg database.SetFeaturedPostParams) error {
			setCalled = true
			return nil
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	form := url.Values{}
	form.Set("post_slot_1", "5")
	form.Set("post_slot_2", "5")
	form.Set("post_slot_3", "0")
	form.Set("project_slot_1", "0")
	form.Set("project_slot_2", "0")
	form.Set("project_slot_3", "0")

	rs, err := http.PostForm(ts.URL+"/admin/featured", form)
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	assert.Equal(t, rs.StatusCode, http.StatusUnprocessableEntity)
	assert.False(t, setCalled)

	body, err := io.ReadAll(rs.Body)
	if err != nil {
		t.Fatal(err)
	}
	assert.StringContains(t, string(body), "more than one slot")
}
