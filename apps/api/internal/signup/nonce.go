package signup

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
)

// RandomURLSafe returns n bytes of crypto/rand encoded as unpadded
// base64url. 16 bytes (= 128 bits of entropy) is the floor SIGNUP-02
// names for state; SIGNUP-03 reuses it for nonce.
//
// SIGNUP-02/03 binding and single-use semantics belong to the store that
// will wrap this producer — keep this helper pure.
func RandomURLSafe(n int) (string, error) {
	if n <= 0 {
		return "", errors.New("signup: RandomURLSafe requires n > 0")
	}
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("signup: read random: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// ValidateNonce compares the nonce stored at authorize time against the value
// extracted from the ID token's nonce claim. Returns nil only when both are
// non-empty and equal. Uses constant-time comparison to prevent timing oracles.
func ValidateNonce(stored, fromIDToken string) error {
	if stored == "" {
		return errors.New("signup: stored nonce is empty")
	}
	if fromIDToken == "" {
		return errors.New("signup: ID-token nonce claim is absent or empty")
	}
	if subtle.ConstantTimeCompare([]byte(stored), []byte(fromIDToken)) != 1 {
		return errors.New("signup: nonce mismatch")
	}
	return nil
}
