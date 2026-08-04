# AviaSurveil360 Agent Harness Readiness Completion Plan

> Execute this plan only after the user separately authorizes implementation.
> Steps use checkbox (`- [ ]`) syntax for tracking.

- **Date:** 2026-06-29
- **Status:** ready-for-verification
- **Owner:** Product engineering workflow
- **Type:** Agent harness documentation, output contract, verification guardrails
- **Authority:** Root `AGENTS.md`; `docs/exec-plans/` is the active plan folder.
- **Source boundary:** This plan is based on the local OpenAI Harness
  Engineering working note at
  `/Users/marlonjd/Downloads/openai-harness-engineering-tam-kapsamli.md`, the
  official source identity
  `https://openai.com/tr-TR/index/harness-engineering/`, the existing
  AviaSurveil360 harness runbook, and current repo evidence. Do not claim a
  fresh full-text extraction from the OpenAI page unless a future agent actually
  retrieves and reviews the page body.

**Goal:** Move AviaSurveil360 from a partial runbook-first harness adaptation to
a complete, canonical, repo-discoverable agent harness package.

**Current implementation note (2026-06-29):** The canonical
`docs/agent-harness/` package is implemented with `index.md`,
`output-contract.md`, `registry.md`, `verification-matrix.md`, and
`entropy-cleanup-checklist.md`. `tests/harness-docs-smoke.test.js` was added as
a direct local Node built-ins smoke gate. `AGENTS.md`, `MANIFEST.md`, and
`docs/demo-handoff/AGENT_HARNESS_RUNBOOK.md` now point future
agents to the canonical harness entrypoint. The older harness engineering
adaptation plan already carries a continuation note to this readiness plan and
remains `superseded`. Local checks passed for `git diff --check`,
`node tests/harness-docs-smoke.test.js`,
`node tests/demo-boundary-smoke.test.js`, the required `rg` link search, and
the `.github` workflow check. No GitHub Actions, hosted runner, remote CI, paid
automation, backend, database, API, real auth, real upload, real AI, real
regulatory ingestion, notification service, production audit-log behavior, or
framework migration was added or claimed. Remaining gate:
stakeholder/user sign-off before moving to `completed`.

**Maintenance note (2026-07-23):** The user explicitly authorized an
English-only documentation policy and an entropy cleanup pass. All 53 paired
`.turkce.md` companions were removed after confirming that each had an English
canonical counterpart. `AGENTS.md` was reduced from 318 to 121 lines and now
uses direct Markdown routes to the architecture, planning, plan-index,
registry, verification, output, and cleanup authorities. `CLAUDE.md` and the
legacy harness runbook were reduced to thin compatibility adapters.
`ARCHITECTURE.md` and `docs/PLANS.md` now own the missing system and planning
maps. Forward Plans 1–4 no longer request Turkish companion evidence. Fresh
local results: `node tests/harness-docs-smoke.test.js` passed,
`node tests/demo-boundary-smoke.test.js` passed, `git diff --check` passed, and
both adaptive `harness.py audit` and `harness.py check` returned 0 errors and 0
warnings. This is a verified local adaptive harness repair, not `CERT000`
certification or a production-readiness claim.

**Architecture:** Keep `AGENTS.md` as the short highest-authority map, then move
working harness detail into a dedicated `docs/agent-harness/` package. The
package must expose one entrypoint index, one output contract, one surface
registry, one verification matrix, and one entropy cleanup checklist, with a
small Node smoke gate that prevents the package from drifting out of sync with
`docs/exec-plans/index.md`.

**Tech Stack:** Markdown, existing static demo files, existing Vanilla
JavaScript smoke-test style, Node.js built-ins only. No package manager,
backend, database, API, CI service, real authentication, real upload, real AI,
real regulatory ingestion, notification service, production audit-log, or
framework migration.

**Cost and CI boundary:** Verification is local-only. Do not add GitHub Actions,
`.github/workflows`, hosted runners, remote CI, paid CI minutes, or any
automation that could create cost for this private repository.

---

## Objective

Make the AviaSurveil360 harness fully ready for future Codex/Claude work by
creating the missing canonical harness artifacts that a new agent can discover
without relying on chat memory:

1. `docs/agent-harness/index.md` as the harness index and first stop.
2. `docs/agent-harness/output-contract.md` as the required response/evidence
   contract.
3. `docs/agent-harness/registry.md` as the source, plan, evidence, test, and
   demo-surface registry.
