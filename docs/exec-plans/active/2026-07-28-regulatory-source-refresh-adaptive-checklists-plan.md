# Regulatory Source Refresh And Adaptive Checklists

## Objective

Extend the `candidate-only` regulatory-knowledge pilot with an idempotent NCAA
public-source synchronization foundation and an advisory risk-based checklist
scope model.

The user-visible outcome is:

- a hash-addressed local source context for every public document listed across
  NAMCATS pages 1-3 and the NAMCAR-NAMCATS index;
- a tracked source manifest that distinguishes download identity, content hash,
  version/date signals, and extraction state;
- harness-readable derived applicability assessments that cite exact source
  hashes, pages, clauses, extracted-text locators, and review boundaries
  without copying full regulatory text into Git;
- a six-month reconciliation and annual expert-validation policy with
  event-driven change intake;
- checklist scope recommendations that distinguish mandatory, focused, and
  rotational-sampling controls without silently changing a published checklist
  or treating missing history as compliance.
- a mandatory checklist-generation contract: every proposed question exposes
  both its scope classification and its regulatory trace, or an explicit
  blocking source-mapping gap; neither may be silently absent at publication.

## Scope

- Download only public documents reachable from the three paginated NCAA
  NAMCATS library pages plus the public NAMCAR-NAMCATS index linked there.
- Store downloaded bytes and extracted text under ignored
  `.local/aviasurveil360/regulatory-sources/`.
- Generate a tracked JSON manifest under `docs/regulatory-sources/` containing
  exact URLs, filenames, observed page metadata, byte counts, SHA-256 digests,
  and extraction status.
- Generate compact tracked derived-context records for reviewed source subsets;
  keep full extracted text in the ignored vault and reference it by hash,
  locator, page, and clause.
- Add a reusable bounded synchronization script with content-type, PDF
  signature, file-count, and source-host checks.
- Extend the regulatory mapping contract with refresh policy and advisory
  checklist-scope recommendations.
- Add a governed onboarding path for existing checklist questions. Treat their
  operational history as a candidate input, not as regulatory authority; map,
  review, and version them against the current approved source chain before
  publication.
- Present refresh governance and current six-question recommendations in the
  Admin Regulatory Library and Checklist Builder.
- Preserve immutable published checklist identity and Department Manager
  publication authority.

## Explicit Exclusions

- No ICAO or NCAA self-assessment portal access.
- No authenticated form submission, credential use, CAPTCHA, or restricted
  source access.
- No bulk mirroring beyond the three public NAMCATS pagination pages.
- No Git storage of downloaded PDF/XLSX bytes or extracted full text.
- No autonomous legal interpretation, checklist publication, Finding,
  enforcement, certification, or closure decision.
- No automatic omission of a mandatory, safety-critical, changed, overdue,
  unknown, or insufficient-history control.
- No use of an existing checklist, prior answer, or clean history as a
  substitute for an applicable, versioned regulatory source and approved CAA
  procedure.
- No claim that a downloaded source has been clause-mapped or technically
  validated merely because its bytes and hash are present.
- No root legacy-demo changes, branch operations, commit, push, deployment, or
  production mutation.

## Assumptions And Ownership Boundaries

- Public NCAA download URLs are authorized read-only source inputs for this
  local candidate.
- The local source vault is development context, not a production regulatory
  repository.
- Six-month reconciliation detects source-set/content changes; annual technical
  validation remains a human regulatory-owner gate. A new publication may
  trigger an earlier review.
- AI recommendations remain advisory. The Inspector or Department Manager must
  accept or adjust scope with a recorded rationale.
- Existing Audits remain pinned to their exact checklist version. A validated
  mapping change produces a new checklist Draft/version for future use.

## Repository Orientation And Affected Interfaces

- Source synchronization:
  - `scripts/regulatory/sync-ncaa-namcats.mjs`
  - ignored `.local/aviasurveil360/regulatory-sources/`
  - `docs/regulatory-sources/ncaa-namcats-manifest.json`
  - `docs/regulatory-sources/derived/`
- Regulatory contract and candidate projections:
  - `api/openapi/source/schemas/platform.json`
  - generated Go and TypeScript transport artifacts
  - `apps/web/src/backend/backend.ts`
  - `apps/web/src/mock/seed-data.ts`
  - `apps/api/internal/configuration/workspace.go`
  - `apps/api/internal/testprofile/regulatory_pilot.go`
- User interfaces:
  - `apps/web/src/features/admin/regulatory-library-page.tsx`
  - `apps/web/src/features/admin/checklist-builder-page.tsx`
