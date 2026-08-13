# AviaSurveil360 Harness Engineering Adaptation Plan

- **Date:** 2026-06-29
- **Status:** superseded
- **Owner:** Product engineering workflow
- **Type:** Agent harness, verification, and repo-operating model adaptation
- **Authority:** Root `AGENTS.md`; `docs/exec-plans/` is the active plan folder.
- **Source boundary:** The user supplied `https://openai.com/tr-TR/index/harness-engineering/` and the local Turkish Markdown working note at `/Users/marlonjd/Downloads/openai-harness-engineering-tam-kapsamli.md`. Direct fetches of the OpenAI Turkish and English article URLs returned a JavaScript/Cloudflare challenge during this run, so this plan treats the URL as the official source identity and adapts the user-provided Markdown summary plus current repo evidence. It does not claim official full-text extraction from the OpenAI page.

**Current implementation note (2026-06-29):** Phases 0-6 are executed locally. `docs/demo-handoff/AGENT_HARNESS_RUNBOOK.md` now includes the task routing, decision manifest, verification matrix, and `tests/demo-boundary-smoke.test.js` gate. README/MANIFEST package-truth cleanup is complete. The runbook was applied to the CAA Governance browser QA lane; desktop governance paths and the mobile Planning Approval content screenshot are `verified locally`. The former mobile blocker is closed in `docs/exec-plans/completed/2026-06-29-governance-browser-qa-mobile-blocker.md`. Production-readiness is not claimed.

**Continuation note (2026-06-29):** This plan is superseded for final readiness
tracking because the user identified that the repo still lacks a canonical
`docs/agent-harness/` package with a harness index, output contract, registry,
verification matrix, entropy cleanup checklist, and mechanical docs smoke gate.
The completion work is now tracked in
`docs/exec-plans/active/2026-06-29-agent-harness-readiness-completion-plan.md`.

## Sources Used

- OpenAI page requested by the user: `https://openai.com/tr-TR/index/harness-engineering/`
- OpenAI English article URL attempted during this run: `https://openai.com/index/harness-engineering/`
- User-provided local Turkish Markdown working note: `/Users/marlonjd/Downloads/openai-harness-engineering-tam-kapsamli.md`
- Repo source-of-truth docs listed in `AGENTS.md`, especially `README.md`, `README.turkce.md`, `MANIFEST.md`, `docs/product-specs/product-plan/MVP_SCOPE_AND_ROADMAP.md`, `docs/product-specs/workflows/FINDING_CAP_EVIDENCE_WORKFLOW.md`, `docs/product-specs/data-and-rules/STATUS_PERMISSION_SECURITY.md`, `docs/demo-handoff/CODEX_DEMO_ONLY_PROMPT.md`, and `docs/demo-evidence/BUILD_SUMMARY.md`.

## Adapted Repository Output

This adaptation is now represented in repo-backed artifacts instead of chat-only guidance:

- `docs/demo-handoff/AGENT_HARNESS_RUNBOOK.md` is the compact operating runbook for future agents.
- `docs/exec-plans/active/2026-06-29-aviasurveil-harness-engineering-adaptation-plan.md` is the implementation plan and decision record.
- `docs/exec-plans/index.md` routes the active harness plan and keeps one concrete next todo.
- `docs/exec-plans/completed/2026-06-29-governance-browser-qa-mobile-blocker.md` records the current durable browser-QA blocker.

Current classification:

- Harness runbook: **ready for use** for future AviaSurveil360 repo work.
- First harness application: desktop CAA Governance paths are **verified locally**.
- Remaining harness evidence gap: canonical `docs/agent-harness/` package and
  output contract are not yet implemented; tracked in
  `docs/exec-plans/active/2026-06-29-agent-harness-readiness-completion-plan.md`.
- Product readiness: **demo-only**; **production-readiness not claimed**.

## Objective

Adapt harness engineering to AviaSurveil360 so future Codex/Claude work is less dependent on chat memory and more dependent on repo-backed, falsifiable artifacts.

