package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"kibou/thisisyouryear/internal/domain"
)

// signingMethod is the single algorithm we issue and accept. Pinning it and
// enforcing it on verification prevents algorithm-substitution attacks (e.g. a
// forged token claiming "none" or an asymmetric alg).
var signingMethod = jwt.SigningMethodHS256

// Claims is the verified, transport-agnostic result of validating an access
// token: the identity the token asserts.
type Claims struct {
	UserID string
}

// TokenIssuer issues and verifies short-lived HS256 access tokens.
type TokenIssuer struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

// NewTokenIssuer builds an issuer signing with secret and stamping tokens with
// the given lifetime.
func NewTokenIssuer(secret string, ttl time.Duration) *TokenIssuer {
	return &TokenIssuer{secret: []byte(secret), ttl: ttl, now: time.Now}
}

// Issue returns a signed access token for userID together with its expiry.
func (i *TokenIssuer) Issue(userID string) (token string, expiresAt time.Time, err error) {
	jti, err := randomHex(16)
	if err != nil {
		return "", time.Time{}, err
	}

	now := i.now()
	expiresAt = now.Add(i.ttl)
	claims := jwt.RegisteredClaims{
		Subject:   userID,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
		ID:        jti,
	}

	signed, err := jwt.NewWithClaims(signingMethod, claims).SignedString(i.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign access token: %w", err)
	}
	return signed, expiresAt, nil
}

// Verify validates a token's signature, algorithm, and expiry and returns its
// claims. Any failure (bad signature, wrong algorithm, expired, malformed) is
// reported uniformly as domain.ErrUnauthenticated so the caller need not
// distinguish the reasons a token is unusable.
func (i *TokenIssuer) Verify(token string) (Claims, error) {
	var claims jwt.RegisteredClaims
	parsed, err := jwt.ParseWithClaims(token, &claims,
		func(t *jwt.Token) (any, error) {
			if t.Method.Alg() != signingMethod.Alg() {
				return nil, fmt.Errorf("unexpected signing method %q", t.Method.Alg())
			}
			return i.secret, nil
		},
		jwt.WithValidMethods([]string{signingMethod.Alg()}),
		jwt.WithTimeFunc(i.now),
	)
	if err != nil || !parsed.Valid || claims.Subject == "" {
		return Claims{}, domain.ErrUnauthenticated
	}
	return Claims{UserID: claims.Subject}, nil
}
