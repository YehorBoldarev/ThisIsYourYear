package service

import (
	"context"
	"time"

	"kibou/thisisyouryear/internal/domain"
)

// AdminUserRepository is the persistence contract for operator commands. Unlike
// the per-user ports it is not scoped to one user: it is reachable only from the
// `admin` CLI on the host, never from an HTTP route.
type AdminUserRepository interface {
	ByEmail(ctx context.Context, email string) (*domain.User, error)
	List(ctx context.Context) ([]*domain.User, error)
	SetPasswordHash(ctx context.Context, id, passwordHash string) error
}

// SessionRevoker ends every session a user has (e.g. after a password reset).
type SessionRevoker interface {
	RevokeAllForUser(ctx context.Context, userID string, at time.Time) error
}

// AdminService implements the operator's maintenance commands.
type AdminService struct {
	users       AdminUserRepository
	sessions    SessionRevoker
	hasher      PasswordHasher
	newPassword func() (string, error)
	now         func() time.Time
}

// NewAdminService wires an AdminService. newPassword generates the temporary
// password for ResetPassword (auth.GenerateTemporaryPassword in production).
func NewAdminService(users AdminUserRepository, sessions SessionRevoker, hasher PasswordHasher, newPassword func() (string, error)) *AdminService {
	return &AdminService{users: users, sessions: sessions, hasher: hasher, newPassword: newPassword, now: time.Now}
}

// ListUsers returns every account, oldest first.
func (s *AdminService) ListUsers(ctx context.Context) ([]*domain.User, error) {
	return s.users.List(ctx)
}

// ResetPassword gives the account a new temporary password and signs it out on
// every device, returning the password so the operator can pass it on. An
// unknown email is domain.ErrNotFound.
func (s *AdminService) ResetPassword(ctx context.Context, email string) (string, error) {
	user, err := s.users.ByEmail(ctx, email)
	if err != nil {
		return "", err
	}
	password, err := s.newPassword()
	if err != nil {
		return "", err
	}
	hash, err := s.hasher.Hash(password)
	if err != nil {
		return "", err
	}
	if err := s.users.SetPasswordHash(ctx, user.ID, hash); err != nil {
		return "", err
	}
	// Old refresh sessions would otherwise keep a lost/compromised device
	// signed in; revoke them all.
	if err := s.sessions.RevokeAllForUser(ctx, user.ID, s.now()); err != nil {
		return "", err
	}
	return password, nil
}
