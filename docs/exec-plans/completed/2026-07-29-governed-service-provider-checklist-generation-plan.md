# Governed Service Provider Checklist Generation And Publication

> **For agentic workers:** REQUIRED SUB-SKILL: use
> `superpowers:subagent-driven-development` or `superpowers:executing-plans`
> when implementation is explicitly authorized. Execute one independently
> verifiable task at a time and stop at the stated external authority gates.

**Goal:** Build a fail-closed, department-governed path from exact regulatory
sources to an immutable, applicable, executable checklist version.

**Architecture:** Versioned catalogs, source snapshots, canonical JSON
Schemas, content-addressed generation runs, department-scoped review decisions,
and separate publication decisions form the system of record. OpenAPI, Go,
TypeScript, mock, and HTTP implementations expose the same commands, while
Audits pin exact published content and question snapshots.

**Tech Stack:** JSON Schema, Node.js contract tooling, Go, PostgreSQL
migrations, OpenAPI-generated Go/TypeScript transport, React/Vite, Vitest, and
Playwright.

## Global Constraints

- Preserve exact record, organization, provider-scope, department, target,
  source, clause, revision, version, actor, and idempotency identity.
- Do not create a Technical Expert role. Technical approval belongs to a
  currently assigned responsible Department Manager.
- Admin and generation providers may create or edit candidates but may never
  technically approve or publish them.
- Fail closed on unresolved source gaps, ambiguous mandatory ownership,
  unknown identities, invalid hashes, stale revisions, conflicting duplicate
  outputs, and digest mismatches.
- Published configuration, review history, source snapshots, and in-progress
  Audit packages are append-only or immutable as specified below.
- Preserve the root legacy demo and existing historical `CTV-CABIN-1` records;
  do not treat them as proof of this governed workflow.
- Work remains `candidate-only` and `release pending`. No local result is
  `production-ready`.
- Work on the current branch. Do not commit, push, deploy, access production,
  or modify external systems without separate explicit authorization.

## Status

`active` — Gate 0 and Tasks 1–5 are `verified locally` and independently
accepted. Task 6 is `verified locally` and independently accepted after fix
round 5. Tasks 7–9 remain pending. Work remains
`candidate-only`, `release pending`, and `production-ready: not established`.
The real OPS/AOC pilot remains `blocked`.

## Objective

Build the governed path that turns controlled regulatory material into a
complete, traceable inspection checklist candidate, routes that candidate to
the responsible Department Manager for technical approval, records publication
as a separate decision, and makes the resulting immutable checklist version
selectable and executable in an Audit.

The first acceptance milestone establishes the complete 20-entry
service-provider scope and ownership catalog, then proves one OPS / Air
Operator path end to end. It does not claim that 20 official checklist
libraries are complete merely because the generation and governance engine
works.

## User-Visible Outcome

An authorized user can:

1. identify an organization and all of its active oversight scopes, such as Air
   Operator, CAMO, and ATO;
2. select an oversight scope and inspection type;
3. create a generation run from exact, versioned regulatory sources;
4. receive a fully populated candidate containing the compliance crosswalk,
   requirements, practical questions, verification methods, expected Evidence,
   applicability, source gaps, and citations;
5. submit the candidate to every responsible Department Manager;
6. let each manager approve, return, or reject only material within the
   manager's department scope;
7. publish the technically approved candidate through a separate, recorded
   Department Manager action;
8. select the immutable published version for an applicable Audit;
9. let an assigned Inspector run the checklist, answer questions, attach
   Evidence, and create an eligible Potential Finding through the existing
   lifecycle; and
10. create a new candidate rather than changing published versions or
    in-progress Audits when a source changes.

## Scope

- Normalize the supplied 20-row service-provider scope and responsible-unit
  matrix into stable IDs without replacing its exact labels.
- Support multiple active service-provider/authorization scopes for one
  organization.
- Add department and organizational-unit membership needed to constrain a
  `manager` principal to the manager's actual approval scope.
- Formalize JSON Schema contracts for generation input, compliance mapping
  output, inspection-checklist output, and manager review decisions.
- Treat supplied CC material as a versioned secondary State compliance
  crosswalk input and comparator, never as a substitute for current primary
  sources.
- Partition CC material used as input from unseen holdout material used for
  evaluation.
- Preserve the existing hash-addressed public NAMCATS collection, clause
  locators, refresh policy, adaptive-scope model, and immutable checklist
  snapshots.
- Add a controlled generation/import seam that can initially use a
  Codex-assisted batch run and later accept an independently authorized model
  provider without changing the domain workflow.
- Generate a complete candidate, not an empty checklist shell.
- Add Department Manager technical review and separate publication commands.
- Connect published, applicable checklist versions to Audit package assembly
  and existing Inspector execution.
- Prove the governed path using the existing OPS / Air Operator PQ 4.450 /
  CE-7 pilot and its six questions.

## Explicit Exclusions

- No autonomous legal conclusion, official compliance determination,
  enforcement action, certification decision, Finding decision, or checklist
  publication.
- No new Technical Expert or Technical Standards Manager role.
- No assumption that a generic `manager` role grants department authority.
- No automatic approval of a visible source gap or unresolved applicability.
- No mutation of a published checklist version or an Audit already pinned to
  one.
- No re-download, re-OCR, or replacement of the verified 58-document public
  NAMCATS baseline unless its bounded refresh process detects a change.
- No full official checklist rollout for all 20 provider scopes in this first
  milestone. That requires source completeness and the responsible managers'
  domain validation after the pilot.
- No production external-model credentials, provider procurement, or
  production AI integration. Those require a separate governance and security
  decision.
- No XML contract unless a named integration requires it. JSON plus JSON
  Schema is canonical; XML may be an export adapter later.
- No root legacy-demo change, branch operation, commit, push, deployment, or
  production-system mutation.
- No production-readiness claim. The implementation remains `candidate-only`
  until separate release evidence exists.

## Independent Review Closure Baseline

The 2026-07-29 independent review did not accept plan checkmarks or historical
test claims as proof. It found one P0 authorization defect, five P1 milestone
failures, and two P2 evidence/documentation failures. This plan is not accepted
until each closure gate below has fresh behavioral evidence:

| Review finding | Required closure | Owning task/gate |
|---|---|---|
| P0 ordinary Admin direct publication | Remove the normal OpenAPI/router/service publication path or confine fixture creation to an internal non-user-facing test-profile boundary; prove normal Admin HTTP and service calls cannot publish | Gate 0 |
| P1 missing department-scoped lifecycle | Persist current department/unit assignments and enforce review, joint approval, return/reject, revision invalidation, technical approval, and separate publication server-side | Tasks 2, 3, and 6 |
| P1 missing 20-provider/multi-scope/typed-target model | Install the exact catalog, effective organization scopes, and typed targets without collapsing identities | Tasks 1 and 2 |
| P1 missing canonical generation/import seam | Add canonical JSON Schemas, bounded source snapshots and hashes, content-addressed runs, complete linked output, and fail-closed import | Tasks 3 and 4 |
| P1 checklist selection is template-ID-only | Enforce provider scope, department, inspection type, target, qualifiers, and effective period before pinning an Audit package | Task 7 |
| P1 CC/source governance is documentation-only | Persist secondary-crosswalk partitions, blind holdout separation, hashes, clause locators, impact reviews, and new candidate revisions | Tasks 3 and 8 |
| P2 missing suites can produce false-green commands | Add an explicit verification-inventory preflight, meaningful negative tests, mock and HTTP E2E, migration/recovery checks, and expected suite counts | Task 9 |
| P2 legacy technical-expert language conflicts with authority | Migrate new language to responsible Department Manager technical review while retaining explicit legacy-read compatibility only | Task 1 |

Plan prose, checkmarks, fixture labels, snapshots, and generated transport
parity are never sufficient closure evidence on their own. Each row requires
the named behavioral test and inspected persisted/audit result. A P0/P1 row
cannot be deferred to rollout or reclassified as a production dependency.

## Settled Product Decisions

1. The system produces two related outputs:
   - a CC-like regulatory compliance crosswalk; and
   - a practical inspection checklist derived from approved requirements.
2. A generated Draft is a complete candidate. `GENERATED_DRAFT` means “not yet
   approved or published,” not “blank” or “unfinished by design.”
3. Primary authority comes from exact authorized ICAO PQ/Annex/SARP versions,
   current NAMCAR/NAMCATS material, and controlled NCAA procedures.
4. Supplied CC rows are secondary `STATE_COMPLIANCE_CROSSWALK` input. They may
   inform candidate generation or independently evaluate it, but the same rows
   must not serve both purposes in one evaluation.
5. JSON plus JSON Schema is the canonical interchange contract.
6. Source/hash change monitoring is event-driven and at least monthly; a full
   reconciliation occurs every six months; a formal validity review occurs at
   least annually.
7. Admin may manage sources, start or import a generation run, edit a candidate,
   and submit it for review. Admin cannot technically approve or publish it.
8. The responsible Department Manager performs technical review within the
   manager's department scope and separately decides publication.
9. Technical approval and publication are two commands, two audit events, and
   two timestamps even when the same manager performs both.
10. Cross-department material requires approval from every required owning
    department before publication.
11. A visible unresolved source gap blocks technical approval in this
    milestone. Escalation may record the issue but does not bypass it.
12. Organizations may hold several provider scopes simultaneously. Checklist
    applicability is composed from active provider authorization, inspection
    type, configuration/operation qualifiers, target identity, source scope,
    and recorded risk/history—not one coarse organization type.
13. Published checklist versions are immutable, and every Audit remains pinned
    to the exact version it started with.
14. Positive lifecycle tests may use an explicitly synthetic, internal
    test-profile source package whose gaps and owner decisions are controlled
    by the test. That fixture proves workflow mechanics only. It must not alter,
    approve, publish, or relabel the real source-bound OPS/AOC pilot; the real
    pilot must separately prove technical-approval/publication denial until
    acceptance criterion 16 is satisfied.

Existing `EXPERT_REVIEW_REQUIRED` data remains readable during migration, but
new workflow language is `TECHNICAL_REVIEW_REQUIRED`, whose authorized reviewer
is the relevant Department Manager.

## Source Roles And Evaluation Boundary

| Source class | Role | May establish current authority alone? |
|---|---|---:|
| Authorized ICAO PQ / Annex / SARP | Primary international requirement | No; exact version and applicability still require review |
| Current NAMCAR / NAMCATS | Primary national regulatory source | No; source owner confirms currency and applicability |
| Controlled NCAA procedure/manual/tool | Primary implementation source | No; responsible Department Manager validates use |
| Supplied CC crosswalk | Secondary State crosswalk/input/comparator | No |
| Generated compliance mapping | Candidate derived record | No |
| Generated inspection checklist | Complete candidate operational artifact | No |
| Department Manager technical decision | Department-scoped validation | Yes for the configured workflow, but it is not publication |
| Department Manager publication decision | Creates an official immutable checklist version | Yes, after all required technical approvals |

