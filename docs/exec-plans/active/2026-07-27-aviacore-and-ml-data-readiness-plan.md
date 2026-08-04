# AviaCore And ML Data Readiness Plan

**Status:** `in progress` — Tasks 1–2 and separately authorized Task 3A/3B
are `verified locally` and independently accepted. Task 4, including explicit
authorization for forward-only migration `000022`, has one source-consistent
workspace transition implemented, `verified locally`, and independently
accepted. The AviaCore v3 machine-contract semantic mismatch is repaired and
independently accepted locally; Task 5 is independently accepted and
`verified locally`.
Every AviaCore Phase 2.3+ slice remains separately gated.

**Reviewed task count:** 9 work packages. Task 3 is a non-executable umbrella
over one AviaCore successor-contract slice and one AviaSurveil mirror/codegen
slice; Task 7 is a non-executable umbrella over nine separately authorized
AviaCore slices. The plan therefore contains 18 executable slices.

**Objective:** Give every persisted AviaSurveil360 datum exactly one approved
source-to-contract disposition, publish approved facts to AviaCore360 through
the approved producer protocol, prove privacy-safe reconciled local ingestion,
and produce a local synthetic analytics candidate plus an experiment-readiness
scaffold with point-in-time lineage.

**User-visible outcome:** Operational users continue using AviaSurveil360
without a data-platform dependency, while authorized data and ML teams can
inspect reproducible candidate data products whose source contract, platform
tenant, inspected organization, effective/known time, revision, privacy class,
and local quality evidence are known. No source-bound analytics or ML readiness
is implied.

**Dependencies:**

- The Identity And Realistic Data Foundation plan must be `completed` or
  explicitly accepted at its local milestone.
- The AviaCore360 repository must remain available separately through
  `AVIACORE_ROOT`, defaulting locally to
  `/Users/marlonjd/Developer/monorepos/aviaCore`.
- AviaCore's active
  `aviasurveil-production-data-feed-readiness-remediation` plan is the
  ingestion/data-platform authority. Its Phase 2.3 and each later phase require
  their own explicit execution authority.
- AviaCore's path-bound normative remediation plan and approved contract roots
  must not be edited, relocated, or reinterpreted from this repository.

## Current Contract Boundary

AviaCore contract version `3.0.0` currently freezes:

- producer: `aviasurveil360`;
- delivery mode: `event_api`;
- endpoint: `POST /v3/aviasurveil/event-batches`;
- content type:
  `application/vnd.aviacore.aviasurveil-events.v3+json`;
- direct mTLS with TLS 1.3 and source/tenant SAN mapping;
- at-least-once delivery with producer transactional outbox;
- 1-100 events per batch, 1 MiB maximum per event, and 10 MiB maximum per
  batch;
- producer-owned globally unique UUID event IDs;
- idempotent AviaCore admission by `event_id` and content digest;
- accepted, duplicate, mixed-terminal, conflict, validation, identity,
  authorization, size, rate, unavailable, and redacted-error outcomes;
- durable acknowledgement only after exact raw event and sealed landing
  manifest bindings.

The approved catalog contains 17 event types covering Audit, checklist,
Finding, CAP, Evidence-reference, information-request, correction,
supersession, reopen, escalation, and terminal transitions. That catalog is
not assumed to cover identity, planning approval, assignments, reports,
communications, notifications, risk, administration, or every ML feature.
Task 1 must classify every missing fact before either repository changes a
contract.

The 17-event catalog is not sufficient evidence of complete platform coverage.
It omits or only partially covers identity/membership, organizations, planning
and approval, assignments, versioned templates/questions/regulatory
references, Potential Findings, report/document versions and decisions,
communications/notifications/reminders, private advisory risk, administration,
and deletion/replay-suppression facts. Operational state such as sessions,
tokens, cursors, jobs, attempts, and object pointers still requires an explicit
operational-only, DQ-only, sensitive-restricted, or forbidden disposition.

The current package also has known pre-execution integrity blockers:

- correction and supersession schemas allow only
  `finding_open -> finding_open`, while the lifecycle contract allows those
  events across many states;
- link and satellite hash contracts name registry projections that the current
  key/grain entries do not provide;
- the privacy contract requires a tombstone event that the event catalog does
  not define; and
- the existing negative vectors do not exercise the behavioral branch matrix.

Task 2 must freeze the complete successor scope; Task 3A must then resolve the
contradictions through a versioned forward AviaCore contract and issue a new
behavioral digest/authorization; Task 3B must separately mirror and generate
producer validators. All accepted v1 bytes and existing v2 behavioral/
authorization records remain unchanged before AviaCore Phase 2.3 or producer
implementation. The digest-controlled historical normative plan is evidence
and must not be edited.

## Scope

- Cross-repository source-to-contract inventory and gap classification.
- Producer-side immutable event envelope, serializers, transactional outbox,
  mTLS publisher, retries, acknowledgements, conflict handling, replay,
  backfill, and evidence.
- Joint contract revisions when both repository owners approve them.
- End-to-end local synthetic ingestion into AviaCore raw, stage, Data Vault,
  Business Vault, candidate marts, semantic views, and quality state.
- ML data dictionary, labels, point-in-time feature views, leakage checks,
  dataset versions, lineage, and access controls.
- Exact source/outbox/AviaCore reconciliation and failure injection using the
  accepted synthetic workload profiles.

## Explicit Exclusions

- Direct writes from AviaSurveil into AviaCore databases, object storage, Raw
  Vault, marts, or feature tables.
- Kafka, RabbitMQ, Kubernetes, Feast, MLflow, OpenMetadata, Spark, Databricks,
  or any technology prohibited by AviaCore's locked direction.
- Exactly-once claims. Delivery is at-least-once and ingestion is idempotent.
- Raw Evidence bytes, filenames, unrestricted free text, Internal CAA Notes,
  investigation notes, provider tokens, session credentials, or secrets.
- Automatic enforcement, Finding closure, certificate action, legal decision,
  or unreviewed risk scoring from analytics or ML.
- Training any model. This plan reaches only
  `local synthetic analytics-candidate, verified locally` and an
  `experiment-readiness scaffold`; AviaCore L1/L2/L3 remain blocked.
- AWS deployment or production/source-bound canary data.

## Ownership Boundary

| Surface | Owner |
|---|---|
| Transactional writes, event semantics, UUID allocation, serialization, producer outbox, delivery attempts | AviaSurveil360 |
| Contract registry, authenticated admission, immutable raw landing, quarantine, Data Vault, DQ, marts, semantic metrics | AviaCore360 |
| Privacy purpose, retention, legal hold, deletion, production source approval | Joint security/privacy, legal/records, and product owners |
| ML feature and label meaning | Data/ML owner plus AviaSurveil domain owner |
| Connected environment, mTLS identity, SLO, recovery, and release | Platform and operations owners |
| Behavioral contract digest and authorization currentness | AviaCore contract governance plus producer/domain and privacy owners |
| Data-product contract, Data Vault key/grain, DQ, and publication | AviaCore data-product governance |

