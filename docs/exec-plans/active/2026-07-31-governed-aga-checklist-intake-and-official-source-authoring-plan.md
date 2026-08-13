# Governed AGA Checklist Intake And Official-Source Authoring Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use
> Execute this plan only after the user separately authorizes implementation.
> Use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add two distinct, fail-closed ways to create governed immutable
checklist Drafts: one from an untrusted existing AGA checklist candidate and
one from current approved official regulatory/controlled CAA-procedure
sources, without allowing either intake path to imply source authority,
technical approval, publication, or executable Audit-package eligibility.

**Current topology note (2026-08-13):** ClamAV and Gotenberg are retired from
all active runtime profiles. The maintained candidate topology uses a
fail-closed disabled scanner and the native Go document renderer; Mailpit is
local-development-only. The historical service references below describe
earlier candidate evidence or plan scope and are not current deployment
requirements.

**Architecture:** Extend the accepted governed-checklist lifecycle with a
register-first archive receipt aggregate and immutable existing-checklist
candidate snapshots, then converge both authoring paths at a governed Draft
boundary. Bind every resolved question to a typed, ordered, authority-attested
official source chain and a generalized candidate source-binding set. Derive
required Department owners on the server from reviewed provider/target/source
responsibility facts. Keep source-authority acceptance, question-mapping
attestation, Department Manager technical review, Department Manager
publication, and computed Audit-package eligibility as separate append-only
facts. Reuse Task 6 scope, publication, and Audit-pinning foundations while
generalizing currentness/impact locks beyond generation-run-only lineage; do
not create a parallel publication mechanism.

**Tech Stack:** OpenAPI 3.1 JSON source and generated YAML/Go/TypeScript,
Go HTTP/domain/worker services, PostgreSQL forward-only migrations, private
MinIO-compatible object storage, ClamAV, bounded Poppler PDF inspection in an
isolated worker, React/Vite, Vitest, Playwright, Node contract tests, and the
repository verification harness.

## Global Constraints

- This plan was the only deliverable during planning. Implementation now has
  separate user authorization through the active execution request; all local
  work remains candidate-only and the external decision gates below remain
  separately blocked.
- Preserve the root HTML/CSS/Vanilla JavaScript demo as the immutable legacy
  oracle. Do not import AGA files, extracted text, or candidate records into
  the root demo.
- Keep all React/Vite and Go/PostgreSQL results `candidate-only`. Local
  verification does not establish release or production readiness.
- Do not create, switch, rename, or delete a branch or worktree. Do not stage,
  commit, push, deploy, upload externally, or modify an external system unless
  separately and explicitly authorized in the task that performs that action.
- Do not add a `technicalExpert`, `sourceOwner`, `reviewer`, or equivalent new
  top-level application `Role`. Preserve the current eight-role vocabulary and
  model source-owner/curator and reviewer authority as scoped, effective-dated
  functional assignments for an authenticated internal CAA user.
- Existing checklist material is always non-authoritative candidate input.
  Only a current official regulatory/controlled CAA-procedure chain whose
  every required link has a separate append-only source-authority acceptance
  fact can supply authority for a resolved regulatory trace. Source discovery,
  source metadata, currentness activation, and candidate mapping attestation
  cannot substitute for source-authority acceptance.
- Client-supplied department/unit values are untrusted selection hints. The
  server derives immutable required-owner facts from current reviewed
  provider-scope, typed-target, inspection-type, and source/procedure
  responsibility records. Missing or ambiguous ownership fails closed.
- Never overwrite an import receipt, candidate snapshot, Draft revision,
  source attestation, technical-review decision, publication decision,
  published checklist version, in-progress Audit, pinned question snapshot,
  or Audit-package bytes.
- Every question must expose a non-empty `scopeRecommendation`, an explicit
  `regulatoryTrace` discriminant, exact origin, and reconciliation state.
  Incomplete mapping is the visible literal `SOURCE_MAPPING_REQUIRED`, never
  an empty citation or partially resolved trace.
- Technical approval and publication are separate Department Manager
  decisions against the same immutable candidate digest. Publication and
  executable Audit-package eligibility are also separate.
- Automatic deferral remains forbidden. A `DEFER_ELIGIBLE` recommendation is
  advisory and requires the existing clean-history, full-scope baseline,
  source-currentness, guardrail, technical-review, and publication gates.
- Use English for plans, evidence, tests, code identifiers, API fields, UI
  copy, and repository documentation.
- Each implementation task requires focused test coverage and a regression
  gate before its status advances.
- The networked intake worker must never execute Poppler directly. Untrusted
  PDFs cross a separately named one-shot parser sandbox with a separate network
  namespace, read-only inputs/root, bounded scratch/output, and fail-closed
  startup.
- Preserve unrelated user changes. The planning baseline already contains
  untracked `apps/web/.local/`, `deliverables/`, and three
  `scripts/start-plan1-visual-review-*.mjs` files; they are outside this plan.

---

## Status

- Design: approved by the user on 2026-07-31.
- Plan artifact: active.
- Initial planning documentation verification: `verified locally` with
  `node tests/harness-docs-smoke.test.js` and tracked-file
  `git diff --check` on 2026-07-31. The plan was untracked at that check, so
  ordinary `git diff --check` did not inspect this file; the corrected-plan
  verification section names a separate check that does.
- Corrected-plan documentation verification: `verified locally` on 2026-07-31
  with `node tests/harness-docs-smoke.test.js`, tracked-file
  `git diff --check`, the explicit plan whitespace check, and a balanced-
  Markdown-fence check. Three independent focused read-only passes reported no
  remaining Critical, Important, or Minor authority/privacy, security/lifecycle,
  or contract/test finding.
- Implementation: Gate 0 and the local candidate slices for Tasks 1–8 are
  `verified locally`; Task 9 is explicitly `blocked` on its real Form 048 and
  expansion prerequisites; Task 10 final evidence/verification is
  `verified locally`.
- Runtime outcome: `candidate-only` by design; the current-worktree task-owned
  full profile established migration-28 PostgreSQL, MinIO, ClamAV, Gotenberg,
  API, worker, and scheduler readiness, and connected synthetic mechanism
  checks passed. Real archive intake, real-authority decisions, browser
  qualification, and production paths remain unestablished.
- External real-source validation: `blocked` pending source-owner and
  responsible Department Manager decisions identified below.
- Release: `release pending`.
- Production-ready: not established.

## Objective And User-Visible Outcome

An authorized internal user can open the Checklist Builder and choose one of
two visibly different entry paths:

1. **Existing checklist intake.** The user receives an AGA ZIP into a private
   quarantine boundary, sees its archive hash, register, files, hashes, parse
   receipts, identity conflicts, and errors, then explicitly imports one
   accepted PDF as `EXISTING_CHECKLIST_CANDIDATE`. The imported candidate
   preserves original wording, available review guidance/operational intent,
   and any supplied result history with page/row provenance. The user creates
   a new immutable Draft from that candidate and sees field-by-field
   differences against the approved source chain. Import alone never creates
   an approved checklist, published version, or Audit package.
2. **Official-source authoring.** The user selects only current source clauses
   whose exact regulatory-authority and controlled CAA-procedure links already
   carry independent source-owner authority acceptances. The server resolves
   the ordered chain, source identities, versions, `sha256:` digests, clauses,
   locators, currentness events, and authority-attestation identities from
   persisted records rather than trusting client-supplied citation strings.
   Every question carries all mandatory scope, trace, applicability,
   currentness, verification objective, and expected Evidence fields. Any
   incomplete question makes official-source-only creation fail atomically
   with no Draft. Visible `SOURCE_MAPPING_REQUIRED` questions are available
   only on existing-candidate and hybrid Drafts and cannot pass later gates.

Both paths converge only at the immutable governed Draft boundary. A new
`HYBRID_RECONCILED` Draft is allowed when existing wording is compared with
the current approved source chain, but the official chain remains the sole
authority and every difference is visible. The responsible current Department
Manager then makes a technical-review decision and, separately, a publication
decision. The Audit-package composer independently computes eligibility at
the moment a new Audit is created.

## Scope

This plan covers:

- safe archive receipt, quarantine, malware scan, ZIP preflight, complete
  register/file inventory, SHA-256 hashing, PDF parse receipts, identity
  review, replay, cleanup, and failure recovery;
- immutable existing-checklist candidate import and question extraction review;
- new Draft creation from an existing candidate;
- new Draft creation directly from current approved official sources;
- immutable hybrid reconciliation and field-level differences;
- scoped source-owner/curator and reviewer assignments without new top-level
  roles;
- separate mechanical currentness activation, source-authority acceptance, and
  candidate mapping/applicability attestations;
- separate Department Manager technical review and publication;
- fail-closed Audit-package eligibility and source-change impact Drafts;
- PostgreSQL, Go domain/application/worker, HTTP/OpenAPI, generated transport,
  mock, React, and auditee-projection implications;
- the AGA register and all 52 forms as an immutable inventory, followed by one
  representative end-to-end vertical slice using `FSS-AGA-FORM-048.pdf`;
- a gated Phase 2 expansion path for the remaining forms; and
- acceptance evidence and handoff decisions for real source owners and
  Department Managers.

## Explicit Exclusions

- No raw AGA PDF/ZIP byte artifact, page image, full extracted-text dump,
  screenshot, or candidate record is added to Git or to the root legacy demo.
  The separately user-authorized all-form handoff is a narrow exception for
  bounded parser-derived candidate question strings/provenance and printed
  reference strings only; it remains candidate-only and raw-byte-free.
- No bulk candidate import of all 52 forms occurs before the representative
  vertical slice includes the current Admin's real Form 048 identity/boundary
  decisions, immutable real candidate, and visible real source-gap Draft,
  passes every listed mechanism/safety gate, and a human explicitly authorizes
  Phase 2 expansion.
- No AGA wording, result, title, metadata, form approval block, register row,
  or historical manager signature is treated as regulatory authority,
  source-owner attestation, technical approval, publication, or current system
  decision.
- No regulatory mapping, applicability determination, expected Evidence, or
  controlled-procedure identity is invented. Synthetic mappings remain test
  fixtures and are labelled synthetic.
- No OCR is silently applied. A scanned/unparseable page receives an explicit
  parse result and requires a separately reviewed extraction attempt.
- No legal, enforcement, certificate, compliance, closure, or production
  readiness conclusion is automated.
- No existing published version, in-progress Audit, or Audit package is
  rewritten after a source refresh or reconciliation.
- No external upload, production bucket, production database, external
  regulatory website, email, Slack/Teams, deployment, or release workflow is
  touched by this plan's implementation.
- No generalized document-management product, arbitrary Office-file importer,
  or unbounded archive format support is introduced.

## Conservative Assumptions And Ownership

- The supplied ZIP is the intended AGA intake artifact because both supplied
  copies are byte-identical. The runtime still verifies the chosen path/hash;
  filename equality is not identity.
- Blank result/status/comment fields mean `NOT_SUPPLIED`, not successful,
  compliant, clean, or historically verified. Any non-blank historical value
  remains `SUPPLIED_UNVERIFIED` until a named human validates its provenance
  and comparability.
- Form 048 is a representative engineering slice, not a declaration that it
  has priority, complete regulatory mapping, current approval, or suitability
  for execution.
- Generic Admin owns archive receipt, file identity resolution, candidate
  import, and mechanical Draft preparation. Admin does not own source truth,
  technical approval, publication, or Audit eligibility.
- Required Department ownership is never copied from an authoring request.
  The server derives it from the globally current reviewed responsibility
  facts. A raw `+` or `/` label, missing normalization, conflicting owner, or
  absent current provider/target relationship creates a blocking owner fact;
  it never selects a manager by convenience.
- A current `REGULATORY_SOURCE_OWNER` functional assignee owns source identity,
  currentness/applicability/mapping attestation only within the exact assigned
  source/department/provider scope.
- The responsible current Department Manager owns checklist technical review
  and the separate publication decision for the exact department/unit scope.
  Joint ownership requires a technical decision from every current required
  owner; after that, one current member of the required-owner set invokes the
  single separate publication command with all technical decision IDs.
- A current `CHECKLIST_REVIEWER` assignee owns comments and non-binding
  recommendations only. Auditees own none of the intake/review decisions.
- Phase 1 implements immutable functional-assignment storage and resolution but
  no real grant/revoke product route. Only canonical synthetic test fixtures
  may seed assignments until a named governance owner decides the real
  grant/revoke actor, anti-self-authorization rule, and approval evidence. A
  missing provisioning decision is `blocked`, never an implied Admin power.
- Existing Task 6 source/currentness/impact facts are authoritative for
  mechanism design. They do not prove a real Form 048 mapping. Missing real
  official source or CAA-procedure facts remain `blocked`.
- Private MinIO-compatible storage, ClamAV, and isolated Poppler are the local
  candidate implementation dependencies because the repository already uses
  those boundaries. Production qualification, retention, backup, monitoring,
  and deployment remain outside this plan.
- A resolved trace needs all mandatory locator/page/section/clause fields. A
  source whose structure cannot provide them stays `SOURCE_MAPPING_REQUIRED`;
  the implementation must not substitute an empty string or guessed locator.
- Phase 2 expansion requires the real Form 048 candidate/source-gap slice plus
  an explicit named human authorization after the mechanism/safety gate;
  approval of this plan does not authorize expansion or implementation.

## Settled Design Decision

The approved design is **Alternative A — register-first dedicated intake
aggregate**.

- Phase 1 inventories the complete archive, creates immutable receipts and the
  pending decision packet, and—only after the current Admin supplies the real
  identity/boundary decisions—proves Form 048 through candidate import, visible
  source-gap Draft creation, reconciliation review, role denials, and fail-
  closed publication/package behavior.
- Phase 2 expands candidate import only after that real slice and the separate
  mechanism/safety gate pass.
- Phase 3 proves official-source-only Draft creation independently of AGA
  wording or metadata.
- The model and UI keep `EXISTING_CHECKLIST_CANDIDATE` and
  `REGULATORY_TRACE` entry paths distinct. `HYBRID_RECONCILED` is a new Draft
  origin, not a source type and not an elevation of legacy text.

Rejected alternatives:

1. **Bulk-import all 52 forms first.** Rejected because one parser or identity
   error would be multiplied before authority, replay, reconciliation, and
   rollback semantics are proven.
2. **Reuse `regulatory_generation_runs` as the archive and candidate receipt.**
   Rejected because a generation run assumes governed structured source input,
   whereas an external ZIP needs quarantine, file identity, parser receipts,
   and non-authoritative lineage of its own.
3. **Treat the intake as a CLI-only operation.** Rejected as the primary design
   because users need durable receipts, role-aware review, visible blockers,
   and exact mock/HTTP/React parity. A read-only verifier CLI remains useful as
   acceptance support, not as the product lifecycle.

## Planning-Time AGA Inventory Facts

These facts were obtained by read-only inspection outside the repository. They
are planning evidence only and must be reverified by the implemented intake
boundary before any candidate is created.

- Two supplied attachment paths were inspected:
  `/Users/marlonjd/.codex/attachments/f4b2d810-0dae-47c7-9d87-5baef3c883cc/AGA - Checklists and Form.zip`
  and
  `/Users/marlonjd/.codex/attachments/97803f0d-ac16-4b6f-83fb-f40ec4746ffb/AGA - Checklists and Form.zip`.
- Both copies have archive SHA-256
  `dd819cfa6a670760e0cfceed94496e2e466dc53bac13e6fd792b1128314d6e32`.
- The archive is 12,227,415 bytes and contains 53 PDF entries: one register and
  52 AGA forms. Total uncompressed size is 14,026,975 bytes.
- The register SHA-256 is
  `29ed8384693b615926fc42a0ca4654be2ea9a36b0946f217975571ca0ad9564f`.
  It lists `FSS-AGA-FORM-001` through `FSS-AGA-FORM-053`, with no 049 and with
  `035A`, for exactly 52 forms.
- The representative vertical slice is `FSS-AGA-FORM-048.pdf`, SHA-256
  `495aa7b0a1edca1ac5e874e6a63f50b47c6d207aa264cc390970a7db1acdc6e3`.
  Its visible/register identity is “Checklist for the surveillance of an
  aerodrome”; it has 9 pages and 28 visible protocol questions dated
  31 March 2023.
- Form 048's embedded PDF metadata title says “Check list for inspection of
  the maintenance arrangements”. The implemented receipt must therefore show
  `IDENTITY_REVIEW_REQUIRED`; a current authenticated Admin principal must
  append an actor/digest-bound resolution before candidate import. Only that
  Admin may select the register/visible title as the
  human-readable identity, without deleting the conflicting metadata; the
  implementation plan itself makes no selection.
- The inspected archive had no path traversal, absolute path, `..` segment,
  symlink, encryption, duplicate entry name, type/magic mismatch, CRC error, or
  configured planning-limit breach. All 53 PDFs yielded non-zero text in the
  planning environment. Those observations do not waive runtime validation.
- Temporary inspection occurred under a task-owned `/private/tmp` directory.
  All 66 temporary artifacts were removed and the directory's absence was
  verified. Nothing was extracted into the repository.

The fresh planning Git baseline was branch `main` at
`d2ff2ef` (`main...origin/main`). The only pre-existing status entries were
untracked `apps/web/.local/`, `deliverables/`, and
`scripts/start-plan1-visual-review-localhostrun.mjs`,
`scripts/start-plan1-visual-review-serveo.mjs`, and
`scripts/start-plan1-visual-review-tunnel.mjs`. They remain user-owned and
outside this plan.

## Repository Orientation And Affected Interfaces

- Root HTML/CSS/Vanilla JavaScript and root scenario tests are the immutable
  legacy behavior/demo oracle. They provide regression evidence only.
- `api/openapi/source/` is the transport source of truth;
  `api/openapi/aviasurveil360.yaml`, generated Go, and generated TypeScript are
  synchronized outputs, never independent contract authorities.
- `apps/api/migrations/` is the forward-only PostgreSQL lifecycle. Existing
  migrations 21 and 25–27 contain governed checklist, source currentness, and
  impact-Draft foundations that migration 28 must extend without rewriting.
- `apps/api/internal/regulatory/` owns governed source/candidate authoring and
  currentness/impact logic. `apps/api/internal/checklistgovernance/` owns
  Department Manager review/publication and package applicability. New
  untrusted archive behavior belongs in `apps/api/internal/checklistintake/`.
- `apps/api/internal/httpapi/` owns authenticated typed handlers and problem
  projections. `apps/api/internal/application/auditee_projections.go` remains
  the auditee privacy allowlist boundary.
- `apps/web/src/backend/` owns mock/HTTP interface parity. The Admin Checklist
  Builder and Department Manager checklist management surfaces render the new
  states; auditee surfaces must not learn them.
- `scripts/verify-governed-checklist-test-inventory.mjs`, OpenAPI/Node tests,
  Go tests, Vitest, Playwright, root smoke tests, the harness matrix, and
  `git diff --check` form the acceptance boundary.
- The external attachment path and private runtime object store hold source
  bytes. Git may retain only English plan/spec/evidence metadata such as
  hashes, counts, receipt identities, command results, and blocker status.

## Terms And Lifecycle Boundaries

| Boundary | Meaning | Must not imply |
| --- | --- | --- |
| Document inventory | Immutable archive/file/register identities, sizes, hashes, validation, parse, identity, and cleanup receipts. | Candidate import, source authority, Draft, approval, publication, or Audit eligibility. |
| Candidate import | One immutable snapshot of an accepted external form and explicitly reviewed question/intent/history fields, with origin `EXISTING_CHECKLIST_CANDIDATE`. | Regulatory trace, validation, currentness, approval, publication, or execution. |
| Candidate/Draft creation | A new immutable governed checklist revision created from an existing candidate, or from a complete authority-accepted official source chain. | Technical approval, publication, or execution. |
| Hybrid reconciliation | A new `HYBRID_RECONCILED` Draft comparing candidate fields to the current official source chain with exact diffs. | Elevation of candidate wording/history to authority. |
| Source-authority attestation | Append-only accept/return of one exact source version/hash/currentness/chain-role fact by the exact source assignee. | Candidate applicability/mapping acceptance, Department Manager technical approval, or publication. |
| Candidate mapping attestation | Append-only accept/return of the complete source-chain digest, applicability, verification method/objective, and Evidence for one exact immutable Draft digest. | Source-version authority, Department Manager technical approval, or publication. |
| Technical-review decision | Append-only Department Manager accept/return/reject decision for the exact current Draft digest and exact server-derived Department ownership. | Publication. |
| Publication decision | Separate append-only Department Manager decision for the same technically approved immutable digest and all required owners. | Eligibility for every future Audit regardless of later source changes or target scope. |
| Audit-package eligibility | Server-computed point-in-time view that an exact published version is current and applicable to a requested Audit/target/inspection type; materialization rechecks under transaction locks. | Mutable approval, execution authority, or mutation/invalidation of an already pinned in-progress Audit. |

The lifecycle is:

```text
RECEIVED archive
  -> inventory and parse receipts
  -> explicit file identity resolution when required
  -> immutable EXISTING_CHECKLIST_CANDIDATE
  -> immutable GENERATED_DRAFT
  -> optional immutable HYBRID_RECONCILED successor
  -> source-authority acceptance(s) exist for every resolved chain link
  -> candidate source-mapping attestation(s)
  -> DEPARTMENT_REVIEW
  -> separate TECHNICALLY_APPROVED decision(s)
  -> separate PUBLISHED decision/version
  -> computed eligibility for a new Audit package
```

The official path starts only at selected current source clauses whose required
typed chain links carry exact source-authority acceptances and enters at
`GENERATED_DRAFT` with question origin `REGULATORY_TRACE`; it does not pass
through a document inventory or existing-candidate record.

## Roles, Functional Assignments, And Exact User Journeys

The authorization service must evaluate authenticated principal, current
top-level role, current internal-CAA application membership, functional
assignment scope, server-derived candidate Department ownership, and resource
identity on every read and command. A current Department Manager membership is
additionally required only for Department Manager decisions; it is not a
hidden prerequisite that converts source owners or reviewers into managers.
UI hiding is not authorization.

### Source owner / regulatory curator

This is the append-only functional assignment
`REGULATORY_SOURCE_OWNER`, not a new top-level role. The assignment contains
`assignmentId`, `rootId`, optional `supersedesId`, `userId`, exact internal-CAA
membership identity, `departmentId`, required `organizationalUnitId`, closed
`scopeKind`, scope fields required by that kind, `effectiveFrom`,
`effectiveTo`, `status`, grant-authority evidence identity, `grantedBy`,
`reason`, and audit timestamps. A source-owner scope cannot omit
`sourceIdentity` unless `scopeKind=REVIEWED_SOURCE_SET` names a persisted,
reviewed source-set identity. That scope kind requires its exact source-set
version/digest, provider-scope identity, typed target, inspection type,
Department, and unit; every nullable or wildcard field fails closed and null
never means global authority.

The only Phase 1 source-owner scope kinds are `SOURCE_IDENTITY` and
`REVIEWED_SOURCE_SET`. `SOURCE_IDENTITY` requires exact source identity,
permitted chain role, Department, and unit and authorizes only that link's
authority decision. `REVIEWED_SOURCE_SET` requires exact source-set version/
digest, provider-scope identity, typed target, inspection type, Department, and
unit and authorizes only the complete candidate mapping. Its absent
`sourceIdentity` is never interpreted as global. No broad department-only or
catch-all source-owner scope exists.

Journey and permissions:

1. Open the source review queue for only the assignment's current scope.
2. Read relevant candidate provenance/receipt identities and candidate
   comparison fields after Admin import, but not the Admin-only batch/file
   inventory or unrelated organization/Audit data.
3. Verify the persisted typed source chain, including every link's role,
   official source identity, immutable version, `sha256:` digest,
   locator/page/section/clause, currentness event, and independent
   source-authority acceptance. Applicability, controlled CAA-procedure
   relationship, verification objective, and expected Evidence remain a
   separate candidate-mapping review.
4. Append either a source-version authority accept/return decision before that
   source can enter a resolved Draft, or—only with one complete matching
   `REVIEWED_SOURCE_SET` assignment—a candidate mapping accept/return
   attestation against the exact immutable candidate digest and full source-
   chain digest. Link-scoped authority alone cannot attest the complete mapping.
   Neither decision can be edited or reused after its bound digest changes, and
   one decision cannot satisfy the other.
5. May create or revise source mappings within the assigned scope only when the
   same principal is also either an Admin or a current Department Manager whose
   server-derived required-owner scope includes the Draft. There is no separate
   or implicit “base author” capability; the source-owner assignment alone does
   not grant archive upload or Draft mutation.
6. Cannot technically approve, publish, materialize an Audit package, assert
   compliance, or validate a source outside the assignment.

### Department Manager with a current department assignment

Journey and permissions:

1. May create or edit a Draft only when its server-derived required-owner set
   includes the user's exact current department/unit, and may read its
   source/candidate reconciliation data.
2. May return or reject a Draft with a required reason.
3. May technically approve only the exact leaf revision owned by the current
   department after all source-owner attestations and fail-closed question
   gates pass. Joint ownership requires every required current owner; no one
   owner can stand in for another.
4. After every required owner has supplied a current technical decision, one
   current member of that required-owner set may invoke the single separate
   publication command for the same immutable digest and complete technical-
   decision ID set. The technical command cannot publish, and publication
   cannot synthesize or duplicate technical approval.
5. Cannot activate source currentness or attest regulatory authority unless
   the same user separately holds a matching `REGULATORY_SOURCE_OWNER`
   assignment.
6. Cannot execute an Inspector checklist merely because the user is a manager.

### Generic manager without a current department assignment

The manager is denied:

- archive inventory and candidate import;
- authoring, revision, submission, source-review queue, reviewer queue,
  technical approval, publication, and package materialization;
- direct reads by guessed identifier; and
- any fallback to stale, expired, future, or missing department memberships.

Every denial is server-side, returns the repository-standard problem response,
and creates no receipt, candidate, decision, publication, version, package, or
audit-log success event.

### Generic Admin

Journey and permissions:

1. May receive a ZIP into the candidate-only quarantine boundary, view all
   inventory/parse/error receipts, append a human-readable identity resolution,
   and explicitly import an accepted file as an existing checklist candidate.
2. May create/edit immutable candidate Draft revisions, initiate an
   official-source Draft only from already current, authority-accepted source
   chains, and submit the exact current leaf for review. Client-selected
   Department ownership is never trusted.
3. May use the already accepted mechanical source-currentness activation
   command when exact source version/hash facts are present. That event only
   records mechanical currentness; it is not source-owner authority or an
   applicability attestation.
4. Cannot append a source-owner acceptance unless separately assigned
   `REGULATORY_SOURCE_OWNER` for the exact scope.
5. Cannot technically approve, publish, or make a published checklist
   executable for an Audit. There is no Admin bypass route or service.

### Reviewer

This is the append-only functional assignment `CHECKLIST_REVIEWER`, not a new
top-level role. It is effective-dated and scoped to a department/unit and,
when needed, provider/source scope.

The only Phase 1 reviewer scope kinds are `DEPARTMENT_CHECKLISTS`,
`PROVIDER_CHECKLISTS`, and `REVIEWED_SOURCE_SET_CHECKLISTS`. All require exact
Department/unit; the latter two additionally require the exact provider-scope
or reviewed-source-set version/digest respectively. No missing field broadens
the queue.

Journey and permissions:

1. Read the exact assigned Draft, candidate origin, source state,
   reconciliation diff, and blockers.
2. Append comments and a non-binding `RECOMMEND`, `RETURN_RECOMMENDED`, or
   `NO_RECOMMENDATION` record against the exact digest. A Department Manager
   may append a separate acknowledgement/disposition, but the reviewer record
   itself never becomes a blocking decision or veto.
3. Cannot change original candidate facts, source mapping, source currentness,
   scope classification, technical review, publication, or Audit eligibility.
4. A reviewer recommendation never satisfies a required source-owner or
   Department Manager decision.

### Auditee

The auditee receives no route, navigation item, count, identifier, search
result, export, notification, or error detail for inventory batches, extraction-
review packets/identity decisions, candidate documents, parsing receipts,
mappings, source-owner deliberations, reviewer
comments, technical-review queues, publication deliberations, internal
blockers, or private risk/history signals.

The auditee can see only the existing organization-scoped projection of a
published checklist that has already been pinned into an authorized Audit.
That projection exposes only auditee-facing question text, requested Evidence,
answer state, and released Audit context. It excludes source-owner comments,
internal CAA notes, other organizations, and unpublished versions.

### Permission matrix

| Capability | Source owner assignment | Current Department Manager | Manager without assignment | Admin | Reviewer assignment | Auditee |
| --- | --- | --- | --- | --- | --- | --- |
| Receive archive | deny | deny | deny | allow | deny | deny |
| View inventory/candidate | candidate provenance in assignment scope only; inventory deny | candidate in own department only; inventory deny | deny | inventory and candidate allow | candidate in reviewer scope only; inventory deny | deny |
| View raw extraction-review packet | deny | deny | deny | allow | deny | deny |
| Resolve file identity | deny | deny | deny | allow | deny | deny |
| Import existing candidate | assignment alone denies; only a current Admin principal may act | deny | deny | allow | deny | deny |
| Create/revise Draft | assignment alone denies; only a separately authorized Admin or matching current Department Manager may author | own-department allow | deny | allow | deny | deny |
| Activate mechanical source currentness | deny | deny | deny | candidate-only allow through the existing exact transition command | deny | deny |
| Attest source authority or candidate mapping/applicability | per-link authority allow; mapping only with complete matching `REVIEWED_SOURCE_SET` | only with the same exact separate source-owner assignment rules | deny | only with the same exact separate source-owner assignment rules | deny | deny |
| Recommend review | optional comment | allow as part of manager review | deny | comment only | scoped allow | deny |
| Technical approval | deny | own current department only | deny | deny | deny | deny |
| Publication | deny | own current department only, separate command | deny | deny | deny | deny |
| Evaluate Audit-package eligibility | deny | own current department read/evaluate | deny | internal diagnostic read only | deny | deny |
| Materialize/compose eligible package for new Audit | deny | own current department allow through the existing exact-scope command | deny | deny | deny | deny |
| Execute answers in a materialized package | deny | deny as manager alone | deny | deny | deny | deny; only separately assigned Inspector/Lead authority executes, while the Auditee receives the authorized existing projection |

## Data Model And Immutable Lifecycle Proposal

### New import aggregate

Add forward-only migration
`apps/api/migrations/000028_governed_checklist_intake_and_authoring.up.sql`
and increment `apps/api/migrations/migrations.go` to version 28. Do not edit
applied migrations 21 or 25–27.

Create these append-only or immutable tables:

1. `checklist_import_batches`
   - `import_batch_id` UUID primary key;
   - `receipt_number` unique human-readable identity;
   - original filename and normalized display name;
   - expected and observed archive SHA-256, byte count, MIME/magic type;
   - `policy_version`, uploader identity, received/started/completed timestamps;
   - `status`: `RECEIVED`, `INVENTORY_VALIDATING`, `INVENTORY_COMPLETE`, or
     `INVENTORY_FAILED`;
   - entry/form/register/page/byte counts, manifest digest, failure code/detail,
     quarantine object key/version, scan state, and cleanup state;
   - semantic request digest and idempotency key with a unique operation scope.
