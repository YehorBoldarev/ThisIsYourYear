package http

import (
	"time"

	"kibou/thisisyouryear/internal/domain"
	"kibou/thisisyouryear/internal/service"
)

// RegisterRequest is the body of POST /auth/register.
type RegisterRequest struct {
	Email       string `json:"email" validate:"required,email"`
	DisplayName string `json:"displayName" validate:"required,max=100"`
	// bcrypt hashes at most 72 bytes, so the password is capped there.
	Password string `json:"password" validate:"required,min=8,max=72"`
	// InviteCode is required (and checked by the service) only when sign-up is
	// invite-only; it is ignored otherwise.
	InviteCode string `json:"inviteCode,omitempty" validate:"max=200"`
}

// RegistrationInfoResponse tells the sign-up page whether to ask for an invite
// code (GET /auth/registration).
type RegistrationInfoResponse struct {
	InviteRequired bool `json:"inviteRequired"`
}

// LoginRequest is the body of POST /auth/login.
type LoginRequest struct {
	Email      string `json:"email" validate:"required,email"`
	Password   string `json:"password" validate:"required"`
	RememberMe bool   `json:"rememberMe"`
}

// UserResponse is the public projection of a user. It deliberately omits the
// password hash and any internal fields.
type UserResponse struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"displayName"`
	CreatedAt   time.Time `json:"createdAt"`
}

// TokenResponse is returned by login and refresh: the access token (kept in
// memory by the client) plus the authenticated user. The refresh token is not in
// the body — it is delivered in an HttpOnly cookie.
type TokenResponse struct {
	AccessToken string       `json:"accessToken"`
	ExpiresAt   time.Time    `json:"expiresAt"`
	User        UserResponse `json:"user"`
}

func toUserResponse(u *domain.User) UserResponse {
	return UserResponse{
		ID:          u.ID,
		Email:       u.Email,
		DisplayName: u.DisplayName,
		CreatedAt:   u.CreatedAt,
	}
}

func tokenResponse(c *service.Credentials) TokenResponse {
	return TokenResponse{
		AccessToken: c.AccessToken,
		ExpiresAt:   c.AccessExpiresAt,
		User:        toUserResponse(c.User),
	}
}
