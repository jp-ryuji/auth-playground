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
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jp-ryuji/auth-playground/apps/api/internal/oidc"
	"github.com/jp-ryuji/auth-playground/apps/api/internal/signup"
	"github.com/jp-ryuji/auth-playground/apps/oauth-login/hydra"
	"github.com/jp-ryuji/auth-playground/apps/oauth-login/kratos"
	"github.com/jp-ryuji/auth-playground/apps/oauth-login/login"
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
	t.Parallel()
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

	doc, err := oidc.NewClient(ts.URL, ts.Client()).Fetch(ctx)
	if err != nil {
		t.Fatalf("oidc.Fetch: %v", err)
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
	t.Parallel()

	good := signup.AuthorizeParams{
		ClientID:      "test-rp",
		RedirectURI:   "http://127.0.0.1:8080/auth/callback",
		Scopes:        []string{"openid"},
		State:         "s",
		Nonce:         "n",
		CodeChallenge: "c",
	}
	goodDoc := &oidc.Document{AuthorizationEndpoint: "https://issuer.example/oauth2/auth"}

	cases := map[string]struct {
		doc *oidc.Document
		mut func(*signup.AuthorizeParams)
	}{
		"nil doc":                       {nil, func(*signup.AuthorizeParams) {}},
		"empty authorization_endpoint":  {&oidc.Document{}, func(*signup.AuthorizeParams) {}},
		"missing client_id":             {goodDoc, func(p *signup.AuthorizeParams) { p.ClientID = "" }},
		"missing redirect_uri":          {goodDoc, func(p *signup.AuthorizeParams) { p.RedirectURI = "" }},
		"nil scopes":                    {goodDoc, func(p *signup.AuthorizeParams) { p.Scopes = nil }},
		"scopes without openid":         {goodDoc, func(p *signup.AuthorizeParams) { p.Scopes = []string{"profile"} }},
		"missing state":                 {goodDoc, func(p *signup.AuthorizeParams) { p.State = "" }},
		"missing nonce":                 {goodDoc, func(p *signup.AuthorizeParams) { p.Nonce = "" }},
		"missing code_challenge":        {goodDoc, func(p *signup.AuthorizeParams) { p.CodeChallenge = "" }},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
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

	doc, err := oidc.NewClient(issuer, http.DefaultClient).Fetch(ctx)
	if err != nil {
		t.Fatalf("oidc.Fetch(%s): %v (is `make up` running?)", issuer, err)
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
	// Clause: cryptographically random, ≥128 bits of entropy.
	state, err := signup.RandomURLSafe(16) // 16 bytes = 128 bits
	if err != nil {
		t.Fatalf("RandomURLSafe: %v", err)
	}
	raw, err := base64.RawURLEncoding.DecodeString(state)
	if err != nil {
		t.Fatalf("DecodeString(%q): %v", state, err)
	}
	if len(raw) < 16 {
		t.Errorf("state encodes %d bytes, want ≥16 (128 bits)", len(raw))
	}

	nonce, err := signup.RandomURLSafe(16)
	if err != nil {
		t.Fatalf("RandomURLSafe(nonce): %v", err)
	}
	pkce, err := signup.NewPKCE()
	if err != nil {
		t.Fatalf("NewPKCE: %v", err)
	}

	// Clause: bound to a short-lived server-side record.
	store := signup.NewStore(10 * time.Minute)
	if err := store.Save(state, signup.AuthState{State: state, Nonce: nonce, Verifier: pkce.Verifier}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := store.Consume(state)
	if err != nil {
		t.Fatalf("Consume: %v", err)
	}
	if got.State != state {
		t.Errorf("AuthState.State = %q, want %q", got.State, state)
	}
	if got.Nonce != nonce {
		t.Errorf("AuthState.Nonce = %q, want %q", got.Nonce, nonce)
	}
	if got.Verifier != pkce.Verifier {
		t.Errorf("AuthState.Verifier = %q, want %q", got.Verifier, pkce.Verifier)
	}

	// Clause: single-use — second Consume on the same key must fail.
	if _, err := store.Consume(state); err == nil {
		t.Fatal("second Consume succeeded; want error (single-use)")
	}

	// Unknown key must fail.
	if _, err := store.Consume("unknown-key"); err == nil {
		t.Fatal("Consume with unknown key succeeded; want error")
	}

	// Clause: short-lived — expired record must be rejected.
	state2, _ := signup.RandomURLSafe(16)
	shortStore := signup.NewStore(time.Millisecond)
	_ = shortStore.Save(state2, signup.AuthState{State: state2})
	time.Sleep(5 * time.Millisecond)
	if _, err := shortStore.Consume(state2); err == nil {
		t.Fatal("Consume on expired record succeeded; want error")
	}
}

func TestSignup_SIGNUP_03_NonceBoundAndValidatedAtCallback(t *testing.T) {
	t.Parallel()
	// Clause: cryptographically random, ≥128 bits.
	nonce, err := signup.RandomURLSafe(16)
	if err != nil {
		t.Fatalf("RandomURLSafe: %v", err)
	}
	raw, err := base64.RawURLEncoding.DecodeString(nonce)
	if err != nil {
		t.Fatalf("DecodeString(%q): %v — nonce must be valid base64url", nonce, err)
	}
	if len(raw) < 16 {
		t.Errorf("nonce encodes %d bytes, want ≥16 (128 bits)", len(raw))
	}

	// Clause: bound to the same record as state.
	state, err := signup.RandomURLSafe(16)
	if err != nil {
		t.Fatalf("RandomURLSafe(state): %v", err)
	}
	pkce, err := signup.NewPKCE()
	if err != nil {
		t.Fatalf("NewPKCE: %v", err)
	}
	store := signup.NewStore(10 * time.Minute)
	if err := store.Save(state, signup.AuthState{State: state, Nonce: nonce, Verifier: pkce.Verifier}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := store.Consume(state)
	if err != nil {
		t.Fatalf("Consume: %v", err)
	}
	if got.Nonce != nonce {
		t.Errorf("AuthState.Nonce = %q, want %q (must be bound to same record as state)", got.Nonce, nonce)
	}

	// Clause: validated against the ID-token nonce claim at callback.
	if err := signup.ValidateNonce(got.Nonce, nonce); err != nil {
		t.Errorf("ValidateNonce(match): unexpected error: %v", err)
	}
	if err := signup.ValidateNonce(got.Nonce, "wrong-nonce"); err == nil {
		t.Error("ValidateNonce(mismatch): expected error, got nil")
	}
	if err := signup.ValidateNonce(got.Nonce, ""); err == nil {
		t.Error("ValidateNonce(empty fromIDToken): expected error, got nil")
	}
}

func TestSignup_SIGNUP_04_PKCEVerifierStaysOnBFF(t *testing.T) {
	t.Parallel()
	const (
		fakeAuthorizeEndpoint = "https://issuer.example/oauth2/auth"
		clientID              = "test-rp"
		redirectURI           = "http://127.0.0.1:8080/auth/callback"
	)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/.well-known/openid-configuration" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                           r.Host,
			"authorization_endpoint":           fakeAuthorizeEndpoint,
			"token_endpoint":                   "https://issuer.example/oauth2/token",
			"jwks_uri":                         "https://issuer.example/.well-known/jwks.json",
			"code_challenge_methods_supported": []string{"S256"},
		})
	}))
	t.Cleanup(ts.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	doc, err := oidc.NewClient(ts.URL, ts.Client()).Fetch(ctx)
	if err != nil {
		t.Fatalf("oidc.Fetch: %v", err)
	}

	store := signup.NewStore(10 * time.Minute)
	handler := &signup.LoginHandler{
		Doc:         doc,
		Store:       store,
		ClientID:    clientID,
		RedirectURI: redirectURI,
		Scopes:      []string{"openid", "offline"},
	}

	doRequest := func(t *testing.T) (loc, state string, cookie *http.Cookie) {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, "/auth/login", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		// Clause: 302 redirect.
		if rec.Code != http.StatusFound {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusFound)
		}

		loc = rec.Header().Get("Location")
		u, err := url.Parse(loc)
		if err != nil {
			t.Fatalf("url.Parse(Location %q): %v", loc, err)
		}
		state = u.Query().Get("state")

		for _, c := range rec.Result().Cookies() {
			if c.Name == "auth_state" {
				cookie = c
				break
			}
		}
		return loc, state, cookie
	}

	// --- First request -------------------------------------------------------

	loc1, state1, cookie1 := doRequest(t)

	u1, _ := url.Parse(loc1)
	q1 := u1.Query()

	// Clause: challenge is on the wire.
	challenge1 := q1.Get("code_challenge")
	if challenge1 == "" {
		t.Fatal("code_challenge missing from redirect URL")
	}

	// Clause: code_verifier key is absent from the redirect URL.
	if q1.Get("code_verifier") != "" {
		t.Error("code_verifier key found in redirect URL; MUST NOT leave BFF")
	}

	// Retrieve the server-side record to inspect the verifier.
	authState1, err := store.Consume(state1)
	if err != nil {
		t.Fatalf("store.Consume: %v", err)
	}

	// Clause: challenge matches S256(verifier).
	sum := sha256.Sum256([]byte(authState1.Verifier))
	expectedChallenge := base64.RawURLEncoding.EncodeToString(sum[:])
	if challenge1 != expectedChallenge {
		t.Errorf("code_challenge = %q, want S256(verifier) = %q", challenge1, expectedChallenge)
	}

	// Clause: verifier != challenge (verifier is the pre-hash input).
	if authState1.Verifier == challenge1 {
		t.Error("Verifier == Challenge; they must differ (verifier is the pre-hash value)")
	}

	// Clause: verifier value does not appear anywhere in the redirect URL.
	if strings.Contains(loc1, authState1.Verifier) {
		t.Errorf("code_verifier value found in redirect URL: %q", loc1)
	}

	// Clause: auth-state cookie set and binds browser to the pending record.
	if cookie1 == nil {
		t.Fatal("auth_state cookie not set")
	}
	if !cookie1.HttpOnly {
		t.Error("auth_state cookie: HttpOnly must be true")
	}
	if cookie1.SameSite < http.SameSiteLaxMode {
		t.Errorf("auth_state cookie: SameSite = %v, want ≥ Lax", cookie1.SameSite)
	}
	if cookie1.Value != state1 {
		t.Errorf("auth_state cookie value = %q, want state %q", cookie1.Value, state1)
	}

	// --- Second request (per-request freshness) ------------------------------

	loc2, state2, _ := doRequest(t)

	// Clause: each request produces a fresh verifier and state.
	if state2 == state1 {
		t.Error("state identical across two requests; PKCE must be per-request")
	}
	u2, _ := url.Parse(loc2)
	challenge2 := u2.Query().Get("code_challenge")
	if challenge2 == challenge1 {
		t.Error("code_challenge identical across two requests; PKCE must be per-request")
	}
}