The generated bundle records source IDs, versions, hashes, clause locators,
generation-policy version, schema version, provider adapter/version, and content
digest. Chat history is never the system of record.

## Service Provider Catalog Baseline

The source matrix is:

`/Users/marlonjd/.codex/attachments/5360b407-94c2-479e-bea4-7e2caf0cf15d/Service Provider.docx`

Its recorded SHA-256 is
`a097081bc74850bce95a87687d9dab228e4f40ef93fd7a65ded11f110998d4e9`.
The normalized tracked catalog becomes the durable input; runtime behavior must
not depend on the attachment remaining at that path.

| Code | Exact provider label | Exact oversight topics | Raw responsible CAA unit |
|---|---|---|---|
| `AIR_OPERATOR` | Air Operator (AOC Holder) | Flight Operations; Cabin Safety; Operational Control; Crew Training; Dangerous Goods; SMS; Security; Manuals | Flight Operations Inspectorate (FOI) |
| `AMO` | Approved Maintenance Organization (AMO) | Maintenance Procedures; Personnel; Tooling; Facilities; Quality System; SMS; Records | Airworthiness Inspectorate |
| `CAMO` | Continuing Airworthiness Management Organization (CAMO) | Airworthiness Management; Maintenance Programme; Reliability Programme; Technical Records | Airworthiness Inspectorate |
| `ATO` | Approved Training Organization (ATO) | Training Programmes; Instructors; Examinations; Training Records; Simulators | Personnel Licensing & Training Department |
| `FSTD` | Flight Simulation Training Device (FSTD) | Simulator Qualification; Configuration; Maintenance; Records | Personnel Licensing & Training Department |
| `AERODROME_OPERATOR` | Aerodrome Operator | Runways; Taxiways; Aprons; Lighting; RFFS; Wildlife; Obstacle Control; Emergency Plan; SMS | Aerodrome Inspectorate |
| `ANSP` | Air Navigation Service Provider (ANSP) | ATS; ATC Procedures; Airspace Management; SMS; Contingency Plans | Air Navigation Services Inspectorate |
| `CNS_PROVIDER` | Communication Service Provider (CNS) | Communication Systems; Navigation Aids; Surveillance Systems; Maintenance | CNS Inspectorate |
| `AIS_AIM_PROVIDER` | AIS/AIM Provider | AIP; NOTAM; Charts; Data Quality; Digital AIM | AIS/AIM Inspectorate |
| `MET_PROVIDER` | Meteorological Service Provider (MET) | Aviation Weather Services; Forecasting; Observations; MET Reports | Meteorological Oversight Unit |
| `SAR_ORGANIZATION` | Search and Rescue (SAR) Organization | Rescue Coordination; SAR Plans; Exercises; Readiness | SAR Oversight Unit |
| `GROUND_HANDLING` | Ground Handling Organization | Passenger Handling; Ramp Operations; Load Control; Baggage; Aircraft Servicing; SMS | Ground Operations / Flight Operations Inspectorate |
| `FUEL_PROVIDER` | Fuel Service Provider | Fuel Storage; Fuel Quality; Refuelling Procedures; Equipment; Personnel | Aerodrome Inspectorate |
| `CARGO_REGULATED_AGENT` | Cargo Terminal / Regulated Agent | Cargo Acceptance; Dangerous Goods; Security Controls; Documentation | Aviation Security (AVSEC) + Dangerous Goods Office |
| `AVSEC_PROVIDER` | Aviation Security Service Provider | Passenger Screening; Access Control; Hold Baggage Screening; Staff Training | AVSEC Inspectorate |
| `RPAS_UAS_OPERATOR` | RPAS/UAS Operator | Flight Operations; Remote Pilot Competency; Maintenance; Operational Risk Assessment; C2 Link | RPAS / Flight Operations Inspectorate |
| `DOA` | Aircraft Design Organization (DOA) | Design Approval; Compliance Demonstration; Configuration Control | Airworthiness Certification Department |
| `POA` | Production Organization (POA) | Production System; Quality Assurance; Product Conformity | Airworthiness Certification Department |
| `AEMC` | Aviation Medical Centre (AeMC) | Medical Facilities; Equipment; Medical Records; Personnel | Aeromedical Department |
| `AME` | Aviation Medical Examiner (AME) | Medical Examinations; Certification Procedures; Record Keeping | Aeromedical Department |

Catalog rules:

- Keep all 20 codes distinct. CNS, AIS/AIM, MET, and SAR may have a navigation
  UI grouping, but they never inherit ANSP applicability merely from grouping.
- Keep coarse `organizationType` separate from oversight scope.
- Treat provider type as an approval/oversight scope, not necessarily a legal
  organization type. FSTD may target a device, AME a regulated person, and
  Aerodrome/Fuel/AeMC/Cargo may need facility or location targets.
- Preserve every raw responsible-unit label. Normalize it to one or more
  organizational units and a parent approval department.
- `+` and `/` ownership labels remain `REVIEW_REQUIRED` until NCAA confirms
  whether they mean joint ownership, a combined unit, or consultation. The
  system fails closed and cannot infer publication authority from punctuation.
- The normalized responsibility link carries `PRIMARY`, `JOINT`, or
  `CONSULTED`, plus `approvalRequired`.

## Authority Matrix

| Action | Admin | Responsible Department Manager | Other Department Manager | Inspector / Lead |
|---|---:|---:|---:|---:|
| Maintain source metadata/catalog | Yes | View/comment | View | View as needed |
| Start/import generation run | Yes | Yes, own scope | No | No |
| Edit complete candidate before review | Yes | Yes, own scope | No | No |
| Submit candidate for review | Yes | Yes, own scope | No | No |
| Return/reject technical review | No | Yes, own scope | No | No |
| Technically approve | No | Yes, own scope | No | No |
| Publish approved version | No | Yes, own scope; separate action | No | No |
| Select applicable published version for Audit | No | Yes | No | Lead only when authorized |
| Execute assigned checklist | No | No | No | Yes |

A `manager` role without a current department/unit assignment has no technical
approval or publication authority.

## Workflow And State Machine

```text
GENERATED_DRAFT
      |
      v
DEPARTMENT_REVIEW
      |--------------------> REJECTED
      |
      +----> RETURNED ----> new candidate revision ----> DEPARTMENT_REVIEW
      |
      v
TECHNICALLY_APPROVED
      |
      v
PUBLISHED
```

- A candidate is complete at `GENERATED_DRAFT`.
- Submission freezes the reviewed candidate revision and source digests.
- Editing a submitted or returned candidate creates a new revision and
  invalidates prior approvals for affected ownership scopes.
- Joint candidates reach `TECHNICALLY_APPROVED` only after all required
  department decisions are present for the exact revision.
- Publication creates an immutable `ChecklistTemplateVersion` snapshot and a
  separate publication decision.
- Source changes create an impact record and a new generated revision. They do
  not rewrite the preceding graph.

## Canonical JSON Contracts

Create these schemas under `docs/regulatory-sources/schemas/`:

- `service-provider-catalog.schema.json`
- `regulatory-generation-request.schema.json`
- `compliance-mapping-candidate.schema.json`
- `inspection-checklist-candidate.schema.json`
- `department-review-decision.schema.json`

The generation request minimally contains:

```json
{
  "schemaVersion": "1.0.0",
  "requestId": "GENREQ-OPS-AOC-0001",
  "serviceProviderScopeIds": ["AIR_OPERATOR"],
  "inspectionType": "RAMP_INSPECTION",
  "target": {
    "kind": "ORGANIZATION",
    "organizationId": "ORG-FLY-NAMIBIA"
  },
  "sourceSnapshotIds": ["ICAO-OPS-SNAPSHOT-1", "NCAA-NAMCATS-SNAPSHOT-1"],
  "secondaryCrosswalkPartitionId": "CC-OPS-TRAIN-1",
  "generationPolicyVersion": "regulatory-checklist-v1",
  "requestedOutputs": [
    "COMPLIANCE_MAPPING",
    "INSPECTION_CHECKLIST"
  ]
}
```

The candidate bundle minimally contains:

```json
{
  "schemaVersion": "1.0.0",
  "generationRunId": "GENRUN-OPS-AOC-0001",
  "inputDigest": "sha256:...",
  "status": "GENERATED_DRAFT",
  "serviceProviderScopeIds": ["AIR_OPERATOR"],
  "requiredOwnerUnitIds": ["FOI"],
  "mappings": [
    {
      "mappingId": "MAP-OPS-AOC-0001",
      "requirement": "A reviewable requirement statement.",
      "relationship": "PARTIALLY_ADDRESSED",
      "applicability": "DIRECT",
      "sourceClauseIds": ["ICAO-A6-I-4.2.2.2", "NAMCATS-121-07-006"],
      "sourceGap": null,
      "rationale": "A bounded, cited rationale."
    }
  ],
  "questions": [
    {
      "questionId": "RAMP-001",
      "mappingIds": ["MAP-OPS-AOC-0001"],
      "prompt": "A practical inspection question.",
      "verificationMethod": "Physical observation and record reconciliation",
      "expectedEvidence": ["Inspector observation", "Controlled record"],
      "allowedAnswers": [
        "COMPLIANT",
        "NON_COMPLIANT",
        "OBSERVATION",
        "NOT_APPLICABLE",
        "NOT_CHECKED"
      ],
      "mandatoryCore": true,
      "safetyCritical": true
    }
  ]
}
```

Every referenced provider, unit, source, clause, mapping, and target identity
must resolve. Unknown IDs, missing Evidence expectations, missing citations,
invalid hashes, or an output claiming `TECHNICALLY_APPROVED`/`PUBLISHED` fail
closed.

## Repository Orientation And Affected Interfaces

### Existing predecessor foundations to preserve

- `docs/exec-plans/active/2026-07-28-regulatory-knowledge-checklist-pilot-plan.md`
- `docs/exec-plans/active/2026-07-28-regulatory-source-refresh-adaptive-checklists-plan.md`
- `scripts/regulatory/sync-ncaa-namcats.mjs`
- `docs/regulatory-sources/ncaa-namcats-manifest.json`
- `docs/regulatory-sources/derived/`
- immutable `checklist_template_versions` and inspection-package version
  bindings.

### Product authority

- `docs/product-specs/data-and-rules/CONCEPTUAL_DATA_MODEL.md`
- `docs/product-specs/data-and-rules/STATUS_PERMISSION_SECURITY.md`
- `docs/product-specs/workflows/AUDIT_CHECKLIST_WORKFLOW.md`
- `docs/product-specs/modules/AUDIT_PLANNING.md`
- `docs/product-specs/workflows/SURVEILLANCE_PLANNING_WORKFLOW.md`
- `docs/product-specs/screen-specs/DEPARTMENT_MANAGER_WORKSPACES.md`
- `docs/regulatory-sources/README.md`

