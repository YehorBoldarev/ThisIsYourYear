// Package auth holds the cryptographic building blocks of authentication:
// password hashing, access-token (JWT) issuing/verification, and opaque
// refresh-token generation. It is pure infrastructure — it depends only on the
// domain vocabulary for its error results and knows nothing about HTTP or Mongo.
package auth

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"

	"kibou/thisisyouryear/internal/domain"
)

// PasswordHasher hashes and verifies passwords using bcrypt at a fixed cost.
type PasswordHasher struct {
	cost int
}

// NewPasswordHasher returns a hasher at the given bcrypt cost. Costs outside the
// range bcrypt supports fall back to the library default, so a misconfiguration
// weakens to a safe value rather than failing closed at runtime.
func NewPasswordHasher(cost int) PasswordHasher {
	if cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
		cost = bcrypt.DefaultCost
	}
	return PasswordHasher{cost: cost}
}

// Hash returns the bcrypt hash of the plaintext password.
func (h PasswordHasher) Hash(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), h.cost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(b), nil
}

// Compare reports whether plain matches the stored hash. A mismatch is returned
// as domain.ErrInvalidCredentials so callers surface a uniform authentication
// failure without inspecting bcrypt internals.
func (h PasswordHasher) Compare(hash, plain string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)); err != nil {
		return domain.ErrInvalidCredentials
	}
	return nil
}
