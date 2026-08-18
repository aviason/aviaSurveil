# Prior-Audit-Aware Question Recommendations

Date: 2026-08-18
Status: active — deterministic server/mock policy, append-only persistence boundary, contract transport, Manager presentation, focused tests, builds, public showcase seed, immutable `namibia/demo` release, and recommendation-specific qualification slice are `verified locally`; full connected all-role qualification is `blocked` after the recommendation slice; `candidate-only`; production readiness not claimed

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
4. Provide a visible “show full catalog”/override path. Including or restoring
   a question requires no reason. Removing or deferring a question from the
   final scope is permitted only when the server returns `canDefer=true`; it
   requires an explicit `DEFER` deviation plus manager rationale, and both are
   carried into the frozen scope decision.
5. Keep the full catalog searchable, paginated, and selectable after a
   question is omitted from the suggested set.

6. Add a server-owned, Manager-only
   `CanonicalQuestionRecommendationSummary` and return it as the required
   `recommendationSummary` on every Manager catalog response, including an
   empty or filtered page. It contains human-readable `organizationLabel`,
   `providerScopeLabel`, `regulatedTargetLabel`, `locationLabel`,
   canonical `generalInspectionType` and its
   `generalInspectionTypeLabel`, and `auditTypeFocusLabel`; server-clock
   `evaluationAsOf`; fixed `historyWindowMonths`, `historyWindowStart`, and
   `historyWindowEnd`; `comparableAuditCount`, including zero;
   `historyDeferredCount`; `focusConfigured`, canonical `focusType`, and
   `focusInspectionTypeCodes`; and `recommendationEvaluationDigest`. The
   summary is computed independently of catalog search, facet, and page
   filters. Manager presentation contracts expose human-readable Audit labels
   only and do not expose immutable Audit IDs. The summary, internal history
   signals, evaluation digest, and manager rationale are absent from every
   Auditee API response, network payload, and DOM projection.
7. Add an explicit prior-Audit explanation panel to the Manager checklist
   step. It renders the server summary for zero, one, or multiple comparable
   Audits and shows an aggregate warning when optional questions were withheld.
   Do not derive this panel or its counts from the visible 25-row catalog page.
   Load every withheld row through a dedicated, Manager-only,
   scope-bound `historyDeferredQuestions` cursor query that ignores catalog
   search, facet, and visible-page filters. Traverse until `nextCursor=null`,
   reject duplicate question-version IDs, and require the unique loaded count
   to equal `recommendationSummary.historyDeferredCount` before enabling a
   bulk action.
8. Define result wording without upgrading an arbitrary observation to
   validated-clean evidence. `latestComparableResult` means the chronologically
   latest outcome from an eligible immutable `FINAL`/`LOCKED` comparable Audit
   and is labelled “Latest comparable outcome,” never “latest validated
   result.” A validated-clean statement is shown only from observations that
   pass the full validated-clean truth table. The deferred-question projection
   therefore also returns `validatedCleanAuditCount` and
   `lastValidatedCleanAt`; the panel renders “validated clean in N of M
   comparable Audits” plus the date only when those values prove it. Missing,
   unknown, non-final, generic `NOT_APPLICABLE`, or merely accepted-remediation
   observations must not contribute to `validatedCleanAuditCount`. Each listed
   question also shows its human-readable prompt, state/classification, server
   rationale, and human-readable outcome summary.
9. Provide one Manager action to restore all and only the current
   history-deferred membership set:
   `historyDeferred=true`, `recommendationState=RECENTLY_VERIFIED`,
   `classification=DEFER_ELIGIBLE`, `includedByDefault=false`,
   `canDefer=true`, `canSelect=true`, and still applicable and authorized for
   the active scope revision. `OUTSIDE_FOCUS` rows are explicitly excluded.
   Restoring requires no reason and unions the eligible IDs into the current
   pending selection without losing an unrelated pending addition or removal.
   One user action may use ordered, revision-safe batches of at most 500 IDs;
   each batch uses the latest acknowledged revision and an idempotent operation
   identity. A repeated action is a no-op. A scope/type change or stale response
   is rejected before client state mutation. An authorization/revision failure
   stops later batches, preserves acknowledged and unrelated pending edits,
   identifies unapplied IDs, refreshes server state, and displays a retryable
   error instead of silently claiming full restoration. Restored selections
   still pass selection review, snapshot, deviation, and freeze gates.
