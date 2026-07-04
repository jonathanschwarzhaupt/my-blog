package main

import (
	"flag"
	"os"
	"time"
)

const defaultBaseURL = "http://localhost:4000"

type options struct {
	addr           string
	baseURL        string
	dbDSN          string
	dbMaxConns     int
	dbMinConns     int
	dbMaxConnLife  time.Duration
	dbMaxIdleTime  time.Duration
	limiterRPS     float64
	limiterBurst   int
	limiterEnabled bool
	displayVersion bool
}

func parseOptions() *options {
	opts := &options{}

	flag.StringVar(&opts.addr, "addr", ":4000", "HTTP network address")
	flag.StringVar(&opts.baseURL, "base-url", defaultBaseURL, "Public base URL used for absolute links (e.g. RSS)")
	flag.StringVar(&opts.dbDSN, "db-dsn", os.Getenv("BLOG_DB_DSN"), "PostgreSQL DSN")
	flag.IntVar(&opts.dbMaxConns, "db-max-conns", 25, "PostgreSQL max open connections")
	flag.IntVar(&opts.dbMinConns, "db-min-conns", 5, "PostgreSQL min/idle connections")
	flag.DurationVar(&opts.dbMaxConnLife, "db-max-conn-lifetime", time.Hour, "PostgreSQL max connection lifetime")
	flag.DurationVar(&opts.dbMaxIdleTime, "db-max-idle-time", 15*time.Minute, "PostgreSQL max connection idle time")
	flag.Float64Var(&opts.limiterRPS, "limiter-rps", 2, "Rate limiter requests-per-second per client")
	flag.IntVar(&opts.limiterBurst, "limiter-burst", 4, "Rate limiter burst size per client")
	flag.BoolVar(&opts.limiterEnabled, "limiter-enabled", true, "Enable rate limiting")
	flag.BoolVar(&opts.displayVersion, "version", false, "Display version and exit")

	flag.Parse()

	return opts
}
