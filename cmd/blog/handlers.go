package main

import (
	"encoding/json"
	"net/http"

	"github.com/jonathanschwarzhaupt/home-blog/internal/vcs"
	"github.com/jonathanschwarzhaupt/home-blog/ui/templ/layout"
)

type healthcheckResponse struct {
	Status  string `json:"status"`
	Mode    string `json:"mode"`
	Version string `json:"version"`
}

// healthcheck reports mode (admin/public) rather than a dev/staging/prod
// environment — this app has no such concept (single-environment homelab
// deployment, see ADR-0003); mode is the axis that actually varies between
// the two feature-gated instances a probe might be hitting.
func (app *application) healthcheck(w http.ResponseWriter, r *http.Request) {
	mode := "public"
	if layout.Features.Admin {
		mode = "admin"
	}

	data := healthcheckResponse{
		Status:  "available",
		Mode:    mode,
		Version: vcs.Version(),
	}

	js, err := json.Marshal(data)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}
