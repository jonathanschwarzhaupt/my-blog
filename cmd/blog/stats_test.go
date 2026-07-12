package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/jonathanschwarzhaupt/home-blog/internal/assert"
)

func TestStats_RendersKnownMetrics(t *testing.T) {
	app := newTestApplication()

	registry := prometheus.NewRegistry()
	goroutines := prometheus.NewGauge(prometheus.GaugeOpts{Name: "go_goroutines"})
	goroutines.Set(7)
	registry.MustRegister(goroutines)
	app.metricsRegistry = registry

	ts := httptest.NewServer(app.routes())
	defer ts.Close()

	rs, err := http.Get(ts.URL + "/admin/stats")
	if err != nil {
		t.Fatal(err)
	}
	defer rs.Body.Close()

	body, err := io.ReadAll(rs.Body)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, rs.StatusCode, http.StatusOK)

	html := string(body)
	assert.StringContains(t, html, "Goroutines")
	assert.StringContains(t, html, "7")
}