### Persistence and domain services

- new `apps/api/migrations/000021_regulatory_checklist_governance.up.sql`
- `apps/api/internal/identity/`
- `apps/api/internal/organizations/`
- new `apps/api/internal/regulatory/`
- new `apps/api/internal/checklistgovernance/`
- `apps/api/internal/application/template_workflow.go`
- `apps/api/internal/application/clean_state_creation.go`
- `apps/api/internal/configuration/`
- `apps/api/internal/assignments/`
- `apps/api/internal/inspections/`

### Transport contracts

- `api/openapi/source/openapi.json`
- `api/openapi/source/schemas/domain.json`
- `api/openapi/source/schemas/platform.json`
- `api/openapi/source/paths/platform.json`
- `api/openapi/source/paths/workflows.json`
- generated OpenAPI, Go, and TypeScript artifacts.

### React surfaces

- `apps/web/src/backend/backend.ts`
- `apps/web/src/backend/transport-mappers.ts`
- `apps/web/src/backend/http-backend.ts`
- mock backend and canonical seed/profile data
- `apps/web/src/features/admin/regulatory-library-page.tsx`
- `apps/web/src/features/admin/checklist-builder-page.tsx`
- `apps/web/src/features/checklists/checklist-management-page.tsx`
- `apps/web/src/features/organizations/organization-registry-page.tsx`
- `apps/web/src/features/planning/new-audit-wizard.tsx`
- `apps/web/src/features/inspections/inspection-package-builder-page.tsx`
- existing checklist runner and Potential Finding flow.

## Ordered Tasks

### Gate 0 — Remove the ordinary Admin publication bypass

This security gate precedes taxonomy and generation work. It is complete only
when normal runtime profiles have no Admin command capable of creating a
`PUBLISHED` checklist version.

- [x] Write failing OpenAPI, router, service, and HTTP tests proving that
  `POST /v1/admin/checklist-template-versions` is absent from normal transport,
  an Admin principal cannot call a direct-publish service, and no generated Go
  or TypeScript client exposes the operation.
- [x] Remove the operation from
  `api/openapi/source/paths/platform.json`, its input from
  `api/openapi/source/schemas/platform.json`, the canonical router, handler,
  application service, generated transports, and normal mock/HTTP backends.
- [x] Replace the current clean-state contract assertion with a regression
  assertion that direct publication is forbidden in normal profiles.
- [x] If historical fixtures still require `CTV-CABIN-1`, create them only
  through an internal `apps/api/internal/testprofile` bootstrap function that
  is not registered in OpenAPI, routers, generated clients, visible actions,
  or production-capable profiles.
- [x] Preserve `CTV-CABIN-1` and Audits already pinned to it as historical
  immutable demo records; do not synthesize technical-approval decisions for
  them.
- [x] Run the focused contract, application, HTTP, generated-artifact, and demo
  regression tests. Inspect the normal route table and compiled generated
  clients, not only a mocked 403 response.

Primary files:

- `api/openapi/source/paths/platform.json`
- `api/openapi/source/schemas/platform.json`
- `api/openapi/tests/clean-state-creation.test.mjs`
- `api/openapi/tests/governed-checklist-publication-boundary.test.mjs`
- `apps/api/internal/httpapi/canonical_api.go`
- `apps/api/internal/httpapi/clean_state_creation_api.go`
- `apps/api/internal/application/clean_state_creation.go`
- `apps/api/internal/testprofile/`
- generated Go and TypeScript transport artifacts

Gate 0 expected proof:

```bash
./scripts/check-contracts.sh
node --test \
  api/openapi/tests/clean-state-creation.test.mjs \
  api/openapi/tests/governed-checklist-publication-boundary.test.mjs
cd apps/api
env GOCACHE=/private/tmp/avia-regulatory-go-cache \
  go test ./internal/application ./internal/httpapi ./internal/testprofile
```

Expected: all commands pass; the ordinary Admin publication operation is
absent from OpenAPI/router/generated clients; an internal historical bootstrap
has no user-facing route; existing immutable demo records still load.

### Task 1 — Freeze product authority, taxonomy, and contract language

- [x] Add a versioned, tracked service-provider catalog containing exactly the
  20 source rows, topics, raw unit labels, aliases, target kinds, and
  normalization status.
- [x] Update product specifications so the responsible Department Manager
  performs technical approval and publication as two separate decisions.
- [x] Replace new uses of `EXPERT_REVIEW_REQUIRED` with
  `TECHNICAL_REVIEW_REQUIRED`; document backward compatibility for existing
  records.
- [x] Add explicit permission rows for generation, technical review, and
  publication.
- [x] Add a regression test proving no `technicalExpert`, `technical_expert`,
  or equivalent application role/permission is introduced in product specs,
  OpenAPI role enums, Go roles, TypeScript roles, seed profiles, or visible
  actions.
- [x] Record that one organization can have several provider scopes and that a
  scope may target an organization, person, facility, device, system, asset, or
  location.
- [x] Preserve normalized parent approval departments only where the supplied
  matrix is unambiguous. Represent `+` and `/` rows as `REVIEW_REQUIRED` with
  no inferred semantic relationship or `approvalRequired` owner until NCAA
  confirms whether they mean joint ownership, a combined unit, or consultation.
- [x] Prove catalog validation rejects the 19-entry, 21-entry, duplicate-code,
  duplicate-label, AeMC/AME-collapsed, ANSP-family-collapsed, altered-topic,
  altered-raw-owner, provider-specific target-kind and normalized-owner
  reassignment, unknown-target-kind, and silently normalized or semantically
  inferred ambiguous variants.

Primary files:

- `docs/regulatory-sources/catalogs/service-provider-catalog.v1.json`
- `docs/regulatory-sources/schemas/service-provider-catalog.schema.json`
- the product-authority files listed above
- `tests/service-provider-catalog.test.mjs`

### Task 2 — Add persistence, department scope, and multi-scope organizations

- [x] Add `caa_departments`, `caa_organizational_units`,
  `caa_department_memberships`, `service_provider_types`,
  `service_provider_topics`, `service_provider_topic_links`,
  `service_provider_unit_responsibilities`, and
  `organization_service_provider_scopes`, plus typed
  `regulated_targets`/scope-target links for organization, person, facility,
  device, system, asset, and location subjects.
- [x] Preserve the existing coarse `organizations.organization_type` for
  compatibility; do not overload it with oversight scopes.
- [x] Model authorization/certificate identity, effective dates, status,
  operation/activity qualifiers, and optional target identifiers.
- [x] Extend the authenticated principal/authority context with resolved
  effective-dated department/unit assignments while keeping `manager` as the
  role. A role with no current assignment resolves to no technical or
  publication authority.
- [x] Update `apps/api/migrations/migrations.go` to version 21 and add a
  full-schema inventory test that fails if any governed table, foreign key,
  uniqueness constraint, append-only guard, or applicability index is absent.
- [x] Add migration/recovery tests for empty-database installation, upgrade
  from version 20, rollback before any governed record exists, rollback refusal
  after governed records exist, and forward repair that preserves published,
  review, source, and Audit history.
- [x] Add database and service tests proving Air Operator + CAMO + ATO can be
  active for one organization and that expired scopes cannot select new
  checklists.
- [x] Add typed-target tests proving FSTD resolves to a device, AME to a person,
  and facility/system/asset/location targets are not coerced to an organization
  ID.
- [x] Prove that provider scopes and manager memberships do not leak through
  organization-scoped Auditee projections.

### Task 3 — Persist regulatory inputs, generation lineage, and review records

- [x] Formalize source version and normalized clause records without copying
  full regulatory text into Git.
- [x] Import supplied CC rows as versioned
  `STATE_COMPLIANCE_CROSSWALK`/`SUPPLIED_WORKING_COPY` records.
- [x] Create non-overlapping training/input and blind holdout partitions using
  stable row/Annex/section identities and database constraints that reject one
  identity appearing in both partitions for the same evaluation.
- [x] Add generation-run input/output digests, schema/policy/provider versions,
  exact provider scopes, inspection type, typed target identity, source
  snapshot IDs/hashes/clause locators, immutable artifacts, and failure state.
- [x] Extend `template_draft_versions` into revisioned generated candidates and
  add required-owner, review-decision, and publication-decision records.
- [x] Make review and publication decisions append-only and attributable by
  actor, current assignment scope, candidate revision, exact content digest,
  reason, timestamp, operation ID, and idempotency key.
- [x] Add uniqueness/conflict constraints so one input digest cannot acquire
  conflicting output, one idempotency identity cannot change semantic payload,
  and a published version cannot refer to an unapproved digest.
- [x] Preserve immutable published template versions and existing Audit
  bindings.

### Task 4 — Implement the controlled generation/import seam

- [x] Add a `RegulatoryGenerationProvider` boundary that accepts only a
  validated generation request and returns only a validated candidate bundle.
- [x] Implement deterministic fixtures and an imported-result provider first.
  A Codex-assisted batch run may produce the JSON artifact outside the runtime;
  import validation, lineage, and governance remain inside the system.
- [x] Add commands to prepare a bounded request, validate an output, and import
  it idempotently:

```bash
node scripts/regulatory/prepare-checklist-generation.mjs --request GENREQ-OPS-AOC-0001
node scripts/regulatory/validate-checklist-candidate.mjs path/to/candidate.json
node scripts/regulatory/import-checklist-candidate.mjs path/to/candidate.json
```

- [x] Reject output that lacks citations, changes provider/source identity,
  uses unknown mapping/clause/target IDs, supplies invalid hashes, contains
  unsupported claims, asserts an authoritative state, or duplicates an input
  digest with different content.
- [x] Produce both compliance mappings and practical questions in one linked
  candidate bundle; reject an empty shell or a bundle missing either output.
- [x] Prove every question contains mapping IDs, exact citations, verification
  method, expected Evidence, allowed answers, and mandatory/safety flags.
- [x] Prove an identical bounded request and identical output return the same
  generation run/content identity, while the same digest with different bytes
  fails visibly without candidate or publication side effects.
- [x] Keep external production model adapters and credentials outside this
  milestone.

### Task 5 — Expose complete candidates to Admin

- [x] Add OpenAPI operations to list source snapshots, create/import a
  generation run, inspect failures, edit a candidate revision, and submit it
  for department review.
- [x] Regenerate and inspect Go and TypeScript clients, then implement identical
  mock and HTTP shapes. Contract parity is necessary but does not replace
  authorization and persisted-state assertions.
- [x] Update Regulatory Library to show exact source versions, hashes,
  locators, crosswalk role, applicability, gaps, and generation lineage.
