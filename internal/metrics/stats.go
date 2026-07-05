// Package metrics reads the application's own Prometheus registry
// (cmd/blog's metrics.go/httpmetrics.go/dbpoolcollector.go, issue #21) into
// a display-friendly snapshot for the admin stats page — a "right now"
// view, not historical graphs, which stays Grafana's job.
package metrics

import (
	"sort"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"

	"github.com/jonathanschwarzhaupt/my-blog/internal/vcs"
)

// Stats is a "right now" snapshot. Nothing is withheld here; the frontend
// can de-emphasize or omit fields later without needing a second round of
// backend work.
type Stats struct {
	Version              string
	Uptime               time.Duration
	Goroutines           float64
	DBPoolMaxConns       float64
	DBPoolAcquired       float64
	DBPoolIdle           float64
	DBPoolTotal          float64
	HTTPRequestsByStatus map[string]float64 // key: "METHOD STATUS", e.g. "GET 200"
}

func Gather(registry *prometheus.Registry, startedAt time.Time) (Stats, error) {
	families, err := registry.Gather()
	if err != nil {
		return Stats{}, err
	}

	stats := Stats{
		Version:              vcs.Version(),
		Uptime:               time.Since(startedAt),
		HTTPRequestsByStatus: make(map[string]float64),
	}

	for _, mf := range families {
		switch mf.GetName() {
		case "go_goroutines":
			stats.Goroutines = firstMetricValue(mf)
		case "blog_db_pool_max_conns":
			stats.DBPoolMaxConns = firstMetricValue(mf)
		case "blog_db_pool_acquired_conns":
			stats.DBPoolAcquired = firstMetricValue(mf)
		case "blog_db_pool_idle_conns":
			stats.DBPoolIdle = firstMetricValue(mf)
		case "blog_db_pool_total_conns":
			stats.DBPoolTotal = firstMetricValue(mf)
		case "blog_http_requests_total":
			for _, m := range mf.GetMetric() {
				var method, status string
				for _, l := range m.GetLabel() {
					switch l.GetName() {
					case "method":
						method = l.GetValue()
					case "status":
						status = l.GetValue()
					}
				}
				stats.HTTPRequestsByStatus[method+" "+status] = m.GetCounter().GetValue()
			}
		}
	}

	return stats, nil
}

// HTTPRequestCount is one (method, status) label pair's request count.
type HTTPRequestCount struct {
	Label string
	Count float64
}

// SortedHTTPRequests returns HTTPRequestsByStatus as a slice ordered by
// label — map iteration order is randomized in Go, and this is display
// data, so a deterministic order matters here even though it wouldn't for
// the underlying metric itself.
func (s Stats) SortedHTTPRequests() []HTTPRequestCount {
	labels := make([]string, 0, len(s.HTTPRequestsByStatus))
	for label := range s.HTTPRequestsByStatus {
		labels = append(labels, label)
	}
	sort.Strings(labels)

	result := make([]HTTPRequestCount, len(labels))
	for i, label := range labels {
		result[i] = HTTPRequestCount{Label: label, Count: s.HTTPRequestsByStatus[label]}
	}
	return result
}

func firstMetricValue(mf *dto.MetricFamily) float64 {
	if len(mf.GetMetric()) == 0 {
		return 0
	}
	m := mf.GetMetric()[0]
	if g := m.GetGauge(); g != nil {
		return g.GetValue()
	}
	if c := m.GetCounter(); c != nil {
		return c.GetValue()
	}
	return 0
}
