package auth

import (
	"testing"

	"github.com/stretchr/testify/require"

	"kibou/thisisyouryear/internal/domain"
)

func TestPasswordHasher_HashAndCompare(t *testing.T) {
	// A low cost keeps the test fast; production cost comes from config.
	h := NewPasswordHasher(10)

	hash, err := h.Hash("correct horse battery staple")
	require.NoError(t, err)
	require.NotEmpty(t, hash)
	require.NotEqual(t, "correct horse battery staple", hash, "must not store plaintext")

	require.NoError(t, h.Compare(hash, "correct horse battery staple"))
}

func TestPasswordHasher_Compare_Mismatch(t *testing.T) {
	h := NewPasswordHasher(10)
	hash, err := h.Hash("right")
	require.NoError(t, err)

	err = h.Compare(hash, "wrong")
	require.ErrorIs(t, err, domain.ErrInvalidCredentials)
}

func TestPasswordHasher_HashesAreSalted(t *testing.T) {
	h := NewPasswordHasher(10)
	a, err := h.Hash("same")
	require.NoError(t, err)
	b, err := h.Hash("same")
	require.NoError(t, err)
	require.NotEqual(t, a, b, "bcrypt salts each hash, so equal inputs differ")
}
