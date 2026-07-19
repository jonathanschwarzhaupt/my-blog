package blog

import (
	"net/url"
	"testing"

	"github.com/jonathanschwarzhaupt/home-blog/internal/assert"
)

func TestParseProjectFilters_Defaults(t *testing.T) {
	f := ParseProjectFilters(url.Values{})

	assert.Equal(t, f.Page, 1)
	assert.Equal(t, f.Sort, "curated")
	assert.Equal(t, f.From, "")
	assert.Equal(t, f.To, "")
}

func TestParseProjectFilters_InvalidValuesFallBackToDefaults(t *testing.T) {
	f := ParseProjectFilters(url.Values{
		"page": {"not-a-number"},
		"sort": {"sideways"},
	})

	assert.Equal(t, f.Page, 1)
	assert.Equal(t, f.Sort, "curated")
}

func TestProjectSortLink_PreservesDateRangeButNotPage(t *testing.T) {
	f := ProjectFilters{Page: 3, Sort: "newest", From: "2020-01-01", To: "2020-12-31"}

	got := f.SortLink("oldest")

	assert.Equal(t, got, "/projects?from=2020-01-01&sort=oldest&to=2020-12-31")
}

func TestProjectPageLink_PreservesSortAndDateRange(t *testing.T) {
	f := ProjectFilters{Page: 1, Sort: "oldest", From: "2020-01-01"}

	got := f.PageLink(2)

	assert.Equal(t, got, "/projects?from=2020-01-01&page=2&sort=oldest")
}

func TestProjectPageLink_OmitsPageParamForPageOne(t *testing.T) {
	f := ProjectFilters{Page: 2, Sort: "curated"}

	got := f.PageLink(1)

	assert.Equal(t, got, "/projects")
}

func TestProjectClearDateRangeLink_DropsDatesPreservesSort(t *testing.T) {
	f := ProjectFilters{Page: 2, Sort: "oldest", From: "2020-01-01", To: "2020-12-31"}

	got := f.ClearDateRangeLink()

	assert.Equal(t, got, "/projects?sort=oldest")
}

func TestProjectBareDefaults_ProduceCleanURL(t *testing.T) {
	f := ProjectFilters{Page: 1, Sort: "curated"}

	assert.Equal(t, f.PageLink(1), "/projects")
	assert.Equal(t, f.ClearDateRangeLink(), "/projects")
}

func TestProjectSortLink_CuratedOmitsSortParam(t *testing.T) {
	f := ProjectFilters{Page: 1, Sort: "oldest"}

	got := f.SortLink("curated")

	assert.Equal(t, got, "/projects")
}
