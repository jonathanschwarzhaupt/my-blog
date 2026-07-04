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

	"github.com/jonathanschwarzhaupt/my-blog/internal/database"
	"github.com/jonathanschwarzhaupt/my-blog/internal/models"
	"github.com/jonathanschwarzhaupt/my-blog/internal/vcs"
)

type application struct {
	logger  *slog.Logger
	db      database.Querier
	limiter *rateLimiter
	baseURL string
}

func main() {
	opts := parseOptions()

	if opts.displayVersion {
		fmt.Println(vcs.Version())
		os.Exit(0)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

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

	limiter := newRateLimiter(opts.limiterRPS, opts.limiterBurst, opts.limiterEnabled)
	limiter.startCleanup(time.Minute, 3*time.Minute)

	baseURL := opts.baseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	if baseURL == defaultBaseURL {
		logger.Warn("base-url is unset or still the default; RSS feed links will be unusable in production", "base-url", baseURL)
	}

	app := &application{
		logger:  logger,
		db:      database.New(pool),
		limiter: limiter,
		baseURL: strings.TrimSuffix(baseURL, "/"),
	}

	if err := serve(ctx, app, opts.addr); err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
}