For this repo, the harness is not only `AGENTS.md`. It is the full working environment that guides an agent:

- repository instructions and source-of-truth docs
- active plans and next todos
- smoke tests and browser QA gates
- demo-only guardrails
- evidence summaries
- decision records that explain why a change was made and how it will be verified

The outcome should be an AviaSurveil360-specific agent harness that makes the next agent answer these questions before editing:

1. What product rule or plan controls this task?
2. Which component is safe to edit?
3. What evidence proves the change worked?
4. What regression could this introduce?
5. Where is the durable result recorded?

## Scope

### In Scope

- Define an AviaSurveil360 harness map across `AGENTS.md`, `docs/`, `docs/exec-plans/`, `docs/demo-evidence/BUILD_SUMMARY.md`, `tests/`, and the static demo files.
- Create a practical runbook for future agents in `docs/demo-handoff/` instead of adding a broad new top-level folder.
- Add a decision-manifest pattern for material plan/prototype changes:
  - evidence observed
  - root cause or opportunity
  - targeted component
  - predicted fix
  - at-risk regression
  - verification command or browser path
  - final verdict
- Connect the harness to the current AviaSurveil360 reality:
  - frontend-only static demo
  - mock data and client-side state
  - no backend/database/API/auth/upload/AI service
  - Finding -> CAP -> Evidence -> CAA Review -> Closure lifecycle
  - auditee isolation
  - internal note vs auditee-visible comment separation
  - `docs/exec-plans/index.md` as active plan routing
- Identify the smallest useful verification suite for agentic work:
  - syntax checks
  - deterministic Node smoke tests
  - browser click-through and visual QA
  - demo-only boundary review
  - terminology consistency review
- Keep source-bound, demo-only, local verification, stakeholder review, and production-readiness claims separate.

### Out Of Scope

- No backend, database, API, real authentication, real file upload, real email, real AI service, real regulatory ingestion, real notification service, real audit-log infrastructure, or framework migration.
- No claim that harness engineering makes the demo production-ready.
- No automatic self-modifying agent loop, CI bot, cloud agent runner, or benchmark optimization system.
- No regulatory/legal/enforcement decision automation.
- No branch, commit, push, or GitHub comment action unless the user explicitly asks.
- No broad rewrite of `AGENTS.md` until the runbook proves which instructions are missing.
- No movement of existing active plans to `completed/` unless their objective and verification have been inspected and classified.

## Assumptions

- The repo currently contains a runnable static prototype and the package metadata now describes it as a planning pack plus frontend-only static clickable demo.
- The current source-of-truth hierarchy remains:
  - root `AGENTS.md`
  - product specs under `docs/product-specs/`
  - active plan index at `docs/exec-plans/index.md`
  - build evidence at `docs/demo-evidence/BUILD_SUMMARY.md`
  - executable smoke checks under `tests/`
- The first executable artifact remains a frontend-only clickable demo unless the user explicitly changes scope.
- Existing dirty worktree changes in CSS/JS/tests may belong to ongoing work and must not be reverted by this harness pass.
- Future agents should prefer a small, repo-backed runbook over repeating long chat prompts.

## Harness Engineering Translation