// --- Hydra login challenge (apps/oauth-login) ------------------------------

func TestSignup_SIGNUP_05_ResolveLoginChallengeViaHydraAdmin(t *testing.T) {
	t.Parallel()

	const wantChallengeID = "test-challenge-abc123"

	// receivedChallenge captures the challenge ID the stub Hydra Admin received.
	// Buffered size 1: stub goroutine never blocks if the test is slow to drain.
	receivedChallenge := make(chan string, 1)

	hydraStub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/admin/oauth2/auth/requests/login" {
			http.NotFound(w, r)
			return
		}
		receivedChallenge <- r.URL.Query().Get("login_challenge")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"challenge":                       wantChallengeID,
			"request_url":                     "https://hydra.example/oauth2/auth",
			"requested_scope":                 []string{"openid", "offline"},
			"requested_access_token_audience": []string{},
			"skip":                            false,
			"subject":                         "",
		})
	}))
	t.Cleanup(hydraStub.Close)

	// Kratos stub returns 401 (no session) so SIGNUP-06 redirect fires.
	kratosStub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(kratosStub.Close)

	h := &login.Handler{
		HydraAdmin:        hydra.NewClient(hydraStub.URL, hydraStub.Client()),
		KratosPublic:      kratos.NewClient(kratosStub.URL, kratosStub.Client()),
		OAuthLoginBaseURL: "http://oauth-login.example",
	}

	// Clause: handler calls Hydra Admin with the exact challenge ID.
	t.Run("calls Hydra Admin with exact challenge ID", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/login?login_challenge="+wantChallengeID, nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		select {
		case got := <-receivedChallenge:
			if got != wantChallengeID {
				t.Errorf("Hydra Admin received challenge %q, want %q", got, wantChallengeID)
			}
		default:
			t.Fatal("handler did not call Hydra Admin GET /admin/oauth2/auth/requests/login")
		}

		// SIGNUP-06 redirects to Kratos when no session exists.
		if rec.Code != http.StatusFound {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusFound)
		}
	})

	// Clause: MUST NOT trust query-string fields beyond login_challenge.
	t.Run("ignores query params beyond login_challenge", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet,
			"/login?login_challenge="+wantChallengeID+"&evil=injected&subject=forged",
			nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		select {
		case got := <-receivedChallenge:
			if got != wantChallengeID {
				t.Errorf("Hydra Admin received challenge %q, want %q", got, wantChallengeID)
			}
		default:
			t.Fatal("handler did not call Hydra Admin")
		}
	})

	// Clause: missing login_challenge must be rejected before reaching Hydra Admin.
	t.Run("missing login_challenge returns 400", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/login", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		select {
		case <-receivedChallenge:
			t.Error("Hydra Admin was called despite missing login_challenge")
		default:
		}

		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})
}