## Progress

- [x] (2026-07-27) AviaCore cloned separately to
  `/Users/marlonjd/Developer/monorepos/aviaCore`; worktree observed clean on
  `main`.
- [x] (2026-07-27) AviaCore AGENTS, architecture, plan index, active managed
  plan, normative plan, producer protocol, and event catalog inspected.
- [x] (2026-07-27) Independent cross-repository plan review completed. It
  identified incomplete 17-event coverage, v1 semantic/hash/tombstone
  contradictions, source/tenant/organization conflation, premature ML claims,
  late Data Vault/data-product design, producer privacy/recovery gaps, and the
  one-slice-per-task authority boundary. Implementation and runtime
  verification remain `not run`.
- [x] (2026-07-29) Identity/data predecessor acceptance verified from the
  completed plan record and final evidence.
- [x] (2026-07-29) Task 1 coverage register implemented, independently
  accepted, and `verified locally`: 89 final relations and 874 post-CREATE/
  ALTER columns have one explicit disposition; 67 OpenAPI operations, 24
  literal Audit actions, and 24 literal internal outbox topics have exact
  source-to-contract mappings. The v1 17-event catalog is byte-compared from
  AviaCore; Potential Finding/report and unbounded-text facts remain extension
  candidates or forbidden inline. Review accepted with no Critical/Important
  findings.
- [x] (2026-07-29) Burak Karahan / owner separately authorized Task 3A and
  approved the closed v3 extension event families, common envelope, bounded
  ordered question snapshot, zero-overlap compatibility, source-consistent
  historical event-API backfill, indefinite immutable retention, legal-hold
  restriction, replay/publication tombstone rule, and forward-fix recovery.
- [x] (2026-07-29) Task 3A: AviaCore v3 successor contract root, behavioral
  identity, and authorization envelope are independently accepted and
  `verified locally`. The historical v1 Phase 1 verifier remains a
  snapshot-bound predecessor record and is not a v3 acceptance gate.
- [x] (2026-07-29) Task 3B: AviaSurveil's exact 140-artifact local mirror,
  read-only lock gate, generated types, and full locked-schema validator are
  independently accepted and `verified locally`.
- [x] (2026-07-29) Burak Karahan / owner separately authorized Task 4 and the
  next forward-only migration `000022_aviasurveil_data_feed_outbox`.
- [x] (2026-07-29) Task 4: local v3 event construction, encrypted immutable
  producer storage, SQLC queries, fenced delivery state, and focused fresh
  PostgreSQL migration/rollback/idempotency proofs are `verified locally` and
  independently accepted. `CreateAuditWorkspace` is the one source-consistent
  transition: it writes the exact causal `audit.planned`/`audit.started` pair;
  read-only and final-state claims remain explicit non-events/dispositions.
- [x] (2026-07-30) Task 5: the direct-mTLS batch publisher is independently
  accepted and `verified locally`. Its RED/GREEN direct-mTLS configuration
  guard is `verified locally`. The earlier v3
  protocol/OpenAPI mismatch was repaired in AviaCore under owner authority and
  independently accepted: the exact protocol and OpenAPI now bind
  `/v3/aviasurveil/event-batches` to
  `application/vnd.aviacore.aviasurveil-events.v3+json`. A later
  owner-authorized v3 receipt-binding repair makes the expected item-set
  digest algorithm and every successful item digest explicit. Current
  behavioral identity `d87ec3649ff0f3b5f3871e90496eac2b1177dbec9f26fea72ced825d0beff121`
  and authorization identity
  `201cbed1f998b60506293efdae81c060b29d3d6e30696257785b4ec02be92c0e` are
  mirrored and locked locally. The repair is `candidate-only`, has no runtime
  ingestion or Phase 2.3 execution, and opens only this authorized Task 5.
  The local candidate now has a separate `data-feed-worker` target, closed
  mounted-secret worker configuration, TLS 1.3 direct mTLS validation,
  digest-bound receipt handling, exact v3 content/item-set digests, bounded
  jitter retry with eighth-attempt operator quarantine, fenced PostgreSQL
  acknowledgement/retry/quarantine state transitions, and scoped decrypted
  event reconstruction that rechecks payload and canonical hashes. Focused
  race/contract acceptance and disposable PostgreSQL migration/lease/receipt
  tests are `verified locally`. The final independently repeated acceptance
  uses a fresh dynamic loopback PostgreSQL port, has a fresh evidence root and
  scoped Docker cleanup, and passed with no Critical or Important finding.
  Telemetry records bounded named delivery outcomes plus pending-age and
  acknowledgement-lag measurements without payload values. This remains
  `candidate-only`; `production-ready: not established`.
- [x] (2026-07-30) Separate AviaCore Phase 2.3 execution authority completed
  and independently accepted `verified locally` as a local-fixture v3
  event-API admission boundary. It supplies no admitted/raw manifest or
  connected runtime evidence.
- [x] (2026-07-30) Separately authorized AviaCore Phase 2.4 is independently
  accepted `verified locally`: forward-only candidate PostgreSQL canonical
  reservation, immutable attempts/receipts, separate submission/landing
  manifest metadata, quarantine-v2, replay-suppression tombstones, legacy
  plaintext-quarantine writer shutdown, and receipt-bound fenced acknowledgement
  passed local and networkless PostgreSQL checks. This does not wire a connected
  admission runtime or emit a source-bound admitted/raw manifest.
- [ ] (2026-07-30) Task 6 is in progress. The candidate has an approval-bound
  replay/backfill request model, deterministic scope digest, forward-only
  migration `000024`, immutable replay-run/source-event snapshots, and a
  separate fenced replay delivery lane. Focused RED/GREEN tests prove that a
  replay receipt updates only that lane and its append-only attempt history;
  the original producer delivery state remains unchanged. The dedicated
  replay/backfill/reconciliation commands, exact synthetic manifest contract,
  and local recovery aggregate are `verified locally`. The actual AviaCore
  admission/raw manifest, coordinated two-system recovery, RPO/RTO exercise,
  and independent Task 6 acceptance remain `not run` and `blocked` on a
  separately authorized connected AviaCore runtime/raw-manifest slice.

## Tasks

### Task 1: Produce The Complete Source-To-AviaCore Coverage Register

**Files**

- Create `docs/product-specs/data-and-rules/AVIACORE_DATA_FEED_COVERAGE.md`.
- Create `docs/product-specs/data-and-rules/aviacore-data-feed-coverage.json`.
- Create `tests/aviacore-data-feed-coverage.test.mjs`.
- Read from, but do not modify without separate authority:
  `$AVIACORE_ROOT/contracts/aviasurveil-production/v1/`.

**Work**

- [x] Derive a closed inventory from every migration relation/column,
  structured JSON pointer, versioned object/metadata field, OpenAPI mutation,
  domain command/transition, audit action, outbox topic, projection, and
  materialized or external persistent record. Bind it to migration, OpenAPI,
  object-schema, and command-registry fingerprints so an added, removed, or
  renamed source fails the gate.
