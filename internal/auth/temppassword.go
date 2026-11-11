package auth

import (
	"crypto/rand"
	"math/big"
	"strings"
)

// tempPasswordAlphabet omits look-alike characters (0/o, 1/l/i) so a temporary
// password can be read aloud or retyped from a message without mistakes.
const tempPasswordAlphabet = "abcdefghjkmnpqrstuvwxyz23456789"

// GenerateTemporaryPassword returns a random, readable password such as
// "k7pq-m3xd-r9tw-h2nf": four groups of four characters from a 31-symbol
// alphabet (~79 bits of entropy), well above the 8-character minimum.
func GenerateTemporaryPassword() (string, error) {
	const groups, groupLen = 4, 4
	max := big.NewInt(int64(len(tempPasswordAlphabet)))
	parts := make([]string, groups)
	for g := range parts {
		var b strings.Builder
		for range groupLen {
			n, err := rand.Int(rand.Reader, max)
			if err != nil {
				return "", err
			}
			b.WriteByte(tempPasswordAlphabet[n.Int64()])
		}
		parts[g] = b.String()
	}
	return strings.Join(parts, "-"), nil
}
