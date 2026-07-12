package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jonathanschwarzhaupt/home-blog/internal/assert"
)

func TestNotFound_IsClientErrorNotFound(t *testing.T) {
	app := newTestApplication()
	rr := httptest.NewRecorder()

	app.notFound(rr)

	assert.Equal(t, rr.Code, http.StatusNotFound)
}

func TestStyleNotFound_UpgradesExplicitNotFoundCall(t *testing.T) {
	app := newTestApplication()

	handler := app.styleNotFound(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		app.notFound(w)
	}))

	req := httptest.NewRequest(http.MethodGet, "/anything", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, rr.Code, http.StatusNotFound)
	assert.StringContains(t, rr.Body.String(), "Page not found")
}

func TestStyleNotFound_UpgradesUnmatchedRoutes(t *testing.T) {
	app := newTestApplication()

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	rs, err := http.Get(ts.URL + "/this-path-does-not-exist")
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	body, err := io.ReadAll(rs.Body)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, rs.StatusCode, http.StatusNotFound)
	assert.StringContains(t, string(body), "Page not found")
	assert.StringContains(t, string(body), "Back to Home")
}

func TestStyleNotFound_LeavesOtherStatusesUntouched(t *testing.T) {
	app := newTestApplication()

	handler := app.styleNotFound(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		w.Write([]byte("I'm a teapot"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/anything", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, rr.Code, http.StatusTeapot)
	assert.Equal(t, rr.Body.String(), "I'm a teapot")
}
