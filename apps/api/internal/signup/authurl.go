package signup

import (
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"

	"github.com/jp-ryuji/auth-playground/apps/api/internal/oidc"
)

// codeChallengeMethodS256 is the only PKCE transform SIGNUP-01 permits.
const codeChallengeMethodS256 = "S256"

// AuthorizeParams carries everything SIGNUP-01 must place on the wire.
//
// State/Nonce/CodeChallenge are inputs to this builder; their generation,
// binding, and single-use storage are SIGNUP-02..04 concerns and live in
// the call site that consumes BuildAuthorizeURL.
type AuthorizeParams struct {
	ClientID      string
	RedirectURI   string
	Scopes        []string // MUST contain "openid"
	State         string
	Nonce         string
	CodeChallenge string // already S256-hashed verifier; method is fixed
}

// BuildAuthorizeURL returns an OIDC authorize URL whose scheme/host/path
// come from doc.AuthorizationEndpoint and whose query satisfies SIGNUP-01:
// response_type=code, scope including openid, state, nonce, code_challenge,
// code_challenge_method=S256.
//
// Pure function: no I/O, no randomness. Errors on missing required input
// or an unparseable authorization_endpoint.
//
// client_id and redirect_uri are not named in SIGNUP-01 but are required
// for the URL to be operationally meaningful; we validate them as a
// sanity check, separately from the spec clauses.
func BuildAuthorizeURL(doc *oidc.Document, p AuthorizeParams) (string, error) {
	if doc == nil {
		return "", errors.New("signup: discovery document is nil")
	}
	if doc.AuthorizationEndpoint == "" {
		return "", errors.New("signup: discovery document has empty authorization_endpoint")
	}
	if err := validateParams(p); err != nil {
		return "", err
	}

	u, err := url.Parse(doc.AuthorizationEndpoint)
	if err != nil {
		return "", fmt.Errorf("signup: parse authorization_endpoint: %w", err)
	}

	q := u.Query()
	q.Set("response_type", "code")
	q.Set("client_id", p.ClientID)
	q.Set("redirect_uri", p.RedirectURI)
	// RFC 6749 §3.3: scope is a space-delimited list.
	q.Set("scope", strings.Join(p.Scopes, " "))
	q.Set("state", p.State)
	q.Set("nonce", p.Nonce)
	q.Set("code_challenge", p.CodeChallenge)
	q.Set("code_challenge_method", codeChallengeMethodS256)
	u.RawQuery = q.Encode()

	return u.String(), nil
}

func validateParams(p AuthorizeParams) error {
	if p.ClientID == "" {
		return errors.New("signup: client_id is required")
	}
	if p.RedirectURI == "" {
		return errors.New("signup: redirect_uri is required")
	}
	if len(p.Scopes) == 0 {
		return errors.New("signup: scopes is required and must include \"openid\"")
	}
	if !slices.Contains(p.Scopes, "openid") {
		return errors.New("signup: scopes must include \"openid\"")
	}
	if p.State == "" {
		return errors.New("signup: state is required")
	}
	if p.Nonce == "" {
		return errors.New("signup: nonce is required")
	}
	if p.CodeChallenge == "" {
		return errors.New("signup: code_challenge is required")
	}
	return nil
}