2. `checklist_import_files`
   - immutable batch/ordinal/original-path/normalized-path identity;
   - SHA-256, CRC32, compressed/uncompressed sizes, MIME/magic, page count;
   - `validation_status`: `PENDING`, `ACCEPTED`, or `REJECTED`;
   - `parse_status`: `NOT_ATTEMPTED`, `PARSED`, `PARSE_FAILED`, or
     `UNSUPPORTED`;
   - immutable `initial_identity_match_state`: `REGISTER_MATCHED`,
     `IDENTITY_REVIEW_REQUIRED`, or `NOT_REGISTERED`; human resolution never
     updates this terminal inventory fact;
   - immutable `initial_candidate_import_state`: `ELIGIBLE`,
     `REQUIRES_IDENTITY_RESOLUTION`, or `INELIGIBLE`, derived by the finalizer
     from intake safety plus the initial match result;
   - embedded metadata, visible title, register title, form code/date/version,
     digest-bound intent-scoped private object-version identity, and explicit
     error code/detail;
   - unique `(import_batch_id, normalized_path)` and
     `(import_batch_id, ordinal)` constraints.
3. `checklist_import_phase_receipts`
   - one append-only accepted result per required batch phase/file selected by
     the CAS finalizer, with a foreign key to the winning terminal attempt event;
   - common phase/input/policy/result-digest/outcome fields plus a closed typed
     result payload: validation facts; exact object facts; scanner identity/
     signature/result; parser name/version/pages/output bytes/output digest plus
     exact private parser-output object intent/key/version/hash/byte identity;
     register/match facts; or scratch-cleanup marker/result. A payload for one
     phase prohibits every other phase's fields;
   - no absolute temporary path is persisted or returned.
4. `checklist_register_entries`
   - immutable register file/row/page/order, exact form code/title/version/status
     text, linked file identity when matched, and match state;
   - duplicate form codes and unmatched/extra files remain explicit receipt
     errors; no ordinal or filename inference silently fixes them.
5. `checklist_import_identity_resolutions`
   - immutable resolution/root identity, monotonic revision, nullable
     `supersedes_resolution_id`, exact expected prior leaf ID/digest (or the
     explicit first-resolution state), actor/time/reason, selected display
     identity/source, nullable transcription reason/receipt required only for
     `HUMAN_TRANSCRIPTION`, every competing register/visible/metadata value,
     expected file hash and terminal-manifest digest, operation/idempotency
     identity, and canonical semantic digest;
   - one initial root per file, unique `(file_id,revision)`, and unique
     `supersedes_resolution_id` make the one unsuperseded current leaf
     deterministic. Insert compare-and-swaps the expected leaf under a file lock;
     replay returns the same row and concurrent/divergent/stale corrections are
     `409` with no effect;
   - `ImportFileView.effectiveIdentityStatus` and `currentResolution` are derived
     without changing `initial_identity_match_state`. An unambiguous
     `REGISTER_MATCHED` file needs no human resolution; a conflict becomes
     `RESOLVED` only while an exact current resolution leaf exists. A later
     correction appends a successor resolution and requires any corrected import
     to create a new candidate snapshot; it never rewrites the earlier receipt,
     resolution, candidate, or terminal manifest. The resolution command is
     legal only for `IDENTITY_REVIEW_REQUIRED`; `REGISTER_MATCHED` needs none and
     `NOT_REGISTERED` remains ineligible rather than allowing Admin to invent a
     register match.
6. `checklist_import_extraction_review_packets` and
   `checklist_import_extraction_review_proposals`
   - one immutable `AGA_EXTRACTION_REVIEW_V1` packet per exact accepted
     file/terminal-manifest/parser-receipt tuple, with file hash, parser output
     object key/version/hash/byte identity, outcome `READY` or `FAILED`, proposal
     count, generator policy/version, created time, canonical packet/failure
     digest, and typed error. `READY` requires all proposals; `FAILED` prohibits
     proposals and can never authorize import. A failed or obsolete packet cannot
     be regenerated under a different policy in place; a parser/generator-policy
     correction requires a new import batch and parser receipt, hence a new tuple;
   - immutable proposals ordered by stable ordinal with exact original text,
     text digest, page/section/row/region/text-span locator, parser provenance,
     and proposed boundary kind. Text is private Admin-only intake data, never a
     regulatory citation or authority fact;
   - the packet is capped by `AGA_ZIP_PDF_V1`, cannot be regenerated in place,
     and is separate from the terminal batch manifest.
7. `checklist_import_extraction_decision_sets` and
   `checklist_import_extraction_decisions`
   - append-only decision-set/root/revision identity, nullable superseded set,
     exact packet/file/manifest/parser/file-hash digests, expected prior current
     decision-set leaf (or explicit no-decision state), current Admin actor/
     membership, reason/time, operation/idempotency identity, and canonical
     semantic digest;
   - one initial root per packet, unique revision and supersedes constraints,
     and file/packet locking produce one deterministic current leaf. Every set
     completely covers the ordered packet through explicit accept/split/merge/
     transcribe/exclude decisions; split/merge carries all input/output span
     identities and exclusion requires a reason;
   - candidate import either appends a complete decision set and candidate/
     questions atomically or explicitly reuses the exact current decision-set
     leaf for an identity-only corrected candidate; identical command replay
     returns the original set/candidate. A boundary correction appends a
     decision-set successor and a new candidate snapshot;
     stale, concurrent, partial, overlapping, or divergent decisions conflict
     without changing any prior packet, decision, candidate, or manifest.

`ExtractionDecisionInput` is a strict `oneOf` discriminated by `decisionKind`;
every variant has `additionalProperties=false`, a non-empty reason, and the exact
expected proposal ID/digest facts it consumes:

- `ACCEPT` consumes exactly one proposal and prohibits client output ID, text,
  digest, or locator. The server copies the original text and ordered source
  span(s) and creates one output seed ID/digest.
- `SPLIT` consumes exactly one proposal and requires 2–20 ordered non-empty,
  non-overlapping half-open UTF-8 byte-offset subspans at rune boundaries that
  are wholly inside and collectively cover the proposal; uncovered bytes may be
  only ASCII tab/LF/CR/space (`09/0A/0D/20`). It prohibits client output IDs,
  text, or digests; the server rereads each exact subspan and derives every
  output seed ID/text/digest/provenance.
- `MERGE` consumes 2–20 proposal IDs/digests that are adjacent in proposal order.
  It prohibits client output fields; the server derives one output seed whose
  text is the exact ordered source text joined with the literal V1 separator
  `\n` and retains every ordered source-span locator.
- `TRANSCRIBE` consumes exactly one proposal and requires non-empty replacement
  text plus transcription basis. It prohibits client output ID/digest/locator;
  the server preserves the original bytes/digest/locator, creates the output
  seed ID, and derives the replacement digest/provenance.
- `EXCLUDE` consumes exactly one proposal, requires an exclusion reason, and
  prohibits replacement/subspan/output fields; it creates no candidate question.

Across one decision set, inputs appear exactly once, `MERGE` groups do not
overlap another decision, decision order is minimum consumed proposal ordinal,
and split outputs are ordered by subspan. The server generates every persisted
decision/output ID and digest and rejects client-supplied versions. Persisted
`ExtractionDecisionView` variants add those server outputs while retaining the
same discriminant and prohibited-field rules. Candidate questions carry one or
more ordered source-span locators so a merge never collapses provenance.

8. `checklist_import_object_intents`
   - immutable batch/file purpose `ARCHIVE_QUARANTINE` or `PARSER_OUTPUT`, unique
     server-generated intent-scoped object
     key, expected hash/bytes/policy, exact service/intent/batch tags, expiry,
     and state `PENDING`, `VERIFIED`, or `FAILED`; no physical object is shared
     across batches in Phase 1;
   - exact object version plus observed hash/bytes/trusted checksum or rehash,
     verification/finalization
     time, and typed failure. Intent commit precedes conditional immutable PUT;
     only exact-version verification may make the object usable by downstream
     phases; intake-safety eligibility still waits for the terminal manifest.
     A successful `PDF_PARSE` event cannot commit until its bounded canonical
     structured output is finalized under its own `PARSER_OUTPUT` intent and
     exact version. That private object survives scratch cleanup, is never
     exposed by direct URL, and is the only Task 4 proposal input; it is not a
     second PDF parsing path.
9. `checklist_import_attempts` and append-only
   `checklist_import_attempt_events`
   - attempt/root/predecessor identity, ordinal, closed phase
     `ARCHIVE_VALIDATE`, `OBJECT_FINALIZE`, `ARCHIVE_SCAN`, `ENTRY_VALIDATE`,
     `PDF_SCAN`, `PDF_PARSE`, `REGISTER_PARSE`, `IDENTITY_MATCH`, or
     `SCRATCH_CLEANUP`, optional file identity, input/policy,
     lease-owner/expiry, monotonic lease generation/fencing token, start time,
     and bounded dependency identity;
   - `OBJECT_FINALIZE` is batch-only and finalizes the archive-quarantine
     object. A file-scoped `PDF_PARSE` attempt owns creation/recovery and exact-
     version verification of its `PARSER_OUTPUT` intent before its successful
     terminal event; there is no unlisted parser-output-finalization phase;
   - each event stores its immutable event/attempt identity, terminal state,
     phase-result digest or typed failure, worker/fencing token, and completion
     time; `attempt_id` is unique and foreign-keyed to its exact attempt;
   - deterministic internal validation and external object/scan/parse calls all
     use this fenced attempt model. Lease/renewal and exactly one terminal
     `SUCCEEDED`, `FAILED`, or
     `ABANDONED` event enforced by a unique attempt foreign key. A retry links
     to the abandoned/failed predecessor and never overwrites it; attempt events
     never carry a batch manifest. Terminal insertion compare-and-swaps the
     exact attempt ID, current owner/token, unexpired lease, and absence of a
     terminal event using database transaction time. Expiry recovery atomically
     appends `ABANDONED` and creates the linked retry; a stale worker receives a
     typed conflict.

PostgreSQL constraints and triggers must prevent UPDATE/DELETE of completed
receipts, accepted file/register/match facts, extraction packets/proposals,
identity resolutions, extraction decision sets/decisions, attempts, and attempt
events. State transition functions may only append attempts and move a batch
forward through the fixed phase DAG. One compare-and-
swap finalization transition selects the terminal event for every executed
phase, writes completed file/register facts, the single canonical manifest
digest, intake-safety eligibility, and terminal batch status. It refuses to run
until the unconditional scratch-cleanup chain is closed. A failed batch retains
receipts but is not candidate-eligible.

The batch finalizer hashes exactly `AGA_IMPORT_MANIFEST_V1`, a strict terminal
union keyed by `terminalOutcome=INVENTORY_COMPLETE` or `INVENTORY_FAILED`.
Common fields are schema/policy version, batch ID, expected archive hash,
nullable fully observed archive hash, observed byte count, fixed ordered phase
nodes, completed register facts ordered by page/row/entry ID, completed file
facts ordered by archive ordinal then normalized path, fixed counts, batch
`intakeSafetyEligible`, each file's immutable `initialCandidateImportState`
(`ELIGIBLE`, `REQUIRES_IDENTITY_RESOLUTION`, or `INELIGIBLE`), and error codes
ordered by policy order then file ordinal. Exact
archive-quarantine object key/version/checksum is required only when
`OBJECT_FINALIZE` succeeded and is otherwise prohibited. Each successful
`PDF_PARSE` node/file fact separately requires its exact private parser-output
intent/key/version/hash/bytes; a non-successful parse prohibits them. A failure
may therefore terminate before archive PUT with no object, register, file, scan,
or parser identity.

Manifest file facts contain only the immutable initial identity-match result.
Post-terminal identity resolutions, extraction packets/proposals, extraction
decision sets, and candidates are deliberately excluded; they bind the exact
manifest digest but can never change its bytes or terminal status. Candidate
import eligibility is derived from `intakeSafetyEligible`, the file's initial
state, and the exact current resolution leaf; it is not stored back into the
manifest.

Each phase node is keyed by phase plus optional file ordinal and has state
`SUCCEEDED`, `FAILED`, `ABANDONED_EXHAUSTED`, or
`NOT_RUN_DUE_TO_PREDECESSOR`. Executed states require the selected fenced
terminal-event ID and result digest; `NOT_RUN_DUE_TO_PREDECESSOR` prohibits
those fields and requires the blocking phase key/error code. Batch phases are
ordered `ARCHIVE_VALIDATE`, `OBJECT_FINALIZE`, `ARCHIVE_SCAN`; after safe entry
enumeration, every PDF adds `ENTRY_VALIDATE`, `PDF_SCAN`, and `PDF_PARSE` in
archive-ordinal order. Exactly one register `REGISTER_PARSE` follows successful
parsing of all safe PDFs. Then every non-register PDF adds `IDENTITY_MATCH` in
archive-ordinal order, each depending on its own successful parse and the
successful register parse. This canonical topological order is independent of
the register's archive ordinal. Missing/duplicate registers fail the register
phase and schedule no identity match.

`SCRATCH_CLEANUP` is an unconditional fenced `finally` phase after all other
known nodes are terminal or marked `NOT_RUN_DUE_TO_PREDECESSOR`. It is present
even when archive validation fails before entry enumeration and can never be
`NOT_RUN_DUE_TO_PREDECESSOR`. Phase 1 executes one phase at a time; after the
first hard failure it stops ordinary work, marks every known downstream ordinary
node not run, and still runs cleanup. Any later parallel policy requires a new
policy/manifest version and tests.

For an executed phase, the selected event is the lowest-ordinal valid fenced
`SUCCEEDED`; if no success exists after a hard error or exhausted retry chain,
it is that closed chain's highest-ordinal terminal event. The finalizer refuses
an active/retryable chain, an unmarked downstream gap, or a success manifest
without every required successful node. It excludes timestamps, worker/lease
identity, lease tokens, query order, and non-winning retry history. Serialize
the typed struct with the exact existing `regulatory.CanonicalSHA256`
canonical-JSON algorithm and store its `sha256:` digest; PostgreSQL JSONB/text
ordering is never a digest input. A cleanup success is required for
`INVENTORY_COMPLETE`; an exhausted cleanup failure selects its terminal event
and forces `INVENTORY_FAILED`, `intakeSafetyEligible=false`, and
`TEMP_CLEANUP_FAILED`; no mutable candidate-eligibility flag exists.
Golden vectors cover pre-PUT ZIP rejection,
archive-malware failure, per-file-malware failure, parser failure, identity
conflict completion, cleanup failure, crash-before-cleanup/restart, and full
success; reordered-query, retry-chain, replay, and competing-finalizer tests
must reproduce exact bytes/digests.

### Existing-checklist candidate snapshot

Create:

1. `existing_checklist_candidates`
   - `existing_candidate_id`, `candidate_root_id`, monotonic revision, nullable
     `supersedes_existing_candidate_id` plus its exact content digest, exact
     import batch/file/register/initial-match receipt IDs, identity basis
     `REGISTER_MATCHED` or `ADMIN_RESOLUTION`, the
     resolution ID/digest pair required exactly for `ADMIN_RESOLUTION` and
     prohibited for `REGISTER_MATCHED`, exact extraction packet and decision-set
     ID/digest pairs, candidate origin
     `EXISTING_CHECKLIST_CANDIDATE`, candidate schema version, source file
     SHA-256/object version, form identity/date/version, creator, creation
     reason/time, question count, and canonical content digest;
   - one initial root per import file, root=self/revision=1 for the initial row,
     inherited root plus revision+1 for a correction, unique predecessor and
     `(candidate_root_id,revision)` constraints, and a required expected-current-
     leaf CAS make the one unsuperseded leaf deterministic;
   - no status named approved, validated, published, or executable.
2. `existing_checklist_candidate_questions`
   - immutable question identity/order and exact original wording;
   - one or more ordered page/section/row/region/text-span locators and exact
     extraction proposal/decision/output-seed IDs;
   - exact review guidance or operational-intent text when supplied;
   - original expected-Evidence text when supplied;
   - original applicability/scope/result/status/comment values when supplied;
   - per-field provenance and state `SUPPLIED_UNVERIFIED`, `NOT_SUPPLIED`,
     `UNREADABLE`, or `HUMAN_TRANSCRIBED_WITH_RECEIPT`;
   - extraction confidence is advisory and cannot satisfy human review.
The import service creates all candidate rows in one transaction only after
the selected file has accepted validation/scan/parse/register/match/cleanup
receipts, effective identity is either unambiguous `REGISTER_MATCHED` or bound to
the exact current Admin resolution leaf, and a complete extraction decision set
is appended or replayed against the exact packet/file/manifest digests. The
transaction locks and rereads the identity, extraction-decision, and—on a
correction—candidate current leaves before inserting the candidate and
questions. A failure creates no decision set, candidate, or partial questions.

`ExistingChecklistCandidateView` projects the exact root/revision/predecessor
pair and strict `candidateCurrentness`: `CURRENT` prohibits successor fields;
`SUPERSEDED` requires the exact immediate successor ID/revision/content digest.
Correction import locks the candidate root and names its expected current leaf.
An old snapshot stays readable but cannot create another Draft, correction, or
hybrid reconciliation successor.
Any already-created Draft/published version remains byte-for-byte historical;
after its source candidate is superseded it receives the derived blocker
`EXISTING_CANDIDATE_SUPERSEDED` for new validation, technical approval,
publication, or Audit-package materialization. A fresh Draft from the corrected
current candidate is a new governed Draft root and must pass every normal source,
mapping, technical, and publication gate. Existing in-progress Audit bytes never
change.

### Official source authority and candidate source bindings

First create the immutable functional-scope catalog that
`REVIEWED_SOURCE_SET` assignments reference:

- `governed_reviewed_source_sets` stores `reviewedSourceSetId`, `rootId`,
  monotonic version, optional `supersedesSetId` plus its exact digest, schema
  version, canonical digest, authority-evidence identity, creator, reason,
  created time, and provenance `CANONICAL_TEST_FIXTURE` or
  `GOVERNANCE_DIRECTIVE`;
- `governed_reviewed_source_set_links` stores the set version, stable ordinal,
  exact `sourceId`, `sourceVersionId`, `sourceHash`, `sourceClass`, and permitted
  `chainRole`, with foreign-key/check constraints to that exact immutable
  official source version and its required `sha256:` hash/class. The canonical
  ordered list is the candidate chain's distinct complete source-version/hash/
  class/role tuples in first-use order; every candidate link must be covered
  exactly and no extra set entry is silently ignored;
- `governed_checklist_functional_assignments` with
  `scopeKind=REVIEWED_SOURCE_SET` has a composite foreign key to exact set ID,
  version, and digest and separately pins provider scope, typed target,
  inspection type, Department, and unit. A set successor never broadens an
  existing assignment; a new assignment successor must pin it explicitly; and
- reviewed source sets are authority-scope descriptors, not source approval or
  mapping acceptance. Phase 1 permits only the canonical synthetic fixture
  seed. Real set creation remains blocked with real assignment provisioning
  until the named governance directive defines its actor/evidence path; there
  is no Admin creation route.

The reviewed-source-set digest is
`regulatory.CanonicalSHA256(reviewedSourceSetV1Payload)`, where the typed
payload contains exactly `schemaVersion=REVIEWED_SOURCE_SET_V1`, set/root/
version identities, nullable superseded-set ID and digest, and the ordered link tuples
`(ordinal,sourceId,sourceVersionId,sourceHash,sourceClass,chainRole)`, with
`sourceHash` matching `^sha256:[0-9a-f]{64}$`. Actor, evidence, reason, and
timestamps remain immutable metadata but are excluded
from this content digest. Creation and assignment tests use checked-in golden
payload bytes/digests; query order and PostgreSQL JSON representation are never
digest inputs.

`governed_reviewed_source_sets` is distinct from candidate-specific
`governed_candidate_source_binding_sets`: the former limits who may decide,
while the latter freezes the exact candidate clauses, versions, hashes,
locators, and attestations. Orphan, wrong-version/digest, wrong-candidate-scope,
partial/extra/reordered coverage, and client-invented set identities fail
closed. A source version, hash, or class change requires a new reviewed-set
successor and a separately effective assignment successor; the prior assignment
does not cover the changed tuple.

Create these append-only facts before extending Draft lineage:

1. `regulatory_source_authority_attestations`
   - immutable `decisionId`, `decisionRootId`, optional
     `supersedesDecisionId`, outcome `ACCEPT` or `RETURN`, and decision against
     exact source identity, source version ID, `sha256:` digest, source class,
     intended chain role, and currentness event;
   - exact current `REGULATORY_SOURCE_OWNER` assignment, actor, reason,
     operation/idempotency identities, semantic digest, and decision time;
   - source discovery, `PUBLIC_REFERENCE`, effective dates, or a currentness
     event cannot synthesize an accepted authority decision.
2. `governed_candidate_source_binding_sets`
   - immutable binding-set ID, candidate/draft revision and content digest,
     source-chain policy `OFFICIAL_CHECKLIST_SOURCE_CHAIN_V1`, canonical chain
     digest, and creation identity/time;
   - one exact set per governed Draft revision that contains any `RESOLVED`
     question.
3. `governed_candidate_source_chain_links`
   - binding-set/question/chain ordinal and role
     `REGULATORY_AUTHORITY`, `NATIONAL_REQUIREMENT`,
     `CONTROLLED_CAA_PROCEDURE`, or `DERIVED_CONTEXT`;
   - persisted clause/source/version/hash/locator/currentness-event identity and
     exact accepted source-authority attestation ID;
   - a unique ordered identity and no client-authored authority string.
4. `governed_required_owner_resolution_facts`
   - immutable candidate/revision/content-digest identity, provider scope,
     typed target, inspection type, relevant source/procedure responsibility
     rows, catalog/version identities, canonical input digest, and resolution
     time;
   - outcome `RESOLVED` with ordered Department/unit owners and joint
     `approvalRequired` facts, or `REVIEW_REQUIRED` with exact missing/
     ambiguous responsibility identities. Only `RESOLVED` populates the
     existing `candidate_required_owner_assignments` rows.

`template_draft_versions` and its binding set use preallocated immutable IDs
and named composite `DEFERRABLE INITIALLY DEFERRED` foreign keys in both
directions. Add `template_draft_versions.governed_source_binding_set_id`; the
constraint `template_draft_versions_governed_binding_fk` covers binding-set ID
plus candidate ID/revision/content digest and references the identically ordered
unique key on `governed_candidate_source_binding_sets`. The reverse constraint
`governed_binding_sets_candidate_fk` covers candidate ID/revision/content
digest and references the existing exact candidate identity. Both rows and all
ordered links are inserted in one transaction; the deferred constraints plus
the deferred `governed_candidate_binding_required_at_commit` constraint trigger
require a binding for official/hybrid rows at commit while allowing null only
for an eligible source-gap row. UPDATE/DELETE guards, orphan rejection,
rollback, replay, and concurrent insert tests freeze this choice.

`OFFICIAL_CHECKLIST_SOURCE_CHAIN_V1` requires at least one accepted
`REGULATORY_AUTHORITY` link and one accepted `CONTROLLED_CAA_PROCEDURE` link
for every resolved question. A national or derived-context link is additive
and cannot satisfy either required role. Every link must be current when the
binding set is committed. Missing authority, an authority return, mixed
currentness, or a changed hash leaves an existing-candidate/hybrid question at
`SOURCE_MAPPING_REQUIRED` and makes official-source-only creation fail with no
Draft.

### Governed Draft lineage and origins

Extend the existing governed candidate records additively with:

- nullable `entry_path`: `EXISTING_CHECKLIST_CANDIDATE` or
  `REGULATORY_TRACE`; null is reserved only for historical non-governed rows;
- nullable `existing_candidate_id`, nullable existing `generation_run_id`, and
  nullable `governed_source_binding_set_id` under the truth table below;
- nullable `legacy_authority_state`, whose only value
  `PRE_V28_UNATTESTED` is required exactly for migrated version-21–27 governed
  generation-run rows and prohibited on every new row;
- `lineage_kind`: `INITIAL_DRAFT`, `CANDIDATE_SUCCESSOR`,
  `HYBRID_RECONCILIATION`, or `SOURCE_IMPACT_REVIEW`;
- exact candidate root, predecessor candidate/draft ID and digest;
- mapping-completion counts and explicit blocker digest;
- for every new version-28 row, exact creation basis `ADMIN`,
  `REQUIRED_DEPARTMENT_MANAGER`, or `SOURCE_IMPACT_SYSTEM`, actor/membership or
  exact impact-trigger/service identity as applicable, server-derived owner-
  resolution input/digest, and exact Department/unit scope at creation.
  `SOURCE_IMPACT_SYSTEM` is not a user role and is legal only for the idempotent
  source-impact successor command. Pre-v28 compatibility rows leave these new
  provenance fields null rather than inventing history.

| Row kind | `entry_path` | Existing candidate | Generation run | Source binding set | Root/predecessor |
| --- | --- | --- | --- | --- | --- |
| Historical non-governed | null | null | null | null | existing legacy rules only |
| Version-21–27 governed generation-run root/successor/impact | backfill `REGULATORY_TRACE` | null | required and unchanged per existing row | null; expose `PRE_V28_UNATTESTED`, never fabricate an attestation | preserve exact existing root/predecessor/impact linkage |
| New version-28 generation-run initial | `REGULATORY_TRACE` | null | required | required | root=self; predecessor null |
| New version-28 generation-run successor | inherited | null | required and unchanged from predecessor/root | required when any question is `RESOLVED`; a gap impact may be null | root inherited; predecessor required |
| Existing initial/source-gap Draft | `EXISTING_CHECKLIST_CANDIDATE` | required | null | null unless a resolved question exists | root=self; predecessor null |
| Existing successor | inherited | required and unchanged | null | required when any question is resolved | root inherited; predecessor required |
| Hybrid reconciliation | `EXISTING_CHECKLIST_CANDIDATE` | required and unchanged | null | required | root inherited; predecessor required |
| Official-source initial Draft | `REGULATORY_TRACE` | null | null; direct authoring must not synthesize a generation run | required | root=self; predecessor null |
| Official-source successor | inherited | null | null | required | root inherited; predecessor required |
| Source-impact review | inherited | inherited under its root rules | inherited under its root rules | required iff at least one question remains `RESOLVED`; when present it excludes every stale link and differs from predecessor | root inherited; predecessor required |

A source activation whose replacement lacks authority acceptance or complete
reconciliation creates a visible gap successor with
`SOURCE_MAPPING_REQUIRED`; it must not fabricate an accepted replacement
binding set merely to satisfy the impact row shape.

Migration 28 orders the compatibility change atomically: add nullable columns
and new stores first; backfill every `generation_run_id IS NOT NULL` version-27
row with `entry_path=REGULATORY_TRACE`,
`legacy_authority_state=PRE_V28_UNATTESTED`, and a deterministic
`lineage_kind`; validate counts and
all pre-migration content/publication/Audit digests; then install the new
constraints/triggers and remove the old sentinel guards. Derive
`SOURCE_IMPACT_REVIEW` from the existing impact link first,
`HYBRID_RECONCILIATION` from the frozen question-origin snapshots second,
otherwise `INITIAL_DRAFT` when predecessor is null and `CANDIDATE_SUCCESSOR`
when it is present. Do not insert source-authority/mapping attestations or a
binding set, governed required-owner-resolution fact, new author basis, or
blocker digest during backfill.

Pre-v28 governed candidates, publications, and in-progress Audit packages stay
readable byte-for-byte. Their literal `PRE_V28_UNATTESTED` lineage is denied any
new validation, technical approval, publication, or package materialization
with `LEGACY_AUTHORITY_UNATTESTED` until an authorized new version-28 successor
freezes a complete binding set and current attestations. Existing decisions and
packages are not revoked or rewritten. Upgrade tests cover every old root/
successor/impact shape and prove zero synthesized acceptance, decision,
publication, or history mutation.

`PRE_V28_GENERATION_RUN` is row-local and migration-only, not inherited by a
new row. If source activation reaches a current pre-v28 leaf before any attested
successor, the impact transaction creates one modern all-gap
`GENERATION_RUN` successor with the same generation-run/root identity,
`SOURCE_IMPACT_SYSTEM` creation basis, fresh current owner-resolution/blocker
facts, null binding, no legacy state, and visible `SOURCE_MAPPING_REQUIRED`;
it never treats the legacy source snapshots as accepted. An authorized later
successor on that chain keeps the same generation-run ID and may become
resolved only with a complete binding/current attestations. A subsequent impact
also uses `GENERATION_RUN`. Test the exact chain legacy root → modern gap impact
→ attested modern successor → later impact. A different generation run starts a
new `INITIAL_DRAFT` root; it cannot be spliced into an existing root.

Every governed row requires `entry_path`, `lineage_kind`, candidate root,
content digest, and schema version. Every **new version-28** governed row also
requires its creation basis, owner-resolution digest, and blocker digest.
`PRE_V28_UNATTESTED` rows require those new fields to remain null and prohibit a
`governed_required_owner_resolution_facts` row. `entry_path IS NOT NULL`—never
`generation_run_id IS NOT NULL`—is the
governed discriminator for immutability, review-queue inclusion, leaf checks,
command eligibility, source-currentness locks, impact linkage, and package
materialization. Migration 28 must replace/generalize the existing
generation-run sentinel constraints, guards, indexes, command references, and
currentness bindings without changing any version-27 candidate, decision,
published version, Audit snapshot, or package bytes/digest.

Retain the existing per-question origin enum exactly:
`REGULATORY_TRACE`, `EXISTING_CHECKLIST_CANDIDATE`, and
`HYBRID_RECONCILED`. An existing-candidate Draft initially uses
`EXISTING_CHECKLIST_CANDIDATE` and normally has
`SOURCE_MAPPING_REQUIRED`. A hybrid successor uses `HYBRID_RECONCILED` only
for questions whose official mapping and field comparison are persisted.

### Regulatory trace discriminated union

Split immutable trace content from decision projections. Define strict OpenAPI
`GovernedRegulatoryTraceContent` `oneOf` variants and equivalent Go/TypeScript
domain validation, then define `GovernedRegulatoryTraceView` as that content
plus separately derived mapping-review and technical-review projections:

- `SOURCE_MAPPING_REQUIRED` requires `state`, non-empty `gapReason`, ordered
  `missingFields`, and `lastReviewedAt`; it rejects source identity, citation,
  or technical-approval claims that make the gap look resolved.
- `RESOLVED` requires non-empty:
  ordered `sourceChain`, `sourceChainDigest`, `applicability`, typed
  `applicabilityDisposition`, `applicabilityRationale`,
  `verificationObjective`, `verificationMethod`, non-empty
  `expectedEvidence`, and `currentnessState=CURRENT`. Every source-chain link
  requires its role, source identity/title/type, immutable version, canonical
  `sha256:` digest matching `^sha256:[0-9a-f]{64}$`, persisted clause ID,
  locator/page/section/clause, exact currentness event ID, and accepted
  source-authority attestation ID.

