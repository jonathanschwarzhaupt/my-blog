package main

import (
	"context"
	"encoding/xml"
	"net/http"
	"time"

	"github.com/jonathanschwarzhaupt/home-blog/internal/models"
)

type rssFeed struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Description string    `xml:"description"`
	Items       []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
	GUID        string `xml:"guid"`
}

func (app *application) feed(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	dbPosts, err := app.db.ListPosts(ctx)
	if err != nil {
		app.serverError(w, r, models.WrapDBError(err))
		return
	}

	items := make([]rssItem, len(dbPosts))
	for i, p := range dbPosts {
		post := models.PostFromDatabase(p)
		link := app.baseURL + "/posts/" + post.Slug

		items[i] = rssItem{
			Title:       post.Title,
			Link:        link,
			Description: post.SoWhat,
			PubDate:     post.PublishedAt.Format(time.RFC1123Z),
			GUID:        link,
		}
	}

	feed := rssFeed{
		Version: "2.0",
		Channel: rssChannel{
			Title:       "Jonathan Schwarzhaupt's Blog",
			Link:        app.baseURL,
			Description: "Learning and sharing in public, daily.",
			Items:       items,
		},
	}

	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	w.Write([]byte(xml.Header))

	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if err := enc.Encode(feed); err != nil {
		app.logger.Error(err.Error(), "method", r.Method, "uri", r.URL.RequestURI())
	}
}
