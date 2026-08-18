# Prior-Audit-Aware Question Recommendations

Date: 2026-08-18
Status: active — plan created; existing adaptive-scope specification and recommendation path inspected; implementation not started; release and production readiness are separate gates

## Objective and user-visible outcome

When a Department Manager plans a new Audit, the system will use comparable
completed Audit history to recommend the questions that should be inspected
now. It will visibly explain the history and risk signals behind every
recommendation, remove only questions that are safe to defer, and keep the
complete approved catalog available for an explicit manager override.

The implementation must include two durable, deterministic qualification
examples:

1. **Multiple prior Audits with mixed importance.** The same organization,
   provider scope, regulated target, and inspection type have three completed
   prior Audits. Their question history intentionally mixes mandatory,
   safety-critical, open/repeat-finding, recently clean optional, older clean,
   and changed-source questions. The new plan must keep mandatory, open,
   repeat, changed, and due controls in the default recommendation while
   excluding a recently clean optional control from the default set with a
   visible `RECENTLY_VERIFIED`/`DEFER_ELIGIBLE` rationale. The omitted question
   remains selectable from the full catalog.
2. **Exactly one prior Audit.** The same comparable scope has one completed
   prior Audit. Mandatory, unknown-history, open/repeat, and changed-source
   questions remain recommended. A single clean result for an optional
   question is not sufficient longitudinal evidence for silent omission; it
   remains suggested or explicitly marked uncertain until the manager makes a
   recorded deferral decision.

The manager should be able to distinguish these outcomes in the planning
screen without opening historical records one by one.

## Scope

Included:

- immutable, source-backed prior-Audit history fixtures for the two examples;
- exact comparability rules for organization, provider scope, regulated target,
  location, inspection type, checklist/catalog version, and question version;
- question-level recommendation states, classifications, history signals, and
  rationale/guardrail fields;
- server-side filtering of default recommendations with full-catalog override;
- Manager planning UI presentation in the checklist/question selection step;
- API/OpenAPI/generated transport updates where the response contract changes;
- focused Go, React, integration, and browser acceptance evidence.

Explicit exclusions:

- deleting or mutating a historical Audit, checklist answer, finding, CAP,
  Evidence record, or published catalog version;
- treating “no finding recorded” or missing history as proof of compliance;
- automatic legal, enforcement, certificate, or publication decisions;
- hidden client-only filtering that cannot be reproduced from the server
  response;
- replacing the approved catalog with synthetic questions;
- public deployment, release-lock mutation, credential changes, or production
  claims as part of implementation. Those require a separate authorized
  release gate.

## Assumptions and ownership boundaries

- AviaSurveil owns the recommendation query, recommendation response contract,
  immutable Audit-history projections, and Manager presentation.
- The Workspace owns deployment manifests, immutable release locks, and cloud
  apply safety; it does not define recommendation semantics.
- A prior result is comparable only when the server proves the exact scope
  identity and applicable catalog/question version. Similar labels alone are
  insufficient.
- Mandatory, safety-critical, newly changed, overdue, open-Finding,
  repeat-Finding, and unknown-history questions cannot be automatically
  omitted.
- Deferral is advisory until the manager records an explicit reason; the final
  frozen question scope remains append-only and audit logged.

## Repository orientation and affected interfaces

Primary surfaces:

- `apps/api/internal/httpapi/canonical_aga_api.go` — catalog selection,
  prior-history projection, recommendation states, and query predicates.
- `apps/api/internal/preproddata/canonicalaga/` — governed catalog and
  applicability semantics; historical rows must remain immutable.
- `apps/api/migrations/` — only if a missing durable recommendation/history
  fact requires a forward-only schema change.
- `apps/api/internal/httpapi/generated/` and `apps/web/src/generated/` —
  generated transport contracts when OpenAPI changes.
- `apps/web/src/features/planning/new-audit-wizard.tsx` and its tests —
  recommendation display, full-catalog override, and frozen selection.
- `apps/web/src/mock/mock-engine.ts` — demo parity fixtures, clearly labelled
  synthetic and never exposed through the HTTP production artifact.