| Harness engineering idea | AviaSurveil360 adaptation | First durable artifact |
|---|---|---|
| `AGENTS.md` as map, not encyclopedia | Keep `AGENTS.md` authoritative and route deeper details into `docs/`, plans, evidence summaries, and this runbook. | `docs/demo-handoff/AGENT_HARNESS_RUNBOOK.md` |
| Repo as record system | Move durable product, architecture, verification, blockers, and handoff decisions into repo artifacts instead of chat memory. | `docs/exec-plans/`, `docs/demo-evidence/BUILD_SUMMARY.md`, future `docs/exec-plans/` |
| Agent-readable application | Make the static demo bootable and inspectable through local browser paths, screenshots, console checks, and role-flow smoke tests. | Browser QA paths in the runbook |
| Agent-readable codebase | Prefer stable, explicit data/state/helpers/tests that Codex can read in context; avoid hidden decisions in chat or external docs. | Harness map + task routing |
| Mechanical architecture and taste | Convert product rules and UI quality expectations into tests, smoke checks, checklists, and future lint/structural gates. | Verification matrix + future quality gates |
| Merge philosophy adjusted to capacity | Use scoped, short-lived diffs and follow-up plans, but keep this repo's explicit no-branch/no-commit/no-GitHub-write rules. | Git/workspace hygiene section |
| Autonomy ladder | Increase agent autonomy only after docs, smoke tests, browser QA, evidence capture, and plan-index discipline are in place. | Agent autonomy ladder in the runbook |
| Entropy cleanup | Add recurring doc gardening, stale metadata cleanup, and targeted refactor/QA tasks so bad patterns do not spread. | Entropy cleanup section in the runbook |
| Decision observability | Every material edit carries a prediction and a verification result. | Decision Manifest section in execution plans and the tech-debt tracker |
| Guarded action space | Agents may improve docs, demo, and tests inside this repo, but may not create backend/production claims or perform branch/GitHub write actions without explicit user direction. | `AGENTS.md` plus runbook |

## Current Repo Harness Inventory

### Instructions

- `AGENTS.md` is already strong and should remain the highest local authority.
- It defines the demo-first boundary, product rules, plan lifecycle, and verification expectations.
- The likely missing layer is not more global policy. It is a compact task-entry runbook that tells agents which source docs, plan rows, tests, and browser paths to use for common AviaSurveil360 work.

### Product Source Of Truth

- Product core: `docs/product-specs/product-plan/*`
- Workflows: `docs/product-specs/workflows/*`
- Modules: `docs/product-specs/modules/*`
- Data/security: `docs/product-specs/data-and-rules/*`
- Demo/build handoff: `docs/demo-handoff/*`
- Scenarios: `docs/product-specs/scenarios/*`

Harness implication: agents should not invent a new lifecycle. The operative lifecycle is:

```text
Surveillance Plan -> Audit / Inspection -> Checklist -> Finding / Observation -> CAP -> Evidence -> CAA Review -> Closure -> Dashboard / Report
```

### Active Planning Surface

`docs/exec-plans/index.md` currently tracks:

- base demo plan
- NCAA Platform V2 and MVP plan
- CAA governance workflow and multi-role expansion plan

Harness implication: future work should begin by checking whether it extends one of these active lanes or needs a new plan row.

### Executable Verification Surface

The repo has no `package.json`, but it does have deterministic Node smoke tests:

- `tests/approval-smoke.test.js`
- `tests/checklist-approval-smoke.test.js`
- `tests/checklist-management-smoke.test.js`
- `tests/governance-render-smoke.test.js`
- `tests/inspection-execution-smoke.test.js`
- `tests/planning-render-smoke.test.js`
- `tests/planning-release-smoke.test.js`
- `tests/report-approval-smoke.test.js`
- `tests/audit-work-queue-smoke.test.js`

Harness implication: a future runbook should list these as targeted gates and avoid implying that a package-manager test script exists.

### Evidence Surface

`docs/demo-evidence/BUILD_SUMMARY.md` is the best current evidence artifact. It already separates:

- frontend-only demo status
- mock persistence
- simulated offline behavior
- regulatory trace guardrails
- AI-draft guardrails
- browser verification status
- production gaps

Harness implication: do not treat chat summaries as the durable evidence layer when this file should be updated.

## Phases

### Phase 0 - Baseline And Source Boundary

**Goal:** record the current harness baseline without changing product behavior.