Immutable Draft content contains no decision ID or mutable review-state field.
For a resolved trace with no decision, the view projects
`mappingReviewState=SOURCE_OWNER_REVIEW_REQUIRED` and
`technicalReviewState=TECHNICAL_REVIEW_REQUIRED`; for a source gap it projects
mapping state `SOURCE_MAPPING_REQUIRED` and technical state `NOT_AVAILABLE`.
Accepted/returned mapping state, source-mapping attestation ID, technical
decision state, and technical decision ID are read-only projections derived
from separate append-only rows for the exact candidate digest and excluded
from the canonical content digest. A decision never changes stored trace
bytes. A mapping/source change creates a successor Draft and requires a new
attestation.

If any resolved field is unknown or legitimately unavailable, the question
stays `SOURCE_MAPPING_REQUIRED` on an existing-candidate or hybrid Draft;
official-source-only creation rejects the command atomically. Empty strings,
empty arrays, placeholder/free-text authority citations, partial chains, and
partial resolved traces are rejected at contract and domain boundaries.

### Scope recommendation

Every Draft and published question retains the Task 6 required
`scopeRecommendation` fields: classification, input signals, operational
history basis, rationale, guardrails, approval-review state, and
`automaticDeferral=false`.

For AGA intake:

- absent result history sets `unknownHistory=true` and
  `operationalHistoryBasis=NOT_SUPPLIED_BY_CANDIDATE`;
- supplied result history remains `SUPPLIED_UNVERIFIED` until the responsible
  owner confirms comparability and completeness;
- no blank historical column becomes clean history;
- an imported “mandatory” or “safety” marker is a candidate observation, not
  a final scope classification.

### Reconciliation diff

Create `governed_question_reconciliations` as an immutable child of the new
Draft revision. Persist exact before/after values and a reason for:

- wording;
- verification objective/method;
- expected Evidence;
- applicability and rationale;
- scope classification, rationale, signals, and guardrails;
- complete typed authority chain and regulatory trace, including every
  source/link role, identity/version/hash, authority decision, currentness
  event, and locator/page/section/clause;
- question split/merge/exclusion lineage; and
- original operational intent and result-history disposition.

Each field has an exact `before`, `after`, `beforeDigest`, `afterDigest`,
`outcome`, `reason`, and provenance identity. Outcomes are `UNCHANGED`,
`CHANGED`, `ADDED`, `REMOVED`, or `UNMAPPED`. Split/merge/exclusion rows also
carry every predecessor/successor question identity. `UNMAPPED` forces
`SOURCE_MAPPING_REQUIRED`. The server computes diff outcomes from immutable
snapshots; clients cannot submit trusted boolean flags such as
`wordingChanged` without matching before/after digests. The generated
contract, Go/HTTP/mock projections, and React UI must expose every listed
family, including verification method/objective, applicability rationale,
operational intent, and history disposition.

### Functional assignments, comments, and attestations

Create:

- `governed_checklist_functional_assignments` with immutable root/supersedes
  identity, assignment type `REGULATORY_SOURCE_OWNER` or
  `CHECKLIST_REVIEWER`, exact internal-CAA subject/membership, closed scope
  kind and required scope fields, UTC effective interval, grant/revoke event,
  grant-authority evidence, actor, reason, and non-overlap constraints;
- `governed_checklist_review_comments`, append-only against exact candidate ID,
  revision, digest, field/question target, visibility `INTERNAL_CAA`, and
  optional non-binding recommendation;
- `governed_reviewer_recommendation_dispositions`, append-only Department
  Manager `ACKNOWLEDGED`, `ADOPTED_AS_RETURN`, or
  `NOT_ADOPTED_WITH_REASON` records against exact recommendation ID and
  candidate digest. The technical command appends its dispositions atomically;
  only `ADOPTED_AS_RETURN` appends the existing manager-owned return decision;
- `governed_source_mapping_attestations`, append-only decisions with
  `decisionId`, `decisionRootId`, optional `supersedesDecisionId`, outcome
  `ACCEPT` or `RETURN`, complete source-chain digest/currentness/applicability
  facts or an exact incomplete-proposal digest, exact candidate digest, reason,
  and timestamp. Every mapping decision requires one current
  `REGULATORY_SOURCE_OWNER` assignment whose
  `scopeKind=REVIEWED_SOURCE_SET` covers the complete ordered chain and the
  candidate's provider/target/inspection/Department/unit scope; a link-only or
  partial-set assignment cannot accept **or return** the complete mapping. An
  `ACCEPT` additionally requires complete mapping facts; `RETURN` may bind the
  exact incomplete proposal. These rows are decision projections and are never
  embedded into the candidate content digest.

Both attestation tables store a canonical immutable `decisionSubjectDigest`.
For source authority it binds source class, source/version/hash/currentness/
chain role; for mapping it binds candidate revision/content digest, complete
chain digest or exact incomplete-proposal digest, and mapping scope. There is
one root per subject digest. A successor must name the
current leaf of the same root and subject; a unique `supersedesDecisionId`
constraint makes concurrent successors conflict with `409`. Legal transitions
are initial `ACCEPT` or `RETURN`, `RETURN` to later `ACCEPT`, and `ACCEPT` to
`RETURN`. `ACCEPT` to `ACCEPT` is allowed only after an assignment transfer and
must cite the prior accepted decision as `carriedFromDecisionId`; it is still
an ordinary new `ACCEPT`, not a third outcome. Identical idempotent replay
returns the same decision. Any other same-outcome or cross-subject successor
conflicts.

Every validation, approval, publication, and new-package transaction resolves
and locks the deterministic current leaf and accepts only current `ACCEPT`.
Superseding an acceptance with `RETURN` immediately blocks new effects and
atomically records an impact event: source-authority return fans out through
the existing shared impact aggregate/links, while mapping return creates an
idempotent gap successor for that candidate root. Previously published and
pinned Audit bytes remain immutable. A historical acceptance never becomes
current again merely because a later assignment expires or is transferred.

Resolve functional authority by selecting the globally latest effective
successor per assignment root before filtering by subject or scope. Null,
missing, overlapping, future, expired, revoked, inactive-department, inactive-
unit, or cross-scope facts fail closed; null scope never means global. A source
owner/reviewer needs current internal-CAA application membership but not a
Department Manager membership. Expired/revoked assignments cannot authorize a
new command. Earlier decisions remain historically attributable but cannot be
replayed onto a successor digest or changed source hash. After an assignment
transfer, the new current assignee must append the ordinary transfer-bound
`ACCEPT` successor described above before a new technical/publication command;
transfer never silently reuses authority.

Effective intervals are half-open UTC intervals evaluated at server/database
command time: `effectiveFrom <= commandTime < effectiveTo`, with null
`effectiveTo` meaning no scheduled end. Phase 1 requires exactly one current
source-owner assignment for each exact source link/scope before that link's
authority decision, and separately exactly one current complete
`REVIEWED_SOURCE_SET` assignment before a candidate mapping decision. Zero,
multiple, partial-chain, or candidate-scope-mismatched matches are
`FUNCTIONAL_ASSIGNMENT_REQUIRED` and create no acceptance. Different owners may
accept different source links, but none may attest the complete mapping without
the full-set assignment. Joint authority within either decision is outside
Phase 1 until a governance directive defines aggregation. Reviewer assignments
may overlap because recommendations are non-binding; queue union deduplicates
by candidate/digest and preserves every assignment ID.

Phase 1 exposes no real grant/revoke command. Canonical synthetic fixtures may
insert assignment facts only under the canonical test profile. A real
provisioning route remains blocked until the named governance owner supplies
the grant/revoke actor, approval evidence, and anti-self-authorization rule;
implementation must not default that power to Admin.

### Technical review, publication, and package eligibility

Reuse the existing required-owner, review-decision, publication, template
version, source-currentness, impact-Draft, package, and Audit pin tables.
Generalize their generation-run-only source lookup to the immutable candidate
source-binding set; do not create a second review/publication lifecycle.

- Each technical decision requires current accepted source-authority facts for
  every chain link, a current complete-set candidate-mapping `ACCEPT` for the
  exact Draft digest, and an actor who is one
  exact current server-derived required owner. The compatibility projection
  becomes `TECHNICALLY_APPROVED` only after every required owner has appended
  its separate matching approval.
- Publication is one separate command and one publication-decision row invoked
  by one actor who is a current member of the server-derived required-owner
  set. It cites the complete set of distinct current technical decision IDs—one
  from every required owner—and requires unchanged digest, current sources,
  current attestations, and no blockers. Joint ownership never creates multiple
  publication rows and one owner's technical approval never substitutes for
  another's.
- `template_draft_versions.status`, exposed through
  `regulatory.CandidateView` and OpenAPI `GovernedCandidateView`, remains a
  compatibility projection; append-only decision rows are the source of truth.
- Add a computed `GovernedAuditPackageEligibilityView` with
  `eligible`, `evaluatedAt`, `publishedVersionId`, `candidateDigest`, exact
  Audit scope inputs, source-currentness digest, and ordered `blockingIssues`.
  It is not a mutable approval flag.
- Generalize the existing source-impact aggregate with trigger kind
  `SOURCE_CURRENTNESS_ACTIVATED`, `SOURCE_AUTHORITY_RETURNED`, or
  `SOURCE_MAPPING_RETURNED` plus exact immutable trigger-event ID. Each trigger
  creates one aggregate and one idempotent link/successor per affected candidate
  root. Activation or authority return fans out through all bound roots; mapping
  return targets its one root. Affected published versions become ineligible for
  **new** Audit packages.
- Eligibility views are advisory. Official-source creation, hybrid
  reconciliation, every source-binding-set/successor commit, source
  attestation, technical approval, publication, and package materialization
  acquire deterministic transaction-scoped locks for every proposed or bound
  source identity and re-read source currentness, attestation leaves, impact
  state, candidate leaf identity, and digests in the same transaction that
  commits their effect.
- Existing-candidate correction import, Draft creation, hybrid reconciliation,
  validation-claim append, mapping `ACCEPT`, technical approval, publication,
  and package materialization also acquire the existing-candidate root lock
  whenever that lineage exists and re-read its current leaf. The global order is
  existing-candidate root, governed Draft root, then source identities sorted
  bytewise. A correction/effect race therefore commits either the correction and
  a typed stale blocker or the earlier effect against the then-current
  candidate—never both from an unlocked stale read.
- Previously published versions, technical/publication decisions, in-progress
  Audits, pinned question snapshots, responses, findings, and Audit bytes stay
  unchanged and readable through their historical scope.

## Safe Archive And PDF Handling Contract

### Fixed intake limits (`AGA_ZIP_PDF_V1`)

- Exactly one ZIP per request; MIME and magic must both identify ZIP.
- Maximum archive bytes: 50 MiB.
- Maximum central-directory records: 128, including explicit directory rows.
- Maximum normalized path depth: 4; maximum normalized path length: 512 UTF-8
  bytes; maximum filename component: 255 UTF-8 bytes.
- Maximum compressed or uncompressed bytes per PDF: 20 MiB.
- Maximum total uncompressed bytes: 100 MiB.
- Maximum per-file and whole-archive expansion ratio: 20:1.
- Maximum 250 PDF pages per file and 2,000 PDF pages per batch.
- Maximum extracted UTF-8 text: 20 MiB per file and 100 MiB per batch.
- Maximum canonical structured parser-output object: 25 MiB per PDF, including
  all JSON structure and text; output is streamed/hash-counted rather than
  trusted from declared length.
- Maximum `AGA_EXTRACTION_REVIEW_V1` proposed question spans: 1,000 per PDF;
  each span must be non-empty and wholly inside the bounded parsed text/page
  coordinates. Maximum original text per proposal is 64 KiB and maximum
  aggregate proposal text is 8 MiB per packet.
- Maximum resulting candidate question seeds: 2,000 per decision set. Maximum
  `TRANSCRIBE` replacement is 32 KiB and aggregate transcribed text is 512 KiB.
- Multipart receipt JSON, identity-resolution JSON, and extraction-preparation
  JSON are each at most 64 KiB. Candidate-import JSON and every serialized
  extraction-review response page are each at most 1 MiB. These byte limits use
  UTF-8 serialized bytes, preserve the existing 1 MiB canonical JSON guard, and
  are enforced in addition to item counts; no task may raise the global guard.
- Maximum parser wall time: 30 seconds per PDF and 5 minutes per batch.
- Scanner deadlines: 5 seconds to connect, 120 seconds for the archive call,
  60 seconds for each exact PDF call, and 10 minutes cumulative scan wall time
  per batch. Preserve the existing maximum signature age of 48 hours and reject
  a future-dated signature timestamp. The pinned clamd configuration is
  `StreamMaxLength=52428800`,
  `MaxFileSize=52428800`, `MaxScanSize=115343360`, `MaxRecursion=4`, and
  `MaxFiles=128`; a daemon limit/warning is a hard failure, never CLEAN.
- Parser concurrency: one file for `AGA_ZIP_PDF_V1`. Any increase requires a
  new policy/manifest version after separate load and cleanup evidence.
- Allowed file entries: PDF only. Directories may organize paths but are never
  persisted as candidate files. Reject nested archives, executable content,
  symlinks, devices, multi-disk ZIP, ZIP64, encrypted entries, duplicate
  normalized paths, encrypted/password-protected PDFs, and MIME/magic mismatch.
- Classify a directory only when the central external attributes and the raw
  local/central filename bytes consistently identify a directory and both names
  end in exactly one `/`; inconsistent attributes or additional trailing
  slashes fail. After classification, `NormalizeZipPathV1` removes that one
  directory slash, decodes valid UTF-8, rejects C0/C1 controls, DEL, bidi
  controls, NUL, backslash, absolute/drive/UNC prefixes, empty/`.`/`..`
  components, normalizes each remaining component to Unicode NFC, and compares
  collision keys using NFC plus Unicode simple case folding. The persisted
  display path retains the safe NFC spelling. Directory rows participate in
  duplicate and file/directory-prefix collision checks; no file may also be a
  directory or be nested below a path classified as a file. A directory record
  must use `STORE`, zero compressed and uncompressed sizes, zero CRC32, no data
  descriptor, and no payload byte range; any data-bearing or deflated directory
  is `ZIP_STRUCTURE_MISMATCH` or `UNSUPPORTED_ENTRY_TYPE`.
- Only ZIP methods `STORE` and `DEFLATE` are supported. Local headers and the
  central directory must first agree byte-for-byte on raw filename bytes, then
  agree on flags, method, CRC32, and compressed/uncompressed sizes after safe
  parsing; normalized equality alone cannot excuse a raw-name mismatch. Only
  the UTF-8 and standard data-descriptor flags are accepted. A standard
  non-ZIP64 descriptor is accepted only when its CRC32 and sizes exactly match
  the central record; every other flag/descriptor fails. Entry byte ranges
  must be in-bounds, non-overlapping, and end before the central directory;
  trailing executable bytes and malformed end records fail.

### Path and byte validation order

1. Stream the request to a server-owned mode-0700 temporary directory while
   computing SHA-256 and enforcing 50 MiB; never trust `Content-Length` alone.
2. Compare optional client expected hash with observed hash before storage.
3. Parse the ZIP central directory and corresponding local headers without
   expanding content. Apply `NormalizeZipPathV1`, structural equality/range
   checks, symlink/device mode checks, supported-method checks, and every fixed
   count/size/ratio/encryption/ZIP64/multi-disk limit.
4. Persist an upload intent before object-store PUT. Write the immutable
   quarantine object under a unique server-generated intent/batch-scoped key;
   record the content hash separately and do not deduplicate physical objects
   across batches in Phase 1. The original conditional PUT atomically carries
   service, intent, batch, archive-hash, and policy tags plus the object-store
   checksum request; no later tag call establishes ownership. Never use the
   uploaded filename as an object key or filesystem path. Enable bucket
   versioning, HEAD-verify the exact version, byte count, tags, and a trusted
   object-store SHA-256 checksum before transactional finalization; if the
   backend cannot return a trusted checksum, reopen that exact version and
   recompute SHA-256. An orphan reconciler may inspect/delete only the exact
   version owned by an expired intent and never bytes referenced by another
   receipt.

   If PUT is accepted but its response/version ID is lost, recovery enumerates
   versions only at the exact persisted intent key. It finalizes only when
   exactly one version has the atomic expected tags, bytes, and trusted checksum
   or exact-version rehash; zero/multiple/mismatched versions are reported and
   preserved for operator resolution, never guessed or broadly deleted.
5. Require a successful ClamAV `CLEAN` receipt for the exact archive bytes
   within the frozen scanner configuration and deadline above.
   Bind the receipt to archive digest/byte count plus scanner engine, signature
   version/time, and scanned-at time. `SCAN_UNAVAILABLE`, stale signatures,
   limits/warnings, timeout, scanner/protocol error, or non-clean result fails
   closed and prevents expansion.
6. Stream each allowed entry without shell extraction, enforcing actual emitted
   byte and expansion-ratio counters while recomputing uncompressed size,
   `sha256:` digest, and CRC32. Recheck local/central facts and PDF magic. Then
   INSTREAM-scan the exact decompressed PDF bytes and persist a digest-bound
   per-file CLEAN receipt before parsing; archive CLEAN alone is insufficient.
7. Pass one CLEAN PDF at a time by fixed argument array to the separately named
   one-shot `checklist-pdf-parser` sandbox, never directly from the networked
   worker. The parser service uses `network_mode: none`; the worker communicates
   only through an AF_UNIX socket on a task-owned control volume. The parser has
   no Docker socket, TCP listener, database/object-store/ClamAV credentials, or
   other service secrets. It rehashes the bounded read-only input, starts one
   fixed-argument Poppler subprocess per request, and returns only bounded
   structured output with input/output digests and Poppler version. The sandbox
   has non-root UID, read-only root/input, private size-bounded tmpfs/output,
   dropped capabilities, `no-new-privileges`, a named custom seccomp profile
   whose `socket` rules allow only `AF_UNIX` and reject every other family,
   including `AF_INET`, `AF_INET6`, `AF_NETLINK`, and `AF_PACKET`, bounded
   memory/CPU/PIDs/FDs/file
   size, and command timeout. It detects encrypted PDFs
   non-interactively before text extraction. If namespace, UID, socket
   ownership, limits, seccomp, input digest, or output bound cannot be verified,
   parsing fails closed and partial output is discarded.
   After validating the bounded canonical structured result, the worker commits
   a `PARSER_OUTPUT` intent and conditionally writes it to a unique private
   object key with the same atomic ownership/checksum and exact-version recovery
   protocol as the archive. Only exact-version verification permits the
   `PDF_PARSE` success event/receipt; scratch cleanup never deletes this durable
   receipt artifact.
8. Append a fenced attempt row before every deterministic internal validation,
   archive-object finalization, scan, parse, identity, and cleanup phase, and append
   exactly one terminal event for that attempt afterward; attempt events commit
   independently and never carry the batch manifest. External execution is at-
   least-once across crashes. An expired leased attempt receives one `ABANDONED`
   event and a linked retry rather than an overwrite; each accepted phase/file
   result later cites its exact winning terminal attempt event.
9. After all safe PDFs are parsed, recognize exactly one register using the
   server-owned `AGA_REGISTER_V1` normalized header/table-schema contract, parse
   immutable rows, then identity-match only non-register PDFs in archive order.
   A missing/duplicate/ambiguous register or incomplete register parse fails
   closed. No identity-match attempt can start before the successful register-
   parse receipt exists.
10. Run the fenced `SCRATCH_CLEANUP` phase unconditionally on both the success
    and failure paths before terminal finalization. Delete only task-owned
    temporary files carrying the signed batch marker. A cleanup failure records
    `TEMP_CLEANUP_FAILED`, raises an operational alert, and leaves the batch
    nonterminal/ineligible until its retry chain succeeds or is exhausted. On
    worker startup, resume that same fenced cleanup chain for a signed marker and
    expired lease; never delete an arbitrary directory.
11. Only after cleanup closes, exactly one compare-and-swap batch-finalization
    transaction selects every required winning phase event, writes accepted per-
    file/register/identity facts, computes and stores the canonical batch
    manifest digest, batch intake-safety eligibility, and each file's immutable
    initial candidate-import state, and commits the terminal batch status.
    Competing finalizers conflict or replay the same manifest. A hard
    validation/scan/parse/register/match failure or exhausted cleanup failure
    makes the batch `INVENTORY_FAILED`; no file in that batch is candidate-
    eligible. A fully parsed identity conflict may end `INVENTORY_COMPLETE` with
    `intakeSafetyEligible=true` and file state
    `REQUIRES_IDENTITY_RESOLUTION`; its derived effective eligibility becomes
    true only while an exact current resolution leaf exists, without changing
    the manifest.

### Explicit receipt error codes

Use stable problem/receipt codes:

`ARCHIVE_HASH_MISMATCH`, `ARCHIVE_TOO_LARGE`, `ENTRY_COUNT_EXCEEDED`,
`UNSAFE_ENTRY_PATH`, `DUPLICATE_NORMALIZED_PATH`, `UNSUPPORTED_ENTRY_TYPE`,
`ZIP_STRUCTURE_MISMATCH`, `UNSUPPORTED_COMPRESSION_METHOD`,
`ENCRYPTED_ENTRY`, `ZIP64_UNSUPPORTED`, `ENTRY_SIZE_EXCEEDED`,
`TOTAL_SIZE_EXCEEDED`, `EXPANSION_RATIO_EXCEEDED`, `CRC_MISMATCH`,
`MAGIC_MISMATCH`, `MALWARE_DETECTED`, `SCAN_UNAVAILABLE`,
`SCAN_SIGNATURE_STALE`, `SCAN_LIMIT_EXCEEDED`, `SCAN_TIMEOUT`,
`PDF_ENCRYPTED`, `PDF_PAGE_LIMIT_EXCEEDED`, `PDF_TEXT_LIMIT_EXCEEDED`,
`PDF_PARSE_TIMEOUT`, `PDF_PARSE_FAILED`, `PARSER_OUTPUT_TOO_LARGE`,
`PARSER_OUTPUT_FINALIZATION_FAILED`, `PARSER_OUTPUT_OBJECT_MISMATCH`,
`PARSER_SANDBOX_UNAVAILABLE`,
`REGISTER_PARSE_INCOMPLETE`, `REGISTER_FILE_MISMATCH`, `IDENTITY_CONFLICT`,
`IDENTITY_RESOLUTION_STALE`, `EXTRACTION_PROPOSAL_LIMIT_EXCEEDED`,
`EXTRACTION_PROPOSAL_TEXT_LIMIT_EXCEEDED`, `EXTRACTION_RESPONSE_LIMIT_EXCEEDED`,
`EXTRACTION_REVIEW_STALE`, `CANDIDATE_IMPORT_BODY_TOO_LARGE`,
`OBJECT_VERSION_MISMATCH`, `OBJECT_TAG_MISMATCH`,
`OBJECT_CHECKSUM_MISMATCH`, `OBJECT_FINALIZATION_FAILED`,
`BATCH_FINALIZATION_CONFLICT`, and `TEMP_CLEANUP_FAILED`.

No error may produce a success toast, silent omission, candidate ID, Draft ID,
decision, publication, or Audit effect.

### Idempotency and replay

- Every mutation carries `operationId`, canonical `Idempotency-Key` HTTP
  header, and a canonical semantic request digest. The multipart JSON receipt
  is required as the first part; when it repeats `idempotencyKey`, the handler
  requires exact equality with the header before advancing to the archive part;
  mismatch is `400` with zero archive-content reads or effect.
- First successful archive receipt returns `201`; replay of the same principal,
  operation scope, idempotency key, expected archive hash, and exact bytes
  returns `200` with the original immutable receipt and no new object/parser
  attempts.
- Same idempotency key with different bytes, expected hash, selected file,
  field decisions, or semantic payload returns `409 IDEMPOTENCY_CONFLICT` and
  changes nothing.
- A new key for the same archive hash may create a new receipt only when a new
  reason is supplied. It receives its own intent-scoped immutable object; there
  is no cross-batch physical-object deduplication in Phase 1. Replay of the same
  operation reuses only its original intent, exact object version, and receipt.
- Failed attempts are replay-stable. A retry after parser/policy correction is
  a new operation linked to the failed receipt; failures are never overwritten.
- Candidate import and Draft creation use exact expected file/candidate/draft
  digests and create all rows atomically. Concurrent identical commands return
  the same result; concurrent divergent commands lose with a typed conflict.

## API And OpenAPI Contract

Add these operations to `api/openapi/source/paths/platform.json`; define strict
schemas in `api/openapi/source/schemas/platform.json`; regenerate
`api/openapi/aviasurveil360.yaml`,
`apps/api/internal/httpapi/generated/api.gen.go`, and
`apps/web/src/generated/transport/api-types.ts` with
`./scripts/generate-contracts.sh`, then prove zero drift with
`./scripts/check-contracts.sh`.

### Admin-only inventory operations

| Method and path | Operation ID | Result |
| --- | --- | --- |
| `POST /v1/admin/governed-checklist/import-batches` | `createAdminChecklistImportBatch` | Multipart request with a first JSON `receipt` part followed by exactly one `archive` part; the receipt contains operation/idempotency/expected hash/reason and is checked against the header before archive streaming; returns immutable batch receipt. |
| `GET /v1/admin/governed-checklist/import-batches/{importBatchId}` | `getAdminChecklistImportBatch` | Batch summary, counts, manifest digest, scan/cleanup state, and blocking issues. |
| `GET /v1/admin/governed-checklist/import-batches/{importBatchId}/files` | `listAdminChecklistImportFiles` | Ordered register/file identities, immutable initial match/import states, derived effective identity/import eligibility and current-resolution summary, and validation/parse states; no raw extracted text download. |
| `GET /v1/admin/governed-checklist/import-batches/{importBatchId}/receipts` | `listAdminChecklistImportReceipts` | Ordered append-only scan/parser/cleanup receipts and typed errors. |
| `POST /v1/admin/governed-checklist/import-batches/{importBatchId}/files/{importFileId}/extraction-reviews` | `createAdminChecklistImportFileExtractionReview` | Idempotently streams the exact private parser-output object and atomically creates one READY/FAILED packet receipt plus all-or-zero immutable proposals; it does not reparse PDF bytes. |
| `GET /v1/admin/governed-checklist/import-batches/{importBatchId}/files/{importFileId}/extraction-review` | `getAdminChecklistImportFileExtractionReview` | Cursor-paginated private packet projection with exact parser/file/manifest/packet digests, competing identities, ordered original text/locators/proposal digests, and strict current decision-set state; it is never source authority. |
| `POST /v1/admin/governed-checklist/import-batches/{importBatchId}/files/{importFileId}/identity-resolutions` | `resolveAdminChecklistImportFileIdentity` | CAS-appends a selected human-readable identity against the exact expected current resolution state and returns the derived file view. |
| `POST /v1/admin/governed-checklist/import-batches/{importBatchId}/files/{importFileId}/candidate-imports` | `createAdminExistingChecklistCandidate` | Atomically appends or explicitly reuses one exact extraction decision set and creates one immutable existing candidate bound to exact current identity/packet facts; identical replay returns the original. |

The multipart endpoint must require the receipt first, must not consume archive
content until its idempotency key matches the header, and must not buffer the
full archive in memory.
`scripts/generate-go-contracts.mjs` must emit the strict trace union and the
typed multipart operation shape instead of `json.RawMessage`/missing body.
The React HTTP backend implements one narrowly scoped multipart builder from
the generated receipt type plus browser `File`; no generated or hand-written
surface accepts an arbitrary client filesystem path.

The extraction-review GET is the sole raw proposed-text intake projection. It
is authenticated and Admin-authorized before file lookup; guessed/cross-batch
IDs return the same non-leaking response and no other role or functional
assignment can call it. Its stable order is `(proposalOrdinal,proposalId)`,
default page size 50, maximum 100 and 1 MiB serialized bytes, with an opaque
server cursor; the server returns fewer than the requested count when required
by the byte bound. Every page
repeats the exact immutable packet/file/manifest/parser digests and exposes
`currentDecisionSet` as a strict `NO_DECISION|DECIDED` union; `DECIDED` requires
root/leaf/revision/digest/current-leaf token, while `NO_DECISION` prohibits them.
The cursor binds the packet digest and decision-state/current-leaf token; a
changed leaf makes the old cursor stale and requires a fresh first page.
Original text and locators are excluded from logs, list/count/search/
notification projections, source-owner/reviewer/manager routes, and every
Auditee response. HTTP responses use `Cache-Control: no-store`; the HTTP React
path keeps the packet only in component memory, clears it on route/session/
principal change, and never writes it to service-worker caches, localStorage,
IndexedDB, analytics, or error telemetry. Mock mode uses synthetic fixtures only
and never seeds real AGA text.

### Capability-scoped authoring and review operations

| Method and path | Operation ID | Result |
| --- | --- | --- |
| `GET /v1/governed-checklist/source-review-queue` | `listGovernedChecklistSourceReviewQueue` | Deterministically ordered, cursor-paginated source-authority and candidate-mapping work in the current source-owner assignment scope; excludes Admin inventory and unrelated candidates. |
| `GET /v1/governed-checklist/source-review-items/{reviewItemId}` | `getGovernedChecklistSourceReviewItem` | Complete assignment-scoped source/version/chain/currentness/attestation detail or candidate mapping/reconciliation/blocker detail; excludes batch/file inventory. |
| `GET /v1/governed-checklist/reviewer-queue` | `listGovernedChecklistReviewerQueue` | Deterministically ordered, cursor-paginated Draft/reconciliation/blocker work in the current reviewer assignment scope. |
| `POST /v1/governed-checklist/source-versions/{sourceVersionId}/authority-attestations` | `attestRegulatorySourceAuthority` | Appends a source-owner accept/return decision for an exact source/version/hash/currentness/chain-role fact; it is not a candidate mapping decision. |
| `GET /v1/governed-checklist/existing-candidates/{existingCandidateId}` | `getExistingChecklistCandidate` | Scoped immutable original question/intent/history projection and lineage. |
| `POST /v1/governed-checklist/existing-candidates/{existingCandidateId}/drafts` | `createDraftFromExistingChecklistCandidate` | New immutable Draft with explicit source gaps and exact department scope. |
| `POST /v1/governed-checklist/official-source-drafts` | `createOfficialSourceChecklistDraft` | New immutable Draft from server-resolved current official source clauses. |
| `GET /v1/governed-checklist/candidates/{candidateId}` | `getGovernedChecklistDraft` | Role-projected immutable Draft, origin, complete reconciliation, current attestation projections, comments, and ordered blockers for an Admin, matching required Department Manager, source owner, or reviewer. |
| `POST /v1/governed-checklist/candidates/{candidateId}/hybrid-reconciliations` | `createHybridReconciledChecklistDraft` | New immutable successor plus server-computed field diffs. |
| `GET /v1/governed-checklist/candidates/{candidateId}/review-comments` | `listGovernedChecklistReviewComments` | Scoped append-only internal comments. |
| `POST /v1/governed-checklist/candidates/{candidateId}/review-comments` | `createGovernedChecklistReviewComment` | Reviewer/authorized internal comment against exact digest. |
| `POST /v1/governed-checklist/candidates/{candidateId}/source-mapping-attestations` | `attestGovernedChecklistSourceMapping` | Complete-`REVIEWED_SOURCE_SET` owner accept/return against exact assignment, full chain, mapping scope, and candidate digest. |
| `POST /v1/governed-checklist/published-versions/{publishedVersionId}/audit-package-eligibility-evaluations` | `evaluateGovernedChecklistAuditPackageEligibility` | Non-mutating internal evaluation for one exact published version and proposed Audit/target/inspection scope; returns computed eligibility and ordered blocker codes. |

