package models_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jonathanschwarzhaupt/my-blog/internal/assert"
	"github.com/jonathanschwarzhaupt/my-blog/internal/database"
)

// seedProjects inserts one project per entry in dates (each with a unique,
// test-scoped slug) and registers cleanup. Returns the inserted projects'
// ids in the same order as dates.
func seedProjects(t *testing.T, pool *pgxpool.Pool, dates []time.Time) []int64 {
	t.Helper()

	q := database.New(pool)
	ids := make([]int64, len(dates))
	for i, d := range dates {
		slug := fmt.Sprintf("projects-pagination-test-%d-%d", time.Now().UnixNano(), i)
		project, err := q.InsertProject(t.Context(), database.InsertProjectParams{
			Name:      slug,
			Slug:      slug,
			CreatedAt: pgtype.Timestamptz{Time: d, Valid: true},
		})
		if err != nil {
			t.Fatal(err)
		}
		ids[i] = project.ID
	}

	t.Cleanup(func() {
		pool.Exec(context.Background(), "DELETE FROM projects WHERE id = ANY($1)", ids)
	})

	return ids
}

func TestListProjectsFiltered_NoFiltersReturnsAllOrderedNewestFirst(t *testing.T) {
	pool := realTestPool(t)
	q := database.New(pool)

	base := time.Date(2016, time.January, 1, 0, 0, 0, 0, time.UTC)
	ids := seedProjects(t, pool, []time.Time{base, base.AddDate(0, 0, 1), base.AddDate(0, 0, 2)})

	rows, err := q.ListProjectsFiltered(t.Context(), database.ListProjectsFilteredParams{
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

func TestListProjectsFiltered_SortOldestReversesOrder(t *testing.T) {
	pool := realTestPool(t)
	q := database.New(pool)

	base := time.Date(2016, time.February, 1, 0, 0, 0, 0, time.UTC)
	ids := seedProjects(t, pool, []time.Time{base, base.AddDate(0, 0, 1), base.AddDate(0, 0, 2)})

	rows, err := q.ListProjectsFiltered(t.Context(), database.ListProjectsFilteredParams{
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

func TestListProjectsFiltered_DateRangeNarrowsResults(t *testing.T) {
	pool := realTestPool(t)
	q := database.New(pool)

	base := time.Date(2016, time.March, 1, 0, 0, 0, 0, time.UTC)
	ids := seedProjects(t, pool, []time.Time{base, base.AddDate(0, 0, 5), base.AddDate(0, 0, 10)})

	from := base.AddDate(0, 0, 3)
	to := base.AddDate(0, 0, 7)

	rows, err := q.ListProjectsFiltered(t.Context(), database.ListProjectsFilteredParams{
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

func TestListProjectsFiltered_PaginationBoundaries(t *testing.T) {
	pool := realTestPool(t)
	q := database.New(pool)

	base := time.Date(2016, time.April, 1, 0, 0, 0, 0, time.UTC)
	dates := make([]time.Time, 5)
	for i := range dates {
		dates[i] = base.AddDate(0, 0, i)
	}
	ids := seedProjects(t, pool, dates)

	from := base
	to := base.AddDate(0, 0, 4)

	page1, err := q.ListProjectsFiltered(t.Context(), database.ListProjectsFilteredParams{
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

	lastPage, err := q.ListProjectsFiltered(t.Context(), database.ListProjectsFilteredParams{
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

	pastEnd, err := q.ListProjectsFiltered(t.Context(), database.ListProjectsFilteredParams{
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
