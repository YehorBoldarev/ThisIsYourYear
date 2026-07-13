package http

import (
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// rateLimiter is a per-key fixed-window request limiter. It is intended for the
// public auth endpoints, keyed by client IP, to blunt brute-force and
// credential-stuffing without needing an external store.
//
// A fixed window (N requests per window) is deliberately simple: it is cheap,
// has predictable memory, and is more than adequate for "basic" abuse
// protection. Stale windows are swept lazily (at most once per window) so the
// map cannot grow without bound and no background goroutine is needed.
type rateLimiter struct {
	mu        sync.Mutex
	windows   map[string]*ipWindow
	limit     int
	window    time.Duration
	now       func() time.Time
	nextSweep time.Time
}

// ipWindow tracks one key's request count within the current window.
type ipWindow struct {
	count   int
	resetAt time.Time
}

// newRateLimiter builds a limiter allowing limit requests per window per key.
// now may be nil, in which case time.Now is used; injecting it keeps the
// limiter deterministic in tests.
func newRateLimiter(limit int, window time.Duration, now func() time.Time) *rateLimiter {
	if now == nil {
		now = time.Now
	}
	return &rateLimiter{
		windows: make(map[string]*ipWindow),
		limit:   limit,
		window:  window,
		now:     now,
	}
}

// allow records a request for key and reports whether it is within the limit.
// When the limit is exceeded it also returns how long until the window resets,
// so callers can advertise a Retry-After.
func (rl *rateLimiter) allow(key string) (ok bool, retryAfter time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := rl.now()
	rl.sweep(now)

	w, exists := rl.windows[key]
	if !exists || !now.Before(w.resetAt) {
		// First request, or the previous window has elapsed: start a new one.
		rl.windows[key] = &ipWindow{count: 1, resetAt: now.Add(rl.window)}
		return true, 0
	}
	if w.count >= rl.limit {
		return false, w.resetAt.Sub(now)
	}
	w.count++
	return true, 0
}

// sweep drops expired windows. It runs at most once per window duration so its
// cost is amortized away under load. The caller must hold rl.mu.
func (rl *rateLimiter) sweep(now time.Time) {
	if now.Before(rl.nextSweep) {
		return
	}
	for key, w := range rl.windows {
		if !now.Before(w.resetAt) {
			delete(rl.windows, key)
		}
	}
	rl.nextSweep = now.Add(rl.window)
}

// middleware throttles requests by client IP, returning 429 with the standard
// error envelope once a key exceeds its limit.
func (rl *rateLimiter) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ok, retryAfter := rl.allow(clientIP(r))
		if !ok {
			writeRateLimited(w, retryAfter)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// writeRateLimited emits the shared error envelope for a throttled request,
// including a Retry-After header (whole seconds, at least 1).
func writeRateLimited(w http.ResponseWriter, retryAfter time.Duration) {
	seconds := max(int(math.Ceil(retryAfter.Seconds())), 1)
	w.Header().Set("Retry-After", strconv.Itoa(seconds))
	writeJSON(w, http.StatusTooManyRequests, errorBody{Error: errorDetail{
		Code:    "rate_limited",
		Message: "too many requests",
	}})
}