Author access at these shared routes is a closed rule: Admin, or a current
Department Manager present in the server-derived required-owner set. A source-
owner assignment adds no author power; a source owner authors only when the same
principal independently satisfies one of those two rules. Source-owner/reviewer
assignments grant only the operations described in the permission matrix. Existing
Department Manager technical-approval/publication routes remain separate and
are strengthened; do not add aliases that combine them.

Both scoped detail reads resolve the actor's current assignment/Department
scope before returning a projection. A guessed, cross-assignment, stale-
assignment, or Auditee identifier returns the same non-leaking `404`; source
owner/reviewer detail views never include Admin batch/file inventory or another
assignment's comments.

Creation/append operations return `201` on the first committed semantic
command and `200` with the original view on an identical replay. Reads and the
non-mutating eligibility evaluation return `200`. Validation returns the
repository-standard `400`, authentication `401`, capability/scope denial
`403`, absent scoped resource `404`, stale/idempotency conflict `409`, bounded
payload breach `413`, unsupported media `415`, and parser/scan dependency
failure as the existing non-leaking `503` problem shape. No error response
contains extracted question text, object keys, or local temporary paths.
The extraction-preparation exception is a committed immutable deterministic
`FAILED` receipt: first call and identical replay both return typed `413` with
the same non-sensitive packet receipt ID/digest and never a success envelope or
toast; a changed semantic request remains `409`.
Every path-keyed request also carries the path identities in its generated
command type. A path/body mismatch returns non-leaking `400` before a service
call and creates no receipt, decision, or audit success event. Cursor lists use
stable `(createdAt,id)` ordering, maximum page size 100, default 50, and an
opaque server-issued cursor.

### Required inputs and outputs

- `CreateChecklistImportExtractionReviewInput` contains both path identities,
  exact terminal-manifest/file/parser-receipt/output hash/byte facts, generator
  policy, reason, and operation/idempotency identities; it never exposes or
  accepts a private object key/version. The server resolves and opens only the
  exact receipt-bound parser-output version, rehashes/counts it while streaming,
  and inserts the packet header plus every proposal in one transaction after
  count/digest validation. A deterministic policy/size failure atomically stores
  one idempotent `FAILED` packet with zero proposals and returns the matching
  typed `413` receipt; crash/DB/internal failure rolls back all packet/proposal
  rows. `READY` replay returns the original summary and no second object read.
- `ResolveChecklistImportFileIdentityInput` contains both path identities,
  expected file hash and terminal-manifest digest, operation/idempotency/reason,
  the selected identity source/value while preserving all competing values, and
  strict `expectedResolutionState`. `NO_RESOLUTION` prohibits a leaf; `CURRENT`
  requires exact root/leaf/revision/digest/current-leaf token. The server locks
  the file, rederives the current leaf, and rejects stale, invented, or partial
  state with zero effect. `REGISTER`, `VISIBLE`, or `PDF_METADATA` selection must
  byte-match its stored value; `HUMAN_TRANSCRIPTION` requires a non-empty new
  value plus separate transcription reason/receipt and remains candidate-only.
  No selection can resolve `NOT_REGISTERED` or create a register/source fact.
- `ChecklistImportFileView` exposes immutable `initialIdentityMatchState`,
  immutable `initialCandidateImportState`, derived `effectiveIdentityStatus`/
  `effectiveCandidateImportEligible`, and strict `currentResolution` union.
  `NO_RESOLUTION` prohibits resolution fields; `CURRENT` requires root/leaf/
  revision/digest/actor/time/current-leaf token. Neither variant rewrites the
  terminal file or manifest facts. The effective boolean covers only terminal
  safety plus identity readiness; it never claims that extraction decisions or
  candidate creation already exist.
- `ChecklistImportExtractionReviewPage` repeats exact batch/file/manifest/
  parser/packet identities and digests, competing identity values, the strict
  current decision-set union, and a stable bounded page of original proposed
  text/span/locator/provenance facts. It contains no regulatory trace, source-
  authority, validation, approval, or publication claim.
- `CreateExistingChecklistCandidateInput` contains both path identities, exact
  file/hash/manifest/packet digests, exact expected identity-resolution state,
  reason/idempotency facts, strict `candidateLineageAction`, and strict
  `extractionDecisionAction`. `INITIAL` prohibits candidate root/leaf fields and
  requires that the file has no candidate root; `CORRECT_CURRENT` requires exact
  root/current-leaf/revision/content-digest/token and creates revision+1.
  `APPEND` requires exact expected `NO_DECISION|DECIDED` state and an ordered
  complete accept/split/merge/transcribe/exclude decision array;
  `REUSE_CURRENT` requires the exact current root/leaf/revision/digest/token and
  prohibits decision fields. First import must append; an identity-only
  correction may reuse; a boundary correction must append a successor. The
  server locks and rereads the identity, extraction-decision, and candidate
  current leaves as applicable, verifies complete non-overlapping packet
  coverage, then atomically appends/reuses the decision set and creates
  candidate/questions. Missing, stale, partial, extra, overlapping, or cross-
  packet facts create neither artifact.
- `CreateOfficialSourceChecklistDraftInput` contains exact
  provider/target/inspection scope, template identity, change reason, and an
  ordered array of question proposals. It carries no trusted Department owner.
  Each proposal names ordered persisted source-clause IDs; the server resolves
  and validates the complete authority-accepted source chain, derives source
  title/version/hash/currentness/locator facts and required owners, and rejects
  client disagreement. A missing/partial/unapproved/stale chain fails the whole
  command; this path never creates a source-gap Draft or generation run. The
  immutable source-binding set and command/input/output digests are the direct
  authoring provenance.
- `CreateDraftFromExistingChecklistCandidateInput` contains exact candidate
  root/current-leaf/revision/digest/token, provider/target/inspection scope,
  selected question IDs, initial field dispositions, reason, and idempotency
  facts. It cannot claim resolved traces or trusted Department ownership. The
  server locks the candidate root and denies a superseded/stale snapshot. It
  persists the exact owner-resolution inputs/result/digest; missing or ambiguous
  owners remain visible blockers.
- `CreateHybridReconciledChecklistDraftInput` contains expected predecessor
  revision/digest, exact proposed provider/target/inspection scope (or an
  explicit `inheritScope=true`), and ordered current question proposals linked
  to original question IDs and ordered persisted source clauses. The server
  resolves the authority chain, rederives the complete owner set for changed
  scope, and computes every diff; a client cannot carry predecessor owners into
  a changed scope.
- `GovernedQuestionView.regulatoryTrace` becomes the strict discriminated
  content-plus-projection view defined above.
  `GovernedQuestionReconciliationView` carries wording, verification
  objective/method, expected Evidence, applicability/rationale, scope
  classification/rationale/signals/guardrails, complete typed authority chain,
  split/merge/exclusion identities, original operational intent, and result-
  history disposition with exact before/after/digests/outcome/reason.
- `GovernedCandidateView.lineage` becomes a strict `oneOf` discriminated by
  `lineageType`: `PRE_V28_GENERATION_RUN`, `GENERATION_RUN`,
  `EXISTING_CANDIDATE`, or `DIRECT_OFFICIAL_SOURCE`. Every variant requires
  `entryPath`, `lineageKind`, candidate root, and the predecessor ID/digest pair
  exactly when non-initial. `PRE_V28_GENERATION_RUN` requires non-null
  `generationRunId`, null existing-candidate/binding IDs,
  `legacyAuthorityState=PRE_V28_UNATTESTED`, and the frozen legacy
  `sourceSnapshots`, `scopeFactIds`, and `crosswalkPartitionIds` projections;
  this variant is prohibited for a row created by version 28. `GENERATION_RUN`
  requires a non-null generation-run ID; every successor must equal the
  generation-run ID stored by both its predecessor and root. It requires null
  existing-candidate ID, no legacy state, its exact legacy arrays, and the
  binding ID/digest when a question is resolved.
  `EXISTING_CANDIDATE` requires non-null
  `existingCandidateId`, null generation run, no legacy arrays/state, and a
  nullable binding pair only for an all-gap Draft. `DIRECT_OFFICIAL_SOURCE`
  requires null generation/existing IDs, no legacy arrays/state, and a non-null
  binding pair. `SOURCE_IMPACT_REVIEW` uses its root's modern lineage family and
  the truth-table binding rule, except that any new row after a migrated
  `PRE_V28_GENERATION_RUN` leaf is `GENERATION_RUN`, never PRE_V28.
  `additionalProperties=false` and per-variant
  prohibited fields reject every other combination. Historical non-governed
  rows remain on their existing legacy view and never masquerade as governed.
- `GovernedSourceReviewDetailView` is a separate strict `oneOf` keyed by
  `reviewItemKind`. `SOURCE_AUTHORITY` requires review item/source/version/hash/
  `sourceClass`/chain-role/currentness IDs, the authorized assignment ID, and
  the `currentDecision` union below; it prohibits candidate, reconciliation,
  comment, and import fields. `CANDIDATE_MAPPING` requires
  review item/candidate/revision/content and chain digests, binding-set ID when
  present, exact reviewed-source-set ID/version/digest, provider/target/
  inspection/Department/unit scope, role-safe candidate provenance receipt IDs,
  complete reconciliation, ordered blockers, and the same `currentDecision`
  union; it prohibits batch/file/object/parser inventory and source-authority-
  only fields. `currentDecision` is itself a strict `oneOf` discriminated by
  `decisionState`: `NO_DECISION` requires only that discriminant and prohibits
  root ID, leaf ID, outcome, subject digest, semantic digest, and leaf token;
  `DECIDED` requires all of `decisionRootId`, `decisionLeafId`,
  `outcome=ACCEPT|RETURN`, `decisionSubjectDigest`, `leafSemanticDigest`, and
  `currentLeafToken`. The server verifies that root and leaf belong to the same
  subject-bound chain and that the leaf/token are current. Both the outer and
  nested variants use `additionalProperties=false`; no partial decision tuple,
  nullable placeholder, or stale/non-leaf token can be projected or accepted
  by a subsequent command.
- Every view contains stable identifiers and exact version/digest fields.
  List responses are ordered deterministically and paginated with bounded page
  size. Unknown/extra properties fail schema validation.
- Problem responses distinguish authentication, capability, department scope,
  stale digest, mapping gap, source stale, invalid applicability, missing
  owner, technical review absent, publication absent, package ineligible,
  parse failure, and idempotency conflict without leaking private content.

## Go Domain, Application, Worker, And Persistence Boundaries

Create a focused `apps/api/internal/checklistintake/` package rather than
adding archive/parsing concerns to the already large regulatory files.

### File responsibilities and interfaces

- `apps/api/internal/checklistintake/types.go`
  defines batch/file/receipt/candidate enums and immutable command/view types.
- `apps/api/internal/checklistintake/policy.go`
  defines `Policy`, `NormalizeZipPathV1`, `ValidateEntryPath`, local/central
  structural equality, and `ValidateCentralDirectory` for exact
  `AGA_ZIP_PDF_V1` limits.
- `apps/api/internal/checklistintake/archive.go`
  defines streaming `ArchiveInspector.Inspect(ctx, io.ReaderAt, size)` and
  returns ordered immutable entry facts without filesystem extraction.
- `apps/api/internal/checklistintake/register.go`
  recognizes/parses `AGA_REGISTER_V1` and produces immutable register/match
  facts only after all safe PDF parse receipts exist.
- `apps/api/internal/checklistintake/pdf_parser.go`
  defines `PDFParser.Parse(ctx, PDFParseRequest) (PDFParseResult, error)` and a
  fail-closed `ParserSandbox` adapter; the networked worker never starts
  Poppler directly and the test adapter is deterministic.
- `apps/api/internal/checklistintake/authorization.go`
  evaluates Admin receipt/import and scoped internal read capabilities.
- `apps/api/internal/checklistintake/service.go`
  exposes `ReceiveBatch`, `CreateExtractionReview`, `GetExtractionReview`, `ResolveIdentity`,
  `CreateExistingCandidate`, and read projections, coordinating object store,
  scanner, parser, canonical digests, transactions, audit events, and cleanup.
- `apps/api/internal/checklistintake/store.go`
  defines transaction/store interfaces and PostgreSQL error mapping.
- `apps/api/internal/checklistintake/postgres_store.go`
  implements version-28 persistence with no in-memory authority fallback.
- `apps/api/internal/worker/checklistintake/worker.go`
  claims pending batches, appends fenced attempts for every fixed-DAG phase
  under bounded leases, appends one terminal event per attempt, runs the single
  CAS batch finalizer, and reconciles only signed owned scratch/upload intents
  on startup.
- `apps/api/internal/platform/objectstore/object_store.go` and
  `minio_store.go` expose an exact-version conditional put whose request
  atomically includes ownership tags/checksum fields, plus stat/open/scoped-
  list/delete operations, trusted checksum or exact-version rehash, and bucket
  versioning. Intake has no post-PUT ownership-tag path and no caller can resolve
  mutable latest bytes for a receipt.
- `apps/api/internal/platform/scanner/clamav.go` and
  `deploy/local/clamav/entrypoint.sh` bind scanner calls and daemon capacity to
  the frozen `AGA_ZIP_PDF_V1` deadlines and byte/file/recursion limits.
- `apps/api/internal/regulatory/draft_authoring.go`
  creates official-source and existing-candidate Drafts through the existing
  candidate persistence boundary.
- `apps/api/internal/regulatory/reconciliation.go`
  computes immutable before/after diffs and hybrid lineage.
- `apps/api/internal/regulatory/source_attestation.go`
  validates functional assignment/source/candidate scope, resolves immutable
  decision roots/current leaves, and separately appends per-link source-
  authority and complete-source-set candidate-mapping decisions.
- `apps/api/internal/regulatory/required_owners.go`
  derives immutable required owners from current reviewed provider/target/
  inspection/source responsibility and returns explicit ambiguity blockers.
- `apps/api/internal/identity/checklist_functional_authority.go`
  resolves the globally latest effective assignment successor before subject/
  scope filtering, with no Department Manager membership fallback.
- `apps/api/internal/checklistgovernance/eligibility.go`
  centralizes publication and Audit-package eligibility blocker evaluation so
  mock, HTTP, and package composition use the same ordered codes.
- `apps/api/internal/httpapi/governed_checklist_intake_api.go`
  handles typed multipart receipt and inventory routes.
- `apps/api/internal/httpapi/governed_checklist_authoring_api.go`
  handles capability-scoped Draft/reconciliation/comment/attestation routes.
- `apps/api/cmd/checklist-pdf-parser/main.go` is the only production Poppler
  entrypoint and accepts content-hash-bound input/output paths supplied by the
  sandbox boundary.
- `apps/api/cmd/worker/main.go`, `apps/api/internal/platform/config/config.go`,
  `apps/api/Dockerfile`, `deploy/local/compose.yaml`,
  `deploy/local/compose.test.yaml`, and
  `deploy/local/checklist-pdf-parser-seccomp.json` wire the networked candidate-
  only worker, separate no-network parser image/service, AF_UNIX-only socket
  seccomp policy, exact Poppler version, private versioned buckets, ClamAV, and
  resource/security limits.

The handler must authenticate before reading a body where practical, apply a
bounded request reader, stream bytes once, and never log filenames, extracted
text, question wording, or uploaded bytes. Audit events record identities,
hashes, outcomes, actor/scope, and blocker codes only.

### Canonical Go and TypeScript service signatures

Use these layer-specific names consistently in domain, handlers, generated
adapters, mock, HTTP backend, and tests. The mapping after the signatures is
normative; an adapter may translate names but never field optionality, union
variants, envelope shape, or semantics:

```go
type CommandResult[T any] struct {
    View     T
    Replayed bool
}

type PageInput struct {
    Cursor string
    Limit  int
}

func (service *Service) ReceiveBatch(
    ctx context.Context,
    actor identity.Principal,
    command ReceiveBatchCommand,
    archive io.Reader,
) (CommandResult[ImportBatchView], error)

func (service *Service) GetBatch(
    ctx context.Context,
    actor identity.Principal,
    importBatchID string,
) (ImportBatchView, error)

func (service *Service) ListFiles(
    ctx context.Context,
    actor identity.Principal,
    importBatchID string,
    page PageInput,
) (ImportFilePage, error)

func (service *Service) ListReceipts(
    ctx context.Context,
    actor identity.Principal,
    importBatchID string,
    page PageInput,
) (ImportReceiptPage, error)

func (service *Service) CreateExtractionReview(
    ctx context.Context,
    actor identity.Principal,
    command CreateExtractionReviewCommand,
) (CommandResult[ExtractionReviewSummaryView], error)

func (service *Service) GetExtractionReview(
    ctx context.Context,
    actor identity.Principal,
    importBatchID string,
    importFileID string,
    page PageInput,
) (ExtractionReviewPage, error)

func (processor *Processor) ProcessBatch(
    ctx context.Context,
    importBatchID string,
) (ImportBatchView, error)

func (service *Service) ResolveIdentity(
    ctx context.Context,
    actor identity.Principal,
    command ResolveIdentityCommand,
) (CommandResult[ImportFileView], error)

func (service *Service) CreateExistingCandidate(
    ctx context.Context,
    actor identity.Principal,
    command CreateExistingCandidateCommand,
) (CommandResult[ExistingCandidateView], error)

func (service *AuthoringService) ListSourceReviewQueue(
    ctx context.Context,
    actor identity.Principal,
    page PageInput,
) (SourceReviewQueuePage, error)

func (service *AuthoringService) GetSourceReviewItem(
    ctx context.Context,
    actor identity.Principal,
    reviewItemID string,
) (GovernedSourceReviewDetailView, error)

func (service *AuthoringService) ListReviewerQueue(
    ctx context.Context,
    actor identity.Principal,
    page PageInput,
) (ReviewerQueuePage, error)

func (service *AuthoringService) GetExistingCandidate(
    ctx context.Context,
    actor identity.Principal,
    existingCandidateID string,
) (ExistingCandidateView, error)

func (service *AuthoringService) GetGovernedDraft(
    ctx context.Context,
    actor identity.Principal,
    candidateID string,
) (GovernedCandidateDetailView, error)

func (service *AuthoringService) AttestSourceAuthority(
    ctx context.Context,
    actor identity.Principal,
    command SourceAuthorityAttestationCommand,
) (CommandResult[SourceAuthorityAttestationView], error)

func (service *AuthoringService) CreateDraftFromExistingCandidate(
    ctx context.Context,
    actor identity.Principal,
    command CreateDraftFromExistingCandidateCommand,
) (CommandResult[CandidateView], error)

func (service *AuthoringService) CreateOfficialSourceDraft(
    ctx context.Context,
    actor identity.Principal,
    command CreateOfficialSourceDraftCommand,
) (CommandResult[CandidateView], error)

func (service *AuthoringService) CreateHybridReconciliation(
    ctx context.Context,
    actor identity.Principal,
    command CreateHybridReconciliationCommand,
) (CommandResult[CandidateView], error)

func (service *AuthoringService) ListReviewComments(
    ctx context.Context,
    actor identity.Principal,
    candidateID string,
    page PageInput,
) (ReviewCommentPage, error)

func (service *AuthoringService) CreateReviewComment(
    ctx context.Context,
    actor identity.Principal,
    command CreateReviewCommentCommand,
) (CommandResult[ReviewCommentView], error)

func (service *AuthoringService) AttestSourceMapping(
    ctx context.Context,
    actor identity.Principal,
    command SourceMappingAttestationCommand,
) (CommandResult[SourceMappingAttestationView], error)

func (service *EligibilityService) EvaluateAuditPackage(
    ctx context.Context,
    actor identity.Principal,
    input AuditPackageEligibilityInput,
) (AuditPackageEligibilityView, error)
```

```ts
export interface GovernedBackendCommandResult<T> {
  view: T;
  replayed: boolean;
}

export interface GovernedChecklistIntakeBackend {
  receiveBatch(
    input: CreateChecklistImportBatchReceiptInput,
    archive: File,
    options?: BackendRequestOptions,
  ): Promise<GovernedBackendCommandResult<ChecklistImportBatchView>>;
  getBatch(input: { importBatchId: string }, options?: BackendRequestOptions): Promise<ChecklistImportBatchView>;
  listFiles(input: { importBatchId: string; cursor?: string; limit?: number }, options?: BackendRequestOptions): Promise<ChecklistImportFilePage>;
  listReceipts(input: { importBatchId: string; cursor?: string; limit?: number }, options?: BackendRequestOptions): Promise<ChecklistImportReceiptPage>;
  createExtractionReview(input: CreateChecklistImportExtractionReviewInput, options?: BackendRequestOptions): Promise<GovernedBackendCommandResult<ChecklistImportExtractionReviewSummaryView>>;
  getExtractionReview(input: { importBatchId: string; importFileId: string; cursor?: string; limit?: number }, options?: BackendRequestOptions): Promise<ChecklistImportExtractionReviewPage>;
  resolveIdentity(input: ResolveChecklistImportFileIdentityInput, options?: BackendRequestOptions): Promise<GovernedBackendCommandResult<ChecklistImportFileView>>;
  createExistingCandidate(input: CreateExistingChecklistCandidateInput, options?: BackendRequestOptions): Promise<GovernedBackendCommandResult<ExistingChecklistCandidateView>>;
}

export interface GovernedChecklistAuthoringBackend {
  listSourceReviewQueue(input: { cursor?: string; limit?: number }, options?: BackendRequestOptions): Promise<GovernedSourceReviewQueuePage>;
  getSourceReviewItem(input: { reviewItemId: string }, options?: BackendRequestOptions): Promise<GovernedSourceReviewDetailView>;
  listReviewerQueue(input: { cursor?: string; limit?: number }, options?: BackendRequestOptions): Promise<GovernedReviewerQueuePage>;
  getExistingCandidate(input: { existingCandidateId: string }, options?: BackendRequestOptions): Promise<ExistingChecklistCandidateView>;
  getGovernedDraft(input: { candidateId: string }, options?: BackendRequestOptions): Promise<GovernedCandidateDetailView>;
  attestSourceAuthority(input: AttestRegulatorySourceAuthorityInput, options?: BackendRequestOptions): Promise<GovernedBackendCommandResult<RegulatorySourceAuthorityAttestationView>>;
  createDraftFromExisting(input: CreateDraftFromExistingChecklistCandidateInput, options?: BackendRequestOptions): Promise<GovernedBackendCommandResult<GovernedCandidateView>>;
  createOfficialSourceDraft(input: CreateOfficialSourceChecklistDraftInput, options?: BackendRequestOptions): Promise<GovernedBackendCommandResult<GovernedCandidateView>>;
  createHybridReconciliation(input: CreateHybridReconciledChecklistDraftInput, options?: BackendRequestOptions): Promise<GovernedBackendCommandResult<GovernedCandidateView>>;
  listReviewComments(input: { candidateId: string; cursor?: string; limit?: number }, options?: BackendRequestOptions): Promise<GovernedChecklistReviewCommentPage>;
  createReviewComment(input: CreateGovernedChecklistReviewCommentInput, options?: BackendRequestOptions): Promise<GovernedBackendCommandResult<GovernedChecklistReviewCommentView>>;
  attestSourceMapping(input: AttestGovernedChecklistSourceMappingInput, options?: BackendRequestOptions): Promise<GovernedBackendCommandResult<GovernedSourceMappingAttestationView>>;
  evaluateAuditPackageEligibility(input: EvaluateGovernedChecklistAuditPackageEligibilityInput, options?: BackendRequestOptions): Promise<GovernedAuditPackageEligibilityView>;
}
```

The exact Go-domain → OpenAPI/generated TypeScript view mapping is:
`ImportBatchView` → `ChecklistImportBatchView`, `ImportFilePage` →
`ChecklistImportFilePage`, `ImportReceiptPage` →
`ChecklistImportReceiptPage`, `ExtractionReviewSummaryView` →
`ChecklistImportExtractionReviewSummaryView`, `ExtractionReviewPage` →
`ChecklistImportExtractionReviewPage`, `ImportFileView` →
`ChecklistImportFileView`, `ExistingCandidateView` →
`ExistingChecklistCandidateView`, `SourceReviewQueuePage` →
`GovernedSourceReviewQueuePage`, `ReviewerQueuePage` →
`GovernedReviewerQueuePage`, `CandidateView` → `GovernedCandidateView`,
`SourceAuthorityAttestationView` →
`RegulatorySourceAuthorityAttestationView`, `ReviewCommentPage` →
`GovernedChecklistReviewCommentPage`, `ReviewCommentView` →
`GovernedChecklistReviewCommentView`, `SourceMappingAttestationView` →
`GovernedSourceMappingAttestationView`, and `AuditPackageEligibilityView` →
`GovernedAuditPackageEligibilityView`. Identically named detail views map
one-to-one. Task 1 schema/generator tests freeze each OpenAPI/generated Go/
generated TypeScript view shape and reject any extra, missing, nullable, or
cross-variant field. As each mapped domain type becomes available, its owning
Tasks 3–7 test generated request → domain-command conversion, domain view →
generated response conversion, and generated JSON encode/decode round trips.
Task 7 runs the consolidated mapping suite for every row above; Task 8 compares
the complete mapped mock and HTTP payloads.

`CreateChecklistImportBatchReceiptInput` is the JSON multipart part; it does
not contain file bytes or a client filesystem path. The OpenAPI generated
types are authoritative for field optionality; the backend interfaces may
alias them but must not redefine a looser shape. Every Go create/append service
returns `CommandResult[T]`; the HTTP handler uses `Replayed` to select `201` or
`200` without a second lookup. The HTTP backend converts that status into
`GovernedBackendCommandResult<T>`, and the mock returns the identical envelope,
so React can render first-commit versus replay without transport-specific
state. Reads and eligibility evaluation never return a replay flag.

## React, Mock, HTTP, And Auditee Projections

Keep `apps/web/src/features/admin/checklist-builder-page.tsx` as the route
orchestrator and split new responsibilities into focused components:

- `apps/web/src/features/admin/checklist-intake-panel.tsx`: archive receipt action, policy summary, batch
  state, register/file table, hashes, parser/identity/errors, replay status;
- `apps/web/src/features/admin/checklist-candidate-review.tsx`: exact original wording, page/row locator,
  operational intent, result-history provenance, question-boundary decisions,
  and explicit candidate creation;
- `apps/web/src/features/admin/checklist-draft-editor.tsx`: distinct “Existing checklist candidate” and
  “Approved official sources” entry paths, immutable revision controls, and
  source-gap editing;
- `apps/web/src/features/admin/checklist-reconciliation-diff.tsx`: side-by-side before/after values for
  wording, verification objective/method, Evidence, applicability/rationale,
  scope/rationale/signals/guardrails, complete authority chain,
  split/merge/exclusion, operational intent, and history disposition with exact
  digests/outcomes/reasons;
- `apps/web/src/features/admin/checklist-publication-blockers.tsx`: ordered blockers and disabled-action
  explanation shared by Admin and Department Manager projections.
- `apps/web/src/features/checklists/source-review-queue.tsx`: assignment-scoped authority/mapping queue with no
  Admin file inventory and explicit empty/expired/revoked states;
- `apps/web/src/features/checklists/checklist-reviewer-queue.tsx`: assignment-scoped Draft/reconciliation queue
  whose recommendations remain visibly advisory.

UI requirements:

- Always show a candidate-only banner and origin badge:
  `EXISTING_CHECKLIST_CANDIDATE`, `REGULATORY_TRACE`, or
  `HYBRID_RECONCILED`.
- Inventory cards show archive/file SHA-256, byte/page counts, register match,
  validation/scan/parse/identity state, parser version, and cleanup result.
- An identity conflict shows every competing title. Resolving the display
  identity never hides or overwrites the conflict.
- Every question shows scope classification/rationale/guardrails, source
  currentness, every typed source-chain link/role/authority decision with exact
  identity/version/hash and locator/page/section/clause,
  applicability/rationale, verification objective/method, expected Evidence,
  separately projected mapping/technical decisions, reconciliation diff, and
  all blocking reasons.
- `SOURCE_MAPPING_REQUIRED` is a prominent status with named missing fields.
  It never renders as an empty citation area or a visually complete question.
- Technical approval, publication, and Audit eligibility use separate status
  sections. Buttons are never combined.
- A visible action must work, navigate, create an immutable artifact, or be
  disabled with a specific server-derived reason. No toast-only success or
  fake local-only approval.
- Preserve keyboard order, labelled inputs, table/list semantics, focus
  restoration, error summary links, 44px touch targets, 390px no-overflow, and
  desktop/tablet/mobile layouts. Hashes and locators wrap without horizontal
  page overflow.

Transport and parity work:

- Extend `apps/web/src/backend/backend.ts` with focused intake/authoring
  interfaces; do not expose raw file paths or mutate the existing review
  interface into a combined service.
- Extend `apps/web/src/backend/http-backend.ts` using generated types and exact
  multipart/JSON/idempotency headers.
- Extend `apps/web/src/mock/mock-engine.ts` with the same immutable lifecycle,
  authorization denials, blocker codes, idempotency, and projections. Mock
  success cannot be broader than HTTP success.
- Add canonical synthetic ZIP/PDF facts in test code generated under a
  task-owned temporary directory; do not check in copied AGA bytes or text.
- Keep actual AGA acceptance path-driven through an environment variable and
  hash assertion. Browser tests use only synthetic files.
- Confirm `apps/api/internal/application/auditee_projections.go`, React auditee
  routes, API serializers, search/counts, and notifications cannot project
  inventory/candidate/review data.

## Exact Fail-Closed Gates

Use one ordered blocker evaluator in Go and mirror its exact output in mock
tests. A Draft may be saved with blockers, but the following actions are
denied with no side effects as specified.

The canonical blocker order and codes are:

1. identity/scope: `ORIGIN_REQUIRED`, `DEPARTMENT_ASSIGNMENT_REQUIRED`,
   `FUNCTIONAL_ASSIGNMENT_REQUIRED`, `REQUIRED_OWNER_MISSING`,
   `REQUIRED_OWNER_AMBIGUOUS`, `OWNER_RESOLUTION_MISMATCH`,
   `SCOPE_RECOMMENDATION_REQUIRED`, `SCOPE_RATIONALE_REQUIRED`,
   `SCOPE_HISTORY_BASIS_REQUIRED`;
