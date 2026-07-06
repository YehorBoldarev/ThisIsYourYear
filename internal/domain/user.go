package domain

import (
	"fmt"
	"strings"
	"time"
)

// User is a registered account. It is a pure domain entity: it carries no
// storage- or transport-specific concerns (no Mongo tags, no JSON tags). The
// persistence layer maps it to and from its own document model, and the
// transport layer maps it to and from DTOs.
//
// PasswordHash is part of the entity because authentication is a domain concern,
// but it must never be exposed at the API boundary; the transport DTOs simply do
// not include it.
type User struct {
	ID           string
	Email        string
	DisplayName  string
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// NewUser constructs a valid User from already-processed inputs (the password is
// expected to be hashed by the caller, since hashing is an infrastructure
// concern that lives outside the domain). Email is normalized and all required
// invariants are enforced; violations are reported as ErrValidation. The
// returned user has no ID or timestamps until it is persisted.
func NewUser(email, displayName, passwordHash string) (*User, error) {
	email = NormalizeEmail(email)
	displayName = strings.TrimSpace(displayName)

	if email == "" {
		return nil, fmt.Errorf("%w: email is required", ErrValidation)
	}
	if displayName == "" {
		return nil, fmt.Errorf("%w: display name is required", ErrValidation)
	}
	if passwordHash == "" {
		return nil, fmt.Errorf("%w: password hash is required", ErrValidation)
	}

	return &User{
		Email:        email,
		DisplayName:  displayName,
		PasswordHash: passwordHash,
	}, nil
}

// NormalizeEmail canonicalizes an email address for storage and lookup so that
// addresses differing only by surrounding whitespace or letter case are treated
// as the same account.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