10. Make recommendation filtering follow the selected scope in two ordered
    layers. Layer one is authoritative sealed-catalog membership plus exact
    authorized organization/provider-scope/regulated-target and general
    inspection-type applicability; full catalog cannot bypass it. Layer two is
    advisory audit-type focus and may suppress only optional Suggested rows;
    mandatory, safety-critical, open, repeat, overdue, changed, and unknown
    precedence still wins. Canonical aliases `RAMP` and `CABIN` normalize to
    `RAMP_INSPECTION` and `CABIN_INSPECTION`. The complete supported mapping is:

    `generalInspectionType` is the normalized canonical `applicationType`
    persisted on the active scope draft; it is the exact applicability and
    comparable-history partition used by Layer one. `auditTypeFocus` is the
    Layer-two mapping from that canonical value to approved question
    `inspectionTypeCodes`. The Manager-controlled advanced `checklistFocus`
    facet is only an additional visible-list filter and must not alter either
    server default, summary count, or deferred membership.

    | Canonical audit type | `focusConfigured` | Focus inspection-type codes |
    |---|---:|---|
    | `RAMP_INSPECTION` | `true` | `ON_SITE_INSPECTION`, `PERIODIC_SURVEILLANCE` |
    | `CABIN_INSPECTION` | `true` | `DOCUMENT_AND_RECORD_REVIEW`, `PERIODIC_SURVEILLANCE` |
    | `CHANGE_APPROVAL` | `true` | `CHANGE_APPROVAL` |
    | `DOCUMENT_AND_RECORD_REVIEW` | `true` | `DOCUMENT_AND_RECORD_REVIEW` |
    | `FOLLOW_UP` | `true` | `FOLLOW_UP` |
    | `INITIAL_CERTIFICATION` | `true` | `INITIAL_CERTIFICATION` |
    | `ON_SITE_INSPECTION` | `true` | `ON_SITE_INSPECTION` |
    | `PERIODIC_SURVEILLANCE` | `true` | `PERIODIC_SURVEILLANCE` |
    | `RENEWAL` | `true` | `RENEWAL` |
    | `SPECIAL_PURPOSE` | `true` | `SPECIAL_PURPOSE` |

    A blank pre-scope value reports `focusConfigured=false` and applies no
    advisory focus; an unsupported nonblank type is invalid and fails closed.
    With no comparable history, Layer two still excludes optional
    out-of-focus questions from Suggested now. Full catalog may reveal those
    questions but cannot bypass Layer one, authorization, `canSelect`, or the
    mandatory floor.

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
6. Add deterministic profile `prior-audit-no-history-scope-filter` with these
   exact question-version IDs and golden membership:
   `Q-NO-HISTORY-IN-FOCUS-OPTIONAL`,
   `Q-NO-HISTORY-OUTSIDE-FOCUS-OPTIONAL`,
   `Q-NO-HISTORY-OUTSIDE-FOCUS-MANDATORY`,
   `Q-NO-HISTORY-WRONG-PROVIDER`, `Q-NO-HISTORY-WRONG-TARGET`, and
   `Q-NO-HISTORY-WRONG-GENERAL-TYPE`. Its exact Suggested set is the in-focus
   optional plus the protected mandatory (`suggestedCount=2`); its exact full
   applicable set additionally contains the out-of-focus optional
   (`fullApplicableCount=3`). Wrong-provider, wrong-target, and wrong-general-
   type IDs are absent from both. Assert HTTP/mock/UI parity and
   `suggestedCount < fullApplicableCount`; a 1,310-row Suggested result fails
   this oracle.