- Product model/workflow and focused tests.

## Ordered Tasks

### Task 1 - Source Synchronization And Context

- [x] Implement bounded, idempotent three-page discovery and download.
- [x] Download every unique PDF listed across pages 1-3 and the linked index
  into the ignored local vault.
- [x] Verify PDF signatures, workbook ZIP signature, byte counts, and SHA-256
  digests.
- [x] Extract searchable text where supported, run resumable OCR for every
  individual page without a text layer, and record literal extraction results.
- [x] Write the tracked source manifest without copying full document content
  into the repository.

### Task 2 - Refresh And Adaptive-Scope Model

- [x] Add refresh cadence, event-driven review, source-change state, and
  publication guardrails to the typed regulatory contract.
- [x] Add advisory per-question scope classifications and current candidate
  recommendations.
- [x] Keep missing/insufficient history closed: it cannot justify automatic
  deferral.
- [x] Keep published checklist identity immutable; recommendations affect a
  proposed inspection package or future Draft only.

### Task 3 - User Experience And Documentation

- [x] Show source-refresh status, next reconciliation, next annual validation,
  and change-to-Draft workflow in the Regulatory Library.
- [x] Show mandatory/focused/rotational recommendations and exact rationales in
  Checklist Builder.
- [x] Document source refresh, clause-diff impact analysis, and adaptive-scope
  guardrails in the product model/workflow.

### Task 4 - Verification

- [x] Test all-page source discovery/manifest stability without requiring
  network.
- [x] Run contract drift, TypeScript, focused component/domain, Go, demo build,
  legacy boundary, responsive browser, cleanup, and diff checks.
- [x] Record literal download/OCR/extraction/test results and unresolved technical
  validation gaps.

### Task 5 - Harness-Readable Part 127 / Part 140 Assessment

- [x] Record exact Part 127 and Part 140 source identities, hashes, extracted
  text locators, page/section evidence, and version-authority caveats.
- [x] Classify Part 127 as operation-type conditional and Part 140 as
  organisation/SMS-level applicable without treating either as an automatic
  direct basis for every cabin/ramp question.
- [x] Record per-question candidate implications and keep every legal,
  authority, publication, and controlled-procedure decision human-gated.
- [x] Route the derived assessment through the regulatory-source README,
  harness registry, conceptual data model, plan, and durable tracker.
- [x] Validate JSON shape, source hashes/locators, Markdown links, harness docs,
  demo boundaries, and diff cleanliness.

### Task 6 - Make Scope And Regulatory Trace Mandatory At Checklist Generation

**Status:** Task 6 implementation and focused local verification are `verified
locally`; independent read-only implementation and final-delta review are
accepted with no Critical or Important finding. The result remains
`candidate-only`, `release pending`, and `production-ready: not established`.
Earlier Task 1-5 evidence does not substitute for this requirement. See
[Task 6 local evidence](../../demo-evidence/REGULATORY_SOURCE_REFRESH_TASK6_2026-07-31.md).

**Files**

- Modify `api/openapi/source/schemas/platform.json` and generated transport
  artifacts to carry `scopeRecommendation` and `regulatoryTrace` state for
  every proposed/published checklist question.
- Modify the governed checklist-generation/publication application boundary to
  reject publication when either state is absent, unresolved, or stale.
- Modify `apps/web/src/features/admin/checklist-builder-page.tsx` so every
  generated question visibly shows its classification and regulatory trace; a
  missing trace renders `SOURCE_MAPPING_REQUIRED`, never an empty citation.
- Add focused API, backend, and React tests for positive, source-gap,
  stale-source, mandatory, rotational-sample, and deferral branches.

**Work**

- [x] Define the per-question generation record as two required views:
  `scopeRecommendation` (classification, signals, history basis, rationale,
  guardrails, and approval state) and `regulatoryTrace` (source title,
  immutable source version/hash, locator/page/clause, source type,
  applicability, national/CAA-procedure mapping, verification objective, and
  expected Evidence).
- [x] Allow a Draft question with an explicit `SOURCE_MAPPING_REQUIRED` gap so
  a reviewer can complete it, but prevent that question from being published,
  deferred, or presented as validated until its trace is technically approved
  and a Department Manager records the separate publication decision.
- [x] Apply a hybrid migration rule to existing checklist questions: retain
  wording, result history, and known operational intent as candidate input;
  use the current approved regulatory/CAA-procedure chain as the sole
  authority; create a new immutable Draft/version rather than rewriting a
  published version or an in-progress Audit.
