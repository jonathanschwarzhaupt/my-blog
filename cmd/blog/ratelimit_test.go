package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jonathanschwarzhaupt/home-blog/internal/assert"
)

func newTestHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func TestRateLimiter_BlocksOverLimit(t *testing.T) {
	rl := newRateLimiter(1, 1, true) // 1 req/s, burst of 1

	handler := rl.middleware(newTestHandler())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Real-IP", "1.2.3.4")

	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req)
	assert.Equal(t, rec1.Code, http.StatusOK)

	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req)
	assert.Equal(t, rec2.Code, http.StatusTooManyRequests)
}

func TestRateLimiter_IndependentPerIP(t *testing.T) {
	rl := newRateLimiter(1, 1, true)

	handler := rl.middleware(newTestHandler())

	reqA := httptest.NewRequest(http.MethodGet, "/", nil)
	reqA.Header.Set("X-Real-IP", "1.1.1.1")

	reqB := httptest.NewRequest(http.MethodGet, "/", nil)
	reqB.Header.Set("X-Real-IP", "2.2.2.2")

	// Exhaust IP A's burst.
	recA1 := httptest.NewRecorder()
	handler.ServeHTTP(recA1, reqA)
	assert.Equal(t, recA1.Code, http.StatusOK)

	recA2 := httptest.NewRecorder()
	handler.ServeHTTP(recA2, reqA)
	assert.Equal(t, recA2.Code, http.StatusTooManyRequests)

	// IP B is unaffected.
	recB1 := httptest.NewRecorder()
	handler.ServeHTTP(recB1, reqB)
	assert.Equal(t, recB1.Code, http.StatusOK)
}

func TestRateLimiter_DisabledBypassesLimit(t *testing.T) {
	rl := newRateLimiter(1, 1, false) // disabled

	handler := rl.middleware(newTestHandler())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Real-IP", "1.2.3.4")

	for i := 0; i < 5; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, rec.Code, http.StatusOK)
	}
}

func TestRateLimiter_EvictsStaleClients(t *testing.T) {
	rl := newRateLimiter(1, 1, true)

	handler := rl.middleware(newTestHandler())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Real-IP", "9.9.9.9")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, rec.Code, http.StatusOK)

	rl.mu.Lock()
	_, ok := rl.clients["9.9.9.9"]
	rl.mu.Unlock()
	assert.True(t, ok)

	// Backdate the client's lastSeen so it looks stale, then evict.
	rl.mu.Lock()
	rl.clients["9.9.9.9"].lastSeen = time.Now().Add(-10 * time.Minute)
	rl.mu.Unlock()

	rl.evictStale(3 * time.Minute)

	rl.mu.Lock()
	_, ok = rl.clients["9.9.9.9"]
	rl.mu.Unlock()
	assert.False(t, ok)
}

func TestRateLimiter_TokensRefillOverTime(t *testing.T) {
	rl := newRateLimiter(10, 1, true) // 10 req/s, burst of 1 -> refills every 100ms

	handler := rl.middleware(newTestHandler())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Real-IP", "5.5.5.5")

	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req)
	assert.Equal(t, rec1.Code, http.StatusOK)

	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req)
	assert.Equal(t, rec2.Code, http.StatusTooManyRequests)

	time.Sleep(150 * time.Millisecond)

	rec3 := httptest.NewRecorder()
	handler.ServeHTTP(rec3, req)
	assert.Equal(t, rec3.Code, http.StatusOK)
}

func TestRealIP_PrefersCfConnectingIP(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Cf-Connecting-Ip", "203.0.113.9")
	req.Header.Set("X-Real-IP", "198.51.100.1")
	req.RemoteAddr = "127.0.0.1:12345"

	assert.Equal(t, realIP(req), "203.0.113.9")
}

func TestRealIP_FallsBackToXForwardedFor(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.9, 10.0.0.1")
	req.RemoteAddr = "127.0.0.1:12345"

	assert.Equal(t, realIP(req), "203.0.113.9")
}

func TestRealIP_FallsBackToRemoteAddr(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "198.51.100.1:54321"

	assert.Equal(t, realIP(req), "198.51.100.1")
}

func TestRealIP_RemoteAddrWithoutPort(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "not-a-valid-host-port"

	assert.Equal(t, realIP(req), "not-a-valid-host-port")
}
