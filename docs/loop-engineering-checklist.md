# Loop engineering checklist

Use this before an agent loop runs in a repository. A loop finds work, changes code, checks the result, writes down what happened, and decides the next step. The repository has to make each of those steps obvious. A prompt cannot substitute for them.

Score each item **present**, **partial**, or **missing**. Start an unattended loop only when every item in "Harness" and "Stop condition" is present.

## 1. Name the loop

- [ ] One sentence says what this loop is for (for example: "implement the next requirement ID until its test passes").
- [ ] The work queue lives in the repo (spec IDs, issues, or a task file), not in a chat transcript.
- [ ] One run does one unit of work. The unit is small enough to review (one requirement, one bug, one PR).
- [ ] The loop has an owner who reads the diff. Shipping stays a human decision.
- [ ] Token budget is stated: how often it runs, and whether it may open pull requests or only report.

## 2. Harness

The harness is the environment one agent runs inside. The loop sits on top of it.

- [ ] A cold-start doc (`AGENTS.md`, `CLAUDE.md`, or equivalent) lists the real commands, layout, and conventions. Every command in that doc exists in the repo.
- [ ] One entry point runs the checks (`make test`, `npm test`, or a single script). A second command runs the check for the current unit only.
- [ ] The same commands pass locally and in CI.
- [ ] Dependencies install from lockfiles. Tool versions are pinned. CI actions are pinned to commit SHAs when the repo claims they are.
- [ ] A new agent can boot the system: env example, seed or migrate step, and a readiness command (`make ready` or equivalent).
- [ ] Secrets are not required for the default check. Live or credentialed checks are a separate command, off by default.
- [ ] Lint and format are part of the check the loop runs, or explicitly deferred with a reason.
- [ ] The design contract and the README agree. Where they disagree, one file is declared the winner and the other points at it.
- [ ] Status comments match the code. A header that says "nothing is implemented" while tests exist will make the next run redo finished work.

## 3. Verifiable stop condition

This is the item that makes the loop safe to leave.

- [ ] "Done" is a command outcome, not the agent's summary. Example: `TestFeature_REQ_08` passes.
- [ ] A skipped or pending test is not a pass for the loop's goal. CI may stay green for a scaffold; the loop's goal command must fail while the current unit is unfinished.
- [ ] The goal names the next unit only. Passing that unit does not authorize starting the one after it.
- [ ] The checker is separate from the author. Tests, a review skill, or a second agent re-reads the requirement and the diff. The author does not delete or weaken the check to go green.
- [ ] Failure output names the unit and the spec or ticket it belongs to.

## 4. Memory

The model forgets between runs. The repo must not.

- [ ] A state file outside the chat records done, next, blocked, and the last command result (for example `docs/loops/state.md`).
- [ ] The loop reads that file at the start of a run and updates it at the end.
- [ ] Specs or tickets remain the contract. The state file is only a bookmark.
- [ ] Tried approaches that failed are written down so the next run does not repeat them.

## 5. Skills

- [ ] Project knowledge the agent would otherwise guess is in a skill or in `AGENTS.md`: how to pick the next unit, which files it may touch, which command proves it, when to stop.
- [ ] The skill is scoped to this loop. A generic "be helpful" instruction is not a skill.
- [ ] Build, test, and "we do not do it this way" rules are written once and reused by every run.

## 6. Isolation

- [ ] Parallel runs use separate git worktrees or branches. Two agents do not share one working tree.
- [ ] Each run's branch maps to one unit of work.
- [ ] The repo documents the worktree convention if more than one agent will run.

## 7. Automations

Add a schedule only after sections 2 and 3 are present.

- [ ] The trigger is explicit: a cadence (`/loop`, cron, GitHub Actions schedule) or a condition that stays true until the stop command passes (`/goal`).
- [ ] The scheduled prompt calls the skill. It does not paste a long one-off instruction that will rot.
- [ ] Runs that find nothing archive themselves. Runs that find work land where a human will see them.
- [ ] A human can stop the loop (unsubscribe, kill the watcher, disable the workflow) without editing product code.

## 8. Connectors

- [ ] A filesystem-only loop is enough when the work queue and the checks are in the repo.
- [ ] MCP or other connectors are added only for a system the loop must read or write (issue tracker, CI, chat). Each connector has a purpose named in the skill.
- [ ] Connectors that can open pull requests, post messages, or change tickets are limited to the rollout level below.

## 9. Rollout

Promote one level at a time. Stay at a level until the checker has been right across several real runs.

- [ ] **L1 report.** The loop reads the queue and writes findings. It does not edit product code.
- [ ] **L2 assisted.** The loop edits on a branch and opens a review. A human merges.
- [ ] **L3 unattended.** The loop merges or deploys only for a class of change whose check has been trustworthy. Everything else stays at L2.

## 10. Leave these out until the loop is already working

- [ ] A second work queue (a ticket board that duplicates the specs).
- [ ] A JavaScript or other task runner that wraps a single real command and can fail a different way.
- [ ] Empty directories for future docs.
- [ ] Product surfaces the current loop does not need (a UI, a new service).
- [ ] A scheduled "fix CI" or "triage everything" loop while the stop condition is still soft.
- [ ] Subagent definitions that repeat the same instructions as the skill. Add a second agent when the maker must not grade its own result and the tests are not enough.

## Go / no-go

| Gate | Required to start |
| --- | --- |
| L1 report | Sections 1, 4, and 5 present. The loop only writes the state file. |
| L2 assisted | Sections 2 and 3 present as well. One unit per branch. Human merges. |
| L3 unattended | Section 7 present. Section 3 has been right for repeated L2 runs. Budget and stop switch exist. |
