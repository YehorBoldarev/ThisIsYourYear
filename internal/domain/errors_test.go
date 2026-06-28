package domain

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClassifiers(t *testing.T) {
	// Wrapping must preserve identity through errors.Is.
	wrappedNotFound := fmt.Errorf("loading user: %w", ErrNotFound)
	wrappedConflict := fmt.Errorf("inserting user: %w", ErrConflict)
	wrappedValidation := fmt.Errorf("bad id: %w", ErrValidation)

	require.True(t, IsNotFound(wrappedNotFound))
	require.True(t, IsConflict(wrappedConflict))
	require.True(t, IsValidation(wrappedValidation))

	// Classifiers must not cross-match.
	require.False(t, IsConflict(wrappedNotFound))
	require.False(t, IsNotFound(wrappedConflict))
	require.False(t, IsValidation(wrappedNotFound))

	// Unrelated errors classify as none.
	other := fmt.Errorf("network down")
	require.False(t, IsNotFound(other))
	require.False(t, IsConflict(other))
	require.False(t, IsValidation(other))
}
