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

import "testing"

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

func TestSignup_SIGNUP_01_AuthorizeURLBuiltFromDiscovery(t *testing.T) {
	pending(t, "SIGNUP-01",
		"apps/api MUST build the authorize URL from authorization_endpoint returned by discovery; "+
			"MUST include response_type=code, scope openid, state, nonce, code_challenge, code_challenge_method=S256")
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
