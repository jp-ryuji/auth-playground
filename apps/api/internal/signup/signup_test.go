package signup_test

// Spec-to-test mapping for docs/specs/10-flows/signup-first-login.md.
//
// Conventions:
//   - One Test function per requirement ID (SIGNUP-01..14).
//   - Function name embeds the ID: TestSignup_SIGNUP_NN_<slug>.
//   - Each test currently calls t.Skip with a short quote of the spec text;
//     implementing the package means replacing each Skip with real assertions.
//   - Acceptance-criteria style scenarios live at the bottom under
//     TestSignup_Acceptance_* and reference the IDs they exercise.
//
// Run:  go test ./internal/signup/...
// Goal: every SIGNUP-NN green → promote the spec from Draft to Stable.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jp-ryuji/oidc-portfolio-api/internal/discovery"
	"github.com/jp-ryuji/oidc-portfolio-api/internal/signup"
)

const specHref = "docs/specs/10-flows/signup-first-login.md"

// pending fails-soft so the suite stays green while the scaffold is in place.
// Swap to t.Fatalf once you start implementing the corresponding requirement.
func pending(t *testing.T, id, quote string) {
	t.Helper()
	t.Skipf("not implemented: %s — %q (see %s#%s)", id, quote, specHref, lower(id))
}

func lower(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if 'A' <= c && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}

// --- Authorization request -------------------------------------------------

// TestSignup_SIGNUP_01_AuthorizeURLBuiltFromDiscovery is the hermetic
// proof of SIGNUP-01: the authorize URL's scheme/host/path come from
// discovery's authorization_endpoint and the query contains every
// required parameter, including the S256 code_challenge_method.
func TestSignup_SIGNUP_01_AuthorizeURLBuiltFromDiscovery(t *testing.T) {
	const (
		fakeAuthorizeEndpoint = "https://issuer.example/oauth2/auth"
		clientID              = "test-rp"
		redirectURI           = "http://127.0.0.1:8080/auth/callback"
	)

	// Stub the OP. The Issuer field is left blank intentionally — SIGNUP-01
	// only mandates that authorization_endpoint come from discovery; the
	// issuer-mismatch check belongs to SIGNUP-12 (ID-token iss validation).
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/.well-known/openid-configuration" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                                r.Host, // unused by this test
			"authorization_endpoint":                fakeAuthorizeEndpoint,
			"token_endpoint":                        "https://issuer.example/oauth2/token",
			"jwks_uri":                              "https://issuer.example/.well-known/jwks.json",
			"code_challenge_methods_supported":      []string{"S256"},
		})
	}))
	t.Cleanup(ts.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	doc, err := discovery.NewClient(ts.URL, ts.Client()).Fetch(ctx)
	if err != nil {
		t.Fatalf("discovery.Fetch: %v", err)
	}
	if got, want := doc.AuthorizationEndpoint, fakeAuthorizeEndpoint; got != want {
		t.Fatalf("doc.AuthorizationEndpoint = %q, want %q", got, want)
	}

	pkce, err := signup.NewPKCE()
	if err != nil {
		t.Fatalf("NewPKCE: %v", err)
	}
	state, err := signup.RandomURLSafe(16)
	if err != nil {
		t.Fatalf("RandomURLSafe(state): %v", err)
	}
	nonce, err := signup.RandomURLSafe(16)
	if err != nil {
		t.Fatalf("RandomURLSafe(nonce): %v", err)
	}

	raw, err := signup.BuildAuthorizeURL(doc, signup.AuthorizeParams{
		ClientID:      clientID,
		RedirectURI:   redirectURI,
		Scopes:        []string{"openid", "offline"},
		State:         state,
		Nonce:         nonce,
		CodeChallenge: pkce.Challenge,
	})
	if err != nil {
		t.Fatalf("BuildAuthorizeURL: %v", err)
	}

	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("url.Parse(%q): %v", raw, err)
	}

	// Clause: built from discovery authorization_endpoint.
	if got, want := u.Scheme, "https"; got != want {
		t.Errorf("scheme = %q, want %q (must come from discovery)", got, want)
	}
	if got, want := u.Host, "issuer.example"; got != want {
		t.Errorf("host = %q, want %q (must come from discovery)", got, want)
	}
	if got, want := u.Path, "/oauth2/auth"; got != want {
		t.Errorf("path = %q, want %q (must come from discovery)", got, want)
	}

	q := u.Query()

	// Clause: response_type=code.
	if got, want := q.Get("response_type"), "code"; got != want {
		t.Errorf("response_type = %q, want %q", got, want)
	}

	// Clause: scope contains at least "openid".
	scopes := strings.Fields(q.Get("scope"))
	if !slices.Contains(scopes, "openid") {
		t.Errorf("scope = %q, must contain %q", q.Get("scope"), "openid")
	}

	// Clause: state present (and matches what was passed).
	if got := q.Get("state"); got != state {
		t.Errorf("state = %q, want %q", got, state)
	}

	// Clause: nonce present (and matches what was passed).
	if got := q.Get("nonce"); got != nonce {
		t.Errorf("nonce = %q, want %q", got, nonce)
	}

	// Clause: code_challenge present (and matches the PKCE producer).
	if got := q.Get("code_challenge"); got != pkce.Challenge {
		t.Errorf("code_challenge = %q, want %q", got, pkce.Challenge)
	}

	// Clause: code_challenge_method=S256.
	if got, want := q.Get("code_challenge_method"), "S256"; got != want {
		t.Errorf("code_challenge_method = %q, want %q", got, want)
	}

	// Guard against accidental duplicate query keys.
	for _, key := range []string{"response_type", "scope", "state", "nonce", "code_challenge", "code_challenge_method", "client_id", "redirect_uri"} {
		if n := len(q[key]); n != 1 {
			t.Errorf("query[%q] has %d values, want 1", key, n)
		}
	}

	// Operational sanity (not a SIGNUP-01 clause): client_id and redirect_uri
	// are needed for the URL to be meaningful but are out of scope for the
	// requirement text. Verify they round-trip the inputs.
	if got := q.Get("client_id"); got != clientID {
		t.Errorf("client_id = %q, want %q", got, clientID)
	}
	if got := q.Get("redirect_uri"); got != redirectURI {
		t.Errorf("redirect_uri = %q, want %q", got, redirectURI)
	}
}

