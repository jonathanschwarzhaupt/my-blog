package main

import (
	"flag"
	"os"
	"time"
)

type options struct {
	addr           string
	dbDSN          string
	dbMaxConns     int
	dbMinConns     int
	dbMaxIdleTime  time.Duration
	displayVersion bool
}

func parseOptions() *options {
	opts := &options{}

	flag.StringVar(&opts.addr, "addr", ":4001", "HTTP network address")
	flag.StringVar(&opts.dbDSN, "db-dsn", os.Getenv("BLOG_DB_DSN"), "PostgreSQL DSN")
	flag.IntVar(&opts.dbMaxConns, "db-max-conns", 25, "PostgreSQL max open connections")
	flag.IntVar(&opts.dbMinConns, "db-min-conns", 5, "PostgreSQL min/idle connections")
	flag.DurationVar(&opts.dbMaxIdleTime, "db-max-idle-time", 15*time.Minute, "PostgreSQL max connection idle time")
	flag.BoolVar(&opts.displayVersion, "version", false, "Display version and exit")

	flag.Parse()

	return opts
}