- [x] Give every persisted datum exactly one disposition:
  approved event field, approved snapshot contract, approved reference-data
  contract, approved data-product derivation, contract-extension candidate,
  operational/DQ-only, sensitive-restricted, or forbidden. A datum is never
  omitted merely because it is not currently proposed for analytics or ML.
- [x] Permit `approved data-product derivation` only when every derivation
  input is already transported through an approved event, snapshot, or
  reference-data contract and the product proves deterministic field-level
  lineage. A downstream product is not a transport mechanism; otherwise the
  source datum needs a direct transport contract or an explicit non-feed/
  DQ-only disposition.
- [x] For every datum define source authority, contract family and grain,
  entity references, authenticated platform `tenant_id`, `source_system`,
  inspected/owning `organization_id`, actor organization, visibility/purpose
  scope, effective time, known time, producer revision/sequence, null
  semantics, correction/supersession behavior, PII class, minimization
  rationale, retention/legal-hold/deletion rule, expected volume, consumer,
  lineage, and feature/label eligibility. Never derive tenant identity from a
  payload organization.
- [x] Compare the complete inventory against all 17 approved event types and
  list exact gaps. At minimum inspect identity lifecycle, organization,
  planning approvals, team/question assignments, report versions/decisions,
  documents, communications metadata, notifications, calendar/reminders, risk
  inputs/outputs, and administration/audit state.
- [x] Keep unrestricted free text, Evidence bytes, filenames, Internal CAA
  Notes, investigation notes, person names, contact values, and credentials
  forbidden inline unless a later explicit policy changes their class.
- [x] Require every proposed feature and label to trace to at least one
  approved source fact. Reject "collect everything" fields with no named
  purpose or owner.
- [x] Produce three independent completeness proofs: static persisted-source
  inventory, command/transition-to-contract coverage, and profile-manifest to
  producer-event/AviaCore-acknowledgement reconciliation. Synthetic rows alone
  do not prove an unexecuted mutation branch.
- [x] Include two platform tenants with colliding local business IDs, and one
  CAA tenant with multiple inspected Auditee organizations, in key, RLS,
  object-policy, join, restore, and reconciliation mutation tests.

**Verification**

Run:

    node --test tests/aviacore-data-feed-coverage.test.mjs
    node tests/harness-docs-smoke.test.js
    git diff --check

Expected: 100 percent of persisted source data and authoritative mutations
have exactly one fingerprint-bound disposition; unknown, stale, omitted, and
duplicate dispositions fail.

**Acceptance**

- No working data path is silently omitted.
- No sensitive field is admitted merely because it exists.
- The current 17-event scope and every contract-extension candidate are
  explicit.

### Task 2: Approve The Complete Successor Contract Scope

This task freezes decisions and paths only. It does not edit an accepted
contract root, issue a behavioral digest, copy producer contracts, generate
code, or authorize Phase 2.3.

**Files**

- Modify the Task 1 coverage register with owner decisions and exact
  dispositions.
- Create
  `docs/product-specs/data-and-rules/AVIACORE_SUCCESSOR_CONTRACT_DECISIONS.md`.
- Create
  `docs/product-specs/data-and-rules/aviacore-successor-contract-decisions.json`.
- Create `tests/aviacore-successor-contract-decisions.test.mjs`.
- Read only the accepted predecessor roots:
  `$AVIACORE_ROOT/contracts/aviasurveil-production/v1/`,
  `$AVIACORE_ROOT/schemas/events/aviasurveil/v1/`,
  `$AVIACORE_ROOT/contracts/aviasurveil-production/v2/behavioral-contract-manifest.json`,
  and
  `$AVIACORE_ROOT/contracts/aviasurveil-production/v2/authorization-policy.json`.

**Work**

- [x] Review every Task 1 extension candidate with producer/domain,
  contract-governance, data-platform, data-product, privacy/retention, and
  Data/ML owners.
- [x] Add only facts with a named analytical/ML purpose, authority, grain,
  privacy class, retention class, correction rule, and owner.
- [x] Select and record one not-yet-existing successor contract/schema/
  behavioral-manifest root. Use a compatible minor contract revision for
  additive scope or a new major contract with overlap policy for breaking
  changes. Existing v1 bytes and the current v2 behavioral/authorization
  records are immutable predecessor artifacts, not candidate change paths.
- [x] Resolve the known correction/supersession lifecycle scope, missing hash
  projections, tombstone/replay-suppression semantics, and required exhaustive
  behavioral branches as part of the final successor scope.
- [x] Preserve all source IDs, organization identity, versions, effective
  times, known/availability times, and human-decision provenance needed for
  point-in-time analysis.
- [x] Approve explicit event, source-consistent bootstrap/snapshot,
  reference-data, and data-product contract families. Keep the protocol's
  `approved_snapshot` delivery mode disabled unless a separately versioned
  snapshot protocol is approved.
- [x] Choose one bootstrap path before outbox implementation: historical
  event-API backfill from a source-consistent cut, or a separately approved
  snapshot/reference protocol. Bind expected IDs/counts/digests, original
  effective and known time, producer revision, tombstones, cursor/resume, and
  reconciliation; never present a table dump as an original real-time event.
- [x] Separate authenticated platform `tenant_id`, `source_system`,
  inspected/owning `organization_id`, actor organization, and visibility/
  purpose scope in envelopes, keys, policies, and reconciliation.
- [x] Freeze Data Vault 2 business keys with source namespace, hubs, links,
  satellites, effectivity/record-tracking satellites, PIT/bridge needs,
  hash/hashdiff inputs, insert-only historization, late/corrected/superseded
  behavior, record source, and DQ gates before AviaCore Phase 2.5.
- [x] Freeze every data product's owner, purpose/legal basis, grain, input
  transport contracts, deterministic lineage, freshness, retention/legal
  hold/deletion, access, privacy, DQ, correction, and publication policy before
  implementation.
- [x] Keep payloads closed with `additionalProperties: false`; bounded
  classified maps require explicit field-path policy.
- [x] Record the exact compatibility/overlap window, version negotiation,
  rollback/forward-fix rule, authorization owners, and separately executable
  AviaCore-cut and AviaSurveil-mirror slices.

**Verification**

Run:

    node --test tests/aviacore-successor-contract-decisions.test.mjs
    node --test tests/aviacore-data-feed-coverage.test.mjs
    node tests/harness-docs-smoke.test.js
    git diff --check

Expected: every coverage candidate and predecessor contradiction has one
owner-approved successor disposition and exact future root; no predecessor
artifact changes and no digest/codegen output exist.

**Acceptance**

- Every persisted datum has one approved transport/product/non-feed
  disposition, and every product input already has an approved transport.
- Bootstrap, reference, Data Vault, product, privacy, compatibility, and
  branch-vector scope is final before a successor digest is calculated.
