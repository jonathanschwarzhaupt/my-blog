package main

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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

// recordAttrs flattens a slog.Record's attributes into a map for easy
// assertions in tests.
func recordAttrs(r slog.Record) map[string]any {
	attrs := make(map[string]any)
	r.Attrs(func(a slog.Attr) bool {
		attrs[a.Key] = a.Value.Any()
		return true
	})
	return attrs
}

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

func TestLogRequest_LogsAfterHandlerRunsWithStatusAndDuration(t *testing.T) {
	rec := &recordingHandler{}
	app := &application{logger: slog.New(rec)}

	handler := app.logRequest(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Millisecond)
		w.WriteHeader(http.StatusTeapot)
	}))

	req := httptest.NewRequest(http.MethodGet, "/posts/hello", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if len(rec.records) != 1 {
		t.Fatalf("got %d records; want 1", len(rec.records))
	}

	attrs := recordAttrs(rec.records[0])
	assert.Equal(t, attrs["status"], any(int64(http.StatusTeapot)))

	duration, ok := attrs["duration_ms"].(int64)
	if !ok {
		t.Fatalf("duration_ms attr missing or wrong type: %v", attrs["duration_ms"])
	}
	if duration < 5 {
		t.Fatalf("got duration_ms: %d; want >= 5", duration)
	}
}

func TestLogRequest_DefaultsStatusToOKWhenWriteHeaderNeverCalled(t *testing.T) {
	rec := &recordingHandler{}
	app := &application{logger: slog.New(rec)}

	handler := app.logRequest(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("no explicit WriteHeader call"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/posts/hello", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	attrs := recordAttrs(rec.records[0])
	assert.Equal(t, attrs["status"], any(int64(http.StatusOK)))
}
