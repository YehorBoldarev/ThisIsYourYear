package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"kibou/thisisyouryear/internal/domain"
)

// adminUsers extends the in-memory user repo with the admin-only methods.
type adminUsers struct{ *memUserRepo }

func (a adminUsers) List(_ context.Context) ([]*domain.User, error) {
	out := make([]*domain.User, 0, len(a.users))
	for _, u := range a.users {
		out = append(out, cloneUser(u))
	}
	return out, nil
}

func (a adminUsers) SetPasswordHash(_ context.Context, id, hash string) error {
	for _, u := range a.users {
		if u.ID == id {
			u.PasswordHash = hash
			return nil
		}
	}
	return domain.ErrNotFound
}

func TestAdminResetPassword(t *testing.T) {
	users := newMemUserRepo()
	sessions := newMemSessionRepo()
	auth := NewAuthService(users, sessions, fakeHasher{}, &fakeTokens{}, AuthConfig{RefreshTTLRemember: time.Hour, RefreshTTLDefault: time.Minute})
	ctx := context.Background()

	_, err := auth.Register(ctx, "a@b.com", "Alice", "password123", "")
	require.NoError(t, err)
	creds, err := auth.Login(ctx, "a@b.com", "password123", true, "ua", "1.2.3.4")
	require.NoError(t, err)

	admin := NewAdminService(adminUsers{users}, sessions, fakeHasher{}, func() (string, error) { return "temp-pass-word-1234", nil })
	pw, err := admin.ResetPassword(ctx, "A@B.com")
	require.NoError(t, err)
	require.Equal(t, "temp-pass-word-1234", pw)

	// The old password no longer works, the temporary one does.
	_, err = auth.Login(ctx, "a@b.com", "password123", false, "ua", "1.2.3.4")
	require.ErrorIs(t, err, domain.ErrInvalidCredentials)
	_, err = auth.Login(ctx, "a@b.com", pw, false, "ua", "1.2.3.4")
	require.NoError(t, err)

	// The pre-reset session was signed out.
	_, err = auth.Refresh(ctx, creds.RefreshToken, "ua", "1.2.3.4")
	require.ErrorIs(t, err, domain.ErrUnauthenticated)
}

func TestAdminResetPassword_UnknownEmail(t *testing.T) {
	admin := NewAdminService(adminUsers{newMemUserRepo()}, newMemSessionRepo(), fakeHasher{}, func() (string, error) { return "x", nil })
	_, err := admin.ResetPassword(context.Background(), "nobody@example.com")
	require.ErrorIs(t, err, domain.ErrNotFound)
}

func TestAdminListUsers(t *testing.T) {
	users := newMemUserRepo()
	auth := NewAuthService(users, newMemSessionRepo(), fakeHasher{}, &fakeTokens{}, AuthConfig{RefreshTTLRemember: time.Hour, RefreshTTLDefault: time.Minute})
	for _, e := range []string{"a@b.com", "c@d.com"} {
		_, err := auth.Register(context.Background(), e, "U", "password123", "")
		require.NoError(t, err)
	}
	admin := NewAdminService(adminUsers{users}, newMemSessionRepo(), fakeHasher{}, nil)
	list, err := admin.ListUsers(context.Background())
	require.NoError(t, err)
	require.Len(t, list, 2)
}
