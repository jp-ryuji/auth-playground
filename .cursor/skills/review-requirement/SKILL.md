---
name: review-requirement
description: >-
  Check one auth-playground requirement against its spec and test without
  editing code. Use when reviewing a SIGNUP, OV, M2M, EXCH, or RETURN change,
  or after implementing a single requirement ID.
---

# Review one requirement

Read the requirement and the diff. Do not edit files. Do not treat a green `make test` as proof while `pending()` remains for this ID.

## Checklist

- The diff implements the ID under review and no later ID.
- The test for this ID no longer calls `pending()`.
- The test fails if the requirement is violated. It does not assert a weaker stand-in.
- `make prove PROVE_ID=<id>` passes.
- Tokens, client secrets, and Hydra Admin response bodies are not exposed to the browser.
- Interactive, M2M, and token-exchange clients stay separate.

## Report

Lead with pass or fail. Name the requirement ID. List only the mismatches. A pass needs no extra suggestions.