// TestSignup_SIGNUP_01_BuildAuthorizeURL_Invalid covers the negative
// cases for the URL builder: every clause SIGNUP-01 names plus the
// operational fields are required. Co-located with the positive test
// because they share the same requirement and fixture surface.
func TestSignup_SIGNUP_01_BuildAuthorizeURL_Invalid(t *testing.T) {
	good := signup.AuthorizeParams{
		ClientID:      "test-rp",
		RedirectURI:   "http://127.0.0.1:8080/auth/callback",
		Scopes:        []string{"openid"},
		State:         "s",
		Nonce:         "n",
		CodeChallenge: "c",
	}
	goodDoc := &discovery.Document{AuthorizationEndpoint: "https://issuer.example/oauth2/auth"}

	cases := []struct {
		name string
		doc  *discovery.Document
		mut  func(*signup.AuthorizeParams)
	}{
		{"nil doc", nil, func(*signup.AuthorizeParams) {}},
		{"empty authorization_endpoint", &discovery.Document{}, func(*signup.AuthorizeParams) {}},
		{"missing client_id", goodDoc, func(p *signup.AuthorizeParams) { p.ClientID = "" }},
		{"missing redirect_uri", goodDoc, func(p *signup.AuthorizeParams) { p.RedirectURI = "" }},
		{"nil scopes", goodDoc, func(p *signup.AuthorizeParams) { p.Scopes = nil }},
		{"scopes without openid", goodDoc, func(p *signup.AuthorizeParams) { p.Scopes = []string{"profile"} }},
		{"missing state", goodDoc, func(p *signup.AuthorizeParams) { p.State = "" }},
		{"missing nonce", goodDoc, func(p *signup.AuthorizeParams) { p.Nonce = "" }},
		{"missing code_challenge", goodDoc, func(p *signup.AuthorizeParams) { p.CodeChallenge = "" }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := good
			c.mut(&p)
			if _, err := signup.BuildAuthorizeURL(c.doc, p); err == nil {
				t.Fatalf("BuildAuthorizeURL: expected error, got nil")
			}
		})
	}
}

