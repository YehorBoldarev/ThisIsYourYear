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

func (a adminUsers) Delete(_ context.Context, id string) error {
	if _, ok := a.users[id]; !ok {
		return domain.ErrNotFound
	}
	delete(a.users, id)
	return nil
}

// adminSessions adds the count/erase methods to the in-memory session repo.
type adminSessions struct{ *memSessionRepo }

func (a adminSessions) CountForUser(_ context.Context, userID string) (int64, error) {
	var n int64
	for _, s := range a.sessions {
		if s.UserID == userID {
			n++
		}
	}
	return n, nil
}

func (a adminSessions) DeleteAllForUser(_ context.Context, userID string) (int64, error) {
	var n int64
	for id, s := range a.sessions {
		if s.UserID == userID {
			delete(a.sessions, id)
			n++
		}
	}
	return n, nil
}

// countStore is a UserDataStore holding per-user document counts, recording
// the order in which collections are erased.
type countStore struct {
	name   string
	counts map[string]int64
	log    *[]string
}

func (c countStore) CountForUser(_ context.Context, userID string) (int64, error) {
	return c.counts[userID], nil
}

func (c countStore) DeleteAllForUser(_ context.Context, userID string) (int64, error) {
	*c.log = append(*c.log, c.name)
	n := c.counts[userID]
	delete(c.counts, userID)
	return n, nil
}

func adminDeps(users *memUserRepo, sessions *memSessionRepo, newPassword func() (string, error)) AdminDeps {
	var log []string
	return AdminDeps{
		Users:       adminUsers{users},
		Sessions:    adminSessions{sessions},
		Topics:      countStore{name: "topics", counts: map[string]int64{}, log: &log},
		Goals:       countStore{name: "goals", counts: map[string]int64{}, log: &log},
		Hasher:      fakeHasher{},
		NewPassword: newPassword,
	}
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

	admin := NewAdminService(adminDeps(users, sessions, func() (string, error) { return "temp-pass-word-1234", nil }))
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
	admin := NewAdminService(adminDeps(newMemUserRepo(), newMemSessionRepo(), func() (string, error) { return "x", nil }))
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
	admin := NewAdminService(adminDeps(users, newMemSessionRepo(), nil))
	list, err := admin.ListUsers(context.Background())
	require.NoError(t, err)
	require.Len(t, list, 2)
}

func TestAdminSummaryAndDeleteUser(t *testing.T) {
	users := newMemUserRepo()
	sessions := newMemSessionRepo()
	auth := NewAuthService(users, sessions, fakeHasher{}, &fakeTokens{}, AuthConfig{RefreshTTLRemember: time.Hour, RefreshTTLDefault: time.Minute})
	ctx := context.Background()

	alice, err := auth.Register(ctx, "a@b.com", "Alice", "password123", "")
	require.NoError(t, err)
	bob, err := auth.Register(ctx, "bob@b.com", "Bob", "password123", "")
	require.NoError(t, err)
	_, err = auth.Login(ctx, "a@b.com", "password123", true, "ua", "1.2.3.4")
	require.NoError(t, err)
	_, err = auth.Login(ctx, "bob@b.com", "password123", true, "ua", "1.2.3.4")
	require.NoError(t, err)

	deps := adminDeps(users, sessions, nil)
	deps.Topics.(countStore).counts[alice.ID] = 3
	deps.Goals.(countStore).counts[alice.ID] = 17
	deps.Topics.(countStore).counts[bob.ID] = 1
	admin := NewAdminService(deps)

	sum, err := admin.Summary(ctx, "A@B.com")
	require.NoError(t, err)
	require.Equal(t, "a@b.com", sum.User.Email)
	require.Equal(t, int64(3), sum.Topics)
	require.Equal(t, int64(17), sum.Goals)
	require.Equal(t, int64(1), sum.Sessions)

	deleted, err := admin.DeleteUser(ctx, "a@b.com")
	require.NoError(t, err)
	require.Equal(t, *sum, *deleted, "deletes exactly what the summary showed")
	require.Equal(t, []string{"goals", "topics"}, *deps.Topics.(countStore).log,
		"children first: goals before their topics")

	_, err = users.ByEmail(ctx, "a@b.com")
	require.ErrorIs(t, err, domain.ErrNotFound, "the account is gone")
	_, err = auth.Login(ctx, "a@b.com", "password123", false, "ua", "1.2.3.4")
	require.ErrorIs(t, err, domain.ErrInvalidCredentials)

	// Other accounts are untouched.
	other, err := admin.Summary(ctx, "bob@b.com")
	require.NoError(t, err)
	require.Equal(t, int64(1), other.Topics)
	require.Equal(t, int64(1), other.Sessions)

	_, err = admin.DeleteUser(ctx, "a@b.com")
	require.ErrorIs(t, err, domain.ErrNotFound, "deleting twice is a clean not-found")
}
