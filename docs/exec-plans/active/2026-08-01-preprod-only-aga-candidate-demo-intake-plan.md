# Preprod-Only AGA Candidate Demo Intake Implementation Plan

> **Status (2026-08-10):** `paused / retired by canonical successor`. The
> historical evidence remains, but the overlay runtime, schema provisioners,
> API/UI, commands, scripts, and tests were physically removed after the user
> selected canonical Task 9 `delete` and post-deletion qualification passed.
> Do not resume this duplicate product.

> **For agentic workers:** REQUIRED SUB-SKILL: Use
> Execute this plan only after the user separately authorizes implementation.
> The 2026-08-01 request authorizes this plan artifact only; do not
> start code, loader, database, browser, or runtime work from that request.

**Goal:** Load the exact accepted raw-byte-free AGA review package into a
disposable, Admin-only, read-only local-preprod demo projection while keeping
all form, question, source, risk, decision, publication, delivery, release,
and production boundaries visibly fail-closed.

**Architecture:** Add a package-bound AGA demo overlay beside, not inside, the
frozen synthetic preprod profiles and the real governed AGA intake lifecycle.
A separate one-shot PostgreSQL-only loader validates the exact ZIP and its
manifest before the first target write, records an append-only operational
intent outside the disposable namespace, and atomically seals immutable rows
in a dedicated `preprod_aga_demo` schema. A preprod-tagged API dependency
exposes only paginated reads to an authenticated CAA Admin; the ordinary API
has no backing demo reader and fails closed. The React Checklist Builder shows
the projection only when that capability is available. There is no mutation
route and no bridge from the demo schema to real intake, candidate, source,
decision, assignment, publication, delivery, Finding, or Audit records.

**Tech Stack:** Existing Go one-shot preprod loader and append-only control
store patterns, bounded Go ZIP/JSON validation, PostgreSQL transactional
projection tables, OpenAPI 3.1 and generated Go/TypeScript transport, the
existing OIDC/session boundary, React/Vite, Node contract tests, Go unit and
integration tests, Vitest, Playwright, Docker Compose local-preprod services,
and repository verification scripts.

## Global Constraints

- Planning is the only authorized deliverable in the current turn. Do not
  modify application code, tests, schemas, generated contracts, compose
  services, package contents, or runtime state until the user explicitly asks
  to execute this plan.
- Preserve the accepted raw-byte-free package byte-for-byte. Do not edit,
  regenerate, repackage, commit, publish, upload, or copy its question text
  into a new tracked fixture.
- Preserve the root HTML/CSS/Vanilla JavaScript demo as the immutable legacy
  oracle. The AGA package and projection must never enter the root demo.
- Do not place the accepted AGA material in any existing `smoke`, `acceptance`,
  `realistic`, or `stress` profile. Those catalogs remain frozen and
  synthetic-only. This plan adds a separately versioned overlay contract.
- Do not write the demo package into migration-28 real intake or governance
  tables. In particular, do not create a real import batch, extraction
  decision, existing-checklist candidate, source binding, source authority
  attestation, source mapping attestation, required-owner fact, functional
  assignment, review decision, publication decision, template version,
  Finding, Audit record, notification, outbox item, or provider delivery.
- The demo load itself is PostgreSQL-only. It must not receive Keycloak,
  Mailpit, MinIO, SMTP, lifecycle-client, object-store, worker, scheduler, or
  production credentials and must make zero calls to those systems.
- The AGA loader must be a separate fixed-entrypoint binary whose dependency
  graph contains only the bounded package reader, AGA control store, and
  PostgreSQL overlay store. Withholding provider configuration from the
  existing connected preprod loader is insufficient.
- The normal preprod API, tagged AGA reader, and one-shot AGA writer use
  different non-superuser PostgreSQL credentials. The normal API has no
  privilege on `preprod_aga_demo`; the tagged reader has `SELECT` only on
  sealed views; the one-shot writer is the only principal with overlay DDL or
  DML privilege. `PUBLIC` has no overlay privilege.
- The demo may reuse an already reconciled disposable local-preprod base run
  for authentication and shell data. That prerequisite is separately created
  and separately evidenced; the AGA overlay operation cannot provision or
  alter any identity, organization, provider account, email, object, queue, or
  delivery record.
- Load exactly the 52 supplied form identities. Preserve missing
  `FSS-AGA-FORM-049` and present `FSS-AGA-FORM-035A`; never synthesize a
  replacement identity.
- Load exactly the 1,310 supplied candidate question boundaries across the 31
  forms that contain boundaries. Candidate text and locators remain
  non-authoritative review material.
- The 21 forms with no detected boundary must contain zero demo questions and
  expose the literal `QUESTION_EXTRACTION_REVIEW_REQUIRED`. No heuristic,
  fallback, form title, source reference, blank row, or UI placeholder may be
  converted into a question.
- Every one of the 1,310 questions remains `NON_AUTHORITATIVE_CANDIDATE` and
  `SOURCE_MAPPING_REQUIRED`. The demo projection is immutable, so it cannot
  resolve that state itself.
- For the 1,261 questions with one or more question-level source proposals,
  proposal metadata is a review hint only. A future real governed record may
  leave `SOURCE_MAPPING_REQUIRED` only after exact source bytes, a SHA-256 of
  those exact bytes, effective date, clause/page locator, applicability, and a
  named source-owner attestation are all present and bound to the same source
  version and immutable question digest.
- The 49 questions with no question-level source proposal remain explicitly
  `UNMAPPED_NO_QUESTION_LEVEL_SOURCE_PROPOSAL`. Never inherit a form-level
  proposal, infer a reference from wording, or reuse a neighboring question's
  proposal.
- Preserve all proposed risk fields as provisional input. Never populate an
  approved risk band, approved safety-critical flag, Finding severity, or
  automatic approval. The 14 `PROPOSED_REVIEW_REQUIRED` questions must also
  expose `EXPERT_RISK_REVIEW_REQUIRED` as a blocking state.
- The operational one-shot loader authorization is not an Admin product
  decision, source-owner attestation, technical approval, publication
  decision, release approval, or production approval. UI and evidence must
  never describe it as one.
- Only successful exact-CAA-Admin projection responses and screens retain the
  literal labels `candidate-only`, `release pending`, and
  `production-ready: not established`. Denied and unavailable requests use one
  neutral, label-free not-found outcome.
- This plan does not advance or satisfy Task 9 of the active Governed AGA
  Checklist Intake plan. It also does not advance the blocked Local Preprod
  Release Candidate plan.
- Use English for code, tests, plans, evidence, UI copy, status values, and
  repository documentation.
- During later implementation, begin each task with focused test coverage and
  run the task regression gate before advancing its status.
- Preserve unrelated user changes. Do not overwrite the currently modified
  governed AGA plan/index row, untracked actor-bound decision work,
  `apps/web/.local/`, the accepted package, or its existing untracked tests.
- Work on the current branch. Do not create, switch, rename, or delete a
  branch or worktree. Do not stage, commit, push, deploy, or modify an external
  system unless that exact action is separately authorized.

---

## Status

- Design: Gate 0 contract/spec freeze `verified locally`; later implementation
  remains pending its ordered task gates.
- Plan artifact: active; implementation authorized on 2026-08-01.
- Gate 0 verification: `verified locally` on 2026-08-01 with
  `node --test tests/aga-candidate-preprod-demo-plan-contract.test.mjs`,
  `node tests/harness-docs-smoke.test.js`, and `git diff --check`.
- Task 1 implementation: `verified locally` on 2026-08-01 with synthetic
  path/ZIP/manifest/JSON/state/digest rejection cases and an explicit accepted
  ZIP validation of fixed outer/JSON/archive/register identities and the
  52/1,310/174 inventory. No package content was changed.
- Task 6 implementation: `verified locally` on 2026-08-02 with the
  capability-gated read-only panel, filtered sealed reads, no-store/telemetry
  suppression, focused web tests, both web build profiles, and tagged API
  checks. Connected browser/privacy evidence remains `not run`.
- Tasks 7–9 implementation: `not run`.
- Task 7 static boundary subset: `verified locally` on 2026-08-02 with
  read-only transport, provider/domain/import/storage/telemetry exclusion,
  separate credential wiring, normal-artifact guard, and nonzero two-spec
  Playwright discovery. Connected role, browser, PostgreSQL privilege,
  corruption, provider-delta, and residue qualification remains `not run`.
- Task 2 implementation: `verified locally` on 2026-08-02 for the separate
  append-only intent, authorization, result, and cleanup-tombstone control
  plane. Connected PostgreSQL and cleanup qualification remain `not run`.
- Task 3 implementation: `verified locally` on 2026-08-02 with the isolated
  overlay schema/role bootstrap contract, fixed-state/foreign-key/immutable
  guards, order-bound relationship digests, final in-transaction seal, and
  static mutation-scope checks. Connected PostgreSQL privilege, transaction,
  forbidden-table-delta, replay, and cleanup qualification remain `not run`.
- Task 4 implementation: `verified locally` on 2026-08-02 with a separate
  fixed-entrypoint AGA command/image, private-file-only configuration and
  authorization, isolated Compose service, and normal-artifact/import closure
  checks. No container, database, provider, or external system was started or
  changed.
- Task 5 implementation: `verified locally` on 2026-08-02 with generated
  GET-only transport, closed response schemas, authorization-first neutral
  no-store routes, sealed-view reader boundary, normal/tagged artifact split,
  and separate normal/reader credentials in the tagged Compose service.
  Connected sealed-reconciliation, role, and privacy qualification remain
  `not run`.
- Loader/runtime verification: `not run`.
- Browser/privacy verification: `not run`.
- Product status: `candidate-only`.
- Release: `release pending`.
- Production-ready: not established.

## Objective And User-Visible Outcome

In an exact disposable local-preprod namespace that already has a separately
reconciled base profile, an authenticated CAA Admin can open a read-only
**AGA Candidate Demo** panel in Checklist Builder and see:

1. the accepted package identity and its fixed status labels;
2. all 52 ordered form identities and their original hashes, titles, kinds,
   page counts, and candidate risk proposals;
3. a 21-form extraction-review queue whose rows show
   `QUESTION_EXTRACTION_REVIEW_REQUIRED` and exactly zero questions;
4. the 1,310 candidate question boundaries on the remaining 31 forms, with
   their exact package text digest, ordinal, protocol code when supplied,
   page, locator, and original candidate text;
5. a 1,310-question source-gap queue split into 1,261
   `PROPOSAL_PRESENT_REVIEW_REQUIRED` rows and 49
   `UNMAPPED_NO_QUESTION_LEVEL_SOURCE_PROPOSAL` rows;
6. all 2,329 question-level source proposal links and all 174 unique proposed
   source-reference identities as non-authoritative metadata; and
7. provisional risk distributions plus a dedicated 14-item
   `EXPERT_RISK_REVIEW_REQUIRED` blocker queue.

The panel has no approve, map, attest, assign, publish, deliver, import,
severity, safety-critical, Finding, or Audit action. An unavailable capability
or any identity, state, count, digest, privacy, target, or reconciliation
mismatch renders a specific fail-closed explanation and no candidate fallback.