- [x] Update Checklist Builder to show a complete candidate with requirement,
  question, verification method, expected Evidence, mandatory/critical flags,
  citations, and owner units.
- [x] Make it impossible for Admin to technically approve or publish through
  UI, visible actions, OpenAPI, generated clients, HTTP handlers, mock methods,
  or direct service calls.
- [x] Keep validation errors attached to exact fields and source identities.

### Task 6 — Implement Department Manager review and publication

- [x] Add a department-filtered review queue to
  `/department-manager/checklist-management`.
- [x] Add return, reject, technically approve, and publish commands with
  revision, actor, effective department assignment, reason, timestamp,
  operation ID, idempotency key, and candidate content digest.
- [x] Require all `approvalRequired` departments on joint material before the
  candidate can become `TECHNICALLY_APPROVED`.
- [x] Block submission or technical approval while any mandatory owner remains
  `REVIEW_REQUIRED`, any source gap is unresolved, or any referenced source
  hash/identity no longer matches the frozen candidate revision.
- [x] Editing reviewed content creates a new revision and invalidates every
  decision whose owned content or source lineage changed; unaffected joint
  decisions may remain only when an explicit stable ownership/digest proof is
  recorded.
- [x] Test `GENERATED_DRAFT`, `DEPARTMENT_REVIEW`, `RETURNED`, revised and
  resubmitted, `REJECTED`, `TECHNICALLY_APPROVED`, and `PUBLISHED`, including
  illegal transition, stale revision, duplicate identical command, conflicting
  duplicate command, generic-manager denial, cross-department denial, joint
  partial approval, and joint complete approval.
- [x] Use an explicitly synthetic internal test-profile candidate for the
  positive technically-approved/published path. In the same suite, prove the
  real OPS/AOC candidate with its controlled-procedure/Part 127/Part 140 gaps
  cannot become `TECHNICALLY_APPROVED` or `PUBLISHED`.
- [x] Record technical approval and publication as separate audit events.
- [x] Reconfirm Gate 0 remains closed after the new publication commands are
  added: no Admin endpoint, service, visible action, mock helper, or generated
  client may publish.
- [x] Publish an immutable snapshot only from the exact technically approved
  revision and content digest. The publication transaction must fail if the
  recomputed digest, approved digest, and persisted candidate bytes differ.
- [x] Deny cross-department manager actions server-side, not only in the UI.

### Task 7 — Compose and execute applicable Audit packages

- [x] Filter selectable published versions by the organization's active
  provider scopes, inspection type, target kind/identity, owning department,
  effective date, and operation/configuration qualifiers.
- [x] Allow applicable modules to compose one inspection package without
  duplicating question identity.
- [x] Pin the package to exact published checklist versions and question
  snapshots, approved content digest, provider-scope identity, typed target,
  inspection type, qualifiers, and applicability decision.
- [x] Add negative selection tests for inactive/expired scope, wrong
  department, wrong inspection type, wrong target kind/identity, outside
  effective period, missing qualifier, unpublished version, and stale
  applicability.
- [x] Run the OPS / Air Operator pilot through the existing checklist runner.
- [x] Verify answer states, comments, required Evidence, section completion,
  submission, and eligible Potential Finding creation.
- [x] Verify an Inspector cannot mutate questions, regulatory mappings,
  publication identity, or another Inspector's unauthorized assignment while
  executing the package.
- [x] Prove that a source change or newly published version leaves the
  in-progress Audit byte-for-byte unchanged.

### Task 8 — Add refresh impact, holdout evaluation, and rollout gate

- [x] Connect source/hash changes to affected clauses, mappings, candidates,
  questions, and provider scopes.
- [x] Create a new impact-review candidate; never silently edit a published
  version.
- [x] Prove a source change cannot mutate a published checklist, prior review
  decision, publication event, or in-progress Audit package, and cannot reuse
  the superseded candidate revision for a new publication.
- [x] Evaluate the Air Operator output against only the reserved CC holdout and
  record mapping precision/recall-style review counts, unsupported claims,
  missed rows, and manager corrections.
- [x] Assert the holdout evaluator rejects any run whose row identities overlap
  generation input, even if content hashes or filenames differ.
- [x] Preserve mandatory/safety-critical/source-changed/open-or-repeat-Finding
  adaptive-scope guardrails from the predecessor plan.
- [x] Produce a rollout-readiness decision for the remaining provider scopes.
  Do not label them official until their sources and responsible managers are
  ready.

Suggested follow-on rollout groups, each gated by its own source package and
manager validation, are:

1. OPS: Air Operator, Ground Handling, and RPAS/UAS.
2. AIR: AMO, CAMO, DOA, and POA.
3. PEL/Aeromedical: ATO, FSTD, AeMC, and AME.
4. Aerodrome: Aerodrome Operator and Fuel Service Provider.
5. Air Navigation: ANSP, CNS, AIS/AIM, MET, and SAR, while retaining their
   separate provider identities and owners.
6. AVSEC/Dangerous Goods: Cargo/Regulated Agent and AVSEC Service Provider.

These groupings are rollout views, not inherited regulatory applicability.

### Task 9 — Verify, document, and hand off

- [x] Add `scripts/verify-governed-checklist-test-inventory.mjs`. It must fail
  before invoking a runner when any required test/spec/schema/migration/script
  is missing, print the exact missing paths, and assert expected suite/file
  counts so Node/Vitest/Playwright cannot silently omit named files.
- [x] Add contract, schema, migration, authorization, workflow, idempotency,
  privacy, immutability, UI, and browser tests that assert persisted state,
  audit records, denials, and unchanged digests rather than only labels,
  snapshots, fixture presence, or mocked success.
- [x] Capture one mock and one HTTP end-to-end OPS / Air Operator flow from
  generated candidate through Department review, technical approval, separate
  publication, applicability selection, Inspector execution, submission, and
  eligible Potential Finding creation using only the explicitly synthetic
  internal test-profile source package.
- [x] In mock and HTTP, separately exercise the real source-bound OPS/AOC
  candidate and prove the unresolved controlled-procedure/Part 127/Part 140
  gates stop technical approval and publication without creating a version or
  audit event.
- [x] Run the same negative authority matrix in service, HTTP, and mock tests:
  Admin approval/publication, generic manager without assignment, OPS manager
  against AIR/PEL/AVSEC, incomplete joint ownership, stale revision, conflicting
  duplicate command, unresolved source gap, invalid hash, and wrong target.
- [x] Add Auditee projection tests after governed data exists, proving source
  reviews, manager deliberations, department memberships, other organizations'
  scopes, private CAA data, and internal audit events are absent.
- [x] Run mock and HTTP Playwright scenarios with an isolated profile at
  1440×900 and 390×844. Fail on console error, failed authorization expectation,
  missing/extra visible action, inaccessible dialog/control, or document-level
  horizontal overflow; clean task-owned browser/Vite processes afterward.
- [x] Run migration installation, version-20 upgrade, pre-data rollback, and
  post-data forward-repair tests against local PostgreSQL. Record literal row,
  constraint, version, and history-preservation results.
- [x] Record literal evidence counts and unresolved source/owner decisions.
- [x] After Task 8 behavior is verified, consolidate the operator-facing
  regulatory-change response in the existing canonical
  `docs/product-specs/workflows/AUDIT_CHECKLIST_WORKFLOW.md`: triggers,
  ownership, impact review, fail-closed gaps, new Draft/technical
  approval/publication, pinned in-progress Audits, forward repair, and required
  evidence. Link to that workflow from `docs/regulatory-sources/README.md`
  instead of duplicating the rules. Do not create a separate regulatory-change
  runbook unless real operational owners, notification channels, and SLAs
  require one.
- [x] Update this plan, the plan index, technical-debt tracker, product docs,
  architecture map, manifest, and a focused demo-evidence record.
- [x] Keep the outcome `candidate-only`, `release pending`, and
  `production-ready: not established`.

Primary verification files:

- `scripts/verify-governed-checklist-test-inventory.mjs`
- `tests/service-provider-catalog.test.mjs`
- `tests/regulatory-generation-contracts.test.mjs`
- `tests/governed-checklist-lifecycle-smoke.test.mjs`
- `api/openapi/tests/governed-checklist-contract.test.mjs`
- `api/openapi/tests/governed-checklist-publication-boundary.test.mjs`
- `apps/web/src/backend/governed-checklist-http-parity.test.ts`
- `apps/web/src/features/checklists/checklist-management-page.test.tsx`
- `apps/web/src/features/inspections/inspection-package-builder-page.test.tsx`
- `apps/web/tests/e2e/regulatory-checklist-governance.spec.ts`
- `apps/web/tests/e2e/regulatory-checklist-governance.http.spec.ts`

## Commands And Expected Observations

### Verification inventory preflight

```bash
node scripts/verify-governed-checklist-test-inventory.mjs
```

Expected: the command prints every required schema, migration, script, contract
test, Go package/test, React test, and Playwright spec with its expected suite
count, then exits 0. A missing path or zero-test runner is a hard failure. Do
not run the aggregate acceptance commands until this preflight passes.

### Admin publication boundary

```bash
./scripts/check-contracts.sh
node --test \
  api/openapi/tests/clean-state-creation.test.mjs \
  api/openapi/tests/governed-checklist-publication-boundary.test.mjs
cd apps/api
env GOCACHE=/private/tmp/avia-regulatory-go-cache \
  go test ./internal/application ./internal/httpapi ./internal/testprofile
```

Expected: normal OpenAPI, router, generated clients, mock, HTTP, and application
services expose no Admin direct-publish command. Historical bootstrap is
internal to the test profile and existing `CTV-CABIN-1` records still load.

### Catalog, source, and JSON contracts

```bash
node --test \
  tests/service-provider-catalog.test.mjs \
  tests/regulatory-generation-contracts.test.mjs \
  tests/ncaa-regulatory-source-sync.test.mjs \
  tests/ncaa-regulatory-derived-context.test.mjs
```

Expected: exactly 20 unique active provider codes; exact source labels/topics
and raw ownership; distinct AeMC/AME and ANSP/CNS/AIS/MET/SAR identities;
valid generation contracts; non-overlapping CC input/holdout partitions; and
stable source hashes.

### OpenAPI and generated transport parity

```bash
./scripts/check-contracts.sh
node --test \
  api/openapi/tests/contract-examples.test.mjs \
  api/openapi/tests/governed-checklist-contract.test.mjs \
  api/openapi/tests/governed-checklist-publication-boundary.test.mjs
```

Expected: OpenAPI source, generated Go, generated TypeScript, examples, mock,
and HTTP shapes agree with no drift; no Admin publication operation reappears.
The governed contract tests exercise complete payloads and invalid examples,
not only operation IDs or schema presence.

### Migration and recovery

