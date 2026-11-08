package http

import (
	"net/http"
	"time"
)

// RefreshCookieConfig describes how the refresh-token cookie is written. The
// cookie is always HttpOnly and SameSite=Lax; Secure and Domain come from config.
// Secure defaults to on (COOKIE_SECURE=true) and is turned off only when the app
// is served over plain HTTP (local development, a LAN-only deployment), where a
// Secure cookie would never be stored.
type RefreshCookieConfig struct {
	Name   string
	Path   string
	Domain string
	Secure bool
}

// set writes the refresh-token cookie, expiring in step with the session.
func (c RefreshCookieConfig) set(w http.ResponseWriter, rawToken string, expiresAt time.Time) {
	//nolint:gosec // G124: Secure is config-driven (see RefreshCookieConfig); HttpOnly and SameSite are fixed.
	http.SetCookie(w, &http.Cookie{
		Name:     c.Name,
		Value:    rawToken,
		Path:     c.Path,
		Domain:   c.Domain,
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   c.Secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// clear removes the refresh-token cookie (used on logout and failed refresh).
func (c RefreshCookieConfig) clear(w http.ResponseWriter) {
	//nolint:gosec // G124: Secure is config-driven (see RefreshCookieConfig); HttpOnly and SameSite are fixed.
	http.SetCookie(w, &http.Cookie{
		Name:     c.Name,
		Value:    "",
		Path:     c.Path,
		Domain:   c.Domain,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   c.Secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// read returns the raw refresh token from the request cookie, or "" if absent.
func (c RefreshCookieConfig) read(r *http.Request) string {
	ck, err := r.Cookie(c.Name)
	if err != nil {
		return ""
	}
	return ck.Value
}