All authenticated non-Admin roles, including other internal CAA roles and
Auditee, receive no list, count, text, search result, direct-ID result,
notification, export, or existence signal. The browser does not persist
candidate content in local storage, IndexedDB, Cache Storage, service-worker
caches, analytics, telemetry payloads, logs, traces, metrics, retained
screenshots, video, or browser-test artifacts. Browser requests are bound to
the current subject, membership, and session revision; logout, subject change,
or back/forward restoration aborts/invalidate reads and removes rendered
candidate content before it can be reused.

## Architecture Decision And Rejected Alternatives

### Selected: immutable preprod demo overlay

The selected design adds a package-specific overlay contract and dedicated
`preprod_aga_demo` schema. It reuses only the exact disposable target
fingerprint, the successful base-run result binding, the existing authenticated
Admin session boundary, and whole-namespace cleanup. It never invokes the
real governed AGA command path.

This is the only approach that simultaneously preserves the synthetic profile
contract, prevents a demo load from masquerading as real governance, permits
Admin-only review of all supplied candidate boundaries, and makes cleanup
whole-namespace and recoverable.

### Rejected: materialize real governed intake/candidate rows

Writing `checklist_import_*`, `existing_checklist_candidates`, governed source
bindings, or Draft rows would make the demo indistinguishable from Task 9
progress and would bypass the missing Admin, source-owner, and manager facts.
It is forbidden even if every row were labelled candidate-only.

### Rejected: add AGA data to a frozen preprod profile

The four existing profiles are deterministic synthetic catalogs. Adding
derived real-world candidate text would silently revise their versioned count,
privacy, and source boundaries. The overlay must therefore have its own
contract, intent, authorization, result, and reconciliation records.

### Rejected: ship a static browser JSON demo

A static asset would bypass server-side Admin authorization, make direct asset
URLs an Auditee/privacy leak, encourage client persistence, and provide no
transactional loader or fail-closed reconciliation. The browser must obtain
data only through the authenticated, no-store API.

## Exact Accepted Input Boundary

The loader accepts one regular, non-symlink ZIP file at an explicit absolute
path. It must validate the outer file before opening entries and then validate
the exact entry set, manifest, JSON bytes, and semantic contract before any
database connection capable of writing is used.

| Fact | Required value |
|---|---|
| ZIP file | `AGA_ALL_FORMS_SOURCE_RISK_DRAFT_2026-08-01.zip` |
| ZIP byte count | `336524` |
| ZIP SHA-256 | `sha256:30700a88aeb5b26514bf7eb76bef050deb08b96294db94117d185de5c9f163b2` |
| Package JSON | `AGA_ALL_FORMS_SOURCE_RISK_DRAFT.json` |
| Package JSON byte count | `3370312` |
| Package JSON SHA-256 | `sha256:5ebcce2d70ee22fef4165b490cb6e4b276ad776f40dbaf12e5cea85c9da91b15` |
| Package manifest SHA-256 | `sha256:1be7b37e78a320da51cf7069b033240f1ad032b045d3e3cd5746c4b2115c19dc` |
| Package version | `AGA_ALL_FORMS_SOURCE_RISK_DRAFT_V1` |
| Package status | `PENDING_ADMIN_AND_SOURCE_OWNER_REVIEW` |
| Original archive identity only | `sha256:dd819cfa6a670760e0cfceed94496e2e466dc53bac13e6fd792b1128314d6e32`, `12227415` bytes, 53 PDFs |
| Register identity only | `sha256:29ed8384693b615926fc42a0ca4654be2ea9a36b0946f217975571ca0ad9564f` |

The original 53 PDFs are identity/provenance facts only. They must not be
mounted, opened, copied, stored, fetched, or reconstructed by this demo load.
The accepted ZIP must contain exactly its current directory plus the seven
manifested review files and `MANIFEST.sha256`; any PDF, nested archive,
symlink, device, encrypted entry, duplicate path, unsafe path, unmanifested
file, extra file, or PDF magic bytes causes an atomic rejection.

The semantic validator also enforces:

- `candidateOnly: true` and no authoritative/accepted/approved/published state;
- 52 unique, ordered form codes and form hashes;
- the complete exact form-code set `001` through `034`, `035A`, `036` through
  `048`, and `050` through `053` under the `FSS-AGA-FORM-` prefix; numeric
  `035` and `049` are absent;
- missing 049 and present 035A exactly as supplied;
- 31 forms with 1,310 unique question proposal IDs and valid text digests;
- 21 forms with zero questions and the exact form-code set below;
- 174 unique source-coverage references, 274 form-source proposal links, and
  2,329 question-source proposal links;
- exactly 1,261 questions with proposals and 49 with neither a question-level
  source reference nor a proposal;
- proposed risk distribution 50 control assurance, 457 high operational, 14
  review required, and 789 safety critical; and
- form-level proposed-risk distribution 11 control assurance, 23 high
  operational, 4 review required, and 14 safety critical; 789 proposed
  `safetyCritical: true` and 521 proposed `false` values;
- exactly 28 Form 048 `EXACT_SOURCE_BACKED` package-extraction rows and 1,282
  `EXTRACTED_CANDIDATE` rows; and
- all 1,310 package-native
  `CANDIDATE_INTERPRETATION_REQUIRES_EXPERT_REVIEW` states, separately from
  the 14-item `PROPOSED_REVIEW_REQUIRED` hard-blocker subset; and
- no supplied decision, attestation, assignment, approval, publication,
  delivery, release, production, approved severity, or approved
  safety-critical fact.

The exact zero-boundary form-code set is:

`FSS-AGA-FORM-001`, `FSS-AGA-FORM-003`, `FSS-AGA-FORM-004`,
`FSS-AGA-FORM-005`, `FSS-AGA-FORM-007`, `FSS-AGA-FORM-008`,
`FSS-AGA-FORM-025`, `FSS-AGA-FORM-026`, `FSS-AGA-FORM-029`,
`FSS-AGA-FORM-032`, `FSS-AGA-FORM-033`, `FSS-AGA-FORM-035A`,
`FSS-AGA-FORM-036`, `FSS-AGA-FORM-038`, `FSS-AGA-FORM-039`,
`FSS-AGA-FORM-042`, `FSS-AGA-FORM-043`, `FSS-AGA-FORM-044`,
`FSS-AGA-FORM-045`, `FSS-AGA-FORM-046`, and `FSS-AGA-FORM-052`.

## Demo State And Authority Model

| Projection | Persisted/read state | Non-negotiable behavior |
|---|---|---|
| Package | `SEALED_PREPROD_DEMO_PROJECTION` | Operational seal only; not an Admin decision or import acceptance |
| All forms | `NON_AUTHORITATIVE_FORM_IDENTITY` | Preserve supplied identity and provenance; no real candidate row |
| 21 zero-boundary forms | `QUESTION_EXTRACTION_REVIEW_REQUIRED` | Question count remains zero; no invention or fallback |
| 31 forms with boundaries | `CANDIDATE_QUESTION_BOUNDARIES_PRESENT` | Does not mean extracted questions were accepted |
| All 1,310 questions | `NON_AUTHORITATIVE_CANDIDATE` | Original candidate text is review-only |
| All 1,310 questions | `SOURCE_MAPPING_REQUIRED` | Immutable in the demo schema; no transition route |
| All 1,310 questions | `CANDIDATE_INTERPRETATION_REQUIRES_EXPERT_REVIEW` | Package-native risk-review state remains explicit; it is not reduced to the 14-band subset |
| 28 Form 048 questions | `EXACT_SOURCE_BACKED` package provenance | Provenance only; it does not resolve source mapping or create authority |
| 1,282 remaining questions | `EXTRACTED_CANDIDATE` package provenance | Preserve the candidate-extraction distinction separately from authority state |
| 1,261 questions | `PROPOSAL_PRESENT_REVIEW_REQUIRED` | Proposal metadata satisfies none of the real authority gates by itself |
| 49 questions | `UNMAPPED_NO_QUESTION_LEVEL_SOURCE_PROPOSAL` | No form-level or neighboring fallback |
| All proposed risk bands | `PROVISIONAL_RISK_PROPOSAL` | Approved band, safety-critical decision, and Finding severity remain null |
| 14 review-required questions | `EXPERT_RISK_REVIEW_REQUIRED` | Remains a blocker; no automatic approval |
| Capability | `READ_ONLY_PREPROD_DEMO` | Admin list/detail reads only; no writes or exports |

Question-level proposed source locators, URLs, and hashes remain proposal
metadata. Even where the package proposes a local SHA-256 or page, the demo
does not possess an authority-bound exact source byte object, effective date,
applicability decision, or named source-owner attestation. Consequently no
proposal is considered resolved.

The six package-local source-resolution facts are jointly necessary but never
sufficient for a real governed resolution. A future governed workflow also
requires the complete authoritative source chain, currentness/applicability
facts, immutable Draft binding, scoped source-authority acceptance, candidate
mapping attestation, required functional assignments, and the separately
governed technical/publication decisions.

A later, separately authorized real governed workflow may create new real
records after all six source requirements are satisfied. It must never mutate
or promote the sealed demo row, and it must independently satisfy the active
Governed AGA Checklist Intake plan.

## Scope

### Included

- versioned documentation and a machine-readable plan contract;
- exact accepted-package ZIP, manifest, JSON, count, set, digest, and state
  validation without raw PDF bytes;
- a package-specific append-only intent, one-time operational authorization,
  result, and reconciliation contract outside the disposable target;
- an atomic, immutable PostgreSQL-only demo projection in the exact disposable
  local-preprod namespace;
- a preprod-only API reader/capability with paginated Admin-only endpoints;
- an Admin Checklist Builder demo panel for form, extraction, source-gap, and
  provisional-risk review;
- fail-closed package, target, state, privacy, cache, and no-side-effect tests;
- connected local-preprod loader, API, browser, cleanup, and residue evidence;
  and
- final documentation using literal evidence and readiness labels.

### Excluded

- importing or reparsing the original AGA PDFs;
- changing the accepted raw-byte-free package;
- completing or bypassing real AGA Task 9;
- real Admin identity/extraction decisions;
- source downloads, source-byte ingestion, source-currentness activation,
  legal interpretation, source authority, or source mapping acceptance;
- risk acceptance, safety-critical approval, or Finding severity derivation;
- functional-assignment provisioning or required-owner resolution;
- Draft creation, technical approval, publication, executable package
  eligibility, Audit creation, Finding creation, or CAA/provider delivery;
- real organization/provider records or production data;
- production configuration, deployment, release, or readiness claims; and
- selective overlay deletion. Disposal is whole-namespace only.

## Assumptions And Ownership

- The package path is supplied explicitly at execution time. Its current
  untracked workspace presence is not a durable repository dependency.
- The accepted package is the only candidate-content oracle. Committed unit
  fixtures use short synthetic strings and never copy accepted question text.
- An exact successful base `IntentManifest`/`ResultManifest` exists in the
  append-only preprod control store before the overlay operation. Missing,
  failed, drifted, or non-disposable base evidence blocks the overlay.
- An existing authenticated CAA Admin from that separately loaded base profile
  is used for UI verification. The overlay creates and changes zero users.
- Loader-operation authorization is supplied as an ephemeral 0600 file bound
  to the overlay intent and valid for at most 15 minutes. It is an operational
  authority only.
- Product Admin, source owner, Department Manager, release owner, and
  production owner decisions remain absent and are not implied by local demo
  access.

## Repository Orientation And Interfaces

### Existing authorities to preserve

