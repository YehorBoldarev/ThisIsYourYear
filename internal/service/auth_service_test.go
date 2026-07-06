package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"kibou/thisisyouryear/internal/domain"
)

// --- in-memory fakes (no database, no crypto cost) ---

type memUserRepo struct {
	users map[string]*domain.User // keyed by id
	seq   int
}

func newMemUserRepo() *memUserRepo { return &memUserRepo{users: map[string]*domain.User{}} }

func (m *memUserRepo) Create(_ context.Context, u *domain.User) error {
	for _, e := range m.users {
		if e.Email == u.Email {
			return fmt.Errorf("duplicate email: %w", domain.ErrConflict)
		}
	}
	m.seq++
	u.ID = fmt.Sprintf("user-%d", m.seq)
	m.users[u.ID] = cloneUser(u)
	return nil
}

func (m *memUserRepo) ByEmail(_ context.Context, email string) (*domain.User, error) {
	norm := domain.NormalizeEmail(email)
	for _, u := range m.users {
		if u.Email == norm {
			return cloneUser(u), nil
		}
	}
	return nil, domain.ErrNotFound
}

func (m *memUserRepo) ByID(_ context.Context, id string) (*domain.User, error) {
	if u, ok := m.users[id]; ok {
		return cloneUser(u), nil
	}
	return nil, domain.ErrNotFound
}

func cloneUser(u *domain.User) *domain.User { c := *u; return &c }

type memSessionRepo struct {
	sessions map[string]*domain.Session // keyed by id
	byHash   map[string]string          // token hash -> session id
	seq      int
}

func newMemSessionRepo() *memSessionRepo {
	return &memSessionRepo{sessions: map[string]*domain.Session{}, byHash: map[string]string{}}
}

func (m *memSessionRepo) Create(_ context.Context, s *domain.Session) error {
	m.seq++
	s.ID = fmt.Sprintf("sess-%d", m.seq)
	m.sessions[s.ID] = cloneSession(s)
	m.byHash[s.RefreshTokenHash] = s.ID
	return nil
}

func (m *memSessionRepo) ByTokenHash(_ context.Context, hash string) (*domain.Session, error) {
	id, ok := m.byHash[hash]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return cloneSession(m.sessions[id]), nil
}

func (m *memSessionRepo) Revoke(_ context.Context, id string, at time.Time) error {
	s, ok := m.sessions[id]
	if !ok {
		return domain.ErrNotFound
	}
	when := at
	s.RevokedAt = &when
	return nil
}

func (m *memSessionRepo) RevokeAllForUser(_ context.Context, userID string, at time.Time) error {
	for _, s := range m.sessions {
		if s.UserID == userID && s.RevokedAt == nil {
			when := at
			s.RevokedAt = &when
		}
	}
	return nil
}

func cloneSession(s *domain.Session) *domain.Session { c := *s; return &c }

type fakeHasher struct{}

func (fakeHasher) Hash(plain string) (string, error) { return "hashed:" + plain, nil }
func (fakeHasher) Compare(hash, plain string) error {
	if hash == "hashed:"+plain {
		return nil
	}
	return domain.ErrInvalidCredentials
}

type fakeTokens struct{ seq int }

func (f *fakeTokens) Issue(userID string) (string, time.Time, error) {
	f.seq++
	return fmt.Sprintf("access-%s-%d", userID, f.seq), time.Now().Add(time.Minute), nil
}

func newTestService() (*AuthService, *memSessionRepo) {
	users := newMemUserRepo()
	sessions := newMemSessionRepo()
	svc := NewAuthService(users, sessions, fakeHasher{}, &fakeTokens{}, AuthConfig{
		RefreshTTLRemember: 744 * time.Hour,
		RefreshTTLDefault:  24 * time.Hour,
	})
	return svc, sessions
}

// --- tests ---

func TestRegister_Success(t *testing.T) {
	svc, _ := newTestService()
	user, err := svc.Register(context.Background(), "Alice@Example.com", "Alice", "password123")
	require.NoError(t, err)
	require.NotEmpty(t, user.ID)
	require.Equal(t, "alice@example.com", user.Email)
	require.Equal(t, "hashed:password123", user.PasswordHash)
}

func TestRegister_DuplicateEmail(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()
	_, err := svc.Register(ctx, "a@b.com", "Alice", "password123")
	require.NoError(t, err)

	_, err = svc.Register(ctx, "A@B.com", "Alice2", "password456")
	require.ErrorIs(t, err, domain.ErrConflict)
}

