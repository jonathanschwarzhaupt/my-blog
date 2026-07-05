package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alexedwards/scs/pgxstore"
	"github.com/alexedwards/scs/v2"
	"github.com/go-playground/form/v4"

	"github.com/jonathanschwarzhaupt/my-blog/internal/database"
	"github.com/jonathanschwarzhaupt/my-blog/internal/models"
	"github.com/jonathanschwarzhaupt/my-blog/internal/vcs"
	"github.com/jonathanschwarzhaupt/my-blog/ui/templ/layout"
)

type application struct {
	logger         *slog.Logger
	db             database.Querier
	formDecoder    *form.Decoder
	sessionManager *scs.SessionManager
}

func main() {
	opts := parseOptions()

	if opts.displayVersion {
		fmt.Println(vcs.Version())
		os.Exit(0)
	}

	layout.Features.Admin = true

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

	sessionManager := scs.New()
	sessionManager.Store = pgxstore.New(pool)
	sessionManager.Lifetime = 12 * time.Hour

	app := &application{
		logger:         logger,
		db:             database.New(pool),
		formDecoder:    form.NewDecoder(),
		sessionManager: sessionManager,
	}

	if err := serve(ctx, app, opts.addr); err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
}