- [x] Confirm the OpenAI harness page source boundary in this plan: official URL provided; direct body fetch blocked by JavaScript/Cloudflare challenge; local Turkish Markdown working note available.
- [x] Translate the article's usable principles into AviaSurveil360 terms: repo as record system, agent-readable demo, mechanical checks, decision manifest, autonomy ladder, and entropy cleanup.
- [x] Inspect current dirty worktree before any edits and classify task-owned vs unrelated changes.
- [x] Confirm whether the next task extends an existing active plan or this harness plan.

**Verification:**

- Plan source boundary is explicit.
- No production/demo claims are blended.
- No unrelated CSS/JS/test changes are reverted.

### Phase 1 - AviaSurveil360 Agent Harness Runbook

**Goal:** create a compact, repo-local runbook for future agents.

**Status:** implemented in `docs/demo-handoff/AGENT_HARNESS_RUNBOOK.md`.

**Preferred file:**

- `docs/demo-handoff/AGENT_HARNESS_RUNBOOK.md`

**Runbook content:**

- task routing table:
  - docs-only product change
  - static prototype behavior change
  - UI/visual polish
  - verification/status readout
  - stakeholder handoff prompt
  - plan lifecycle update
- required source reads by task type
- active plan index usage
- verification gates by risk level
- demo-only guardrails
- browser/GUI hygiene notes
- exact status language:
  - `verified locally`
  - `ready-for-verification`
  - `blocked`
  - `demo-only`
  - `source-bound`
  - `production-readiness not claimed`
- "do not edit" list for source/reference files and unrelated dirty work.

**Verification:**

- The runbook is shorter than `AGENTS.md` and references it instead of duplicating every rule.
- It uses AviaSurveil360 terminology consistently.
- It does not weaken demo-only constraints.

### Phase 2 - Decision Manifest Pattern

**Goal:** make future harness/product changes falsifiable.

**Status:** implemented in `docs/demo-handoff/AGENT_HARNESS_RUNBOOK.md`.

Add a reusable section template to the runbook:

```markdown
## Decision Manifest Entry

- Evidence observed:
- Root cause or opportunity:
- Targeted component:
- Change made:
- Predicted fix:
- At-risk regression:
- Verification:
- Verdict:
```

Use it when a change is more than a typo or narrow mechanical correction.

**Verification:**

- The template is present.
- At least one example maps to an AviaSurveil360 task, such as "auditee can see internal note" or "CAP acceptance accidentally closes finding."

### Phase 3 - Verification Matrix

**Goal:** give agents a concrete test ladder instead of a vague "check it" instruction.

**Status:** implemented in `docs/demo-handoff/AGENT_HARNESS_RUNBOOK.md`.

Create a matrix with these levels:

1. **Docs-only:** markdown/content review, changed links/paths, terminology consistency.
2. **JS logic:** `node --check` for touched JS plus targeted Node smoke test.
3. **Static prototype workflow:** local browser click-through of touched role path.
4. **Visual/UI change:** desktop and mobile viewport screenshot review; no overlapping text; no template-like generic UI.
5. **Boundary-sensitive change:** explicit no-backend/no-real-upload/no-real-AI/no-production-claim review.

Suggested current commands:

```bash
node --check js/data.js
node --check js/helpers.js
node --check js/approval.js
node --check js/planning.js
node --check js/checklists.js
node --check js/inspection.js
node --check js/reports.js
node --check js/views.js
node --check js/app.js
node tests/approval-smoke.test.js
node tests/checklist-approval-smoke.test.js
node tests/checklist-management-smoke.test.js
node tests/governance-render-smoke.test.js
node tests/inspection-execution-smoke.test.js
node tests/planning-render-smoke.test.js
node tests/planning-release-smoke.test.js
node tests/report-approval-smoke.test.js
node tests/audit-work-queue-smoke.test.js
```

**Verification:**

- Matrix names exact files/commands that exist.
- It does not require `npm test` or another non-existent package script.
- Browser checks remain local/static unless a task explicitly needs hosting.

### Phase 4 - Mechanical Quality Gates And Entropy Cleanup

**Goal:** convert repeated review preferences into mechanical checks and cleanup tasks.

