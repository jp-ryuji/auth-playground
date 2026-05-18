# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
make up              # start full stack (Hydra, Kratos, Postgres, Self-Service UI, Mailpit)
make down            # stop containers (volumes preserved)
make test            # run all Go tests across the workspace
make test-signup     # run only the SIGNUP-NN suite (hermetic, no stack required)
make test-oauth-login # run only apps/oauth-login tests
make test-signup-live # run live tests against the running stack (requires make up)
make discovery       # curl Hydra's .well-known/openid-configuration
```

To run a single test:
```bash
cd apps/api && go test -v -run TestSignup_SIGNUP_01 ./internal/signup/...
```

## Architecture

The repo is a Go workspace (`go.work`) with two modules under `apps/`:

- **`apps/api`** — OIDC Relying Party + BFF + Resource Server. Owns the product session (opaque HttpOnly cookie), runs the Authorization Code + PKCE flow against Hydra, validates JWTs from Hydra's `jwks_uri`, and holds refresh tokens server-side. Also acts as a confidential OAuth client for `client_credentials` (outbound M2M) and RFC 8693 token exchange — these use **separate Hydra client registrations** from the interactive RP client.
- **`apps/oauth-login`** — stateless Hydra login/consent orchestrator. Receives `login_challenge` / `consent_challenge` from Hydra via browser redirect, calls the Hydra Admin API to accept/reject, then issues the `redirect_to` HTTP redirect. Holds Hydra Admin credentials; has no product session state.

The browser never sees OAuth tokens. `apps/web` (planned Next.js thin UI) holds only the opaque session cookie from `apps/api`. Kratos Self-Service UI is a separate ORY-provided process for identity flows (registration, login, recovery).

### Request flow (sign-up / first login)

```
Browser → apps/web → apps/api → Hydra /oauth2/auth
                                    ↓ login_challenge redirect
                              apps/oauth-login → Kratos Self-Service UI ↔ Kratos
                              apps/oauth-login → Hydra Admin (accept login)
                                    ↓ consent_challenge redirect
                              apps/oauth-login → Hydra Admin (accept consent)
                                    ↓ code redirect to apps/api /callback
apps/api exchanges code → tokens, sets session cookie
```

### Key packages

| Path | Role |
|------|------|
| `apps/api/internal/signup/` | Auth URL building, PKCE, state, nonce, callback handling — maps 1-to-1 to `SIGNUP-NN` requirement IDs |
| `apps/api/internal/oidc/` | OIDC discovery fetch and metadata cache |
| `apps/api/internal/config/` | Env-var config (`HYDRA_ISSUER`, `CLIENT_ID`, `REDIRECT_URI`, etc.) |
| `apps/oauth-login/hydra/` | Hydra Admin API client (get/accept login and consent requests) |
| `apps/oauth-login/login/` | HTTP handlers for login/consent challenge flow |

### Spec-driven development

`docs/specs/` is the **design contract**. Requirement IDs (`OV-NN`, `SIGNUP-NN`, etc.) appear in spec files, test names (`TestSignup_SIGNUP_01_*`), and code comments. When implementing a requirement, find its ID in `docs/specs/10-flows/signup-first-login.md`, replace the corresponding `t.Skip` in `signup_test.go` with real assertions, and promote the spec from Draft to Stable in the same PR when all its IDs are green.

Where `docs/specs/` and `README.md` disagree, the spec wins.

### Three-grant rule

`apps/api` uses three **distinct Hydra OAuth client registrations**, each with its own secrets and token lifecycle:

1. **Interactive RP** — Authorization Code + PKCE for browser login
2. **M2M** — `client_credentials` for outbound server-to-server calls
3. **Token exchange** — RFC 8693 for downstream-audience tokens

Never mix tokens or credentials across these three registrations.

## Security

### Pre-commit secret scanning

Each developer must run once per clone (from the repo root):
```bash
pip install pre-commit   # once per machine
pre-commit install       # once per clone, must be run from repo root
```

### CI

- `test.yml` — runs `make test` on push and PRs; Actions pinned to commit hashes.
- `security-audit.yml` — TruffleHog deep-history secret scan via `jp-ryuji/security-config` reusable workflow; also pinned to a commit hash.

To update pinned hashes: install `pinact` via `aqua i` then run `pinact run`.