- `docs/product-specs/data-and-rules/PREPROD_IDENTITY_AND_DATA_PROFILE.md`
  owns the frozen profile, target, loader, authorization, and cleanup
  contracts.
- `docs/product-specs/modules/CHECKLIST_BUILDER_AND_RUNNER.md` owns candidate
  question/source behavior.
- `docs/product-specs/modules/ADMIN_CONFIGURATION.md` owns Admin's intake
  limits.
- `docs/product-specs/data-and-rules/CONCEPTUAL_DATA_MODEL.md` and
  `STATUS_PERMISSION_SECURITY.md` own AGA authority and privacy semantics.
- `docs/exec-plans/active/2026-07-31-governed-aga-checklist-intake-and-official-source-authoring-plan.md`
  remains the real intake authority and stays blocked at Task 9.
- `apps/api/internal/preproddata/` owns existing synthetic loader primitives.
  Its frozen `IntentManifest` and profiles must not be broadened to candidate
  content.
- `apps/api/migrations/000028_governed_checklist_intake_and_authoring.up.sql`
  owns real governed intake tables. The demo store must not write them.
- `apps/api/internal/httpapi/` and `api/openapi/source/` own authenticated HTTP
  transport.
- `apps/web/src/features/admin/checklist-builder-page.tsx` owns the Admin
  Checklist Builder surface.
- `deploy/local/compose.yaml`, `scripts/load-preprod-data.sh`, and
  `scripts/test-normal-artifact-boundary.sh` own the current one-shot loader
  and normal-artifact isolation patterns.

### New package-validation interface

Create `apps/api/internal/preproddata/agacandidatedemo` with a bounded reader
whose public seam is equivalent to:

```go
type AcceptedPackage struct {
    Identity PackageIdentity
    Forms []FormCandidate
    SourceCoverage []SourceProposal
}

type PackageReader interface {
    ReadAndValidate(context.Context, string, ExpectedPackage) (AcceptedPackage, error)
}
```

`ReadAndValidate` receives an absolute ZIP path and an exact expected contract.
It uses `Lstat`, rejects links/non-regular files, limits outer and uncompressed
bytes, validates the exact ZIP digest and entry set, verifies every manifest
line, rejects duplicate JSON keys and unknown fields, validates text digests,
and returns no partially trusted object on any error.

### New overlay intent and operation

Do not add AGA content to `profiles.Profile` or
`preprod-intent-manifest/v1`. Add sibling types:

```go
type IntentManifest struct {
    SchemaVersion string
    RunID string
    Operation string
    BaseRunID string
    BaseIntentDigest string
    BaseResultDigest string
    PackageZipDigest string
    PackageJSONDigest string
    PackageManifestDigest string
    ExpectedCounts map[string]int64
    ExpectedDistributions map[string]map[string]int64
    ExpectedRelationshipDigests map[string]string
    CanonicalizationContract string
    CodeDigest string
    ContractDigest string
    Target TargetFingerprint
    IntentDigest string
    TargetFingerprintDigest string
}
```

The schema is `preprod-aga-candidate-demo-intent/v1`; the only load operation
is `LOAD_AGA_CANDIDATE_DEMO_OVERLAY`. Its separate authorization schema is
`preprod-aga-candidate-demo-operation-authorization/v1`. The target includes
the exact local-preprod PostgreSQL identity plus the successful base run,
intent, and result digests. No Keycloak, Mailpit, MinIO, queue, or delivery
operation is part of this intent.

`ExpectedRelationshipDigests` uses one versioned, domain-separated canonical
encoding per persisted table and ordered relationship. It binds every
persisted/displayed form, question, proposal, source, risk, provenance,
authority, locator, and state field—not only counts or question-text digests.
The target contains PostgreSQL system identifier, host, port, compose project,
the complete successful base `TargetFingerprintDigest`, base
profile/version/run/intent/result digests, and the overlay schema. Provider
namespace identity may be bound as inert predecessor identity metadata, but
the AGA loader receives neither provider network nor provider credentials.

### New immutable PostgreSQL projection

The loader, not a production migration, creates and owns only the disposable
`preprod_aga_demo` schema with these tables:

- `package_intents`
- `packages`
- `forms`
- `form_source_proposals`
- `source_reference_catalog`
- `questions`
- `question_source_proposals`
- `package_seals`

Every table uses schema-qualified SQL. Rows are insert-only; update/delete
guards always reject. Child inserts reject after the corresponding package
seal exists. The transaction recomputes and compares every expected
relationship digest before it inserts a digest-bearing seal as its final
statement. That seal is the sole database readability commit point; an
external result record may be reconstructed from it but cannot make an
unsealed or unreconciled package readable.

The result reconciles counts plus deterministic digests over all persisted
values, including ordered form identities/hashes, zero-boundary membership,
question protocol/page/locator/text/provenance/state, source proposal metadata
and authority gaps, source-gap membership, risk band/domain/rationale/flag and
review state, form risk, and the 14 expert-blocker IDs.

### New read-only API surface

Add these bounded `GET` operations to the OpenAPI contract:

- `/v1/admin/governed-checklist/aga-candidate-demo/capability`
- `/v1/admin/governed-checklist/aga-candidate-demo/summary`
- `/v1/admin/governed-checklist/aga-candidate-demo/forms`
- `/v1/admin/governed-checklist/aga-candidate-demo/forms/{formCode}`
- `/v1/admin/governed-checklist/aga-candidate-demo/questions`

List endpoints require cursor pagination and a limit from 1 through 100.
Question filters are exact enums for form, extraction queue, source-gap
category, proposed risk band, and expert blocker. There are no `POST`, `PUT`,
`PATCH`, `DELETE`, export, bulk action, or mutation operations.

The capability and reader dependency is injected only by a `preproddemo` API
profile that validates the exact local-preprod target and successful overlay
seal. The normal and canonical-test profiles inject no reader; capability is
unavailable and every data route returns an indistinguishable not-found
response. The normal API must not link the loader package.

Every outcome, including denial and error, sets `Cache-Control: private,
no-store`, `Pragma: no-cache`, and `Vary: Cookie` (plus `Authorization` only
when that credential mechanism is supported). The exact CAA Admin check runs
before route-specific path, cursor, filter, or resource validation and before
any reader call. Logs, traces, and metrics exclude candidate-derived counts,
IDs, digests, states, filters, URLs, paths, text, request bodies, and response
bodies; retained observability may contain only generic transport outcome and
duration.

## Machine-Readable Planning Contract

The future Gate 0 contract test consumes the following block. Prose may add
stricter behavior but must not weaken or silently change these values.

<!-- PREPROD_AGA_CANDIDATE_DEMO_CONTRACT:BEGIN -->
```json
{
  "schemaVersion": "preprod-aga-candidate-demo-contract/v1",
  "contractId": "aviasurveil360-preprod-aga-candidate-demo",
  "contractVersion": "1.1.0",
  "target": {
    "environment": "local-preprod",
    "databaseName": "aviasurveil360_local_preprod",
    "databaseOwner": "aviasurveil360_preprod_loader",
    "composeProject": "aviasurveil360-local-preprod",
    "overlaySchema": "preprod_aga_demo",
    "operation": "LOAD_AGA_CANDIDATE_DEMO_OVERLAY",
    "cleanup": "DROP_RECREATE_EXACT_WHOLE_NAMESPACE_ONLY",
    "requiredRuntimeBindings": [
      "POSTGRES_SYSTEM_IDENTIFIER",
      "POSTGRES_HOST",
      "POSTGRES_PORT",
      "BASE_TARGET_FINGERPRINT_DIGEST",
      "BASE_PROFILE_VERSION",
      "BASE_RUN_ID",
      "BASE_INTENT_DIGEST",
      "BASE_RESULT_DIGEST"
    ],
    "databaseAccess": {
      "normalApi": "NO_OVERLAY_PRIVILEGE",
      "taggedReader": "SELECT_SEALED_VIEWS_ONLY",
      "oneShotWriter": "OVERLAY_DDL_DML_ONLY",
      "public": "NO_OVERLAY_PRIVILEGE"
    }
  },
  "input": {
    "zipFile": "AGA_ALL_FORMS_SOURCE_RISK_DRAFT_2026-08-01.zip",
    "zipBytes": 336524,
    "zipSha256": "sha256:30700a88aeb5b26514bf7eb76bef050deb08b96294db94117d185de5c9f163b2",
    "packageVersion": "AGA_ALL_FORMS_SOURCE_RISK_DRAFT_V1",
    "packageStatus": "PENDING_ADMIN_AND_SOURCE_OWNER_REVIEW",
    "candidateOnly": true,
    "jsonBytes": 3370312,
    "jsonSha256": "sha256:5ebcce2d70ee22fef4165b490cb6e4b276ad776f40dbaf12e5cea85c9da91b15",
    "manifestSha256": "sha256:1be7b37e78a320da51cf7069b033240f1ad032b045d3e3cd5746c4b2115c19dc",
    "sourceArchiveSha256": "sha256:dd819cfa6a670760e0cfceed94496e2e466dc53bac13e6fd792b1128314d6e32",
    "sourceArchiveBytes": 12227415,
    "registerSha256": "sha256:29ed8384693b615926fc42a0ca4654be2ea9a36b0946f217975571ca0ad9564f"
  },
  "expected": {
    "formIdentities": 52,
    "formsWithCandidateBoundaries": 31,
    "questionExtractionReviewRequiredForms": 21,
    "candidateQuestions": 1310,
    "questionsWithSourceProposals": 1261,
    "questionsExplicitlyUnmapped": 49,
    "questionSourceProposalLinks": 2329,
    "formSourceProposalLinks": 274,
    "uniqueSourceReferences": 174,
    "proposedRiskBands": {
      "PROPOSED_CONTROL_ASSURANCE": 50,
      "PROPOSED_HIGH_OPERATIONAL": 457,
      "PROPOSED_REVIEW_REQUIRED": 14,
      "PROPOSED_SAFETY_CRITICAL": 789
    },
    "expertRiskReviewBlockers": 14,
    "formProposedRiskBands": {
      "PROPOSED_CONTROL_ASSURANCE": 11,
      "PROPOSED_HIGH_OPERATIONAL": 23,
      "PROPOSED_REVIEW_REQUIRED": 4,
      "PROPOSED_SAFETY_CRITICAL": 14
    },
    "proposedSafetyCritical": { "true": 789, "false": 521 },
    "packageExtractionStates": {
      "EXACT_SOURCE_BACKED": 28,
      "EXTRACTED_CANDIDATE": 1282
    },
    "packageRiskReviewStates": {
      "CANDIDATE_INTERPRETATION_REQUIRES_EXPERT_REVIEW": 1310
    }
  },
  "zeroBoundaryFormCodes": [
    "FSS-AGA-FORM-001",
    "FSS-AGA-FORM-003",
    "FSS-AGA-FORM-004",
    "FSS-AGA-FORM-005",
    "FSS-AGA-FORM-007",
    "FSS-AGA-FORM-008",
    "FSS-AGA-FORM-025",
    "FSS-AGA-FORM-026",
    "FSS-AGA-FORM-029",
    "FSS-AGA-FORM-032",
    "FSS-AGA-FORM-033",
    "FSS-AGA-FORM-035A",
    "FSS-AGA-FORM-036",
    "FSS-AGA-FORM-038",
    "FSS-AGA-FORM-039",
    "FSS-AGA-FORM-042",
    "FSS-AGA-FORM-043",
    "FSS-AGA-FORM-044",
    "FSS-AGA-FORM-045",
    "FSS-AGA-FORM-046",
    "FSS-AGA-FORM-052"
  ],
  "fixedStates": {
    "formIdentity": "NON_AUTHORITATIVE_FORM_IDENTITY",
    "zeroBoundaryForm": "QUESTION_EXTRACTION_REVIEW_REQUIRED",
    "question": "NON_AUTHORITATIVE_CANDIDATE",
    "sourceMapping": "SOURCE_MAPPING_REQUIRED",
    "proposalPresent": "PROPOSAL_PRESENT_REVIEW_REQUIRED",
    "noQuestionProposal": "UNMAPPED_NO_QUESTION_LEVEL_SOURCE_PROPOSAL",
    "risk": "PROVISIONAL_RISK_PROPOSAL",
    "packageRiskReview": "CANDIDATE_INTERPRETATION_REQUIRES_EXPERT_REVIEW",
    "exactSourceBackedProvenance": "EXACT_SOURCE_BACKED",
    "extractedCandidateProvenance": "EXTRACTED_CANDIDATE",
    "reviewRiskBlocker": "EXPERT_RISK_REVIEW_REQUIRED"
  },
  "sourceResolutionRequirements": [
    "EXACT_SOURCE_BYTES",
    "EXACT_SOURCE_BYTES_SHA256",
    "EFFECTIVE_DATE",
    "CLAUSE_OR_PAGE_LOCATOR",
    "APPLICABILITY",
    "NAMED_SOURCE_OWNER_ATTESTATION"
  ],
  "sourceResolutionBoundary": "PACKAGE_LOCAL_MINIMUM_NECESSARY_NOT_SUFFICIENT_FOR_GOVERNED_RESOLUTION",
  "reconciliation": {
    "canonicalizationContract": "AVIA_AGA_CANDIDATE_DEMO_CANONICAL_V1",
    "allPersistedAndExposedFieldsDigestBound": true,
    "sealInsertedOnlyAfterInTransactionReconciliation": true,
    "committedSealIsDatabaseReadabilityReceipt": true
  },
  "privacy": {
    "allowedTopLevelRoles": ["admin"],
    "requiredOrganizationScope": "exact-CAA",
    "nonAdminOutcome": "NOT_FOUND_WITHOUT_EXISTENCE_SIGNAL",
    "browserPersistence": "FORBIDDEN",
    "httpCache": "NO_STORE_ALL_OUTCOMES",
    "denial": "AUTHORIZATION_BEFORE_PARSE_OR_LOOKUP_NEUTRAL_LABEL_FREE_NOT_FOUND",
    "candidateObservability": "FORBIDDEN",
    "retainedRealPackageBrowserArtifacts": "FORBIDDEN"
  },
  "forbiddenEffects": [
    "ADMIN_PRODUCT_DECISION",
    "EXTRACTION_DECISION",
    "REAL_IMPORT_BATCH",
    "REAL_IMPORT_FILE_OR_RECEIPT",
    "EXTRACTION_REVIEW_PACKET",
    "REAL_CHECKLIST_CANDIDATE",
    "GOVERNED_DRAFT",
    "SOURCE_CURRENTNESS_ACTIVATION",
    "SOURCE_AUTHORITY_ATTESTATION",
    "SOURCE_MAPPING_ATTESTATION",
    "FUNCTIONAL_ASSIGNMENT",
    "TECHNICAL_APPROVAL",
    "PUBLICATION",
    "TEMPLATE_VERSION",
    "AUDIT_PACKAGE_ELIGIBILITY",
    "AUDIT_RECORD_OR_PACKAGE",
    "FINDING_RECORD",
    "NOTIFICATION",
    "OUTBOX_ITEM",
    "CAA_OR_PROVIDER_DELIVERY",
    "PRODUCTION_RECORD",
    "FINDING_SEVERITY",
    "AUTOMATIC_SAFETY_CRITICAL_APPROVAL",
    "PRODUCTION_READINESS_CLAIM"
  ],
  "labels": [
    "candidate-only",
    "release pending",
    "production-ready: not established"
  ]
}
```
<!-- PREPROD_AGA_CANDIDATE_DEMO_CONTRACT:END -->