7. Add deterministic profile `prior-audit-history-deferred-multipage` with a
   fixture-owned exact sorted `expectedHistoryDeferredIds` oracle containing
   `Q-HISTORY-DEFERRED-001` through `Q-HISTORY-DEFERRED-026`, a page size of
   25, and `historyDeferredCount=26`. The first and second cursor pages contain
   exactly 25 and 1 unique IDs. The fixture also contains explicit
   not-restored IDs for `OUTSIDE_FOCUS`, `canDefer=false`, `canSelect=false`,
   no-longer-applicable, and unauthorized cases. Assert the exact restored and
   not-restored lists, cursor completeness, idempotent replay, preservation of
   unrelated pending add/remove edits, sequential batches of at most 500,
   stale-scope response rejection, and visible partial-failure/retry behavior.
8. Add or retain these exact Go test names:
   `TestScopeRecommendation_NoHistoryScopeFilterGolden`,
   `TestScopeRecommendation_FocusMappingAllSupportedTypes`,
   `TestScopeRecommendation_ValidatedResultDisplaySemantics`,
   `TestPriorAuditRecommendation_RecommendationSummaryManagerOnly`,
   `TestPriorAuditRecommendation_HistoryDeferredCursorCompleteness`,
   `TestPriorAuditRecommendation_RestoreAllRevisionSafety`, and
   `TestPriorAuditRecommendation_ListDetailMockSnapshotParity`. Add matching
   named cases `no-history scope filter golden`, `history-deferred cursor and
   restore-all golden`, and `list detail mock snapshot parity` to
   `src/mock/prior-audit-recommendations.test.ts` and
   `src/features/planning/new-audit-wizard.test.tsx`.
9. Require list, question detail, mock, and frozen recommendation snapshot to
   match exactly for `recommendationState`, `classification`,
   `includedByDefault`, `canDefer`, history/comparable/validated-clean counts,
   latest-comparable and last-validated-clean fields, signal codes, rationale,
   guardrails, focus configuration, and evaluation digest. Detail reload may
   not fall back to AI advisory or lose recommendation evidence. Add an
   authenticated browser oracle named
   `manager-prior-audit-recommendation-oracle` for exact Suggested/full counts,
   present/absent IDs, summary counts, restore-all membership, and Auditee
   privacy at 1440x900 and 390x844.

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
  approved catalog and can be included or restored without a reason. Only its
  later removal or deferral from final scope requires `canDefer=true`, an
  explicit `DEFER` deviation, and manager rationale.
- Every omitted/uncertain question exposes history count, last result, signal
  basis, rationale, and guardrails in the question presentation or the
  prior-Audit explanation panel.
- When history-based omissions exist, the checklist shows an aggregate
  warning, lists each omitted question with its reason, and offers one action
  to restore all history-deferred optional questions; restoring does not
  require a manager reason. The list is loaded through its dedicated cursor to
  completion, has unique IDs, and its count equals the server-owned summary
  count independently of the visible catalog page, search, or facets.
- `recommendationSummary` is present even when the result page is empty or the
  comparable-Audit count is zero. Its human-readable labels, fixed history
  window, focus configuration, counts, and evaluation digest are server owned;
  Manager UI payloads contain no immutable Audit IDs, and Auditee API/network/
  DOM projections contain none of the summary or internal recommendation data.
- The UI labels an arbitrary newest eligible result only as “Latest comparable
  outcome.” It claims “validated clean in N of M” and shows a last validated-
  clean date only from truth-table-valid clean observations; all other outcome
  classes are excluded from that count.
- Restore all selects exactly the history-deferred `RECENTLY_VERIFIED` +
  `DEFER_ELIGIBLE` + `includedByDefault=false` + `canDefer=true` +
  `canSelect=true` + still-applicable/authorized set and never an
  `OUTSIDE_FOCUS` row. It preserves unrelated pending edits, is idempotent,
  respects 500-ID revision-safe batches, rejects stale scope/type responses,
  and makes authorization/revision partial failure visible and retryable.
- The checklist explicitly reports whether the selected scope has no
  comparable history, one prior Audit, or multiple prior Audits, and applies
  the same provider/target/general-type plus audit-type focus boundary in all
  three cases.
