package signup

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

// PKCE bundles the verifier and challenge for one authorize request.
//
// SIGNUP-04 will require that Verifier never leaves apps/api; this type
// is a pure producer and imposes no storage policy. The call site that
// builds the authorize URL is responsible for persisting Verifier and
// emitting only Challenge.
type PKCE struct {
	Verifier  string // RFC 7636 §4.1: 43–128 chars from the unreserved set
	Challenge string // RFC 7636 §4.2: base64url(SHA256(Verifier)) without padding
	Method    string // Always "S256"
}

// NewPKCE generates a fresh verifier (32 random bytes encoded as
// base64url-without-padding = 43 chars, well within RFC 7636 bounds) and
// its S256 challenge.
func NewPKCE() (PKCE, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return PKCE{}, fmt.Errorf("signup: read random for pkce verifier: %w", err)
	}
	verifier := base64.RawURLEncoding.EncodeToString(buf)
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])
	return PKCE{
		Verifier:  verifier,
		Challenge: challenge,
		Method:    codeChallengeMethodS256,
	}, nil
}
