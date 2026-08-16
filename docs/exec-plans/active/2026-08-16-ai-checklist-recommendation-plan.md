# Offline AI Checklist Recommendations And Deterministic Reuse

This ExecPlan is a living implementation record. Keep its progress,
decisions, discoveries, verification, and outcome synchronized with the
working tree.

## Status

- Plan status: `active`.
- Scope: the canonical approved AGA catalog and the connected Department
  Manager New Audit flow.
- Current evidence: implementation is `verified locally` through the clean
  connected `namibia/dev` all-role qualification and runtime failure matrix;
  exact `namibia/demo` immutable release and public qualification remain
  pending.

## Objective

Generate a complete AI advisory enrichment for all 1,310 immutable approved
question versions before catalog import, validate it with closed deterministic
contracts, import it without creating an approval or publication gate, and use
the imported values plus immutable prior-audit history to produce an
explainable checklist recommendation in the real API and Manager UI.

## User-visible outcome

The Manager can choose an audit type and use linked dropdown facets for form,
domain, topic, AI risk band, applicability, and prior-audit state. The product
shows a deterministic recommendation grouped into required, recommended, and
previously verified/deferred questions. Every item explains why it is included
or omitted, and every omitted question remains searchable and re-addable. A
new catalog question version, unresolved Finding/CAP, non-clean evidence,
scope change, or due recurrence re-enters the recommendation automatically.

## Boundaries

- The Aviation source package remains the sole approved question source. Its
  root digest, question IDs, versions, text, ordering, and counts remain
  unchanged.
- AI output is an `AI_ADVISORY` enrichment and never an approval, attestation,
  source manifest, publication decision, legal conclusion, or deployment
  blocker.
- No runtime LLM, expert approval command, bulk approval, personal MFA, or
  second publication gate is introduced.
- The full 1,310-question catalog remains available. Recommendation staging
  is explicit and does not silently preselect every question.
- Unknown AI risk or applicability is never treated as low risk; it remains in
  the reviewable recommendation set with an explicit unknown reason.
- Target and control tenant history are keyed independently and cannot affect
  one another.

## Ordered work

1. Freeze the offline input, AI model/prompt/run metadata, controlled codes,
   question identity join, recommendation policy, and deterministic artifact
   digest. Add validator and replay/drift tests.
2. Add an append-only AI enrichment data plane and idempotent foundation →
   roster → approved catalog → AI enrichment loader path. Replays must be
   no-op; artifact or catalog drift must fail closed.
3. Add server-owned catalog facets and recommendation projections. Evaluate
   audit-type applicability, AI advisory risk, immutable question-version
   changes, prior final audit coverage, Finding/CAP/evidence state, and
   recurrence policy without calling AI.
4. Replace free-text catalog filters with linked single/multi-select controls,
   add recommendation groups and reason details, and keep all exact question
   selection commands bound to the existing scope snapshot/digest flow.
5. Add focused Go/Node/React/API contract tests, local connected qualification,
   responsive Manager coverage, and update the canonical demo evidence.

## Acceptance criteria

- Exactly 1,310 enrichment rows join the exact approved question-version
  identities and source catalog root digest.
- Invalid, duplicate, missing, stale, unknown-code, or reordered artifact input
  fails validation without a partial write.
- Second import is idempotent and drift is fail-closed.
- Facet options are server-derived, linked to active filters, and never based
  only on the current 25-row page.
- Recommendation explanations are visible, deterministic, and re-addable.
- A prior clean question can be deferred only when its version, scope,
  recurrence, Finding/CAP, and evidence conditions allow it; unresolved or
  non-clean work is always reintroduced.
- Native API/web gates, `git diff --check`, and the connected Manager flow pass.

## Progress

- [x] Gate 0: read workspace and nested Surveil instructions; capture clean
  repository status with `make repos-status`.
- [x] Offline AI advisory artifact and validator. The checked-in artifact has
  exactly 1,310 rows, deterministic provenance, and a semantic digest.
- [x] Enrichment migration, loader, replay, and drift boundary. Migration 48,
  append-only immutable enrichment, real PostgreSQL replay/drift/permission
  checks, and foundation → roster → catalog → enrichment bootstrap pass.
- [x] Faceted catalog and deterministic recommendation API. Server-derived
  linked facets, prior-scope history, evidence state, recurrence, open work,
  and advisory reason codes are covered by focused Go tests and local HTTP.
- [x] Connected responsive Manager UI. Prompt-first cards, linked
  multi-select facets, explicit staging, immutable IDs, and desktop/mobile
  dossier behavior are implemented and covered by web gates.
- [x] Regression, local connected E2E, and evidence update. The clean local
  `namibia/dev` scenario `local-ai-dev-20260816-r4` passed (`1 passed`, 3.2m)
  with 1,310 cursor-traversed questions, target/control isolation, real upload
  `CLEAN` / `PENDING_CAA_REVIEW`, evidence-based closure, Final issuance, Admin
  evidence visibility, and 390x844 responsive checks. Generated credential files
  and browser processes were absent after the run. The exact runtime failure
  matrix also passed gateway-only exposure, exact network membership,
  dependency recovery, worker restart, and generated-secret log scanning.
  Public demo remains pending.

## Verification notes

- The local `namibia/demo` profile intentionally uses the fail-closed disabled
  scanner, so an upload there remains `QUARANTINED` / `NOT_READY`; no
  development-only scanner bypass was added to the demo profile. The required
  clean evidence qualification therefore runs on `namibia/dev`, while exact
  public `namibia/demo` qualification must use its managed scan integration.
- A previous local E2E failure caused by the new linked facet replacing a
  free-text field was corrected in the test to exercise the real multi-select
  control. A second presentation-contract failure caused by locale grouping
  (`1,310` versus `1310`) was corrected in the pagination label.
- Roster replay now treats an issuer mismatch as drift instead of silently
  accepting an idempotent row count; the regression test fails closed before a
  stale identity authority can reach the connected session flow.

## Execution prompt

Continue this plan from its current progress. Preserve immutable approved
catalog identity and all unrelated user changes. Work on the current branch,
read the nested repository instructions before edits, use `apply_patch`, keep
source/docs/tests in English, run `make repos-status` at each checkpoint, and
never add an AI runtime or an approval/publication gate. Verify focused native
tests before broader gates and record literal evidence labels.