- Missing owner decisions block Task 3 without changing either contract root.

### Task 3: Cut The Successor Contract, Then Lock The Producer Bundle

This heading is a tracking umbrella, not joint authority. Slice 3A is the
separately authorized AviaCore successor-contract cut. After 3A passes and
stops, Slice 3B is the separately authorized AviaSurveil read-only mirror,
lock, and code-generation update. Neither slice authorizes Phase 2.3.

**Files**

- In Slice 3A, create only the new Task 2-selected AviaCore successor roots and
  branch vectors; update its managed active plan/index. Never modify accepted
  v1 bytes, the existing v2 behavioral/authorization records, legacy approval
  evidence, or the digest-controlled historical normative plan.
- In Slice 3B, create `integrations/aviacore/contracts/`,
  `integrations/aviacore/contract-lock.json`,
  `scripts/check-aviacore-contracts.sh`, and
  `tests/aviacore-contract-lock.test.mjs`; modify `MANIFEST.md` and
  `docs/index.md`.

**Work**

- [x] Slice 3A: add an exhaustive package-integrity matrix and implement the
  Task 2-approved successor corrections for lifecycle/schema consistency,
  complete link/satellite hash projections, tombstone/replay-suppression
  events, bootstrap/reference/product contracts, tenant/organization
  separation, and every approved coverage extension.
- [x] Slice 3A: require positive and negative vectors for every allowed/
  forbidden transition, required/null field, enum, identity/tenant mismatch,
  timestamp rule, privacy branch, correction/supersession reference,
  bootstrap/resume branch, and version-compatibility/overlap branch.
- [x] Slice 3A: issue a new successor behavioral-contract digest and a separate
  current owner-authorization envelope. Keep authorization state outside the
  behavioral digest and preserve all predecessor approvals unchanged.
- [x] Stop after AviaCore repository checks and independent contract-owner
  acceptance. Record the exact AviaCore commit/content state, successor root,
  behavioral digest, authorization digest/currentness, and evidence root.
- [x] Slice 3B: copy only the final approved producer-facing protocol, event
  catalog, envelope/payload schemas, lifecycle, privacy, temporal, hash,
  state-machine, bootstrap/snapshot, reference-data, recovery, compatibility/
  overlap, OpenAPI surface, and every associated conformance vector needed to
  build, backfill, recover, and test the producer independently. Do not vendor
  AviaCore implementation or internal Vault/product artifacts.
- [x] Record the exact AviaCore commit/content state, behavioral-contract
  digest, contract-set version, exact successor source inventory, per-file
  SHA-256, aggregate root digest, source path, and authorization-envelope
  identity/currentness separately.
- [x] Make the check command read-only by default and require an explicit,
  separately authorized update mode to replace the producer copy.
- [x] Fail on local/source drift, missing or extra schema, stale/expired/
  revoked/superseded authorization, predecessor-root mutation, or lock-version
  mismatch.
- [x] Generate Go test types/validators from the locked JSON Schemas without
  importing AviaCore implementation code and run the complete branch matrix
  against the producer serializer.

**Verification**

Run Slice 3A from AviaCore, then stop:

    make -C /Users/marlonjd/Developer/monorepos/aviaCore aviasurveil-v3-contract-check
    make -C /Users/marlonjd/Developer/monorepos/aviaCore aviasurveil-v3-protocol-check
    make -C /Users/marlonjd/Developer/monorepos/aviaCore repo-check
    git -C /Users/marlonjd/Developer/monorepos/aviaCore diff --check

After separate Slice 3B authorization, run:

    AVIACORE_ROOT=/Users/marlonjd/Developer/monorepos/aviaCore ./scripts/check-aviacore-contracts.sh
    node --test tests/aviacore-contract-lock.test.mjs
    ./scripts/check-contracts.sh
    git diff --check

Task 3A evidence, `verified locally`: `make ... aviasurveil-v3-contract-check`
passed 8 tests; `make ... aviasurveil-v3-protocol-check` passed with behavioral
digest `48a6beac9891df3f1becb262e686cd3f010c7072ca0e90c938605672270db530`,
authorization digest `f1388795d2bde4fc5b32ae5897abb3b7a4f178a7a083445fb937f46e8b482c31`,
and status `candidate_only`; `make ... repo-check` passed 46 tests plus scans
and repository validation; `git diff --check` passed. The independent read-only
review found no Critical or Important finding. The v1 historical Phase 1 target
remains a snapshot-bound historical failure and is not a v3 acceptance gate.
Task 3B evidence, `verified locally`: the read-only local mirror has 140 exact
source artifacts at AviaCore `8df4b0cb871d3bb4604a8cc52e3b826db029e008`,
contract version `3.0.0`, aggregate root
`372ad14c94009b0ab46b47989eb91fd0f09382a069ea5dc30ee07b10c7e7e078`,
and the Task 3A behavioral/authorization digests above. The lock carries only
Task 3B mirror currentness, not producer-runtime or Phase 2.3 authority. Its
default checker, 3/3 lock tests, locked-schema Go validator (39 positive, 39
negative, and 11 branch cases), existing contracts 16/16, docs smoke, and
`git diff --check` passed; independent review found no Critical or Important
finding. Task 4 and AviaCore Phase 2.3 remain `not run` and separately
unauthorized.

**Acceptance**

- AviaCore remains the contract authority; AviaSurveil builds without a sibling
  checkout but can prove exact provenance against one.
- The two slices have separate authority and evidence; a joint change is
  invalid.
- No producer implementation or AviaCore Phase 2.3 starts against the current
  contradictory v1 package or before Slice 3B passes.

### Task 4: Implement The Immutable Producer Feed Outbox

**Files**

- Create `apps/api/internal/datafeed/`.
- Create `apps/api/internal/datafeed/store/postgres/queries.sql`.
- Add `apps/api/internal/datafeed/store/postgres/` generated SQLC artifacts.
- Create the next available forward-only migration determined from the
  execution-time migration ledger. Do not reserve or assume a number in this
  plan.
- Modify `apps/api/sqlc.yaml`.
- Modify authoritative application transaction boundaries that emit approved
  facts.
- Add unit and PostgreSQL integration tests.

**Interfaces**

- Produce one immutable canonical event row keyed by global UUID `event_id`.
- Produce immutable delivery-attempt rows keyed by attempt identity.
- Persist contract/event version, `source_system`, authenticated platform
  `tenant_id`, inspected/owning and actor organization identities,
  visibility/purpose scope, payload, canonical content digest, effective and
  known/availability time, producer sequence/revision, operation/correlation/
  causation IDs, lease generation, acknowledgement receipt, and terminal
  outcome.

**Work**

- [x] Allocate producer-owned UUID event IDs and fail closed against the
  locally locked v3 schema before durable persistence.
