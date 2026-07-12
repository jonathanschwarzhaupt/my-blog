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

// seedPostsWithTags inserts one post per entry in tagSets (each with a
// unique, test-scoped slug and the given tags) and registers cleanup to
// delete them afterward. Returns the inserted posts' ids in the same order
// as tagSets.
func seedPostsWithTags(t *testing.T, pool *pgxpool.Pool, tagSets [][]string) []int64 {
	t.Helper()

	q := database.New(pool)
	ids := make([]int64, len(tagSets))
	for i, tags := range tagSets {
		slug := fmt.Sprintf("tags-test-%d-%d", time.Now().UnixNano(), i)
		post, err := q.InsertPost(t.Context(), database.InsertPostParams{
			Title:       slug,
			Slug:        slug,
			Body:        "body",
			SoWhat:      "it matters",
			Tags:        tags,
			PublishedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
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

func TestListPostsFiltered_TagNarrowsResults(t *testing.T) {
	pool := realTestPool(t)
	q := database.New(pool)

	uniqueTag := fmt.Sprintf("test-tag-%d", time.Now().UnixNano())
	ids := seedPostsWithTags(t, pool, [][]string{
		{uniqueTag, "other"},
		{"unrelated"},
		{uniqueTag},
	})

	rows, err := q.ListPostsFiltered(t.Context(), database.ListPostsFilteredParams{
		Tag:        pgtype.Text{String: uniqueTag, Valid: true},
		PageLimit:  100,
		PageOffset: 0,
	})
	if err != nil {
		t.Fatal(err)
	}

	var gotIDs []int64
	for _, r := range rows {
		gotIDs = append(gotIDs, r.ID)
	}

	assert.Equal(t, len(gotIDs), 2)
	assert.Equal(t, gotIDs[0] == ids[0] || gotIDs[0] == ids[2], true)
	assert.Equal(t, gotIDs[1] == ids[0] || gotIDs[1] == ids[2], true)
}

func TestListDistinctTags_CollapsesDuplicatesAndOrdersAlphabetically(t *testing.T) {
	pool := realTestPool(t)
	q := database.New(pool)

	prefix := fmt.Sprintf("zzz-test-%d", time.Now().UnixNano())
	tagA := prefix + "-alpha"
	tagB := prefix + "-beta"

	seedPostsWithTags(t, pool, [][]string{
		{tagB, tagA},
		{tagA},
	})

	tags, err := q.ListDistinctTags(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	var gotOurs []string
	for _, tag := range tags {
		if tag == tagA || tag == tagB {
			gotOurs = append(gotOurs, tag)
		}
	}

	assert.Equal(t, len(gotOurs), 2)
	assert.Equal(t, gotOurs[0], tagA) // alphabetically first
	assert.Equal(t, gotOurs[1], tagB)
}
