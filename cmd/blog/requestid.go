package main

import (
	"context"
	"math/rand/v2"
	"net/http"
	"strconv"
)

// contextKey is unexported so values set by this package can never collide
// with a context key from another package using the same string.
type contextKey string

const requestIDContextKey contextKey = "requestID"

// requestID assigns a short correlation identifier to each request — not a
// security token (math/rand/v2 is fine, no crypto/rand needed) and not
// distributed tracing (there's one process and one database, nothing to
// propagate a trace context to) — just enough to grep every log line
// produced by one specific request, including an error logged mid-handler.
func requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := strconv.FormatUint(rand.Uint64(), 36)

		w.Header().Set("X-Request-Id", id)

		ctx := context.WithValue(r.Context(), requestIDContextKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// requestIDFromContext returns "" if requestID's middleware never ran for
// this context (e.g. a test constructing a request directly).
func requestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(requestIDContextKey).(string)
	return id
}