- [x] Require every generation decision to name its origin as
  `REGULATORY_TRACE`, `EXISTING_CHECKLIST_CANDIDATE`, or `HYBRID_RECONCILED`.
  `HYBRID_RECONCILED` means the legacy question was matched to the approved
  trace and any wording, Evidence, or scope changes are visible to the
  reviewer; legacy content is never elevated to an authority source.
- [x] Prevent publication when any question lacks a visible classification,
  trace, rationale, source-currentness result, or required technical review.
  A source hash/version change invalidates the affected trace and creates a
  new impact-review Draft rather than silently retaining eligibility.

**Verification And Acceptance**

- [x] Add a contract test proving that a generated Draft always returns both
  views or the literal `SOURCE_MAPPING_REQUIRED` state.
- [x] Add backend/publication tests that reject a missing trace, stale source,
  unresolved source gap, omitted classification, and automatic deferral of a
  mandatory, safety-critical, or unknown-history question.
- [x] Add a React test proving the Checklist Builder shows classification,
  inclusion rationale, source title/version/locator, applicability, expected
  Evidence, and legacy/hybrid origin for every question.
- [x] Add a migration test proving an existing checklist produces a new
  candidate version, preserves the historical published version, and cannot
  become authoritative without source mapping, technical approval, and a
  separate publication decision.
- [x] Run focused contract, Go, React, demo-boundary, and `git diff --check`
  gates. Record fresh literal results and retain the `candidate-only` boundary
  until the responsible owners complete their review.

## Commands And Expected Observations

```bash
node scripts/regulatory/sync-ncaa-namcats.mjs
node scripts/regulatory/sync-ncaa-namcats.mjs --verify-only
```

Expected: exactly 57 unique PDFs plus the linked index are present with stable
hashes; repeated execution does not redownload unchanged valid bytes.

```bash
./scripts/check-contracts.sh
npm --prefix apps/web run typecheck
npm --prefix apps/web test -- src/features/admin/admin-secondary-pages.test.tsx src/backend/transport-mappers.test.ts
npm --prefix apps/web run build:demo
```

Expected: generated contract drift is clean and focused UI/domain gates pass.

```bash
env GOCACHE=/private/tmp/avia-regulatory-go-cache go test ./internal/configuration ./internal/httpapi
node --test tests/checklist-management-smoke.test.js tests/manager-checklist-management-smoke.test.js tests/demo-boundary-smoke.test.js
git diff --check
```

Expected: candidate HTTP/configuration code compiles, legacy checklist and demo
boundaries remain intact, and the diff has no whitespace errors.

## Verification And Acceptance Criteria

- Every downloaded source has an exact URL, stable filename, size, and SHA-256.
- The sync is bounded to the authorized host/path/page and rejects unexpected
  content.
- Source presence, extraction, clause mapping, expert validation, and checklist
  publication remain distinct states.
- A source change creates an impact-review proposal, never a silent checklist
  mutation.
- Adaptive recommendations cite exact history/risk signals and retain mandatory
  controls.
- Insufficient history cannot be presented as a clean compliance history.
- The current six-question pilot remains `EXPERT_REVIEW_REQUIRED`.
- Local evidence remains `candidate-only`.
- A checklist question cannot be published unless its scope classification and
  regulatory trace are both visible, current, technically approved, and
  accompanied by the required inclusion/defer rationale.
- Existing checklist content is a migration candidate and historical
  operational input only; the current approved regulatory/CAA-procedure trace
  remains the authority. A reconciled result is published only as a new
  immutable version.

## Risks, Dependencies, Idempotence, And Recovery

- NCAA may change filenames, pagination, or content without changing a visible
  title. SHA-256 and source-set diffs fail visibly.
- Some large/image-heavy PDFs may yield limited extracted text. Extraction state
  is recorded separately and visual/source review remains required.
- A public page may contain drafts and superseded documents. Download presence
  is not authority; the manifest preserves observed labels and technical experts
  decide current applicability.
- The sync writes to an ignored, explicit local vault and is idempotent for
  unchanged valid bytes. Recovery is deletion of that exact vault subtree and
  regeneration from the manifest/script; no production state is involved.
- Contract/UI changes are additive and can be reverted at file level without
  rewriting the immutable published checklist.

## Progress

- 2026-07-28: Plan created from the explicit request to download the supplied
  NAMCATS page-1 sources and make regulatory refresh plus experience-informed
  checklist scope a supported system behavior.