## Implementation Plan

### Gate 0 — Freeze the overlay contract before functional code

**Files**

- Modify:
  `docs/product-specs/data-and-rules/PREPROD_IDENTITY_AND_DATA_PROFILE.md`
- Modify: `docs/product-specs/modules/CHECKLIST_BUILDER_AND_RUNNER.md`
- Modify: `docs/product-specs/modules/ADMIN_CONFIGURATION.md`
- Modify: `docs/product-specs/data-and-rules/CONCEPTUAL_DATA_MODEL.md`
- Modify: `docs/product-specs/data-and-rules/STATUS_PERMISSION_SECURITY.md`
- Modify: `docs/product-specs/screen-specs/SCREEN_INVENTORY_AND_FORMS.md`
- Create: `tests/aga-candidate-preprod-demo-plan-contract.test.mjs`
- Preserve without editing:
  `deliverables/AGA_ALL_FORMS_SOURCE_RISK_DRAFT_2026-08-01.zip`
- Preserve without editing:
  `tests/aga-all-forms-source-risk-draft.test.mjs`

**Steps**

- [ ] Add a failing plan-contract test that extracts the fenced JSON above and
  asserts every digest, count, state, zero-form identity, required source fact,
  target binding, database-access boundary, reconciliation rule, privacy rule,
  forbidden effect, and label.
- [ ] Record the red result caused by the product specs not yet naming the
  overlay distinction and fixed states.
- [ ] Add the smallest product-spec changes that define a separately versioned
  `aga-candidate-demo@1.1.0` overlay and explicitly preserve the synthetic
  profile and real governed-intake boundaries.
- [ ] State in every relevant spec that the overlay is read-only, immutable,
  preprod-only, Admin-only, and incapable of satisfying Task 9.
- [ ] Add the complete 52-code set, 21-form extraction rule, 28/1,282 package
  extraction-provenance split, 1,261/49 source split, six necessary-but-not-
  sufficient source facts, all-1,310 risk-review state, 14 hard blockers,
  provisional-risk rule, and fixed successful-Admin status labels.
- [ ] Keep the package-specific test opt-in through an explicit package path;
  never make ordinary CI depend on an untracked deliverable. Treat the existing
  directory-based package test as current-workspace corroboration only; the
  bounded ZIP-reader test is the portable acceptance input.

**Commands**

```bash
node --test tests/aga-candidate-preprod-demo-plan-contract.test.mjs
node tests/harness-docs-smoke.test.js
git diff --check
```

**Expected observation**

The new contract test passes, documentation links remain valid, and no
functional code or runtime state has changed. The accepted ZIP remains an
explicit Task 1 input; Gate 0 does not depend on the current untracked
extracted directory.

**Gate acceptance**

- The overlay is a named versioned exception beside frozen profiles, not a
  profile mutation.
- Real governed intake and production/release status remain unchanged.
- Implementation cannot begin until this gate is accepted.

### Task 1 — Build the bounded accepted-package validator

**Files**

- Create: `apps/api/internal/preproddata/agacandidatedemo/contract.go`
- Create: `apps/api/internal/preproddata/agacandidatedemo/package_reader.go`
- Create: `apps/api/internal/preproddata/agacandidatedemo/json_keys.go`
- Create: `apps/api/internal/preproddata/agacandidatedemo/package_reader_test.go`
- Create:
  `apps/api/internal/preproddata/agacandidatedemo/testdata/synthetic-valid-package.zip`

The committed ZIP fixture must contain only short synthetic strings and must
not copy accepted package text, names, URLs, or source references.

**Steps**

- [ ] Write failing tests for outer symlink/non-regular file, file replacement
  between validation stages, wrong byte count,
  wrong ZIP hash, unsafe/duplicate/extra entries, PDF/nested archive content,
  expansion/entry limits, manifest mismatch, JSON duplicate/unknown fields,
  trailing JSON, invalid digests, invalid candidate states, and count/set
  drift.
- [ ] Write failing tests for a zero-boundary form containing a question, a
  49-row unmapped category containing a proposal/reference, a proposed source
  treated as authoritative, an approved risk/severity field, and any supplied
  decision/publication/assignment fact.
- [ ] Open the package once beneath the trusted parent with no-follow and
  close-on-exec semantics, `fstat` that held descriptor as a regular file, and
  hash and parse that same descriptor without reopening the path. Cap the outer
  file at 1 MiB, each entry at 8 MiB, total expanded bytes at 16 MiB, and
  entries at exactly the accepted set plus directory. Reject encrypted,
  data-descriptor ambiguity, link/device modes, unsafe names, and non-regular
  content.
- [ ] Validate all seven manifest entries and the manifest file digest before
  decoding the package JSON.
- [ ] Decode with duplicate-key detection, `DisallowUnknownFields`, an exact
  EOF check, bounded depth/collection sizes, and no permissive number coercion.
- [ ] Hash each decoded question's UTF-8 `originalText` and require its exact
  `textDigest`; require unique proposal IDs and stable form/ordinal ordering.
- [ ] Require the complete form-code/hash set, form risk, proposed safety flag,
  extraction provenance, all-row risk-review state, source authority state,
  decision state, and every proposal/source field exactly as packaged.
- [ ] Return typed candidate records only after the entire archive and
  semantic contract succeeds. Do not expose partially parsed records.
- [ ] Add an opt-in integration test that receives
  `AVIA_AGA_DEMO_PACKAGE_FILE`, validates the exact accepted ZIP, and prints
  counts/digests only.

**Commands**

```bash
go -C apps/api test ./internal/preproddata/agacandidatedemo -run 'Package|Contract'
AVIA_AGA_DEMO_PACKAGE_FILE="$PWD/deliverables/AGA_ALL_FORMS_SOURCE_RISK_DRAFT_2026-08-01.zip" \
  go -C apps/api test ./internal/preproddata/agacandidatedemo -run AcceptedPackage
```

**Expected observation**

Synthetic positive/negative cases pass; the exact accepted package reports
52/21/31/1,310/1,261/49/2,329/174/14 and the fixed digests without logging
question text or source URLs.

### Task 2 — Add an independent overlay intent, authorization, and result

**Files**

- Create: `apps/api/internal/preproddata/agacandidatedemo/manifest.go`
- Create: `apps/api/internal/preproddata/agacandidatedemo/authorization.go`
- Create: `apps/api/internal/preproddata/agacandidatedemo/control_store.go`
- Create: `apps/api/internal/preproddata/agacandidatedemo/result.go`
- Create: `apps/api/internal/preproddata/agacandidatedemo/manifest_test.go`
- Create: `apps/api/internal/preproddata/agacandidatedemo/authorization_test.go`
- Create: `apps/api/internal/preproddata/agacandidatedemo/control_store_test.go`
- Preserve semantics in: `apps/api/internal/preproddata/manifest.go`
- Preserve semantics in: `apps/api/internal/preproddata/profiles/profiles.go`

**Steps**

- [ ] Start with failing tests proving the existing frozen profile manifest
  rejects AGA input and remains byte/behavior compatible.
