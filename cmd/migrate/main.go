package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/jonathanschwarzhaupt/home-blog/internal/vcs"
	"github.com/jonathanschwarzhaupt/home-blog/sql/schema"
)

func main() {
	dsn := flag.String("db-dsn", os.Getenv("BLOG_DB_DSN"), "PostgreSQL DSN")
	displayVersion := flag.Bool("version", false, "Display version and exit")
	flag.Parse()

	if *displayVersion {
		fmt.Println(vcs.Version())
		os.Exit(0)
	}

	args := flag.Args()
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: migrate -db-dsn=<dsn> <up|down|status|...> [args...]")
		os.Exit(1)
	}
	command := args[0]

	db, err := sql.Open("pgx", *dsn)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.PingContext(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	goose.SetBaseFS(schema.FS)

	if err := goose.SetDialect("postgres"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := goose.RunWithOptionsContext(context.Background(), command, db, ".", args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