- 2026-07-28: Downloaded and hash-verified the bounded 21-file collection
  (20 PDFs plus one workbook), totaling 543,839,488 bytes. The manifest records
  7,136 PDF pages, 5,812 pages with searchable text, and five PDFs requiring
  OCR/source-owner review because no searchable text was detected.
- 2026-07-28: Re-ran the live synchronizer after completion; all 21 source
  files reported `REUSED_PAGE_METADATA_UNCHANGED`, proving that unchanged
  verified bytes are not downloaded again.
- 2026-07-28: User expanded the authorized scope to NAMCATS pages 2-3 and
  requested OCR wherever required. The plan was reopened; Apple Vision OCR is
  local, page-checkpointed, resumable, and applies to image-only pages even
  inside otherwise searchable PDFs.
- 2026-07-28: Added the typed six-month/annual/event-driven refresh policy and
  advisory scope model. The current scenario retains two `MANDATORY_CORE`
  questions, expands PBE to `FOCUSED_FULL`, and keeps three questions as
  `ROTATIONAL_SAMPLE`; none is `DEFER_ELIGIBLE` because history is
  insufficient.
- 2026-07-28: Contract 16/16, source-sync 2/2, full React 651/651, focused
  React 21/21, Go configuration/HTTP, root smoke 3/3, TypeScript, demo build,
  source verification, representative PDF visual review, and diff checks pass.
  In-app browser checks at configured 1440x900 and 390x844 viewports show the
  new Regulatory Library and Checklist Builder content with no document
  overflow and no warning/error logs; task-owned tabs and Vite listener were
  cleaned up.
- 2026-07-28: Completed the expanded three-page baseline at 57 unique PDFs
  plus one index workbook (605,250,466 bytes and 12,069 PDF pages). Embedded
  text covered 10,736 pages. Local Apple Vision OCR was requested for 1,333
  pages across 13 PDFs and recognized text on 1,324 of them, producing ordered
  context for 12,060 pages and 26,284,924 characters. Nine pages produced no
  text; representative visual review confirmed blank pages.
- 2026-07-28: Re-ran the complete synchronizer; all 58 documents reported
  `REUSED_PAGE_METADATA_UNCHANGED`. The strengthened local verifier confirmed
  exact page-level listing counts, unique identities, signatures, sizes,
  hashes, extraction strategy, and access-boundary flags.
- 2026-07-28: Fresh final verification passed: contracts 16/16, source-sync
  4/4, full React 651/651, focused React 21/21, Go configuration/HTTP, root
  smoke 4/4, TypeScript, demo build, source verification, PDF visual review,
  and diff checks. The 1440x900 and 390x844 browser checks showed no horizontal
  overflow and no warning/error logs. The source page showed 58 files, the
  tracked manifest, six-month reconciliation, and annual expert validation;
  Checklist Builder showed two `MANDATORY_CORE`, one `FOCUSED_FULL`, and three
  `ROTATIONAL_SAMPLE` recommendations.
- 2026-07-28: User requested that the Part 127 / Part 140 conclusions and text
  provenance be persisted in a harness-readable form. The plan was reopened
  for a compact derived-context artifact; full regulatory text remains outside
  Git and technical-expert validation remains a separate state.
- 2026-07-28: Added the source-bound Part 127 / Part 140 JSON assessment and
  human-readable companion. The record binds three source versions to exact
  manifest hashes and local full-text locators, cites 13 page/section evidence
  records, and states candidate implications for all six pilot questions.
  Focused derived-context/source-sync tests pass 10/10; harness docs, demo
  boundary, JSON parsing, full-text locator/page-marker inspection, routing,
  and diff checks pass.
- 2026-07-31: Task 6 added mandatory per-question `scopeRecommendation` and
  `regulatoryTrace` views, exact origin values, literal source-gap Drafts,
  fail-closed review/publication/deferral/package validation, source-staleness
  impact Drafts, and immutable legacy-to-hybrid reconciliation. OpenAPI,
  generated transports, semantic mock, canonical HTTP/PostgreSQL persistence,
  and Checklist Builder now project the same shape. Fresh local contract, Go,
  React, demo-boundary, local-source verification, and 1440×900/390×844 HTTP
  profile checks passed. The positive path remains the explicit synthetic
  profile; independent Task 6 acceptance and final cleanup checks are pending.