- [ ] Add failing tests for missing/failed/mismatched base results, wrong
  PostgreSQL system identity, non-local-preprod targets, target/run drift,
  package/contract/code digest drift, expired/future/broad authorization,
  token replay, intent collision, result collision, cleanup tombstones,
  kill-points during every control-record publish, and control-store links or
  permissive modes.
- [ ] Implement the sibling manifest schema and bind exact accepted-package,
  complete base target/profile/run/result, code, product-contract, count,
  distribution, canonicalization, and per-table/per-relationship digests.
- [ ] Implement a separate 15-minute single-use authorization whose only
  operation is `LOAD_AGA_CANDIDATE_DEMO_OVERLAY`. Retain only the token's
  SHA-256 and public claims.
- [ ] Store intent, authorization consumption, result, run binding, and cleanup
  tombstone records under private `aga-demo/` directories by writing canonical
  0600 bytes to a same-directory temporary file, validating its digest/length,
  `fsync`ing it, atomically publishing without replacement, then `fsync`ing
  the directory. Never truncate or overwrite a record.
- [ ] Define deterministic relationship digests for every persisted/displayed
  field and ordered relationship, not only forms, counts, and queue splits.
- [ ] Define exact replay as a read-only live target/seal verification. A
  state-changing retry requires fresh single-use authorization; a cleanup
  tombstone makes the old run non-replayable and requires a fresh run/intent.
  The same run ID with different content remains a permanent conflict.

**Commands**

```bash
go -C apps/api test ./internal/preproddata/agacandidatedemo -run 'Intent|Authorization|ControlStore|Result'
go -C apps/api test ./internal/preproddata ./internal/preproddata/profiles
node --test tests/preprod-data-boundary.test.mjs
```

**Expected observation**

The new overlay control plane is append-only and exact-package-bound; existing
synthetic intent/profile tests remain unchanged and green.

### Task 3 — Materialize an atomic immutable demo projection

**Files**

- Create: `apps/api/internal/preproddata/agacandidatedemo/store.go`
- Create: `apps/api/internal/preproddata/agacandidatedemo/postgres_store.go`
- Create: `apps/api/internal/preproddata/agacandidatedemo/postgres_roles.go`
- Create: `apps/api/internal/preproddata/agacandidatedemo/loader.go`
- Create: `deploy/local/preprod/aga-demo-role-provision.sql`
- Modify: `scripts/init-local-preprod-namespace.sh`
- Create: `apps/api/internal/preproddata/agacandidatedemo/postgres_store_test.go`
- Create: `apps/api/internal/preproddata/agacandidatedemo/loader_test.go`

**Steps**

- [ ] Write failing PostgreSQL tests for a non-disposable database, wrong
  owner/system ID/compose project, absent base result, dirty or divergent demo
  schema, partial package, count mismatch, digest mismatch, second different
  package, update/delete attempts, post-seal inserts, direct state changes, and
  normal-API/reader/writer privilege separation.
- [ ] Write failing tests proving no SQL mutation targets a table outside
  `preprod_aga_demo` and no real domain command service is called.
- [ ] Create the dedicated schema and insert-only tables with primary/foreign
  keys, fixed-state checks, exact non-null constraints, and immutable guards.
- [ ] Provision `preprod_aga_demo_owner`, `preprod_aga_demo_writer`,
  `preprod_aga_demo_reader`, and the separate normal-preprod-API login role
  during exact disposable-namespace initialization through
  `aga-demo-role-provision.sql`; this bootstrap is not available to either API
  or the AGA loader. Give the writer/reader only their own 0600 secrets. Revoke
  `PUBLIC` and normal-API privilege, grant the reader only `CONNECT`, schema
  `USAGE`, and `SELECT` on sealed views, and set its sessions
  transaction-read-only. The normal API must have no overlay `CONNECT`, schema,
  table, sequence, function, or DDL privilege.
- [ ] Persist complete form identities, candidate question boundaries,
  original candidate text and digest, form/question proposal links, source
  reference catalog, source-gap category, provisional risk fields, all-row
  package risk-review state, package extraction provenance, and expert
  blockers. Keep all approved/authoritative fields absent by schema design.
- [ ] Persist the six source-resolution requirements as fixed contract values,
  not inferred completion flags. A package proposal can never satisfy them.
- [ ] Insert intent, package, children, then recompute/compare every canonical
  digest and insert the final reconciliation-bearing seal in one transaction.
  Any failure rolls back every demo-schema row.
- [ ] Make a committed valid seal sufficient for database readability and
  recover the external result only from that seal. Readers reject every absent,
  malformed, or digest-mismatched seal.
- [ ] Make read-only exact replay re-preflight the live target and seal. Make
  any cleaned, missing, divergent, or different-digest target fail without
  writes; only a fresh intent can load after cleanup.
- [ ] Add before/after row-count and digest assertions for the real governed
  intake, authority, assignment, publication, template, Audit, Finding,
  notification, outbox, and delivery table families.

**Commands**

```bash
go -C apps/api test ./internal/preproddata/agacandidatedemo -run 'Postgres|Loader|Atomic|Immutable|Forbidden'
go -C apps/api test ./internal/checklistintake ./internal/checklistgovernance ./internal/regulatory
```

**Expected observation**

An accepted transaction seals exactly one demo package; every negative case
leaves zero new demo rows, every forbidden real table has an exact zero delta,
and replay is deterministic.

### Task 4 — Expose a separate one-shot overlay command without provider capability

**Files**

- Create: `apps/api/cmd/preprod-aga-candidate-demo-loader/main.go`
- Create: `apps/api/cmd/preprod-aga-candidate-demo-loader/main_test.go`
- Create: `scripts/load-aga-candidate-demo.sh`
- Create: `scripts/test-aga-candidate-demo-loader.sh`
- Modify: `apps/api/Dockerfile`
- Modify: `deploy/local/compose.yaml`
- Modify: `tests/preprod-data-boundary.test.mjs`
- Modify: `scripts/test-normal-artifact-boundary.sh`

**Steps**

- [ ] Add failing CLI tests for missing absolute package/config/control paths,
  permissive modes, symlinks, wrong command, wrong target, missing base result,
  authorization on argv/environment, and secret/text leakage.
- [ ] Add `prepare-aga-demo`, `verify-aga-demo-authorization`,
  `run-aga-demo`, and `verify-aga-demo` subcommands. `prepare` writes the
  immutable intent before target writes; `verify` is read-only.
- [ ] Keep package path and operational authorization in separate private
  files/config. Do not place token values on argv or in exported variables.
- [ ] Add a `preprod-aga-candidate-demo-loader` Compose service using its own
  fixed-entrypoint image and a distinct profile. Give it only PostgreSQL,
  read-only package/config/authorization mounts, a writable external control
  store, read-only root, bounded tmpfs, no new privileges, dropped
  capabilities, bounded PIDs/memory/CPU, and the preprod database network.
- [ ] Do not provide or depend on Keycloak, Mailpit, MinIO, SMTP, lifecycle
  client, object storage, queue, API, worker, or scheduler configuration.
- [ ] Require `umask 077`, 0600 files, absolute paths, and exact mount targets
  in the wrapper. Reject package paths outside an explicit regular file.
- [ ] Extend static boundary tests to prove the normal API/worker/scheduler
  binaries still do not link the AGA loader and that the AGA loader's import
  closure contains no Keycloak, Mailpit, MinIO, SMTP, lifecycle, object-store,
  queue, worker, or scheduler client.
- [ ] Prove the demo loader service has no provider services in `depends_on`,
  no provider secrets, and no provider hostnames or credentials.

**Commands**

```bash
go -C apps/api test ./cmd/preprod-aga-candidate-demo-loader -run AGACandidateDemo
node --test tests/preprod-data-boundary.test.mjs
bash scripts/test-normal-artifact-boundary.sh
bash scripts/test-aga-candidate-demo-loader.sh
```

**Expected observation**

The isolated operation can validate and load only the demo PostgreSQL schema;
provider/delivery capability is absent rather than merely unused.

### Task 5 — Add a fail-closed Admin-only read API

**Files**

- Modify: `api/openapi/source/paths/platform.json`
- Modify: `api/openapi/source/schemas/platform.json`
- Create: `api/openapi/tests/aga-candidate-demo-contract.test.mjs`
- Regenerate: `api/openapi/aviasurveil360.yaml`
- Regenerate: `apps/api/internal/httpapi/generated/api.gen.go`
- Regenerate: `apps/web/src/generated/transport/api-types.ts`
- Create: `apps/api/internal/agacandidatedemo/types.go`
- Create: `apps/api/internal/agacandidatedemo/service.go`
- Create: `apps/api/internal/agacandidatedemo/postgres_reader_preproddemo.go`
- Create: `apps/api/internal/agacandidatedemo/unavailable_default.go`
- Create: `apps/api/internal/httpapi/aga_candidate_demo_api.go`
- Create: `apps/api/internal/httpapi/aga_candidate_demo_api_test.go`
- Modify: `apps/api/internal/httpapi/canonical_api.go`
- Modify: `apps/api/cmd/api/profile_normal.go`
- Modify: `apps/api/cmd/api/profile_normal_test.go`
- Create: `apps/api/cmd/api/profile_preproddemo.go`
- Create: `apps/api/cmd/api/profile_preproddemo_test.go`
- Modify: `apps/api/cmd/api/main.go`
- Modify: `apps/api/Dockerfile`
- Modify: `deploy/local/compose.yaml`

**Steps**

- [ ] Start with failing OpenAPI tests for the five exact `GET` operations,
  fixed states/labels, pagination bounds, source requirements, 49-row unmapped
  discriminant, expert blocker, null approved fields, and absence of mutations.
- [ ] Generate transport only through repository scripts and prove generated
  files are current.
- [ ] Add failing HTTP tests for unauthenticated, Auditee, each non-Admin CAA
  role, wrong organization, direct form-code guessing, invalid cursor/filter,
  unsealed data, drifted reconciliation, unavailable capability, and reader
  error. Every denied request must authorize before parse or reader access and
  return the same label-free not-found body, headers, and content length with a
  zero-call reader spy.
- [ ] Add the neutral read service and bounded cursor queries. It returns only
  sealed/reconciled data and never imports the loader package or exposes a
  write interface.
- [ ] Register the read routes in the canonical transport with a nil/unavailable
  dependency by default. All default-profile data routes fail closed.
- [ ] Change the normal profile build constraint to
  `!canonicaltest && !preproddemo`; add a `preproddemo` profile that accepts
  only development plus the complete exact disposable target/base binding and
  injects a second, read-only sealed-view PostgreSQL reader pool. The default
  API has no overlay DSN or privilege.
- [ ] Add a distinct `preprod-aga-demo-api` image target and Compose service.
  The normal API image must remain free of the tagged reader wiring.
- [ ] Enforce exact CAA Admin authority before path/cursor/filter parsing,
  limit 1–100, stable cursor ordering, no-store headers on every outcome,
  neutral denial bodies, `Vary: Cookie`, and candidate-free observability.
- [ ] Return all fixed candidate/source/risk states from persisted server data;
  never derive approval from a proposal or allow client state override.

**Commands**

```bash
bash scripts/check-contracts.sh
node --test api/openapi/tests/aga-candidate-demo-contract.test.mjs
go -C apps/api test ./internal/agacandidatedemo ./internal/httpapi ./cmd/api
go -C apps/api test -tags=preproddemo ./internal/agacandidatedemo ./internal/httpapi ./cmd/api
go -C apps/api build -tags=preproddemo ./cmd/api
bash scripts/test-normal-artifact-boundary.sh
```

