# Tokens, claims, and audiences

Status: Draft
Last updated: 2026-05-12
Scope: the contract for tokens flowing between `apps/api`, Hydra, and downstream resource servers — token taxonomy, claim shape, audience and scope rules, validation order, lifetime defaults. Sits below [system-overview.md](../00-architecture/system-overview.md) and is referenced by every flow spec under `10-flows/`.

References:
- [RFC 7519](https://www.rfc-editor.org/rfc/rfc7519) — JWT
- [RFC 9068](https://www.rfc-editor.org/rfc/rfc9068) — JWT Profile for OAuth 2.0 Access Tokens
- [RFC 9700](https://www.rfc-editor.org/rfc/rfc9700) — OAuth 2.0 Security BCP
- [system-overview.md](../00-architecture/system-overview.md), flow specs in `10-flows/`

## Purpose

Collect the cross-cutting rules that the three grants individually reference: where each token type may live, which `aud` and `scope` it may carry, which claims a resource server is allowed to trust, what validation looks like at the receiving end, and how long tokens are valid by default. The three-grant separation rule from [OV-04](../00-architecture/system-overview.md#ov-04)..[OV-07](../00-architecture/system-overview.md#ov-07) is enforced here in concrete terms.

## Out of scope

- Cookie format and product-session shape (separate spec: `sessions-and-cookies.md`).
- JWKS rotation mechanics (separate spec: `discovery-and-jwks.md`).
- Policy-engine integration with claim values (deferred).

## Token taxonomy

| Token | Issuer | Holder | Consumer | Trust zone |
| ----- | ------ | ------ | -------- | ---------- |
| ID token (user) | Hydra | `apps/api` only | `apps/api` (RP) | Server-side. Used to determine who signed in. |
| Access token (user session) | Hydra | `apps/api` only | `apps/api` RS code paths, or input to RFC 8693 exchange | Server-side. Never sent to the browser. |
| Refresh token (user session) | Hydra | `apps/api` only | `apps/api` token endpoint calls | Server-side. Rotated; never sent to the browser. |
| Access token (M2M `client_credentials`) | Hydra | `apps/api` background job | Downstream RS | Server-side. Separate cache from user tokens. |
| Access token (RFC 8693 exchange) | Hydra | `apps/api` user-context handler | Downstream RS | Server-side. Separate cache; invalidated on session end. |
| Product session cookie | `apps/api` | Browser | `apps/api` | Browser-readable as opaque blob. Not a JWT. Not derived from any Hydra token. |

> The product session cookie is **not** a token in the OAuth sense; it is `apps/api`'s own session identifier. It appears here only to make explicit that the browser receives **no** OAuth/OIDC token.

## Requirements

### Token format

- <a id="tca-01"></a>**TCA-01.** All Hydra-issued access tokens used in this repo **MUST** be JWTs verifiable against the JWKS from discovery ([OV-01](../00-architecture/system-overview.md#ov-01)). Opaque access tokens **MUST NOT** be issued by Hydra for this design.
- <a id="tca-02"></a>**TCA-02.** Access-token JWTs **SHOULD** follow the JWT Profile for OAuth 2.0 (RFC 9068): `typ = at+jwt`, mandatory `iss`, `exp`, `aud`, `sub`, `client_id`, `iat`, `jti`.

### Audience (`aud`)

- <a id="tca-03"></a>**TCA-03.** Every access token **MUST** carry a single intended audience or a closed list of audiences declared on the issuing Hydra OAuth client. Wildcard or runtime-widened audiences are **NOT** permitted.
- <a id="tca-04"></a>**TCA-04.** Each grant uses a distinct audience: interactive RP-internal calls (when applicable), M2M downstream audiences, and exchange downstream audiences are non-overlapping ([OV-06](../00-architecture/system-overview.md#ov-06)). A downstream API **MUST** be reachable via either M2M *or* exchange — never both audiences on the same token.

### Scope

- <a id="tca-05"></a>**TCA-05.** `scope` requested at the token endpoint **MUST** be the least privilege required by the calling code path. Allowed scopes per (client, audience) **MUST** be declared in `apps/api` configuration and reviewed in code review.
- <a id="tca-06"></a>**TCA-06.** Resource servers **MUST** enforce scope **after** signature and `aud` validation. A valid signature with insufficient scope yields `403`, not `200`.

### Claims sourced server-side

- <a id="tca-07"></a>**TCA-07.** Custom claims placed into ID tokens or access tokens by `apps/oauth-login` (or by Hydra session-claim hooks) **MUST** be derived from server-trusted state — Kratos identity traits, tenant lookup tables, role assignments. Values that originate in browser-controlled inputs (query params, form fields, cookies) **MUST NOT** be propagated into token claims.
- <a id="tca-08"></a>**TCA-08.** `tenant_id`, when present in a token, **MUST** be resolved from the authenticated subject's tenant assignment at issuance time. It **MUST NOT** be selectable by the client at the authorize request.
- <a id="tca-09"></a>**TCA-09.** Role and permission claims (`org_role`, `permissions`, …) **MUST** reflect the assignment **at issuance time**. Resource servers **MUST NOT** cache role/permission claims beyond the token's `exp`.

### Validation order at the RS

- <a id="tca-10"></a>**TCA-10.** A resource server (including `apps/api` acting as RS) **MUST** validate access tokens in this order, failing closed at the first error:
  1. JWT signature against JWKS from discovery
  2. `iss` matches configured issuer
  3. `exp` and `nbf`/`iat` within skew (≤ 120s recommended)
  4. `aud` includes this RS's identifier
  5. `scope` covers the operation
  6. Tenant-scoped authorization ([OV-13](../00-architecture/system-overview.md#ov-13))
  7. Resource-specific RBAC/ABAC/ReBAC checks
- <a id="tca-11"></a>**TCA-11.** Introspection (RFC 7662) **MAY** supplement local JWT validation when the issuer advertises `introspection_endpoint`. When used, the introspection result **MUST NOT** widen what local validation rejected; introspection only **further restricts** acceptance.

### Lifetime defaults

- <a id="tca-12"></a>**TCA-12.** Default token lifetimes (deployment-tunable; values below are the playground baseline):

  | Token | Default lifetime |
  | ----- | ---------------- |
  | User-session ID token | ≤ 1 h |
  | User-session access token | ≤ 1 h |
  | User-session refresh token | ≤ 30 d, with rotation |
  | M2M access token (`client_credentials`) | ≤ 1 h |
  | Exchanged access token (RFC 8693) | ≤ 15 min |
  | Product session cookie | ≤ 24 h absolute, ≤ 1 h idle |

- <a id="tca-13"></a>**TCA-13.** Refresh-token rotation **MUST** be enabled. A used refresh token **MUST** be invalidated on rotation; reuse **MUST** revoke the entire session family.

### Storage and propagation

- <a id="tca-14"></a>**TCA-14.** Each of the four server-side token kinds — user-session access, user-session refresh, M2M access, exchanged access — **MUST** live in its **own** keyed store. Sharing storage between any two kinds is **NOT** permitted ([OV-05](../00-architecture/system-overview.md#ov-05)).
- <a id="tca-15"></a>**TCA-15.** Hydra-issued tokens **MUST NOT** be written to: HTTP responses delivered to the browser, browser cookies, browser-readable headers, application logs at info level or above, or third-party telemetry. Debug logs at trace level **MAY** include token id (`jti`) but not the token value.

## Claim shape (example)

The interactive-session access token issued by Hydra is expected to carry at least:

```json
{
  "iss": "https://hydra.example.com",
  "sub": "<kratos-identity-id>",
  "aud": ["https://api.example.com"],
  "client_id": "interactive-rp",
  "exp": 1731000000,
  "iat": 1730996400,
  "jti": "<unique-id>",
  "scope": "openid profile document:read document:write",
  "tenant_id": "tenant-a",
  "org_role": "admin",
  "permissions": ["document:read", "document:write"]
}
```

`tenant_id`, `org_role`, and `permissions` are illustrative; the exact custom-claim contract is decided per release and documented under `30-cross-cutting/multi-tenancy.md` (to be written) and the authorization spec (to be written). They are bound by [TCA-07](#tca-07)..[TCA-09](#tca-09).

## Acceptance criteria

- [ ] A test enumerates the four server-side token stores and asserts each token kind is found only in its own store. (TCA-14)
- [ ] A token with valid signature but missing `aud` is rejected by the RS. (TCA-10 step 4)
- [ ] A token with valid signature and matching `aud` but insufficient `scope` produces `403`, not `200`. (TCA-06)
- [ ] A claim derived from a tampered or browser-supplied input does not appear in any issued token. (TCA-07, TCA-08)
- [ ] Refresh-token reuse (replaying the previous refresh after rotation) revokes the session family and forces re-login. (TCA-13)
- [ ] No token-value strings appear in INFO/WARN/ERROR logs or in `apps/web` network traces. (TCA-15)
- [ ] Removing or rotating a JWKS key invalidates tokens signed with the removed key as soon as the discovered JWKS no longer advertises it (within cache TTL). (TCA-10 step 1; full mechanics in `discovery-and-jwks.md`)

## Open questions

- Whether `tenant_id` is required on **every** access token or only on tokens whose audience is tenant-aware. Likely required when present in the user's effective context; specify in the authorization spec.
- Whether `permissions` is a flat list or structured (e.g. `{ resource: action[] }`). Flat is simpler for v1; structured is friendlier to policy engines.
- Whether exchanged tokens carry `act` (actor) claims when the call originates from a support/admin path. Defer until the support path spec exists.
