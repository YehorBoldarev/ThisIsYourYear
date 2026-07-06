package domain

import "time"

// Session is a server-persisted refresh session. Each successful login creates
// one; it is the server-side record that makes refresh tokens revocable. The raw
// refresh token is never stored — only RefreshTokenHash (a SHA-256 hex digest of
// the opaque token handed to the client).
//
// A session is the authority on whether a refresh token is still valid: it is
// active only while it has not been revoked and has not expired.
type Session struct {
	ID               string
	UserID           string
	RefreshTokenHash string
	RememberMe       bool
	UserAgent        string
	IP               string
	ExpiresAt        time.Time
	RevokedAt        *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// IsActive reports whether the session may still be used to refresh at the given
// instant: it must be neither revoked nor past its expiry.
func (s *Session) IsActive(now time.Time) bool {
	return s.RevokedAt == nil && now.Before(s.ExpiresAt)
}