```bash
cd apps/api
env GOCACHE=/private/tmp/avia-regulatory-go-cache \
  go test ./migrations ./tests/integration \
  -run 'Test(RegulatoryChecklistGovernance|GovernedChecklistMigrationRecovery|Department|OrganizationServiceProvider|RegulatedTarget|Task2|SessionManagerInjectsOnlyCurrentEffectiveDepartmentAssignments)' \
  -count=1
```

Expected: fresh install and version-20 upgrade reach migration 21 with the
expected tables, constraints, indexes, and catalog rows; pre-data rollback is
allowed; post-data rollback is refused; forward repair preserves source,
candidate, review, publication, and Audit-package history.

### Go domain and authorization

```bash
cd apps/api
env GOCACHE=/private/tmp/avia-regulatory-go-cache \
  go test \
  ./internal/identity \
  ./internal/organizations \
  ./internal/regulatory \
  ./internal/checklistgovernance \
  ./internal/configuration \
  ./internal/application \
  ./internal/assignments \
  ./internal/inspections \
  ./internal/httpapi
```

Expected: a manager without department membership is denied; an OPS manager
cannot approve AIR/PEL material; joint ownership waits for all approvals;
duplicate/stale commands are safe; Admin cannot approve/publish; published
versions and in-progress packages remain immutable. Tests inspect decision
rows, actor/scope/revision/reason/timestamp/idempotency fields, state
transitions, and content digests.

### React and root scenario checks

```bash
npm --prefix apps/web run typecheck
npm --prefix apps/web test -- \
  src/features/admin/admin-secondary-pages.test.tsx \
  src/features/organizations/organization-registry-page.test.tsx \
  src/features/checklists/checklist-management-page.test.tsx \
  src/features/inspections/inspection-package-builder-page.test.tsx \
  src/features/checklists/checklist-runner-page.test.tsx
npm --prefix apps/web run test:contract:http -- \
  src/backend/governed-checklist-http-parity.test.ts
npm --prefix apps/web run build:demo
npm --prefix apps/web run build:http
node --test \
  tests/governed-checklist-lifecycle-smoke.test.mjs \
  tests/checklist-management-smoke.test.js \
  tests/checklist-approval-smoke.test.js \
  tests/manager-checklist-management-smoke.test.js \
  tests/inspection-execution-smoke.test.js \
  tests/demo-boundary-smoke.test.js
```

Expected: Admin generation, Department Manager review/publication, organization
multi-scope selection, exact Audit pinning, and Inspector execution all pass in
mock and HTTP-capable surfaces without weakening the root legacy boundary.
Legacy smoke tests remain regression evidence only; the governed lifecycle and
HTTP parity tests are mandatory acceptance evidence.

### Browser scenario

```bash
npm --prefix apps/web run test:e2e:mock -- \
  tests/e2e/regulatory-checklist-governance.spec.ts
npm --prefix apps/web run test:e2e:http -- \
  tests/e2e/regulatory-checklist-governance.http.spec.ts
```

Expected: the Air Operator candidate moves through separate review and
publication actions, becomes selectable for the matching organization, and is
executed without console errors, authority leaks, or mobile overflow in both
mock and HTTP profiles. Negative role/scope attempts return the expected denial
and do not change persisted state. The positive path uses only the explicit
synthetic test-profile source package; the real source-bound pilot remains
blocked and creates no approval/publication effect. Run 1440×900 and 390×844
with an isolated test browser profile and clean up task-owned
Playwright/Vite/Chrome processes before handoff.

### Final repository gates

```bash
node tests/harness-docs-smoke.test.js
git diff --check
git status --short
```

Expected: plan/product/harness links are valid, no whitespace errors exist, and
only intentional files plus pre-existing unrelated user files are changed.
Compare final status to the captured initial status; do not infer workspace
integrity from `git diff --check`.

## Verification And Acceptance Criteria

The milestone is accepted only when all of the following are proven:

1. The catalog contains exactly the 20 supplied provider identities, with exact
   labels/topics/raw ownership and no unintended collapsing.
2. One organization can hold Air Operator + CAMO + ATO simultaneously, while
   inactive or expired scopes cannot select a new checklist.
3. A provider scope can correctly target non-organization subjects such as an
   FSTD device or AME person.
4. Every generation request/output validates against the canonical JSON
   Schemas, includes exact provider scopes, typed target, inspection type,
   source snapshots/hashes, and schema/policy/provider versions, and resolves
   every referenced identity.
5. Identical bounded inputs are content-addressed/idempotent; conflicting output
   for one digest fails visibly.
6. The generated Air Operator candidate is complete and contains both the
   crosswalk and practical checklist.
7. Every question has at least one mapping, exact source trace, verification
   method, expected Evidence, allowed answer set, and mandatory/safety flags.
8. AI/imported output cannot claim or cause technical approval or publication.
9. Department scope is enforced server-side. OPS cannot approve AIR/PEL;
   generic manager identity alone is insufficient, and current effective
   assignment is required.
10. Joint ownership blocks technical approval until all required decisions are
    present.
11. Technical approval and publication create separate, attributable audit
    events with actor, assignment scope, revision, content digest, reason,
    timestamp, operation ID, and idempotency key.
12. Admin's ordinary direct-publish path is absent from OpenAPI, routers,
    services, generated clients, mock, HTTP, and visible actions. Any retained
    historical bootstrap is internal, explicitly test-profile-only, and
    non-user-facing.
13. Publication creates a new immutable version; later source/candidate changes
    do not alter published versions or in-progress Audits, and publication
    fails unless persisted bytes recompute to the exact technically approved
    digest.
14. An applicable published checklist is selectable, assignable, runnable, and
    capable of producing an eligible Potential Finding through the existing
    lifecycle. Before criterion 16 is externally satisfied, positive-path proof
    uses only the explicit synthetic test-profile source package and is not
    evidence that the real OPS/AOC pilot is approved or published.
15. Auditee projections reveal only the auditee's organization-scoped
    operational records and no internal source review, manager deliberation, or
    other organizations' provider scopes.
16. The controlled NCAA Operations surveillance/ramp-inspection procedure,
    current Part 140 authority, and exact Part 127 applicability are confirmed
    by the responsible source owner/Department Manager before the pilot is
    called technically approved or published.
17. Final evidence remains literal: `verified locally` only after fresh checks;
    otherwise `not run` or `blocked`. The verification-inventory preflight,
    expected suite counts, mock E2E, HTTP E2E, and migration/recovery checks all
    pass. No production-readiness claim is made.

## Required Evidence Matrix

Every acceptance criterion has a primary proof artifact. Passing adjacent or
predecessor tests does not substitute for the named proof.

| Criterion | Primary required proof |
|---:|---|
| 1 | `tests/service-provider-catalog.test.mjs` exact-value and collapse-negative cases |
| 2 | Go/PostgreSQL multi-scope tests plus UI selection test for active/expired scopes |
| 3 | Typed-target Go contract tests for FSTD, AME, facility, system, asset, and location |
| 4 | `tests/regulatory-generation-contracts.test.mjs` valid and invalid schema corpus |
| 5 | Generation service/import tests for identical replay and conflicting content |
| 6 | Persisted Air Operator candidate fixture inspected for both linked output arrays |
| 7 | Per-question schema plus behavioral import tests for all required trace/answer/flag fields |
| 8 | Import-state rejection tests and Admin approval/publication denial tests |
| 9 | Service and HTTP authority matrix using current department assignments |
| 10 | Joint-owner partial/complete/stale/edit-invalidated decision tests |
| 11 | Database assertions for two separate commands, timestamps, decisions, and audit events |
| 12 | OpenAPI/router/generated-client absence test plus normal Admin HTTP/service denial |
| 13 | Digest recomputation, source-change, later-publication, and in-progress Audit immutability tests |
| 14 | Synthetic test-profile mock and HTTP E2E from applicable selection through eligible Potential Finding, plus real-pilot approval/publication denial |
| 15 | Auditee projection tests seeded with internal reviews, deliberations, multiple organizations, and private CAA records |
| 16 | Recorded source-owner/Department Manager confirmations bound to exact source versions and applicability decision |
| 17 | Inventory preflight, literal command log, initial/final status comparison, and explicit claim boundary |

## Risks And Dependencies

- The supplied scope matrix names inspectorates, offices, units, and
  departments inconsistently. Raw labels are preserved; normalized approval
  ownership remains blocked until NCAA confirms ambiguous rows.
- Current Part 140 authority/supersession, exact Part 127
  operation/configuration applicability, and the controlled OPS surveillance
  procedure remain source-owner dependencies.
- CC quality may contain historical interpretations or stale national
  references. It is never silently promoted to primary current law.
- Benchmark leakage would make quality scores meaningless. Input and holdout
  identities are immutable and disjoint.
- One checklist per provider type is too coarse. Module composition and typed
  applicability prevent a monolith but add review complexity.
- A model may produce plausible unsupported text. Schema validation alone is
  insufficient; exact citations, fail-closed gaps, holdout evaluation, and
  Department Manager review remain mandatory.
- FSTD, AME, facilities, systems, and locations expose target-model gaps if the
  implementation assumes every regulated subject is an organization.
- The current Go Admin path can immediately create a `PUBLISHED` version. It
  must be closed before the new workflow can satisfy authority acceptance.
- Existing predecessor artifacts use `EXPERT_REVIEW_REQUIRED`; migration must
  preserve their readability without inventing approvals.

## Idempotence And Recovery

- Catalog imports upsert only by stable catalog version/code and reject
  conflicting content for the same digest.
- Source versions, generation inputs/outputs, review decisions, publications,
  and Audit package snapshots are append-only records.
- Repeating a generation import with the same idempotency key and identical
  digest returns the existing run; a different digest is a conflict.
- Repeating a review or publication command returns the recorded decision only
  when actor, revision, scope, and payload agree.
- A returned candidate is revised by creating a new revision, not overwriting
  the reviewed one.
- Migration rollback is allowed only before new governed records are accepted;
  afterward, recovery is forward repair because published and review history
  must not be deleted.
- The version-21 migration records whether governed records have been accepted.
  Its rollback test must refuse destructive reversal after that point; recovery
  creates additive repair migrations and never edits the applied migration or
  deletes decision/publication/source/Audit history.
- A publication retry returns the existing publication only when actor,
  effective assignment, candidate revision, approved digest, reason, operation
  identity, and idempotency identity all match. Any difference is a conflict
  and creates no version or audit event.
- A source-impact retry resolves to the same impact record for the same old/new
  source hashes. Different source bytes or affected-clause sets require a new
  impact identity.
- Failed or interrupted Codex/model work leaves a failed/incomplete generation
  run with no candidate publication effect. It can be retried from the exact
  stored request.
- Existing `CTV-CABIN-1` and Audits that reference it remain historical
  immutable demo records; the migration does not retroactively mark them
  technically approved.

## Progress