- [x] Write the `CreateAuditWorkspace` business mutation, Audit event,
  authorized sync change, internal outbox work, and the exact locked v3
  `audit.planned`/`audit.started` causally linked event pair in the same
  PostgreSQL transaction. The pair is reconstructed only when a released plan
  and newly created workspace establish both source facts; unsupported or
  ambiguous inspection types, or an absent local writer, fail closed and roll
  back the entire transition.
- [x] Require every currently registered approved command/transition to emit
  its exact fact or carry a tested explicit non-event disposition. Read-only
  operations remain non-events; workspace materialization is the single
  source-consistent v3 pair above, not a table-scraped final-state claim.
- [x] Use locked-schema validation, deterministic canonical JSON, and exact
  payload/content SHA-256 bindings.
- [x] Keep persisted feed payload ciphertext immutable; the lock rejects an
  unsupported correction/supersession shape before storage.
- [x] Separate pending delivery, append-only attempt outcome, acknowledged
  state, and replay tombstone. A local commit cannot become acknowledged.
- [x] Add scoped indexes, bounded claim query, fenced lease generation,
  stale-lease denial, dead-letter state, and append-only attempt history.
- [x] Enforce a closed event-type catalog, PII minimization, authenticated
  platform tenant binding, organization-scoped reads, AES-GCM payload storage,
  indefinite immutable retention, legal-hold tombstone denial, and replay
  suppression. Dedicated worker-role provisioning and network publisher are
  Task 5 work.
- [x] Keep persisted attempt/dead-letter diagnostics value-free. No pointer or
  arbitrary diagnostic payload is accepted.
- [x] Prove a rollback leaves neither a test domain mutation nor feed event,
  repeat operation/event insertion is denied, migration 21 upgrades to 22
  without historical rewrite, and no backfill is performed by migration.

**Verification**

Run:

    ./scripts/check-sqlc.sh
    go -C apps/api test -race -p 1 -count=1 ./internal/datafeed ./internal/application ./tests/integration
    git diff --check

Expected: the authorized workspace transition creates its exact immutable,
causally linked v3 lifecycle pair in the same transaction, and no event can be
updated into different content.

**Acceptance**

- The producer can survive AviaCore unavailability without losing facts or
  blocking user transactions.
- Feed completeness begins at the domain transaction, not from later table
  scraping.
- A second durable payload copy has a named purpose, access policy, retention,
  deletion, and non-resurrection proof.

**Execution record (2026-07-29)**

Task 4 is `verified locally` and independently accepted. The only mapped
authoritative transition is `CreateAuditWorkspace`: after locking and reading
its released plan, it validates an exact supported inspection type and, in the
same transaction, writes the workspace, Audit event, authorized sync change,
internal outbox row, and causally linked locked-v3 `audit.planned` plus
`audit.started` rows. The event pair is unavailable unless the local writer is
explicitly configured; an absent writer or unsupported/ambiguous source type
returns a closed error and leaves no business or feed row. The local candidate
has no publisher, mTLS client, network call, AviaCore Phase 2.3 execution, or
Task 5 implementation.

Fresh checks: `./scripts/check-sqlc.sh`; `go -C apps/api test -race -p 1
-count=1 ./internal/datafeed ./internal/application`; the focused migration/
workspace integration gate; full `go -C apps/api test -race -p 1 -count=1
./tests/integration` with disposable PostgreSQL and MinIO (`ok`, 60.969s);
locked-contract and coverage tests (9/9); harness docs smoke; and `git diff
--check`. Independent read-only re-review found no Critical or Important
finding. This evidence is `candidate-only`; release remains `release pending`
and `production-ready: not established`.

### Task 5: Implement The Direct-mTLS Batch Publisher

**Files**

- Create `apps/api/cmd/data-feed-worker/main.go`.
- Create `apps/api/internal/datafeed/publisher.go`.
- Create `apps/api/internal/datafeed/mtls_client.go`.
- Modify `apps/api/Dockerfile` with a separately named worker target.
- Add local configuration schema and secret-file wiring.
- Add worker, protocol, retry, and telemetry tests.
- Create `scripts/test-aviacore-data-feed-publisher.sh` and a command/evidence
  contract test. The script owns a protocol-faithful TLS receiver, run-scoped
  evidence root, exact certificate/profile inputs, and scoped cleanup.

**Work**

- [x] Claim bounded pending events with fenced leases and build batches of
  1-100 items within exact event and batch byte limits.
- [x] Use TLS 1.3 direct mTLS, a secret-mounted client key/certificate, an
  approved CA bundle fingerprint, and source/tenant SAN mapping.
- [x] Reject plaintext, forwarded-client-certificate headers, unknown CA/SAN,
  expired or revoked material, wrong source/tenant, and unsafe endpoint
  configuration.
- [x] Implement full-jitter bounded exponential retry using the approved
  1-second base, 60-second cap, and operator alert after eight attempts.
- [x] Persist 201 accepted, 200 duplicate, 207 mixed-terminal, 409 conflict,
  422 validation, 401/403 identity, 413 size, 429 rate, 503 unavailable, and
  redacted 500 outcomes without logging payload values.
- [x] Mark acknowledged only from an AviaCore receipt that binds request,
  batch, attempt, event, digest, and exact acknowledged winner.
- [x] Treat HTTP 207 per item. Reject a missing, duplicate, unmatched,
  wrong-event, wrong-digest, or batch-level-only receipt before advancing any
  affected event; independently persist valid terminal items.
- [x] Quarantine contract/conflict failures for operator disposition; retry
  only retryable outcomes. Freeze retryable/permanent/manual-review taxonomy,
  maximum time/attempt policy, quarantine owner/SLA, append-only resolution,
  correction/supersession/replay options, and terminal non-retry behavior.
- [x] Emit low-cardinality metrics and redacted traces for pending age,
  throughput, outcome, retry, dead letter, and acknowledgement lag.

**Verification**

Run:

    ./scripts/test-aviacore-data-feed-publisher.sh acceptance

The aggregate must enumerate the Go race packages, positive/negative TLS and
receipt branches, retry/quarantine cases, value/secret scan, exact evidence
paths, and cleanup command. It must fail if the receiver or evidence root is
reused. Expected: the protocol-faithful TLS receiver proves that
duplicates converge on one acknowledged winner, conflict never mutates
content, transient failures retry, permanent failures remain visible, and no
secret or payload value appears in logs.

**Acceptance**

- The implementation conforms exactly to the selected event API mode.
- Unsupported export/snapshot modes remain disabled.
- Quarantine and dead-letter views expose no raw payload or forbidden value and
  every item has an owned terminal disposition.

### Task 6: Add Replay, Backfill, Reconciliation, And Recovery

**Status:** `in progress` — the immutable, approval-bound replay/backfill
lanes, reconciliation command, local evidence contract, and closed synthetic
recovery aggregate are `verified locally`. The actual AviaCore
admission/raw-manifest comparison, coordinated two-system recovery, RPO/RTO
measurement, and independent Task 6 acceptance remain `not run` and `blocked`
on a separately authorized connected AviaCore runtime/raw-manifest slice.

**Files**

