# Prior-Audit-Aware Question Recommendations

Date: 2026-08-18
Status: active — deterministic server/mock policy, append-only persistence boundary, contract transport, Manager presentation, focused tests, builds, and local qualification-bootstrap replay are `verified locally`; connected recommendation qualification and release are `not run`/`release pending`; production readiness not claimed

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
- The API previously projected only a latest clean timestamp. The recommendation
  policy now has an exact `ComparableAuditKey`, distinct eligible Audit
  aggregation, fixed `evaluationAsOf` from the server Clock dependency, clean
  truth-table guards, precedence, and a server-side `includedByDefault` filter
  that is independent from `recommendationState`.
- The exact deterministic fixture oracles are checked in at
  `apps/api/tests/fixtures/prior-audit-recommendations/prior-audit-multi-history.json`
  and `apps/api/tests/fixtures/prior-audit-recommendations/prior-audit-single-history.json`.
  The Go and mock adapters use the same stable question-version IDs and profile
  names: `prior-audit-multi-history` and `prior-audit-single-history`.
- `CanonicalQuestionAIAdvisory` remains an advisory-only input. The server
  emits a separate `CanonicalQuestionRecommendation` with classification,
  inclusion, deferral, history count, signal codes, rationale, and guardrails.
- Migration `000055_prior_audit_recommendations.up.sql` adds append-only
  recommendation evaluations, question snapshots, manager deviations, and
  scope freeze digests. Planning submission and selection preview/commit retain
  the protected mandatory floor and the exact freeze receipt without mutating
  historical rows.
- The Manager wizard consumes `includedByDefault=true` for Suggested now and
  clears that filter for the full approved catalog. The omitted multi-history
  question remains selectable, while the single clean-history question remains
  `UNCERTAIN_SIGNAL`, `includedByDefault=true`, and `canDefer=false`.
- The authorized Namibia/demo scope/catalog release is now deployed and
  publicly verified under immutable lock `sha256:643b4b…`; that deployment is
  evidence for the separate release task, not completion of this plan.
- Recommendation-specific local evidence is now `verified locally`:
  `scripts/test-prior-audit-recommendations.sh`, the exact focused Go and
  integration commands, mock golden tests, `scripts/generate-contracts.sh`,
  `scripts/check-contracts.sh`, React `93/93` and `728/728`, both web builds,
  `scripts/test-qualification-bootstrap.sh`, `make qualification-scenario
  TARGET=namibia/demo CONFIRM=namibia/demo:all-role-e2e` (`1 passed`, result
  `/.state/qualification/namibia/demo/results/namibia-demo-20260818t14221787052150z-e51022ecf8b34d199eb2b3f551cfc9a5.jsonl`),
  full visible-actions `4 passed`, full accessibility `5 passed`, and current
  Browser 390px/date/recommendation/full-catalog/Auditee-privacy smoke.
  Public-origin qualification is `not run` because `qualification-smoke`
  could not resolve the selected public DNS origin, and release-check is
  blocked by the dirty checkout required to preserve unrelated changes.

## Implementation progress and evidence

- [x] Phase 0 decision table and exact comparability boundary implemented in
  `apps/api/internal/application/prior_audit_recommendations.go`.
- [x] Phase 1 deterministic multi/single history fixtures and JSON golden
  oracles added; three eligible FINAL/LOCKED Audits and one eligible
  FINAL/LOCKED Audit are asserted exactly.
- [x] Phase 2 precedence, validated-clean truth table, fixed clock boundary,
  mandatory floor, full-catalog override, and auditee-safe projection added.
- [x] Phase 3 OpenAPI/generated transport, HTTP query filter, mock parity, and
  Manager wizard history/rationale presentation added.
- [x] Phase 3 append-only migration and Planning snapshot/freeze persistence
  added without updates/deletes to historical Audit/checklist/Finding/CAP/
  Evidence/report/catalog rows.
- [x] Focused server/mock/integration tests, contract checks, typecheck,
  `build:demo`, `build:http`, qualification-bootstrap replay, full React,
  visible-actions, accessibility, and diff checks are `verified locally`.
- [ ] Connected recommendation-specific public qualification and immutable
  release evidence remain `not run`/`release pending`; no production readiness
  claim is made.

## Outcome notes

The recommendation implementation is `candidate-only` and `verified locally`
for deterministic policy, mock parity, contract generation, append-only schema,
local build, qualification-bootstrap replay, and disposable local HTTP/
PostgreSQL qualification. The public connected qualification gate is `not run`
due unresolved public DNS, and release remains
`release pending` because the Workspace release check rejects the intentionally
dirty source checkout; commit/push/deploy/release actions are outside this task.
Production readiness not claimed.

## Execution Prompt

Continue from Phase 0. Read this plan, the adaptive-scope workflow, the Audit
Planning module, the current API recommendation query, and the Manager wizard.
Implement the two deterministic history fixtures first, then make the server
recommendation contract explainable and fail-closed. Preserve all historical
Audit/checklist/catalog rows, keep the full catalog override available, run the
focused and connected gates, and stop before any release or production action
unless the user separately authorizes that exact release. Report every gate as
`verified locally`, `not run`, `blocked`, `candidate-only`, or `release pending`.
