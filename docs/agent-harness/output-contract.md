# AviaSurveil360 Agent Harness Output Contract

Use this contract for status, readiness, implementation closeout, and plan
lifecycle responses. It keeps local evidence, demo scope, and production gaps
separate.

## Required Final Readout

For status, readiness, or execution closeout, use this shape when a structured
readout is useful:

```text
Done:
Remaining:
Blocked:
Verification:
Next:
```

Keep each section short and evidence-bound. If a command was not run, say
`not run` and why. If a check failed and cannot be fixed in the current task,
say `blocked` and record the durable note required by `../../AGENTS.md`.

## Status Labels

- `verified locally` - the stated local commands or browser checks ran in this
  task and passed.
- `blocked` - required evidence, access, environment, or source-owner input is
  unavailable and the gap cannot be resolved in the current task.
- `not run` - a verification step was skipped or unavailable; the reason must
  be stated.
- `ready-for-verification` - implementation or docs are in place and local
  checks are complete, but stakeholder/user sign-off or another explicit review
  remains before moving to `completed`.
- `demo-only` - behavior is static, mock, browser-local, or source-shaped for
  stakeholder feedback.
- `source-bound` - the result depends on a verified source document, source
  owner, or external authority input.
- `production-readiness not claimed` - local/demo proof does not establish
  production security, regulatory, evidence, notification, audit-log, mobile,
  offline, reporting, or operational readiness.

## Evidence Rules

- Claims about passing checks require fresh command output from the current
  task.
- Claims about visible UI require local browser or screenshot evidence when the
  task changes a user-facing workflow.
- Claims about product rules require a source doc, smoke test, or explicit plan
  reference.
- Do not use chat-only notes as durable evidence when the repo has a matching
  plan, build summary, note, or test surface.
- If evidence is partial, name the exact coverage gap.

## Plan Lifecycle Output Rules

When a task touches `../exec-plans/`:

- Update `../exec-plans/index.md` only when status or next todo actually changes.
- Keep one concrete next todo per active row.
- Do not move a plan to `../exec-plans/completed/` without inspected objective
  completion, required verification, and explicit stakeholder/user sign-off.
- Use `../exec-plans/tech-debt-tracker.md` for durable blockers, accepted
  risks, missing evidence, regulatory assumptions, or owner handoffs that need
  to survive the current plan.
- Keep the plan file and index row consistent before reporting completion.

## Browser And GUI Cleanup Reporting

If browser automation or GUI testing runs:

- use a temporary test profile where practical
- inspect for leftover Chrome, Chrome Helper, webdriver, Playwright, Puppeteer,
  or test-runner processes before reporting completion
- report cleanup status or any unresolved leftover process risk

This is not required for docs-only tasks that do not start browser automation.

## Forbidden Claims

Do not claim or imply that AviaSurveil360 has production security, real
authentication, real uploads, real AI, real regulatory ingestion, real
notification delivery, immutable audit-log behavior, production evidence
chain-of-custody, remote CI coverage, deployment readiness, or legal/regulatory
decision automation unless a future task adds matching implementation and
evidence.

For legacy static-demo harness work, the correct boundary is:

```text
demo-only; verified locally when local checks pass; production-readiness not claimed
```

For React/Go/PostgreSQL or first-party Auth candidate work, use the same
literal evidence labels but do not call the implemented candidate
frontend-only. It remains `candidate-only` and `release pending` until its
separate runtime and release evidence exists.
