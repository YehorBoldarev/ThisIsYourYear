package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"kibou/thisisyouryear/internal/domain"
)

func TestTokenIssuer_IssueAndVerify(t *testing.T) {
	iss := NewTokenIssuer("test-secret", time.Hour)

	token, exp, err := iss.Issue("user-123")
	require.NoError(t, err)
	require.NotEmpty(t, token)
	require.WithinDuration(t, time.Now().Add(time.Hour), exp, time.Minute)

	claims, err := iss.Verify(token)
	require.NoError(t, err)
	require.Equal(t, "user-123", claims.UserID)
}

func TestTokenIssuer_Verify_Expired(t *testing.T) {
	iss := NewTokenIssuer("test-secret", time.Hour)
	// Freeze issuing an hour in the past so the token is already expired now.
	iss.now = func() time.Time { return time.Now().Add(-2 * time.Hour) }
	token, _, err := iss.Issue("user-123")
	require.NoError(t, err)

	// Verify with the real clock.
	iss.now = time.Now
	_, err = iss.Verify(token)
	require.ErrorIs(t, err, domain.ErrUnauthenticated)
}

func TestTokenIssuer_Verify_WrongSecret(t *testing.T) {
	signer := NewTokenIssuer("real-secret", time.Hour)
	token, _, err := signer.Issue("user-123")
	require.NoError(t, err)

	attacker := NewTokenIssuer("other-secret", time.Hour)
	_, err = attacker.Verify(token)
	require.ErrorIs(t, err, domain.ErrUnauthenticated)
}

func TestTokenIssuer_Verify_Tampered(t *testing.T) {
	iss := NewTokenIssuer("test-secret", time.Hour)
	token, _, err := iss.Issue("user-123")
	require.NoError(t, err)

	// Flip the last character of the signature.
	tampered := token[:len(token)-1]
	if token[len(token)-1] == 'a' {
		tampered += "b"
	} else {
		tampered += "a"
	}
	_, err = iss.Verify(tampered)
	require.ErrorIs(t, err, domain.ErrUnauthenticated)
}

func TestTokenIssuer_Verify_Garbage(t *testing.T) {
	iss := NewTokenIssuer("test-secret", time.Hour)
	_, err := iss.Verify("not-a-jwt")
	require.ErrorIs(t, err, domain.ErrUnauthenticated)
}