- 2026-07-29: The user confirmed that the product must generate usable
  checklists for the supplied service-provider scopes, not merely store source
  documents.
- 2026-07-29: The user selected the responsible Department Manager as both the
  department-scoped technical approver and publication owner, with approval and
  publication retained as separate decisions.
- 2026-07-29: The supplied two-page service-provider matrix was inspected and
  normalized into 20 distinct candidate catalog codes. AeMC and AME are
  separate; ANSP, CNS, AIS/AIM, MET, and SAR are not collapsed.
- 2026-07-29: Repository audit confirmed that the source/OCR/refresh and
  six-question OPS/AOC pilot foundations are already `verified locally`, while
  department scope, multi-provider organizations, generation runs, manager
  review, and manager publication are not implemented.
- 2026-07-29: Planning complete. No implementation or runtime verification has
  started under this plan.
- 2026-07-29: An independent read-only regulatory-governance review returned
  local `NO-GO`. Fresh checks confirmed the ordinary Admin direct-publication
  path remains, the planned catalog/schemas/migration/domain packages/scripts
  and focused E2E are absent, and aggregate Node/Vitest commands can pass while
  named governed test files are missing.
- 2026-07-29: The plan was revised to add security-first Gate 0, exact closure
  ownership for every P0/P1/P2 finding, migration/recovery proof, complete
  negative authority/lifecycle matrices, mock and HTTP browser evidence, an
  explicit verification-inventory preflight, and a criterion-to-proof matrix.
  Implementation remains `not run`.
- 2026-07-29: Gate 0 `verified locally`. Focused tests first failed because
  `POST /v1/admin/checklist-template-versions`, its generated clients, router,
  and exported application service still existed. The normal OpenAPI path and
  inputs, router/handler, direct-publish service, generated Go/TypeScript
  transports, and full-profile test caller were removed. The internal
  `testprofile.Reset` bootstrap remains the sole checked fixture creator for
  historical `CTV-CABIN-1`; it has no normal OpenAPI/router/generated-client or
  visible-action registration and no synthetic technical-approval decision.
  Fresh results: `./scripts/check-contracts.sh` exit 0; the two focused Node
  boundary files 5/5 pass; `go test ./internal/application ./internal/httpapi
  ./internal/testprofile` exit 0; focused historical-demo Vitest 28/28 pass;
  Web typecheck and `git diff --check` exit 0. Work remains `candidate-only`,
  `release pending`, and `production-ready: not established`.
- 2026-07-29: Gate 0 fix round 1 `verified locally`. The local full-profile
  harness now invokes a one-shot `canonicaltest` binary outside the normal API
  router to call `testprofile.BootstrapHistoricalFullProfileChecklist` before
  browser work. The bootstrap creates only the historical `CTV-CABIN-1` /
  `TPL-CABIN-2026` version-1 relationship and its six ordered question versions;
  it creates no approval decision or audit event. A real isolated PostgreSQL
  helper test passed after the initial focused red compile failure for the
  missing internal bootstrap. The broader Docker full-profile test was started
  but interrupted before its bootstrap/E2E phase; its exact task-owned Compose
  project was removed, so that broader result is `not run`.
- 2026-07-29: Gate 0 fix round 2 `verified locally`. The internal historical
  bootstrap now preflights its complete immutable identity, template snapshot,
  six exact `QV-<question>-V1` question versions, ordered relationships, and
  template-master owner/revision/pointer before any write. An identical
  existing fixture is accepted idempotently; a partial or semantically
  conflicting fixture fails closed in one transaction with no added
  relationship side effect. New focused real-PostgreSQL integration coverage
  proved exact question-version identity plus identity/template/question/master
  conflict rejection. The amended real-PostgreSQL harness helper, its compiled
  binary, shell syntax, contract, normal Go, focused web, typecheck, and diff
  gates all passed. The broader Docker full-profile test remains `not run`.
- 2026-07-29: Task 1 `verified locally`. Added the tracked
  `service-provider-catalog.v1.json` and Ajv-validated canonical schema with
  exactly the supplied 20 identities, exact labels/topics/raw owner labels,
  aliases, target kinds, and fail-closed `REVIEW_REQUIRED` ambiguity handling
  for every raw `+` or `/` owner label. Product authority now assigns separate
  technical approval and publication decisions to the responsible Department
  Manager; new `TECHNICAL_REVIEW_REQUIRED` records coexist with explicit
  readable legacy `EXPERT_REVIEW_REQUIRED` compatibility. Fresh results:
  focused Task 1 Node tests 3/3; catalog/source regression 13/13; harness-docs
  smoke exit 0; OpenAPI contracts 16/16; web typecheck exit 0; `git diff
  --check` exit 0. Work remains `candidate-only`, `release pending`, and
  `production-ready: not established`.
- 2026-07-29: Task 1 fix round 1 `verified locally`. Ambiguous raw `+` and
  `/` owners now omit the relationship and every owner field; the schema
  rejects any semantic relationship or owner inference for those rows. Each
  provider definition now fixes its exact target kinds and responsibility
  object, so a normalized owner or target-kind reassignment also fails closed.
  The historical source-bound OPS/AOC test profile now uses responsible
  Department Manager terminology while retaining its legacy
  `EXPERT_REVIEW_REQUIRED` status and no approval history. Fresh results:
  focused Task 1 tests 3/3; catalog/source regression 13/13; Go testprofile
  and HTTP API packages passed; harness-docs smoke exit 0; OpenAPI contracts
  16/16; web typecheck exit 0; `git diff --check` exit 0. Work remains
  `candidate-only`, `release pending`, and `production-ready: not established`.
- 2026-07-29: Task 2 `verified locally`. Migration 21 adds immutable
  department/unit, provider catalog/topic/responsibility, typed-target, and
  multi-scope organization facts while retaining the coarse organization type.
  It seeds exactly 20 tracked providers, preserves ambiguous `+`/`/` ownership
  as `REVIEW_REQUIRED` with no inferred approval owner, resolves only current
  Department Manager department/unit assignments into authenticated principals,
  and supplies a CAA-side active-scope selection seam. Real PostgreSQL tests
  prove the version-20 upgrade, pre-data guarded rollback, post-data rollback
  refusal, forward preservation of representative source/review/published
  template/Audit-package history, active Air Operator + CAMO + ATO scopes,
  expiry exclusion, typed FSTD/AME identities, and Auditee projection
  isolation. The focused migration/identity/organizations/integration gate,
  relevant application/HTTP regressions, contract check, harness-docs smoke,
  and diff check passed. Work remains `candidate-only`, `release pending`, and
  `production-ready: not established`.
- 2026-07-29: Task 2 fix round 1 `verified locally`. Immutable membership and
  provider-scope facts now use one-successor chains, so revocation, suspension,
  expiry, renewal, and correction do not mutate history and only the effective
  successor resolves. Composite department/unit integrity, active-unit joins,
  authenticated-session assignment injection, same-kind cross-organization
  target denial, migration-owned baseline protection, adopted-history rollback
  refusal, and idempotent version-21 derived-index repair are behaviorally
  covered. The database inventory now compares exact provider source values,
  target kinds, topic order, normalization, responsibility owners, semantic
  constraints, governed triggers, and index definitions. Fresh results:
  focused real-PostgreSQL Task 2 suite 10/10; identity, organizations, session,
  application, and HTTP API packages passed; forward-only migration policy
  passed; OpenAPI contracts 16/16; harness-docs smoke and `git diff --check`
  passed. Work remains `candidate-only`, `release pending`, and
  `production-ready: not established`.
- 2026-07-29: Task 2 fix round 2 `verified locally`. Membership roots are now
  unique per subject/department/unit chain, public append-only department/unit
  status facts determine effective authority, and successor-aware indexes lead
  with subject/organization and root identity. Repair replaces a missing or
  wrong derived applicability index without changing migration version 21 or
  governed history. A real authenticated Auditee HTTP request excludes
  provider scopes and internal membership data. The corrected focused command
  selected and passed all 12 Task 2 tests, including `TestTask2*`; identity,
  organizations, platform/session, application, HTTP API, migration policy,
  contracts (16/16), harness docs, and diff checks passed. Work remains
  `candidate-only`, `release pending`, and `production-ready: not established`.
- 2026-07-29: Task 2 fix round 3 `verified locally`. Full catalog inventory
  now verifies each governed trigger's relation, timing, event mask, and
  function, and exact status-fact indexes and semantics. Derived-index repair
  compares normalized complete definitions, corrects a same-column unique
  partial impostor, is idempotent, and preserves representative source/review/
  template/Audit/membership/scope history without approval synthesis. The
  focused gate selected 13 tests, including the session-assignment test; all
  package, migration-policy, contract (16/16), harness, and diff gates passed.
  Work remains `candidate-only`, `release pending`, and
  `production-ready: not established`.
- 2026-07-29: Task 2 fix round 4 `verified locally` and independently
  APPROVED with no findings. The inventory now names both immutable
  status-root uniqueness indexes and both effective indexes and compares their
  complete normalized definitions. Forward-repair proof compares complete
  review, department-membership, and provider-scope rows before and after
  repair in addition to source/template/Audit-package snapshots and digest.
  The focused real-PostgreSQL Task 2/session gate, five Go package regressions,
  guarded migration policy, contracts (16/16), harness documentation smoke,
  and diff check passed. The task-owned PostgreSQL instance was removed after
  review. Work remains `candidate-only`, `release pending`, and
  `production-ready: not established`.
- 2026-07-29: Task 3 `verified locally`. Migration 21 now persists immutable
  versioned regulatory sources and normalized clause locators without source
  text, including the supplied Annex 6 Part I CC working-copy metadata and
  five stable Annex/section rows. Database constraints partition stable CC row
  identities between generation input and blind holdout. Immutable generation
  runs pin canonical digests, schema/policy/provider-catalog versions, exact
  provider scopes, inspection target, source snapshot hashes/locators,
  request/output artifacts, and explicit failure state. Generated candidates,
  required owners, Department Manager review decisions, and separate
  publication decisions pin exact candidate revisions/digests, actor assignment,
  reason, timestamp, operation, idempotency, and semantic payload identities.
  The database rejects invalid hashes, unknown source/scope/target identities,
  partition overlap, conflicting input/output, changed idempotency payloads,
  unapproved publication, and mutation/deletion of governed facts. Fresh real
  PostgreSQL focused Task 3, Task 2/3 recovery, source/catalog, Go regression,
  OpenAPI contract, harness-documentation, and diff checks passed. The
  candidate-only OPS/AOC source/owner gaps and ambiguous ownership remain
  `blocked`; `production-ready: not established`.
