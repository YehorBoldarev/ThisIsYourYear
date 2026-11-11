// Package service holds the application's use-cases. Services orchestrate domain
// entities and depend only on interfaces they declare here, so business logic is
// testable without a database or HTTP server and the concrete adapters are
// injected at wiring time (dependency inversion).
package service

import (
	"context"
	"crypto/subtle"
	"strings"
	"time"

	"kibou/thisisyouryear/internal/auth"
	"kibou/thisisyouryear/internal/domain"
)

// UserRepository is the persistence contract AuthService needs for users.
type UserRepository interface {
	Create(ctx context.Context, u *domain.User) error
	ByEmail(ctx context.Context, email string) (*domain.User, error)
	ByID(ctx context.Context, id string) (*domain.User, error)
}

// SessionRepository is the persistence contract for refresh sessions.
type SessionRepository interface {
	Create(ctx context.Context, s *domain.Session) error
	ByTokenHash(ctx context.Context, hash string) (*domain.Session, error)
	Revoke(ctx context.Context, id string, at time.Time) error
	RevokeAllForUser(ctx context.Context, userID string, at time.Time) error
}

// PasswordHasher hashes and verifies passwords.
type PasswordHasher interface {
	Hash(plain string) (string, error)
	Compare(hash, plain string) error
}

// AccessTokenIssuer mints short-lived access tokens.
type AccessTokenIssuer interface {
	Issue(userID string) (token string, expiresAt time.Time, err error)
}

// AuthConfig carries the refresh-session lifetimes and the sign-up policy.
type AuthConfig struct {
	RefreshTTLRemember time.Duration
	RefreshTTLDefault  time.Duration
	// InviteCode, when non-empty, makes registration invite-only: Register
	// accepts only this code. Empty means open registration.
	InviteCode string
}

// Credentials is the result of a successful login or refresh: the access token
// for the API plus the raw refresh token the transport layer stores in a cookie.
// The raw refresh token exists only here and in the response; only its hash is
// persisted.
type Credentials struct {
	User             *domain.User
	AccessToken      string
	AccessExpiresAt  time.Time
	RefreshToken     string
	RefreshExpiresAt time.Time
}

// AuthService implements registration, login, refresh, and logout.
type AuthService struct {
	users    UserRepository
	sessions SessionRepository
	hasher   PasswordHasher
	tokens   AccessTokenIssuer
	cfg      AuthConfig
	now      func() time.Time
}

// NewAuthService wires an AuthService from its collaborators.
func NewAuthService(users UserRepository, sessions SessionRepository, hasher PasswordHasher, tokens AccessTokenIssuer, cfg AuthConfig) *AuthService {
	return &AuthService{
		users:    users,
		sessions: sessions,
		hasher:   hasher,
		tokens:   tokens,
		cfg:      cfg,
		now:      time.Now,
	}
}

// InviteRequired reports whether registration needs an invite code.
func (s *AuthService) InviteRequired() bool { return s.cfg.InviteCode != "" }