- `apps/web/tests/e2e/qualification-all-role.spec.ts` — authenticated Manager
  acceptance when the connected qualification gate is authorized.
- `docs/product-specs/workflows/AUDIT_CHECKLIST_WORKFLOW.md` and
  `docs/product-specs/modules/AUDIT_PLANNING.md` — semantic source of truth.

## Ordered implementation

### Phase 0 — Baseline and decision table

1. Record the current recommendation response fields and states.
2. Map existing question-level history, finding/CAP/Evidence state, source
   currentness, and comparability queries to the decision table below.
3. Identify any gap where the current API returns a recommendation state but
   not the evidence needed to explain it.

| Signal | Multi-Audit expected result | Single-Audit expected result |
|---|---|---|
| Mandatory/safety-critical | `MANDATORY_CORE`, suggested | `MANDATORY_CORE`, suggested |
| Open or repeat finding | `FOCUSED_FULL`, suggested | `FOCUSED_FULL`, suggested |
| Recently clean optional | `DEFER_ELIGIBLE`/`RECENTLY_VERIFIED`, omitted by default | not silently omitted; `UNCERTAIN_SIGNAL` or suggested |
| Older clean optional | rotational or due according to full-scope interval | suggested until longitudinal evidence exists |
| Source/mapping changed | `FOCUSED_FULL`, suggested | `FOCUSED_FULL`, suggested |
| Missing or incomparable history | `UNCERTAIN_SIGNAL`, suggested | `UNCERTAIN_SIGNAL`, suggested |

### Phase 1 — Deterministic history fixtures

1. Add a fixture profile with stable IDs for the multi-Audit example:
   `prior-audit-multi-history`.
2. Add a fixture profile with stable IDs for the single-Audit example:
   `prior-audit-single-history`.
3. Ensure each prior Audit has immutable scope identity, catalog/question
   version, result, result timestamp, and finding/CAP/Evidence linkage.
4. Ensure fixture history is isolated by organization and cannot appear for a
   different target organization or provider scope.
5. Keep all fixture questions in the approved catalog; use enrichment only as
   advisory risk metadata.

### Phase 2 — Server recommendation policy

1. Aggregate comparable history per immutable question version instead of
   treating the latest Audit alone as the complete history.
2. Apply fail-closed precedence: mandatory/open/repeat/changed/overdue/unknown
   signals outrank clean-history deferral.
3. Require more than one comparable completed Audit, a valid full-scope
   baseline, unchanged source/mapping, and a configured interval before a
   clean optional question can be omitted by default.
4. Return, for each question, its recommendation state, classification,
   inclusion decision, history count, last comparable result, signal codes,
   rationale, guardrails, and whether the manager may override it.
5. Preserve deterministic ordering and cursor behavior for both the
   recommendation view and the full approved catalog.

### Phase 3 — Contract and UI integration

1. Update OpenAPI/source schemas and regenerate Go/TypeScript transport types
   if the response fields change.
2. Make the default planning selection consume the server recommendation, not
   a client-side recreation of history logic.
3. Show a compact recommendation summary and expandable per-question history
   basis; distinguish “not recommended to repeat” from “not eligible”.
4. Provide a visible “show full catalog”/override path. Any deferral override
   requires a reason and is carried into the frozen scope decision.
5. Keep the full catalog searchable, paginated, and selectable after a
   question is omitted from the suggested set.

### Phase 4 — Verification and qualification

1. Add focused server tests for both fixture profiles and precedence rules.
2. Add React tests for suggested/omitted/uncertain states, rationale display,
   override, and frozen selection.
3. Add an integration test that replays history and proves old Audit/checklist
   rows are unchanged.
4. Run the connected qualification flow only with an exact authorized target,
   private credential custody, and cleanup evidence.
5. Run browser checks for desktop and 390x844 mobile layouts, including the
   recommendation explanation and full-catalog override.

### Phase 5 — Release gate

1. Run source, contract, build, and qualification checks.
2. Obtain explicit release authorization for any image/lock/bootstrap change.
3. Publish only immutable, source-bound artifacts and verify public health and
   app-shell digests before claiming release evidence.

