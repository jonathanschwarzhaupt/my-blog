package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"

	"github.com/jonathanschwarzhaupt/home-blog/internal/vcs"
)

// newMetricsRegistry builds a Prometheus registry with the Go runtime and
// process collectors (goroutines, GC, memory, CPU, open file descriptors —
// all for free, no custom code needed) plus a build_info gauge exposing
// the running binary's version, the standard pattern used by most
// Prometheus-instrumented Go services. HTTP and DB-pool metrics are
// registered separately, since they need per-application state this
// constructor doesn't have.
func newMetricsRegistry() *prometheus.Registry {
	reg := prometheus.NewRegistry()

	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	buildInfo := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "blog_build_info",
		Help: "Build information about the running binary. Value is always 1; the version is in the label.",
	}, []string{"version"})
	buildInfo.WithLabelValues(vcs.Version()).Set(1)
	reg.MustRegister(buildInfo)

	return reg
}