4. `docs/agent-harness/verification-matrix.md` as the exact local verification
   ladder.
5. `docs/agent-harness/entropy-cleanup-checklist.md` as the maintenance and
   drift-control checklist.
6. `tests/harness-docs-smoke.test.js` as a deterministic guard that proves the
   expected files and index links exist.

This is a documentation-and-guardrail readiness plan. It does not change product
behavior or claim production readiness.

## Scope

### In Scope

- Create a dedicated `docs/agent-harness/` package even though the earlier
  adaptation used `docs/demo-handoff/AGENT_HARNESS_RUNBOOK.md`.
  This exception is intentional: the user explicitly identified the missing
  harness index/output surfaces, and a canonical package is the clearest fix.
- Keep `AGENTS.md` short and map-like by adding only a pointer to the new
  harness index if needed.
- Convert the current runbook content into a more discoverable structure:
  - entrypoint and routing in `index.md`
  - output/status/evidence rules in `output-contract.md`
  - source and artifact inventory in `registry.md`
  - command ladder in `verification-matrix.md`
  - cleanup/drift rules in `entropy-cleanup-checklist.md`
- Preserve the product specs and execution-plan lifecycle.
- Update `docs/demo-handoff/AGENT_HARNESS_RUNBOOK.md` so it points
  to the canonical harness package instead of being the only harness surface.
- Update `MANIFEST.md` so the harness package is visible in the repo inventory.
- Update `docs/exec-plans/index.md` so this plan is the active next harness todo.
- Add a Node smoke test that checks:
  - all required harness files exist
  - `docs/exec-plans/index.md` links this plan and the harness adaptation plan
  - `docs/agent-harness/index.md` links the output contract, registry,
    verification matrix, entropy cleanup checklist, runbook, plan index, and
    demo summary
  - `output-contract.md` contains the literal status labels `verified locally`,
    `blocked`, and `not run`
  - harness docs do not claim production readiness
- Keep every verification path local: direct `node` commands, `git diff --check`,
  `rg`, and local browser checks only when needed. Do not propose or add GitHub
  Actions.

### Out Of Scope

- No backend, database, API, real authentication, real authorization, real file
  upload, document storage, real email/SMS/notification service, real AI
  service, real regulatory ingestion, production audit log, production evidence
  chain-of-custody, production security readiness, CI service, autonomous merge
  bot, or framework migration.
- No GitHub Actions, `.github/workflows`, hosted CI runners, remote CI checks,
  paid CI minutes, scheduled workflows, or status-check automation. This private
  repo must not gain any workflow that can create cost.
- No branch creation, branch switching, commit, push, PR, or GitHub comment.
- No movement of existing ready-for-verification product plans to
  `completed/` without explicit stakeholder/user sign-off.
- No broad rewrite of product source-of-truth docs.
- No claim that a better harness makes AviaSurveil360 production-ready.

## Assumptions

- `docs/exec-plans/` remains the active plan folder per `AGENTS.md`.
- The current worktree may contain prior user/agent changes; execution must not
  revert unrelated README, MANIFEST, CSS, JS, demo summary, plan, note, or test
  changes.
- The current prototype remains a frontend-only static demo using
  `index.html`, `css/`, `js/`, mock data, and browser-only demo persistence.
- There is no `package.json`; verification must use direct `node` commands.
- This repo must stay local-test-only. Agents may document local verification
  commands, but must not add or recommend GitHub Actions or any other paid
  remote CI path.
- The OpenAI Harness Engineering local note emphasizes short map-like
  `AGENTS.md`, repo-versioned knowledge base, executable plans, agent-readable
  UI/log/metric/trace, mechanical architecture boundaries, custom lint or
  structural tests, and automatic documentation maintenance. This plan adapts
  those ideas to AviaSurveil360 without copying production-platform scope.

## File Structure

### Create

- `docs/agent-harness/index.md`
  - First-stop harness index for agents.
  - Explains when to use `AGENTS.md`, product docs, plans, evidence summaries,
    smoke tests, and browser QA.
  - Links all canonical harness package files.
- `docs/agent-harness/output-contract.md`
  - Required agent response and evidence contract.
  - Defines `Done / Remaining / Blocked / Verification / Next` output for
    Turkish readouts.
  - Defines `verified locally`, `blocked`, `not run`, `ready-for-verification`,
    `demo-only`, `source-bound`, and `production-readiness not claimed`.
  - Defines when to update plan index, demo summary, notes, and completed index.