## Concrete commands and expected observations

From `apps/surveil`:

```bash
go test ./apps/api/internal/httpapi/... ./apps/api/internal/preproddata/canonicalaga ./apps/api/internal/qualificationbootstrap
npm --prefix apps/web test -- --run src/features/planning/new-audit-wizard.test.tsx src/ui/application-shell.test.tsx
npm --prefix apps/web run typecheck
npm --prefix apps/web run build:demo
npm --prefix apps/web run build:http
bash scripts/test-qualification-bootstrap.sh
git diff --check
```

Expected observations:

- multi-Audit fixture excludes only the recently clean optional question from
  default suggestions and retains mandatory/open/repeat/changed questions;
- single-Audit fixture does not treat one clean optional result as clean
  longitudinal history;
- the full-catalog request still returns every approved question and the same
  deterministic order;
- replay changes no historical Audit/checklist/finding/CAP/Evidence rows;
- the browser shows recommendation rationale without runtime diagnostics or
  role-privacy leakage.

## Verification and acceptance criteria

- The multi-Audit example contains at least three comparable completed Audits
  and visibly distinguishes important/safety-critical, open/repeat, clean
  optional, and changed questions.
- The single-Audit example contains exactly one comparable completed Audit and
  visibly shows the conservative uncertainty rule.
- A question omitted from “Suggested now” remains available through the full
  approved catalog and can be restored with an explicit reason.
- Every omitted/uncertain question exposes history count, last result, signal
  basis, rationale, and guardrails.
- No mandatory, safety-critical, open, repeat, overdue, changed, or unknown
  question is silently omitted.
- Recommendation behavior is identical for API and HTTP/React surfaces for the
  same immutable fixture and scope.
- Historical records and published catalog/question versions remain byte and
  row immutable.
- The final frozen selection records the recommendation snapshot, manager
  changes, reasons, and deterministic selection digest.
- No Auditee projection exposes internal recommendation history, risk signals,
  or manager deferral rationale unless the product contract explicitly allows
  it.

## Risks, dependencies, idempotence, and recovery

- **History comparability drift:** fail closed to `UNCERTAIN_SIGNAL` and keep
  the question suggested; never widen history matching heuristically.
- **Over-deferral:** mandatory and risk-signal precedence prevents automatic
  omission; a full-scope interval remains enforced.
- **Catalog evolution:** published catalog/question rows remain immutable; a
  changed source creates a new version or append-only applicability fact.
- **Replay:** fixture loaders and recommendation projections must be
  idempotent under the same immutable IDs and deterministic ordering.
- **Release failure:** retain the prior immutable lock and roll forward only
  after a candidate passes source, artifact, and exact-target gates.

## Current progress, decisions, and discoveries

- `AUDIT_CHECKLIST_WORKFLOW.md` already defines adaptive scope, visible
  classifications, history signals, guardrails, and fail-closed omission
  rules.
- The API already exposes prior-history projections and recommendation states,
  but the exact two requested fixtures and acceptance oracle were not present
  in the active plan/test inventory.
- The authorized Namibia/demo scope/catalog release is now deployed and
  publicly verified under immutable lock `sha256:643b4b…`; that deployment is
  evidence for the separate release task, not completion of this plan.
- The local PostgreSQL qualification harness passed the catalog applicability
  upgrade/replay boundary. Recommendation-specific two-history behavior is
  still `not run` until this plan is implemented.

## Outcome notes

This plan currently records the required examples, policy decision table, and
acceptance contract. No recommendation-policy source change is claimed by the
plan itself.

## Execution Prompt

Continue from Phase 0. Read this plan, the adaptive-scope workflow, the Audit
Planning module, the current API recommendation query, and the Manager wizard.
Implement the two deterministic history fixtures first, then make the server
recommendation contract explainable and fail-closed. Preserve all historical
Audit/checklist/catalog rows, keep the full catalog override available, run the
focused and connected gates, and stop before any release or production action
unless the user separately authorizes that exact release. Report every gate as
`verified locally`, `not run`, `blocked`, `candidate-only`, or `release pending`.