func TestSignup_SIGNUP_06_RedirectToKratosWhenNoSession(t *testing.T) {
	t.Parallel()

	const challengeID = "challenge-signup06"
	const sentCookie = "ory_kratos_session=old-session-token"

	receivedKratosCookie := make(chan string, 1)

	hydraStub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"challenge":                       challengeID,
			"request_url":                     "https://hydra.example/oauth2/auth",
			"requested_scope":                 []string{"openid"},
			"requested_access_token_audience": []string{},
			"skip":                            false,
			"subject":                         "",
		})
	}))
	t.Cleanup(hydraStub.Close)

	kratosStub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/sessions/whoami" {
			receivedKratosCookie <- r.Header.Get("Cookie")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(kratosStub.Close)

	h := &login.Handler{
		HydraAdmin:        hydra.NewClient(hydraStub.URL, hydraStub.Client()),
		KratosPublic:      kratos.NewClient(kratosStub.URL, kratosStub.Client()),
		OAuthLoginBaseURL: "http://oauth-login.example",
	}

	req := httptest.NewRequest(http.MethodGet, "/login?login_challenge="+challengeID, nil)
	req.Header.Set("Cookie", sentCookie)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	// Clause: redirects to Kratos self-service registration browser flow.
	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusFound)
	}

	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "/self-service/registration/browser") {
		t.Errorf("Location %q does not contain /self-service/registration/browser", loc)
	}

	parsed, err := url.Parse(loc)
	if err != nil {
		t.Fatalf("parse Location: %v", err)
	}
	returnTo, err := url.QueryUnescape(parsed.Query().Get("return_to"))
	if err != nil {
		t.Fatalf("unescape return_to: %v", err)
	}
	if !strings.Contains(returnTo, "/login/resume") {
		t.Errorf("return_to %q does not contain /login/resume", returnTo)
	}
	if !strings.Contains(returnTo, "login_challenge="+challengeID) {
		t.Errorf("return_to %q does not contain login_challenge=%s", returnTo, challengeID)
	}

	// Clause: browser's Cookie header is forwarded to Kratos whoami.
	select {
	case got := <-receivedKratosCookie:
		if got != sentCookie {
			t.Errorf("Kratos received Cookie %q, want %q", got, sentCookie)
		}
	default:
		t.Error("Kratos /sessions/whoami was not called")
	}
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
