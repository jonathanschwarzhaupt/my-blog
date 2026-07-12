package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/jonathanschwarzhaupt/home-blog/internal/assert"
)

func TestHTTPMetrics_RecordsRequestCountAndDuration(t *testing.T) {
	reg := newMetricsRegistry()
	metrics := newHTTPMetrics(reg)

	handler := metrics.middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))

	req := httptest.NewRequest(http.MethodGet, "/anything", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	ts := httptest.NewServer(promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	defer ts.Close()

	body := getBody(t, ts.URL)

	assert.StringContains(t, body, `blog_http_requests_total{method="GET",status="418"} 1`)
	assert.StringContains(t, body, `blog_http_request_duration_seconds_count{method="GET",status="418"} 1`)
}
