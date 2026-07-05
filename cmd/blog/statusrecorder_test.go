package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jonathanschwarzhaupt/my-blog/internal/assert"
)

func TestStatusRecorder_CapturesExplicitWriteHeader(t *testing.T) {
	rec := &statusRecorder{ResponseWriter: httptest.NewRecorder()}

	rec.WriteHeader(http.StatusTeapot)

	assert.Equal(t, rec.status, http.StatusTeapot)
}

func TestStatusRecorder_DefaultsToOKWhenWriteCalledFirst(t *testing.T) {
	rec := &statusRecorder{ResponseWriter: httptest.NewRecorder()}

	rec.Write([]byte("hello"))

	assert.Equal(t, rec.status, http.StatusOK)
}

func TestStatusRecorder_FirstWriteHeaderCallWins(t *testing.T) {
	rec := &statusRecorder{ResponseWriter: httptest.NewRecorder()}

	rec.WriteHeader(http.StatusNotFound)
	rec.WriteHeader(http.StatusInternalServerError)

	assert.Equal(t, rec.status, http.StatusNotFound)
}

func TestStatusRecorder_Unwrap(t *testing.T) {
	underlying := httptest.NewRecorder()
	rec := &statusRecorder{ResponseWriter: underlying}

	if rec.Unwrap() != underlying {
		t.Fatal("Unwrap did not return the underlying ResponseWriter")
	}
}