// Register creates a new account with a bcrypt-hashed password. When sign-up is
// invite-only, a missing or wrong inviteCode is a *domain.FieldError on
// "inviteCode" (checked first, before any hashing). A duplicate email surfaces
// as domain.ErrConflict.
func (s *AuthService) Register(ctx context.Context, email, displayName, password, inviteCode string) (*domain.User, error) {
	if err := s.checkInvite(inviteCode); err != nil {
		return nil, err
	}
	hash, err := s.hasher.Hash(password)
	if err != nil {
		return nil, err
	}
	user, err := domain.NewUser(email, displayName, hash)
	if err != nil {
		return nil, err
	}
	if err := s.users.Create(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

// checkInvite enforces invite-only registration. The comparison is constant
// time so response timing reveals nothing about the code; guessing is further
// throttled by the per-IP auth rate limit.
func (s *AuthService) checkInvite(inviteCode string) error {
	if !s.InviteRequired() {
		return nil
	}
	provided := strings.TrimSpace(inviteCode)
	if provided == "" {
		return &domain.FieldError{Field: "inviteCode", Message: "is required"}
	}
	if subtle.ConstantTimeCompare([]byte(provided), []byte(s.cfg.InviteCode)) != 1 {
		return &domain.FieldError{Field: "inviteCode", Message: "is not valid"}
	}
	return nil
}

// Login authenticates a user and issues a fresh session. Unknown email and wrong
// password are both reported as domain.ErrInvalidCredentials so the two cases are
// indistinguishable to a client (no user enumeration).
func (s *AuthService) Login(ctx context.Context, email, password string, rememberMe bool, userAgent, ip string) (*Credentials, error) {
	user, err := s.users.ByEmail(ctx, email)
	if err != nil {
		if domain.IsNotFound(err) {
			return nil, domain.ErrInvalidCredentials
		}
		return nil, err
	}
	if err := s.hasher.Compare(user.PasswordHash, password); err != nil {
		return nil, err
	}
	return s.issueSession(ctx, user, rememberMe, userAgent, ip)
}

// Refresh validates a refresh token and rotates it: the presented session is
// revoked and a new one issued. A token that maps to an already-revoked session
// is treated as reuse of a rotated-away token — a possible theft — so the whole
// lineage of the user's sessions is revoked and the request is rejected.
func (s *AuthService) Refresh(ctx context.Context, rawRefresh, userAgent, ip string) (*Credentials, error) {
	if rawRefresh == "" {
		return nil, domain.ErrUnauthenticated
	}

	session, err := s.sessions.ByTokenHash(ctx, auth.HashRefreshToken(rawRefresh))
	if err != nil {
		if domain.IsNotFound(err) {
			return nil, domain.ErrUnauthenticated
		}
		return nil, err
	}

	now := s.now()
	if !session.IsActive(now) {
		// A revoked (already-rotated) token presented again indicates reuse:
		// revoke every active session for the user as a precaution. An expired
		// token is simply rejected (the TTL index reaps it).
		if session.RevokedAt != nil {
			_ = s.sessions.RevokeAllForUser(ctx, session.UserID, now)
		}
		return nil, domain.ErrUnauthenticated
	}

	if err := s.sessions.Revoke(ctx, session.ID, now); err != nil {
		return nil, err
	}

	user, err := s.users.ByID(ctx, session.UserID)
	if err != nil {
		if domain.IsNotFound(err) {
			return nil, domain.ErrUnauthenticated
		}
		return nil, err
	}
	return s.issueSession(ctx, user, session.RememberMe, userAgent, ip)
}

// Logout revokes the session behind the given refresh token. It is idempotent:
// an empty or unknown token is a no-op success, so logging out never reveals
// whether a token existed.
func (s *AuthService) Logout(ctx context.Context, rawRefresh string) error {
	if rawRefresh == "" {
		return nil
	}
	session, err := s.sessions.ByTokenHash(ctx, auth.HashRefreshToken(rawRefresh))
	if err != nil {
		if domain.IsNotFound(err) {
			return nil
		}
		return err
	}
	return s.sessions.Revoke(ctx, session.ID, s.now())
}

// issueSession mints an access token and persists a new refresh session, whose
// lifetime depends on rememberMe.
func (s *AuthService) issueSession(ctx context.Context, user *domain.User, rememberMe bool, userAgent, ip string) (*Credentials, error) {
	accessToken, accessExpiresAt, err := s.tokens.Issue(user.ID)
	if err != nil {
		return nil, err
	}

	rawRefresh, err := auth.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	ttl := s.cfg.RefreshTTLDefault
	if rememberMe {
		ttl = s.cfg.RefreshTTLRemember
	}

	session := &domain.Session{
		UserID:           user.ID,
		RefreshTokenHash: auth.HashRefreshToken(rawRefresh),
		RememberMe:       rememberMe,
		UserAgent:        userAgent,
		IP:               ip,
		ExpiresAt:        s.now().Add(ttl),
	}
	if err := s.sessions.Create(ctx, session); err != nil {
		return nil, err
	}

	return &Credentials{
		User:             user,
		AccessToken:      accessToken,
		AccessExpiresAt:  accessExpiresAt,
		RefreshToken:     rawRefresh,
		RefreshExpiresAt: session.ExpiresAt,
	}, nil
}
