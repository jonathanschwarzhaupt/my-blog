package models_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jonathanschwarzhaupt/home-blog/internal/assert"
	"github.com/jonathanschwarzhaupt/home-blog/internal/database"
)

// seedPosts inserts one post per entry in dates (each with a unique,
// test-scoped slug) and registers cleanup to delete them afterward. Returns
// the inserted posts' ids in the same order as dates.
func seedPosts(t *testing.T, pool *pgxpool.Pool, dates []time.Time) []int64 {
	t.Helper()

	q := database.New(pool)
	ids := make([]int64, len(dates))
	for i, d := range dates {
		slug := fmt.Sprintf("pagination-test-%d-%d", time.Now().UnixNano(), i)
		post, err := q.InsertPost(t.Context(), database.InsertPostParams{
			Title:       slug,
			Slug:        slug,
			Body:        "body",
			SoWhat:      "it matters",
			Tags:        []string{},
			PublishedAt: pgtype.Timestamptz{Time: d, Valid: true},
		})
		if err != nil {
			t.Fatal(err)
		}
		ids[i] = post.ID
	}

	t.Cleanup(func() {
		pool.Exec(context.Background(), "DELETE FROM posts WHERE id = ANY($1)", ids)
	})

	return ids
}

func TestListPostsFiltered_NoFiltersReturnsAllOrderedNewestFirst(t *testing.T) {
	pool := realTestPool(t)
	q := database.New(pool)

	base := time.Date(2015, time.January, 1, 0, 0, 0, 0, time.UTC)
	ids := seedPosts(t, pool, []time.Time{base, base.AddDate(0, 0, 1), base.AddDate(0, 0, 2)})

	rows, err := q.ListPostsFiltered(t.Context(), database.ListPostsFilteredParams{
		SortOldest: false,
		PageLimit:  100,
		PageOffset: 0,
	})
	if err != nil {
		t.Fatal(err)
	}

	var gotIDs []int64
	for _, r := range rows {
		if r.ID == ids[0] || r.ID == ids[1] || r.ID == ids[2] {
			gotIDs = append(gotIDs, r.ID)
		}
	}

	assert.Equal(t, len(gotIDs), 3)
	assert.Equal(t, gotIDs[0], ids[2]) // newest first
	assert.Equal(t, gotIDs[1], ids[1])
	assert.Equal(t, gotIDs[2], ids[0])
}

func TestListPostsFiltered_SortOldestReversesOrder(t *testing.T) {
	pool := realTestPool(t)
	q := database.New(pool)

	base := time.Date(2015, time.February, 1, 0, 0, 0, 0, time.UTC)
	ids := seedPosts(t, pool, []time.Time{base, base.AddDate(0, 0, 1), base.AddDate(0, 0, 2)})

	rows, err := q.ListPostsFiltered(t.Context(), database.ListPostsFilteredParams{
		SortOldest: true,
		PageLimit:  100,
		PageOffset: 0,
	})
	if err != nil {
		t.Fatal(err)
	}

	var gotIDs []int64
	for _, r := range rows {
		if r.ID == ids[0] || r.ID == ids[1] || r.ID == ids[2] {
			gotIDs = append(gotIDs, r.ID)
		}
	}

	assert.Equal(t, len(gotIDs), 3)
	assert.Equal(t, gotIDs[0], ids[0]) // oldest first
	assert.Equal(t, gotIDs[1], ids[1])
	assert.Equal(t, gotIDs[2], ids[2])
}

func TestListPostsFiltered_DateRangeNarrowsResults(t *testing.T) {
	pool := realTestPool(t)
	q := database.New(pool)

	base := time.Date(2015, time.March, 1, 0, 0, 0, 0, time.UTC)
	ids := seedPosts(t, pool, []time.Time{base, base.AddDate(0, 0, 5), base.AddDate(0, 0, 10)})

	from := base.AddDate(0, 0, 3)
	to := base.AddDate(0, 0, 7)

	rows, err := q.ListPostsFiltered(t.Context(), database.ListPostsFilteredParams{
		FromDate:   pgtype.Timestamptz{Time: from, Valid: true},
		ToDate:     pgtype.Timestamptz{Time: to, Valid: true},
		SortOldest: true,
		PageLimit:  100,
		PageOffset: 0,
	})
	if err != nil {
		t.Fatal(err)
	}

	var gotIDs []int64
	for _, r := range rows {
		if r.ID == ids[0] || r.ID == ids[1] || r.ID == ids[2] {
			gotIDs = append(gotIDs, r.ID)
		}
	}

	assert.Equal(t, len(gotIDs), 1)
	assert.Equal(t, gotIDs[0], ids[1])
}

func TestListPostsFiltered_PaginationBoundaries(t *testing.T) {
	pool := realTestPool(t)
	q := database.New(pool)

	base := time.Date(2015, time.April, 1, 0, 0, 0, 0, time.UTC)
	dates := make([]time.Time, 5)
	for i := range dates {
		dates[i] = base.AddDate(0, 0, i)
	}
	ids := seedPosts(t, pool, dates)

	// Page 1 of 2-per-page, oldest first, scoped to just this test's own
	// date range so unrelated rows in the table don't affect the count.
	from := base
	to := base.AddDate(0, 0, 4)

	page1, err := q.ListPostsFiltered(t.Context(), database.ListPostsFilteredParams{
		FromDate:   pgtype.Timestamptz{Time: from, Valid: true},
		ToDate:     pgtype.Timestamptz{Time: to, Valid: true},
		SortOldest: true,
		PageLimit:  2,
		PageOffset: 0,
	})
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, len(page1), 2)
	assert.Equal(t, page1[0].ID, ids[0])
	assert.Equal(t, page1[1].ID, ids[1])
	assert.Equal(t, page1[0].TotalCount, int64(5))

	// Last page (offset 4, only 1 remaining row)
	lastPage, err := q.ListPostsFiltered(t.Context(), database.ListPostsFilteredParams{
		FromDate:   pgtype.Timestamptz{Time: from, Valid: true},
		ToDate:     pgtype.Timestamptz{Time: to, Valid: true},
		SortOldest: true,
		PageLimit:  2,
		PageOffset: 4,
	})
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, len(lastPage), 1)
	assert.Equal(t, lastPage[0].ID, ids[4])

	// Past the end — empty, not an error
	pastEnd, err := q.ListPostsFiltered(t.Context(), database.ListPostsFilteredParams{
		FromDate:   pgtype.Timestamptz{Time: from, Valid: true},
		ToDate:     pgtype.Timestamptz{Time: to, Valid: true},
		SortOldest: true,
		PageLimit:  2,
		PageOffset: 6,
	})
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, len(pastEnd), 0)
}