- `docs/agent-harness/registry.md`
  - Inventory of repo surfaces: instructions, product source docs, plans,
    evidence docs, smoke tests, browser QA paths, static demo files, and
    forbidden production surfaces.
  - Provides routing table by task type.
- `docs/agent-harness/verification-matrix.md`
  - Exact command ladder for docs-only, JS logic, workflow, visual/UI, and
    boundary-sensitive tasks.
  - Lists current `node --check` and smoke-test commands.
- `docs/agent-harness/entropy-cleanup-checklist.md`
  - Tracks drift risks: stale README/MANIFEST claims, duplicated runbook rules,
    active/completed plan drift, evidence-label drift, stale blocker notes,
    source-bound uncertainty, and demo-vs-production language.
- `tests/harness-docs-smoke.test.js`
  - Node built-in smoke test for harness package structure and status wording.
  - This test is run manually/local by agents. It is not wired into GitHub
    Actions or any remote CI system.

### Modify

- `AGENTS.md`
  - Add one short pointer under Source Of Truth or Checks to
    `docs/agent-harness/index.md`.
  - Do not duplicate the full harness package.
- `MANIFEST.md`
  - Add the new `docs/agent-harness/` package and
    `tests/harness-docs-smoke.test.js`.
- `docs/demo-handoff/AGENT_HARNESS_RUNBOOK.md`
  - Mark it as the applied runbook and redirect future canonical harness
    navigation to `docs/agent-harness/index.md`.
  - Remove or shorten duplicated sections only if the new harness package fully
    owns them.
- `docs/exec-plans/index.md`
  - Add this plan as `active`.
  - Keep exactly one next concrete todo for the active row.
  - Leave older ready-for-verification plans in place unless explicitly
    superseded or signed off.
- `docs/exec-plans/active/2026-06-29-aviasurveil-harness-engineering-adaptation-plan.md`
  - Add a short continuation note pointing to this readiness completion plan.
  - Do not mark it completed without stakeholder/user sign-off.

### Do Not Modify Unless Evidence Requires It

- `index.html`, `css/`, `js/`
- Existing product source docs under `docs/00_...` through `docs/10_...`
- Existing stakeholder/source materials
- Existing browser screenshot artifacts under `/private/tmp`

## Phases

### Phase 0 - Baseline Classification

**Goal:** freeze the current truth before adding the canonical harness package.

- [ ] Read `AGENTS.md`.
- [ ] Read `docs/demo-handoff/AGENT_HARNESS_RUNBOOK.md`.
- [ ] Read `docs/exec-plans/index.md`.
- [ ] Read
  `docs/exec-plans/active/2026-06-29-aviasurveil-harness-engineering-adaptation-plan.md`.
- [ ] Read `docs/demo-evidence/BUILD_SUMMARY.md`.
- [ ] Read the local OpenAI working note sections around:
  - short `AGENTS.md` as table of contents
  - repo-versioned docs structure
  - executable plans
  - mechanical checks and documentation maintenance
- [ ] Run `git status --short` and classify existing dirty worktree changes as
  pre-existing versus task-owned.

**Verification:**

- The execution agent can state the current gap precisely:
  `runbook exists; canonical docs/agent-harness index/output/registry package
  does not yet exist`.
- No product behavior files are touched in Phase 0.

### Phase 1 - Canonical Harness Package

**Goal:** create the missing first-stop harness package.

- [ ] Create `docs/agent-harness/index.md` with:
  - purpose
  - source hierarchy
  - harness package map
  - task routing summary
  - demo-only boundary
  - links to output contract, registry, verification matrix, entropy cleanup,
    runbook, plan index, and demo summary
- [ ] Create `docs/agent-harness/output-contract.md` with:
  - required final readout shapes
  - status label definitions
  - evidence update rules
  - plan lifecycle output rules
  - browser/GUI cleanup reporting rules
  - forbidden claims
- [ ] Create `docs/agent-harness/registry.md` with:
  - instruction surfaces
  - product source docs
  - plan index, evidence summary, and tech-debt tracker
  - static demo files
  - smoke tests
  - browser QA paths
  - task-to-source routing table
- [ ] Create `docs/agent-harness/verification-matrix.md` with:
  - Level 1 docs-only
  - Level 2 JS logic
  - Level 3 static workflow
  - Level 4 visual/UI
  - Level 5 boundary-sensitive
  - exact commands copied from existing runnable tests only
  - a local-only warning: do not add GitHub Actions or remote CI for this private
    repository
