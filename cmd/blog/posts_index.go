package main

import (
	"context"
	"net/http"
	"time"

	"github.com/jonathanschwarzhaupt/my-blog/ui/templ/pages/blog"
)

func (app *application) postsIndex(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	posts, ok := app.listPostsOrServerError(ctx, w, r)
	if !ok {
		return
	}

	app.render(w, r, http.StatusOK, blog.PostsIndex(posts))
}