2. trace/source: `REGULATORY_TRACE_REQUIRED`, `SOURCE_MAPPING_REQUIRED`,
   `LEGACY_AUTHORITY_UNATTESTED`,
   `SOURCE_CHAIN_INCOMPLETE`, `SOURCE_CHAIN_DIGEST_MISMATCH`,
   `SOURCE_AUTHORITY_REQUIRED`, `SOURCE_AUTHORITY_RETURNED`,
   `SOURCE_IDENTITY_MISMATCH`, `SOURCE_VERSION_MISMATCH`,
   `SOURCE_HASH_MISMATCH`, `SOURCE_LOCATOR_REQUIRED`, `SOURCE_STALE`,
   `SOURCE_OWNER_ATTESTATION_REQUIRED`, `SOURCE_OWNER_ATTESTATION_RETURNED`;
3. applicability/content: `APPLICABILITY_REQUIRED`,
   `APPLICABILITY_RATIONALE_REQUIRED`, `VERIFICATION_OBJECTIVE_REQUIRED`,
   `EXPECTED_EVIDENCE_REQUIRED`, `RECONCILIATION_REQUIRED`,
   `RECONCILIATION_DIGEST_MISMATCH`;
4. guardrails: `AUTOMATIC_DEFERRAL_FORBIDDEN`,
   `MANDATORY_DEFERRAL_FORBIDDEN`, `SAFETY_CRITICAL_DEFERRAL_FORBIDDEN`,
   `UNKNOWN_HISTORY_DEFERRAL_FORBIDDEN`, `INSUFFICIENT_HISTORY`,
   `NON_COMPARABLE_HISTORY`, `SOURCE_CHANGED_REQUIRES_FULL_SCOPE`,
   `OPEN_FINDING_REQUIRES_FULL_SCOPE`,
   `REPEAT_FINDING_REQUIRES_FULL_SCOPE`,
   `OVERDUE_CONTROL_REQUIRES_FULL_SCOPE`;
5. decisions/digests: `STALE_CANDIDATE`,
   `EXISTING_CANDIDATE_SUPERSEDED`,
   `REVIEWER_RECOMMENDATION_DISPOSITION_REQUIRED`,
   `TECHNICAL_REVIEW_REQUIRED`, `TECHNICAL_REVIEW_DIGEST_MISMATCH`,
   `PUBLICATION_DECISION_REQUIRED`, `PUBLISHED_DIGEST_MISMATCH`;
6. Audit applicability: `AUDIT_TARGET_SCOPE_MISMATCH`,
   `AUDIT_INSPECTION_TYPE_MISMATCH`, `AUDIT_EFFECTIVE_PERIOD_MISMATCH`,
   `AUDIT_QUALIFIER_MISMATCH`, and `AUDIT_PACKAGE_CONTENT_MISMATCH`.

Return all applicable codes in this order. UI copy may explain a code, but
mock, HTTP, Go, PostgreSQL evidence, and React tests compare the codes and
order exactly.

### Source-authority acceptance

`AttestSourceAuthority(ACCEPT)` is denied when the source identity/version/
`sha256:` digest/class/chain role/currentness event is missing or differs from
persisted facts, the source is stale, or the actor lacks one exact current
source-owner assignment. It does not require a prior authority acceptance.
`RETURN` is allowed for a scoped proposed source with a required reason even
when the proposal is incomplete; it never creates acceptance. Neither branch
mutates an existing candidate. Both branches must extend the exact decision
root/current leaf under the source lock. A `RETURN` that supersedes a previous
`ACCEPT` records the source-authority impact event and new append-only gap
successors described above; stale earlier acceptance is never reused.

### Candidate source-mapping attestation

`AttestSourceMapping(ACCEPT)` is denied when:

- origin is absent or inconsistent with entry path;
- `scopeRecommendation` or its rationale/signals/history/guardrails is absent;
- regulatory trace is partial, empty, client-spoofed, stale, or
  `SOURCE_MAPPING_REQUIRED`;
- any required authority-chain link, source authority acceptance, source
  identity/version/`sha256:` digest/currentness event, or locator/page/section/
  clause is absent;
- proposed applicability or its rationale is absent or differs from the exact
  server-bound proposal; prior mapping acceptance is not required;
- verification objective/method or expected Evidence is empty;
- persisted source facts differ from the question snapshot;
- exactly one current complete `REVIEWED_SOURCE_SET` assignment is missing,
  ambiguous, partial-chain, expired, revoked, or outside the exact candidate
  provider/target/inspection/Department/unit scope;
- candidate/draft revision or digest is stale; or
- the Draft's bound existing-candidate snapshot is superseded (for `ACCEPT`;
  scoped `RETURN` remains allowed to remove a prior acceptance); or
- original candidate facts or reconciliation diff are missing/inconsistent.

`AttestSourceMapping(RETURN)` is allowed for a scoped complete or incomplete
proposal, including `SOURCE_MAPPING_REQUIRED`, with a non-empty reason. It
records no accepted state and cannot be projected as validation. Validation
claims, technical approval, publication, and package materialization require a
later current-leaf `ACCEPT` decision for the exact immutable digest. A `RETURN`
that supersedes an accepted mapping atomically records its candidate-root impact
and gap successor; it never reactivates an older acceptance.

### Technical approval is denied when

- any source-attestation gate above is incomplete;
- the Draft's bound existing-candidate snapshot has a newer current leaf;
- any question remains `SOURCE_MAPPING_REQUIRED`;
- any required department owner is absent or not currently assigned;
- source owner returned the mapping or attested an older digest;
- the technical-approval command omits the Department Manager's explicit
  `ACKNOWLEDGED`, `ADOPTED_AS_RETURN`, or `NOT_ADOPTED_WITH_REASON`
  disposition for a current reviewer recommendation. The reviewer record
  itself is never a veto; only `ADOPTED_AS_RETURN` creates the existing
  manager-owned binding return decision;
- any mandatory, safety-critical, overdue, open/repeat finding, source-changed,
  unknown-history, insufficient-history, or non-comparable-history guardrail is
  incorrectly deferred or omitted;
- `automaticDeferral` is true;
- a `DEFER_ELIGIBLE` recommendation lacks validated clean comparable history,
  a completed full-scope baseline within the approved interval, unchanged
  sources, rationale, and explicit human approval; or
- mappings/questions/required owners/content digest differ from the submitted
  immutable leaf.

### Publication is denied when

- technical approval is absent, returned, rejected, incomplete for joint
  owners, attached to another revision/digest, or no longer current;
- any trace/scope/source/applicability/rationale/guardrail gate is incomplete;
- any source version/hash/currentness changed after technical approval;
- the published candidate lineage binds a superseded existing-candidate
  snapshot;
- source-owner assignments or required Department Manager assignments do not
  cover the exact scope at decision time;
- a newer immutable candidate leaf exists;
- the publication request does not cite the complete distinct current
  technical-decision set, one per required owner, or its actor is not one of
  those current required owners;
- publication would reuse a template version identity/content digest with
  different bytes; or
- the actor attempts an Admin, source-owner-only, reviewer, manager-without-
  assignment, Inspector, or Auditee bypass.

### New Audit-package materialization is denied when

- no separately published immutable version exists;
- the published version/candidate/question/package digest does not match;
- source currentness is stale or a source-impact Draft is unresolved;
- the published lineage binds a superseded existing-candidate snapshot;
- target/provider/department/unit/inspection type/effective dates/qualifiers do
  not exactly match the published applicability;
- any question is absent, reordered, partially projected, unmapped, or
  automatically deferred;
- required mandatory/safety/unknown-history guardrails are not preserved; or
- the actor is not a current Department Manager matching the existing exact-
  scope package-composition authority.

Inspector/Lead execution authorization is evaluated later when answers or
other execution effects are written to the already materialized package. It is
not a precondition for Department Manager package composition and is never
inferred from manager status.

Denial of a new package never mutates an already materialized in-progress
Audit. Its exact published version, question snapshots, source hashes,
responses, findings, and package bytes remain pinned.

## Ordered Implementation Tasks

The original planning task did not authorize implementation. The current user
request separately authorizes execution in order. Tasks 1–10 are complete only
after their focused
RED, GREEN, regression, and read-only review evidence is recorded. Gate 0 is a
contract-freeze checkpoint: every Gate-0-owned test must be GREEN, while later
tests are registered as phased expectations and are not required to exist or
pass until their owning task. No task may finish with one of its own tests RED.
Do not start any Phase 2 AGA expansion before Gate 6 passes **and** the real
Form 048 identity/boundary decisions, immutable candidate, and visible source-
gap Draft have been created by the exact authorized actors.

### Execution disposition (2026-07-31)

The user separately authorized completion of this plan after the planning-only
Gate 0 checkpoint. The following is the authoritative implementation ledger:

| Scope | Result | Boundary |
| --- | --- | --- |
| Gate 0 | `verified locally` | Frozen vocabulary, authority/privacy rules, limits, archive facts, read-only verifier, and phased inventory are green. |
| Task 1 | `verified locally` | OpenAPI/source/generated transport contracts and strict discriminators are synchronized; no external authority is implied. |
| Task 2 | `verified locally` | Forward-only migration 28, append-only store types, and fail-closed functional-assignment resolution are implemented; task-owned migration-28 PostgreSQL execution and focused integration checks passed. |
| Task 3 | `verified locally` | Bounded archive/PDF policy, parser boundary, worker/service, cleanup, and Admin HTTP boundary are covered; live MinIO/ClamAV/Poppler execution is `blocked`. |
| Task 4 | `verified locally` | Candidate-only extraction review, identity-resolution, replay, and Form 048-shaped synthetic mechanism are covered; real Admin identity/extraction decisions are `blocked`. |
| Task 5 | `verified locally` | Synthetic dual authoring, candidate lineage, strict traces, and fail-closed authoring boundary are covered; official source facts remain `blocked`. |
| Task 6 | `verified locally` | Synthetic reconciliation, source-attestation separation, review queue/comments, and publication/eligibility boundary are covered; real owner/manager decisions remain `blocked`. |
| Task 7 | `verified locally` | Source-authority/mapping/publication/package discriminators and transport mapping are covered without a parallel publication path. |
| Task 8 | `verified locally` | Candidate-only mock/HTTP React surfaces, parity, typecheck, builds, and skipped browser contracts are covered; live browser/runtime dependencies are `blocked`. |
| Task 9 | `blocked` | The user explicitly authorized continuation on 2026-08-01. Synthetic mechanism and task-owned connected-runtime checks are green, but no additional AGA form is imported because the real Form 048 Admin identity/28 boundary packet, immutable candidate/source-gap Draft, and named expansion authorization remain absent. |
| Task 10 | `verified locally` | Metadata-only evidence, plan/index/tracker/harness updates, cleanup, final checks, and independent read-only review are recorded below. |

This table supersedes the planning-only “not run” wording that preceded
authorization. It does not convert any external source-owner, reviewed-source
set, functional-assignment, Department Manager, deployment, or production
decision into a local result.

### Gate 0 — Freeze contract vocabulary, authority, limits, and test inventory

**Status:** `verified locally` (independent read-only review `APPROVED`).

**Files**

- Modify `docs/product-specs/modules/CHECKLIST_BUILDER_AND_RUNNER.md`.
- Modify `docs/product-specs/modules/AUDIT_PLANNING.md`.
- Modify `docs/product-specs/modules/ADMIN_CONFIGURATION.md`.
- Modify `docs/product-specs/data-and-rules/CONCEPTUAL_DATA_MODEL.md`.
- Modify `docs/product-specs/data-and-rules/STATUS_PERMISSION_SECURITY.md`.
- Modify `docs/product-specs/screen-specs/SCREEN_INVENTORY_AND_FORMS.md`.
- Modify `docs/product-specs/screen-specs/DEPARTMENT_MANAGER_WORKSPACES.md`.
- Modify `scripts/verify-governed-checklist-test-inventory.mjs`.
- Create `tests/governed-checklist-intake-plan-contract.test.mjs`.
- Create `tests/aga-checklist-archive-inventory.test.mjs`.
- Create `tests/governed-checklist-intake-security.test.mjs`.

**Interfaces**

- Produces the exact lifecycle enums, permission table, blocker codes,
  `AGA_ZIP_PDF_V1` limits, AGA expected archive/register/form-048 hashes, and
  fail-closed test inventory used by all later tasks.

- [ ] Write Gate-0-owned plan/spec contract tests for the frozen authority
  chain, server-derived ownership, functional-assignment blocked provisioning,
  role matrix, strict trace vocabulary, parser limits, and AGA verifier facts.
- [ ] Run the Gate-0-owned Node tests and record the initial specification/
  inventory failures, then update only the owning specifications/inventory.
- [ ] Update the seven product specifications with the settled lifecycle,
  authority, role/assignment, privacy, and fail-closed language in this plan.
- [ ] Extend the verification inventory with explicit phases
  `gate0`, `task1`, `task2`, `task3`, `task4`, `task5`, `task6`, `task7`,
  `task8`, `task9`, and `final`. A phase enforces only artifacts owned through
  that phase; `task9` may report its explicit authorization/real-slice
  dependency as `blocked` but never passed or skipped, and `final` enforces
  every required path and invokes Vitest/Playwright runner discovery, not only
  source-text regex counts.
- [ ] Implement the read-only AGA verifier so it requires
  `AGA_CHECKLIST_ARCHIVE`, streams hashes and ZIP metadata, writes nothing,
  rejects unsafe entries, and asserts the planning facts above.
- [ ] Run `node scripts/verify-governed-checklist-test-inventory.mjs --phase
  gate0` plus all Gate-0-owned tests. Expect every Gate-0-owned check GREEN and
  the read-only AGA inventory test to pass against the supplied external path;
  no Task 1 OpenAPI test is created or required yet.
- [ ] Obtain a read-only Gate 0 review confirming no top-level role, source
  authority, or publication bypass was introduced.

### Task 1 — Define strict OpenAPI and generated transport contracts

**Status:** `verified locally` for the candidate contract slice.

**Files**

- Modify `api/openapi/source/schemas/platform.json`.
- Modify `api/openapi/source/paths/platform.json`.
- Modify `api/openapi/tests/regulatory-question-governance-contract.test.mjs`.
- Modify `api/openapi/tests/governed-checklist-publication-boundary.test.mjs`.
- Create `api/openapi/tests/governed-checklist-intake-contract.test.mjs`.
- Create `api/openapi/tests/governed-checklist-authoring-contract.test.mjs`.
- Modify `scripts/generate-go-contracts.mjs`.
- Modify `api/openapi/tests/go-contract-generation.test.mjs`.
- Regenerate `api/openapi/aviasurveil360.yaml`.
- Regenerate `apps/api/internal/httpapi/generated/api.gen.go`.
- Regenerate `apps/web/src/generated/transport/api-types.ts`.

**Interfaces**

- Produces the exact operations, request/view/problem types, discriminated
  trace/candidate-lineage/source-review-detail unions, reconciliation fields,
  eligibility view, and generated client signatures described in the API
  section.

- [ ] Write RED tests for both strict trace-content variants, independent
  decision projections, ordered authority chains, server-derived owner output,
  every lineage row/impact variant, both source-review detail variants, all
  queue/read/mutation operations, exact multipart cardinality, and complete
  reconciliation fields. Reject every prohibited cross-variant combination and
  every partial `currentDecision` tuple, including missing root/leaf/outcome/
  subject/semantic/token fields, decision fields under `NO_DECISION`, and stale
  or non-leaf tokens at the owning runtime boundary.
  Cover the Admin extraction-review GET, bounded ordered page, private proposed-
  text fields, immutable-initial/derived-effective identity view, and strict
  no-current/current identity-resolution and extraction-decision-set unions.
  For inner extraction decisions, cover valid `ACCEPT`, `SPLIT`, `MERGE`,
  `TRANSCRIBE`, and `EXCLUDE` input/view variants plus every missing, extra,
  partial, client-output, and cross-variant field combination.
  Freeze per-item `maxLength`, `maxItems`, and route-specific request/response
  byte-limit problem codes; serialized aggregate byte enforcement remains an
  owning handler/service test and cannot weaken the existing global guard.
  Require both path and body identities in every keyed
  mutation schema; reject omitted, empty, extra, partial, overlapping, and
  structurally inconsistent payloads, and require expected version/digest/
  currentness tokens. Runtime path/body equality, persisted staleness, and zero-
  effect rejection belong to the owning handler tasks, not OpenAPI alone.
- [ ] Run the focused OpenAPI tests; expect schema/operation failures.
- [ ] Add the schemas and paths, preserving existing operation IDs and the
  separate Department Manager technical/publication routes.
- [ ] Extend the Go generator so every multi-member governed `oneOf` produces a
  strict discriminated typed decoder rather than `json.RawMessage`, and
  multipart produces exactly
  one binary `archive` plus one typed JSON `receipt` field. Regenerate all three
  tracked artifacts with `./scripts/generate-contracts.sh`.
- [ ] Run `./scripts/check-contracts.sh` and all governed OpenAPI tests; expect
  zero source/bundle/Go/TypeScript drift and no Admin direct-publish route.
- [ ] Review generated Go optionality: every resolved trace field must be
  non-optional in its variant, and the source-gap variant must not serialize a
  misleading empty resolved trace. Do the same for all candidate-lineage and
  source-review detail variants plus identity-resolution/current-decision-set/
  extraction-action variants, including required-null versus prohibited
  generation-run/binding/legacy/private fields.

### Task 2 — Add migration 28, immutable stores, and scoped assignments

**Status:** `verified locally`; connected PostgreSQL execution remains
`blocked`.

**Files**

- Create `apps/api/migrations/000028_governed_checklist_intake_and_authoring.up.sql`.
- Modify `apps/api/migrations/migrations.go`.
- Create `apps/api/internal/checklistintake/types.go`.
- Create `apps/api/internal/checklistintake/store.go`.
- Create `apps/api/internal/checklistintake/postgres_store.go`.
- Create `apps/api/internal/identity/checklist_functional_authority.go` and
  tests.
- Create `apps/api/tests/integration/governed_checklist_intake_migration_test.go`.
- Create `apps/api/tests/integration/governed_checklist_assignment_authority_test.go`.

**Interfaces**

- Produces version-28 tables/constraints/indexes, immutable store transaction
  methods, exact functional assignment lookup, idempotency, and migration
  recovery used by intake and authoring services.

- [ ] Write migration RED tests for fresh install, version-27 upgrade, the
  complete lineage truth table, governed-discriminator guards/indexes,
  reviewed-source-set roots/versions/links/assignment FK, source-authority/
  candidate-binding sets, identity-resolution roots/leaves, extraction packets/
  proposals/decision-set roots/leaves, existing-candidate roots/revisions/
  current leaves, upload intents, fenced phase attempts, all
  table/column/enum/check/index/foreign-key invariants, append-only protection,
  assignment roots/successors/scopes/effective dates, attestation root/current-
  leaf transitions, the two named deferred candidate/binding-set FKs, and
  semantic idempotency. Include insert/commit, rollback, orphan, update/delete,
  concurrent-successor, replay, and accepted-then-returned cases.
- [ ] Add identity/extraction persistence RED tests proving terminal manifest/
  file facts never change; one initial root and one current leaf per file/packet;
  exact packet/manifest/parser/file digest FKs; complete ordered proposal
  coverage; and replay-stable, stale/concurrent/divergent correction conflicts.
- [ ] Write authorization persistence RED tests for current, future, expired,
  revoked, cross-department, cross-source, manager-without-assignment, orphan/
  wrong-version/wrong-digest/client-invented reviewed set, wrong candidate
  scope, and partial/extra/reordered set coverage cases. Recompute the checked-in
  digest golden from persisted links; reject missing/mutated/reordered source
  version/hash/class/role tuples. Prove a later current source version leaves the
  old set/digest unchanged, requires a new reviewed-set successor plus a new
  assignment successor, and is denied by the prior assignment.
- [ ] Run focused migration/integration tests and record the expected missing
  version-28 failures.
- [ ] Add the forward-only migration and store types. Do not add a destructive
  down migration; use forward repair for post-data recovery.
- [ ] Replace/generalize the migration constraints, indexes, store lookups, and
  persistence sentinels owned by this task. Prove a version-27 upgrade leaves
  prior candidate JSON/digests, decisions, published versions, Audit snapshots,
  and package bytes unchanged, backfills the exact lineage kind plus
  `PRE_V28_UNATTESTED`, leaves new author/owner-resolution/blocker fields and
  facts absent, inserts no binding/attestation/decision/publication, and denies
  every new effect until an attested successor exists. Runtime queue/command/currentness sentinel
  replacement belongs to Tasks 5–7; source locking is conditional on a proposed
  or persisted binding and a pure source-gap row has no source identity to lock.
- [ ] Implement store transaction primitives that later tasks use to atomically
  create terminal receipts, candidates/questions, assignments/attestations/
  comments, lineage, deferred binding pairs, and typed conflicts. Do not claim
  later HTTP/application behavior in this task.
- [ ] Run migration/store tests against a task-owned disposable PostgreSQL
  database. Prove failed candidate import and Draft creation leave zero partial
  rows while failure receipts remain append-only.
- [ ] Implement global-latest-successor-before-filtering functional authority,
  strict null-scope rejection, and internal-CAA membership checks. Canonical
  tests may seed assignments; assert no real grant/revoke route or Admin
  self-grant exists.
- [ ] Review locks and indexes for deterministic replay, batch claiming,
  current-assignment lookup, candidate leaf resolution, owner derivation,
  generalized source binding, and shared source-impact queries.

### Task 3 — Implement bounded archive receipt, parser worker, and cleanup

**Status:** `verified locally` for bounded policy and fail-closed boundaries;
live MinIO/ClamAV/Poppler qualification remains `blocked`.

**Files**

- Create `apps/api/internal/checklistintake/policy.go` and tests.
- Create `apps/api/internal/checklistintake/archive.go` and tests.
- Create `apps/api/internal/checklistintake/register.go` and tests.
- Create `apps/api/internal/checklistintake/pdf_parser.go` and tests.
- Create `apps/api/internal/checklistintake/parser_sandbox.go` and tests.
- Create `apps/api/internal/checklistintake/authorization.go` and tests.
- Create `apps/api/internal/checklistintake/service.go` and tests.
- Create `apps/api/internal/worker/checklistintake/worker.go` and tests.
- Create `apps/api/cmd/checklist-pdf-parser/main.go` and tests.
- Modify `apps/api/cmd/worker/main.go` and tests.
- Modify `apps/api/internal/platform/config/config.go` and tests.
- Modify `apps/api/internal/platform/objectstore/object_store.go`,
  `apps/api/internal/platform/objectstore/minio_store.go`, and their tests.
- Modify `apps/api/internal/platform/scanner/clamav.go` and tests.
- Modify `apps/api/Dockerfile`.
- Modify `deploy/local/clamav/entrypoint.sh` and add configuration-policy tests.
- Create `deploy/local/checklist-pdf-parser-seccomp.json`.
- Modify `deploy/local/compose-policy.json`.
- Modify `deploy/local/compose.yaml`, `deploy/local/compose.test.yaml`, and
  `tests/local-compose-policy.test.mjs`.
- Create `apps/api/internal/httpapi/governed_checklist_intake_api.go` and tests.
- Modify `apps/api/internal/httpapi/canonical_api.go` and
  `apps/api/internal/httpapi/canonical_api_test.go`.
- Create `scripts/test-governed-checklist-intake-profile.sh`.

**Interfaces**

- Consumes Task 1 generated contracts and Task 2 stores.
- Produces `ReceiveBatch`, worker claim/receipt transitions, immutable generic
  register/match facts, `ResolveIdentity`, and inventory read projections for
  Task 4/UI.

- [ ] Write RED unit/table tests for `NormalizeZipPathV1`, local/central header
  mismatch, range overlap/bounds/trailing bytes, methods, every path/type/size/
  count/ratio/page/text/timeout/CRC/hash/magic/ZIP/PDF-encryption/ZIP64/
  duplicate error code, exact emitted-byte counters, and both cleanup paths.
  Include 129 directory-only records, valid `dir/`, inconsistent directory
  attributes, nonzero stored/deflated/descriptor-bearing directories, file/
  directory/prefix collisions, and all scanner boundaries.
  Generate malicious/synthetic archives only under task-owned temp.
- [ ] Write RED register-DAG tests with the register first, middle, and last;
  missing/duplicate/ambiguous registers; incomplete table rows; unmatched/extra
  forms; and an assertion that no `IDENTITY_MATCH` attempt starts before the
  exact successful `REGISTER_PARSE` receipt. Matching must use immutable parsed
  facts, never ordinal or filename inference.
- [ ] Add checked-in `AGA_IMPORT_MANIFEST_V1` golden payload bytes/digests for
  pre-PUT ZIP rejection, archive-malware failure, per-file-malware failure,
  parser failure, identity-conflict completion, cleanup failure, crash-before-
  cleanup/restart, and full success. Prove every fixed-DAG phase state and
  conditional object/event field, unconditional cleanup, retry-winner selection,
  query-order independence, active/retryable-chain refusal, unmarked-gap
  refusal, replay, exact parser-output intent/version/hash/bytes on every
  successful parse, and competing-finalizer byte/digest identity.
- [ ] Write RED HTTP tests for authentication-before-body, streaming limit,
  multipart cardinality, idempotent replay, path/body mismatch before service,
  mismatch conflict, permission denial, and no side effects. For identity
  resolution, prove first/current expected-state variants, immutable initial
  state/manifest, derived effective status, stable replay, current-leaf
  correction, stale/concurrent/divergent `409` behavior, and denial for both
  `REGISTER_MATCHED` and `NOT_REGISTERED` initial states. Reject a named-source
  value mismatch and accept human transcription only with its separate exact
  reason/receipt while preserving every original value.
- [ ] Run focused tests; expect missing policy/service/handler failures.
- [ ] Implement central-directory inspection and streaming entry validation
  without shell extraction or client-derived paths.
- [ ] Implement generic `AGA_REGISTER_V1` recognition/parsing and form matching
  after every safe PDF parse, persisting exact original text/hash/page/row
  provenance. Require exactly one register; never infer an absent form or start
  matching before its successful parse receipt.
- [ ] Extend the object-store boundary with version-specific conditional put
  carrying atomic tags/checksum request fields, stat/open, scoped list, and
  delete operations; enable MinIO bucket
  versioning. Wire intent/tag/trusted-checksum-or-exact-version-rehash/HEAD/
  finalization, archive and exact decompressed-
  PDF ClamAV scans with signature freshness, the separate fixed-argument
  no-network parser sandbox, at-least-once attempts with one terminal result,
  worker leases, and owned scratch/orphan reconciliation.
- [ ] Implement and register exactly the Task 3 Admin operations—receive, batch
  read, file list, receipt list, and identity resolution—with exact problem
  mappings. Task 4 owns extraction-review create/read and candidate import.
- [ ] Run focused unit/HTTP/integration tests with scanner/parser fakes, then
  `scripts/test-governed-checklist-intake-profile.sh` with real
  PostgreSQL/MinIO/ClamAV and the pinned parser image. From inside the parser,
  prove the AF_UNIX job succeeds while the named seccomp profile rejects
  AF_INET/AF_INET6/AF_NETLINK/AF_PACKET socket syscalls and DNS/TCP, secrets, Docker socket, and
  writable root/input are unavailable. Test input-digest mismatch, parser
  unavailable, output/file-size limit, SIGKILL, and partial-output discard.
- [ ] Prove a `PDF_PARSE` success cannot commit before exact private parser-
  output PUT/version/hash/byte verification; cover 25 MiB/boundary+1, conditional
  collision, accepted-response-lost recovery, mismatched version/tag/checksum,
  crash before/after output intent/PUT/event, and scratch cleanup preserving the
  durable exact output while deleting only temporary copies.
- [ ] Verify success, archive-clean/file-dirty, stale signatures, scan limit,
  exact allowed scanner boundaries and boundary-plus-one, dial/archive/PDF/
  batch scan timeout, clamd capacity configuration, encrypted PDF, parser
  timeout, SIGTERM, worker restart, and crash points
  before/after external execution, object PUT, HEAD verification, and terminal
  commit. Include wrong checksum/tag/version, conditional-write collision,
  PUT-accepted/response-lost recovery,
  stale-worker late success before/after fenced retry creation, competing
  finalizers, register first/middle/last, crash before/during/after cleanup,
  restart of the same fenced cleanup chain, crash between attempt completion
  and finalization, and
  two batches with the same hash where one expired intent cannot remove the
  surviving exact bytes. Every case preserves attempts/intents, reconciles only
  owned objects/scratch, and creates no candidate.

### Task 4 — Inventory the AGA register and prepare the Form 048 owner packet

**Status:** `verified locally` for synthetic extraction/identity mechanics;
real Admin identity and extraction decisions remain `blocked`.

**Files**

- Create `apps/api/internal/checklistintake/extraction_review.go` and tests.
- Extend `apps/api/internal/checklistintake/service.go` and store tests.
- Extend `apps/api/internal/httpapi/governed_checklist_intake_api.go` and tests
  with extraction-review create/read and candidate-import routes.
- Modify `apps/api/internal/httpapi/canonical_api.go` and
  `apps/api/internal/httpapi/canonical_api_test.go` for that route.
- Create `apps/api/tests/integration/aga_form_048_candidate_intake_test.go`.
- Create `scripts/regulatory/inventory-checklist-archive.mjs` as a read-only
  receipt verifier; do not make it a separate authority path.

**Interfaces**

- Consumes the accepted immutable batch/file/phase/register/match receipts from
  Task 3 plus the exact receipt-bound private parser-output object version,
  without reparsing PDF bytes or rewriting terminal facts.
- Produces a private candidate-only decision packet for Form 048 identity plus
  28 proposed extraction boundaries. The real
  identity resolution, extraction decisions, and
  `EXISTING_CHECKLIST_CANDIDATE` remain `blocked` until a current authenticated
  Admin principal separately supplies them. A Department Manager or functional
  assignment cannot substitute. Synthetic fixtures prove candidate mechanics
  but are permanently ineligible for promotion as the real Form 048 record.

- [ ] Write RED tests asserting the external AGA archive hash, 53 PDF entries,
  one register, 52 unique form identities, missing 049, present 035A, register
  hash, Form 048 hash/page/question count, and title conflict.
- [ ] Write RED integration tests proving candidate import is blocked before
  the exact current Admin-bound identity-resolution leaf required by the Form
  048 conflict and all question-boundary decisions, rejects path/body ID
  mismatch before service, rejects a synthetic-as-real command, denies every
  non-Admin role, and creates no partial rows on any error. Cover unambiguous
  `REGISTER_MATCHED` import without a fabricated human resolution separately.