// TestSignup_SIGNUP_01_LiveDiscovery exercises the same construction
// path against the running Hydra in compose.yaml. Opt-in via env var so
// CI without the stack stays green.
//
// Convention exception: this is the only requirement ID with a second
// test function. It tests the same SIGNUP-01 clauses against a different
// fixture (live OP) — see plan-signup-01-against-the-functional-petal.md.
//
// Note: this does NOT GET the constructed authorize URL — the interactive
// RP client is not yet seeded (docker/hydra/hydra.yml flags
// scripts/seed-clients.sh as TODO). Following the URL is acceptance work.
func TestSignup_SIGNUP_01_LiveDiscovery(t *testing.T) {
	if os.Getenv("AUTH_PLAYGROUND_LIVE") != "1" {
		t.Skip("set AUTH_PLAYGROUND_LIVE=1 to run; requires `make up` (see Makefile)")
	}

	const issuer = "http://127.0.0.1:4444/"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	doc, err := discovery.NewClient(issuer, http.DefaultClient).Fetch(ctx)
	if err != nil {
		t.Fatalf("discovery.Fetch(%s): %v (is `make up` running?)", issuer, err)
	}
	if doc.AuthorizationEndpoint == "" {
		t.Fatalf("live discovery returned empty authorization_endpoint")
	}
	// Real OPs may host authorization on a different host than the issuer;
	// Hydra dev keeps them aligned. Assert the prefix, not the host.
	if !strings.HasPrefix(doc.AuthorizationEndpoint, issuer) {
		t.Fatalf("authorization_endpoint = %q, want prefix %q", doc.AuthorizationEndpoint, issuer)
	}
	if !slices.Contains(doc.CodeChallengeMethods, "S256") {
		t.Fatalf("code_challenge_methods_supported = %v, want to contain %q", doc.CodeChallengeMethods, "S256")
	}

	pkce, err := signup.NewPKCE()
	if err != nil {
		t.Fatalf("NewPKCE: %v", err)
	}
	state, err := signup.RandomURLSafe(16)
	if err != nil {
		t.Fatalf("RandomURLSafe(state): %v", err)
	}
	nonce, err := signup.RandomURLSafe(16)
	if err != nil {
		t.Fatalf("RandomURLSafe(nonce): %v", err)
	}

	raw, err := signup.BuildAuthorizeURL(doc, signup.AuthorizeParams{
		ClientID:      "auth-playground-rp", // placeholder; not yet seeded in Hydra
		RedirectURI:   "http://127.0.0.1:8080/auth/callback",
		Scopes:        []string{"openid", "offline"},
		State:         state,
		Nonce:         nonce,
		CodeChallenge: pkce.Challenge,
	})
	if err != nil {
		t.Fatalf("BuildAuthorizeURL: %v", err)
	}

	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("url.Parse(%q): %v", raw, err)
	}
	q := u.Query()

	if got, want := q.Get("response_type"), "code"; got != want {
		t.Errorf("response_type = %q, want %q", got, want)
	}
	if scopes := strings.Fields(q.Get("scope")); !slices.Contains(scopes, "openid") {
		t.Errorf("scope = %q, must contain %q", q.Get("scope"), "openid")
	}
	if q.Get("state") == "" {
		t.Error("state is empty")
	}
	if q.Get("nonce") == "" {
		t.Error("nonce is empty")
	}
	if q.Get("code_challenge") == "" {
		t.Error("code_challenge is empty")
	}
	if got, want := q.Get("code_challenge_method"), "S256"; got != want {
		t.Errorf("code_challenge_method = %q, want %q", got, want)
	}

	t.Logf("live authorize URL built: %s", raw)
}

func TestSignup_SIGNUP_02_StateIsRandomBoundSingleUse(t *testing.T) {
	pending(t, "SIGNUP-02",
		"state MUST be cryptographically random (≥128 bits), bound server-side to the browser, and single-use")
}

func TestSignup_SIGNUP_03_NonceBoundAndValidatedAtCallback(t *testing.T) {
	pending(t, "SIGNUP-03",
		"nonce MUST be cryptographically random, bound to the same record as state, "+
			"and validated against the ID-token nonce claim at callback")
}

func TestSignup_SIGNUP_04_PKCEVerifierStaysOnBFF(t *testing.T) {
	pending(t, "SIGNUP-04",
		"PKCE code_verifier MUST be per-request, stored only server-side on apps/api, MUST NOT leave the BFF")
}

// --- Hydra login challenge (apps/oauth-login) ------------------------------

func TestSignup_SIGNUP_05_ResolveLoginChallengeViaHydraAdmin(t *testing.T) {
	pending(t, "SIGNUP-05",
		"apps/oauth-login MUST resolve login_challenge via Hydra Admin; "+
			"MUST NOT trust query-string fields beyond the challenge id")
}

func TestSignup_SIGNUP_06_RedirectToKratosWhenNoSession(t *testing.T) {
	pending(t, "SIGNUP-06",
		"if no Kratos session exists, apps/oauth-login MUST redirect into a Kratos self-service flow")
}

