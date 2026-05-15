# Glossary

Status: Draft
Last updated: 2026-05-12
Scope: shared vocabulary used across `docs/specs/`. Defines wording, not behavior.

Terms here intentionally restate external definitions (OIDC, OAuth 2.0, RFC 8693, ORY docs) **in the context of this repo**. Where a term is narrower here than in the upstream spec, that is called out explicitly.

## Identity (Kratos)

| Term | Meaning here |
| ---- | ------------ |
| **Identity** | A user record in **ORY Kratos**. Its stable id is used as the **subject** when `apps/oauth-login` accepts a Hydra login challenge. |
| **Traits** | Structured profile fields on a Kratos identity (e.g. email, name), defined by the identity schema. Not OAuth scopes; not access-token claims (until the BFF chooses to project them). |
| **Credentials** | How a user proves identity to Kratos (password, OIDC link, …). Stored and verified by Kratos; **distinct** from Hydra OAuth client secrets. |
| **Self-service flow** | Browser-driven Kratos flow (login, registration, recovery, verification, settings). Implemented against Kratos public APIs by the official **Self-Service UI**. |
| **Kratos session** | A Kratos-managed session for an identity, established after a self-service flow completes. Used by `apps/oauth-login` to know **who** the user is before accepting a Hydra login challenge. |

## OAuth 2.0 / OIDC (Hydra side)

| Term | Meaning here |
| ---- | ------------ |
| **OP** (OpenID Provider) | The OIDC role played by **ORY Hydra**. Issues ID tokens and access tokens for user logins. |
| **AS** (Authorization Server) | The OAuth 2.0 role of Hydra. Issues access tokens at the token endpoint for every grant used in this repo (code, `client_credentials`, RFC 8693 exchange). |
| **RP** (Relying Party) | The OIDC client. In this repo the product RP is **`apps/api`**. |
| **RS** (Resource Server) | The API that accepts and validates access tokens. In this repo, also **`apps/api`** — RP and RS share a process. |
| **BFF** (Backend for Frontend) | The server that holds the RP session, owns cookies, and brokers calls between `apps/web` and downstream APIs. Also `apps/api`. |
| **Subject** | OIDC `sub` claim. In this repo the subject is the **Kratos identity id**, set by `apps/oauth-login` when accepting the Hydra login challenge. |
| **Audience** (`aud`) | The intended recipient of a token, validated at the RS. The three grants in this repo produce tokens with **distinct** audiences. |
| **Scope** | OAuth scope string. Defines what an access token authorizes a client to do at a given audience. |
| **Claim** | A name/value pair inside a token. ID-token claims describe the user; access-token claims describe authorization. |
| **Discovery document** | The JSON returned at `{issuer}/.well-known/openid-configuration`. Source of truth in this repo for `authorization_endpoint`, `token_endpoint`, `jwks_uri`, and any optional endpoints Hydra advertises. |
| **JWKS** | JSON Web Key Set served at `jwks_uri`. The RS verifies JWT signatures against keys from JWKS with cache + rotation handling. |
| **Login challenge** | Opaque id issued by Hydra when it redirects the browser to the login URL. `apps/oauth-login` exchanges it via Hydra Admin API for the login request, then accepts or rejects. |
| **Consent challenge** | Equivalent for the consent step: Hydra → `apps/oauth-login` → Hydra Admin API accept/reject. |

## Grants used at Hydra's token endpoint

| Grant | Purpose in this repo |
| ----- | -------------------- |
| **Authorization Code + PKCE** | Interactive user login from `apps/web` via `apps/api`. Produces ID token, access token, refresh token bound to the user session. |
| **Refresh token rotation** | Renews the access token of an existing interactive session. **Not** a substitute for token exchange. |
| **`client_credentials`** | `apps/api` calls downstream APIs **as itself** when no end user is in scope (M2M). Uses a Hydra OAuth client **separate** from the interactive RP. |
| **RFC 8693 token exchange** (`urn:ietf:params:oauth:grant-type:token-exchange`) | `apps/api` exchanges an issued token for a new access token with different `aud`/`scope`/shape, e.g. to call a downstream API on behalf of the user. **Distinct** from refresh; **distinct** from `client_credentials`. |

## Multi-tenancy and authorization

| Term | Meaning here |
| ---- | ------------ |
| **Tenant** | A logical isolation unit (org, workspace, …). Surfaced as `tenant_id` in claims and propagated through authz checks. |
| **RBAC** | Role-based access control. Application-layer roles (admin, member, viewer) checked at `apps/api`. |
| **ABAC** | Attribute-based access control. Authz uses attributes like `tenant_id`, `department`, `resource_owner`, `environment`. |
| **ReBAC** | Relationship-based access control. Authz follows graph relations (ownership, org tree, team membership). |

## Process layout

| Term | Meaning here |
| ---- | ------------ |
| **`apps/web`** | Next.js thin UI; no OAuth state. Talks to `apps/api` via session cookies. |
| **`apps/api`** | Go BFF; RP + RS + M2M confidential client. Holds session cookies and refresh tokens server-side. |
| **`apps/oauth-login`** | Go login/consent app for Hydra. Drives Hydra Admin API after Kratos has authenticated the user. Owns Hydra Admin credentials. |
| **Kratos Self-Service UI** | Separate process running ORY's reference Node UI for Kratos browser flows. |