- 2026-07-29: Task 3 fix round 1 is `ready-for-verification`, not accepted.
  Independent review correctly found that the original persistence proof did
  not pin exact effective provider-scope facts, resolve effective membership
  successors/status facts, prove a complete generated candidate, or inspect
  full semantic catalog/recovery behavior. Migration 21 and real-PostgreSQL
  tests now bind generation runs to exact current scope root/successor,
  authorization, effective period, organization-compatible target, source
  snapshot, nonempty question-version identity set, matching output digest,
  and immutable candidate revision chain. Decision guards resolve effective
  membership successors and department/unit status at decision time. Positive
  technical approval, separate publication, and immutable published-template
  binding are read back exactly; adopted CC clauses/rows beneath the seeded
  source refuse rollback; and CTV-CABIN-1's real Audit-package binding is
  asserted. The unfiltered integration gate was run against task-owned local
  PostgreSQL. Task 4 remains prohibited until independent re-review accepts
  this fix round; `candidate-only`, `release pending`, and
  `production-ready: not established` remain literal.
- 2026-07-29: Task 3 fix round 2 is `ready-for-verification`, not accepted.
  Migration 21 now removes the contradictory type-only run/provider table and
  uses only exact effective scope facts. Generation input and output digests
  are canonical JSONB SHA-256 content addresses, not shape-only hash strings.
  Decision authority uses null-safe status comparisons and fails closed for
  missing or inactive department/unit facts. Independent negative cases cover
  each exact candidate-lineage invariant; the semantic inventory reads all
  required function bodies and Task 3 constraints/indexes. The precise
  fix-round-2 evidence is recorded in the SDD review package. Fresh focused,
  combined, source/catalog, contract, harness, diff, and unfiltered local
  integration gates were run. Task 4 remains prohibited until independent
  re-review accepts Task 3; work remains `candidate-only`, `release pending`,
  and `production-ready: not established`.
- 2026-07-29: Task 3 fix round 3 is `ready-for-verification`, not accepted.
  The catalog evidence now pins SHA-256 hashes of complete normalized function,
  sorted constraint, and index definitions across the full Task 3 and critical
  authority surface; it explicitly includes all generation-run content-digest,
  state-shape, artifact, and version constraints. A fresh PostgreSQL RED
  confirmed unpinned inventory, then the exact inventory and all required
  local gates passed. The corrected artifact inventory and literal unfiltered
  integration evidence are in the Task 3 report/review package. Task 4 remains
  prohibited until independent re-review; `candidate-only`, `release pending`,
  and `production-ready: not established` remain literal.
- 2026-07-29: Task 3 fix round 3 independently APPROVED. The final re-review
  accepted all five complete guard-function hashes, 23 complete sorted
  constraint surfaces, nine complete critical index definitions, corrected
  evidence inventory, and every previously closed content-addressing,
  exact-scope, effective-authority, publication, rollback, repair-history, and
  historical Audit-binding behavior. Task 3 is `verified locally` and
  complete. The next action is Task 4; work remains `candidate-only`,
  `release pending`, and `production-ready: not established`.
- 2026-07-29: Task 4 is `ready-for-verification`. Canonical closed schemas,
  deterministic/imported providers, bounded request/validation/import commands,
  and an internal synthetic test-profile source package now prove one complete
  candidate/import seam. A transaction persists exact scope/source-clause
  lineage, immutable question version, generated candidate draft, and required
  owner; replay is stable and a conflicting output or absent prerequisite rolls
  back without candidate, decision, publication, or Audit effects. Fresh
  focused Task 4, source/catalog/contracts/CLI, and unfiltered integration
  checks passed on a task-owned PostgreSQL instance. The real OPS/AOC pilot
  remains `blocked`; no external provider, credential, OpenAPI, UI, review, or
  publication path was added. Work remains `candidate-only`, `release pending`,
  and `production-ready: not established`.
- 2026-07-29: Task 4 fix round 1 is `ready-for-verification`, not accepted.
  The real OPS/AOC request is a schema-valid exact source/hash/clause/locator
  and CC-row binding with three closed unresolved gaps; Node and Go validate it
  before deterministically blocking any candidate or persistence effect.
  Synthetic imports pin generation-input (not holdout) rows and complete
  immutable mapping/question snapshots. Replay now reads the complete stored
  graph and fails on missing or conflicting facts. Fresh provider, schema/CLI,
  Task 4 PostgreSQL, combined Tasks 2–4, Task 3 catalog, and local CLI
  import/replay checks passed. Work remains `candidate-only`, `release
  pending`, and `production-ready: not established`.
- 2026-07-29: Task 4 fix round 2 is `ready-for-verification`, not accepted.
  A test-profile-only predecessor bootstrap resolves every real blocked
  OPS/AOC organization, target, scope, CC source/clause/locator, and disjoint
  partition identity without creating any workflow artifact. Synthetic claims
  now use one exact supported registry rather than phrase denial. Replay checks
  exact scope/source/partition/candidate/snapshot/owner values and rejects
  same-count substitutions. The checked-in PostgreSQL integration invokes the
  actual Node CLI twice against its task-created loopback database and inspects
  the graph. Fresh focused, combined, Gate 0, docs, and unfiltered checks are
  `verified locally`; the real pilot remains `blocked`, `candidate-only`, and
  `release pending`.
- 2026-07-29: Task 4 fix round 3 is `ready-for-verification`, not accepted.
  A focused RED showed recomputed arbitrary certification/enforcement text in
  synthetic verification method and expected Evidence was accepted. Node and
  Go now pin requirement, rationale, prompt, verification method, expected
  Evidence, and synthetic source-gap shape to the same bounded exact registry.
  Positive fixture parity and every free-text paraphrase negative are
  `verified locally`; focused PostgreSQL/CLI, combined, Gate 0, documentation,
  and unfiltered checks remain green. The real pilot is still `blocked`,
  `candidate-only`, and `release pending`.
- 2026-07-29: Task 4 fix round 3 independently APPROVED. Recomputed mutations
  of requirement, rationale, prompt, verification method, expected Evidence,
  and source-gap semantics are rejected in Node and Go; persisted blocked-real
  resolution, exact replay values, checked-in Node-to-Go CLI proof, Gate 0, and
  the no-Task-5 boundary remain intact. Task 4 is complete. The next action is
  Task 5; the real pilot remains `blocked`, `candidate-only`, and `release
  pending`.
- 2026-07-29: Task 5 fix round 2 is `ready-for-verification`, not accepted.
  Root-scoped transaction locking closes same-leaf races; successful and
  failed import graphs are atomic and replay-safe; Task 4 Node/Go import parity
  is restored; clean-state React preserves immutable run output; shared
  canonical digests and exact issue vectors align Go, mock, and HTTP; and a
  genuine already-ledgered pre-Task-5 version-21 fixture proves complete,
  idempotent repair without governed-history mutation. Fresh focused gates and
  the unfiltered migration/integration suite passed with task-owned local
  fixtures. Task 6 remains prohibited pending independent review and separate
  authorization; work remains `candidate-only`, `release pending`, and
  `production-ready: not established`.
- 2026-07-29: Task 5 fix round 3 is `ready-for-verification`, not accepted.
  Edit and submit re-check exact command replay after acquiring the
  candidate-root lock, so deterministically overlapping identical callers
  return one successful result and persist one command/Audit effect while
  changed semantics still conflict. The mock run follows edit/submission to the
  exact current leaf while immutable imported run input/output fields remain
  unchanged. A checked-in shared fixture is now exercised by both TypeScript
  mock tests and an actual canonical Go handler backed by task-owned
  PostgreSQL. A delayed post-import reload regression also proves that an
  in-progress Admin rationale edit is not overwritten. Fresh focused, full
  integration, full web, contract, authorization, typecheck, build, artifact,
  documentation, and diff gates passed. Task 6 remains prohibited pending
  independent review and separate authorization; work remains
  `candidate-only`, `release pending`, and `production-ready: not established`.
- 2026-07-29: Task 5 fix round 3 independently APPROVED. Overlapping
  byte-identical edit and submit commands replay one effect after the root
  lock; changed semantics conflict without effect. Mock and the actual
  PostgreSQL-backed canonical Go handler share exact validation, successor,
  leaf, run-state, and immutable-output behavior. The delayed reload edit guard
  is accepted. All prior Task 5 source, import, validation, HTTP, audit, repair,
  React, and Gate 0 findings remain closed. Task 5 is complete; the next action
  is Task 6.
- 2026-07-29: Task 6 fix round 1 is `ready-for-verification`, not accepted.
  Global latest-root effective authority and queue confidentiality, under-lock
  exact-state command revalidation, genuine already-ledgered version-21
  repair, exact mock/joint-owner/blocker parity, same-suite blocked-real zero
  effects, complete selectable React queue and terminal detail, and immutable
  multi-mapping/multi-question publication bytes now have focused RED/GREEN
  evidence. The final unfiltered canonical integration suite passes in
  217.245s; complete Task 6 plus semantic catalog passes in 28.098s; full web
  passes 67 files / 669 tests. The real OPS/AOC request remains `blocked`;
  work remains `candidate-only`, `release pending`, and `production-ready: not
  established`. Task 7 remains prohibited pending independent Task 6
  acceptance.
- 2026-07-29: Task 6 fix round 2 is `ready-for-verification`, not accepted.
  Deterministic barriers now cover different-owner approval, return/reject,
  successor/stale approval, and command-identity interleavings with exact
  winner/loser effects. Current authority denial covers inactive/expired
  membership and inactive department/unit. A frozen pre-Task-6 v21 fixture
  proves complete idempotent repair while preserving seeded history. The mock
  persists separate command, review, publication, Audit, and immutable
  snapshot semantics; canonical HTTP, mock, and React use one typed exact
  blocked-generation result with four unresolved fact identities and zero
  lifecycle effects. React refreshes active queue state after terminal
  transitions while preserving authorized terminal detail. Stable mapping
  ordinals preserve reverse non-lexical mapping bytes through publication.
  Fresh local verification passes complete Task 6 plus semantic catalog in
  43.885s, migration/recovery in 17.663s, corrected unfiltered integration in
  240.152s, full web at 67 files / 672 tests, contracts 16/16, root legacy
  103/103, both builds, artifact scans, docs smoke, demo boundary, and diff
  validation. Exact evidence is in
  `../../../.superpowers/sdd/2026-07-29-governed-service-provider-checklist-generation-plan/task-6-fix-round-2-review-package.md`.
  Work remains `candidate-only`, `release pending`, and `production-ready: not
  established`; the real OPS/AOC request remains `blocked`. Task 7 remains
  prohibited pending independent Task 6 acceptance.