- [ ] Create `docs/agent-harness/entropy-cleanup-checklist.md` with:
  - open cleanup items
  - accepted current risks
  - recurring drift symptoms
  - owner/evidence fields for future updates

**Verification:**

- `find docs/agent-harness -maxdepth 1 -type f | sort` lists the five required
  Markdown files.
- The package does not introduce production-readiness claims.
- The package keeps AviaSurveil360 demo-only boundaries explicit.

### Phase 2 - Output Contract And Registry Integration

**Goal:** make the new package discoverable from existing repo surfaces.

- [ ] Update `AGENTS.md` with a concise pointer to
  `docs/agent-harness/index.md`.
- [ ] Update `MANIFEST.md` to list:
  - `docs/agent-harness/index.md`
  - `docs/agent-harness/output-contract.md`
  - `docs/agent-harness/registry.md`
  - `docs/agent-harness/verification-matrix.md`
  - `docs/agent-harness/entropy-cleanup-checklist.md`
  - `tests/harness-docs-smoke.test.js`
- [ ] Update `docs/demo-handoff/AGENT_HARNESS_RUNBOOK.md` so its
  opening section says the canonical harness entrypoint is
  `docs/agent-harness/index.md`.
- [ ] Update
  `docs/exec-plans/active/2026-06-29-aviasurveil-harness-engineering-adaptation-plan.md`
  with a continuation note:
  `Canonical harness package completion is tracked in
  docs/exec-plans/active/2026-06-29-agent-harness-readiness-completion-plan.md`.
- [ ] Update `docs/exec-plans/index.md`:
  - add this plan as `active`
  - set Next Todo to Phase 1 canonical package creation and smoke gate
  - leave older harness adaptation plan `ready-for-verification` unless a human
    explicitly marks it superseded or completed

**Verification:**

- Every new harness file is reachable from either `AGENTS.md`, `MANIFEST.md`, or
  `docs/agent-harness/index.md`.
- `docs/exec-plans/index.md` has one concrete next todo for this plan.

### Phase 3 - Mechanical Harness Docs Smoke Gate

**Goal:** make harness readiness mechanically checkable.

- [ ] Create `tests/harness-docs-smoke.test.js` using only Node built-ins.
- [ ] Do not create `.github/`, `.github/workflows/`, CI config, package scripts,
  or hosted-runner instructions.
- [ ] The test must check exact required file paths.
- [ ] The test must check that `docs/agent-harness/index.md` links:
  - `output-contract.md`
  - `registry.md`
  - `verification-matrix.md`
  - `entropy-cleanup-checklist.md`
  - `../exec-plans/index.md`
  - `../demo-evidence/BUILD_SUMMARY.md`
  - `../demo-handoff/AGENT_HARNESS_RUNBOOK.md`
- [ ] The test must check that `output-contract.md` contains:
  - `verified locally`
  - `blocked`
  - `not run`
  - `production-readiness not claimed`
- [ ] The test must check that no harness Markdown file contains:
  - `production-ready`
  - `real authentication is implemented`
  - `real upload is implemented`
  - `real AI service is implemented`
  unless it appears in a forbidden-claims or out-of-scope context.
- [ ] Run:

```bash
node tests/harness-docs-smoke.test.js
```

Expected output:

```text
harness-docs-smoke: ok
```

**Verification:**

- The smoke test passes locally.
- Its failure messages name the missing file, missing link, missing status label,
  or forbidden claim.
- No `.github/workflows` path exists or is modified by this task.

### Phase 4 - Verification And Evidence Closeout

**Goal:** prove the harness package is ready locally and record the result
without overstating product readiness.

- [ ] Run Markdown/content review:

```bash
git diff --check
```

- [ ] Run the new harness smoke gate:

```bash
node tests/harness-docs-smoke.test.js
```

- [ ] Run existing demo-boundary smoke because this task touches global
  harness/demonstration guardrails:

```bash
node tests/demo-boundary-smoke.test.js
```

- [ ] Confirm no GitHub Actions or remote CI files were added:

```bash
find .github -maxdepth 3 -type f 2>/dev/null || true
```

Expected output:

```text
```

If `.github/workflows` exists or a workflow file appears, stop and remove the
task-owned workflow change before reporting success. Do not delete unrelated
pre-existing user files without approval; record a blocker if an unrelated
workflow already exists and creates ambiguity.

- [ ] Check changed links by searching exact target paths:

