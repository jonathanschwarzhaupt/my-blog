package blog

import (
	"testing"
	"time"

	"github.com/jonathanschwarzhaupt/home-blog/internal/assert"
)

func TestFormatPostDate_RendersISOFormat(t *testing.T) {
	got := FormatPostDate(time.Date(2026, time.January, 22, 15, 4, 5, 0, time.UTC))
	assert.Equal(t, got, "2026-01-22")
}
