// Package signup is the OIDC sign-up / first-login flow on apps/api (the BFF / RP).
//
// Contract: docs/specs/10-flows/signup-first-login.md (SIGNUP-01..14).
// System context: docs/specs/00-architecture/system-overview.md (OV-01..14).
//
// Status: scaffold. signup_test.go enumerates every normative requirement with
// one t.Skip per ID so the requirement-to-test mapping is in place before any
// production code is written. Implementing the package means converting each
// t.Skip into a real assertion; when the suite is green, signup-first-login.md
// can be promoted from Draft to Stable in the same PR.
package signup
