package main

import (
	"net/http"

	"github.com/jonathanschwarzhaupt/home-blog/ui/templ/pages/blog"
)

func (app *application) about(w http.ResponseWriter, r *http.Request) {
	app.render(w, r, http.StatusOK, blog.About())
}
