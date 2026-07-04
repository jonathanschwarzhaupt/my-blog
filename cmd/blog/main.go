package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jonathanschwarzhaupt/my-blog/internal/models"
	"github.com/jonathanschwarzhaupt/my-blog/internal/vcs"
)

type application struct {
	logger *slog.Logger
	pool   *pgxpool.Pool
}

func main() {
	addr := flag.String("addr", ":4000", "HTTP network address")
	dsn := flag.String("db-dsn", os.Getenv("BLOG_DB_DSN"), "PostgreSQL DSN")
	maxConns := flag.Int("db-max-conns", 25, "PostgreSQL max open connections")
	minConns := flag.Int("db-min-conns", 5, "PostgreSQL min/idle connections")
	maxConnIdleTime := flag.Duration("db-max-idle-time", 15*time.Minute, "PostgreSQL max connection idle time")
	displayVersion := flag.Bool("version", false, "Display version and exit")

	flag.Parse()

	if *displayVersion {
		fmt.Println(vcs.Version())
		os.Exit(0)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := models.OpenPool(ctx, *dsn, models.PoolConfig{
		MaxConns:        int32(*maxConns),
		MinConns:        int32(*minConns),
		MaxConnIdleTime: *maxConnIdleTime,
	})
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	defer pool.Close()

	app := &application{
		logger: logger,
		pool:   pool,
	}

	if err := serve(ctx, app, *addr); err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
}
