package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/jonathanschwarzhaupt/home-blog/internal/assert"
)

func getBody(t *testing.T, url string) string {
	t.Helper()

	rs, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	b, err := io.ReadAll(rs.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestNewMetricsRegistry_ExposesGoAndProcessAndBuildInfo(t *testing.T) {
	reg := newMetricsRegistry()

	ts := httptest.NewServer(promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	defer ts.Close()

	body := getBody(t, ts.URL)

	assert.StringContains(t, body, "go_goroutines")
	assert.StringContains(t, body, "process_cpu_seconds_total")
	assert.StringContains(t, body, "blog_build_info")
}
