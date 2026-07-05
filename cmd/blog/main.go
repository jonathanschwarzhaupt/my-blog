package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/alexedwards/scs/pgxstore"
	"github.com/alexedwards/scs/v2"
	"github.com/go-playground/form/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/jonathanschwarzhaupt/my-blog/internal/database"
	"github.com/jonathanschwarzhaupt/my-blog/internal/models"
	"github.com/jonathanschwarzhaupt/my-blog/internal/vcs"
	"github.com/jonathanschwarzhaupt/my-blog/ui/templ/layout"
)

type application struct {
	logger  *slog.Logger
	db      database.Querier
	baseURL string

	// metrics, metricsRegistry, and startedAt are always constructed,
	// regardless of mode — operational instrumentation applies to both
	// deployments equally and is orthogonal to the admin/public
	// route-gating layout.Features controls. metricsRegistry backs both
	// the /metrics endpoint (main.go) and the admin stats page
	// (stats.go's gatherStats).
	metrics         *httpMetrics
	metricsRegistry *prometheus.Registry
	startedAt       time.Time

	// limiter is only constructed when the admin feature is disabled — the
	// admin deployment is Tailscale-only, so rate limiting adds no real
	// security benefit there, only a chance of throttling legitimate use.
	limiter *rateLimiter

	// formDecoder and sessionManager are only constructed when the admin
	// feature is enabled; routes() only dereferences them inside the
	// admin-gated branch, so nil is safe otherwise.
	formDecoder    *form.Decoder
	sessionManager *scs.SessionManager
}

func main() {
	opts := parseOptions()

	if opts.displayVersion {
		fmt.Println(vcs.Version())
		os.Exit(0)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	features, unrecognized := parseFeatures(opts.features)
	layout.Features = features
	if len(unrecognized) > 0 {
		logger.Warn("ignoring unrecognized -features entries", "unrecognized", unrecognized)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := models.OpenPool(ctx, opts.dbDSN, models.PoolConfig{
		MaxConns:        int32(opts.dbMaxConns),
		MinConns:        int32(opts.dbMinConns),
		MaxConnLifetime: opts.dbMaxConnLife,
		MaxConnIdleTime: opts.dbMaxIdleTime,
	})
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	defer pool.Close()

	baseURL := opts.baseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	if baseURL == defaultBaseURL {
		logger.Warn("base-url is unset or still the default; RSS feed links will be unusable in production", "base-url", baseURL)
	}

	metricsRegistry := newMetricsRegistry()
	metricsRegistry.MustRegister(newDBPoolCollector(pool))
	httpMetrics := newHTTPMetrics(metricsRegistry)

	app := &application{
		logger:          logger,
		db:              database.New(pool),
		baseURL:         strings.TrimSuffix(baseURL, "/"),
		metrics:         httpMetrics,
		metricsRegistry: metricsRegistry,
		startedAt:       time.Now(),
	}

	if layout.Features.Admin {
		sessionManager := scs.New()
		sessionManager.Store = pgxstore.New(pool)
		sessionManager.Lifetime = 12 * time.Hour

		app.formDecoder = form.NewDecoder()
		app.sessionManager = sessionManager
	} else {
		limiter := newRateLimiter(opts.limiterRPS, opts.limiterBurst, opts.limiterEnabled)
		limiter.startCleanup(time.Minute, 3*time.Minute)
		app.limiter = limiter
	}

	// Never route opts.metricsAddr through a public ingress/tunnel — access
	// control here is enforced by network topology (only reachable inside
	// the cluster, scraped in-cluster by Prometheus), not application-level
	// auth. See CODING_STANDARDS.md's Metrics section.
	metricsDone := make(chan error, 1)
	go func() {
		metricsDone <- serveMetrics(ctx, logger, opts.metricsAddr, promhttp.HandlerFor(metricsRegistry, promhttp.HandlerOpts{}))
	}()

	serveErr := serve(ctx, app, opts.addr)

	if err := <-metricsDone; err != nil {
		logger.Error(err.Error())
	}

	if serveErr != nil {
		logger.Error(serveErr.Error())
		os.Exit(1)
	}
}
