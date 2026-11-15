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
	Delete(ctx context.Context, id string) error
}

// UserDataStore is one collection's view of everything a user owns: count it,
// or erase it (account deletion).
type UserDataStore interface {
	CountForUser(ctx context.Context, userID string) (int64, error)
	DeleteAllForUser(ctx context.Context, userID string) (int64, error)
}

// AdminSessionStore ends or erases a user's refresh sessions.
type AdminSessionStore interface {
	UserDataStore
	RevokeAllForUser(ctx context.Context, userID string, at time.Time) error
}

// AdminDeps are AdminService's collaborators.
type AdminDeps struct {
	Users    AdminUserRepository
	Sessions AdminSessionStore
	Topics   UserDataStore
	Goals    UserDataStore
	Hasher   PasswordHasher
	// NewPassword generates ResetPassword's temporary password
	// (auth.GenerateTemporaryPassword in production).
	NewPassword func() (string, error)
}

// AdminService implements the operator's maintenance commands.
type AdminService struct {
	d   AdminDeps
	now func() time.Time
}

// NewAdminService wires an AdminService.
func NewAdminService(d AdminDeps) *AdminService {
	return &AdminService{d: d, now: time.Now}
}

// ListUsers returns every account, oldest first.
func (s *AdminService) ListUsers(ctx context.Context) ([]*domain.User, error) {
	return s.d.Users.List(ctx)
}

// ResetPassword gives the account a new temporary password and signs it out on
// every device, returning the password so the operator can pass it on. An
// unknown email is domain.ErrNotFound.
func (s *AdminService) ResetPassword(ctx context.Context, email string) (string, error) {
	user, err := s.d.Users.ByEmail(ctx, email)
	if err != nil {
		return "", err
	}
	password, err := s.d.NewPassword()
	if err != nil {
		return "", err
	}
	hash, err := s.d.Hasher.Hash(password)
	if err != nil {
		return "", err
	}
	if err := s.d.Users.SetPasswordHash(ctx, user.ID, hash); err != nil {
		return "", err
	}
	// Old refresh sessions would otherwise keep a lost/compromised device
	// signed in; revoke them all.
	if err := s.d.Sessions.RevokeAllForUser(ctx, user.ID, s.now()); err != nil {
		return "", err
	}
	return password, nil
}

// AccountSummary is what an account holds: shown before deletion, and the
// counts actually deleted afterwards.
type AccountSummary struct {
	User     *domain.User
	Topics   int64
	Goals    int64
	Sessions int64
}

// Summary describes the account with this email, or returns domain.ErrNotFound.
func (s *AdminService) Summary(ctx context.Context, email string) (*AccountSummary, error) {
	user, err := s.d.Users.ByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	sum := &AccountSummary{User: user}
	if sum.Topics, err = s.d.Topics.CountForUser(ctx, user.ID); err != nil {
		return nil, err
	}
	if sum.Goals, err = s.d.Goals.CountForUser(ctx, user.ID); err != nil {
		return nil, err
	}
	if sum.Sessions, err = s.d.Sessions.CountForUser(ctx, user.ID); err != nil {
		return nil, err
	}
	return sum, nil
}

// DeleteUser permanently deletes the account and everything it owns, returning
// what was deleted. Children go first and the account last, so an interrupted
// run leaves no data pointing at a missing account and simply rerunning it
// finishes the job. An unknown email is domain.ErrNotFound.
func (s *AdminService) DeleteUser(ctx context.Context, email string) (*AccountSummary, error) {
	user, err := s.d.Users.ByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	deleted := &AccountSummary{User: user}
	if deleted.Goals, err = s.d.Goals.DeleteAllForUser(ctx, user.ID); err != nil {
		return nil, err
	}
	if deleted.Topics, err = s.d.Topics.DeleteAllForUser(ctx, user.ID); err != nil {
		return nil, err
	}
	if deleted.Sessions, err = s.d.Sessions.DeleteAllForUser(ctx, user.ID); err != nil {
		return nil, err
	}
	if err := s.d.Users.Delete(ctx, user.ID); err != nil {
		return nil, err
	}
	return deleted, nil
}
