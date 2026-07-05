package main

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jonathanschwarzhaupt/my-blog/internal/assert"
)

// recordingHandler is a minimal slog.Handler that captures every record
// passed to it, so tests can assert on whether logging happened at all.
type recordingHandler struct {
	records []slog.Record
}

func (h *recordingHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h *recordingHandler) Handle(_ context.Context, r slog.Record) error {
	h.records = append(h.records, r)
	return nil
}

func (h *recordingHandler) WithAttrs(attrs []slog.Attr) slog.Handler { return h }
func (h *recordingHandler) WithGroup(name string) slog.Handler       { return h }

func TestLogRequest_SkipsStaticAssets(t *testing.T) {
	rec := &recordingHandler{}
	app := &application{logger: slog.New(rec)}

	handler := app.logRequest(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/static/css/main.css", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	assert.Equal(t, len(rec.records), 0)
}

func TestLogRequest_LogsOtherRequests(t *testing.T) {
	rec := &recordingHandler{}
	app := &application{logger: slog.New(rec)}

	handler := app.logRequest(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/posts/hello", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	assert.Equal(t, len(rec.records), 1)
}
