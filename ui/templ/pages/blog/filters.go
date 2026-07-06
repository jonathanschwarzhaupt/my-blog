package blog

import (
	"net/url"
	"strconv"
)

const PostsPerPage = 7

// PostFilters is deliberately lenient parsing: an invalid/missing page or
// sort value just falls back to its default rather than producing a
// validation error — this is a read-only browsing page, not a form
// submission, so there's no field to show an error against.
type PostFilters struct {
	Page int
	Sort string // "newest" (default) or "oldest"
	From string // "YYYY-MM-DD" or ""
	To   string // "YYYY-MM-DD" or ""
}

func ParsePostFilters(query url.Values) PostFilters {
	page, err := strconv.Atoi(query.Get("page"))
	if err != nil || page < 1 {
		page = 1
	}

	sort := query.Get("sort")
	if sort != "oldest" {
		sort = "newest"
	}

	return PostFilters{
		Page: page,
		Sort: sort,
		From: query.Get("from"),
		To:   query.Get("to"),
	}
}

// baseQuery carries forward the currently active sort/date-range filters
// (but not page — callers decide whether page belongs in a given link).
func (f PostFilters) baseQuery() url.Values {
	v := url.Values{}
	if f.Sort == "oldest" {
		v.Set("sort", "oldest")
	}
	if f.From != "" {
		v.Set("from", f.From)
	}
	if f.To != "" {
		v.Set("to", f.To)
	}
	return v
}

func linkFrom(v url.Values) string {
	if len(v) == 0 {
		return "/posts"
	}
	return "/posts?" + v.Encode()
}

// SortLink builds a link that switches to the given sort direction,
// preserving the active date range. Deliberately omits page — switching
// sort changes the order of every result, so "page N" from the old view
// doesn't mean anything in the new one.
func (f PostFilters) SortLink(sort string) string {
	v := url.Values{}
	if sort == "oldest" {
		v.Set("sort", "oldest")
	}
	if f.From != "" {
		v.Set("from", f.From)
	}
	if f.To != "" {
		v.Set("to", f.To)
	}
	return linkFrom(v)
}

// PageLink builds a link to a specific page, preserving sort and date range.
// Unlike SortLink, plain pagination doesn't invalidate the current filters.
func (f PostFilters) PageLink(page int) string {
	v := f.baseQuery()
	if page > 1 {
		v.Set("page", strconv.Itoa(page))
	}
	return linkFrom(v)
}

// ClearDateRangeLink preserves sort but drops from/to and resets to page 1.
func (f PostFilters) ClearDateRangeLink() string {
	v := url.Values{}
	if f.Sort == "oldest" {
		v.Set("sort", "oldest")
	}
	return linkFrom(v)
}
