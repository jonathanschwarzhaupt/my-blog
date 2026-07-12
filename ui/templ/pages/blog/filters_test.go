package blog

import (
	"net/url"
	"testing"

	"github.com/jonathanschwarzhaupt/home-blog/internal/assert"
)

func TestParsePostFilters_Defaults(t *testing.T) {
	f := ParsePostFilters(url.Values{})

	assert.Equal(t, f.Page, 1)
	assert.Equal(t, f.Sort, "newest")
	assert.Equal(t, f.From, "")
	assert.Equal(t, f.To, "")
}

func TestParsePostFilters_InvalidValuesFallBackToDefaults(t *testing.T) {
	f := ParsePostFilters(url.Values{
		"page": {"not-a-number"},
		"sort": {"sideways"},
	})

	assert.Equal(t, f.Page, 1)
	assert.Equal(t, f.Sort, "newest")
}

func TestParsePostFilters_NegativePageFallsBackToOne(t *testing.T) {
	f := ParsePostFilters(url.Values{"page": {"-3"}})
	assert.Equal(t, f.Page, 1)
}

func TestSortLink_PreservesDateRangeButNotPage(t *testing.T) {
	f := PostFilters{Page: 3, Sort: "newest", From: "2020-01-01", To: "2020-12-31"}

	got := f.SortLink("oldest")

	assert.Equal(t, got, "/posts?from=2020-01-01&sort=oldest&to=2020-12-31")
}

func TestPageLink_PreservesSortAndDateRange(t *testing.T) {
	f := PostFilters{Page: 1, Sort: "oldest", From: "2020-01-01", To: ""}

	got := f.PageLink(2)

	assert.Equal(t, got, "/posts?from=2020-01-01&page=2&sort=oldest")
}

func TestPageLink_OmitsPageParamForPageOne(t *testing.T) {
	f := PostFilters{Page: 2, Sort: "newest"}

	got := f.PageLink(1)

	assert.Equal(t, got, "/posts")
}

func TestClearDateRangeLink_DropsDatesPreservesSort(t *testing.T) {
	f := PostFilters{Page: 2, Sort: "oldest", From: "2020-01-01", To: "2020-12-31"}

	got := f.ClearDateRangeLink()

	assert.Equal(t, got, "/posts?sort=oldest")
}

func TestBareDefaults_ProduceCleanURL(t *testing.T) {
	f := PostFilters{Page: 1, Sort: "newest"}

	assert.Equal(t, f.PageLink(1), "/posts")
	assert.Equal(t, f.ClearDateRangeLink(), "/posts")
}

func TestParsePostFilters_ParsesTag(t *testing.T) {
	f := ParsePostFilters(url.Values{"tag": {"go"}})
	assert.Equal(t, f.Tag, "go")
}

func TestTagLink_PreservesSortAndDateRangeResetsPage(t *testing.T) {
	f := PostFilters{Page: 3, Sort: "oldest", From: "2020-01-01"}

	got := f.TagLink("go")

	assert.Equal(t, got, "/posts?from=2020-01-01&sort=oldest&tag=go")
}

func TestTagLink_EmptyTagClearsFilter(t *testing.T) {
	f := PostFilters{Page: 1, Sort: "oldest", Tag: "go"}

	got := f.TagLink("")

	assert.Equal(t, got, "/posts?sort=oldest")
}

func TestSortLink_PreservesTag(t *testing.T) {
	f := PostFilters{Page: 1, Sort: "newest", Tag: "go"}

	got := f.SortLink("oldest")

	assert.Equal(t, got, "/posts?sort=oldest&tag=go")
}

func TestPageLink_PreservesTag(t *testing.T) {
	f := PostFilters{Page: 1, Sort: "newest", Tag: "go"}

	got := f.PageLink(2)

	assert.Equal(t, got, "/posts?page=2&tag=go")
}

func TestClearDateRangeLink_PreservesTag(t *testing.T) {
	f := PostFilters{Page: 1, Sort: "newest", From: "2020-01-01", Tag: "go"}

	got := f.ClearDateRangeLink()

	assert.Equal(t, got, "/posts?tag=go")
}

func TestTagFilterLink_URLEncodesTagsWithSpecialCharacters(t *testing.T) {
	got := TagFilterLink("Arrow Flight SQL")
	assert.Equal(t, got, "/posts?tag=Arrow+Flight+SQL")
}
