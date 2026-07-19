package blog

import (
	"net/url"
	"strconv"
)

const ProjectsPerPage = 7

// ProjectFilters mirrors PostFilters (filters.go) minus Tag — Projects has
// no tags column. Same lenient-parsing rationale applies: an invalid page
// or sort value falls back to its default rather than erroring, since this
// is a read-only browsing page with no form field to report against.
type ProjectFilters struct {
	Page int
	Sort string // "curated" (default, by order_key), "newest", or "oldest" (both by created_at)
	From string // "YYYY-MM-DD" or ""
	To   string // "YYYY-MM-DD" or ""
}

func ParseProjectFilters(query url.Values) ProjectFilters {
	page, err := strconv.Atoi(query.Get("page"))
	if err != nil || page < 1 {
		page = 1
	}

	sort := query.Get("sort")
	if sort != "newest" && sort != "oldest" {
		sort = "curated"
	}

	return ProjectFilters{
		Page: page,
		Sort: sort,
		From: query.Get("from"),
		To:   query.Get("to"),
	}
}

func (f ProjectFilters) baseQuery() url.Values {
	v := url.Values{}
	if f.Sort != "curated" {
		v.Set("sort", f.Sort)
	}
	if f.From != "" {
		v.Set("from", f.From)
	}
	if f.To != "" {
		v.Set("to", f.To)
	}
	return v
}

func projectLinkFrom(v url.Values) string {
	if len(v) == 0 {
		return "/projects"
	}
	return "/projects?" + v.Encode()
}

// SortLink builds a link that switches to the given sort direction,
// preserving the active date range. Deliberately omits page, matching
// PostFilters.SortLink's reasoning — a new sort order invalidates the old
// page context.
func (f ProjectFilters) SortLink(sort string) string {
	v := url.Values{}
	if sort != "curated" {
		v.Set("sort", sort)
	}
	if f.From != "" {
		v.Set("from", f.From)
	}
	if f.To != "" {
		v.Set("to", f.To)
	}
	return projectLinkFrom(v)
}

// PageLink builds a link to a specific page, preserving sort and date range.
func (f ProjectFilters) PageLink(page int) string {
	v := f.baseQuery()
	if page > 1 {
		v.Set("page", strconv.Itoa(page))
	}
	return projectLinkFrom(v)
}

// ClearDateRangeLink preserves sort but drops from/to and resets to page 1.
func (f ProjectFilters) ClearDateRangeLink() string {
	v := url.Values{}
	if f.Sort != "curated" {
		v.Set("sort", f.Sort)
	}
	return projectLinkFrom(v)
}
