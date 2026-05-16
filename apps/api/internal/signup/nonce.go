package signup

import (
	"crypto/rand"
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
