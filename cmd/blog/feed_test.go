package main

import (
	"context"
	"encoding/xml"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/jonathanschwarzhaupt/home-blog/internal/assert"
	"github.com/jonathanschwarzhaupt/home-blog/internal/database"
	"github.com/jonathanschwarzhaupt/home-blog/internal/database/mocks"
)

type testRSS struct {
	XMLName xml.Name `xml:"rss"`
	Channel struct {
		Items []struct {
			Title       string `xml:"title"`
			Link        string `xml:"link"`
			Description string `xml:"description"`
			PubDate     string `xml:"pubDate"`
			GUID        string `xml:"guid"`
		} `xml:"item"`
	} `xml:"channel"`
}

func TestFeed_WellFormedNewestFirst(t *testing.T) {
	newer := pgtype.Timestamptz{Time: time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC), Valid: true}
	older := pgtype.Timestamptz{Time: time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC), Valid: true}

	mockDB := &mocks.MockQuerier{
		ListPostsFunc: func(ctx context.Context) ([]database.Post, error) {
			return []database.Post{
				{ID: 2, Title: "Newer Post", Slug: "newer-post", SoWhat: "recent", PublishedAt: newer},
				{ID: 1, Title: "Older Post", Slug: "older-post", SoWhat: "past", PublishedAt: older},
			}, nil
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	rs, err := http.Get(ts.URL + "/feed.xml")
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	body, err := io.ReadAll(rs.Body)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, rs.StatusCode, http.StatusOK)
	assert.Equal(t, rs.Header.Get("Content-Type"), "application/rss+xml; charset=utf-8")

	var feed testRSS
	if err := xml.Unmarshal(body, &feed); err != nil {
		t.Fatalf("feed is not well-formed XML: %v", err)
	}

	assert.Equal(t, len(feed.Channel.Items), 2)
	assert.Equal(t, feed.Channel.Items[0].Title, "Newer Post")
	assert.Equal(t, feed.Channel.Items[0].Link, "http://example.com/posts/newer-post")
	assert.Equal(t, feed.Channel.Items[0].Description, "recent")
	assert.StringContains(t, feed.Channel.Items[0].PubDate, "2026")
	assert.Equal(t, feed.Channel.Items[1].Title, "Older Post")
}

func TestFeed_NoPostsStillWellFormed(t *testing.T) {
	mockDB := &mocks.MockQuerier{
		ListPostsFunc: func(ctx context.Context) ([]database.Post, error) {
			return []database.Post{}, nil
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	rs, err := http.Get(ts.URL + "/feed.xml")
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	body, err := io.ReadAll(rs.Body)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, rs.StatusCode, http.StatusOK)

	var feed testRSS
	if err := xml.Unmarshal(body, &feed); err != nil {
		t.Fatalf("feed is not well-formed XML: %v", err)
	}

	assert.Equal(t, len(feed.Channel.Items), 0)
}

func TestFeed_EscapesXMLSpecialCharacters(t *testing.T) {
	published := pgtype.Timestamptz{Time: time.Now(), Valid: true}

	mockDB := &mocks.MockQuerier{
		ListPostsFunc: func(ctx context.Context) ([]database.Post, error) {
			return []database.Post{
				{
					ID:          1,
					Title:       `Go & Postgres: <tips> "quoted"`,
					Slug:        "go-and-postgres",
					SoWhat:      "A & B < C",
					PublishedAt: published,
				},
			}, nil
		},
	}

	app := newTestApplicationWithDB(mockDB)

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	rs, err := http.Get(ts.URL + "/feed.xml")
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	body, err := io.ReadAll(rs.Body)
	if err != nil {
		t.Fatal(err)
	}

	var feed testRSS
	if err := xml.Unmarshal(body, &feed); err != nil {
		t.Fatalf("feed is not well-formed XML: %v", err)
	}

	assert.Equal(t, len(feed.Channel.Items), 1)
	assert.Equal(t, feed.Channel.Items[0].Title, `Go & Postgres: <tips> "quoted"`)
	assert.Equal(t, feed.Channel.Items[0].Description, "A & B < C")
}
