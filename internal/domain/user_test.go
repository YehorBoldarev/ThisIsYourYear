package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewUser_NormalizesEmailAndTrims(t *testing.T) {
	u, err := NewUser("  Alice@Example.COM  ", "  Alice  ", "hash")
	require.NoError(t, err)
	require.Equal(t, "alice@example.com", u.Email)
	require.Equal(t, "Alice", u.DisplayName)
	require.Equal(t, "hash", u.PasswordHash)
	require.Empty(t, u.ID, "a new user has no identity until persisted")
}

func TestNewUser_Invariants(t *testing.T) {
	cases := map[string]struct{ email, name, hash string }{
		"empty email":        {"   ", "Alice", "hash"},
		"empty display name": {"a@b.com", "  ", "hash"},
		"empty hash":         {"a@b.com", "Alice", ""},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := NewUser(c.email, c.name, c.hash)
			require.ErrorIs(t, err, ErrValidation)
		})
	}
}

func TestNormalizeEmail(t *testing.T) {
	require.Equal(t, "a@b.com", NormalizeEmail("  A@B.CoM "))
}
