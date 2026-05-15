# Flow: Returning-user login

Status: Draft
Last updated: 2026-05-12
Scope: browser flow for a user who already has identity in Kratos and may already have a Kratos session, a Hydra remembered consent, or a valid product session cookie at `apps/api`. Written as a **delta** against [signup-first-login.md](./signup-first-login.md).

References:
- [signup-first-login.md](./signup-first-login.md), [system-overview.md](../00-architecture/system-overview.md), [glossary.md](../glossary.md)
- [RFC 9700](https://www.rfc-editor.org/rfc/rfc9700) §4 — Session handling

## Purpose

Describe the three short-circuits that distinguish returning-user login from first login, and pin down which sign-up requirements still apply unchanged. Read the sign-up spec first; this one only states what changes.

## Out of scope

- Session refresh and refresh-token rotation (separate spec: `refresh-rotation.md`, deferred).
- Logout (separate spec).
- Account switching, account linking, multi-session UX.

## Short-circuit summary

The returning-user flow has three points where work can be skipped, in order of "earliness":

1. **Product session cookie valid at `apps/api`.** No round-trip to Hydra at all; `apps/api` serves the request from the existing app session.
2. **Kratos session valid when `login_challenge` arrives.** `apps/oauth-login` accepts the login challenge without redirecting through the Self-Service UI.
3. **Hydra has a remembered grant** for `(client, subject, requested_scope, requested_audience)`. The consent step is auto-accepted by Hydra without `apps/oauth-login` involvement.

## Sequence

```mermaid
sequenceDiagram
  autonumber
  actor U as User
  participant B as Browser
  participant W as apps/web
  participant BFF as apps/api (BFF / RP)
  participant H as ORY Hydra
  participant OL as apps/oauth-login
  participant K as ORY Kratos

  U->>B: Open protected route
  B->>W: GET protected route
  W->>BFF: Request requires session

  alt Product session cookie valid (short-circuit 1)
    BFF-->>W: 200 (existing session)
    W-->>U: Render signed-in screen
  else No / expired product session
    BFF->>B: 302 to Hydra /oauth2/auth
    B->>H: GET /oauth2/auth
    H->>B: 302 to apps/oauth-login?login_challenge=...
    B->>OL: GET /login?login_challenge=...
    OL->>K: Check Kratos session for the browser

    alt Kratos session valid (short-circuit 2)
      K-->>OL: identity_id present
      OL->>H: Admin: accept login (subject = identity_id)
    else No / expired Kratos session
      OL->>B: 302 to Self-Service UI (login flow)
      Note over B,K: User signs into Kratos; UI redirects back to apps/oauth-login (resume)
      OL->>H: Admin: accept login (subject = identity_id)
    end

    H-->>OL: 200 { redirect_to: Hydra consent URL }
    OL->>B: 302 redirect_to
    B->>H: GET consent

    alt Hydra has remembered grant (short-circuit 3)
      H->>B: 302 directly to apps/api /callback?code=...
    else Consent prompt required
      H->>B: 302 to apps/oauth-login?consent_challenge=...
      B->>OL: GET /consent?consent_challenge=...
      U->>B: Approve scopes
      B->>OL: POST /consent
      OL->>H: Admin: accept consent
      H-->>OL: 200 { redirect_to: apps/api /callback?code=... }
      OL->>B: 302 redirect_to
    end

    B->>BFF: GET /callback?code=...&state=...
    BFF->>H: POST /oauth2/token
    H-->>BFF: id_token, access_token, refresh_token
    BFF->>BFF: Validate; create or refresh app session
    BFF->>B: Set-Cookie session=...
    B->>W: GET protected route
    W-->>U: Render signed-in screen
  end
```

## Requirements

### Product-session short-circuit (`apps/api`)

- <a id="return-01"></a>**RETURN-01.** When `apps/api` receives a request with a valid, non-expired product session cookie, it **MUST** serve the request from the existing session and **MUST NOT** initiate a fresh authorize round-trip.
- <a id="return-02"></a>**RETURN-02.** "Valid" means: the cookie value resolves to an active server-side session record, the record is not past its absolute or idle timeout, and the cached access token is either non-expired or refresh succeeds (refresh policy: separate spec).
- <a id="return-03"></a>**RETURN-03.** A session record marked revoked (logout, admin action, password change) **MUST** be treated as no session, even if the cookie is still presented. The cookie **MUST** be cleared on the response.

### Kratos-session short-circuit (`apps/oauth-login`)

- <a id="return-04"></a>**RETURN-04.** On `GET /login?login_challenge=...`, if a valid Kratos session is bound to the browser, `apps/oauth-login` **MUST** accept the Hydra login challenge directly with `subject = identity_id` and **MUST NOT** redirect through the Self-Service UI.
- <a id="return-05"></a>**RETURN-05.** "Valid Kratos session" means a session resolved via Kratos public API for the browser's Kratos cookie, with state `active` and any AAL / freshness requirements satisfied (AAL handling: future MFA spec).
- <a id="return-06"></a>**RETURN-06.** If the Kratos session indicates a different identity than a still-present product session at `apps/api`, `apps/api` **MUST** treat the mismatch as a session conflict: discard the product session, clear the cookie, and complete the new login. Silent identity swap is **NOT** permitted.

### Consent remember (`apps/oauth-login` / Hydra)

- <a id="return-07"></a>**RETURN-07.** Hydra's "remember consent" feature **MAY** be enabled per client. When enabled, the consent step **MAY** skip the `consent_challenge` round-trip to `apps/oauth-login`.
- <a id="return-08"></a>**RETURN-08.** Remembered grants **MUST** be keyed by `(client_id, subject, requested_scope, requested_audience)`. A new `requested_scope` or `requested_audience` **MUST** force a fresh consent prompt.
- <a id="return-09"></a>**RETURN-09.** When `apps/oauth-login` does receive a `consent_challenge` for a returning user, every requirement from [SIGNUP-08](./signup-first-login.md#signup-08) and [SIGNUP-09](./signup-first-login.md#signup-09) still applies. There is no "trusted returning user" relaxation.

### What still applies from sign-up

- <a id="return-10"></a>**RETURN-10.** All authorization-request rules from sign-up — [SIGNUP-01](./signup-first-login.md#signup-01)..[SIGNUP-04](./signup-first-login.md#signup-04) — apply unchanged: discovery-driven URL, PKCE (S256), single-use `state`, validated `nonce`.
- <a id="return-11"></a>**RETURN-11.** All callback rules from sign-up — [SIGNUP-10](./signup-first-login.md#signup-10)..[SIGNUP-14](./signup-first-login.md#signup-14) — apply unchanged: state verification before code exchange, ID-token validation, server-side refresh-token storage, opaque session cookie.

## Interfaces

No new endpoints. Uses the same endpoints from [signup-first-login.md](./signup-first-login.md#interfaces).

## Acceptance criteria

- [ ] A user with a valid product session cookie hitting a protected route gets a response with no redirect to Hydra. (RETURN-01)
- [ ] A user without a product session but with a valid Kratos session goes through Hydra → `apps/oauth-login` → Hydra → `apps/api/callback` without visiting the Self-Service UI. (RETURN-04)
- [ ] A user without a Kratos session is redirected into the Self-Service UI login flow, then completes the rest of the chain. (RETURN-04 negative case)
- [ ] Revoking a session record on the server immediately stops requests carrying the matching cookie from succeeding, even though the cookie is still set in the browser. The browser receives a cleared cookie on the next response. (RETURN-03)
- [ ] When Hydra has a remembered grant, the browser passes through `/oauth2/auth/consent` without hitting `apps/oauth-login`'s consent endpoint. Network trace confirms no `consent_challenge` round-trip. (RETURN-07, RETURN-08)
- [ ] Adding a new scope to the authorize request forces a fresh consent prompt even for a previously consenting user. (RETURN-08)
- [ ] Tampered `state` or expired ID token at the callback still rejects the request, identical to sign-up. (RETURN-11)

## Open questions

- Idle vs absolute session timeouts: defaults for product session, Kratos session, Hydra session. Likely an ADR.
- Whether `apps/api`'s short-circuit checks the Kratos session liveness on every request (defence in depth) or trusts the product session record until it expires.
- Whether `apps/oauth-login` should challenge for re-authentication (Kratos `aal` step-up) when `acr_values` or `max_age` are present in the authorize request.
