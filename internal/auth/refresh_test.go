package auth

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateRefreshToken_UniqueAndURLSafe(t *testing.T) {
	seen := make(map[string]struct{})
	for range 100 {
		tok, err := GenerateRefreshToken()
		require.NoError(t, err)
		require.NotEmpty(t, tok)
		require.NotContains(t, tok, "=", "raw url encoding is unpadded")
		require.NotContains(t, seen, tok, "tokens must be unique")
		seen[tok] = struct{}{}
	}
}

func TestHashRefreshToken_Deterministic(t *testing.T) {
	tok, err := GenerateRefreshToken()
	require.NoError(t, err)

	h1 := HashRefreshToken(tok)
	h2 := HashRefreshToken(tok)
	require.Equal(t, h1, h2, "hashing is deterministic")
	require.NotEqual(t, tok, h1, "the stored hash must differ from the raw token")
	require.Len(t, h1, 64, "hex-encoded SHA-256 is 64 chars")

	other, err := GenerateRefreshToken()
	require.NoError(t, err)
	require.NotEqual(t, h1, HashRefreshToken(other))
}
