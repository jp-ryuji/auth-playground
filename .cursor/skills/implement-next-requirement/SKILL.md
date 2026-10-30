---
name: implement-next-requirement
description: >-
  Implement one pending auth-playground requirement and stop. Use when the
  user asks to implement the next requirement, continue the signup loop, work
  the next SIGNUP ID, or run the implementation loop.
---

# Implement the next requirement

One requirement per run. `make test` stays green while later tests call `pending()` (`t.Skip`). That is not done.

## Steps

1. Run `make next`. If it prints `NEXT none`, stop. If `docs/specs/10-flows/signup-first-login.md` is still Draft, promote it to Stable in this change.
2. Read `docs/loops/state.md`. If its Next ID differs from `make next`, trust `make next`.
3. Read that requirement in the spec. If `make next` prints a `plan:` path, read that plan too.
4. Implement only that ID. Leave later IDs on `pending()`.
5. Replace that ID's `pending()` call with assertions that fail when the requirement is violated. Do not delete the test.
6. Run the printed proof command (`make prove PROVE_ID=...`). A skip is a failure.
7. Run `make test`.
8. Check the diff against the requirement: the test still quotes the requirement, the diff does not implement the following ID, and `pending()` is gone only for this ID.
9. Update `docs/loops/state.md` using the template below. Set Next from a fresh `make next`.
10. Stop. Do not start the following ID.

Parallel work uses a separate git worktree and branch, one requirement each.

## State file

```markdown
# Loop state

Bookmark between runs. The contract is `docs/specs/`. If this file disagrees with `make next`, trust `make next` and correct this file.

## Done

- ID — one line on what the test now proves

## Next

- ID: <from make next>
- Spec: <path>
- Plan: <path, omit if make next printed none>
- Proof: make prove PROVE_ID=<id>

## Blocked

- None.

## Last run

- Command: `make prove PROVE_ID=<id you just finished>`
- Result: passed or failed, plus anything the next run must not retry
```
