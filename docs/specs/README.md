# Specs

Spec-driven development for `auth-playground`. The repo [README.md](../../README.md) is the high-level pitch; **this directory is the design contract**. Code in `apps/*` is expected to satisfy the requirements stated here, and tests should map to the acceptance criteria each spec defines.

## How to read these

- **Hybrid style.** Architecture and rationale are prose. Contracts (tokens, claims, endpoints, security boundaries) use **RFC 2119** keywords — **MUST**, **SHOULD**, **MAY** — in bold uppercase.
- **Requirement IDs.** Normative statements have stable IDs like `OV-01` (overview) or `SIGNUP-03` (sign-up flow). Tests, code comments, and ADRs reference these IDs so a requirement change is traceable.
- **Status.** Each spec lists `Status: Draft | Stable | Deprecated`. Draft = wording may move; Stable = implementation is expected to match; Deprecated = superseded, see linked replacement.
- **Cross-links.** Specs link by file path. Requirement IDs link to the anchor (`#ov-04`) where they're defined.

## Layout

```text
docs/specs/
  README.md                       this file
  glossary.md                     shared vocabulary
  00-architecture/
    system-overview.md            components, trust boundaries, three-grant rule
  10-flows/
    signup-first-login.md         end-to-end via Hydra + Kratos
    ...                           returning-user login, logout, M2M, token exchange
  20-services/                    per-service contracts (deferred)
  30-cross-cutting/               discovery/JWKS, sessions, multi-tenancy (deferred)
  40-adr/                         architecture decision records (deferred)
```

Folders are numbered so file listings show the natural read order: architecture → flows → services → cross-cutting → ADRs.

## Index

| Spec | Status | Summary |
| ---- | ------ | ------- |
| [glossary.md](./glossary.md) | Draft | Shared terms (OIDC, OAuth, Kratos- and Hydra-specific). |
| [00-architecture/system-overview.md](./00-architecture/system-overview.md) | Draft | Component model, trust boundaries, the three-grant separation rule. |
| [10-flows/signup-first-login.md](./10-flows/signup-first-login.md) | Draft | Authorization Code + PKCE through Hydra and Kratos to a signed-in RP session. |
| [10-flows/returning-user-login.md](./10-flows/returning-user-login.md) | Draft | Delta against sign-up: product-session, Kratos-session, and remembered-consent short-circuits. |
| [10-flows/m2m-client-credentials.md](./10-flows/m2m-client-credentials.md) | Draft | Outbound M2M grant with `private_key_jwt`, audience-scoped tokens, server-side cache. |
| [10-flows/token-exchange-rfc8693.md](./10-flows/token-exchange-rfc8693.md) | Draft | On-behalf-of-user RFC 8693 exchange; server-held subject tokens, invalidate on session end. |
| [30-cross-cutting/tokens-claims-audiences.md](./30-cross-cutting/tokens-claims-audiences.md) | Draft | Token taxonomy, audience/scope rules, server-sourced claims, RS validation order, lifetime defaults. |

## Spec template

````markdown
# <Title>

Status: Draft
Last updated: YYYY-MM-DD
Scope: <one sentence — what this spec decides>
References: <RFCs, vendor docs, sibling specs>

## Purpose
Why this spec exists; what reading it should let you do.

## Out of scope
What this spec deliberately does not decide.

## Requirements
Normative MUST/SHOULD/MAY statements with stable IDs (`<PREFIX>-NN`).

## Interfaces
Endpoints, env vars, claims, cookies — the observable contract.

## Acceptance criteria
Testable statements; each links back to one or more requirement IDs.

## Open questions
Decisions deferred; promote to an ADR under `40-adr/` when resolved.
````

## Conventions

- Diagrams use Mermaid (GitHub renders it inline).
- Endpoint names that are not yet decided are marked `(TBD)` and tracked as open questions, not invented.
- Where the README and a spec disagree, the spec wins; open a PR to update the README in the same change.
