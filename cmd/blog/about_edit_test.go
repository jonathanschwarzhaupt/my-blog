package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/jonathanschwarzhaupt/home-blog/internal/assert"
	"github.com/jonathanschwarzhaupt/home-blog/internal/database"
	"github.com/jonathanschwarzhaupt/home-blog/internal/database/mocks"
)

func TestAboutEdit_LoadsLatestRevision(t *testing.T) {
	mockDB := &mocks.MockQuerier{
		GetLatestAboutRevisionFunc: func(ctx context.Context) (database.AboutRevision, error) {
			return database.AboutRevision{ID: 3, Body: "Current about text."}, nil
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	rs, err := http.Get(ts.URL + "/admin/about/edit")
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	body, err := io.ReadAll(rs.Body)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, rs.StatusCode, http.StatusOK)
	assert.StringContains(t, string(body), "Current about text.")
}

func TestAboutUpdate_Valid(t *testing.T) {
	var gotBody string
	insertCallCount := 0

	mockDB := &mocks.MockQuerier{
		InsertAboutRevisionFunc: func(ctx context.Context, body string) (database.AboutRevision, error) {
			insertCallCount++
			gotBody = body
			return database.AboutRevision{ID: 4, Body: body}, nil
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
	form.Set("body", "Updated about text.")

	rs, err := client.PostForm(ts.URL+"/admin/about/edit", form)
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	assert.Equal(t, rs.StatusCode, http.StatusSeeOther)
	assert.Equal(t, rs.Header.Get("Location"), "/admin/about/edit")
	assert.Equal(t, insertCallCount, 1)
	assert.Equal(t, gotBody, "Updated about text.")
}

func TestAboutUpdate_BlankBodyReRendersForm(t *testing.T) {
	insertCallCount := 0

	mockDB := &mocks.MockQuerier{
		InsertAboutRevisionFunc: func(ctx context.Context, body string) (database.AboutRevision, error) {
			insertCallCount++
			return database.AboutRevision{}, nil
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	form := url.Values{}
	form.Set("body", "")

	rs, err := http.PostForm(ts.URL+"/admin/about/edit", form)
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
	assert.StringContains(t, string(body), "This field cannot be blank")
}
