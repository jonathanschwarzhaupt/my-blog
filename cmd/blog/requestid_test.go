package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jonathanschwarzhaupt/my-blog/internal/assert"
)

func TestRequestID_SetsResponseHeaderAndContext(t *testing.T) {
	var idFromContext string

	handler := requestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		idFromContext = requestIDFromContext(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	headerID := rr.Header().Get("X-Request-Id")

	assert.NotEqual(t, headerID, "")
	assert.Equal(t, idFromContext, headerID)
}

func TestRequestID_UniquePerRequest(t *testing.T) {
	var seen []string

	handler := requestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, requestIDFromContext(r.Context()))
	}))

	for range 5 {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		handler.ServeHTTP(httptest.NewRecorder(), req)
	}

	unique := make(map[string]bool)
	for _, id := range seen {
		unique[id] = true
	}
	assert.Equal(t, len(unique), 5)
}

func TestRequestIDFromContext_EmptyWhenNotSet(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	assert.Equal(t, requestIDFromContext(req.Context()), "")
}