```bash
rg -n "docs/agent-harness|agent-harness/index|output-contract|verification-matrix|entropy-cleanup" AGENTS.md MANIFEST.md docs
```

- [ ] Update this plan's Current Implementation Note with:
  - created harness files
  - verification commands
  - status `verified locally` or `blocked`
  - any coverage gaps
- [ ] Update `docs/exec-plans/index.md`:
  - if all verification passes, status becomes `ready-for-verification`
  - next todo becomes stakeholder/user sign-off before moving to `completed/`
  - if verification fails and cannot be fixed in the execution turn, status
    becomes `blocked` with a note in `docs/exec-plans/`

**Verification:**

- The final status in this plan and `docs/exec-plans/index.md` match.
- Any blocker has a durable note.
- No product behavior or production scope changed.
- No GitHub Actions, hosted runner, remote CI, or paid automation was added.

### Phase 5 - Adoption Gate

**Goal:** make the new harness package the default agent entrypoint without
creating process noise.

- [x] Use the package on one small future repo task.
- [x] Confirm the agent can answer before editing:
  - What source rule controls this task?
  - What output contract applies?
  - Which verification level applies?
  - Where will evidence be recorded?
  - What is explicitly out of scope?
- [x] If the package causes confusion, update `docs/agent-harness/index.md` and
  `output-contract.md` rather than adding more broad text to `AGENTS.md`.
- [x] If the package works, record the result in this plan or a note and move
  this plan toward `completed/` only after user/stakeholder sign-off.

**Verification:**

- Adoption result is recorded in repo docs, not only chat.
- The harness remains concise enough for future agents to actually use.

## Verification

This plan is verified when:

- The plan file exists under `docs/exec-plans/`.
- `docs/exec-plans/index.md` has an active row for this plan.
- The plan includes objective, scope, assumptions, phases, verification, risks,
  dependencies, ownership boundaries, explicit out-of-scope items, and this
  `Execution Prompt`.
- The proposed canonical harness package has exact file paths and a mechanical
  smoke gate.
- Plan creation touches only plan/index artifacts, not prototype behavior.

Execution of the plan is verified when:

- The five canonical harness Markdown files exist.
- `tests/harness-docs-smoke.test.js` passes.
- `node tests/demo-boundary-smoke.test.js` passes.
- `git diff --check` passes.
- `docs/exec-plans/index.md` and this plan agree on final status.
- No backend, database, API, auth, upload, AI, regulatory ingestion,
  notification service, production audit-log, or framework migration is added
  or claimed.
- No GitHub Actions, `.github/workflows`, hosted runner, remote CI, or paid
  automation is added or recommended.

## Risks

- **Harness package sprawl.** Too many docs can recreate the "giant AGENTS"
  problem in another folder. Mitigation: each file has one responsibility and
  the index stays as the entrypoint.
- **Duplicate rules drift.** Existing runbook content may conflict with the new
  package. Mitigation: make `docs/agent-harness/index.md` canonical and shorten
  the old runbook where practical.
- **Plan index noise.** Adding another active plan can make the index harder to
  scan. Mitigation: keep one concrete next todo and move to
  `ready-for-verification` quickly after local checks pass.
- **False readiness language.** "Harness ready" could be mistaken for product
  readiness. Mitigation: every harness file must repeat demo-only and
  production-readiness boundaries.
- **Over-engineering.** A static demo does not need a cloud-grade platform
  harness. Mitigation: use Markdown plus one small Node smoke gate only.
- **Unexpected CI cost.** Private-repo GitHub Actions or hosted runners can
  create cost. Mitigation: local-only verification is a hard boundary; do not
  create `.github/workflows` or remote CI instructions.
- **Unrelated dirty worktree.** Prior local changes may exist. Mitigation:
  preserve them, avoid broad formatting, and report only task-owned changes.

## Dependencies

- Root `AGENTS.md`.
- Existing active plans in `docs/exec-plans/`.
- Existing runbook:
  `docs/demo-handoff/AGENT_HARNESS_RUNBOOK.md`.
- Existing evidence file: `docs/demo-evidence/BUILD_SUMMARY.md`.
- Existing smoke-test pattern under `tests/`.
- Node.js available for direct `node` commands.
- Local shell access for direct commands. No remote CI, GitHub Actions, or hosted
  runner dependency.
- User/stakeholder sign-off before marking this or older harness plans
  completed.

## Ownership Boundaries