- Create `apps/api/cmd/data-feed-replay/main.go`.
- Create `apps/api/internal/datafeed/replay.go`.
- Create `scripts/reconcile-aviacore-feed.sh`.
- Create `scripts/test-aviacore-feed-recovery.sh` and its command/evidence/
  cleanup contract.
- Add operator runbooks under `docs/operations/runbooks/`.
- Add recovery and failure integration tests.

**Work**

- [x] Implement approval-bound replay by event IDs, time window, source,
  tenant, contract version, and terminal outcome without changing event
  content or identity.
- [x] Implement only the Task 3-approved bootstrap/backfill contract from a
  source-consistent cut. Preserve original effective/known times and producer
  revisions, use a new backfill-run identity rather than fabricated occurrence,
  and do not masquerade table dumps as original real-time events.
- [ ] Compare expected producer manifest, canonical feed rows, attempts,
  acknowledgements, AviaCore admitted events, quarantines, raw objects, and
  governed relation counts/digests. The local command already rejects exact
  missing, extra, digest-mutated, or receipt-frontier-mismatched manifest rows;
  it cannot complete the AviaCore-owned fields without the Phase 2.4
  admission/raw manifest.
- [ ] Detect lost receipt, duplicate response, partial mixed batch, worker
  crash, certificate rotation, AviaCore outage, rate limit, schema rejection,
  event conflict, clock skew, and replay interruption.
- [ ] Preserve user workflows while lagging and alarm on approved lag/SLO
  thresholds.
- [ ] Include feed state and acknowledgement frontier in coordinated backup and
  restore. Restore must not reassign event IDs or advance acknowledgements.
- [ ] Run a coordinated two-system recovery without cross-database writes:
  fence producer publishing and AviaCore admission/publication, bind the
  producer outbox frontier to admitted/raw/sealed-manifest and last-trusted-
  publication frontiers, restore separate exact packages, replay only
  unacknowledged events, and prove no deleted datum is resurrected.
- [ ] Measure separate producer, core-feed, and trusted-publication RPO/RTO
  using defined failure/start/ready clocks, and reconcile corrections,
  supersessions, quarantines, tombstones, and replay suppression after restore.

**Verification**

Run:

    ./scripts/test-aviacore-feed-recovery.sh acceptance

The aggregate binds the source-consistent-cut manifest, event/frontier roots,
both system packages, failure matrix, per-store clocks, create-only evidence
root, and scoped cleanup command. Expected: exact eventual reconciliation
after retry/replay, zero unexplained loss, zero content mutation, fail-closed
permanent errors, and no resurrection after coordinated restore.

**Acceptance**

- Operators can explain and recover every event from source transaction through
  acknowledgement.
- Backfill and replay have separate audited semantics.
- Recovery never infers producer acknowledgement from restored downstream
  state and never assigns a new ID to an existing fact.

### Task 7: Complete AviaCore Candidate Ingestion And Local Integration

This heading is a tracking umbrella, not authority to execute multiple
AviaCore phases in one task. Its nine slices are Phase 2.3, 2.4, 2.5, 2.6,
2.7, 3, 4, 5A, and 5B. Each slice requires a separate current-task
authorization, worktree preflight, evidence root, managed-plan/index update,
acceptance, and stop before the next slice.

**Files**

- AviaCore-owned files are governed by its active and normative plans.
- Create `scripts/test-aviacore-phase-slice.sh` as a fail-closed mapper from one
  authorized phase plus one run manifest to the exact AviaCore planned targets.
- Create `scripts/test-aviacore-local-integration.sh` in AviaSurveil.
- Create `scripts/cleanup-aviacore-local-integration.sh`; it may remove only
  identifiers in the exact run manifest and never deletes create-only evidence.
- Create cross-repository fixture/evidence manifests without copying AviaCore
  implementation.
- Modify local Compose only for a dedicated cross-repository integration
  network/profile.

**Work**

- [ ] Obtain separate authority for exactly one AviaCore slice at a time and
  complete Phase 2.3, 2.4, 2.5, 2.6, 2.7, 3, and 4 in their required order.
  Never infer the next authorization from acceptance of the prior slice.
- [ ] Keep AviaCore evidence at its literal local fixture/candidate class until
  producer conformance and connected integration gates run.
- [ ] Start the normal AviaSurveil local-preprod stack and the faithful
  AviaCore candidate ingestion/data-product stack as separate bounded systems.
- [ ] Run the exact producer artifact through AviaCore Phase 5A positive and
  negative conformance vectors.
- [ ] Run Phase 5B end-to-end sandbox integration with direct mTLS and the
  `acceptance` profile.
- [ ] Prove authenticated source/tenant binding, immutable raw landing,
  idempotent admission, quarantine, stage, Raw Vault, Business Vault, DQ,
  publication, marts, and semantic views.
- [ ] For Phase 2.5, prove source-namespaced business keys, complete hubs/
  links/satellites, insert-only historization, effectivity/record tracking,
  PIT/bridge point-in-time behavior, effective-as-of and known-as-of results,
  late/corrected/superseded history, hash parity, deterministic rebuild, and
  no Raw Vault business rules.
- [ ] For Phase 3, prove platform tenant, inspected organization, actor
  organization, purpose/visibility scope, RLS/non-owner roles, pool reset,
  object/quarantine/ops boundaries, pseudonymization and controlled
  re-identification across every layer.
- [ ] For Phase 4, prove capacity, fencing, recovery frontiers, independent
  restore, DQ/publication resumption, retention/deletion non-resurrection, and
  exact local candidate RPO/RTO.
- [ ] Bind both Git commits, worktree states, contract roots, image digests,
  profile manifest, certificates, commands, and cleanup to the run evidence.

**Verification**

For each separately authorized slice, create a new immutable run manifest and
run exactly one command before stopping:

| Slice | Exact planned command mapped by the wrapper |
|---|---|
| 2.3 | `make -C "$AVIACORE_ROOT" aviasurveil-pre-execution-check aviasurveil-contract-check aviasurveil-protocol-check` |
| 2.4 | `make -C "$AVIACORE_ROOT" aviasurveil-idempotency-check` |
| 2.5 | `make -C "$AVIACORE_ROOT" aviasurveil-persistent-vault-products-check aviasurveil-hash-parity-check` |
| 2.6 | `make -C "$AVIACORE_ROOT" aviasurveil-dagster-check aviasurveil-soda-check aviasurveil-publication-check` |
| 2.7 | `make -C "$AVIACORE_ROOT" aviasurveil-evidence-integrity-check aviasurveil-candidate-check` |
| 3 | `make -C "$AVIACORE_ROOT" aviasurveil-security-check` |
| 4 | `make -C "$AVIACORE_ROOT" aviasurveil-resilience-local-check aviasurveil-recovery-local-check` |
| 5A | `make -C "$AVIACORE_ROOT" aviasurveil-producer-conformance-cert` |
| 5B | `make -C "$AVIACORE_ROOT" aviasurveil-sandbox-integration-cert` |