**Expected observation**

Only the exact tagged local-preprod API can obtain a reader. CAA Admin sees
sealed, paginated candidate views; all other profiles and roles fail closed,
and no mutation operation exists.

### Task 6 — Build the read-only Checklist Builder demo panel

**Files**

- Create: `apps/web/src/backend/aga-candidate-demo.ts`
- Modify: `apps/web/src/backend/backend.ts`
- Modify: `apps/web/src/backend/http-backend.ts`
- Modify: `apps/web/src/backend/backend-contracts.ts`
- Modify: `apps/web/src/mock/mock-engine.ts`
- Modify: `apps/web/package.json`
- Modify: `apps/web/playwright.config.ts`
- Create: `apps/web/src/features/admin/aga-candidate-demo-panel.tsx`
- Create: `apps/web/src/features/admin/aga-candidate-demo-panel.test.tsx`
- Modify: `apps/web/src/features/admin/checklist-builder-page.tsx`
- Modify: `apps/web/src/features/admin/checklist-builder-page.test.tsx`
- Create: `apps/web/src/features/admin/aga-candidate-demo.css`

**Steps**

- [ ] Begin with failing Backend-facade and UI tests for unavailable capability,
  loading/error/empty
  states, exact labels/counts, the 21-form extraction queue, 1,261/49 source
  split, six source requirements, provisional risk distribution, 14 expert
  blockers, pagination, and direct route access by a non-Admin shell.
- [ ] Add a capability-gated Checklist Builder entry visible only to an
  authenticated CAA Admin when the preprod demo API says it is available.
- [ ] Add a named read-only AGA capability to the shared `Backend` interface,
  HTTP adapter, mock/default-unavailable implementation, and capability
  registry. `aga-candidate-demo.ts` may contain types and mappers only; it
  must not issue a direct `fetch`.
- [ ] Add `test:e2e:aga-preprod` and a `preprod-aga-demo` Playwright project
  that targets externally started tagged API/web/OIDC services, discovers only
  the two AGA specs, and has trace/screenshot/video disabled. Assert a nonzero
  `playwright test --list` result before running it.
- [ ] Render package summary and four table-first views: all forms,
  extraction-review, source-gap, and risk-review blockers. Use server-side
  filters and pagination; never fetch all 1,310 full question rows at once.
- [ ] Show original candidate text only in the Admin detail view. Retained
  screenshots never use the accepted package; synthetic fixtures alone may
  supply visual evidence.
- [ ] Display `SOURCE_MAPPING_REQUIRED` and the six future evidence
  requirements for all proposal-present rows. Display explicit unmapped copy
  for the 49 rows and no proposed-source card.
- [ ] Present every risk band with a `Provisional` qualifier. Keep approved
  risk, safety-critical decision, and Finding severity absent. Render the 14
  expert blockers as blockers, not warnings that can be dismissed.
- [ ] Provide no action-shaped control for import, accept, map, attest, assign,
  approve, publish, deliver, create Finding, or create Audit. If explanatory
  controls are shown, they are disabled with a specific reason and cannot emit
  a command or success toast.
- [ ] Route requests through the shared adapter with `cache: "no-store"` and
  this capability's telemetry suppression. Do not add candidate data to browser
  storage, offline outbox, mock fallback, service-worker caches, analytics,
  telemetry, error reports, retained traces, screenshots, or video. Abort and
  invalidate requests on logout, principal/session change, and BFCache restore.
- [ ] Preserve keyboard navigation, visible focus, semantic headings/tables,
  accessible status text, responsive overflow, and exact mobile reading order.

**Commands**

```bash
npm --prefix apps/web test -- --run aga-candidate-demo-panel checklist-builder-page
npm --prefix apps/web run typecheck
npm --prefix apps/web run build:demo
npm --prefix apps/web run build:http
node apps/web/scripts/assert-http-artifact.mjs apps/web/dist
```

**Expected observation**

The Admin can review the exact queues without any working or fake mutation;
all status/authority boundaries are visible and accessible on desktop and
mobile.

### Task 7 — Prove privacy, no-side-effect, and fail-closed boundaries

**Files**

- Create: `tests/aga-candidate-preprod-demo-boundary.test.mjs`
- Create: `apps/web/tests/e2e/aga-candidate-demo-privacy.http.spec.ts`
- Create: `apps/web/tests/e2e/aga-candidate-demo-admin.http.spec.ts`
- Modify: `tests/preprod-data-boundary.test.mjs`
- Modify: `scripts/test-normal-artifact-boundary.sh`

**Steps**

- [ ] Add static tests that forbid demo writes to every real intake,
  governance, source-attestation, assignment, decision, publication, template,
  Audit, Finding, notification, outbox, identity, and delivery family.
- [ ] Add a generated-contract test proving there is no mutation or export
  operation, only successful exact-CAA-Admin payloads carry the three labels,
  and every denial is neutral and label-free.
- [ ] Exercise anonymous, Auditee, Inspector, Lead Inspector, Department
  Manager, Finance, General Manager, Executive Director, wrong-organization,
  stale-session, and direct-ID access. Check identical body/headers/content
  length, zero reader calls, repeated valid-versus-invalid direct-ID timing,
  navigation, search, notification, log, and count leakage.
- [ ] For CAA Admin, prove list/detail pagination, exact queue membership, and
  no-store response headers.
- [ ] Inspect localStorage, sessionStorage, IndexedDB, Cache Storage,
  service-worker cache keys, offline outbox, network retry state, telemetry,
  trace, screenshot, video, React/query memory, and BFCache after Admin
  navigation, delayed response, logout, principal switch, and back/forward.
  Candidate content must be absent.
- [ ] Attempt direct SQL reads, updates, deletes, inserts, DDL, trigger change,
  and role escalation through normal-API, tagged-reader, and writer credentials
  after seal; assert each role has only its specified privilege. Attempt
  source/risk state transitions through every HTTP method and assert no route
  or command exists.
- [ ] Run semantic corruption cases with their own recomputed test ZIP hashes:
  51/53 forms, a synthetic question on a zero-boundary form, 1,309/1,311
  questions, changed text digest, 1,260/1,262 proposed rows, 48/50 unmapped
  rows, 13/15 expert blockers, authoritative source state, approved risk,
  supplied decision, and production label. Each must fail before writes.
- [ ] Capture before/after provider user counts, Mailpit message count, MinIO
  object/version count, queue/job counts, and forbidden PostgreSQL table
  digests around the overlay operation. Every delta must be zero.

**Commands**

```bash
node --test tests/aga-candidate-preprod-demo-boundary.test.mjs tests/preprod-data-boundary.test.mjs
go -C apps/api test ./internal/preproddata/agacandidatedemo ./internal/agacandidatedemo ./internal/httpapi
npm --prefix apps/web test -- --run aga-candidate-demo
npm --prefix apps/web run test:e2e:aga-preprod -- aga-candidate-demo-privacy.http.spec.ts aga-candidate-demo-admin.http.spec.ts
bash scripts/test-normal-artifact-boundary.sh
```

**Expected observation**

CAA Admin reads only the sealed projection; every other role and every
corruption/mutation path fails closed without data, existence, provider,
delivery, real-domain, cache, or log leakage.

### Task 8 — Run the exact connected disposable preprod qualification

**Files**

- Create: `scripts/test-aga-candidate-preprod-demo-connected.sh`
- Create: `docs/demo-evidence/PREPROD_AGA_CANDIDATE_DEMO_2026-08-01.md`
- Update this plan's Progress, Decisions, Discoveries, and Outcome sections
  with literal results only after the run.

**Steps**

- [ ] Require explicit base intent/result paths, exact target fingerprint
  digest, load authorization path, and separate cleanup authorization path.
  Missing or drifted predecessor evidence is `blocked`; this qualification must
  not create or reconcile the base profile. Treat its identity/provider
  activity as predecessor evidence, not AGA load activity.
- [ ] Stop task-owned API, worker, scheduler, and browser processes before the
  overlay write. Snapshot forbidden PostgreSQL table digests, provider users,
  Mailpit messages, MinIO objects/versions, and queue/job counts.
- [ ] Prepare the AGA overlay intent against the exact accepted ZIP and base
  result. Supply a separate single-use operational authorization and verify it
  without writing.
- [ ] Run the PostgreSQL-only overlay once. Assert a successful sealed result,
  then rerun read-only live verification and an exact replay. Assert that
  replay after cleanup is rejected by the cleanup tombstone. Record immutable
  intent/result/reconciliation digests without question text.
- [ ] Query the sealed views for exact form set/hashes, counts, 21-form set,
  28/1,282 extraction provenance, 1,261/49 split, 2,329 links, question and
  form risk distributions, all-1,310 risk-review states, 14 blocker IDs, null
  approved fields, role grants, and immutable reconciliation-bearing seal.
- [ ] Compare every pre/post forbidden-system snapshot. Any non-zero delta
  fails the run and blocks completion.
- [ ] Start the tagged preprod demo API and ordinary React web artifact with
  the existing OIDC boundary. Verify CAA Admin and the complete denied-role
  matrix.
- [ ] Run isolated-browser QA at `1440x900`, `1024x768`, and `390x844` for
  summary, extraction queue, source-gap categories, and risk blockers. Check
  keyboard use, console, failed requests, horizontal overflow, and no browser
  persistence. Disable retained screenshot, trace, and video capture for the
  accepted package; do not retain count/status images either.
- [ ] Stop and clean all task-owned browser, Chrome helper, Playwright, Vite,
  API, and loader processes.
- [ ] Use the existing separately authorized whole-namespace
  `DROP_RECREATE_TARGET` cleanup. Never selectively delete the overlay.
- [ ] Record cleanup attestation outside the target and verify zero database,
  Keycloak, Mailpit, MinIO, queue, process, and browser-cache residue.

**Qualification entrypoint**

```bash
AVIA_AGA_DEMO_PACKAGE_FILE="$PWD/deliverables/AGA_ALL_FORMS_SOURCE_RISK_DRAFT_2026-08-01.zip" \
AVIA_AGA_DEMO_BASE_RESULT_FILE=/absolute/private/base-result.json \
AVIA_AGA_DEMO_LOAD_AUTHORIZATION_FILE=/absolute/private/load-authorization.json \
AVIA_AGA_DEMO_CLEANUP_AUTHORIZATION_FILE=/absolute/private/cleanup-authorization.json \
  bash scripts/test-aga-candidate-preprod-demo-connected.sh
```

**Expected observation**

The exact package is `verified locally` only as an immutable Admin-only
preprod demo projection. The AGA overlay itself has zero provider, delivery,
real-domain, publication, or production effects. Whole-namespace cleanup
leaves zero target residue.

### Task 9 — Run the aggregate gate and close evidence without overclaiming

**Files**

- Modify:
  `docs/exec-plans/active/2026-08-01-preprod-only-aga-candidate-demo-intake-plan.md`
- Modify: `docs/exec-plans/index.md`
- Modify: `docs/demo-evidence/PREPROD_AGA_CANDIDATE_DEMO_2026-08-01.md`
- Modify: `MANIFEST.md` only if tracked repository inventory actually changes
  and the manifest contract requires it.

**Steps**