**Status:** implemented with `tests/demo-boundary-smoke.test.js` and recorded in
`docs/demo-handoff/AGENT_HARNESS_RUNBOOK.md`.

The article emphasizes that documentation alone does not keep an agent-produced
codebase coherent. For this repo, the first mechanical gates should stay small:

- test that auditee views cannot render internal CAA notes
- test that CAP acceptance never closes a finding
- test that mock uploads never imply real storage
- test that role navigation does not expose unauthorized internal surfaces
- add a docs check for stale "Markdown-only package" claims after package-truth cleanup
- add a terminology check for `Finding`, `CAP`, `Evidence`, `Due Date`, `Target`, `Overdue`, `Auditee`, `CAA Inspector`, `CAA Manager`, `Internal CAA Note`, and `Comment to Auditee`

**Verification:**

- Any new gate is runnable locally without package-manager assumptions.
- Error text tells a future agent what product rule was violated.
- The gate is tied to a real recurring failure risk, not abstract process preference.

### Phase 5 - Package Truth Cleanup Hook

**Goal:** prevent future agents from trusting stale package metadata.

**Status:** implemented in `README.md`, `README.turkce.md`, and `MANIFEST.md`.

This phase coordinated with the existing V2 plan because `README.md` and `MANIFEST.md` previously said the package was Markdown-only while the repo contains `index.html`, `css/`, `js/`, and `tests/`.

**Candidate changes:**

- Update `README.md` and `README.turkce.md` to describe the planning pack plus frontend-only static demo.
- Update `MANIFEST.md` to include prototype files, tests, build summaries, and plans.
- Keep demo-only and non-production status prominent.

**Verification:**

- README/MANIFEST no longer misroute future agents.
- Turkish companion is updated because the README follows a bilingual pattern.
- No production-readiness claim is introduced.

### Phase 6 - Apply To Next Active Work

**Goal:** use the harness against one real current lane before considering broader AGENTS changes.

**Status:** applied and mobile coverage blocker closed.

Recommended first application:

- CAA Governance Workflow visual/browser QA, because the active index already lists it as the next todo.

Use the runbook to:

- identify the role path
- pick exact tests
- run browser click-through
- record evidence in `docs/demo-evidence/BUILD_SUMMARY.md` or `docs/exec-plans/tech-debt-tracker.md`
- update `docs/exec-plans/index.md` only if status/next todo changes

**Application result (2026-06-29):**

- Syntax checks and deterministic Node smoke tests for the governance lane passed.
- Desktop browser click-through/visual QA is `verified locally` for planning approval, checklist approval, report final lock, inspector work queue/offline field, auditee isolation, and admin question bank.
- Mobile Planning Approval visual QA is `verified locally`: accepted screenshot evidence is `/private/tmp/aviasurveil360-governance-qa/10-mobile-planning-approval-verified.png`, captured through `http://127.0.0.1:4360/` at a 390px viewport.
- The accepted assertion used visible page content: `Planning Approval — PLAN-2026-Q3-OPS` was visible in the viewport and the `Q3 Flight Operations Surveillance Plan` dossier was visible. Console warnings/errors were empty and scrollWidth/clientWidth was `390/390`.
- Durable evidence was recorded in `docs/demo-evidence/BUILD_SUMMARY.md`; the former blocker was closed in `docs/exec-plans/completed/2026-06-29-governance-browser-qa-mobile-blocker.md`.

**Verification:**

- The active index row matches the actual final state.
- Any future coverage gap must be captured as `blocked` or `ready-for-verification`, not hidden in chat.

## Verification For This Plan Creation

Plan creation is complete when:

- this plan exists under `docs/exec-plans/`
- `docs/exec-plans/index.md` has one active row for it
- the plan includes objective, scope, assumptions, phases, verification, risks, dependencies, ownership boundaries, explicit out-of-scope items, and an `Execution Prompt`
- changed Markdown paths are internally linkable
- no prototype behavior files are modified by this plan-creation pass

