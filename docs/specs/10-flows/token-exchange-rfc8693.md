# Flow: Token exchange (RFC 8693)

Status: Draft
Last updated: 2026-05-12
Scope: `apps/api` exchanging a user-session token at Hydra's token endpoint for a **new** access token suitable for a **downstream** API — different `aud`, `scope`, or token shape. Covers on-behalf-of-user delegation.

References:
- [RFC 8693](https://www.rfc-editor.org/rfc/rfc8693) — OAuth 2.0 Token Exchange
- [system-overview.md](../00-architecture/system-overview.md), [glossary.md](../glossary.md)
- [m2m-client-credentials.md](./m2m-client-credentials.md), [tokens-claims-audiences.md](../30-cross-cutting/tokens-claims-audiences.md)

## Purpose

Pin down how `apps/api` calls a downstream API **on behalf of the signed-in user** without exposing the browser-session tokens to the downstream service. This is the third of the three grants the rule [OV-04](../00-architecture/system-overview.md#ov-04)..[OV-07](../00-architecture/system-overview.md#ov-07) requires to stay separate.

## Out of scope

- Outbound calls without a user (use [m2m-client-credentials.md](./m2m-client-credentials.md)).
- Refresh-token rotation. Refresh is **not** an exchange substitute; this spec re-states the rule and enforces it ([OV-07](../00-architecture/system-overview.md#ov-07)).
- Actor-token semantics for service-to-service-on-behalf-of-service delegation. Captured as an open question; not used in the initial deployment.

## Sequence

```mermaid
sequenceDiagram
  autonumber
  actor U as User
  participant W as apps/web
  participant BFF as apps/api (BFF / RP)
  participant SessStore as user-session token store
  participant ExchCache as exchanged-token cache
  participant H as ORY Hydra (token endpoint)
  participant DS as Downstream API (RS)

  U->>W: Action triggering downstream call
  W->>BFF: POST /resource (with session cookie)
  BFF->>SessStore: get session access token (subject = identity_id)
  BFF->>ExchCache: lookup(subject, audience, scope)

  alt Cache hit, not near expiry
    ExchCache-->>BFF: cached exchanged token
  else Miss
    BFF->>H: POST /oauth2/token<br/>grant_type=urn:ietf:params:oauth:grant-type:token-exchange<br/>subject_token=<session AT>, subject_token_type=access_token<br/>audience=<downstream>, scope=<least-priv><br/>client_assertion (exchange client)
    H-->>BFF: access_token (JWT) for downstream, expires_in
    BFF->>ExchCache: store(subject, audience, scope, token, exp - skew)
  end

  BFF->>DS: GET /resource<br/>Authorization: Bearer <exchanged token>
  DS->>DS: Validate JWT (sig via JWKS, iss, aud, exp); check sub
  DS-->>BFF: response
  BFF-->>W: response
```

## Requirements

### Client identity

- <a id="exch-01"></a>**EXCH-01.** Token-exchange requests **MUST** use a Hydra OAuth client registered **exclusively** for exchange. This client **MUST NOT** be reused for interactive RP login or `client_credentials` ([OV-04](../00-architecture/system-overview.md#ov-04)).
- <a id="exch-02"></a>**EXCH-02.** Client authentication for the exchange client follows the same rules as M2M: `private_key_jwt` in production; `client_secret_basic` only in development ([M2M-02](./m2m-client-credentials.md#m2m-02)).

### Request shape

- <a id="exch-03"></a>**EXCH-03.** The request **MUST** target `token_endpoint` from discovery with `grant_type=urn:ietf:params:oauth:grant-type:token-exchange`.
- <a id="exch-04"></a>**EXCH-04.** `subject_token` **MUST** be the **server-held** access token of the user's interactive session (resolved from the session store on `apps/api`). It **MUST NOT** be sourced from request input, headers, or cookies on the request being handled.
- <a id="exch-05"></a>**EXCH-05.** `subject_token_type` **MUST** be `urn:ietf:params:oauth:token-type:access_token` for the initial deployment. Other subject-token types are deferred to a separate decision.
- <a id="exch-06"></a>**EXCH-06.** `audience` **MUST** be set to the target downstream API identifier and **MUST** match a single audience ([OV-06](../00-architecture/system-overview.md#ov-06)).
- <a id="exch-07"></a>**EXCH-07.** `scope` **MUST** be the least-privilege scope required at the downstream API. It **MUST NOT** widen beyond the scopes carried by the `subject_token`.

### Lifecycle and storage

- <a id="exch-08"></a>**EXCH-08.** Exchanged tokens **MUST** be cached server-side, keyed by `(subject, audience, scope)`. The cache **MUST** be distinct from user-session token storage and from M2M token storage ([OV-05](../00-architecture/system-overview.md#ov-05)).
- <a id="exch-09"></a>**EXCH-09.** Cache TTL **MUST** be at most `expires_in - safety_skew` (`safety_skew ≥ 60s`). Exchanged-token lifetimes **SHOULD** be shorter than M2M and session access-token lifetimes ([TCA-06](../30-cross-cutting/tokens-claims-audiences.md#tca-06)).
- <a id="exch-10"></a>**EXCH-10.** When the user session ends (logout, revoke, password change, idle timeout), all cached exchanged tokens for that `subject` **MUST** be invalidated.
- <a id="exch-11"></a>**EXCH-11.** Exchanged tokens **MUST NOT** be sent to the browser, written to `apps/web`-reachable storage, or logged ([OV-08](../00-architecture/system-overview.md#ov-08)).

### Not refresh, not M2M

- <a id="exch-12"></a>**EXCH-12.** Code paths that need a downstream-API token for a user **MUST** call token exchange. They **MUST NOT** call refresh as a substitute, and **MUST NOT** call `client_credentials` "and trust the application to inject user context" ([OV-07](../00-architecture/system-overview.md#ov-07)).
- <a id="exch-13"></a>**EXCH-13.** Code paths that need a downstream-API token without a user **MUST** call `client_credentials` ([m2m-client-credentials.md](./m2m-client-credentials.md)). They **MUST NOT** synthesize a "service subject" via token exchange.

## Interfaces

### Outbound request

```http
POST {token_endpoint}
Content-Type: application/x-www-form-urlencoded

grant_type=urn:ietf:params:oauth:grant-type:token-exchange
&subject_token={user session access token}
&subject_token_type=urn:ietf:params:oauth:token-type:access_token
&audience={downstream-api-id}
&scope={least-privilege-scope}
&client_assertion_type=urn:ietf:params:oauth:client-assertion-type:jwt-bearer
&client_assertion={signed-jwt for exchange client}
```

### Configuration

Same shape as M2M ([m2m-client-credentials.md](./m2m-client-credentials.md#configuration-in-appsapi)) but a distinct client id + key set:

| Setting | Purpose |
| ------- | ------- |
| `OAUTH_EXCHANGE_CLIENT_ID` | Hydra OAuth client id for exchange only |
| `OAUTH_EXCHANGE_PRIVATE_KEY` | Signing key for the exchange client |
| `OAUTH_EXCHANGE_AUDIENCES` | Allowed downstream audiences for exchange |

## Acceptance criteria

- [ ] A user-context handler successfully calls a downstream API; the exchanged token has `aud = <downstream>`, `sub = <Kratos identity id>`, and `client_id = <exchange client>`. (EXCH-01, EXCH-04, EXCH-06)
- [ ] The exchange client id is rejected by Hydra if used with `grant_type=client_credentials` (and vice versa). Enforces [OV-04](../00-architecture/system-overview.md#ov-04). (EXCH-01, EXCH-13)
- [ ] Requesting `scope` widening beyond the session access token's scopes returns an error from Hydra. (EXCH-07)
- [ ] Logging out a user invalidates all cached exchanged tokens for that subject. A subsequent call requires a fresh exchange. (EXCH-10)
- [ ] Exchanged tokens are never present in any HTTP response body to `apps/web` or in any cookie. (EXCH-11)
- [ ] An attempt to call refresh in place of exchange for a downstream call either fails audience validation at the downstream API or is rejected by an early `apps/api` validation. (EXCH-12)
- [ ] Under concurrent requests for the same `(subject, audience, scope)`, the exchange call is single-flighted and at most one token-endpoint call is made per cache miss. (EXCH-08)

## Open questions

- Actor-token use cases (delegation chain: M2M acting on behalf of user). Possible for support tooling; defer until a concrete scenario lands.
- Whether `requested_token_type` is explicitly set, or left to Hydra's default. Default: omit and rely on Hydra config.
- Whether the exchange cache key includes a hash of additional context (e.g. tenant_id) when the downstream API is multi-tenant aware. Likely yes; specify in [tokens-claims-audiences.md](../30-cross-cutting/tokens-claims-audiences.md).