- [x] Run all focused package, intent, loader, store, API, frontend, privacy,
  boundary, connected, cleanup, contract-generation, typecheck, and build
  gates with fresh output.
- [x] Require nonzero discovered Go, Vitest, and Playwright test counts for the
  tagged AGA paths. A missing binary, package, external input, authorization,
  base result, or cleanup authority is `blocked`, never an omitted passing
  aggregate step.
- [x] Run the smallest required full regressions from the verification matrix.
- [x] Run `git diff --check`, explicit whitespace/fence checks for any new
  untracked plan/evidence files, and inspect `git status --short` without
  altering unrelated changes.
- [x] Record exact commands, pass counts, digests, environment, failures,
  retries, process cleanup, and residue results. Do not include question text,
  source URLs, auth tokens, secrets, or private paths in evidence.
- [x] Record implementation as `verified locally` only if every acceptance
  condition below has fresh evidence. Otherwise retain `not run` or `blocked`
  literally with the exact blocker.
- [x] Keep `candidate-only`, `release pending`, and
  `production-ready: not established` in the plan, index, evidence, API, and
  UI. Do not mark real AGA Task 9 or Local Preprod Release Candidate progress.
- [x] Do not commit, push, deploy, publish, or send the package as part of plan
  completion.

**Aggregate commands**

```bash
node --test tests/aga-candidate-preprod-demo-plan-contract.test.mjs tests/aga-candidate-preprod-demo-boundary.test.mjs tests/preprod-data-boundary.test.mjs
bash scripts/check-contracts.sh
AVIA_AGA_DEMO_PACKAGE_FILE="$PWD/deliverables/AGA_ALL_FORMS_SOURCE_RISK_DRAFT_2026-08-01.zip" \
  go -C apps/api test ./internal/preproddata/agacandidatedemo -run AcceptedPackage
go -C apps/api test ./internal/preproddata/agacandidatedemo ./internal/agacandidatedemo ./internal/httpapi ./cmd/preprod-aga-candidate-demo-loader ./cmd/api
go -C apps/api test -tags=preproddemo ./internal/agacandidatedemo ./internal/httpapi ./cmd/api
go -C apps/api build -tags=preproddemo ./cmd/api
bash scripts/test-aga-candidate-demo-loader.sh
npm --prefix apps/web run typecheck
npm --prefix apps/web test -- --run aga-candidate-demo checklist-builder-page
npm --prefix apps/web run build:demo
npm --prefix apps/web run build:http
node apps/web/scripts/assert-http-artifact.mjs apps/web/dist
npm --prefix apps/web run test:e2e:aga-preprod -- --list
node tests/demo-boundary-smoke.test.js
bash scripts/test-normal-artifact-boundary.sh
AVIA_AGA_DEMO_PACKAGE_FILE="$PWD/deliverables/AGA_ALL_FORMS_SOURCE_RISK_DRAFT_2026-08-01.zip" \
AVIA_AGA_DEMO_BASE_RESULT_FILE=/absolute/private/base-result.json \
AVIA_AGA_DEMO_LOAD_AUTHORIZATION_FILE=/absolute/private/load-authorization.json \
AVIA_AGA_DEMO_CLEANUP_AUTHORIZATION_FILE=/absolute/private/cleanup-authorization.json \
  bash scripts/test-aga-candidate-preprod-demo-connected.sh
node tests/harness-docs-smoke.test.js
git diff --check
```

## Verification And Acceptance Criteria

Completion requires fresh literal evidence for every row.

| Area | Required proof |
|---|---|
| Package identity | Exact outer ZIP bytes/hash, manifest hash, JSON bytes/hash, package version/status, original archive/register identity facts |
| Form inventory | Exactly 52 unique supplied identities/hashes and complete 001–034, 035A, 036–048, 050–053 set; numeric 035/049 absent |
| Extraction queue | Exact listed 21 forms, zero questions each, literal `QUESTION_EXTRACTION_REVIEW_REQUIRED`, no invented boundary |
| Candidate questions | Exactly 1,310 unique immutable boundaries across 31 forms, valid text digests, all `NON_AUTHORITATIVE_CANDIDATE`; package provenance exactly 28 `EXACT_SOURCE_BACKED`/1,282 `EXTRACTED_CANDIDATE` |
| Source state | All 1,310 `SOURCE_MAPPING_REQUIRED`; 1,261 proposal-present review rows; 49 explicit unmapped rows; 2,329 proposal links |
| Source gate | Exact bytes, exact-byte SHA-256, effective date, clause/page locator, applicability, and named source-owner attestation required together but insufficient alone for governed resolution; no demo transition |
| Risk | Exact question 50/457/14/789 and form 11/23/4/14 provisional distributions; 789/521 proposed safety split; all 1,310 package risk-review states and 14 hard blockers; zero approved bands, approved safety-critical flags, or Finding severities |
| Loader | Exact disposable target/base result, role-isolated writer/reader/default API access, durable append-only control records, in-transaction reconciliation-bearing seal, live deterministic replay, cleanup tombstone, zero readable partial rows |
| Forbidden real state | Exact zero deltas in real intake, candidate, source authority/mapping, assignments, decisions, publication, templates, Audits, Findings, identity, notifications, outbox, and delivery |
| Provider boundary | Demo loader has no credentials/capability; Keycloak, Mailpit, MinIO, queue, and delivery counts do not change |
| API privacy | Only exact CAA Admin reads; denied roles/direct guesses authorize before parse/lookup, receive one neutral label-free not-found outcome with zero reader calls; no mutation/export routes |
| Browser privacy | No candidate data in Web Storage, IndexedDB, Cache Storage, service worker, offline outbox, telemetry, logs, traces, metrics, retained screenshots/video, memory after logout, or BFCache |
| Fail closed | Every package, target, state, digest, count, privacy, capability, seal, and reconciliation mutation fails before exposure or writes |
| Cleanup | Whole disposable namespace only; append-only cleanup attestation retained; zero target/process/browser residue |
| Claims | `candidate-only`, `release pending`, `production-ready: not established`; no real Task 9 or production/release advancement |

## Risks And Mitigations

| Risk | Mitigation |
|---|---|
| Demo rows are mistaken for real AGA intake | Separate schema, intent, operation, API capability, UI label, no bridge, and forbidden-table delta checks |
| Accepted package drifts or is replaced | Exact outer bytes/hash plus manifest/JSON semantic validation before write |
| Raw PDFs or nested content enter the loader | Exact ZIP entry allowlist, PDF/nested archive/magic rejection, bounded reads, no original archive mount |
| Frozen synthetic profiles are silently changed | Sibling overlay contract; regression tests keep profile manifests byte/behavior compatible |
| Zero-boundary forms gain invented questions | Exact 21-code set, zero-row schema assertion, semantic corruption tests, no fallback UI |
| Proposed source metadata becomes authority | Fixed proposal category, all questions remain mapping-required, six-fact real gate, no mutation route |
| Form-level proposals fill 49 question gaps | Explicit unmapped discriminant and tests forbidding inheritance or neighboring fallback |
| Proposed risk becomes severity or safety approval | Approved fields absent/null by schema, fixed provisional state, no Finding dependency, 14 hard blockers |
| Loader creates provider/delivery effects | Separate PostgreSQL-only binary/image with provider-free import closure, database-only network/secrets, static capability scan, connected zero-delta snapshots |
| Candidate text leaks to another role or browser cache | Exact Admin/CAA checks before parsing, neutral denial, no-store every outcome, no persistence/observability/artifacts, role matrix and logout/BFCache inspection |
| Partial or divergent load survives | Single transaction reconciles before the final digest-bearing seal, crash-point tests, durable append-only control records, live replay preflight, whole-namespace cleanup tombstone |
| Tagged demo capability is enabled elsewhere | Build/profile guard validates exact environment, database, compose project, base result, and seal; default reader unavailable |
| Evidence overclaims readiness | Literal output contract and explicit non-advancement of real AGA and release plans |

## Dependencies

- The accepted raw-byte-free ZIP must be available at execution time and match
  the exact contract. If absent or different, execution is `blocked`; do not
  regenerate it in this plan.
- Existing local-preprod migrations and a successful base-profile result must
  be available in an exact disposable namespace.
- A separately existing CAA Admin and denied-role accounts are needed only for
  connected UI/privacy verification. The overlay cannot create them.
- Current OpenAPI generation, Go, Node, React, Docker, PostgreSQL, OIDC, and
  isolated-browser harnesses must remain usable.
- Whole-namespace cleanup authority must be separately present before a
  connected load begins.
- The dedicated writer, tagged-reader, and normal-API database credentials and
  their least-privilege grants must be provisioned in the exact disposable
  namespace before Task 3/Task 5 connected qualification begins.

## Idempotence And Recovery

- Package validation is read-only and repeatable.
- Intent, authorization consumption, result, and cleanup-tombstone records are
  append-only, atomically published, and never overwritten.
- An exact replay is read-only and returns a result only after live target and
  seal/reconciliation preflight; it never adds demo rows.
- A crash before transaction commit leaves no demo rows. Retry requires a new
  single-use operational authorization against the same immutable intent.
- A crash after the final seal commits but before result append is recovered by
  deriving the missing external result from that already reconciled seal; no
  child row is rewritten and no incomplete projection is readable.
- A partial/unsealed schema or any divergent sealed digest blocks reads and
  further load. Recovery is whole disposable namespace drop/recreate only.
- Cleanup writes a durable tombstone. A cleaned run cannot return historical
  success or be reloaded; a recreated namespace requires a fresh run, intent,
  authorization, and base-result binding.
- Selective deletion or state repair inside `preprod_aga_demo` is forbidden.
- Existing base profile and real governed records are never rollback targets
  for this overlay.

## Progress

- [x] 2026-08-01: Read repository planning, architecture, verification, output,
  active governed AGA, completed preprod loader, and local preprod release-plan
  authorities.
- [x] 2026-08-01: Inspected the accepted raw-byte-free package and independently
  confirmed exact hashes and 52/31/21/1,310/1,261/49/2,329/174/14 boundaries.
- [x] 2026-08-01: Selected the separate immutable preprod overlay architecture
  and rejected real-table, frozen-profile, and static-browser alternatives.
- [x] 2026-08-01: Authored this planning-only ExecPlan and synchronized the
  plan index.
- [x] 2026-08-01: Verified the documentation harness, tracked diff, explicit
  plan whitespace, Markdown fences, and embedded contract JSON/counts.
- [x] 2026-08-01: Incorporated independent read-only review corrections for
  role isolation, package-fidelity digests, seal/recovery, provider-free
  loader construction, privacy/non-retention, and executable acceptance gates.
- [x] 2026-08-01: Gate 0 contract/spec freeze — `verified locally` with the
  new plan-contract test (2/2), documentation smoke, and `git diff --check`.
- [x] 2026-08-01: Task 1 bounded accepted-package validator — `verified
  locally` with synthetic path, ZIP-entry, manifest, JSON, digest, candidate
  state, and bounded-expansion rejections; an explicit accepted ZIP check
  passed its fixed outer/JSON/archive/register identities and 52/1,310/174
  inventory. No raw package text, URL, or PDF bytes were copied into fixtures.
- [x] 2026-08-02: Task 2 independent intent, authorization, result, and
  control store — `verified locally` with append-only 0600 records, hashed
  single-use authorization consumption, exact predecessor evidence, result and
  cleanup-tombstone binding, and non-replayable cleanup checks. No target or
  provider operation occurred.