- [ ] Write RED extraction-review GET tests for Admin-only authorization before
  lookup, guessed/cross-batch IDs, stable ordinal/ID pagination, bounded page
  size, exact repeated packet/file/manifest/parser digests, original text/
  locator provenance, no-current/current decision-set variants, and exclusion
  from every other role/list/search/count/notification projection. A decision-
  leaf change must stale an older continuation cursor; every response must be
  `Cache-Control: no-store` and absent from HTTP logs/telemetry.
- [ ] Write RED extraction-preparation tests for exact receipt-bound parser-
  output version/hash/bytes, streaming rehash, READY/FAILED idempotent replay,
  policy/size `413`, object mismatch, crash/restart, unique exact tuple, and one-
  transaction packet/proposal insert. Child/count/digest/commit failure must
  leave zero packet/proposal rows; a deterministic committed FAILED packet has
  zero children and cannot be retried as READY for that file/manifest/parser-
  receipt tuple. A parser/generator-policy correction requires a new batch and
  parser receipt rather than mutation or ambiguous packet selection.
- [ ] Add boundary and boundary-plus-one tests for 25 MiB parser output, 1,000
  spans, 64 KiB per/8 MiB aggregate proposal text, 2,000 output seeds, 32 KiB
  per/512 KiB aggregate transcription, 64 KiB preparation/identity/receipt JSON,
  1 MiB candidate JSON, and count-plus-1 MiB extraction response paging. Keep
  the repository's global 1 MiB JSON guard unchanged.
- [ ] Add candidate-binding tests for expected no-current/current decision-set
  state, exact current identity root/leaf/token, packet/parser/manifest/file
  digests, strict first/boundary `APPEND` versus identity-only `REUSE_CURRENT`,
  full non-overlapping proposal coverage, replay, concurrent imports, stale
  identity or decision correction, strict candidate `INITIAL|CORRECT_CURRENT`,
  concurrent/stale predecessor correction, and exact current/superseded
  successor projection.
  Exercise every valid persisted `ACCEPT`, `SPLIT`, `MERGE`, `TRANSCRIBE`, and
  `EXCLUDE` result: server-derived IDs/text/digests/ordered provenance, full
  exactly-once proposal consumption, deterministic question ordering, and zero
  question for exclusion. Reject out-of-range/overlapping/incomplete subspans,
  non-adjacent/overlapping merges, client output facts, missing reasons/bases,
  partial sets, and every cross-variant payload with zero rows.
  Byte-compare terminal manifest/file facts and every earlier decision/candidate
  before and after each correction.
- [ ] Run the focused tests; expect missing AGA-specific extraction/candidate
  behavior.
- [ ] Verify the AGA-specific inventory by consuming the exact immutable Task 3
  register/match receipts; do not reparse source bytes, append a second register
  result, or mutate a terminal manifest.
- [ ] Implement/register `CreateExtractionReview`, `GetExtractionReview`,
  `CreateExistingCandidate`, and all three canonical routes with exact current-
  Admin authorization, path/body equality where applicable, stable paging,
  replay/conflict, and atomic store behavior. Preparation streams only the exact
  durable parser-output version and inserts READY packet/proposals in one
  transaction. Candidate import locks/rechecks exact current candidate, identity,
  and extraction-decision leaves and binds packet/file/manifest/parser digests.
  The routes may exercise synthetic fixtures, but the real Form 048 command
  remains blocked until the decisions below exist.
- [ ] Idempotently persist and expose the pending
  `AGA_EXTRACTION_REVIEW_V1` Form 048 packet from the exact accepted parser/
  file/manifest facts and exact durable parser-output object version. Insert the
  packet/proposals in one transaction after full count/digest validation.
  Preserve every competing register/visible/PDF-metadata identity and all 28
  proposed question spans with original text, hashes, and page/row provenance.
  It contains no selected identity, accepted boundary, regulatory mapping, or
  approval claim.
- [ ] Prove the real candidate command remains `blocked` with zero effects
  while the decision packet lacks a current Admin principal, current membership
  identity, decision IDs, exact file/span digests, reasons, and timestamps.
- [ ] Prove the mechanism with a clearly synthetic canonical fixture. If and
  only if the authorized human decisions are later supplied, resume Task 4 by
  appending those exact real identity/extraction decisions and atomically
  creating Form 048. Replay must return the same candidate; a divergent
  decision must conflict. Blank history remains `NOT_SUPPLIED`, never clean.
- [ ] Run the read-only receipt verifier and compare the API/DB inventory with
  the external ZIP. Store only hashes/counts/receipt identities in the
  evidence record; raw source bytes and full extracted text remain excluded.
  The separately user-authorized all-form handoff may contain bounded
  parser-derived question strings/provenance and printed reference strings,
  but it is not a source-text dump or an imported candidate record.

### Task 5 — Create immutable Drafts from existing and official sources

**Status:** `verified locally` for synthetic candidate/official boundary
mechanics; real source authority remains `blocked`.

**Files**

- Create `apps/api/internal/regulatory/draft_authoring.go` and tests.
- Create `apps/api/internal/regulatory/required_owners.go` and tests.
- Create `apps/api/internal/httpapi/governed_checklist_authoring_api.go` and tests.
- Create `apps/api/internal/regulatory/draft_store.go`; do not add archive
  behavior to the existing regulatory store.
- Modify `apps/api/internal/regulatory/generation.go` and
  `apps/api/internal/regulatory/generation_test.go`.
- Modify `apps/api/internal/regulatory/admin.go` and
  `apps/api/internal/regulatory/store.go`.
- Modify `apps/api/internal/httpapi/governed_candidate_admin_api.go`,
  `apps/api/internal/httpapi/canonical_api.go`, and
  `apps/api/internal/httpapi/canonical_api_test.go`.
- Create `apps/api/tests/integration/governed_checklist_dual_authoring_test.go`.
- Create
  `api/openapi/examples/canonical/governed-existing-checklist-draft.json`.
- Create
  `api/openapi/examples/canonical/governed-official-source-draft.json`.

**Interfaces**

- Produces `CreateDraftFromExistingCandidate` and
  `CreateOfficialSourceDraft` services returning existing
  `GovernedCandidateView` with exact entry path and lineage.

- [ ] Write RED tests for an immutable synthetic Form-048-shaped candidate Draft
  whose questions all start with origin `EXISTING_CHECKLIST_CANDIDATE`, explicit
  `SOURCE_MAPPING_REQUIRED`, unknown-history guardrail, and no decision,
  publication, version, or package effects.
- [ ] Write RED tests for official-source-only Draft creation where every
  authority-chain link and owner is server-resolved and complete; include
  unapproved-current, approved-stale, missing controlled procedure, mixed
  authority, partial/reordered chain, client hash/Department disagreement,
  ambiguous/no owner, empty Evidence/rationale, out-of-scope author, and
  synthetic-as-real rejection cases. Assert that no generation run is created
  and the source-binding set plus command/input/output digests are complete.
- [ ] Add handler path/body mismatch zero-effect tests and barrier tests racing
  source-currentness activation against official-source creation; no stale
  `RESOLVED` Draft may commit.
- [ ] Run focused tests and record expected missing service failures.
- [ ] Implement both commands as separate entry paths converging on the same
  immutable candidate persistence and canonical digest service.
- [ ] Require exact current source/currentness/clause/authority-attestation
  identities, complete `OFFICIAL_CHECKLIST_SOURCE_CHAIN_V1`, generalized
  source-binding set, and server-derived required owners for official
  questions. Official-source-only creation fails atomically on any gap; only
  existing-candidate/hybrid Drafts may save `SOURCE_MAPPING_REQUIRED`.
- [ ] Acquire ordered source-identity locks before any official binding-set
  commit and re-read currentness, current authority-attestation leaves, owner
  resolution, and content/chain digests in that same transaction. Keep
  `generation_run_id` null for direct official authoring; do not manufacture a
  generation artifact merely to satisfy legacy guards.
- [ ] Add the two complete synthetic canonical examples and prove the contract
  harness rejects an AGA-derived authority claim, an empty source gap, and a
  partial resolved trace.
- [ ] Prove candidate-to-Draft lineage, exact leaf concurrency, replay, and
  atomicity for Admin, current assigned Department Manager, a source owner who
  is separately one of those exact author types, source-owner assignment alone,
  generic manager, reviewer, and Auditee.
- [ ] Deny Draft creation from a superseded existing-candidate snapshot and add
  a correction-versus-Draft-create barrier race under the shared candidate-root
  lock. Prove a current corrected snapshot starts a new governed Draft root and
  every already-created old Draft remains byte/digest readable but blocked from
  new validation/approval/publication/package effects.
- [ ] Run Go/HTTP/OpenAPI regressions and inspect that existing synthetic
  generation-run import remains byte/digest compatible while both new root
  kinds and every pre-v28/new generation/direct/existing/impact lineage variant
  load through the strict `GovernedCandidateView`; pre-v28 projects only
  `PRE_V28_UNATTESTED`, and no loader filters governed rows by non-null
  generation run. Recheck currentness guards.

### Task 6 — Implement reconciliation and evaluate the Form 048 mechanism gate

**Status:** `verified locally` for synthetic reconciliation/review mechanics;
the real Form 048 mechanism gate remains `blocked`.

**Files**

- Create `apps/api/internal/regulatory/reconciliation.go` and tests.
- Create `apps/api/internal/regulatory/source_attestation.go` and tests.
- Create `apps/api/internal/regulatory/review_queue.go` and tests.
- Create `apps/api/internal/regulatory/review_comments.go` and tests.
- Extend `apps/api/internal/regulatory/review_validation.go` and tests.
- Extend `apps/api/internal/httpapi/governed_checklist_authoring_api.go` and
  tests with source-authority, mapping-attestation, detail, comment, and queue
  routes.
- Modify `apps/api/internal/httpapi/canonical_api.go` and
  `apps/api/internal/httpapi/canonical_api_test.go` to register and prove every
  Task 6 route.
- Create `apps/api/internal/checklistgovernance/eligibility.go` and tests.
- Create `apps/api/tests/integration/aga_form_048_reconciliation_test.go`.

**Interfaces**

- Produces immutable `HYBRID_RECONCILED` successors, strict field-level diffs,
  source-owner attestations, shared blocker evaluation, and the Gate 6
  mechanism result. Phase 2 still requires the separate real Form 048 slice.

- [ ] Write RED tests covering every reconciliation family: wording,
  verification objective/method, expected Evidence, applicability/rationale,
  scope classification/rationale/signals/guardrails, complete authority chain,
  split/merge/exclusion, original operational intent, and result-history
  disposition, including every outcome and unmapped question.
- [ ] Write RED tests proving the AGA source remains non-authoritative and its
  historical Manager approval block cannot satisfy any current decision.
- [ ] Write RED tests for source-authority versus candidate-mapping decision
  separation; initial `ACCEPT`/`RETURN`; return-then-accept; accepted-then-return
  impact; deterministic root/current-leaf selection; concurrent successor;
  replay; transfer-bound ordinary new `ACCEPT`; and all source-owner scope/
  effective-date/digest/hash/source-class denials. Include a two-link/different-owner chain,
  partial/overlapping/full-set mapping assignments, scope mismatch, and exact
  mapping aggregation. Add queue/detail privacy, guessed-ID `404`, cross-
  assignment denial, both strict source-review detail variants, cross-variant/
  extra-private-field rejection, every partial/contradictory `currentDecision`
  tuple, root/leaf/subject mismatch, stale/non-leaf token, Admin-inventory
  exclusion, reviewer comment-only behavior, and every handler path/body
  mismatch with zero effects.
- [ ] Run the focused tests and record expected missing reconciliation and
  attestation behavior.
- [ ] Implement server-computed full reconciliation diffs, source-authority
  attestations, candidate mapping attestations, and scoped source-owner/
  reviewer queues and detail projections. A mapping `ACCEPT` requires the one
  complete `REVIEWED_SOURCE_SET` assignment; per-link owners may differ. Do not
  populate a real Form 048 mapping until the real source owners accept every
  exact current chain link and the complete-set mapping owner separately acts.
- [ ] Acquire ordered locks and re-read source currentness, attestation leaves,
  candidate leaf, and digests in the hybrid/binding-set transaction. Add a
  source-activation barrier race proving no stale `RESOLVED` hybrid commits,
  plus a candidate-correction barrier race proving a superseded candidate or
  Draft cannot create a hybrid successor.
- [ ] Demonstrate the source-gap mechanism end to end with the non-promotable
  canonical Form-048-shaped fixture: inventory, candidate, Draft, visible
  differences/gaps, return/comment flows, and denied technical approval/
  publication/package. If the current Admin decisions are supplied, rerun the
  same assertions against the exact real candidate/gap Draft; never substitute
  the fixture for that Phase 2 prerequisite.
- [ ] When real mappings are later supplied, require source-owner confirmation
  and responsible Department Manager input at the explicit handoff points;
  preserve the source-gap Draft and create a new hybrid successor.
- [ ] Mark Gate 6 passed only when all synthetic mechanism tests are green and
  the real Form 048 state is truthfully recorded as either still `blocked` or
  separately owner-approved. A blocked real mapping does not block proving the
  candidate-only mechanism. However, missing real Admin identity/boundary
  decisions, immutable real candidate, or real source-gap Draft blocks **all**
  Phase 2 expansion; a missing real mapping also blocks real publication but
  need not block expansion after that real source-gap slice exists.

### Task 7 — Prove separate technical approval, publication, eligibility, and source impact

**Status:** `verified locally` for discriminator and fail-closed boundary
mechanics; real owner/manager decisions remain `blocked`.

**Files**

- Extend `apps/api/internal/checklistgovernance/service.go` and tests.
- Extend `apps/api/internal/checklistgovernance/eligibility.go` and tests.
- Extend `apps/api/internal/checklistgovernance/applicability.go` and
  `apps/api/internal/checklistgovernance/applicability_test.go`.
- Extend `apps/api/internal/regulatory/impact.go` and tests.
- Extend `apps/api/internal/httpapi/governed_checklist_manager_api.go` and tests.
- Modify `apps/api/internal/httpapi/canonical_api.go` and
  `apps/api/internal/httpapi/canonical_api_test.go` to register the eligibility
  route and prove every Task 7 route remains present.
- Create
  `apps/api/internal/httpapi/governed_checklist_transport_mapping_test.go`.
- Extend `apps/api/internal/application/governed_checklist_publication_boundary_test.go`.
- Create `apps/api/tests/integration/governed_checklist_intake_lifecycle_test.go`.
- Create `tests/governed-checklist-discriminator-contract.test.mjs`.

**Interfaces**

- Consumes Task 6 blockers/attestations and existing Task 6 currentness/impact
  records.
- Produces separate decisions, published version, computed eligibility, source
  impact Draft, and historical preservation proof.

- [ ] Write RED tests for every validation, technical approval, publication,
  automatic deferral, and package blocker listed above.
- [ ] Cover `EXISTING_CANDIDATE_SUPERSEDED` at validation-claim append, mapping
  `ACCEPT`, technical, publication, and new-package gates, including correction
  races at each shared lock and zero stale side effects. Preserve older
  candidates, Drafts, validation rows, published rows, and in-progress Audit
  bytes exactly; scoped mapping `RETURN` remains permitted to remove authority.
- [ ] Write RED tests showing technical approval creates no publication; a
  publication request without exact prior technical decision fails; published
  state creates no package until scope eligibility is separately evaluated.
  For joint ownership, require N current technical-decision rows, then exactly
  one publication command/row by one current required owner citing all N IDs;
  test replay, competing publishers, missing/duplicate/stale IDs, and every
  path/body mismatch with zero effects.
- [ ] Write RED tests that activate one successor source hash or supersede an
  accepted authority/mapping decision with `RETURN`, create one impact aggregate
  per exact typed trigger plus one idempotent link/gap-successor per affected
  candidate root, deny new packages from stale/returned versions, and
  preserve old published rows/in-progress Audit/question snapshots/package
  bytes byte-for-byte.
- [ ] Add the pre-v28 root → modern same-generation gap impact → attested
  modern successor → later impact proof. Reject legacy-state inheritance,
  synthesized owner/authority facts, and a successor that splices a different
  generation-run ID into the root.
- [ ] Run focused tests and record the expected missing/weak validation failures.
- [ ] Centralize ordered blocker computation and strengthen the existing
  manager services without creating new bypasses.
- [ ] Generalize source-currentness locks to candidate source-binding sets.
  Acquire the same deterministic source-identity locks and re-read currentness,
  impact, leaf, and digests inside source activation, authority/mapping
  attestation, technical approval, publication, and package transactions. Add
  barrier races for every pair and multi-source chains where one changed link
  makes the affected chain stale.
- [ ] Implement and run the allowlist-aware discriminator contract test. It may
  allow `generation_run_id` in explicit lineage persistence, generation-run
  variant projection, and pre-v28 compatibility code, but fails on SQL
  `generation_run_id IS [NOT] NULL` or Go `GenerationRunID` empty/nonempty
  predicates used as authority in queue, command, currentness, impact,
  eligibility, publication, or package symbols. The allowlist contains exact
  file/symbol purposes and rejects new unmatched occurrences; a raw negative
  `rg` is not the gate. A source-gap row with no binding is queue-visible and
  immutable but is never falsely treated as source-lockable.
- [ ] Reuse exact Task 6 impact idempotency/linkage and extend it for candidates
  created through both new entry paths.
- [ ] Run application/HTTP/integration tests with all roles and exact zero-side-
  effect assertions for denial branches.
- [ ] Run the consolidated transport-mapping test over every normative row's Go-
  domain → generated-Go/OpenAPI-JSON conversion. For each owning request and
  response direction, prove complete strict variants, generated JSON round trip,
  and rejection of missing, extra, nullable, private, or cross-variant fields.
  Combine this with Task 1's generated TypeScript shape tests; do not defer a
  domain/transport mismatch to Task 8 UI tests.

### Task 8 — Add exact mock/HTTP/React intake and review projections

**Status:** `verified locally` for candidate-only parity/build slices; live
browser/runtime dependencies remain `blocked`.

**Files**

- Modify `apps/web/src/backend/backend.ts`.
- Modify `apps/web/src/backend/backend-contracts.ts`.
- Modify `apps/web/src/backend/http-backend.ts` and tests.
- Modify `apps/web/vitest.http.config.ts`.
- Modify `apps/web/playwright.config.ts`.
- Modify `scripts/test-http-profile.sh` to add the focused
  `governed-checklist-intake` runtime selector.
- Modify `apps/web/src/mock/mock-engine.ts` and tests.
- Modify `apps/web/src/features/admin/checklist-builder-page.tsx`.
- Create `apps/web/src/features/admin/checklist-builder-page.test.tsx`.
- Create `apps/web/src/features/admin/checklist-intake-panel.tsx` and test.
- Create `apps/web/src/features/admin/checklist-candidate-review.tsx` and test.
- Create `apps/web/src/features/admin/checklist-draft-editor.tsx` and test.
- Create `apps/web/src/features/admin/checklist-reconciliation-diff.tsx` and test.
- Create `apps/web/src/features/admin/checklist-publication-blockers.tsx` and test.
- Create `apps/web/src/features/checklists/source-review-queue.tsx` and test.
- Create `apps/web/src/features/checklists/checklist-reviewer-queue.tsx` and test.
- Modify `apps/web/src/features/checklists/checklist-management-page.tsx` and tests.
- Create `apps/web/src/backend/governed-checklist-intake-parity.test.ts`.
- Extend `apps/web/src/backend/governed-checklist-http-parity.test.ts`.
- Create `apps/web/tests/e2e/governed-checklist-intake.spec.ts`.
- Create `apps/web/tests/e2e/governed-checklist-intake.http.spec.ts`.
- Create `scripts/check-governed-checklist-intake-cleanup.mjs` to inspect only
  task-owned PID/scratch/compose-project manifests and fail when any remains.

**Interfaces**

- Consumes generated Task 1 types and Task 3–7 services.
- Produces exact role-aware Checklist Builder/import-review UI and mock/HTTP
  parity with no auditee leakage.

- [ ] Write React/backend RED tests for both entry paths, origin/status badges,
  register/file receipts, immutable initial versus effective identity state,
  identity current-leaf correction, the paginated Admin-only extraction packet,
  exact boundary-decision state, candidate wording/history,
  every reconciliation field, source authority versus mapping/technical
  projections, assignment-scoped queues, source gaps, ordered blockers,
  immutable successors, and separate approval/publication/eligibility panels.
  Prove route/session/principal change clears raw packet state and that the HTTP
  path writes no packet text to localStorage, IndexedDB, service-worker caches,
  analytics, or error telemetry; mock fixtures remain synthetic-only.
- [ ] Prove every create/append backend method returns the strict
  `{view,replayed}` envelope in mock and HTTP; the first archive receipt renders
  `replayed=false`, identical replay renders `replayed=true`, and neither path
  performs a second lookup or creates a second object/attempt.
- [ ] Write parity RED tests comparing complete semantic mock and actual HTTP
  responses, including the extraction-review read, both scoped governance detail
  reads, missing/extra/private fields, guessed/cross-assignment IDs, and all role
  denials. Only the focused HTTP
  profile may claim live PostgreSQL-backed parity; offline adapter tests claim
  serialization only.
- [ ] Run focused tests; expect missing interfaces/components/routes.
- [ ] Implement backend methods and focused components. Use server-derived
  states/reasons and immutable artifact IDs; never simulate approval locally.
- [ ] Add the intake parity test to `vitest.http.config.ts` and both browser
  specs to the exact mock/HTTP Playwright `testMatch` arrays. Run `vitest list`
  and `playwright test --list` for the literal focused commands; freeze and
  assert nonzero discovered file/test counts before execution.
- [ ] Add auditee negative projection assertions for API, backend, navigation,
  search/counts, and rendered DOM.
- [ ] Run typecheck, focused Vitest, HTTP contract parity, demo/http builds, and
  root demo-boundary smoke tests.
- [ ] Run mock and HTTP Playwright scenarios at 1440×900 and 390×844 with an
  isolated browser profile. Assert keyboard/focus, no horizontal overflow,
  no console/page/request errors, and exact disabled reasons.
- [ ] Clean all task-owned Vite, Playwright, browser, worker, and compose
  processes before recording evidence.
- [ ] Run the cleanup assertion script after success and forced-failure browser
  runs; prove it detects a deliberately retained task-owned sentinel process
  without matching the user's ordinary browser or unrelated services.

### Task 9 — Gate and execute Phase 2 AGA expansion

**Status:** `blocked`; explicit continuation/expansion authorization was
received on 2026-08-01, but no additional form import is permitted until the
real Form 048 mechanism gate, current-Admin identity/boundary decisions, and
connected runtime prerequisites are evidenced.

**Files**

- Extend focused intake adapters/tests only for structurally distinct form
  families proven by the complete inventory.
- Create `apps/api/tests/integration/aga_candidate_expansion_test.go`.
- Extend the evidence package with metadata-only receipt summaries.

**Interfaces**

- Consumes the passed mechanism gate, the real current-Admin Form 048 identity
  and all 28 boundary decisions, the immutable real
  `EXISTING_CHECKLIST_CANDIDATE`, its visible real `SOURCE_MAPPING_REQUIRED`
  Draft, and explicit human expansion authorization. Synthetic evidence cannot
  satisfy this prerequisite; real regulatory mapping may remain `blocked`.
- Produces deterministic candidate receipts for selected additional forms;
  it does not produce bulk approved/published checklists.

- [ ] Confirm in evidence that Gate 6 and Tasks 7–8 passed, all parser/cleanup/
  parity/security tests are green, the exact real Form 048 candidate/source-gap
  Draft prerequisite above exists, and a named human authorized expansion. If
  any prerequisite is absent, record Task 9 `blocked` and import no other form.
- [ ] Group remaining forms by observed layout/parser behavior using immutable
  inventory facts. Add one RED fixture per distinct structure; do not assume
  Form 048 parsing applies to all forms.
- [ ] Import candidates in bounded batches with explicit per-file
  extraction/identity review. Stop the batch on any hard
  error; do not silently skip a form.
- [ ] Prove all 52 register/file identities remain accounted for and every
  candidate/rejected file has an explicit receipt. Do not require every form
  to become a candidate if its parse/identity review is unresolved.
- [ ] Keep every imported form `EXISTING_CHECKLIST_CANDIDATE`; require separate
  source-owner mapping, Draft, technical approval, and publication decisions
  per actual checklist scope.

### Task 10 — Final verification, evidence, and handoff

**Status:** `verified locally` for the raw-byte-free derived-text handoff and
local checks;
external dependencies remain `blocked`.

**Files**

- Create `docs/demo-evidence/GOVERNED_AGA_CHECKLIST_INTAKE_2026-07-31.md`.
- Update this plan's Progress, Decisions, Discoveries, Outcome Notes, and
  literal status.
- Update `docs/exec-plans/index.md` and
  `docs/exec-plans/tech-debt-tracker.md` to actual outcomes.
- Update `docs/agent-harness/registry.md` and
  `docs/agent-harness/verification-matrix.md` with the durable external-archive
  verifier, intake test inventory, and task-owned cleanup commands.

**Interfaces**

- Produces a raw-byte-free derived review package and precise real-owner
  handoff. The package may carry bounded parser-derived question strings and
  provenance, while raw ZIP/PDF bytes, page images, and full extracted-text
  dumps remain excluded.

- [ ] Run the verification inventory preflight; do not run aggregates if a
  required test is missing or reports zero cases.
- [ ] Run every command in the verification matrix below from a clean,
  task-owned runtime and record command, timestamp, exit status, suite/test
  counts, and literal evidence label.
- [ ] Record AGA archive/register/file hashes and receipt IDs in the evidence
  record; do not copy ZIP/PDF bytes, page images, or full extracted text into
  the evidence directory. Keep any user-authorized bounded question-string
  handoff separate from the metadata-only evidence record and explicitly
  candidate-only.
- [ ] Record every source-owner and Department Manager decision as `blocked`
  until an identified real actor supplies it. Do not convert absence into
  inferred approval.
- [ ] Verify only intentional files differ from the starting Git status; do
  not delete or alter pre-existing unrelated files.
- [ ] Run process/browser/container cleanup checks and `git diff --check`.
- [ ] Obtain a final read-only review for plan/spec compliance, authority,
  security, privacy, contract parity, and evidence wording before claiming the
  implementation task complete.

## Commands And Expected Observations

These commands are the execution and handoff contract. The results below are
recorded in Progress and the metadata-only evidence package. Use task-specific
cache/runtime paths; do not repurpose `HOME`.

### Test inventory and external AGA read-only inventory

```bash
node scripts/verify-governed-checklist-test-inventory.mjs --phase final
node --test tests/governed-checklist-intake-plan-contract.test.mjs
env AGA_CHECKLIST_ARCHIVE='/Users/marlonjd/Library/Mobile Documents/com~apple~CloudDocs/Downloads/AGA - Checklists and Form.zip' \
  node --test tests/aga-checklist-archive-inventory.test.mjs
```

Expected: inventory preflight lists every required non-zero suite. The AGA
test reads without extraction into the repository and reports archive SHA-256
`dd819cfa6a670760e0cfceed94496e2e466dc53bac13e6fd792b1128314d6e32`,
53 PDFs, one register, 52 forms, the exact register/Form 048 hashes, missing
049/present 035A, safe paths, and the Form 048 identity conflict. Any missing
external path is `blocked`, not skipped/passed.

### OpenAPI and generated transport synchronization

```bash
./scripts/generate-contracts.sh
./scripts/check-contracts.sh
node --test \
  api/openapi/tests/governed-checklist-intake-contract.test.mjs \
  api/openapi/tests/governed-checklist-authoring-contract.test.mjs \
  api/openapi/tests/regulatory-question-governance-contract.test.mjs \
  api/openapi/tests/governed-checklist-publication-boundary.test.mjs \
  api/openapi/tests/go-contract-generation.test.mjs
```

Expected: OpenAPI source/bundle/generated Go/generated TypeScript agree; strict
trace variants and all operations validate; no Admin publication or combined
approval/publication operation exists.

### Archive/parser/security unit tests

```bash
env GOCACHE=/private/tmp/avia-aga-intake-go-cache \
  go -C apps/api test \
  ./internal/checklistintake \
  ./internal/worker/checklistintake \
  ./internal/platform/scanner \
  ./internal/platform/uploadpolicy \
  ./internal/platform/objectstore \
  -count=1
node --test \
  tests/governed-checklist-intake-security.test.mjs \
  tests/local-compose-policy.test.mjs
./scripts/test-governed-checklist-intake-profile.sh --security-only
```

Expected: every fixed policy and error code has a positive/negative case;
malicious paths/ZIPs/PDFs fail closed; exact archive/PDF scans and replay are
deterministic; live compose proof shows the parser allows only AF_UNIX sockets
and has no AF_INET/AF_INET6/AF_NETLINK/AF_PACKET, DNS/TCP, secrets, Docker
socket, or writable root/input; owned temp/object cleanup passes
on every failpoint without touching an unrelated sentinel; no test writes into
the repository.

### Migration and PostgreSQL integration

```bash
env AGA_CHECKLIST_ARCHIVE='/Users/marlonjd/Library/Mobile Documents/com~apple~CloudDocs/Downloads/AGA - Checklists and Form.zip' \
  GOCACHE=/private/tmp/avia-aga-intake-go-cache \
  go -C apps/api test ./migrations ./tests/integration \
  -tags canonicaltest \
  -run 'Test(GovernedChecklistIntakeMigration|GovernedChecklistAssignmentAuthority|AGAForm048CandidateIntake|GovernedChecklistDualAuthoring|AGAForm048Reconciliation|GovernedChecklistIntakeLifecycle|AGACandidateExpansion)' \
  -count=1
```

Expected: fresh install and version-27 upgrade reach version 28; immutable
constraints, receipts, role denials, candidate/Draft lineage, reconciliation,
separate decisions, impact Draft, replay, rollback/forward repair, and
historical Audit preservation pass in a task-owned disposable database.

### Go application and HTTP boundaries

```bash
env GOCACHE=/private/tmp/avia-aga-intake-go-cache \
  go -C apps/api test \
  ./internal/regulatory \
  ./internal/checklistgovernance \
  ./internal/application \
  ./internal/httpapi \
  ./internal/identity \
  ./internal/inspections \
  ./cmd/api \
  ./cmd/worker \
  ./cmd/checklist-pdf-parser \
  -count=1
node --test tests/governed-checklist-discriminator-contract.test.mjs
```

Expected: exact capability/department/source scope is enforced server-side;
all fail-closed blockers create zero forbidden effects; published and pinned
history remains immutable. The allowlist-aware contract reports zero runtime
use of generation-run presence as governed authority while retaining every
explicit lineage/compatibility occurrence.

### React, mock/HTTP parity, builds, and legacy boundary

