package http

import (
	"context"
	"net"
	"net/http"
	"net/netip"
	"strings"
)

// ClientIPConfig controls how the client's IP is derived when the service runs
// behind a reverse proxy (e.g. Tailscale Funnel or Cloudflare Tunnel).
//
// Forwarding headers are spoofable by anyone who can reach the server directly,
// so they are honoured only when the direct peer is one of TrustedProxies. The
// header must carry exactly one IP: the trusted proxies this is designed for
// overwrite it with the visitor's address, so a comma-separated chain means the
// header did not come from them and is ignored.
type ClientIPConfig struct {
	// TrustedProxies are the peer addresses allowed to state the client IP.
	// Empty disables header trust entirely (the client IP is the peer address).
	TrustedProxies []netip.Prefix
	// Header names the single-IP header to read (e.g. X-Forwarded-For).
	Header string
}

// clientIPMiddleware resolves the client IP once per request and stores it in
// the context, so the rate limiter, session records and access log all agree.
// It must run before those consumers.
func clientIPMiddleware(cfg ClientIPConfig) func(http.Handler) http.Handler {
	header := http.CanonicalHeaderKey(strings.TrimSpace(cfg.Header))
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := resolveClientIP(r, cfg.TrustedProxies, header)
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), clientIPKey, ip)))
		})
	}
}

// resolveClientIP returns the header's IP when the peer is a trusted proxy and
// the header holds exactly one valid IP; otherwise the peer's own address.
func resolveClientIP(r *http.Request, trusted []netip.Prefix, header string) string {
	peer, ok := peerAddr(r.RemoteAddr)
	if !ok {
		return r.RemoteAddr
	}
	if header == "" || !isTrusted(peer, trusted) {
		return peer.String()
	}
	values := r.Header.Values(header)
	if len(values) != 1 {
		return peer.String()
	}
	forwarded, err := netip.ParseAddr(strings.TrimSpace(values[0]))
	if err != nil {
		return peer.String()
	}
	return forwarded.Unmap().String()
}

// peerAddr parses RemoteAddr ("ip:port"), unmapping IPv4-in-IPv6 addresses.
func peerAddr(remoteAddr string) (netip.Addr, bool) {
	if ap, err := netip.ParseAddrPort(remoteAddr); err == nil {
		return ap.Addr().Unmap(), true
	}
	if a, err := netip.ParseAddr(remoteAddr); err == nil {
		return a.Unmap(), true
	}
	return netip.Addr{}, false
}

func isTrusted(peer netip.Addr, trusted []netip.Prefix) bool {
	for _, p := range trusted {
		if p.Contains(peer) {
			return true
		}
	}
	return false
}

// clientIP returns the client IP resolved by clientIPMiddleware. Outside that
// middleware (e.g. handler unit tests) it falls back to the peer address.
func clientIP(r *http.Request) string {
	if ip, ok := r.Context().Value(clientIPKey).(string); ok {
		return ip
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}