- 2026-07-29: Task 6 fix round 3 is `ready-for-verification`, not accepted.
  Approval returns the exact result committed inside the same root-locked
  transaction, including deterministic partial-owner `DEPARTMENT_REVIEW` and
  final-owner `TECHNICALLY_APPROVED` responses. The frozen v21 repair fixture
  pins literal hashes for every normalized Task 6 column, constraint, index,
  trigger, and function body. Manager mock and actual HTTP read boundaries
  expose exact decision, command, actor-assignment, semantic-payload, Audit,
  immutable publication, mapping, and question artifacts with replay/conflict
  parity and no public mutation helper. Version-21 repair now walks candidates
  root-to-leaf, derives root order from exact generation output, inherits
  unchanged successor ordinals, appends new identities, verifies the canonical
  successor digest, and rolls back unrecoverable order without history
  mutation. Fresh verification passes the complete Task 6 slice in `50.428s`,
  Gate 0 / Tasks 3–5 compatibility in `67.953s`, unfiltered canonical
  integration with PostgreSQL and MinIO in `261.887s`, full web at 67 files /
  673 tests, contracts 16/16, root regressions 321/321, domain/auth/HTTP
  packages, typecheck, both builds, artifact scans, docs smoke, and diff
  validation. Exact evidence is in
  `../../../.superpowers/sdd/2026-07-29-governed-service-provider-checklist-generation-plan/task-6-fix-round-3-review-package.md`.
  Work remains `candidate-only`, `release pending`, and `production-ready: not
  established`; the real OPS/AOC request remains `blocked`. Task 7 remains
  prohibited pending independent Task 6 acceptance.
- 2026-07-29: Task 6 fix round 4 is `ready-for-verification`, not accepted.
  Concurrent different-owner approval now proves four independent exact row
  sets—decisions, Audits, required-owner assignments, and governed
  commands—so an extra or unlinked row cannot escape an inner join. One
  checked-in parity contract drives both the semantic mock and the actual
  PostgreSQL-backed canonical handler, pinning approval-only `1/0/0/1`,
  joint-complete `2/0/0/2`, and published `2/1/1/3` effect counts together
  with full identities, replay/conflict behavior, digest tamper, exact ordered
  bytes, and immutable update/delete denial. The frozen pre-Task-6 v21 SQL now
  contains the reviewed edited successor, inherited reverse/non-lexical
  `[MAP-Z-FROZEN-FIRST, MAP-A-FROZEN-SECOND]`, and recoverable appended
  `MAP-N-FROZEN-APPENDED`; repair preserves its stored digest, projection and
  publication bytes, and reviewed history while the unrecoverable negative
  remains failure-atomic. The normalized inventory wording is corrected to
  the six named reviewed relations plus seven function bodies. Fresh
  verification passes Task 6 plus semantic catalog in `24.846s`,
  migration/Gate 0/Tasks 2–5 compatibility in `31.581s`, unfiltered canonical
  integration in `90.025s`, full web at 67 files / 674 tests, contracts
  16/16, Task 4–6 CLI/catalog checks 21/21, Gate 0 contracts 5/5, root
  regressions 321/321, Go domain/authorization/HTTP packages, typecheck, both
  builds, artifact scans, docs smoke, demo boundary, and diff validation.
  Exact evidence is in
  `../../../.superpowers/sdd/2026-07-29-governed-service-provider-checklist-generation-plan/task-6-fix-round-4-review-package.md`.
  Work remains `candidate-only`, `release pending`, and `production-ready: not
  established`; the real OPS/AOC request remains `blocked`. Task 7 remains
  prohibited pending independent Task 6 acceptance.
- 2026-07-29: Task 6 fix round 5 is `ready-for-verification`, not accepted.
  The checked-in manager artifact contract now contains complete literal review
  decision, Audit, publication-decision, and checklist-version rows; both the
  semantic mock and the actual PostgreSQL-backed canonical handler select and
  byte-compare those same exact checkpoint rows, rejecting extra, missing,
  unlinked, or mutated artifacts. The checked-in frozen-successor contract now
  pins literal pre/post decision history, complete Audit bytes, ordered question
  snapshots, repaired candidate projection and digest, publication identity,
  and the complete published mapping/question snapshot. Focused fresh results:
  semantic mock parity 10/10 in 0.287s and canonical PostgreSQL integration
  4/4 in 3.665s. The broader fix-round-5 verification results are recorded in
  the round-5 review package. Work remains `candidate-only`, `release pending`,
  and `production-ready: not established`; the real OPS/AOC pilot remains
  `blocked` with zero lifecycle effects. Task 7 remains prohibited pending
  independent Task 6 acceptance.
- 2026-07-29: Task 6 fix round 5 independent review returned `APPROVED` with
  no Critical or Important findings. It verified that mock and canonical tests
  consume the same complete literal row contract and reject extra, missing,
  unlinked, duplicate, or mutated artifacts; it also verified exact successor
  pre/post decision/Audit history, the sole permitted root repair, ordered
  mapping/question publication bytes, digest recomputation, and version linkage.
  Task 6 is complete and Task 7 is authorized. The real OPS/AOC pilot remains
  `blocked`; work remains `candidate-only`, `release pending`, and
  `production-ready: not established`.
- 2026-07-29: Tasks 7 and 8 independently completed. Task 7 selects only
  current active scope facts with matching provider, responsible department,
  inspection type, typed target, effective date, and exact qualifiers; it
  composes immutable ordered package snapshots and keeps later source/scope
  changes from mutating an in-progress Audit. Task 8 persists append-only
  synthetic source-impact and disjoint holdout records while pinning exact
  unchanged review/publication/Audit/package history. No migration 22 was
  added. Both tasks remain `verified locally`, `candidate-only`, and `release
  pending`; the real OPS/AOC pilot remains `blocked`.
- 2026-07-29: Task 9 independently completed. The verification inventory
  reports 24 required artifacts before any aggregate runner. Positive mock and
  HTTP browser proof is restricted to a separately materialized internal
  synthetic governed package; it pins exact selection, version/digest, and
  question identity, uploads required controlled-record Evidence before an
  eligible Potential Finding, and submits at 1440×900 and 390×844. Mock
  publication does not create a package; an explicit mock-only test seam
  mirrors the canonical HTTP applicability materializer. The HTTP Vitest
  configuration explicitly includes the governed transport test. Auditee
  isolation and the real OPS/AOC zero-effect denial remain intact. Task 9 is
  `verified locally`, `candidate-only`, and `release pending`;
  `production-ready: not established`. The next action is final whole-plan
  verification and evidence handoff.

## Decisions

- Use a governed generator, not free-form chat, with persisted JSON inputs and
  outputs.
- Support Codex-assisted batch generation first through the same validated
  import contract; keep an external in-product model adapter optional and
  separately authorized.
- Build all 20 taxonomy/ownership identities, then prove one complete Air
  Operator path before domain rollout.
- Reuse predecessor source, mapping, refresh, adaptive-scope, checklist
  snapshot, and execution foundations.
- Preserve the existing `manager` role and add department/unit scope rather
  than creating a separate Technical Expert role.
- Fail closed on unresolved source gaps, ambiguous required ownership, stale
  revisions, and unknown identities.

## Discoveries

- The source matrix contains exactly 20 provider rows across two pages, not 19.
- Several entries are oversight scopes rather than organization types: FSTD is
  a device and AME is a person.
- The raw owner labels contain ambiguous `+` and `/` relationships that cannot
  safely be converted to approval rules without NCAA confirmation.
- Current runtime identity has roles and organization scope but no department
  membership.
- Current organizations have one coarse `organization_type`; regulatory
  mappings use free-form service-provider strings.
- Current React manager checklist management is read-only, while ordinary Go
  publication is Admin-only. This conflicts with the agreed product authority.
- Existing immutable checklist versions and Audit package pinning are the
  correct foundation and should be extended rather than replaced.
- Gate 0 removed the ordinary Admin publication path from OpenAPI, the
  canonical router, generated transports, and the application service. The
  normal POST now returns 404; UI-only disabling is not treated as an
  authorization control.
- Required `service-provider-catalog` and
  `regulatory-generation-contracts` Node test paths can be absent while the
  aggregate `node --test` command still exits 0 after running predecessor
  suites. Vitest likewise omitted two absent named files. Final verification
  requires an explicit inventory preflight and expected suite counts.
- The current Audit workspace verifies a released planning item and matching
  template ID/version, then pins a snapshot. It does not yet enforce active
  provider scope, department, inspection type, typed target, qualifiers, or
  effective period.

## Outcome Notes

Gate 0 is `verified locally`; it removes the P0 ordinary Admin publication
bypass but does not establish the governed publication workflow. Current
evidence establishes a locally verified regulatory-source and OPS/AOC
predecessor foundation only.
The governed 20-entry taxonomy and Department Manager technical
approval/publication authority language are `verified locally`. The controlled
Task 4 generation/import path is `verified locally` only for the explicit
synthetic test profile; it is `candidate-only` and `release pending`. The real
OPS/AOC request remains `blocked` before candidate creation because the
controlled procedure, Part 140 authority, and Part 127 applicability are
unresolved.

Completing the repository tasks is necessary but not sufficient to publish the
pilot. Acceptance criterion 16 remains `blocked` until the controlled NCAA OPS
surveillance/ramp-inspection procedure, current Part 140
authority/supersession, exact Part 127 applicability, and ambiguous mandatory
ownership are confirmed by the recorded responsible source owner/Department
Manager. Official content for the remaining provider scopes separately depends
on complete source packages and responsible managers. Production and release
evidence remain outside this plan.

## Execution Prompt

Independently review Task 6 fix round 4 against
`.superpowers/sdd/2026-07-29-governed-service-provider-checklist-generation-plan/task-6-fix-round-4-brief.md`
and its review package. Verify the four independent exact concurrent-approval
inventories, the one shared semantic-mock/actual-handler artifact parity
contract, and the checked-in three-mapping edited-successor repair plus
failure-atomic unrecoverable negative. Preserve the accepted complete
normalized frozen-v21 inventory, the explicit synthetic-only positive path,
and the blocked real OPS/AOC zero-effect proof. Do not begin Task 7, create a
Technical Expert role, change the root legacy demo, alter historical published
versions, commit, push, deploy, or access production systems without separate
authorization.

## Final Outcome

Completed 2026-07-29. Gate 0 and Tasks 1–9 received their required independent
acceptance. Final local verification recorded the fail-closed inventory at
24/24 artifacts; source/catalog contracts at 18/18; focused OpenAPI at 21/21;
migration/recovery integration at 12.280s; focused React at 30/30; root smoke
at 6/6; synthetic mock browser at 2/2 across 1440×900 and 390×844; and the
governed HTTP transport through the literal configured command at 1/1. The
full unfiltered HTTP-profile invocation did not yield a controller-captured
final result after its long race phase and is `not run` as aggregate evidence;
the focused synthetic HTTP profile is `verified locally` and was independently
reviewed. No production, external NCAA confirmation, or official 20-library
claim is made. The real OPS/AOC pilot remains `blocked`, the implementation is
`candidate-only`, release is `release pending`, and `production-ready: not
established`.