- Agents own the harness docs, smoke gate, plan/index updates, and local
  verification evidence for this readiness work.
- Agents do not own CI setup. They must not add GitHub Actions or any paid
  automation for this private repository.
- The user owns stakeholder sign-off, production architecture approval, Git
  branch/commit/push decisions, and external publication.
- Product source documents remain authoritative for AviaSurveil360 behavior.
- The harness package may route agents; it must not redefine regulatory,
  legal, enforcement, evidence, notification, security, or production
  obligations.

## Explicit Out-of-Scope Items

- Backend, database, API, auth, authorization enforcement, upload service,
  document storage, email/SMS/notification service, AI service, regulatory
  ingestion, immutable audit log, production security readiness, CI runners,
  deployment automation, framework migration, cloud agent runner, GitHub write
  automation, or merge bot.
- GitHub Actions, `.github/workflows`, hosted CI, paid minutes, remote scheduled
  checks, or any automation that can create private-repo cost.
- Completing, archiving, or superseding existing plans without inspected local
  verification and explicit user/stakeholder sign-off.
- Rewriting all product specs.
- Modifying original source/reference/stakeholder materials.

## Execution Prompt

```text
You are working in /Users/marlonjd/Developer/web/aviaSurveil360.

Task: execute docs/exec-plans/active/2026-06-29-agent-harness-readiness-completion-plan.md and make the AviaSurveil360 agent harness fully ready as a canonical repo package.

Read first:
- AGENTS.md
- docs/exec-plans/index.md
- docs/exec-plans/active/2026-06-29-agent-harness-readiness-completion-plan.md
- docs/exec-plans/active/2026-06-29-aviasurveil-harness-engineering-adaptation-plan.md
- docs/demo-handoff/AGENT_HARNESS_RUNBOOK.md
- docs/demo-evidence/BUILD_SUMMARY.md
- /Users/marlonjd/Downloads/openai-harness-engineering-tam-kapsamli.md

Goal:
- Create the missing canonical harness package under docs/agent-harness/.
- Keep AGENTS.md short and map-like.
- Add an output contract, registry, verification matrix, entropy cleanup checklist, and a Node smoke gate.
- Preserve demo-only boundaries.

Do:
1. Create docs/agent-harness/index.md.
2. Create docs/agent-harness/output-contract.md.
3. Create docs/agent-harness/registry.md.
4. Create docs/agent-harness/verification-matrix.md.
5. Create docs/agent-harness/entropy-cleanup-checklist.md.
6. Create tests/harness-docs-smoke.test.js using Node built-ins only.
7. Add a concise AGENTS.md pointer to docs/agent-harness/index.md.
8. Update MANIFEST.md and the existing harness runbook so the new package is discoverable.
9. Add a continuation note to docs/exec-plans/active/2026-06-29-aviasurveil-harness-engineering-adaptation-plan.md.
10. Keep docs/exec-plans/index.md synchronized with actual status and one next todo.

Do not:
- Add backend, database, API, real auth, real upload, real AI, real regulatory ingestion, notification service, production audit-log readiness, CI service, framework migration, branch changes, commits, pushes, or GitHub comments.
- Add GitHub Actions, `.github/workflows`, hosted runners, remote CI, paid
  private-repo workflow minutes, scheduled workflows, package-manager test
  scripts, or any automation that can create cost. Verification must stay local:
  direct `node`, `git diff --check`, `rg`, and browser checks only when needed.
- Move any plan to completed without explicit user/stakeholder sign-off.
- Revert unrelated dirty worktree changes.

Verification:
- git diff --check
- node tests/harness-docs-smoke.test.js
- node tests/demo-boundary-smoke.test.js
- rg -n "docs/agent-harness|agent-harness/index|output-contract|verification-matrix|entropy-cleanup" AGENTS.md MANIFEST.md docs
- find .github -maxdepth 3 -type f 2>/dev/null || true

Expected final status:
- If all checks pass: mark this plan ready-for-verification in docs/exec-plans/index.md with next todo "stakeholder/user sign-off before moving to completed".
- If checks cannot pass: mark blocked, create/update `docs/exec-plans/tech-debt-tracker.md`, and keep the gap explicit.

Final response:
- Use Done / Remaining / Blocked / Verification / Next.
- Link the plan, docs/agent-harness/index.md, output-contract.md, registry.md, verification-matrix.md, entropy-cleanup-checklist.md, and smoke test if created.
- State demo-only and production-readiness boundaries clearly.
```
