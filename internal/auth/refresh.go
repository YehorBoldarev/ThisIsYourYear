package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

// refreshTokenBytes is the entropy of an opaque refresh token. 32 bytes (256
// bits) is far beyond guessing range, which is why the token can be stored as a
// fast SHA-256 hash rather than a slow password hash.
const refreshTokenBytes = 32

// GenerateRefreshToken returns a new high-entropy, URL-safe opaque token. This
// raw value is handed to the client exactly once; only its hash is persisted.
func GenerateRefreshToken() (string, error) {
	raw, err := randomBytes(refreshTokenBytes)
	if err != nil {
		return "", fmt.Errorf("generate refresh token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

// HashRefreshToken returns the hex-encoded SHA-256 digest used to store and look
// up a refresh token. Because the input is high-entropy, SHA-256 is appropriate
// (and fast); bcrypt is reserved for low-entropy passwords.
func HashRefreshToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// randomHex returns n cryptographically random bytes as a hex string.
func randomHex(n int) (string, error) {
	b, err := randomBytes(n)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func randomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("read random bytes: %w", err)
	}
	return b, nil
}
