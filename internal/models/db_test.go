package models

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jonathanschwarzhaupt/my-blog/internal/assert"
)

func TestOpenPool_BadDSN(t *testing.T) {
	cfg := PoolConfig{MaxConns: 5, MinConns: 1, MaxConnLifetime: time.Hour, MaxConnIdleTime: time.Minute}

	_, err := OpenPool(context.Background(), "not-a-valid-dsn", cfg)

	assert.NotNil(t, err)
}

func TestOpenPool_MinConnsExceedsMaxConns(t *testing.T) {
	cfg := PoolConfig{MaxConns: 5, MinConns: 10, MaxConnIdleTime: time.Minute}

	_, err := OpenPool(context.Background(), "postgres://irrelevant", cfg)

	assert.NotNil(t, err)
}

func TestOpenPool_RealDatabase(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dsn := os.Getenv("BLOG_DB_DSN")
	if dsn == "" {
		t.Skip("BLOG_DB_DSN not set")
	}

	cfg := PoolConfig{MaxConns: 5, MinConns: 1, MaxConnLifetime: time.Hour, MaxConnIdleTime: time.Minute}

	pool, err := OpenPool(context.Background(), dsn, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	assert.Nil(t, pool.Ping(ctx))
}
