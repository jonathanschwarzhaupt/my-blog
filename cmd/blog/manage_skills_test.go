package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/jonathanschwarzhaupt/home-blog/internal/assert"
	"github.com/jonathanschwarzhaupt/home-blog/internal/database"
	"github.com/jonathanschwarzhaupt/home-blog/internal/database/mocks"
	"github.com/jonathanschwarzhaupt/home-blog/ui/templ/pages/admin"
)

func TestManageSkills_RendersExistingPlusBlankRows(t *testing.T) {
	mockDB := &mocks.MockQuerier{
		ListSkillsByOrderFunc: func(ctx context.Context) ([]database.Skill, error) {
			return []database.Skill{
				{ID: 1, Category: "Languages", Name: "Go", OrderKey: 1},
			}, nil
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	rs, err := http.Get(ts.URL + "/admin/skills")
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
	assert.StringContains(t, html, `value="Languages"`)
	assert.StringContains(t, html, `value="Go"`)
	assert.StringContains(t, html, `name="`+admin.SkillRowCountFieldName+`" value="`+strconv.Itoa(1+admin.BlankSkillRows)+`"`)
	// A blank row for adding a new skill.
	assert.StringContains(t, html, `name="`+admin.SkillNameFieldName(1)+`"`)
}

func TestManageSkillsPost_ReplacesEntireSet(t *testing.T) {
	deleteAllCalled := false
	var inserted []database.InsertSkillParams

	mockDB := &mocks.MockQuerier{
		DeleteAllSkillsFunc: func(ctx context.Context) error {
			deleteAllCalled = true
			return nil
		},
		InsertSkillFunc: func(ctx context.Context, arg database.InsertSkillParams) (database.Skill, error) {
			inserted = append(inserted, arg)
			return database.Skill{ID: int64(len(inserted)), Category: arg.Category, Name: arg.Name, OrderKey: arg.OrderKey}, nil
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
	form.Set(admin.SkillRowCountFieldName, "3")
	form.Set(admin.SkillCategoryFieldName(0), "Languages")
	form.Set(admin.SkillNameFieldName(0), "Go")
	form.Set(admin.SkillOrderFieldName(0), "1")
	form.Set(admin.SkillCategoryFieldName(1), "Languages")
	form.Set(admin.SkillNameFieldName(1), "") // blank name: removed / not a real row
	form.Set(admin.SkillOrderFieldName(1), "2")
	form.Set(admin.SkillCategoryFieldName(2), "Cloud")
	form.Set(admin.SkillNameFieldName(2), "AWS")
	form.Set(admin.SkillOrderFieldName(2), "3")

	rs, err := client.PostForm(ts.URL+"/admin/skills", form)
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	assert.Equal(t, rs.StatusCode, http.StatusSeeOther)
	assert.Equal(t, rs.Header.Get("Location"), "/admin/skills")
	assert.True(t, deleteAllCalled)
	assert.Equal(t, len(inserted), 2)
	assert.Equal(t, inserted[0].Name, "Go")
	assert.Equal(t, inserted[0].OrderKey, float64(1))
	assert.Equal(t, inserted[1].Name, "AWS")
	assert.Equal(t, inserted[1].Category, "Cloud")
}

func TestManageSkillsPost_InvalidOrderReRendersForm(t *testing.T) {
	deleteAllCalled := false

	mockDB := &mocks.MockQuerier{
		DeleteAllSkillsFunc: func(ctx context.Context) error {
			deleteAllCalled = true
			return nil
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	form := url.Values{}
	form.Set(admin.SkillRowCountFieldName, "1")
	form.Set(admin.SkillCategoryFieldName(0), "Languages")
	form.Set(admin.SkillNameFieldName(0), "Go")
	form.Set(admin.SkillOrderFieldName(0), "not-a-number")

	rs, err := http.PostForm(ts.URL+"/admin/skills", form)
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	body, err := io.ReadAll(rs.Body)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, rs.StatusCode, http.StatusUnprocessableEntity)
	assert.False(t, deleteAllCalled)
	assert.StringContains(t, string(body), "Order must be a number")
}

func TestManageSkillsPost_BlankCategoryWithNameReRendersForm(t *testing.T) {
	mockDB := &mocks.MockQuerier{}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	form := url.Values{}
	form.Set(admin.SkillRowCountFieldName, "1")
	form.Set(admin.SkillCategoryFieldName(0), "")
	form.Set(admin.SkillNameFieldName(0), "Go")
	form.Set(admin.SkillOrderFieldName(0), "1")

	rs, err := http.PostForm(ts.URL+"/admin/skills", form)
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	body, err := io.ReadAll(rs.Body)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, rs.StatusCode, http.StatusUnprocessableEntity)
	assert.StringContains(t, string(body), "Category cannot be blank")
}