func TestLogin_Success(t *testing.T) {
	svc, sessions := newTestService()
	ctx := context.Background()
	_, err := svc.Register(ctx, "a@b.com", "Alice", "password123")
	require.NoError(t, err)

	creds, err := svc.Login(ctx, "a@b.com", "password123", true, "ua", "ip")
	require.NoError(t, err)
	require.NotEmpty(t, creds.AccessToken)
	require.NotEmpty(t, creds.RefreshToken)
	require.Equal(t, "a@b.com", creds.User.Email)
	require.Len(t, sessions.sessions, 1, "a session is persisted")
	require.WithinDuration(t, time.Now().Add(744*time.Hour), creds.RefreshExpiresAt, time.Minute,
		"rememberMe uses the long TTL")
}

func TestLogin_DefaultTTLWithoutRemember(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()
	_, err := svc.Register(ctx, "a@b.com", "Alice", "password123")
	require.NoError(t, err)

	creds, err := svc.Login(ctx, "a@b.com", "password123", false, "ua", "ip")
	require.NoError(t, err)
	require.WithinDuration(t, time.Now().Add(24*time.Hour), creds.RefreshExpiresAt, time.Minute)
}

func TestLogin_WrongPassword(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()
	_, err := svc.Register(ctx, "a@b.com", "Alice", "password123")
	require.NoError(t, err)

	_, err = svc.Login(ctx, "a@b.com", "wrong", false, "ua", "ip")
	require.ErrorIs(t, err, domain.ErrInvalidCredentials)
}

func TestLogin_UnknownEmail(t *testing.T) {
	svc, _ := newTestService()
	_, err := svc.Login(context.Background(), "nobody@b.com", "whatever", false, "ua", "ip")
	require.ErrorIs(t, err, domain.ErrInvalidCredentials,
		"unknown email must be indistinguishable from a wrong password")
}

func TestRefresh_RotatesToken(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()
	_, err := svc.Register(ctx, "a@b.com", "Alice", "password123")
	require.NoError(t, err)
	first, err := svc.Login(ctx, "a@b.com", "password123", true, "ua", "ip")
	require.NoError(t, err)

	second, err := svc.Refresh(ctx, first.RefreshToken, "ua", "ip")
	require.NoError(t, err)
	require.NotEqual(t, first.RefreshToken, second.RefreshToken, "the refresh token is rotated")
	require.NotEmpty(t, second.AccessToken)
	require.Equal(t, first.User.ID, second.User.ID)

	// The new token works.
	third, err := svc.Refresh(ctx, second.RefreshToken, "ua", "ip")
	require.NoError(t, err)
	require.NotEmpty(t, third.AccessToken)
}

func TestRefresh_ReuseRevokesLineage(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()
	_, err := svc.Register(ctx, "a@b.com", "Alice", "password123")
	require.NoError(t, err)
	first, err := svc.Login(ctx, "a@b.com", "password123", true, "ua", "ip")
	require.NoError(t, err)

	second, err := svc.Refresh(ctx, first.RefreshToken, "ua", "ip")
	require.NoError(t, err)

	// Reusing the rotated-away first token is treated as theft.
	_, err = svc.Refresh(ctx, first.RefreshToken, "ua", "ip")
	require.ErrorIs(t, err, domain.ErrUnauthenticated)

	// The whole lineage is now revoked: even the previously-valid second token fails.
	_, err = svc.Refresh(ctx, second.RefreshToken, "ua", "ip")
	require.ErrorIs(t, err, domain.ErrUnauthenticated)
}

func TestRefresh_Empty(t *testing.T) {
	svc, _ := newTestService()
	_, err := svc.Refresh(context.Background(), "", "ua", "ip")
	require.ErrorIs(t, err, domain.ErrUnauthenticated)
}

func TestLogout_RevokesSession(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()
	_, err := svc.Register(ctx, "a@b.com", "Alice", "password123")
	require.NoError(t, err)
	creds, err := svc.Login(ctx, "a@b.com", "password123", true, "ua", "ip")
	require.NoError(t, err)

	require.NoError(t, svc.Logout(ctx, creds.RefreshToken))

	_, err = svc.Refresh(ctx, creds.RefreshToken, "ua", "ip")
	require.ErrorIs(t, err, domain.ErrUnauthenticated, "a logged-out token cannot refresh")
}

func TestLogout_UnknownTokenIsNoop(t *testing.T) {
	svc, _ := newTestService()
	require.NoError(t, svc.Logout(context.Background(), "does-not-exist"))
	require.NoError(t, svc.Logout(context.Background(), ""))
}
