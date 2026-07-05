package main

import (
	"testing"

	"github.com/jonathanschwarzhaupt/my-blog/internal/assert"
)

func TestParseFeatures(t *testing.T) {
	tests := []struct {
		name             string
		raw              string
		wantAdmin        bool
		wantUnrecognized []string
	}{
		{name: "empty", raw: "", wantAdmin: false},
		{name: "admin", raw: "admin", wantAdmin: true},
		{name: "admin with whitespace", raw: " admin ", wantAdmin: true},
		{name: "admin among others", raw: "admin,future-feature", wantAdmin: true, wantUnrecognized: []string{"future-feature"}},
		{name: "unrecognized only", raw: "admn", wantAdmin: false, wantUnrecognized: []string{"admn"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			features, unrecognized := parseFeatures(tt.raw)

			assert.Equal(t, features.Admin, tt.wantAdmin)

			if len(unrecognized) != len(tt.wantUnrecognized) {
				t.Fatalf("got unrecognized: %v; want: %v", unrecognized, tt.wantUnrecognized)
			}
			for i := range unrecognized {
				assert.Equal(t, unrecognized[i], tt.wantUnrecognized[i])
			}
		})
	}
}