- Empty history never causes the Suggested view to show optional questions
  outside the selected audit-type focus; the full approved catalog remains
  the explicit, still-authorized override.
- `prior-audit-no-history-scope-filter` returns exactly two Suggested and three
  full-applicable questions with the IDs defined in Phase 4; wrong provider,
  target, and general-type IDs never appear. Every supported canonical audit
  type returns the exact `focusConfigured` and focus-code mapping in Phase 3.
- `prior-audit-history-deferred-multipage` returns exactly 26 unique deferred
  IDs across 25+1 rows and restores exactly its golden eligible set while
  preserving every golden not-restored ID.
- No mandatory, safety-critical, open, repeat, overdue, changed, or unknown
  question is silently omitted.
- Recommendation behavior is identical across list, detail, HTTP, mock, React,
  and frozen snapshot surfaces for the same immutable fixture and scope; no
  detail or snapshot path falls back to advisory-only fields or loses the
  server evaluation digest.
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
- The authorized Namibia/demo recommendation release is publicly deployed under
  the exact lock at
  `/Users/marlonjd/Developer/monorepos/avia/releases/namibia/demo.lock.json`.
  The append-only public history manifest is
  `/Users/marlonjd/Developer/monorepos/avia/apps/surveil/deploy/qualification/namibia-demo-prior-audit-history.json`.
  Its three `FINAL`/`LOCKED` Audits use the same organization, provider scope,
  regulated target, `RAMP_INSPECTION`, and the isolated presentation location
  `Namibia Demo AGA History Showcase`; the earlier
  `Namibia Demo AGA Qualification Operator` rows remain unchanged.
- Recommendation-specific local evidence is now `verified locally`:
  `scripts/test-prior-audit-recommendations.sh`, the exact focused Go and
  integration commands, mock golden tests, `scripts/generate-contracts.sh`,
  `scripts/check-contracts.sh`, focused React tests (`27 passed`), the boolean
  query-filter regression test (`24 passed`), typecheck, both web builds,
  `scripts/test-qualification-bootstrap.sh`, and `git diff --check`.
- Public release evidence is `verified locally`:
  `make release-check TARGET=namibia/demo`, the exact `namibia/demo`
  Terraform plan/apply (`2 to add, 5 to change, 2 to destroy` per rollout),
  `make qualification-smoke TARGET=namibia/demo` (`HTTP 200`, released lock),
  and CloudWatch host evidence showing migration `version=56`, table
  permissions, catalog `questions=1310`, `prior-Audit history loaded: ...
  audits=3 manifest=sha256:701aea...`, API/Caddy listening, and four registered
  tunnel connections.
- The recommendation-specific public qualification result is at
  `/Users/marlonjd/Developer/monorepos/avia/.state/qualification/namibia/demo/results/namibia-demo-20260818t19581787072336z-54b486dbc0ac439fb0e4a720e678cf4a.jsonl`.
  Its `manager-prior-audit-recommendation-oracle` event is `verified locally`
  with `comparableAuditCount=3`, `suggestedCount=1001`,
  `fullCatalogCount=1310`, `historyDeferredCount=1`, the seeded question
  absent from Suggested, and restore/undo successful. The later all-role flow
  is `blocked` at
  `apps/web/tests/e2e/qualification-all-role.spec.ts:1041` because the existing
  Preliminary Review page did not expose the expected `DEPARTMENT_REVIEW`
  status; this does not invalidate the completed recommendation slice.
- Chrome extension control is `blocked` because the Codex Chrome extension is
  not installed in the selected Chrome profile. Computer Use was available for
  read-only inspection, but its public session was expired; the explicitly
  authorized Playwright qualification fallback supplied the connected
  recommendation evidence. Production readiness not claimed.

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
- [x] Phase 3 server-owned Manager-only `recommendationSummary`, cursor-complete
  history-deferred list, validated-clean display semantics, aggregate warning,
  per-question omission reasons, and exact restore-all behavior are implemented
  and verified through focused tests plus the public recommendation slice.