Invoke the mapper as:

    AVIACORE_ROOT=/Users/marlonjd/Developer/monorepos/aviaCore ./scripts/test-aviacore-phase-slice.sh <phase> <run-manifest>

For 5B the mapper also invokes:

    AVIACORE_ROOT=/Users/marlonjd/Developer/monorepos/aviaCore ./scripts/test-aviacore-local-integration.sh acceptance

Every run manifest binds input/evidence/cleanup roots and the expected
readiness/non-claim result. Finish the authorized slice with:

    make -C /Users/marlonjd/Developer/monorepos/aviaCore repo-check
    ./scripts/cleanup-aviacore-local-integration.sh <run-id>

The targets above are planned by AviaCore's normative plan and cannot be
reported as run until implemented. Expected: each phase produces its own
create-only evidence and stops; after 5B, AviaSurveil reaches producer
conformance and the exact local pair reaches only the readiness level allowed
by AviaCore's verified Phase 5 gate.

**Acceptance**

- The same exact events reconcile from source transaction through governed
  AviaCore products.
- Neither repository writes directly to the other's database.
- Completion means all nine separately evidenced slices passed; a single
  umbrella run or combined authorization is invalid.

### Task 8: Qualify The Local Synthetic Analytics Candidate And ML Scaffold

**Files**

- Create `docs/product-specs/data-and-rules/ML_DATA_READINESS.md`.
- Create `docs/product-specs/data-and-rules/ml-feature-register.json`.
- Create `tests/ml-data-readiness-contract.test.mjs`.
- Create `scripts/test-ml-data-readiness.sh` and a command/evidence/cleanup
  contract that binds the dataset/profile/as-of/query/environment inputs.
- AviaCore-owned marts, feature views, dbt tests, Dagster assets, Soda checks,
  and evidence remain in the AviaCore repository.

**Work**

- [ ] Define one authoritative grain, owner, purpose, legal basis, privacy
  class, freshness, retention, quality policy, and lineage root for every data
  product.
- [ ] Define candidate features and labels using effective-as-of and
  known-as-of time. Record `prediction_time`, `occurred_at`, `received_at`,
  feature `available_at`, label observation window, maturity delay, censoring,
  exclusions, and temporal/group split policy; forbid future-information
  leakage.
- [ ] Separate human decisions, operational outcomes, advisory scores, and
  model-generated values. Never treat a model output as a ground-truth label.
- [ ] Version dataset definitions, feature calculations, label rules,
  inclusion/exclusion criteria, correction policy, source contract roots,
  query/code/environment digests, DQ run, and dataset snapshot identity.
- [ ] Add completeness, uniqueness, validity, referential integrity,
  timeliness, distribution, class imbalance, drift, leakage, and tenant/privacy
  checks.
- [ ] Pseudonymize actor identity where direct identity is not required and
  preserve controlled re-identification only under audited authority.
- [ ] Produce reproducible point-in-time dataset snapshots from the same
  contract/profile/run-as-of inputs with identical digests.
- [ ] Record feature-level provenance from source contract/event or approved
  data-product fields through Vault/product relation, transformation/query
  digest, DQ decision, run-as-of, dataset snapshot, and consumer.
- [ ] Derive machine-readable training eligibility and reason codes. Under this
  plan, require `training_allowed: false`,
  `production_ml_readiness: NOT_READY`, and no AviaCore L1/L2/L3 label because
  the evidence is synthetic and not source-bound.
- [ ] Keep AI/ML access read-only or suggestion-only and define fallback,
  owner review, and kill-switch prerequisites for any later model plan.

**Verification**

Run:

    node --test tests/ml-data-readiness-contract.test.mjs
    AVIACORE_ROOT=/Users/marlonjd/Developer/monorepos/aviaCore ./scripts/test-ml-data-readiness.sh acceptance
    make -C /Users/marlonjd/Developer/monorepos/aviaCore aviasurveil-candidate-check
    make -C /Users/marlonjd/Developer/monorepos/aviaCore aviasurveil-dagster-check
    make -C /Users/marlonjd/Developer/monorepos/aviaCore aviasurveil-soda-check
    make -C /Users/marlonjd/Developer/monorepos/aviaCore repo-check

The aggregate records one create-only evidence root and performs only
run-manifest-scoped cleanup. The AviaCore targets remain planned until
implemented. Expected: every candidate feature and label has complete local
point-in-time lineage, privacy policy, quality results, a reproducible dataset
digest, and a fail-closed non-training decision.

**Acceptance**

- Governed synthetic outputs may be called only
  `local synthetic analytics-candidate, verified locally`.
- The dataset package is an `experiment-readiness scaffold`, not AviaCore L2
  and not training eligible.
- AviaCore L1 requires separately authorized Phase 6 real source-bound
  ingestion. L2 additionally requires an approved label/target window,
  point-in-time features, leakage evidence, a source-bound dataset, and
  approved offline evaluation. No `production-ML-ready` claim is made.

### Task 9: Run The Final Feed And ML Data Readiness Gate

**Files**

- Create
  `docs/demo-evidence/AVIACORE_AND_ML_DATA_READINESS_2026-07-27.md` during
  execution.
- Modify this plan, the plan index, tracker, build evidence, and coverage
  register with literal results.
- AviaCore evidence remains under its own create-only evidence roots.

**Work**

- [ ] Run the complete producer contract, migration, Go race, web, local
  preprod, AviaCore conformance, integration, DQ, replay, reconciliation,
  recovery, privacy, lineage, and residue matrix.
- [ ] Before `realistic` or `stress`, calculate event expansion, batch count,
  Raw/Vault/product row growth, object/storage, WAL, memory, duration, and
  cleanup headroom against the frozen local envelope. Partition and resume the
  run; resource insufficiency is a literal blocker, not permission to silently
  downgrade a profile.
- [ ] Run `smoke`, `acceptance`, `realistic`, and `stress` through their exact
  applicable contract/reconciliation gates.
- [ ] Require 100 percent persisted-source-to-contract disposition, 100 percent
  expected event reconciliation, zero unexplained loss/conflict, zero forbidden
  data leakage, and zero unowned feature or label.
- [ ] Exercise AviaCore outage, mTLS rotation, retries, partial batch,
  duplicate, conflict, schema drift, poison event, replay, backfill, and
  restore.
- [ ] Obtain independent producer/domain, AviaCore contract governance,
  data-platform/data-product, security/privacy, legal/retention,
  operations/recovery, and Data/ML reviews. Fix all Critical and Important
  findings.
- [ ] Record exact readiness levels from both repositories without promoting
  local fixture evidence to source-bound, pilot, target, production, or
  production-ML claims.

**Verification**