- 2026-07-31: Task 6 now keeps raw source observations inert until an explicit
  append-only source-currentness activation records the exact
  predecessor/current snapshot hashes. A raw source-change import rolls back
  atomically with no partial lineage; an activation creates the immutable
  impact-review Draft that one or more candidate roots can bind. The mock
  Regulatory Library derives V1/V2 rows from immutable source lineage, and
  the canonical activation transport rejects omitted or unknown fields while
  preserving explicit-null predecessor semantics.
- 2026-07-31: Independent read-only Task 6 review and final delta review found
  no Critical or Important implementation finding. The reviewer confirmed the
  atomic failed-import boundary, append-only currentness chain, exact V1/V2
  mock lineage, closed activation transport, and contract-gate coverage. The
  reviewer did not execute commands; owner-run local evidence remains scoped
  to the Task 6 evidence record.
- 2026-07-31: Final Task 6 closure checks passed: bounded source sync and
  verify-only reported 58 documents / 605,250,466 bytes; contracts passed
  16/16; focused React passed 19/19 and 28/28; Go, demo build, root smoke,
  canonical PostgreSQL/HTTP profile, and documentation smoke passed; and
  `git diff --check` was clean. The disposable profile removed its Docker
  containers, volumes, network, Vite, and runtime directory; final filtered
  process and Docker checks found no Task 6 residue. The pre-existing dirty
  worktree was preserved without staging, commits, or branch operations.

## Decisions

- Use event-driven intake plus six-month reconciliation and annual expert
  validation.
- Keep source bytes/text out of Git; track the source manifest and hashes.
- Keep adaptive scope advisory and closed by default. “No recorded problem” is
  not equivalent to verified compliance.
- Maintain a full-scope maximum interval and never auto-defer mandatory or
  newly changed controls.
- Use a hybrid onboarding model for existing checklists: preserve useful
  question wording and history, but require each question to reconcile to a
  current approved source/CAA-procedure trace before publication.

## Discoveries

- NAMCATS pages 1, 2, and 3 list 20, 20, and 18 PDFs respectively. One PDF is
  repeated across pages 1 and 2, producing 57 unique PDFs plus one index
  workbook.
- The page includes large and potentially image-heavy documents; total local
  source volume is material and must not enter normal Git artifacts.
- The complete public NAMCATS pagination contains multiple versions, drafts,
  and signed documents. Presence in the source collection does not establish
  which version applies.
- Page 2 supplies the October 2024 OPS Parts 91, 121, 127, and 135 documents;
  page 3 supplies the signed 2025 Part 140 document and an older signed Part
  140 version. Exact applicability still requires NCAA technical-expert
  validation.

## Outcome Notes

The all-pages regulatory-source baseline, adaptive-scope candidate, and
harness-readable Part 127 / Part 140 assessment are `verified locally` and
`candidate-only`. The assessment classifies Part 127 as
`OPERATION_TYPE_CONDITIONAL` and Part 140 as `SYSTEM_LEVEL_APPLICABLE`; it
retains direct-source gaps and all expert/publication gates. Full regulatory
bytes, extracted context, and page-level OCR checkpoints remain in the ignored
local vault. Source capture, extraction, and a candidate assessment do not
establish legal authority, publication approval, or expert validation.

Task 6 implementation, focused local verification, and independent read-only
acceptance are `verified locally`. It makes scope classification and
regulatory trace mandatory at checklist generation and publication, while
onboarding existing checklists as non-authoritative candidate inputs through
controlled hybrid reconciliation. Real source-owner and responsible Department
Manager decisions remain external `blocked` decisions; this local result stays
`candidate-only` and `release pending`.

## Execution Prompt

Complete Task 6 before treating the adaptive-scope candidate as a reusable
checklist-generation capability. Every proposed question must show its scope
classification, inclusion/defer rationale, and a current regulatory trace or
the literal blocking state `SOURCE_MAPPING_REQUIRED`. Treat existing
checklists as candidate inputs only; preserve immutable versions and history,
reconcile them to current approved sources and CAA procedures in a new Draft,
then require technical approval and a separate Department Manager publication
decision.

Review the source-bound Part 127 / Part 140 assessment under
`docs/regulatory-sources/derived/` against the exact public source versions and
controlled NCAA Operations surveillance/ramp-inspection procedure. Confirm the
current Part 140 authority/supersession state, operation/configuration-specific
Part 127 applicability, all six question decompositions, and expected Evidence.
If accepted, record technical validation separately from Department Manager
checklist publication; do not mutate published checklists or in-progress
Audits. Preserve the ignored full-text vault, exact source hashes,
`candidate-only` boundaries, advisory scope guardrails, root legacy demo,
current branch, and unrelated user changes.