## Risks

- **Over-broad AGENTS edits:** adding every harness idea to `AGENTS.md` could make future agents slower and less reliable. Mitigation: create a focused runbook first.
- **False production confidence:** a better agent harness could be mistaken for product readiness. Mitigation: preserve demo-only labels and evidence boundaries.
- **Plan index drift:** new plans can make the active index noisy. Mitigation: keep one concrete next todo per row.
- **Stale package truth:** README/MANIFEST can drift again as the repo changes. Mitigation: keep package metadata in the runbook's mechanical gates and update it with future demo scope changes.
- **Unrelated dirty worktree:** existing CSS/JS/test changes may be user or prior-agent work. Mitigation: do not revert or reformat unrelated files.
- **Source-body uncertainty:** the OpenAI page body was not available through direct fetch during this run. Mitigation: state the boundary and adapt only from the user-provided local Markdown summary, official URL identity, and repo evidence.

## Dependencies

- Existing repo-local `AGENTS.md` rules.
- Existing active plans in `docs/exec-plans/`.
- Current static prototype architecture: `index.html`, `css/`, `js/`.
- Current smoke tests under `tests/`.
- Local browser access for click-through verification.
- User approval before any branch, commit, push, GitHub comment, backend work, or production architecture shift.

## Ownership Boundaries

- Agents may update execution plans, runbook docs, smoke tests, and demo files only when the active task calls for it.
- Agents must not modify original stakeholder/source materials unless explicitly asked.
- Agents must preserve bilingual companion docs where the repo already follows that pattern.
- Agents must keep regulatory references as references, not legal advice.
- Agents must not make production security, audit-log, evidence-chain, notification, AI, or offline claims without matching implementation and evidence.
- The user owns product scope changes, production architecture approval, external stakeholder messaging, and any Git/GitHub write action.

## Execution Prompt

```text
You are working in /Users/marlonjd/Developer/monorepos/avia/apps/surveil.

Task: perform the next stakeholder verification/sign-off pass for the AviaSurveil360 Harness Engineering Adaptation plan.

Read first:
- AGENTS.md
- docs/demo-handoff/AGENT_HARNESS_RUNBOOK.md
- docs/exec-plans/index.md
- docs/exec-plans/active/2026-06-29-aviasurveil-harness-engineering-adaptation-plan.md
- docs/exec-plans/active/2026-06-28-caa-governance-workflow-and-roles-plan.md
- docs/demo-evidence/BUILD_SUMMARY.md
- docs/product-specs/data-and-rules/STATUS_PERMISSION_SECURITY.md
- docs/exec-plans/completed/2026-06-29-governance-browser-qa-mobile-blocker.md

Do:
1. Review the harness runbook, package metadata, active plan index, demo summary, and closed blocker note for consistency.
2. Confirm that Phase 0-6 evidence remains `verified locally` and that production-readiness is not claimed.
3. Prepare a compact stakeholder sign-off readout with Done / Remaining / Blocked / Verification / Next.
4. If a stakeholder or user explicitly approves sign-off, move this harness plan to `completed/` per AGENTS.md plan lifecycle rules.
5. If new gaps appear, record them in `docs/exec-plans/` and keep the plan `ready-for-verification` or `blocked` as appropriate.

Do not:
- Add backend, database, API, real authentication, real upload, real AI, real regulatory ingestion, real notification service, or production audit-log readiness.
- Create branch, commit, push, or GitHub comments.
- Reopen the closed mobile Planning Approval blocker unless new evidence invalidates the accepted screenshot.
- Move this plan to completed without inspected sign-off and verification status.

Verification:
- Changed Markdown paths are internally consistent.
- Active plan index, tech-debt tracker, and demo summary agree on status.
- Demo-only boundaries remain intact.

Final response:
- Link the runbook, this plan, and any evidence file changed.
- State Done / Remaining / Blocked / Verification / Next.
```
