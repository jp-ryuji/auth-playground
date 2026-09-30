// Package signup is the OIDC sign-up / first-login flow on apps/api (the BFF / RP).
//
// Contract: docs/specs/10-flows/signup-first-login.md (SIGNUP-01..14).
// System context: docs/specs/00-architecture/system-overview.md (OV-01..14).
//
// Status: SIGNUP-01..07 have real tests. SIGNUP-08..14 and the acceptance
// scenarios still call pending() (t.Skip). `make next` prints the first of
// those. When no pending() calls remain, promote signup-first-login.md from
// Draft to Stable in that same change.
package signup
