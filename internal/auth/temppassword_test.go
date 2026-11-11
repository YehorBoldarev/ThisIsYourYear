package auth

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateTemporaryPassword(t *testing.T) {
	shape := regexp.MustCompile(`^[a-hjkmnp-z2-9]{4}(-[a-hjkmnp-z2-9]{4}){3}$`)
	seen := map[string]bool{}
	for range 50 {
		pw, err := GenerateTemporaryPassword()
		require.NoError(t, err)
		require.Regexp(t, shape, pw)
		require.False(t, seen[pw], "passwords must not repeat")
		seen[pw] = true
	}
}
