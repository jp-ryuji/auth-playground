# Flow: Sign-up / first login

Status: Draft
Last updated: 2026-05-12
Scope: end-to-end browser flow from "open the app while signed out" to "signed-in product session cookie on `apps/api`", using Hydra Authorization Code + PKCE with Kratos self-service registration.

References:
- [OpenID Connect Core 1.0 §3.1](https://openid.net/specs/openid-connect-core-1_0.html#CodeFlowAuth)
- [RFC 7636](https://www.rfc-editor.org/rfc/rfc7636) — PKCE
- [RFC 9700](https://www.rfc-editor.org/rfc/rfc9700) §2.1 — Code flow protection
- [system-overview.md](../00-architecture/system-overview.md), [glossary.md](../glossary.md)

## Purpose

Make the most exercised flow concrete and testable. This is the **reference flow**: returning-user login, MFA, and social login specs link back here and only describe deltas.

## Out of scope

- Returning-user login (separate spec): assumes a Kratos session may already exist.
- MFA, passkeys, social login, account linking.
- Token exchange and M2M flows; this spec only covers the interactive code grant.
- Concrete UI copy in the Self-Service UI.

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
  participant UI as Kratos Self-Service UI
  participant K as ORY Kratos

  U->>B: Open app while signed out
  B->>W: GET protected route
  W->>BFF: Request requires session
  BFF->>B: 302 to Hydra /oauth2/auth<br/>(client_id, redirect_uri, scope, state, nonce, PKCE)

  B->>H: GET /oauth2/auth
  H->>B: 302 to apps/oauth-login?login_challenge=...
  B->>OL: GET /login?login_challenge=...
  OL->>B: 302 to Self-Service UI (registration flow)

  B->>UI: GET registration page
  U->>B: Enter email / password / profile
  B->>UI: POST flow
  UI->>K: public API (registration)
  K-->>UI: identity created, Kratos session set

  UI->>B: 302 back to apps/oauth-login (resume)
  B->>OL: GET resume
  OL->>H: Admin: accept login (subject = kratos identity id)
  H-->>OL: 200 { redirect_to: Hydra consent URL }
  OL->>B: 302 redirect_to

  B->>H: GET consent
  H->>B: 302 to apps/oauth-login?consent_challenge=...
  B->>OL: GET /consent?consent_challenge=...
  U->>B: Approve scopes
  B->>OL: POST /consent
  OL->>H: Admin: accept consent (claims, scopes, audience)
  H-->>OL: 200 { redirect_to: apps/api /callback?code=... }
  OL->>B: 302 redirect_to

  B->>BFF: GET /callback?code=...&state=...
  BFF->>BFF: Verify state, PKCE binding
  BFF->>H: POST /oauth2/token (authorization_code, code_verifier)
  H-->>BFF: id_token, access_token, refresh_token

  BFF->>BFF: Validate ID token (iss, aud, exp, nonce, sig via JWKS)
  BFF->>BFF: Create app session; persist refresh token server-side
  BFF->>B: Set-Cookie: session=...; HttpOnly; Secure; SameSite
  B->>W: GET protected route (now with cookie)
  W-->>U: Render signed-in screen
```

## Requirements

### Authorization request

- <a id="signup-01"></a>**SIGNUP-01.** `apps/api` **MUST** build the authorize URL from `authorization_endpoint` returned by discovery ([OV-01](../00-architecture/system-overview.md#ov-01)). It **MUST** include `response_type=code`, `scope` containing at least `openid`, `state`, `nonce`, `code_challenge`, and `code_challenge_method=S256`.
- <a id="signup-02"></a>**SIGNUP-02.** `state` **MUST** be a cryptographically random value (≥128 bits of entropy), bound to a short-lived server-side record keyed for the browser (cookie or signed token), and single-use.
- <a id="signup-03"></a>**SIGNUP-03.** `nonce` **MUST** be a cryptographically random value, bound to the same record as `state`, and validated against the ID-token `nonce` claim at callback.
- <a id="signup-04"></a>**SIGNUP-04.** The PKCE `code_verifier` **MUST** be generated per request, stored only server-side on `apps/api`, and **MUST NOT** leave the BFF.

### Hydra login challenge (`apps/oauth-login`)

- <a id="signup-05"></a>**SIGNUP-05.** On `GET /login?login_challenge=...`, `apps/oauth-login` **MUST** call Hydra Admin `GET /admin/oauth2/auth/requests/login` to resolve the challenge. It **MUST NOT** trust any query-string fields beyond the challenge id.
- <a id="signup-06"></a>**SIGNUP-06.** If no Kratos session exists for the browser, `apps/oauth-login` **MUST** redirect into a Kratos self-service flow (registration for new users; login for returning users — see returning-user spec).
- <a id="signup-07"></a>**SIGNUP-07.** When Kratos confirms a session, `apps/oauth-login` **MUST** call Hydra Admin `PUT /admin/oauth2/auth/requests/login/accept` with `subject` equal to the Kratos identity id ([OV-11](../00-architecture/system-overview.md#ov-11)), then **MUST** issue an HTTP redirect to the `redirect_to` URL Hydra returns.

### Hydra consent challenge (`apps/oauth-login`)

- <a id="signup-08"></a>**SIGNUP-08.** On `GET /consent?consent_challenge=...`, `apps/oauth-login` **MUST** resolve the challenge via Hydra Admin. The `requested_scope` and `requested_audience` shown to the user **MUST** be the values Hydra returned; `apps/oauth-login` **MUST NOT** silently widen them.
- <a id="signup-09"></a>**SIGNUP-09.** On accept, `apps/oauth-login` **MUST** populate ID-token and access-token claims (`session.id_token`, `session.access_token`) only with values permitted by the Hydra client policy for the requesting client. The `tenant_id` claim, when present, **MUST** be derived server-side from Kratos identity traits or a tenant lookup, **never** from request input.

### Callback at `apps/api`

- <a id="signup-10"></a>**SIGNUP-10.** `apps/api` **MUST** verify the `state` parameter against the bound server-side record **before** exchanging the code. Mismatch **MUST** terminate the flow with an error and clear the bound record.
- <a id="signup-11"></a>**SIGNUP-11.** The code-for-token exchange **MUST** be performed against `token_endpoint` from discovery, include the matching `code_verifier`, and use the RP's client credentials — interactive RP client only, distinct from M2M and exchange clients per [OV-04](../00-architecture/system-overview.md#ov-04).
- <a id="signup-12"></a>**SIGNUP-12.** `apps/api` **MUST** validate the ID token: signature against the JWKS from discovery, `iss` matches the expected issuer, `aud` includes the RP client id, `exp`/`iat` within skew (≤120s recommended), and `nonce` matches the stored value.
- <a id="signup-13"></a>**SIGNUP-13.** `apps/api` **MUST** persist the refresh token server-side, scoped to the new app session. The refresh token **MUST NOT** be sent to the browser.
- <a id="signup-14"></a>**SIGNUP-14.** `apps/api` **MUST** issue a session cookie that satisfies [OV-09](../00-architecture/system-overview.md#ov-09). The cookie value **MUST** be opaque (not a JWT, not any Hydra token).

## Interfaces

### `apps/api`

| Endpoint | Method | Purpose |
| -------- | ------ | ------- |
| `/auth/login` (path TBD) | GET | Initiate sign-in: build authorize URL, persist state/nonce/PKCE record, 302 to Hydra. |
| `/auth/callback` (path TBD) | GET | Receive `code`, `state`; exchange and mint session cookie. |

### `apps/oauth-login`

| Endpoint | Method | Purpose |
| -------- | ------ | ------- |
| `/login` (path TBD) | GET | Handle Hydra `login_challenge`; route to Self-Service UI if no Kratos session. |
| `/login/resume` (path TBD) | GET | Resume after Kratos self-service flow returns. |
| `/consent` (path TBD) | GET | Render consent page from Hydra `consent_challenge`. |
| `/consent` (path TBD) | POST | Submit accept/reject to Hydra Admin. |

Exact paths and parameter names are open until `apps/oauth-login` is scaffolded; the **endpoint roles** above are normative.

### Cookies set during this flow

| Cookie | Set by | Purpose | Notes |
| ------ | ------ | ------- | ----- |
| Product session (name TBD) | `apps/api` | Bind browser to BFF session. | HttpOnly, Secure (non-dev), SameSite≥Lax. |
| Auth-state (name TBD) | `apps/api` | Bind browser to pending `state`/`nonce`/PKCE record. | Same flags; short TTL (≤10 min); cleared at callback. |
| Kratos session | Kratos | Identity session, scoped to Kratos domain. | Not consumed by `apps/api`. |

## Acceptance criteria

A signed-out user opening a protected route in `apps/web` ends up signed in such that:

- [ ] Following the redirect chain Hydra → `apps/oauth-login` → Self-Service UI → `apps/oauth-login` → Hydra → `apps/api/callback` completes without manual intervention beyond entering registration data and approving consent. (SIGNUP-01 .. SIGNUP-11)
- [ ] Tampering with the `state` query parameter at the callback rejects the request and leaves no session cookie set. (SIGNUP-10)
- [ ] Replaying a captured `code` after the first successful exchange is rejected by Hydra; `apps/api` surfaces the error and does not create a session. (SIGNUP-11)
- [ ] An ID token with a mismatched `nonce`, missing `aud`, or expired `exp` is rejected before the session is created. (SIGNUP-12)
- [ ] Inspecting the browser at the end of the flow shows no Hydra-issued tokens — no `access_token`, `id_token`, or `refresh_token` in cookies, local storage, or page state. (OV-08, SIGNUP-13, SIGNUP-14)
- [ ] The OIDC `sub` claim in the issued ID token equals the Kratos identity id of the registered user. (OV-11, SIGNUP-07)
- [ ] `apps/oauth-login` does not echo Hydra Admin API response fields back to the browser beyond the `redirect_to` URL. (SIGNUP-05, SIGNUP-08)

## Open questions

- Whether registration vs. login is selected by `apps/oauth-login` or by the Self-Service UI itself (likely the UI; confirm against Kratos defaults).
- Whether consent is **always** prompted on first login or auto-accepted with a remembered scope set per Hydra config (deployment choice; document the default chosen).
- Whether `tenant_id` resolution for a brand-new identity happens at registration time (Kratos hooks) or at first consent (`apps/oauth-login` lookup). Both viable; ADR pending.