```bash
npm --prefix apps/web run typecheck
npm --prefix apps/web test -- \
  src/features/admin/checklist-builder-page.test.tsx \
  src/features/admin/checklist-intake-panel.test.tsx \
  src/features/admin/checklist-candidate-review.test.tsx \
  src/features/admin/checklist-draft-editor.test.tsx \
  src/features/admin/checklist-reconciliation-diff.test.tsx \
  src/features/admin/checklist-publication-blockers.test.tsx \
  src/features/checklists/source-review-queue.test.tsx \
  src/features/checklists/checklist-reviewer-queue.test.tsx \
  src/features/checklists/checklist-management-page.test.tsx \
  src/backend/governed-checklist-intake-parity.test.ts \
  src/backend/regulatory-question-governance-parity.test.ts
npm --prefix apps/web run test:contract:http -- \
  src/backend/governed-checklist-http-parity.test.ts \
  src/backend/governed-checklist-intake-parity.test.ts
npm --prefix apps/web run build:demo
npm --prefix apps/web run build:http
node apps/web/scripts/assert-http-artifact.mjs apps/web/dist/http
node --test \
  tests/governed-checklist-lifecycle-smoke.test.mjs \
  tests/checklist-management-smoke.test.js \
  tests/checklist-approval-smoke.test.js \
  tests/manager-checklist-management-smoke.test.js \
  tests/demo-boundary-smoke.test.js
```

Expected: UI exposes both distinct paths and exact blockers; mock and HTTP
adapter serialization shapes are equal; this standalone Vitest command does
not claim a live API. Auditee privacy holds, demo/http artifacts are separate,
and the root legacy oracle is unchanged. Live PostgreSQL-backed HTTP parity is
proved only by the focused profile below.

### Browser scenarios and cleanup

```bash
cd apps/web
npm exec vitest -- list \
  --config vitest.http.config.ts \
  src/backend/governed-checklist-intake-parity.test.ts
npm exec playwright -- test \
  --list --project=mock tests/e2e/governed-checklist-intake.spec.ts
npm exec playwright -- test \
  --list --project=http tests/e2e/governed-checklist-intake.http.spec.ts
cd ../..
npm --prefix apps/web run test:e2e:mock -- \
  tests/e2e/governed-checklist-intake.spec.ts
AVIA_HTTP_PROFILE_FOCUSED_E2E=governed-checklist-intake \
  ./scripts/test-http-profile.sh
node scripts/check-governed-checklist-intake-cleanup.mjs --assert-clean
```

Expected: synthetic browser journeys pass at 1440×900 and 390×844 with an
isolated profile, no privacy/authority leak, console/request failure, or
overflow. The focused HTTP profile starts its task-owned API/database, requires
the reset endpoint, asserts nonzero live intake contract and browser cases, and
executes every role denial; a missing runtime is failure, not mock fallback.
After task-owned cleanup, the assertion finds no task-owned process,
PID manifest, scratch directory, browser profile, or compose project, and
compose reports only explicitly retained baseline services. Do not kill or
classify the user's ordinary browser or unrelated processes as task-owned.

### Final repository gates

```bash
node tests/harness-docs-smoke.test.js
git diff --check
if rg -n '[[:blank:]]+$' \
  docs/exec-plans/active/2026-07-31-governed-aga-checklist-intake-and-official-source-authoring-plan.md; then
  exit 1
fi
git status --short
```

Expected: documentation links/commands are valid, no whitespace error exists,
the explicit `rg` check prints no match for the plan even if it is untracked,
and status contains only intentional implementation/evidence changes plus the
known pre-existing unrelated files. `git diff --check` alone is not workspace
integrity evidence.

## Detailed Acceptance And Verification Matrix

| Requirement | Required proof | Surfaces | Acceptance |
| --- | --- | --- | --- |
| AGA archive/register inventory | External path-driven hash/ZIP/register test plus persisted batch/file/receipt integration | Node, Go worker, PostgreSQL, Admin UI | Exact archive/register/Form 048 hashes; 53 PDFs; 52 forms; missing 049; present 035A; all entries accounted; no repo extraction. |
| Deterministic candidate import/replay | Same command twice, divergent semantic replay, fenced retry chains, reordered DB queries, competing finalizers, and crash before/after every phase/cleanup/terminal boundary | Go, PostgreSQL, object store, HTTP, mock | Same operation and `AGA_IMPORT_MANIFEST_V1` golden vector return identical bytes/digest/receipt/object/candidate; divergence is 409; one CAS manifest selects deterministic fenced winners only after cleanup; zero duplicate/partial rows. |
| Identity and extraction review lifecycle | Initial conflict/match, atomic parser-output-to-packet preparation, first resolution, concurrent/stale candidate correction, every strict inner decision, paginated packet, and corrected re-import | OpenAPI, PostgreSQL, object store, Go, HTTP, mock, React | Terminal file/manifest facts never change; packet children are all-or-zero; effective identity and candidate/resolution/decision leaves are deterministic; only exact current leaves and complete packet coverage create a new immutable candidate; superseded candidate blocks new effects; raw text remains Admin-only and within numeric limits. |
| Object ownership and cleanup | Two batches with same archive/parser-output hash, wrong version/tag/checksum, expired intents, cleanup success/failure/exhaustion, and cleanup restart | Object store, Go, PostgreSQL, compose | Each archive and parser output owns a unique exact version; expiring one never removes another receipt's readable unchanged bytes; successful parse binds a durable exact output before scratch cleanup; untrusted/mismatched metadata cannot finalize; unconditional fenced cleanup closes before terminal CAS; no mutable-latest reference, post-terminal unreceipted delete, or unrelated deletion. |
| Immutable new Draft/candidate | Existing-candidate correction roots plus pre-v28 backfill and truth-table insert, update/delete, queue, load, command, validation-claim/correction race, and concurrent successor attempts for every lineage | DB constraints, Go, HTTP | Existing-candidate root/revision/currentness is exact and superseded input blocks new effects, including stale validation claims; `entry_path` governs Draft protection; pre-v28 rows expose only `PRE_V28_UNATTESTED` with no synthesized authority/effect; earlier candidates/Drafts/validation rows stay unchanged; only exact current leaf gains a successor. |
| Explicit source-gap question | Existing candidate Draft without real mapping | OpenAPI, Go, React | Literal `SOURCE_MAPPING_REQUIRED` plus missing fields; no empty citations; validation/approval/publication/package denied. |
| Official-source-only creation | Synthetic authority-accepted/current regulatory plus controlled-procedure chain, activation barrier race, and no AGA/generation-run lineage | Go, PostgreSQL, HTTP, mock, React | Origin `REGULATORY_TRACE`; complete ordered chain and owners server-resolved under locks; gaps/unapproved/stale links create no Draft; source binding/digests are complete and `generation_run_id` is null. |
| Hybrid reconciliation differences | Change every mandatory field and split/merge/exclusion/unmapped case plus source-activation and candidate-correction barrier races | Go, PostgreSQL, contracts, HTTP/mock, React | New immutable `HYBRID_RECONCILED` Draft shows exact before/after/digests/outcome/reason for all content, applicability, scope, chain, intent, and history fields; no stale resolved binding commits and no superseded candidate/Draft creates a hybrid successor. |
| Attestation lifecycle and scope | Initial/return/accept/transfer/concurrent successor/replay with multi-link different owners and one full reviewed set | PostgreSQL, Go, HTTP, mock, React | Deterministic current leaf only; per-link authority and complete-set mapping remain separate; partial/overlap/cross-scope fails closed; return triggers append-only impact and historical bytes remain unchanged. |
| Stale source to impact-review Draft | Activate successor version/hash after publication and race every effect command | Go, PostgreSQL, HTTP, React | One aggregate per event plus N idempotent root links/successors; transaction locks deny stale new effects; no silent mutation or reused decision. |
| Preserve published/in-progress history | Byte/digest compare before and after source refresh/reconciliation | PostgreSQL integration | Old published version, decisions, Audit, pinned questions, answers/findings, and package bytes are identical. |
| Separate technical approval/publication | Execute each command independently, joint-owner N-to-one publication, and attempt skips | Application, PostgreSQL, HTTP, mock, React | N owner technical decisions create no publication; one current required owner cites all N IDs to create exactly one publication row; UI has separate states/actions. |
| Publication fail-closed gates | Table-driven missing trace/scope/currentness/applicability/rationale/review/guardrail cases | OpenAPI, Go, HTTP, mock | Every case returns exact blocker, creates no publication/version/package, and matches across transports. |
| Automatic deferral guardrails | Mandatory/safety/unknown/insufficient/non-comparable/open/repeat/overdue/source-changed cases | Go, mock, React | `automaticDeferral` always false; invalid `DEFER_ELIGIBLE` cannot validate/approve/publish/materialize. |
| Functional assignment and role denials | Source owner, current manager, manager without assignment, Admin, reviewer, Inspector, Auditee; raw extraction read, queue/detail/evaluation/materialization/execution | Go auth, PostgreSQL, HTTP, mock, React | Only Admin reads raw extraction proposals or resolves/imports; global-latest assignment resolution and null-scope fail closed; guessed/cross-scope detail is 404; no real grant/self-grant; reviewer is non-binding; scoped manager materializes while separate Inspector/Lead executes. |
| Mock/HTTP/Go/React exact parity | Canonical complete responses, command replay envelopes, negative responses, and live focused profile | Generated types, Go handler, HTTP backend, mock, React | Equal identifiers, enums, ordering, blocker codes, `{view,replayed}`, missing/extra/private-field rejection, role denials, and side effects; offline adapter tests are not mislabeled live. |
| Auditee privacy projection | Direct route, raw extraction, list/count/search/navigation/notification/DOM tests | Application, HTTP, mock, React | No inventory/extraction/identity/candidate/source/review/deliberation data; only already pinned organization-scoped Audit projection. |
| Demo boundary | Root artifact hashes/smoke plus demo/http build separation | Root Node tests, Vite builds | Root legacy oracle unchanged; no AGA bytes/state in root or HTTP artifact seed. |
| Generated contract synchronization | Writer generator, drift checker, strict trace/lineage/review-detail/identity/extraction-decision unions and multipart generation tests, runner discovery | OpenAPI, Go, TypeScript | No drift or `json.RawMessage`; prohibited lineage/partial-current/private-field combinations fail; exact multipart; extraction GET/signatures stay synchronized; all planned Vitest/Playwright files are discovered; no direct Admin publication route. |
| Parser/import errors | Malicious structures/PDFs, all-directory archive, register first/middle/last or missing/duplicate, frozen clamd limits, per-file scan, seccomp socket probe, timeout/kill, cleanup and object/DB failpoints | Go unit/integration, compose | All directory rows count/normalize safely; no identity match precedes the register receipt; explicit attempt/terminal/cleanup receipts; parser syscall policy has no AF_INET/AF_INET6 or secrets; no silent partial effect; cleanup succeeds before completion or exhausts into immutable failure. |
| Real-owner handoff | Form 048 pending Admin identity/boundary packet, real gap slice, source chain, assignment governance, Department decisions | Go, PostgreSQL, evidence | Synthetic fixtures cannot promote; no Phase 2 expansion before the real candidate/gap Draft; every other missing real decision remains `blocked` with zero unauthorized mapping/approval/publication effect. |
| Process/browser/test cleanup | Isolated-profile run and ownership-filtered process/container inspection | Playwright, Vite, worker, compose | No task-owned process or scratch artifact remains; unrelated user processes untouched. |
| Diff cleanliness | Harness docs, tracked diff, explicit plan-whitespace check, and Git status | Repository | `git diff --check` passes; the untracked-plan check has no match; final status matches intentional scope plus known pre-existing files. |

## Execution Verification Status

The plan is now executing under separate user authorization. Before handoff,
run:

```bash
node tests/harness-docs-smoke.test.js
git diff --check
if rg -n '[[:blank:]]+$' \
  docs/exec-plans/active/2026-07-31-governed-aga-checklist-intake-and-official-source-authoring-plan.md; then
  exit 1
fi
git status --short
```

Record these literally:

- Plan/index/tracker/harness documentation checks: `verified locally`.
- Gate 0 and Tasks 1–8 local contract/mechanism slices: `verified locally`.
- Task 9 real Form 048 mechanism and Phase 2 expansion: `blocked`.
- Real AGA source mapping, official CAA-procedure identity/currentness/
  applicability, assignment provisioning, and responsible Department Manager
  validation: `blocked`.
- Connected runtime, recovery, browser, and external-owner evidence:
  `blocked`.
- Product: `candidate-only`.
- Release: `release pending`.
- Production-ready: not established.

## Risks And Mitigations

1. **PDF/ZIP parser exploitation or resource exhaustion.** Mitigate with
   streaming limits, frozen raw/normalized ZIP structure rules, archive plus
   exact-PDF malware scans, no shell extraction, AF_UNIX-only non-root/no-
   network parser, exact time/memory/page/text/output bounds, explicit attempt
   receipts, and intent/object/scratch reconciliation.
2. **Silent partial import.** Mitigate with terminal batch receipts, per-file
   receipts, atomic candidate creation, immutable intake-safety/initial-file
   state, and derived candidate eligibility only after all hard checks plus any
   required current identity/extraction decisions.
3. **Legacy wording gains authority by proximity.** Mitigate with distinct
   import aggregate, origin badge, strict source union, source-owner
   attestation, reconciliation diff, and publication/package gates.
4. **Role creep or Admin bypass.** Mitigate by preserving top-level roles,
   scoped latest-successor functional assignments, no real provisioning route
   until governance decides it, server-derived owners, separate manager
   commands, direct-ID denial tests, and zero-effect assertions.
5. **Synthetic mappings represented as real.** Mitigate with test-profile-only
   fixtures, explicit synthetic labels, external real-source decision points,
   and evidence language that remains `blocked`/`candidate-only`.
6. **Source changes invalidate future use but corrupt history.** Mitigate with
   generalized candidate source bindings, shared transaction locks, one impact
   aggregate per event, computed new-package eligibility, race tests, and byte-
   level preservation tests for old/in-progress Audits.
7. **Mock accepts more than HTTP.** Mitigate with shared contract vocabulary,
   canonical fixtures, complete response equality, matching denial side
   effects, and HTTP browser coverage.
8. **General parser assumptions fail across 52 forms.** Mitigate with one
   representative vertical slice, structure-family tests, explicit expansion
   gate, per-form receipts, and no silent skips.
9. **Private candidate data leaks to auditees.** Mitigate with projection
   allowlists, no auditee routes, list/count/search/notification tests, and
   internal-only comment visibility.
10. **Local tooling mistaken for production readiness.** Mitigate with literal
    evidence labels, task-owned local services, no external upload/deploy, and
    explicit release/security/operations dependencies.
11. **Nullable non-generation lineage bypasses accepted guards.** Mitigate with
    the closed lineage truth table, `entry_path` governed discriminator,
    generalized queues/commands/currentness locks, version-27 preservation, and
    per-lineage mutation/visibility tests.

## Dependencies And Real-Owner Decision Points

Mechanism implementation may proceed with synthetic fixtures after separate
authorization. These real outcomes cannot be invented and remain `blocked`
until supplied:

1. **AGA identity/extraction review:** a current authenticated Admin principal
   must confirm Form 048's selected human-readable identity and all 28 question
   boundaries, wording, guidance/operational intent, and whether any result
   history is actually present. No Department or functional assignment
   substitutes for Admin on this intake decision.
2. **Functional-assignment provisioning governance:** a named governance owner
   must decide the real reviewed-source-set creation/versioning actor, set
   authority evidence, assignment grant/revoke actor, eligible internal-CAA
   membership, and anti-self-authorization rule. Until then no real set or
   grant/revoke API exists and only canonical synthetic fixtures may seed sets
   and assignments.
3. **Official source chain:** each exact link's assigned source owner/regulatory
   curator must identify and attest the current approved regulatory sources and
   controlled CAA surveillance procedure(s) applicable to Form 048, including
   exact identity/version/`sha256:` digest/locator/page/section/clause, chain
   role, source-authority acceptance, currentness, and supersession. One
   complete `REVIEWED_SOURCE_SET` assignee must separately attest the candidate
   mapping; partial link ownership cannot do so.
4. **Applicability and expected Evidence:** the source owner and responsible
   Department Manager must confirm applicability, verification objectives,
   expected Evidence, and question decomposition. The implementation team must
   not infer them from AGA wording.
5. **Scope and history:** the responsible Department Manager must confirm each
   scope classification, operational-history comparability, mandatory/safety
   guardrails, and any defer recommendation.
6. **Technical review:** every current required Department Manager must record
   the separate technical decision against the exact immutable Draft digest.
7. **Publication:** after every required current Department Manager has made a
   technical decision, one current member of that required-owner set may invoke
   the single separate publication command citing the complete technical-
   decision set. Technical approval does not imply publication.
8. **Phase 2 expansion:** a named product/source owner must authorize expansion
   only after the mechanism/safety gates and the real Admin-decided Form 048
   candidate plus visible source-gap Draft exist. This does not authorize bulk
   source mapping or publication.
9. **Release/production:** security review, parser/container hardening,
   operational monitoring, backup/recovery, privacy verification, coordinated
   deployment, and real environment evidence remain outside local acceptance.

## Idempotence, Recovery, And Rollback

- All immutable database content is digest-bound and all commands carry
  semantic digests; Phase 1 quarantine object keys remain unique per intent,
  not cross-batch content aliases. Replay of a successful or terminal-failed
  command returns its prior result and creates no new attempt; it never
  duplicates a candidate, decision, version, package, or terminal batch manifest.
- Every internal/external phase uses an immutable leased attempt; scanner/parser
  calls remain at-least-once external execution. Exactly one terminal event may
  reference each attempt. A crash before that event causes an `ABANDONED`
  terminal event after lease expiry and a linked retry; all attempts remain
  readable. Attempt completion never writes the batch manifest. After all
  ordinary nodes close or become explicitly not run, the worker completes the
  unconditional fenced cleanup chain. Only then may one CAS finalization
  transaction select winning terminal events, persist all accepted file/
  register/match facts plus one batch manifest/status, and establish immutable
  intake-safety/initial-file states. A conflict may still require a derived
  current identity resolution before candidate import.
  Cleanup retry exhaustion finalizes failure; a crash after terminal commit
  replays the receipt without rerunning dependencies.
- PostgreSQL plus object storage is a recoverable intent/finalization workflow,
  not a cross-system atomic transaction. The database commits an object intent
  before conditional immutable PUT to a unique intent key; that PUT carries all
  ownership tags and checksum request fields atomically. Exact object-version/
  tag/byte/trusted-checksum HEAD verification—or reopening and rehashing that
  exact version—precedes the transaction that marks the intent `VERIFIED` and
  permits downstream scanning; it never establishes candidate eligibility. A
  lost PUT response is recovered only by exact-key
  version enumeration and unique full-fact match. Database rows never point to
  mutable “latest” objects.
- Recovery covers failpoints after intent commit, PUT, HEAD verification,
  ordinary attempt execution, cleanup start/event, terminal transaction start,
  and terminal commit, in that order.
  An expired-intent reconciler may finalize or remove only the exact version
  carrying this service's intent/batch/hash/policy tags and no verified receipt
  reference; it must preserve a same-hash object owned by another batch plus all
  unrelated or unknown objects and report every unresolved intent.
- Startup recovery examines only signed task-owned scratch markers, service-
  tagged object intents, and expired leases. Scratch recovery resumes or links
  the same fenced `SCRATCH_CLEANUP` chain before finalization; it never performs
  an unreceipted post-terminal delete. Manual recovery reports attempts, intents,
  objects, receipts, and candidates by exact IDs and never deletes an unknown
  path/object.
- Migration 28 is forward-only after data exists. A corrective migration must
  preserve all receipts/candidates/decisions. Do not roll back by dropping
  history tables.
- A bad candidate extraction is corrected by a new extraction decision and
  new candidate/Draft lineage, not an UPDATE.
- A stale or bad source mapping is returned and superseded by a new Draft and
  attestation, not edited in place.
- Source-currentness activation, official/hybrid source-binding commits,
  attestation successors, and every later effect-writing consumer share the
  same deterministic transaction-scoped source locks. Recovery never treats a
  pre-lock eligibility read or historical acceptance as authority to bind,
  approve, publish, or materialize.
- A bad publication remains historically attributable. Correct through a new
  approved/published version and existing withdrawal/supersession semantics;
  never rewrite in-progress Audit bytes.

## Progress

- [x] 2026-07-31: Read the repository planning contract, architecture, active
  source-refresh plan, completed governed-generation predecessor, Task 6
  evidence/review packages, verification/output contracts, relevant product
  specifications, tracker, and Git state.
- [x] 2026-07-31: Safely inventoried the external archive in task-owned
  temporary storage, recorded planning hashes/counts/identity conflict, and
  verified cleanup without importing source files into the repository.
- [x] 2026-07-31: Presented three alternatives and obtained user approval for
  Alternative A, the register-first vertical-slice design.
- [x] 2026-07-31: Authored the initial plan and synchronized the plan
  index/tracker.
- [x] 2026-07-31: `node tests/harness-docs-smoke.test.js` and
  tracked-file `git diff --check` passed; status retained only the three
  intentional planning artifacts plus the recorded pre-existing unrelated
  untracked files. The plan itself was untracked and was not covered by that
  ordinary diff check.
- [x] 2026-07-31: Strict read-only review returned `REVISE` for owner
  derivation, official-source authority, nullable lineage, parser isolation,
  assignment/queue/attestation semantics, currentness races, contract
  generation, test discovery, and real-owner stop points.
- [x] 2026-07-31: User authorized correction of those plan defects; the plan
  now incorporates the closed authority/source/lineage/parser/recovery/
  contract/verification decisions recorded below.
- [x] 2026-07-31: Corrected-plan documentation checks passed, the explicit
  untracked-plan whitespace and Markdown-fence checks passed, and three focused
  independent read-only reviews found no remaining Critical, Important, or Minor
  authority/privacy, security/lifecycle, or contract/test finding.
- [x] 2026-07-31: Gate 0 RED recorded with
  `AGA_CHECKLIST_ARCHIVE='…/AGA - Checklists and Form.zip' node --test
  tests/governed-checklist-intake-plan-contract.test.mjs
  tests/governed-checklist-intake-security.test.mjs
  tests/aga-checklist-archive-inventory.test.mjs`: exit 1, 5 tests total,
  1 pass and 4 expected contract/inventory failures. The read-only external
  archive verifier already passed its inventory assertion and wrote nothing.
- [x] 2026-07-31: Gate 0 GREEN recorded with the same path-driven Node command:
  exit 0, 5/5 tests passed. `node
  scripts/verify-governed-checklist-test-inventory.mjs --phase gate0` passed
  with 11 required artifacts. `node tests/harness-docs-smoke.test.js` and
  `git diff --check` passed. The external archive was streamed only through
  `AGA_CHECKLIST_ARCHIVE`; no ZIP/PDF/text was copied into the repository.
- [x] 2026-07-31: Independent read-only review round 1 returned `CHANGES
  REQUIRED` for the missing aggregate expansion assertion and incorrect NUL
  path literal. TDD RED/GREEN repair added the aggregate <=20:1 check and
  actual-NUL assertion.
- [x] 2026-07-31: Independent read-only Gate 0 review round 2 returned
  `CHANGES REQUIRED` for rejecting only double backslashes. TDD RED/GREEN
  repair added focused single-backslash, normal-path, and NUL assertions and
  rejected every backslash.
- [x] 2026-07-31: Final independent read-only Gate 0 review returned
  `APPROVED` with no Critical, Important, or Minor finding. It confirmed the
  eight-role and authority/privacy boundaries, server-derived ownership,
  blocked functional-assignment provisioning, separate publication/eligibility
  facts, no Task 1/runtime/OpenAPI/migration work, and the read-only archive
  verifier's complete security/limit checks.
- [x] 2026-07-31: Final verification rerun passed the Gate-0 Node suite 5/5,
  archive path test 1/1, inventory gate0 (11 artifacts), harness smoke,
  syntax, tracked diff check, and explicit untracked plan/document whitespace
  check. Fresh `git status --short` retained only the recorded Gate 0 changes
  plus the pre-existing unrelated untracked workspace artifacts.
- [x] 2026-07-31: Gate-0 inventory tightening recorded RED in the security test
  (1/2 passed, 1 blocked-state assertion failed), then GREEN with the explicit
  `task9` external real-slice/authorization blocker. The blocker exits 2 only
  after all Task-9 artifacts exist; missing future artifacts remain exit 1 and
  never pass or skip. Final inventory still performs Vitest/Playwright runner
  discovery before reporting that blocker.
- [x] 2026-07-31: Final independent read-only inventory-tightening review
  returned `APPROVED`; no role, authority, privacy, publication, evidence-label,
  Task 1, runtime, OpenAPI, or migration regression was found.
- [x] 2026-07-31: Post-review verification rerun passed the complete Gate-0
  command set: syntax, 5/5 Gate-0 tests, archive path 1/1, inventory gate0
  (11 artifacts), harness smoke, tracked diff check, untracked plan/document
  whitespace check, and fresh Git status.
- [x] 2026-07-31: The user separately authorized execution of the complete
  plan; no branch/worktree, stage, commit, push, deployment, external upload, or
  source-byte import was performed.
- [x] 2026-07-31: Tasks 1–8 completed their recorded contract/domain/UI test
  and implementation checks. The smallest local candidate implementation was
  verified for each owned slice.
  Later-task tests remained phased in the inventory until their owning slice
  existed.
- [x] 2026-07-31: Task 1 GREEN: the focused governed OpenAPI/generated suite
  passed `9/9`; `./scripts/check-contracts.sh` passed `16/16` contract tests
  with generated drift clean; and `node scripts/lint-openapi.mjs` passed.
- [x] 2026-07-31: Task 2 GREEN: the focused Go checklist-intake, identity,
  regulatory, governance, HTTP, worker, and parser packages passed. Migration
  and recovery tests are present, but live PostgreSQL execution is `blocked`
  by the unavailable task-owned database/listen capability.
- [x] 2026-07-31: Task 3 GREEN: bounded policy/archive/parser/worker/HTTP tests
  passed; `./scripts/test-governed-checklist-intake-profile.sh --security-only`
  passed the Task-3 inventory (`18` artifacts), `23/23` security/policy tests,
  and cleanup. Live MinIO/ClamAV/Poppler qualification remains `blocked`.
- [x] 2026-07-31: Task 4 GREEN: extraction-review, identity-resolution,
  replay, and synthetic Form-048-shaped mechanism tests passed. The real
  authenticated Admin identity and extraction decisions remain `blocked` and
  no real candidate was created.
- [x] 2026-07-31: Tasks 5–7 GREEN: synthetic dual-authoring, strict trace,
  reconciliation, source-attestation, review/comment, publication, eligibility,
  discriminator, and transport-boundary tests passed. No source-owner,
  reviewed-source-set, applicability, technical-review, or publication fact
  was invented.
- [x] 2026-07-31: Task 8 GREEN: the exact React/mock parity command passed `11`
  files and `24/24` tests; HTTP parity passed `2` files and `5/5` tests; typecheck,
  demo/http builds, artifact scan (`148` files / `179` inputs), and root legacy
  smoke (`5/5`) passed. Live browser/runtime scenarios remain `blocked` or
  intentionally skipped at the external boundary.
- [x] 2026-07-31: Task 9 was evaluated and intentionally stopped with no
  additional form import. `node scripts/verify-governed-checklist-test-inventory.mjs
  --phase task9` exited `2` with the explicit real Form 048 mechanism and
  Phase 2 expansion-authorization blocker; the candidate-expansion test is
  skipped with the same literal blocker and has zero lifecycle effects.
- [x] 2026-07-31: The read-only archive verifier was rerun against the actual
  external `AGA_CHECKLIST_ARCHIVE` path. It returned the expected archive
  hash/size and `53` entries without extraction; the path-driven Gate-0 suite
  passed the expanded `6/6` green suite. The metadata-only evidence file records
  hashes/counts only and no receipt ID was emitted because no connected intake
  ran.
- [x] 2026-07-31: Task-10 inventory and handoff checks were run. Phase `gate0`
  passed `11` artifacts, phase `task8` passed `42` artifacts, and phase
  `final` reached runner discovery for Vitest and Playwright before exiting
  `2` for the explicit Task-9 blocker. Harness smoke, cleanup, syntax, and
  contract checks passed; all remaining external dependencies are `blocked`.
- [x] 2026-07-31: The exact aggregate migration selector exited `0` with six
  mechanism tests passed and the explicit expansion test skipped as `blocked`.
  The broader Go and scanner aggregates reached unrelated existing
  `httptest`/fake-clamd listener failures (`operation not permitted` in this
  sandbox); governed focused packages remained green and live runtime
  qualification is recorded as `blocked`, not as a product failure or pass.
- [x] 2026-07-31: Vitest/Playwright runner discovery was rerun from the
  `apps/web` working directory and passed with two Vitest cases plus one mock
  and one HTTP Playwright case. No browser was launched, so no task-owned GUI
  process or profile was created.
- [x] 2026-07-31: Final documentation checks passed: harness smoke, tracked
  `git diff --check`, explicit whitespace scan over the two untracked plan/
  evidence documents, syntax checks, and cleanup assertion all exited `0`.
- [x] 2026-07-31: Independent final review initially returned `CHANGES
  REQUIRED` for two important boundaries. TDD RED/GREEN repair removed the
  caller-controlled source-authority boolean: official Draft creation now
  requires a server-side attestation resolver and exact accepted decision
  identity/version/hash/chain-role binding. The repair also added strict raw
  ZIP preflight for ZIP64 sentinels/records, multi-disk archives, local/central
  name/flag/method/CRC/size equality, byte-range overlap, and trailing/gap
  bytes, with focused malformed-archive tests.
- [x] 2026-07-31: Intake receive now remains `PROCESSING` with an immutable
  `ARCHIVE_VALIDATE` receipt until scan/parser/finalizer phases exist; it no
  longer marks an archive inventory complete or safety-eligible immediately.
  Admin receipt listing exposes that phase receipt, and synthetic fixtures use
  raw no-descriptor ZIP entries so the strict boundary is exercised.
- [x] 2026-07-31: The repaired focused Go intake/worker/regulatory/HTTP suite
  passed all listed packages, the migration/integration selector passed six
  mechanism tests with one explicit Task-9 skip, and the new discriminator
  contract passed `1/1`. The required independent read-only reviews are
  recorded below; the final post-repair verdict is `APPROVED`.
- [x] 2026-07-31: The second independent read-only final review returned
  `APPROVED` with no remaining Critical, Important, or Minor finding. It
  confirmed resolver-bound source attestations, strict raw ZIP validation,
  `PROCESSING` plus the sole `ARCHIVE_VALIDATE` receipt, preserved eight-role
  and Auditee boundaries, literal external blockers, and the intended final
  inventory exit `2` after runner discovery.
- [x] 2026-07-31: The final web parity repair kept server-derived mapping review
  projections out of immutable candidate/import/edit digests (including the
  nullable predecessor normalization required for Go parity), kept returned
  candidates out of the active UI queue while preserving terminal detail, and
  updated the Task 6 artifact projection contract. Fresh exact React parity
  passed `11` files/`24` tests, Task 5/6 semantic parity passed `3/3` and
  `16/16`, and HTTP parity remained `2` files/`5` tests.
