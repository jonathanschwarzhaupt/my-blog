package main

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/jonathanschwarzhaupt/home-blog/internal/assert"
)

func TestServe_GracefulShutdown(t *testing.T) {
	app := newTestApplication()
	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- serve(ctx, app, "127.0.0.1:0")
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-errCh:
		assert.Nil(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("serve did not shut down within the expected time")
	}
}

func TestServeMetrics_GracefulShutdown(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- serveMetrics(ctx, logger, "127.0.0.1:0", http.NotFoundHandler())
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-errCh:
		assert.Nil(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("serveMetrics did not shut down within the expected time")
	}
}
