package metrics_test

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/jonathanschwarzhaupt/home-blog/internal/assert"
	"github.com/jonathanschwarzhaupt/home-blog/internal/metrics"
)

func TestGather_ParsesKnownMetrics(t *testing.T) {
	reg := prometheus.NewRegistry()

	goroutines := prometheus.NewGauge(prometheus.GaugeOpts{Name: "go_goroutines"})
	goroutines.Set(42)
	reg.MustRegister(goroutines)

	maxConns := prometheus.NewGauge(prometheus.GaugeOpts{Name: "blog_db_pool_max_conns"})
	maxConns.Set(25)
	reg.MustRegister(maxConns)

	acquired := prometheus.NewGauge(prometheus.GaugeOpts{Name: "blog_db_pool_acquired_conns"})
	acquired.Set(3)
	reg.MustRegister(acquired)

	idle := prometheus.NewGauge(prometheus.GaugeOpts{Name: "blog_db_pool_idle_conns"})
	idle.Set(2)
	reg.MustRegister(idle)

	total := prometheus.NewGauge(prometheus.GaugeOpts{Name: "blog_db_pool_total_conns"})
	total.Set(5)
	reg.MustRegister(total)

	requests := prometheus.NewCounterVec(prometheus.CounterOpts{Name: "blog_http_requests_total"}, []string{"method", "status"})
	requests.WithLabelValues("GET", "200").Add(10)
	requests.WithLabelValues("GET", "404").Add(2)
	reg.MustRegister(requests)

	startedAt := time.Now().Add(-90 * time.Minute)

	stats, err := metrics.Gather(reg, startedAt)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, stats.Goroutines, float64(42))
	assert.Equal(t, stats.DBPoolMaxConns, float64(25))
	assert.Equal(t, stats.DBPoolAcquired, float64(3))
	assert.Equal(t, stats.DBPoolIdle, float64(2))
	assert.Equal(t, stats.DBPoolTotal, float64(5))
	assert.Equal(t, stats.HTTPRequestsByStatus["GET 200"], float64(10))
	assert.Equal(t, stats.HTTPRequestsByStatus["GET 404"], float64(2))

	if stats.Uptime < 89*time.Minute || stats.Uptime > 91*time.Minute {
		t.Fatalf("got uptime %v; want ~90m", stats.Uptime)
	}

	if stats.Version == "" {
		t.Fatal("got empty version")
	}
}

func TestStats_SortedHTTPRequestsIsDeterministic(t *testing.T) {
	stats := metrics.Stats{HTTPRequestsByStatus: map[string]float64{
		"POST 200": 4,
		"GET 404":  2,
		"GET 200":  10,
	}}

	got := stats.SortedHTTPRequests()

	want := []metrics.HTTPRequestCount{
		{Label: "GET 200", Count: 10},
		{Label: "GET 404", Count: 2},
		{Label: "POST 200", Count: 4},
	}

	if len(got) != len(want) {
		t.Fatalf("got %d entries; want %d", len(got), len(want))
	}
	for i := range want {
		assert.Equal(t, got[i], want[i])
	}
}

func TestGather_MissingMetricsDefaultToZero(t *testing.T) {
	reg := prometheus.NewRegistry()

	stats, err := metrics.Gather(reg, time.Now())
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, stats.Goroutines, float64(0))
	assert.Equal(t, stats.DBPoolMaxConns, float64(0))
	assert.Equal(t, len(stats.HTTPRequestsByStatus), 0)
}
