package http

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestResolveClientIP(t *testing.T) {
	gateway := []netip.Prefix{netip.MustParsePrefix("172.30.42.1/32")}
	const xff = "X-Forwarded-For"

	cases := map[string]struct {
		remote  string
		headers []string // values of X-Forwarded-For, one per header line
		trusted []netip.Prefix
		want    string
	}{
		"no proxies configured ignores the header": {
			remote: "203.0.113.7:5000", headers: []string{"198.51.100.1"}, want: "203.0.113.7",
		},
		"trusted peer uses the header": {
			remote: "172.30.42.1:41000", headers: []string{"198.51.100.1"}, trusted: gateway, want: "198.51.100.1",
		},
		"trusted peer with IPv6 client": {
			remote: "172.30.42.1:41000", headers: []string{"2001:db8::7"}, trusted: gateway, want: "2001:db8::7",
		},
		"untrusted peer cannot spoof": {
			remote: "192.168.1.23:5000", headers: []string{"198.51.100.1"}, trusted: gateway, want: "192.168.1.23",
		},
		"trusted peer without header falls back": {
			remote: "172.30.42.1:41000", trusted: gateway, want: "172.30.42.1",
		},
		"comma chain is rejected": {
			remote: "172.30.42.1:41000", headers: []string{"1.2.3.4, 198.51.100.1"}, trusted: gateway, want: "172.30.42.1",
		},
		"repeated header lines are rejected": {
			remote: "172.30.42.1:41000", headers: []string{"1.2.3.4", "198.51.100.1"}, trusted: gateway, want: "172.30.42.1",
		},
		"garbage header falls back": {
			remote: "172.30.42.1:41000", headers: []string{"not-an-ip"}, trusted: gateway, want: "172.30.42.1",
		},
		"IPv4-mapped peer still matches": {
			remote: "[::ffff:172.30.42.1]:41000", headers: []string{"198.51.100.1"}, trusted: gateway, want: "198.51.100.1",
		},
		"CIDR trust": {
			remote: "10.1.2.3:41000", headers: []string{"198.51.100.1"},
			trusted: []netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")}, want: "198.51.100.1",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.RemoteAddr = tc.remote
			for _, v := range tc.headers {
				r.Header.Add(xff, v)
			}
			require.Equal(t, tc.want, resolveClientIP(r, tc.trusted, xff))
		})
	}
}

func TestClientIPMiddleware_StoresResolvedIP(t *testing.T) {
	mw := clientIPMiddleware(ClientIPConfig{
		TrustedProxies: []netip.Prefix{netip.MustParsePrefix("172.30.42.1/32")},
		Header:         "x-forwarded-for", // canonicalised by the middleware
	})
	var seen string
	h := mw(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { seen = clientIP(r) }))

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "172.30.42.1:41000"
	r.Header.Set("X-Forwarded-For", "198.51.100.1")
	h.ServeHTTP(httptest.NewRecorder(), r)
	require.Equal(t, "198.51.100.1", seen)
}

func TestClientIP_FallsBackWithoutMiddleware(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "203.0.113.7:5000"
	require.Equal(t, "203.0.113.7", clientIP(r))
}

// Behind a trusted proxy every request arrives from the same peer address; the
// rate limiter must still keep one bucket per visitor (the forwarded IP).
func TestRateLimiter_BucketsPerResolvedClientIP(t *testing.T) {
	limiter := newRateLimiter(1, time.Minute, nil)
	h := clientIPMiddleware(ClientIPConfig{
		TrustedProxies: []netip.Prefix{netip.MustParsePrefix("172.30.42.1/32")},
		Header:         "X-Forwarded-For",
	})(limiter.middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})))

	send := func(visitor string) int {
		r := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
		r.RemoteAddr = "172.30.42.1:41000" // always the proxy
		r.Header.Set("X-Forwarded-For", visitor)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, r)
		return rec.Code
	}

	require.Equal(t, http.StatusNoContent, send("198.51.100.1"))
	require.Equal(t, http.StatusTooManyRequests, send("198.51.100.1"), "same visitor is limited")
	require.Equal(t, http.StatusNoContent, send("198.51.100.2"), "another visitor has their own bucket")
}