- [x] 2026-08-02: Task 3 atomic immutable PostgreSQL projection — `verified
  locally` with a dedicated schema-only store, exact target/writer preflight,
  fixed-state and foreign-key constraints, append-only guards, ordered
  relationship reconciliation, final seal receipt, static SQL mutation-scope
  checks, and distinct role/bootstrap contract. No PostgreSQL target, provider,
  or real governed table was changed; connected privilege, transaction,
  forbidden-table-delta, replay, and cleanup qualification remain `not run`.
- [x] 2026-08-02: Task 4 separate provider-free one-shot command — `verified
  locally` with private 0600 config/evidence/authorization paths, no token on
  argv or environment, a dedicated Docker target, an isolated PostgreSQL-only
  Compose service, and positive normal-artifact/import closure guards. No
  container, database, provider, or external system was started or changed.
- [x] 2026-08-02: Task 5 fail-closed Admin-only read API — `verified locally`
  with five GET-only generated routes, closed OpenAPI schemas, exact CAA Admin
  authorization before query parsing/reader access, neutral no-store not-found
  denials, sealed-view PostgreSQL reader, and separate tagged reader pool.
  Connected sealed-reconciliation, role, and privacy qualification remain
  `not run`; no database or provider state was changed.
- [x] 2026-08-02: Task 6 read-only Checklist Builder demo panel — `verified
  locally` with capability-only rendering, no mock fallback or command
  control, filtered no-store sealed reads, focused Vitest coverage, typecheck,
  demo/HTTP builds, HTTP artifact scan, and tagged API test/build. Connected
  browser/privacy qualification remains `not run`.
- [x] 2026-08-02: Task 7 static privacy/no-side-effect qualification —
  `verified locally`; connected browser evidence remains `not run`.
- [x] 2026-08-02: Task 8 connected disposable qualification — `verified
  locally` in `run-20260802-local-smoke-47`. The current tagged loader binary
  and product contract digests were bound before role provisioning. The exact
  package produced a committed final in-transaction reconciliation seal; the
  separate reader reconstructed 52 forms, 1,310 questions, all fixed queues,
  relationships, distributions, null-authority fields, and seal digests. The
  rollback-only normal-API auth probe, tagged reader, and one-shot writer
  passed the least-privilege matrix without normal-API overlay access. The
  ordinary HTTP React artifact and tagged API passed the 10-test Admin,
  anonymous, seven-role denial, logout, and no-command browser matrix at
  `1440x900`, `1024x768`, and `390x844`, with retained media zero. Before and
  after forbidden-system snapshots were byte-identical; cleanup used the
  separate authorization, the exact replay was rejected as `non-replayable`,
  and container, volume, network, process, and browser-cache residue was zero.
  No real governed AGA, identity, assignment, source-attestation, decision,
  publication, delivery, Finding, Audit, release, or production record was
  created.
- [x] 2026-08-02: Task 9 aggregate verification — `verified locally`. The
  focused contract boundary passed 17/17 Node tests, OpenAPI examples passed
  16/16, the provider-free loader boundary passed 11/11 Node tests, the five
  untagged and three tagged Go package groups passed, and tagged Go inventory
  discovered 71 tests. Frontend verification passed focused 2-file/4-test and
  full 82-file/718-test Vitest runs, typecheck, both builds, the 148-file/
  180-input HTTP artifact scan, and 10-test Playwright discovery. Run 47
  supplies the connected database, OIDC/browser, privacy, forbidden-delta,
  replay, and cleanup evidence. Documentation smoke, explicit Markdown fence/
  whitespace checks, evidence privacy/mode checks, `git diff --check`, and
  final Docker/process/browser residue inspection passed. No external action
  or governed record was created.

## Decision Log

### 2026-08-01 — Keep the demo outside real governed intake

**Decision:** Use a dedicated immutable `preprod_aga_demo` projection and no
real AGA lifecycle tables.

**Reason:** The accepted package supplies candidate review material, not the
Admin/source-owner/manager facts required by real intake and publication.

### 2026-08-01 — Use an overlay, not a fifth synthetic profile

**Decision:** Keep existing preprod profiles frozen and add a separately
versioned package-bound overlay intent/result.

**Reason:** Candidate AGA text is not synthetic and must not silently change a
profile's deterministic data-source contract.

### 2026-08-01 — Make the overlay loader PostgreSQL-only

**Decision:** Reuse only an already established base result as inert
predecessor evidence and use a separate fixed-entrypoint PostgreSQL-only AGA
loader that has no Keycloak, Mailpit, MinIO, delivery, queue, API, worker, or
scheduler capability.

**Reason:** This makes the prohibition on CAA/provider delivery enforceable by
construction and independently verifiable.

### 2026-08-01 — Seal mapping and risk states in the demo

**Decision:** Provide no transition API. Every question remains mapping
required and every risk value remains provisional for the lifetime of the
projection.

**Reason:** The package does not contain authority-bound exact source bytes,
complete source facts, named attestation, expert risk approval, or authority to
create real decisions.

### 2026-08-01 — Separate database authority from API authority

**Decision:** The normal API, tagged AGA reader, and one-shot AGA writer use
separate least-privilege PostgreSQL credentials. Only the tagged reader may
read sealed overlay views; no normal API credential may access the schema.

**Reason:** GET-only routes and immutable triggers do not protect candidate
data from a shared database owner or superuser.

### 2026-08-01 — Make the final seal the reconciliation receipt

**Decision:** Reconcile every canonical digest inside the load transaction and
insert the digest-bearing seal last. External result-record recovery derives
from that seal and cannot alter readability.

**Reason:** A committed seal must never describe an unreconciled or partially
readable projection after a process crash.

## Discoveries

- The accepted ZIP is 336,524 bytes and contains no PDFs; its package JSON is
  3,370,312 bytes. The original 12,227,415-byte 53-PDF archive appears only as
  identity/provenance metadata.
- The 1,261 proposal-present questions contain 2,329 question-source proposal
  links. The 49 proposal-absent questions also have no question-level source
  reference, making the explicit unmapped split unambiguous.
- The 52 forms contain 274 form-source proposal links and 174 unique source
  references. Form-level proposals must not fill the 49 question-level gaps.
- Some proposed source rows already contain a candidate hash or page. Those
  facts are still proposal metadata because the exact source bytes and named
  authority attestation are absent from this package.
- The existing preprod loader intent is deliberately tied to frozen synthetic
  profiles and a seed. Reusing that schema for AGA candidate data would weaken
  its accepted contract.
- Current preprod API and loader use the exact disposable database owner. The
  overlay therefore requires distinct normal-API, tagged-reader, and one-shot
  writer credentials; an immutable seal alone cannot provide that boundary.
- Current migration-28 governed AGA transport has blocked mutation surfaces;
  treating the demo projection as a real import would falsely advance the
  active plan's blocked Task 9.
- The first connected Task 8 attempt confirmed that an exact predecessor
  receipt can be retained only through an explicit private handoff. Its
  pre-write snapshot must query the actual Keycloak database role and use a
  capability available in the MinIO image; otherwise it is invalid evidence
  and must stop before the overlay writer is invoked.
- Task 1's initial exact ZIP rejection was caused by an over-strict reader that
  incorrectly required proposal metadata to reuse the question-level
  `NOT_ATTESTED` state. The accepted package instead uses the two explicit
  unresolved proposal states `SOURCE_OWNER_ATTESTATION_REQUIRED` and
  `SOURCE_BYTES_NOT_LOCALLY_HASHED_SOURCE_OWNER_ATTESTATION_REQUIRED`; both
  preserve the same non-authoritative boundary. After the reader was corrected
  to accept only those unresolved states (and still reject authoritative ones),
  the exact accepted ZIP passed fixed outer/JSON identity and 52/1,310/174
  validation. No candidate text, source URL, record, provider, database, or
  runtime state was emitted or changed.
- The connected seal verification exposed PostgreSQL `timestamptz`
  microsecond persistence as a hash input: hashing an untruncated Go timestamp
  made a correctly committed seal unverifiable after reload. The writer now
  truncates the sealed timestamp to PostgreSQL precision before it computes
  and inserts the final seal; focused tests and the subsequent connected seal
  verification passed.
- The connected role matrix exposed two operational boundaries that the first
  static grant set did not prove: the normal API needs read-only access to the
  four existing Department Manager authority tables, and its complete auth
  surface needs a positive rollback-only probe in addition to overlay-denial
  tests. The final grant remains table-specific and gives the normal API no
  `preprod_aga_demo` schema access.
- Docker provenance attestations change the local manifest-list image ID even
  when the executable bytes are unchanged. The connected harness therefore
  rebuilds each tagged executable and binds `CodeDigest` to the provider-free
  loader binary SHA-256 obtained with `--network none`, while the product
  contract digest is independently reconstructed from the AGA package
  contract and source OpenAPI fragments.

## Outcome

Gate 0 outcome: the separately versioned overlay and its immutable authority,
privacy, package-fidelity, target, privilege, reconciliation, and cleanup
boundaries are `verified locally` in product specifications and the
machine-readable plan contract. Tasks 1–5 are `verified locally`: Task 3 adds
the isolated schema/role/transaction implementation and final seal receipt,
and Task 4 adds the isolated provider-free command/image boundary without
starting or changing a PostgreSQL target. Tasks 1–9 are `verified locally`,
including the exact sealed-view reconciliation, distinct credential matrix,
full tagged API/OIDC role proof, connected browser/privacy qualification,
replay rejection, forbidden-system zero delta, and whole-namespace cleanup
documented in
`docs/demo-evidence/PREPROD_AGA_CANDIDATE_DEMO_2026-08-01.md`. Task 9's
focused gates and full 82-file/718-test Vitest regression are also `verified
locally`; the plan is `ready-for-verification` pending explicit stakeholder/
user review sign-off before any lifecycle move. No real
candidate, identity, provider, delivery, publication, release, or production
state was created or changed. The intended result remains `candidate-only`,
`release pending`, and
`production-ready: not established`.

- [x] 2026-08-02: Follow-up demo presentation — `verified locally`. The
  Admin-only panel exhausts sealed question cursors and renders all 1,310
  candidate questions in a synthetic Department Manager demo handoff. No
  manager credential, assignment, decision, source attestation, publication,
  or governed lifecycle route was added; the original five GET-only CAA Admin
  routes and all literal labels remain unchanged.

## Execution Prompt

Use the following only after the user explicitly authorizes implementation:

```text
Execute docs/exec-plans/active/2026-08-01-preprod-only-aga-candidate-demo-intake-plan.md one task at a time, beginning with Gate 0. Preserve the accepted raw-byte-free package and all unrelated working-tree changes. Build a separate provider-free AGA loader; use distinct least-privilege normal-API, tagged-reader, and one-shot-writer PostgreSQL credentials; and make the final in-transaction reconciliation seal the sole database readability receipt. Do not write real governed AGA, identity, assignment, source-attestation, decision, publication, delivery, Finding, Audit, release, or production records. Stop immediately on any package, target, count, digest, privilege, privacy, seal, provider-delta, forbidden-table-delta, or cleanup mismatch. Do not commit, push, deploy, upload, or modify external systems without separate exact authorization. Report literal evidence labels and keep candidate-only, release pending, and production-ready: not established.
```
