package main

import (
	"regexp"
	"strings"
)

var slugNonAlnumRun = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(title string) string {
	s := strings.ToLower(title)
	s = slugNonAlnumRun.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

func splitTags(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return []string{}
	}

	parts := strings.Split(raw, ",")
	tags := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			tags = append(tags, p)
		}
	}
	return tags
}