func TestSignup_SIGNUP_07_AcceptLoginWithKratosIdentityAsSubject(t *testing.T) {
	pending(t, "SIGNUP-07",
		"on Kratos session, apps/oauth-login MUST accept the login with subject = Kratos identity id (OV-11)")
}

// --- Hydra consent challenge (apps/oauth-login) ----------------------------

func TestSignup_SIGNUP_08_ConsentShowsHydraSuppliedScopeOnly(t *testing.T) {
	pending(t, "SIGNUP-08",
		"on GET /consent, apps/oauth-login MUST show requested_scope/requested_audience as returned by Hydra; "+
			"MUST NOT silently widen them")
}

func TestSignup_SIGNUP_09_ClaimsAreServerSourced(t *testing.T) {
	pending(t, "SIGNUP-09",
		"id_token/access_token claims MUST be populated only with values permitted by the client policy; "+
			"tenant_id MUST be derived server-side, never from request input")
}

// --- Callback at apps/api --------------------------------------------------

func TestSignup_SIGNUP_10_StateVerifiedBeforeTokenExchange(t *testing.T) {
	pending(t, "SIGNUP-10",
		"apps/api MUST verify state against the bound record BEFORE exchanging the code; "+
			"mismatch MUST terminate the flow and clear the record")
}

func TestSignup_SIGNUP_11_CodeExchangeUsesDiscoveryAndInteractiveClient(t *testing.T) {
	pending(t, "SIGNUP-11",
		"code-for-token exchange MUST hit token_endpoint from discovery, include the code_verifier, "+
			"and use the interactive RP client distinct from M2M and exchange clients (OV-04)")
}

func TestSignup_SIGNUP_12_IDTokenValidated(t *testing.T) {
	pending(t, "SIGNUP-12",
		"apps/api MUST validate ID token: JWKS signature, iss, aud includes RP client id, exp/iat within ≤120s skew, nonce")
}

func TestSignup_SIGNUP_13_RefreshTokenServerOnly(t *testing.T) {
	pending(t, "SIGNUP-13",
		"refresh token MUST be persisted server-side, scoped to the session, MUST NOT be sent to the browser")
}

func TestSignup_SIGNUP_14_OpaqueSessionCookie(t *testing.T) {
	pending(t, "SIGNUP-14",
		"apps/api MUST issue a session cookie satisfying OV-09; cookie value MUST be opaque (not a JWT, not any Hydra token)")
}

// --- Acceptance scenarios --------------------------------------------------
//
// These cover the checklist at the bottom of signup-first-login.md. They
// compose multiple SIGNUP-NN requirements; the comment on each names the IDs
// the scenario exercises. Implement after the unit-level tests above pass.

func TestSignup_Acceptance_HappyPath(t *testing.T) {
	// Exercises SIGNUP-01..11 end-to-end via headless browser / HTTP client.
	pending(t, "ACCEPT-HAPPY",
		"signed-out user → Hydra → oauth-login → SSUI → oauth-login → Hydra → apps/api/callback → signed in")
}

func TestSignup_Acceptance_TamperedStateRejected(t *testing.T) {
	pending(t, "ACCEPT-STATE", "tampering with state at callback rejects and leaves no session cookie (SIGNUP-10)")
}

func TestSignup_Acceptance_CodeReplayRejected(t *testing.T) {
	pending(t, "ACCEPT-REPLAY", "replaying a captured code after first exchange is rejected by Hydra (SIGNUP-11)")
}

func TestSignup_Acceptance_BadIDTokenRejected(t *testing.T) {
	pending(t, "ACCEPT-IDT",
		"ID token with mismatched nonce, missing aud, or expired exp is rejected before session is created (SIGNUP-12)")
}

func TestSignup_Acceptance_NoTokensInBrowser(t *testing.T) {
	pending(t, "ACCEPT-TOK",
		"no Hydra tokens in cookies, local storage, or page state at end of flow (OV-08, SIGNUP-13, SIGNUP-14)")
}

func TestSignup_Acceptance_SubClaimEqualsKratosIdentityID(t *testing.T) {
	pending(t, "ACCEPT-SUB", "OIDC sub claim equals the Kratos identity id of the registered user (OV-11, SIGNUP-07)")
}

func TestSignup_Acceptance_AdminResponseNotLeakedToBrowser(t *testing.T) {
	pending(t, "ACCEPT-LEAK",
		"apps/oauth-login does not echo Hydra Admin response fields beyond redirect_to (SIGNUP-05, SIGNUP-08)")
}