- [x] Phase 3 selected-scope filtering is verified end to end for comparable
  history and through the no-history mock guard: exact provider/target/general
  inspection type first, configured audit-type focus second, protected
  precedence and full catalog override preserved.
- [ ] Phase 3 empty-history exact-ID fixture parity remains a Phase 4 gate; the
  current mock regression proves Suggested is smaller than the full 1,310-row
  catalog but does not close the exact named oracle below.
- [x] Phase 3 append-only migration and Planning snapshot/freeze persistence
  added without updates/deletes to historical Audit/checklist/Finding/CAP/
  Evidence/report/catalog rows.
- [x] Focused server/mock/integration tests, contract checks, typecheck,
  `build:demo`, `build:http`, qualification-bootstrap replay, full React,
  visible-actions, accessibility, and diff checks are `verified locally`.
- [ ] Phase 4 no-history exact-count oracle, all-supported-type focus mapping,
  multi-page deferred cursor, restore-all revision safety, validated-clean
  wording, list/detail/mock/snapshot parity, and Manager/Auditee browser oracles
  are implemented and executed. These exact newly named gates are `not run`;
  the public recommendation slice is independently recorded above.
- [ ] Connected recommendation-specific public qualification and immutable
  release evidence are partially complete: release and recommendation slice are
  `verified locally`, while the later all-role continuation is `blocked`; no
  production readiness claim is made.

## Outcome notes

The recommendation implementation is `candidate-only` and `verified locally`
for deterministic policy, mock parity, contract generation, append-only schema,
local build, qualification-bootstrap replay, public seed/replay, immutable
`namibia/demo` release, and the recommendation-specific connected Playwright
slice. The full all-role continuation is `blocked` at the pre-existing
Preliminary Review status assertion; exact Phase 4 no-history/multi-page/
list-detail-snapshot parity gates remain `not run`. Chrome extension control is
`blocked`, and Computer Use public interaction was not run after session expiry.
Production readiness not claimed.

## Plan amendment — 2026-08-18

The original plan covered server-side history-aware recommendation states,
explainable question fields, and full-catalog override. This amendment adds
only the following implementation and acceptance scope: a server-owned,
Manager-only scope summary that also represents zero history; a filter- and
page-independent cursor-complete history-deferred list; exact validated-clean
display semantics; an aggregate warning and human-readable per-question
omission reasons; a revision-safe restore-all action with exact membership and
no restore reason; complete supported audit-type focus mapping; a deterministic
no-history count/ID oracle that prevents the 1,310-question regression; a
multi-page omission oracle; and list/detail/mock/snapshot/browser parity gates.

Including or restoring a question is not a deferral and requires no reason.
Only removing or deferring it from final scope requires `canDefer=true`, an
explicit `DEFER` deviation, and manager rationale. Restore-all excludes
`OUTSIDE_FOCUS`, preserves unrelated pending edits, and remains bounded by
authorization, applicability, revision, selection-review, snapshot, and freeze
rules. Manager presentation uses human-readable labels and does not expose
immutable Audit IDs; Auditee projections receive no summary, history signals,
evaluation digest, guardrails, or rationale.

The two kill-critic reviews covered these amendment additions only. They do not
claim that any unrelated original criterion was newly reviewed, implemented,
verified, or completed. Original plan content and prior evidence remain
preserved. Amendment implementation and recommendation-specific tests are
`verified locally`; exact Phase 4 gates remain `not run`; public release is
`verified locally`; production readiness not claimed.

## Execution Prompt

Continue from Phase 0. Read this plan, the adaptive-scope workflow, the Audit
Planning module, the current API recommendation query, and the Manager wizard.
Implement the two deterministic history fixtures first, then make the server
recommendation contract explainable and fail-closed. Preserve all historical
Audit/checklist/catalog rows, keep the full catalog override available, run the
focused and connected gates, and stop before any release or production action
unless the user separately authorizes that exact release. Report every gate as
`verified locally`, `not run`, `blocked`, `candidate-only`, or `release pending`.
