package main

import (
	"strings"

	"github.com/jonathanschwarzhaupt/my-blog/ui/templ/layout"
)

// parseFeatures turns a comma-separated -features flag value (e.g.
// "admin,maybe-another-feature") into layout.Features. Unknown names are
// ignored rather than rejected — a Helm values array containing a feature
// this binary version doesn't recognize yet shouldn't fail startup outright,
// mirroring Kubernetes' own tolerance for unknown --feature-gates entries —
// but they're returned alongside the parsed flags so main() can log a
// warning, since a typo here (e.g. "admn") would otherwise silently deploy
// an admin instance with no admin routes and no signal anything is wrong.
func parseFeatures(raw string) (features layout.FeatureFlags, unrecognized []string) {
	for _, name := range strings.Split(raw, ",") {
		name = strings.TrimSpace(name)
		switch name {
		case "":
			// no-op: the default empty -features value splits to [""]
		case "admin":
			features.Admin = true
		default:
			unrecognized = append(unrecognized, name)
		}
	}
	return features, unrecognized
}