- [x] 2026-07-31: TDD follow-up for canonical projection invariance first ran
  `npm --prefix apps/web test -- --run
  src/backend/regulatory-question-governance-parity.test.ts -t
  'excludes server-derived|literal source-gap'` with exit `1` (`3` failed,
  `6` skipped), then restored the smallest projection scrub and reran the full
  file with exit `0` (`9/9` passed). The expanded assertions cover candidate,
  import, and edit digests for both mapping-review projections and the literal
  `SOURCE_MAPPING_REQUIRED`/`NOT_AVAILABLE` technical projection.
- [x] 2026-07-31: The final independent read-only review of the post-repair
  digest/UI changes returned `APPROVED` with no Critical, Important, or Minor
  finding. It confirmed candidate/import/edit and source-gap digest invariance,
  the authority-neutral queue/projection decision, and no regression to the
  strict archive/process/receipt boundary.
- [x] 2026-08-01: The user explicitly authorized continuation into Task 9 and
  Phase 2. The fresh archive verifier remained metadata-only. A current-worktree
  task-owned full profile reached API readiness with PostgreSQL migration `28`,
  healthy API/worker/scheduler/PostgreSQL/MinIO/ClamAV/Gotenberg services, and
  the lifecycle-created local Admin harness identity. Connected synthetic
  mechanism integration tests passed `4/4`, but the Task-9 inventory still
  exited `2` and
  `TestAGACandidateExpansion` remained an explicit `blocked` skip because the
  real Form 048 Admin identity/28 boundary packet, immutable
  candidate/source-gap Draft, and named expansion authorization are absent.
  No additional form was imported and no real-archive source bytes or receipt
  IDs were created or recorded.
- [x] 2026-08-01: Continuation verification completed: Gate-0 inventory passed
  with `11` artifacts; Gate-0 contract/security/discriminator tests passed
  `5/5`; the path-driven archive verifier passed `1/1`; connected synthetic
  migration-28 integration passed `4/4`; governed React passed `10/10`; HTTP
  parity passed `2/2`; security policy passed `23/23`; typecheck, both web
  builds, OpenAPI generation/lint, harness smoke, diff/whitespace, and the
  task-owned cleanup assertion passed. Task-9 and final inventory both exited
  `2` only for the explicit real Form 048 Admin packet/candidate/source-gap
  Draft/named expansion authorization blocker. The disposable runtime and all
  task-owned temporary state were removed; no real-archive bytes or receipt IDs
  were created or recorded.
- [x] 2026-08-01: Independent read-only continuation review returned
  `APPROVED` with no Critical, Important, or Minor finding. It confirmed the
  metadata-only archive boundary, no real AGA import/source bytes/receipt IDs,
  the exact Task-9 blocker, and intact `candidate-only`, `release pending`, and
  `production-ready: not established` labels.
- [x] 2026-08-01: The user selected the long-term controlled-intake option:
  the exact external ZIP may be streamed by the Admin-only local intake service
  through the existing `AGA_ZIP_PDF_V1` boundary, without repository extraction,
  source-byte copying, source-authority decisions, or publication effects. A
  TDD RED run first failed because the API defaulted to an in-memory intake
  store; the smallest fix bound the default to the append-only PostgreSQL store
  whenever a pool is configured, and the focused constructor test then passed.
  The focused HTTP-profile token override was likewise driven RED then GREEN so
  the disposable harness could use a known test token without weakening the
  random default.
- [x] 2026-08-01: A fresh task-owned PostgreSQL/MinIO/ClamAV candidate runtime
  streamed the supplied archive to
  `POST /v1/admin/governed-checklist/import-batches` as server-owned
  `USR-ADMIN-ADA`. The response was HTTP `201`, batch
  `import-4affd478609b4bd8`, observed/expected archive SHA
  `sha256:dd819cfa6a670760e0cfceed94496e2e466dc53bac13e6fd792b1128314d6e32`,
  manifest digest
  `sha256:0e40137deddf57b01254a164b4529400eff3f434c07050c478be628e69264fec`,
  status `PROCESSING`, and receipt
  `import-4affd478609b4bd8-archive-validate` with `53` entries / `53` PDFs
  under `AGA_ZIP_PDF_V1`. The controlled profile exited `0` and cleaned all
  task-owned runtime state. No file extraction, parser output, candidate,
  source-gap Draft, or additional-form import was performed; the real Form 048
  Admin identity/28 decisions, immutable candidate/source-gap Draft, and named
  expansion authorization remain `blocked`.
- [x] 2026-08-01: The bounded-intake hardening loop was completed with fresh
  TDD evidence. New boundary tests first ran RED (non-CAA Admin body read,
  duplicate/missing multipart parts, and ReaderAt inventory were not yet
  enforced), then GREEN after authorization and the required
  `Idempotency-Key` check moved before body processing, duplicate/unknown parts
  were rejected, the archive was spooled to a mode-0600 bounded scratch file,
  and ReaderAt validation avoided full-archive buffering. The independent
  review then found that mismatch was still checked after archive spooling;
  the repair made the receipt part first, returned `400` on mismatch, and the
  new one-byte body test proved zero archive-content reads. The corrected exact
  archive request returned HTTP `201`, batch `import-f0b57aeb10f8383c`, the
  expected/observed SHA and manifest already recorded above, `PROCESSING`, and
  `SECURITY_PHASES_PENDING`; the focused profile exited `0` and removed all
  task-owned runtime state. No extraction, parser output, candidate,
  source-gap Draft, or additional-form import occurred.
- [x] 2026-08-01: Independent read-only re-review after the receipt-first
  repair returned `APPROVED` with no Critical, Important, or Minor finding. It
  confirmed the HTTP `400` mismatch boundary, zero archive-content reads,
  bounded scratch/ReaderAt validation, exact receipt facts, and no authority,
  privacy, publication, or evidence-label bypass.
- [x] 2026-08-01: Final verification rerun passed Gate-0 Node `5/5`, local
  profile contract `14/14`, focused Go boundary `7/7`, archive path inventory,
  inventory `gate0` (`11` artifacts), harness smoke, syntax, `git diff --check`,
  and untracked plan/document whitespace. `task9` and `final` inventory both
  exited the expected `2` for the explicit real Form 048/Admin packet,
  candidate/source-gap Draft, and expansion-authorization blockers.
- [x] 2026-08-01: Source-gap transport-digest parity was repaired with strict
  TDD. The new Go invariant first ran RED because the literal
  `NOT_AVAILABLE` transport projection changed the immutable digest from
  `sha256:1c1cee0ca367092d54c80d1fce792fe3613c52bfec1aa64e3e708e390c1a0ba6`
  to `sha256:81bb76432c4f06c33085d4f73738f761626d26817e9d9e647674b6ae46c8a595`.
  The smallest fix now removes that server-derived projection from both Go
  candidate canonicalization and the persisted output artifact; the focused
  Go invariant and React parity suite are GREEN (`24/24`). Fresh connected
  profiles then passed governed checklist HTTP `2/2` and regulatory source
  refresh HTTP `2/2`, with migration-28 integration `11.156s` and `14.497s`,
  respectively. No source mapping, publication, or additional AGA form was
  imported.
- [x] 2026-08-01: A metadata-only Admin review packet was assembled at
  `deliverables/FURKAN_FORM_048_ADMIN_REVIEW_REQUEST_2026-08-01.zip` with
  archive/register/Form 048 hashes, a 28-row decision table, and identity /
  bounded-expansion templates. Its ZIP SHA-256 is
  `c921dd83c3dff70593d987a465b7cef8ae3f53baf7fb76db4acee752692c07f9`.
  It contains no AGA ZIP/PDF bytes, extracted text, parser output, source
  authority, approval, or production evidence; it has not been sent to an
  external system. Task 9 remains `blocked` pending the actor-bound Admin
  packet and named expansion authorization.
- [x] 2026-08-01: Independent final read-only review of the current worktree
  returned `APPROVED` with no Critical, Important, or Minor findings. It
  confirmed the narrow source-gap digest projection, strict validation,
  metadata-only Admin packet, exact Task-9 blockers, and preserved role,
  authority, privacy, publication, and evidence-label boundaries.
- [x] 2026-08-01: Under the user's explicit continuation authorization, the
  bounded local Form 048 source-read path was run against the exact external
  archive. It verified the archive SHA/size and Form 048 SHA, streamed the
  target PDF only into mode-0600 task-owned temporary storage, ran the pinned
  Poppler text extractor, and removed the temporary source bytes after the
  run. The resulting source-backed handoff contains the exact 28 literal
  protocol-question strings, PDF page/line locators, AGA protocol codes, and
  visible NAMCAR/NAMCATS references; it contains no raw ZIP/PDF bytes. The
  parser receipt is `sha256:adc3a19a56109a44a6e48e9effe4d285a803e63feba836298cf313cdd6039505`
  and the derived proposal packet digest is
  `sha256:4b3a07ea20af11e762a93dc1ae0223400c307f6a0af83d9471769f11082d2a4b`.
  All 28 extraction decisions remain blank `NOT_SUPPLIED`, with
  `SOURCE_MAPPING_REQUIRED`; no candidate, Draft, source-authority decision,
  publication, or Phase 2 import was created.
- [x] 2026-08-01: The source-backed packet test passed `2/2`, the rendered
  Turkish review PDF passed 11-page A4 metadata and visual inspection, the
  package manifest passed `8/8`, and the refreshed ZIP passed `unzip -t`.
  Its SHA-256 is
  `e9ac47ecfd3393e81a43f028343b30cdc7142791050977ca8957f3cdde3dd998`.
  The ZIP is a source-backed, raw-byte-free Admin handoff. Proposal IDs are explicitly labeled
  `PENDING_CURRENT_SERVER_PACKET_BINDING`; the current authenticated Admin
  must rebind decisions through the exact intake packet before any candidate
  command can proceed.
- [x] 2026-08-01: The repaired manifest, embedded ZIP manifest, and ZIP
  integrity were independently rechecked (`8/8`, `unzip -t`, and the same ZIP
  SHA above). The independent read-only Gate 0 review returned `APPROVED` with
  no Critical, Important, or Minor findings. The final phased inventory still
  returned the expected `BLOCKED` result (`exit 2`) because the actor-bound
  Admin identity/28 decisions, immutable candidate/source-gap Draft, and named
  expansion authorization remain external dependencies.
- [x] 2026-08-01: At the user's request, a raw-byte-free all-form source/risk
  review draft was prepared from the supplied archive through the explicit
  `AGA_CHECKLIST_ARCHIVE` path. It inventories all `52` forms and `53` PDF
  entries, preserves the missing `049`/present `035A` fact, records immutable
  archive/register/file hashes, and derives `1,310` question-shaped candidate
  boundaries across `31` forms. The remaining `21` non-protocol forms remain
  inventory records; Form 048 retains the exact `28`-question source-backed
  slice. `174` unique form/question-level NAMCAR/NAMCATS references are kept in
  a separate source-coverage queue.
- [x] 2026-08-01: The all-form review package deliberately keeps every source
  mapping `SOURCE_MAPPING_REQUIRED`, source authority `NOT_ATTESTED`, risk
  interpretation `CANDIDATE_INTERPRETATION_REQUIRES_EXPERT_REVIEW`, decision
  `NOT_SUPPLIED`, candidate import `NOT_IMPORTED`, and publication
  `NOT_AUTHORIZED`. NAMCAR Part 139 remains an official URL proposal without a
  locally hashed source byte; no authority, applicability, severity,
  functional assignment, manager decision, or production evidence was added.
- [x] 2026-08-01: TDD for the all-form handoff recorded the expected RED
  (`node --test tests/aga-all-forms-source-risk-draft.test.mjs`, exit `1`,
  `2` tests / `0` passed) before the review artifacts were completed. The
  GREEN rerun passed `2/2`; the path-driven archive test passed `1/1`, Gate-0
  inventory passed `11` required artifacts, harness smoke passed, manifest
  verification passed `7/7`, ZIP validation passed, explicit untracked
  whitespace scanning passed, and `git diff --check` passed. The resulting
  handoff is
  `deliverables/AGA_ALL_FORMS_SOURCE_RISK_DRAFT_2026-08-01.zip` with SHA-256
  `30700a88aeb5b26514bf7eb76bef050deb08b96294db94117d185de5c9f163b2`.
- [x] 2026-08-01: Independent read-only review initially returned `CHANGES
  REQUIRED` because the package README called the handoff metadata-only while
  it intentionally carried bounded parser-derived question strings. The
  README, plan exclusion, and outcome note were corrected to distinguish
  derived candidate strings/provenance from raw ZIP/PDF bytes, page images, and
  full extracted-text dumps. The repair first reproduced a manifest-mismatch
  RED (`1/2` package tests passed), then regenerated the 7-file manifest and
  returned the package test to `2/2` GREEN.
- [x] 2026-08-01: The independent read-only re-review returned `APPROVED` with
  no Critical, Important, or Minor findings. It confirmed the bounded derived
  question-string disclosure, raw-byte/page-image/full-dump exclusion,
  candidate-only and fail-closed states, source-authority gap, 7/7 manifest,
  ZIP integrity, and no Task-1/runtime/migration changes. The final fresh
  verification rerun exited `0` for the all-form test (`2/2`), archive path
  test (`1/1`), Gate-0 inventory (`11` artifacts), harness smoke, tracked and
  untracked whitespace checks, ZIP `unzip -t`, and fresh Git status capture.
- [x] 2026-08-01: Actor-bound append-only candidate-scope security was added
  with strict TDD. The first focused RED (`GOCACHE=/tmp/avia-go-cache go test
  ./internal/checklistintake -run 'TestActorBoundCandidateScope' -count=1`,
  exit `1`) exposed the missing ledger contract; the history-read RED (same
  command after the test-first immutable-history assertion, exit `1`) exposed
  the missing detached history API; and the authority-boundary/digest RED
  exposed the missing fail-closed fields and canonical digest rejection. The
  final focused suite is GREEN (`4/4` top-level tests, `14/14` subtests), the
  complete `checklistintake` package is GREEN (`26/26` top-level tests plus
  `23/23` subtests), and the focused race run is GREEN. The ledger generates
  decision/root IDs and semantic digests server-side only after a CAA `ADMIN`
  principal has a non-empty subject, session, membership, operation,
  idempotency key, reason, and exact package/scope SHA-256 values. It enforces
  the five-form/250-proposal bounds, stable idempotent replay, predecessor
  CAS, stale-revision rejection, and detached append-only history. The record
  remains candidate-only: source mapping/authority, applicability, risk,
  functional assignment, technical approval, publication, release, and
  production permissions are all fixed `false`. Tests use synthetic fixtures
  only; no real Admin identity, source-owner decision, candidate/Draft,
  publication, or production evidence was created, and Task 9 remains
  `blocked`.
- [x] 2026-08-01: Independent read-only review returned `CHANGES REQUIRED`
  with one Important finding: actor, membership, session, package, and form
  scope facts were still caller assertions. No external decision was created
  while this was repaired. TDD then ran a resolver-bound RED (the updated
  focused command exited `1` because the resolver contract and server-context
  API were absent), followed by GREEN with `4/4` top-level tests and `14/14`
  subtests. The ledger now requires an injected `ActorBoundResolver`; it
  rejects a nil resolver, forged/mismatched principal context, inactive or
  revoked membership/session, expired session, unknown package/scope, and
  resolver form-set or membership drift. Valid-shaped but unknown package and
  scope digests are also rejected. The resolver returns canonical facts, and
  the ledger compares them to the command before generating any leaf. The
  independent read-only re-review returned `APPROVED` with no Critical,
  Important, or Minor findings.

## Decisions

- **2026-08-01 — Permit only a bounded controlled intake stream.** The user's
  selected long-term option authorizes the real intake service to read the exact
  supplied ZIP through the Admin-only local endpoint, but does not remove the
  Gate-0 security limits or any authority/privacy boundary. The service must
  verify the expected hash, inventory the ZIP under `AGA_ZIP_PDF_V1`, persist an
  append-only `PROCESSING` batch and `ARCHIVE_VALIDATE` receipt, and stop before
  scan/parser/finalization when those phases are unavailable. This authorization
  is not a Form 048 identity decision, an extraction boundary decision, source
  authority, candidate/Draft approval, publication, or Phase 2 expansion grant.
- **2026-08-01 — Permit only a derived source-question handoff.** The user's
  continuation authorization permits the local Admin-only path to read the
  exact Form 048 PDF for literal question/provenance preparation after the
  archive receipt, but raw ZIP/PDF bytes remain excluded from Git and no
  source text is treated as regulatory authority. The derived 28-question
  packet is review input only; every Admin extraction decision must bind to
  the current server packet, and `SOURCE_MAPPING_REQUIRED`, candidate
  creation, source-owner attestation, technical approval, publication, and
  Phase 2 expansion remain separate gates.
- **2026-08-01 — Explicit continuation authorization does not waive real
  prerequisites.** The user's authorization permits Task 9 work to resume, but
  it cannot substitute for a current authenticated Admin's Form 048 identity,
  the 28 boundary decisions, the real candidate/source-gap mechanism gate, or
  named expansion authorization. Connected synthetic runtime evidence does not
  substitute for those facts. Until they exist, expansion remains fail-closed
  and no additional form is imported.
- **2026-08-01 — Permit only a derived all-form review draft.** The user's
  request extends the source-question handoff to all `52` AGA forms, but does
  not authorize bulk candidate import or any source/risk/authority decision.
  The derived package may contain form/file hashes, parser candidate boundaries,
  printed regulatory-reference strings, source proposals, and advisory risk
  bands. It must remain raw-byte-free and candidate-only; every Admin/source
  owner decision, exact official source hash/effective date, applicability,
  risk classification, functional assignment, and publication action remains a
  separate fail-closed gate.
- **2026-07-31 — Keep decision projections out of candidate digests.** The
  server-derived `mappingReviewState` projection (and the literal source-gap
  technical projection) is excluded from immutable content/import/edit
  canonicalization; nullable predecessor fields are normalized to match the
  Go `omitempty` transport semantics. A review decision therefore cannot
  rewrite candidate content identity.
- **2026-07-31 — Hide terminal queue entries in the UI only.** The backend
  review queue retains returned/rejected terminal detail for audit/readback,
  while the Department Manager active queue renders only actionable
  `DEPARTMENT_REVIEW` and `TECHNICALLY_APPROVED` candidates.
- **2026-07-31 — Use a dedicated import aggregate.** Archive/file/parser facts
  have different trust and lifecycle semantics from regulatory generation
  runs.
- **2026-07-31 — Use Form 048 as the vertical slice.** It is substantial enough
  to exercise a register match, 28 questions, review guidance, page locators,
  and a real metadata-title conflict without bulk-import risk.
- **2026-07-31 — Preserve eight top-level roles.** Source owner/curator and
  reviewer are scoped functional assignments; Department Manager retains
  separate technical and publication authority.
- **2026-07-31 — Derive required owners server-side.** Authoring requests carry
  scope facts, never trusted Department ownership. Ambiguous/missing reviewed
  responsibility blocks every later decision.
- **2026-07-31 — Separate source authority from candidate mapping.** A source-
  version authority acceptance precedes resolved use; a later mapping
  attestation covers exact candidate applicability/content. Neither decision
  satisfies the other or changes immutable Draft bytes.
- **2026-07-31 — Use `entry_path` as the governed discriminator.** Generalized
  candidate source-binding sets protect existing, official, hybrid, and impact
  lineages; nullable generation-run identity cannot bypass guards or locks.
- **2026-07-31 — Keep real functional-assignment provisioning blocked.** Phase
  1 implements fail-closed storage/resolution and synthetic fixtures only until
  a named governance owner supplies the grant/revoke authority contract.
- **2026-07-31 — Isolate Poppler in an AF_UNIX-only parser service.** The
  networked worker has no direct parser execution path; the parser has
  `network_mode: none`, no service secrets/Docker socket, and bounded read-only
  I/O.
- **2026-07-31 — Keep two authoring paths distinct.** Official-source Drafts
  do not depend on AGA wording; hybrid reconciliation is a successor Draft and
  the official source chain remains sole authority.
- **2026-07-31 — Fail closed on partial traces.** A strict resolved variant or
  literal source-gap variant prevents silent empty citations and false
  validation claims.
- **2026-07-31 — Resolve attestations through immutable decision leaves.**
  Source authority is accepted per link; candidate mapping requires one exact
  complete `REVIEWED_SOURCE_SET`. Returns supersede acceptance for new effects
  and create append-only impact successors without changing history.
- **2026-07-31 — Keep reviewed-source scope distinct from candidate binding.**
  The versioned reviewed set limits functional authority; the binding set
  freezes candidate clauses/hashes/attestations. Real set provisioning remains
  externally blocked with assignment governance.
- **2026-07-31 — Keep direct official authoring out of generation-run
  lineage.** Its source-binding set and command/input/output digests are the
  provenance; `generation_run_id` stays null and `entry_path` remains the
  governed discriminator.
- **2026-07-31 — Use intent-owned quarantine objects in Phase 1.** Same-hash
  batches receive distinct immutable object versions so one expired intent can
  never delete another receipt's bytes. Atomic PUT tags/checksum, fenced
  attempts, and `AGA_IMPORT_MANIFEST_V1` make recovery deterministic.
- **2026-07-31 — Preserve pre-v28 rows as explicitly unattested history.**
  Migration backfills governed lineage but never fabricates source acceptance;
  historical bytes remain readable and new effects require an attested
  successor.
- **2026-07-31 — Preserve the accepted N-to-one publication lifecycle.** Every
  required owner technically approves; one current required owner then invokes
  the single publication command with the complete decision set.
- **2026-07-31 — Require the real source-gap slice before any Phase 2.** A
  current Admin's Form 048 identity/boundary decisions, real candidate, and
  visible gap Draft are mandatory even when regulatory mapping remains blocked.
- **2026-07-31 — Keep post-terminal intake review append-only.** The terminal
  import manifest never changes; identity resolutions, extraction packets and
  strict decision-set successors, and existing-candidate corrections bind its
  digest through their own deterministic current-leaf roots.
- **2026-08-01 — Generate actor-bound candidate-scope leaves server-side.** A
  local candidate-only ledger refuses to append without an injected
  `ActorBoundResolver`. The resolver must validate the authenticated server
  principal against an active CAA Admin membership/current session and resolve
  the immutable reviewed package plus exact form scope; the ledger rechecks
  those canonical facts before generating immutable decision/root IDs, stable
  idempotent replays, predecessor-CAS successors, and detached history reads.
  The leaf fixes all source, applicability, risk, functional-assignment,
  technical-approval, publication, release, and production permissions to
  `false`; it is a security mechanism and test fixture, not a real Admin
  decision or a durable production store.
- **2026-07-31 — Persist parser output before cleanup.** A successful
  `PDF_PARSE` attempt owns an exact private immutable parser-output object
  version before its terminal event. Task 4 prepares proposals only from those
  durable bytes, and unconditional fenced scratch cleanup closes before the
  batch finalizer.
- **2026-07-31 — Lock every stale-candidate effect.** Draft creation, hybrid
  reconciliation, validation claims, mapping acceptance, technical approval,
  publication, and new-package materialization share the existing-candidate-root
  lock and current-leaf reread; historical rows and in-progress Audit bytes stay
  unchanged.
- **2026-07-31 — Test layer mappings when their types exist.** Task 1 freezes
  OpenAPI/generated shapes; Tasks 3–7 test owned domain/transport conversions,
  Task 7 runs the consolidated mapping suite, and Task 8 proves complete mock/
  HTTP payload parity.
- **2026-07-31 — Freeze Gate 0 as specification and inventory evidence only.**
  The lifecycle vocabulary, eight-role boundary, scoped functional assignments,
  server-derived ownership, source-authority separation, AGA limits/hashes, and
  phased verification inventory are documented and locally tested. This gate
  creates no runtime service, migration, OpenAPI contract, or product behavior.
- **2026-07-31 — Keep connected intake fail-closed in the local candidate
  slice.** The in-memory service and HTTP boundary exercise replay, bounded
  receipt, and role checks without pretending to be a PostgreSQL/MinIO/ClamAV/
  Poppler deployment. The migration and runtime integrations remain present as
  explicit, separately verifiable contracts and are not claimed as executed.
- **2026-07-31 — Stop at the real-slice gate.** The supplied archive is an
  external inventory input only. Without the current Admin's Form 048 identity
  and 28 boundary decisions plus named Phase 2 authorization, no additional
  form is imported and no source/manager decision is synthesized.
- **2026-07-31 — Record final status literally.** Local checks are
  `verified locally`; unresolved real-source, assignment, runtime, release,
  and production dependencies are `blocked`; the product is `candidate-only`,
  release is `release pending`, and `production-ready: not established`.
- **2026-07-31 — Bind official authoring to an attestation resolver.** A
  caller-supplied `AuthorityAccepted` boolean is not a decision. Each chain
  link must name a persisted append-only source-authority decision whose
  accepted outcome and exact source/version/hash/chain-role tuple are reread
  through a server-side resolver; missing, returned, forged, or mismatched
  identities fail closed. Synthetic maps are test fixtures only.
- **2026-07-31 — Reject ambiguous ZIP structures before archive/zip.** The
  intake boundary performs raw EOCD/central/local-header preflight and rejects
  ZIP64 sentinels/records, data descriptors, multi-disk metadata, local/central
  disagreement, overlapping ranges, gaps, and trailing bytes before any entry
  is opened. Receipt intake remains `PROCESSING` with only an
  `ARCHIVE_VALIDATE` receipt until scan/parser/finalizer phases can append their
  own immutable receipts.

## Discoveries

- The two supplied ZIP copies are byte-identical, which enables an exact
  planning hash assertion and replay test.
- The register and archive contain 52 forms despite numbering through 053;
  049 is absent and 035A is present. Inventory must preserve this rather than
  manufacture a continuous numeric sequence.
- Form 048's embedded title conflicts with its visible/register title. File
  identity therefore needs an explicit append-only human resolution before
  candidate import.
- Task 6 already supplies origin, scope, trace, source-currentness, impact
  Draft, and historical preservation foundations. The new work should extend
  those boundaries and must not create a parallel approval/publication model.
- The current `GovernedRegulatoryTraceView` requires only `state` at OpenAPI
  level; strict discriminated resolved/source-gap variants are necessary to
  prevent partially populated traces.
- Current source version status/currentness facts do not represent approval
  authority, and the accepted validator binds a single citation. The corrected
  plan therefore adds source-authority decisions and typed ordered chains.
- Existing immutable guards, review queues, candidate loaders, and source locks
  use `generation_run_id` as a sentinel. New existing/hybrid roots require a
  generalized governed discriminator and source-binding set.
- The Go generator currently emits multi-member `oneOf` as `json.RawMessage`
  and ignores multipart bodies; `check-contracts.sh` checks drift but does not
  write tracked artifacts. Task 1 now owns those generator changes and uses the
  writer script first.
- Current Vitest/Playwright configurations explicitly list accepted tests, and
  the HTTP profile has no intake selector. Task 8 now updates discovery/runtime
  configuration and proves nonzero runner discovery.
- Existing core regulatory and checklist governance files are already large.
  Archive parsing belongs in a new `checklistintake` package and new React
  concerns belong in focused components.
- The current `GovernedCandidateView` and loader require a non-null generation
  run, while version-21–27 governed rows have no new authority attestations.
  The corrected lineage union/backfill therefore preserves them as explicit
  `PRE_V28_UNATTESTED` history and gives direct official authoring a distinct
  null-generation variant.
- The current object-store adapter does not expose version/checksum/tag/list/
  delete recovery primitives. Task 3 owns that boundary and must atomically put
  ownership tags rather than inferring them after a crash.
- The supplied archive is available at the current external iCloud path used
  for verification, rather than the historical attachment path in the
  planning notes. The verifier accepts only the explicit environment variable
  and records the same expected hash; it never searches, extracts, or copies
  the archive.
- The local sandbox cannot provide the task-owned PostgreSQL listener and the
  full MinIO/ClamAV/Poppler/browser runtime profile. The corresponding tests
  remain explicit `blocked` dependencies rather than being relabelled as
  passed or production evidence.
- The final phased inventory has non-zero source/test/runner discovery through
  Task 8 and reports Task 9/final as exit `2` for the deliberate real Form 048
  mechanism and Phase 2 expansion gate. This is the expected fail-closed final
  outcome, not a missing-test or skipped-test pass.
- The newly authorized source-read path confirms that the 28 visible Form 048
  questions can be extracted from the exact hash-bound PDF with page/protocol
  provenance. The derived Admin handoff is useful review input, but its
  proposal IDs still require current server packet binding; it cannot stand in
  for an Admin identity decision or extraction decision.

## Outcome Notes

The execution outcome includes the corrected, independently reviewed,
`verified locally` Gate 0 contract plus candidate-only local slices through
Task 8, the explicit Task-9 stop, and the source-backed, raw-byte-free Task-10
handoff. The
OpenAPI/generated transport, forward-only migration, bounded intake/parser
boundary, candidate/reconciliation mechanics, fail-closed HTTP/React surfaces,
and phased inventory are recorded in the working tree. The exact archive was
streamed through the newly authorized Admin-only local intake service and the latest successful run
produced only the immutable `PROCESSING`/`ARCHIVE_VALIDATE` metadata receipt
recorded above. The later bounded source-read run placed only the 28 derived
question strings/provenance in the Admin handoff; raw AGA ZIP/PDF bytes were
removed from temporary storage and were not copied into Git. No external
system, branch, worktree, stage, commit, push, deployment, or production
decision was touched. The later user-authorized all-form handoff adds only
bounded parser-derived question strings/provenance and printed reference
strings for `52` forms; it contains no raw ZIP/PDF bytes, full text dump,
source authority, or candidate records. Real source-owner, assignment,
manager, connected-runtime qualification, Form 048 identity/28 decisions,
source-gap Draft, and Phase 2 expansion dependencies remain `blocked`.

## Execution Prompt

The user has now separately authorized this execution. Future continuation must
still begin only with a new explicit authorization for any scope beyond this
plan. Begin with fresh Git status and literal
plan/index/tracker/evidence statuses. Stop for real functional-assignment
provisioning, Form 048 identity/extraction, source-authority/source-mapping,
Department Manager input, connected runtime qualification, and Phase 2
expansion at the named decision points; do not invent an owner, mapping,
applicability, or approval. Treat `entry_path` as the governed lineage
discriminator, derive owners on the server, lock every bound source in
effect-writing transactions, and run Poppler only through the AF_UNIX/no-network
parser sandbox. Do not commit, push, deploy, or perform branch/worktree
operations without exact current authorization.

## Final Outcome

- Plan: corrected, independently reviewed, `verified locally`, design approved,
  and active with the local candidate implementation recorded.
- Implementation: Gate 0 and Tasks 1–8 `verified locally`; Task 9 `blocked`;
  Task 10 `verified locally`.
- Real-source, assignment, manager, runtime, and Phase 2 expansion decisions:
  `blocked`.
- Product boundary: `candidate-only`.
- Release: `release pending`.
- Production-ready: not established.