The final gate includes:

    ./scripts/check-aviacore-contracts.sh
    ./scripts/check-contracts.sh
    ./scripts/check-sqlc.sh
    go -C apps/api test -race -p 1 -count=1 ./...
    npm --prefix apps/web run typecheck
    npm --prefix apps/web test
    AVIACORE_ROOT=/Users/marlonjd/Developer/monorepos/aviaCore ./scripts/test-aviacore-local-integration.sh smoke
    AVIACORE_ROOT=/Users/marlonjd/Developer/monorepos/aviaCore ./scripts/test-aviacore-local-integration.sh acceptance
    AVIACORE_ROOT=/Users/marlonjd/Developer/monorepos/aviaCore ./scripts/test-aviacore-local-integration.sh realistic
    AVIACORE_ROOT=/Users/marlonjd/Developer/monorepos/aviaCore ./scripts/test-aviacore-local-integration.sh stress
    make -C /Users/marlonjd/Developer/monorepos/aviaCore repo-check
    git diff --check

Expected: all applicable gates pass and both repositories state identical
contract, evidence, and non-claim boundaries.

**Acceptance**

- Producer contract conformance and local sandbox integration are proven.
- AviaCore outputs remain a local synthetic analytics candidate and the ML
  package remains a non-training scaffold.
- The result remains `candidate-only`, `release pending`,
  `training_allowed: false`, `production_ml_readiness: NOT_READY`, and not
  AviaCore L1/L2/L3.
- Success satisfies this predecessor dependency; starting the Local Preprod
  Release Candidate plan still requires its own explicit execution
  authorization.

## Risks And Mitigations

| Risk | Mitigation |
|---|---|
| Existing 17 events omit important data | Complete coverage register, owner scope decision, separate AviaCore successor cut, and separate producer mirror before code |
| "All data" becomes sensitive overcollection | Named purpose/owner, closed field allowlists, privacy classes, forbidden-inline tests |
| User transaction depends on AviaCore | Transactional producer outbox and asynchronous publisher |
| At-least-once becomes a false exactly-once claim | Immutable producer event ID/content and AviaCore idempotent admission |
| Generic outbox semantics mix internal jobs and analytical facts | Dedicated immutable feed event and attempt state |
| Replays rewrite historical truth | New correction/supersession events and append-only attempts |
| Feature leakage creates misleading ML results | Point-in-time availability, known/effective time, leakage tests |
| Synthetic evidence is promoted to production | Literal readiness ladder and separate source/pilot/target gates |
| Cross-repo contract drift | Commit/digest lock, golden vectors, two-repo check |
| Contract package is internally inconsistent | Exhaustive integrity matrix, repaired behavioral digest, and new owner authorization before Phase 2.3 |
| Tenant is confused with inspected organization | Separate authenticated tenant/source/actor/subject organization and visibility keys in every layer |
| Synthetic fixture evidence is called AviaCore L1/L2 | Fixed local-candidate/scaffold labels and machine-readable non-training state |
| One umbrella task bypasses phase authority | Nine separately authorized and evidenced Task 7 slices with a stop after each |

## Idempotence And Recovery

- One business result produces one immutable event identity in the same
  transaction.
- Same event ID and same digest may retry and converge; same ID and different
  digest is an immutable conflict.
- Producer acknowledgement never advances before an exact AviaCore receipt.
- Replay changes attempts, not canonical event content.
- A failed cross-repository run preserves both evidence roots and cleans only
  its named local services, certificates, networks, and profile data.

## Decisions

- Use the already approved `event_api` and direct-mTLS contract; do not design
  Kafka or object-export alternatives.
- Treat AviaCore as a separate bounded repository and runtime.
- Keep a digest-locked producer conformance bundle for independent builds;
  behavioral identity and owner authorization remain separate.
- Use a dedicated feed outbox rather than interpreting every internal worker
  outbox message as an analytical event.
- Use only local synthetic analytics-candidate and experiment-readiness
  scaffold labels here. Source-bound L1/L2 and production ML require later
  separately authorized plans and evidence.

## Discoveries

- AviaCore already has a repository-owner-approved local `contract-ready`
  contract and a managed remediation plan.
- AviaCore Phase 2.1, 2.2, the separately authorized local-fixture Phase 2.3
  admission boundary, and Phase 2.4 candidate persistent-state slice are
  verified locally. A connected admission runtime/raw-manifest slice and later
  work require their own separate authorization.
- The current approved event catalog contains 17 event types and explicitly
  forbids free text, person/contact values, Evidence binaries, filenames,
  finding descriptions, and investigation notes inline.
- The AviaCore normative plan already defines ingestion, immutable raw landing,
  quarantine, Data Vault, Dagster, Soda, DQ, recovery, producer conformance,
  sandbox integration, canary, pilot, and target approval gates.
- The 17 types do not cover all persisted AviaSurveil source families, and the
  current v1 package has correction/supersession, hash-registry, tombstone, and
  negative-vector integrity gaps that block Phase 2.3 until repaired.
- AviaCore's readiness authority defines L1 as real source-bound ingestion and
  L2 as an approved label/window, point-in-time/leakage evidence, source-bound
  dataset, and offline evaluation. This local synthetic plan satisfies neither.

## Outcome Notes

Planning and independent read-only cross-repository inspection only. Contract changes,
AviaCore implementation, AviaSurveil producer implementation, external
identity, connected ingestion, Git publication, and deployment are `not run`.

## Execution Prompt

```text
Execute docs/exec-plans/active/2026-07-27-aviacore-and-ml-data-readiness-plan.md only after the identity/data predecessor is accepted and the user separately authorizes execution. Read both repositories' AGENTS.md, architecture, plan policies, indexes, this complete plan, AviaCore's active aviasurveil-production-data-feed-readiness-remediation plan, its path-bound normative plan, and the exact v1 contract set first.

Treat /Users/marlonjd/Developer/monorepos/aviaCore as a separate repository through AVIACORE_ROOT. AviaSurveil owns transactional writes, event semantics, serialization, outbox, and delivery; AviaCore owns contracts, admission, raw landing, historical integration, DQ, governed products, and controlled ML access. Do not start producer code generation or AviaCore Phase 2.3 until the known correction/supersession, hash-registry, tombstone, and conformance-vector gaps have a new behavioral digest and current authorization. Never write directly across databases, never weaken privacy or tenant boundaries, never claim exactly-once, and never promote local synthetic evidence to source, L1/L2/L3, pilot, target, production, or production-ML evidence.

Execute the authorized slices in their defined order. Task 3 is only a tracking umbrella: separately authorize the AviaCore successor cut, stop, then separately authorize the AviaSurveil mirror/codegen and stop. Task 7 is also only a tracking umbrella: execute exactly one AviaCore Phase 2.3/2.4/2.5/2.6/2.7/3/4/5A/5B slice per separately authorized task and stop after its evidence/index update. Keep `training_allowed: false` and `production_ml_readiness: NOT_READY`. Do not add Kafka, Kubernetes, Feast, MLflow, real PII, Evidence bytes, unrestricted free text, AWS actions, commits, or pushes without separate authorization. Keep both repositories' plans and evidence truthful and stop on any contract-root, privacy, ownership, or readiness mismatch.
```
