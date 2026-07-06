package domain

import (
	"testing"
	"time"
)

func TestSession_IsActive(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	revoked := now.Add(-time.Hour)

	cases := map[string]struct {
		session Session
		want    bool
	}{
		"active": {
			session: Session{ExpiresAt: now.Add(time.Hour)},
			want:    true,
		},
		"expired": {
			session: Session{ExpiresAt: now.Add(-time.Minute)},
			want:    false,
		},
		"revoked": {
			session: Session{ExpiresAt: now.Add(time.Hour), RevokedAt: &revoked},
			want:    false,
		},
		"expired and revoked": {
			session: Session{ExpiresAt: now.Add(-time.Minute), RevokedAt: &revoked},
			want:    false,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if got := c.session.IsActive(now); got != c.want {
				t.Fatalf("IsActive(%v) = %v, want %v", now, got, c.want)
			}
		})
	}
}
