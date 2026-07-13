package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRateLimiter_Allow(t *testing.T) {
	now := time.Date(2026, time.July, 10, 12, 0, 0, 0, time.UTC)
	clock := func() time.Time { return now }
	rl := newRateLimiter(2, time.Minute, clock)

	// First two requests from a key are allowed.
	ok, _ := rl.allow("1.2.3.4")
	require.True(t, ok)
	ok, _ = rl.allow("1.2.3.4")
	require.True(t, ok)

	// The third within the window is rejected, with a positive Retry-After.
	ok, retryAfter := rl.allow("1.2.3.4")
	require.False(t, ok)
	require.Positive(t, retryAfter)
	require.LessOrEqual(t, retryAfter, time.Minute)

	// A different key has its own independent budget.
	ok, _ = rl.allow("5.6.7.8")
	require.True(t, ok)

	// After the window elapses, the original key is allowed again.
	now = now.Add(time.Minute + time.Second)
	ok, _ = rl.allow("1.2.3.4")
	require.True(t, ok)
}

func TestRateLimiter_SweepEvictsExpired(t *testing.T) {
	now := time.Date(2026, time.July, 10, 12, 0, 0, 0, time.UTC)
	clock := func() time.Time { return now }
	rl := newRateLimiter(1, time.Minute, clock)

	rl.allow("1.2.3.4")
	require.Len(t, rl.windows, 1)

	// Advance past the window and the next sweep interval; the next call sweeps
	// the stale entry (and adds the caller's fresh one).
	now = now.Add(2 * time.Minute)
	rl.allow("5.6.7.8")
	require.Len(t, rl.windows, 1)
	_, tracked := rl.windows["1.2.3.4"]
	require.False(t, tracked, "expired window should have been evicted")
}

func TestRateLimiter_Middleware(t *testing.T) {
	now := time.Date(2026, time.July, 10, 12, 0, 0, 0, time.UTC)
	rl := newRateLimiter(1, time.Minute, func() time.Time { return now })

	handler := rl.middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	call := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
		req.RemoteAddr = "203.0.113.7:44444"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}

	// First request passes through.
	require.Equal(t, http.StatusOK, call().Code)

	// Second is throttled: 429, Retry-After, and the standard error envelope.
	rec := call()
	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	require.NotEmpty(t, rec.Header().Get("Retry-After"))
	require.Contains(t, rec.Header().Get("Content-Type"), "application/json")

	var body errorBody
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "rate_limited", body.Error.Code)
}
