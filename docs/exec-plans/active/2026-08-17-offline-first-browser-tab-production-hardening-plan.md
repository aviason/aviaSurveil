# Offline-First Browser Production Hardening

Date: 2026-08-17
Last updated: 2026-08-17 (P1 candidate controls verified locally)
Status: active — Gate 0 through P1 candidate slices `verified locally`; connected DB/object-store/device/owner/release evidence `not run`/`blocked`; `candidate-only`; `release pending`; production offline-safe `blocked`

## Planning authority

This plan is governed by [`docs/PLANS.md`](../../PLANS.md), the repository
[agent guide](../../../AGENTS.md), the
[verification matrix](../../agent-harness/verification-matrix.md), and the
[output contract](../../agent-harness/output-contract.md).

The superseded historical implementation record is the
[React Vite PWA And Go Offline-First Production plan](2026-07-20-react-vite-pwa-go-offline-first-production-plan.md).
It is evidence and a source-location map only. It authorizes no current
implementation, installed-runtime scope, whole-object upload design, 24-hour
grant policy, acceptance criterion, or release action. This plan is the sole
current authority for AviaSurveil browser offline-production hardening.
The active
[App-Shell Cache And Exact-Vector Convergence plan](2026-08-17-app-shell-cache-convergence-plan.md)
owns the current app-shell source slice. This plan may not edit overlapping
source files until that owner records an explicit handoff.

A plan does not authorize branch operations, commits, pushes, deployments,
external writes, origin changes, or production actions. Those actions require
separate current user authority.

## Objective and user-visible outcome

Deliver two independent lanes:

1. A fast online/mock browser demo that is not blocked by production offline
   certification.
2. A separately built and released production browser runtime in which an
   Inspector can work offline for the approved multi-day lease, survive the
   covered failure model, synchronize with at-least-once delivery and exactly-once
   domain effects, and see completion only after a canonical server receipt is
   persisted locally.

The only permitted data-loss claim is:

```text
Covered failure modelinde acknowledged application mutations için zero silent loss.
```

An application mutation is acknowledged only after its entity, immutable
outbox operation, local sequence, and local commit receipt have completed one
IndexedDB transaction and have been read back. The claim excludes pre-save
input, explicit site-data clearing, profile/device/disk loss, browser or OS
defects outside the qualified model, and unverified disaster recovery.

## Scope

### Demo lane

- Online/mock browser UI flows.
- Typecheck, build, mock E2E, focused offline unit/component tests, and static
  app-shell artifact checks.
- An optional desktop Chrome candidate smoke only when needed to detect an
  offline-code build/regression defect before the demo.
- Literal `candidate-only` labeling.
- No real-device, browser-kill, OS-restart, power-loss, or multi-day evidence.
- No production-ready or production offline-safe claim.

### Official production browser matrix

Only these normal browser-tab families are in production scope:

1. iOS/iPadOS Safari.
2. iOS/iPadOS Chrome in the same qualified WebKit/capability lane as Safari.
   No Chrome-specific implementation is permitted. An engine or capability
   mismatch is online-only or unsupported.
3. macOS Safari.
4. macOS Chrome.
5. Android Chrome.
6. Windows Chrome.

### Explicit exclusions

- Installed PWA and every installed/standalone runtime. It is not a feature,
  future phase, qualification row, or production claim in this plan.
- iOS/iPadOS browsers other than Safari and Chrome.
- Android browsers other than Chrome.
- macOS browsers other than Safari and Chrome.
- Windows browsers other than Chrome.
- Private/Incognito and Lockdown Mode.
- Browser beta, dev, and canary channels as production evidence.
- Background Sync as a correctness requirement.
- Automatic merge or last-write-wins conflict handling.
- Automatic deletion of referenced, pending, unknown, rejected, quarantined,
  or sole-copy local work.
- Production deployment, DNS/origin migration, traffic changes, secret writes,
  or cloud infrastructure changes.

Unsupported browsers receive no new implementation. They may receive only a
fail-closed negative admission test.

## Verified repository baseline

The following facts were confirmed by read-only inspection. Runtime fixes and
production tests remain `not run`.

| Surface | Current fact | Required successor behavior |
| --- | --- | --- |
| Service Worker | `apps/web/src/sw.ts` calls `skipWaiting()`, `clients.claim()`, and navigates clients. | Waiting candidate, safe checkpoint, no forced reload or navigation takeover. |
| Update policy | Pending work can still produce `ready-for-automatic-activation`. | Real dirty-form and operation quiescence must gate activation. |
| Quiescence | `ClientQuiescence` exists but production IDB/OPFS/hash/sync/mutation paths do not comprehensively hold tokens. | Bind every relevant lifecycle to shared counters. |
| Admission | `storage-readiness.ts` uses a Chrome/Chromium user-agent gate. | Official allowlist, version lane, capabilities, and runtime canary in that order. |
| IndexedDB | Entity and outbox are transactionally grouped; durability, blocked/versionchange UI, migration-kill evidence, and pull-page/cursor atomicity are incomplete. | Full atomic receipt and migration recovery contract. |
| OPFS | Manifest-first staging, hashing, promotion, and no-delete quarantine exist. | Freeze the full deterministic reconciliation matrix. |
| Upload | Client retries whole-object upload and does not persist resumable session/offset state. | Stable resumable session, part receipts, and split-brain recovery. |
| Idempotency | Server mutation transaction and revision checks exist. | Retention, dependency poison, profile binding, and conflict-resolution operations. |
| Pull | Device identity is not carried end to end through the HTTP client/handler boundary. | Required device/profile-bound cursor and authorization. |
| Authority | Random `deviceInstanceId` is correlation metadata, not security authority. | Non-exportable proof-of-possession profile key. |
| Offline grant | Grant duration is 24 hours and authorization is tied to the issuing session. | Multi-day lease independent of OIDC session lifetime. |
| Scan/finalization | Scan state is not a complete finalization chain and no canonical final receipt exists. | CLEAN-gated server manifest and immutable receipt. |
| Audit/diagnostics | Server audit is partial; no complete event catalog, privacy-safe support bundle, origin freeze, or rollback recovery contract exists. | Complete P1 controls. |

The Surveil working tree was already dirty before this plan update. All
execution must preserve unrelated changes and record a fresh baseline before
the first source edit.

## Ownership and repository orientation

| Owner | Surfaces and evidence responsibility |
| --- | --- |
| AviaSurveil Web | Admission, Service Worker, IndexedDB, OPFS, outbox client, sync client, diagnostics, and browser evidence. |
| AviaSurveil API | Protocol, idempotency, lease, upload, scan, finalization, authoritative event chain, and PostgreSQL evidence. |
| AviaAuth | Current subject/session/revoke/disable authority. No implicit Auth source change is authorized. |
| AviaWorkspace Release | Exact production origin, artifact provenance, release descriptor, and rollback execution through a separate root plan. |
| Product/CAA Operations | Maximum offline duration, conflict/finalization semantics, and audit granularity. |
| Security/Records | Profile authority, idempotency/quarantine retention, access policy, diagnostics redaction, and security acceptance. |
| QA/Device Lab | Failure injection and physical device/browser qualification. |

## Assumptions and affected interfaces

### Assumptions

- Production qualification targets normal browser tabs on one frozen
  same-origin application boundary.
- PostgreSQL and canonical object metadata become authoritative only after the
  matching idempotent server receipt; unacknowledged client state remains local
  candidate state.
- Correctness is foreground-first. Reconnect, visibility, explicit Sync, and
  bounded open-page retry are required; background delivery is optional.
- The browser may suspend or terminate any tab, worker, or process between two
  durable transitions.
- Exact policy values and owner approvals are Gate 0 inputs, not assumptions
  manufactured by implementation.

### Affected interface map

| Interface | Primary surfaces |
| --- | --- |
| Browser admission and update | `apps/web/src/sw.ts`, `apps/web/src/offline/storage-readiness.ts`, `update-coordinator.ts`, `client-quiescence.ts`, bootstrap, and app-shell scripts/tests. |
| Local database and outbox | `apps/web/src/offline/db.ts`, `schema-migrations.ts`, `field-repository.ts`, `outbox.ts`, sync state, and migration/restart tests. |
| OPFS and attachment recovery | OPFS store, attachment recovery, hash worker, upload-session persistence, and attachment browser/unit tests. |
| Transport contract | `api/openapi/source/`, generated TypeScript/Go contracts, canonical examples, and contract checks. |
| Server sync and authority | `apps/api/internal/sync/`, HTTP handlers, identity/session read boundaries, PostgreSQL stores/migrations, and integration tests. |
| Upload, object store, and scan | Inspection attachment service, object-store interfaces, evidence worker projection, upload-session tables, and MinIO/managed-scan tests. |
| Finalization and audit | New canonical finalization service/store/receipt, authoritative event chain, diagnostics schema, and support/release evidence. |

## Ordered implementation

The order below is mandatory. Demo and production acceptance remain separate.

### Gate 0 — Policy and contract freeze

No production source edit begins before this gate records every decision and
owner. Missing production decisions block production but not the demo lane.

#### Policy register

| Policy | Freeze requirement |
| --- | --- |
| Official allowlist | Exactly the six production browser families in this plan. |
| Version policy | Safari/WebKit: release-time Stable N and N-1 with an exact vendor-supported OS/security patch floor. Chrome: Stable N and N-1. Exact OS/browser builds are recorded in every release packet. |
| iOS/iPadOS lane | Safari and Chrome share one WebKit/capability implementation and evidence model; both browser brands still receive qualification rows. |
| macOS lane | Safari and Chrome are separate qualification rows under one platform policy. |
| Android/Windows lanes | Chrome only, Stable N and N-1. |
| Maximum offline lease | Proposed freeze: seven server-time days, capped by package validity. Product and Security must ratify or replace this exact value. |
| Idempotency response retention | Proposed freeze: 400 days after terminal outcome. The operation ID/hash/effect tombstone remains for the canonical record-retention period. |
| Quarantine | Proposed freeze: minimum 180 days. Unresolved and sole-copy material is not automatically purged. Access is restricted to named Records/Security reviewers plus audited break-glass. |
| Manifest canonicalization | `avia-finalization-manifest/v1`: UTF-8 RFC 8785 canonical JSON, stable array ordering, and SHA-256. |
| Origin | One exact `scheme://host[:port]` supplied by the Workspace release owner. Alias and redirect origins are not equivalent. |
| Device/profile authority | Non-exportable proof-of-possession profile key. Random `deviceInstanceId` remains correlation metadata only. |
| Acknowledged-write durability | Request strict IndexedDB durability where the runtime exposes it; otherwise require an explicitly qualified platform-equivalent result. Attachment acknowledgement requires OPFS close, reopen, and hash readback. Any lane with observed acknowledged loss under final sudden-power-loss testing becomes online-only. |
| Update timeout and classification | Freeze client ACK timeout, old-vector support window, security-critical classifier, classification owner, server write-fence order, and unresponsive-client read-only disposition. Unresponsive status never authorizes activation. |
| Evidence owners | Named Web, API, Security, Records, Product, Workspace Release, Support, and Device Lab owners. |

#### Audit event catalog

Freeze these authoritative event names and their required envelope:

```text
operation.accepted
operation.rejected
operation.conflict
operation.quarantined
lease.issued
lease.expired
lease.revoked
lease.authority_changed
upload.opened
upload.part_acknowledged
upload.completed
upload.expired
upload.aborted
upload.quarantined
scan.pending
scan.clean
scan.failed
scan.quarantined
inspection.finalize_requested
inspection.finalize_rejected
inspection.finalized
inspection.reopened
inspection.corrected
```

The envelope contains event ID, authenticated subject, profile-key ID,
operation ID where applicable, package revision, request/payload hash, server
revision, result code, and server timestamp.

#### Gate 0 acceptance

- Every policy has one exact value/disposition and one named owner.
- The exact production origin is recorded or production remains `blocked`.
- The current migration head, offline version vector, dirty paths, and
  app-shell owner handoff are recorded.
- OpenAPI, TypeScript, Go, IndexedDB, and PostgreSQL forward migration order is
  frozen.
- Demo and production claims remain independent.

#### Gate 0 execution record — 2026-08-17

The Gate 0 baseline was captured before any source edit for this plan. The
commands and observations below are fresh local evidence; no live database,
release system, device lab, or external owner system was queried.

| Baseline | Observed result |
| --- | --- |
| Workspace dirty tree | `main...origin/main`; pre-existing dirty tracked paths are `compose/gateway/Caddyfile`, `compose/gateway/Caddyfile.cloud`, `docs/exec-plans/active/2026-08-17-app-shell-cache-oidc-convergence-plan.md`, and `tests/test_workspace.py`; pre-existing untracked paths are `output/` and `presentation/`. |
| Nested repository baseline | `apps/surveil` `8ce5a22bac588ad06c7beaf23c1eb4d4c1a34fa1`, dirty in 11 paths; `shared/auth` `5be167ab1529b371a14c64566651a58c1de73b04`, clean; `shared/data` `9090c5e6e58eeda2834c94bc7bd0265465599573`, clean. `make repos-status` reported `auth clean`, `data clean`, `surveil dirty (11 paths)`. |
| Embedded API migration head | `apps/api/migrations/000049_canonical_audit_type_focus_policy.up.sql`; `apps/api/migrations/migrations.go` reports `LatestVersion = 49`. Applied live-database head: `not run` because no database runtime was opened. |
| Current app/SW/DB/protocol vector | `appShellVersion = 9`, `indexedDbSchemaVersion = 2`, `packageSchemaVersion = 1`, `syncProtocolVersion = 1`, from `apps/web/src/offline/offline-version-contract.ts`. |
| App-shell ownership/handoff | The active App-Shell Cache And Exact-Vector Convergence plan owned `apps/web/src/sw.ts`, update coordination, client quiescence, app-shell manifest/cache/vector generation, and related gateway/cache source. On 2026-08-17 it recorded an explicit handoff of the P0-5 Service Worker/update/quiescence source slice to this plan. It retains release provenance, gateway/cache ownership, and public cutover authority. |

The following values are the exact current policy dispositions. A proposed
value is recorded as a proposed value, not as Product, Security, Records,
Support, or Release approval.

| Policy | Frozen value/disposition | Owner/evidence status |
| --- | --- | --- |
| Official browser allowlist | Exactly iOS/iPadOS Safari; iOS/iPadOS Chrome in the same WebKit/capability lane; macOS Safari; macOS Chrome; Android Chrome; Windows Chrome. Unsupported browsers receive only negative fail-closed admission coverage. | Scope is fixed by this plan; named Product/Security approvers were not supplied. |
| Official version policy | Safari/WebKit Stable N and N-1 with an exact vendor-supported OS/security patch floor; Chrome Stable N and N-1; exact builds belong in each release packet; beta/dev/canary and private/Incognito/Lockdown Mode are excluded. | Exact release-time build list is `not run`/release pending; named Device Lab and Security approvers were not supplied. |
| Maximum offline lease | Proposed `7 server-time days`, capped by package validity; client clock cannot extend it. | Product and Security ratification or replacement is unavailable: production `blocked`. |
| Idempotency response retention | Proposed `400 days after terminal outcome`; operation ID/hash/effect tombstone remains for canonical record-retention period. | Records/Security retention approval and named owner unavailable: production `blocked`. |
| Quarantine retention/access | Proposed minimum `180 days`; unresolved and sole-copy material is never auto-purged; access restricted to named Records/Security reviewers plus audited break-glass. | Named reviewers, break-glass authority, and retention approval unavailable: production `blocked`. |
| Finalization canonicalization | `avia-finalization-manifest/v1`; UTF-8 RFC 8785 canonical JSON, stable array ordering, SHA-256. | Contract value recorded; Product/Records canonicalization approval not supplied. |
| Exact production origin | One permanent exact `scheme://host[:port]` owned by AviaWorkspace Release; alias/redirect origins are not equivalent. Current `https://demo.aviasurveil.com` is a local-origin demo and is not a production value. | Exact production origin and named Workspace Release owner were not supplied: production `blocked`; no origin change was made. |
| Acknowledged-write durability | Request strict IndexedDB durability where exposed; otherwise require an explicitly qualified platform-equivalent result. Attachment acknowledgement requires OPFS close, reopen, and hash readback. Any observed acknowledged loss in final sudden-power-loss qualification makes that lane online-only. | Physical/device qualification is `not run`; Security/QA/Device Lab acceptance is release pending. |
| Unresponsive-client security fence | Normal path requires every client ACK. `ACK_TIMEOUT` never authorizes activation. At security-critical or old-vector deadline, order is `SERVER_MINIMUM_WRITE_VECTOR_COMMITTED` → `RESPONSIVE_CLIENTS_FROZEN_AND_ACKED` → `UNRESPONSIVE_CLIENT_FENCED_READ_ONLY_PENDING_RESUME` → `SECURITY_UPDATE_ENFORCED_SHELL_PENDING`; activation still waits for safe ACK or client exit. | Client ACK timeout, old-vector support window, security-critical classifier, classification owner, and exact fence deadline were not supplied: production `blocked`; no takeover policy was weakened. |
| Named evidence owners | Role boundaries are Web, API, Auth, Workspace Release, Product/CAA Operations, Security/Records, Support, QA/Device Lab. | Named individuals and explicit acceptance receipts were not supplied for Product, Security, Records, Support, Release, or Device Lab: production `blocked`. |

The authoritative audit catalog frozen for implementation is:

```text
operation.accepted
operation.rejected
operation.conflict
operation.quarantined
lease.issued
lease.expired
lease.revoked
lease.authority_changed
upload.opened
upload.part_acknowledged
upload.completed
upload.expired
upload.aborted
upload.quarantined
scan.pending
scan.clean
scan.failed
scan.quarantined
inspection.finalize_requested
inspection.finalize_rejected
inspection.finalized
inspection.reopened
inspection.corrected
```

The event envelope remains event ID, authenticated subject, profile-key ID,
operation ID where applicable, package revision, request/payload hash, server
revision, result code, and server timestamp. The live audit-chain, retention,
and access qualification is `not run`.

Gate 0 is therefore **recorded for the online/mock demo and non-overlapping
candidate slices**. Production offline-safe remains `blocked` on the exact
owner decisions and physical evidence above. No source implementation claim
or production-readiness claim is made by this record.

### P0-1 — Demo/production separation

#### Implementation direction

- Keep the demo build as an online/mock artifact. Offline Ready checkout is not
  a demo requirement.
- If an offline behavior is shown, use a separately identified candidate
  artifact and label every result `candidate-only`.
- Create a distinct production-offline build/release identity, cache namespace,
  version vector, artifact manifest, and acceptance lane.
- Do not allow demo artifact success or demo evidence to satisfy a production
  gate.
- Pin the exact demo artifact for a scheduled demo and do not replace it while
  the demo session is active.

#### Acceptance

- Online/mock UI works without production offline checkout.
- Demo build metadata contains no production-ready or production offline-safe
  claim.
- Production offline runtime has a separate immutable artifact identity and
  release gate.
- No installed-specific implementation, acceptance evidence, or claim is added;
  demo and production normal-tab lanes do not use installation.

#### P0-1 execution record — 2026-08-17

- Added an explicit build-artifact contract with separate `demo`, `http`, and
  `production-offline` lanes. The production-offline lane is HTTP-backed,
  normal-tab only, and remains `candidate-only`.
- Added immutable lane identity and logical cache namespace metadata to the
  emitted `build-artifact.json`; the production-offline lane is
  `aviasurveil360-production-offline-browser-tab` with namespace
  `aviasurveil360-production-offline-app-shell` and acceptance lane
  `production-offline-browser-matrix`.
- Added the `build:offline` candidate build and a distinct
  `dist/production-offline` artifact. The index carries the lane marker, so
  the app-shell fingerprint differs from demo/HTTP artifacts without editing
  business data or storage.
- Focused failing tests were written first in
  `src/app/build-profile.test.ts` and
  `src/app/build-artifact-contract.test.ts`; after implementation they passed
  `5/5`.
- Fresh checks: `npm run typecheck`, `npm run build:demo`,
  `npm run build:http`, `npm run build:offline`,
  `npm run check:build-artifacts`, both HTTP boundary scans, and app-shell
  artifact scans passed. Result: `verified locally`, `candidate-only`.
- No browser acceptance, physical device evidence, release action, or
  production-origin change was run; those remain `not run`/`blocked`.

### P0-2 — Capability and policy admission

Admission order is immutable:

```text
official browser allowlist
  -> supported version lane
  -> secure context
  -> Service Worker capability
  -> IndexedDB transaction capability
  -> OPFS write/read/close/hash/recovery capability
  -> storage persistence/quota capability
  -> runtime canary
  -> Offline Ready
```

#### Implementation direction

- UA/browser metadata may classify the official browser/version lane but is
  never sufficient for capability or security authority.
- Passing capabilities does not make an allowlist-external browser supported.
- Represent admission as `OFFLINE_READY`, `ONLINE_ONLY`, `UNSUPPORTED`, or
  `RECOVERY_REQUIRED` with stable reason codes.
- Use one iOS/iPadOS WebKit implementation for Safari and Chrome. Any engine or
  capability mismatch is online-only or unsupported.
- The runtime canary performs a namespaced IndexedDB transaction, OPFS
  write-close-read-hash recovery check, actual quota-sensitive write, and
  restart/readback integrity check.
- Persistence and quota estimates remain advisory; every actual write handles
  quota failure without deleting user work.
- Private/Incognito, Lockdown Mode, and every unsupported browser fail closed.

#### Acceptance

- An allowlist-external Chromium browser remains unsupported even if every
  capability probe passes.
- iOS/iPadOS Chrome contains no browser-specific OPFS/storage implementation.
- Offline Ready never appears before canary and restart/readback success.
- Site-data clearing produces `LOCAL_DATA_CLEARED`, never a false recovery
  promise.

#### P0-2 execution record — 2026-08-17

- Added a pure official browser classifier for exactly the six allowlisted
  families. iOS/iPadOS Safari and Chrome both resolve to the `webkit` lane;
  Edge, Firefox, Samsung Browser, Opera, Chromium, headless, and unknown
  browser signatures fail closed.
- Added Stable N/N-1 browser policy input with optional exact OS-version floors.
  Missing release-time numeric policy returns `BROWSER_POLICY_UNAVAILABLE`;
  it never silently treats a capability-positive browser as supported.
- Admission now exposes `OFFLINE_READY`, `ONLINE_ONLY`, `UNSUPPORTED`, and
  `RECOVERY_REQUIRED` states with stable reason codes. The sequence is browser
  policy → secure context → Service Worker → IndexedDB → OPFS → persistence /
  quota → restart/readback → runtime canary → grant/vector checks.
- The runtime dependencies now execute namespaced IndexedDB and OPFS
  write-close-read-hash probes plus a restart marker. Site-data loss is
  classified as `LOCAL_DATA_CLEARED` and no recovery promise is emitted.
- Updated the field attestation copy and browser fixtures to name the exact
  official browser/version lane; no unsupported-browser implementation was
  added.
- Focused failing tests were written first in `src/offline/browser-policy.test.ts`
  and `src/offline/storage-readiness.test.ts`. Fresh focused result:
  `25/25` tests passed, including runtime-canary order and failure injection.
  Result: `verified locally`, `candidate-only`.
- Production numeric browser policy, OS/security patch floors, physical
  qualification, and named owner acceptance remain `not run`/`blocked`.

### P0-3 — IndexedDB atomicity and migration

#### Transaction contract

Every acknowledged application mutation writes, in one transaction:

- the entity revision and canonical content hash;
- one immutable outbox operation;
- the local operation sequence;
- a local commit receipt; and
- package/profile binding.

All records commit or none commit. Save success appears only after transaction
completion and a new-transaction readback of entity, outbox, and receipt.

Each pull page and its next cursor also commit in one transaction.

#### Durability policy

- Request strict IndexedDB durability through the narrow database adapter when
  the runtime exposes it, and record the observed mode in qualification
  evidence.
- A transaction completion event and immediate readback are necessary but do
  not alone prove survival of sudden power loss.
- Where no strict durability control is exposed, the lane remains
  `UNKNOWN / REQUIRES TESTING` until the final physical sudden-power-loss class
  qualification passes on the exact supported OS/browser build.
- Any target that loses an acknowledged mutation in the covered physical test
  becomes online-only; the result may not be waived into production support.

#### Migration and recovery

- Expose `blocked`, `versionchange`, and unexpected-close states in the UI.
- Issue bounded cross-tab close requests and elect one migration owner.
- Keep migrations deterministic and network-free.
- Migration failure opens read-only recovery; it never clears storage.
- No rollback path performs an IndexedDB downgrade.
- Keep the predecessor app shell until new-schema integrity passes.
- Bind every transaction, migration, and dirty form to shared quiescence.

#### Acceptance

- After every abort/kill boundary, entity+outbox+sequence+receipt is entirely
  absent or entirely present.
- Local entity revision and hash equal the acknowledged payload.
- Quota failure preserves all earlier data.
- Migration kill produces complete-old, complete-new, or visible read-only
  recovery, never a partially editable schema.
- Pull page without cursor and cursor without pull page are impossible.
- Every qualified lane either records strict/platform-equivalent durability
  evidence or remains online-only.

#### P0-3 execution record — 2026-08-17

- Added per-subject/package operation sequence markers and immutable local
  commit receipts under the existing foundation store. The entity mutation,
  outbox row, sequence advance, request/payload hash, and receipt are written
  in one strict-durability IndexedDB transaction.
- Outbox rows now retain `operationSequence`, `entityRevision`, `entityHash`,
  and `commitReceiptKey`. Save, draft, submit, and attachment-registration
  paths perform a separate transaction readback of the outbox and receipt
  before returning success.
- Dexie requests `chromeTransactionDurability: "strict"` through the narrow
  database adapter. The observed physical durability result remains
  `UNKNOWN / REQUIRES TESTING` until the final device/power-loss gate.
- Added forward-only legacy-row reconciliation on open. It fills missing
  receipt metadata and preserves existing entity/outbox data; it never clears,
  downgrades, or auto-deletes IndexedDB/OPFS work. The schema vector remains
  `{appShell: 9, indexedDb: 2, package: 1, protocol: 1}` because no object
  store/index shape changed and the app-shell owner has not handed off vector
  source ownership.
- Exposed `CLOSED`, `OPENING`, `OPEN`, `BLOCKED`, `VERSION_CHANGE`, and
  `READ_ONLY_RECOVERY` lifecycle states. Migration failure remains visible and
  read-only.
- Focused failing tests were written first for strict durability, atomic
  receipt fields, lifecycle state, and forward reconciliation. Fresh
  typecheck plus field repository, sync engine, and readiness tests passed
  `54/54`. Result: `verified locally`, `candidate-only`.
- Physical sudden-power-loss, storage-pressure, and real-browser durability
  qualification are `not run`; production evidence remains `blocked`.

### P0-4 — OPFS manifest-first recovery

Required states:

```text
MANIFEST_COMMITTED
WRITING_TEMP
FLUSHED
HASH_VERIFIED
PROMOTED
LOCAL_READY
RECOVERY_REQUIRED
QUARANTINED
```

#### Deterministic reconciliation matrix

| Observation | Required action and state |
| --- | --- |
| Manifest only | `RECOVERY_REQUIRED`; permit a restartable write; never show ready or delete the manifest. |
| Partial temp | Verify size/hash; preserve incomplete bytes in `RECOVERY_REQUIRED`; do not auto-delete. |
| Verified temp | Rehash, promote, then atomically commit manifest+registration outbox metadata to `LOCAL_READY`. |
| Temp plus final | If final matches, final is canonical and temp is retained/quarantined. If only temp matches, move the mismatching final to quarantine before promotion. Preserve both until disposition. |
| Final only | If size/hash matches, finish metadata commit; otherwise `QUARANTINED`. |
| READY metadata plus missing final | `RECOVERY_REQUIRED`; recover from a valid temp or block upload/finalization. |
| Hash mismatch | `QUARANTINED`; no upload or finalization. |
| Unknown object | Inventory and quarantine; never auto-delete. |
| Quota failure | Abort the attempted state change, preserve prior data, and expose `RECOVERY_REQUIRED` where needed. |
| Process kill after promotion | Rehash the final object and idempotently complete missing metadata. |
| Process kill before metadata commit | Preserve the promoted object and replay only the IDB metadata/outbox transaction. |

Referenced, pending, unknown, and sole-copy bytes are never automatically
deleted.

#### Acceptance

- `LOCAL_READY` appears only after final-path readback and hash verification.
- Attachment acknowledgement requires writable close, a fresh handle/reopen,
  full size/hash readback, and commit receipt persistence. These steps remain
  subject to final sudden-power-loss qualification because close/readback alone
  is not a physical-media guarantee.
- The design never claims that IndexedDB and OPFS share an atomic transaction.
- Every crash point produces deterministic recovery or quarantine.
- Corrupt bytes never reach upload or finalization.

#### P0-4 execution record — 2026-08-17

- Added the deterministic OPFS recovery-state contract:
  `MANIFEST_COMMITTED`, `WRITING_TEMP`, `FLUSHED`, `HASH_VERIFIED`,
  `PROMOTED`, `LOCAL_READY`, `RECOVERY_REQUIRED`, and `QUARANTINED`.
- Manifest-first staging now records each phase. Promotion preserves the
  referenced temporary copy; it does not delete pending, referenced, unknown,
  or sole-copy bytes.
- Recovery compares final and temporary objects independently. A valid final
  plus mismatching temp is quarantined; a valid temp plus mismatching final is
  quarantined without overwrite; a partial temp remains restartable
  `RECOVERY_REQUIRED`; exact temp recovery reopens and rehashes the promoted
  final path before metadata/outbox commit.
- Hash, size, quota, final-readback, and process-boundary failures preserve
  bytes and produce deterministic recovery/quarantine state. Unknown paths
  become quarantine metadata and remain present.
- Focused failing tests were written first for state transitions, temp
  retention, split-object mismatch, partial temp, and final readback. Fresh
  `npm run typecheck` and OPFS/recovery tests passed `39/39`. Result:
  `verified locally`, `candidate-only`.
- Physical OPFS/device/process-kill, storage-pressure, and sudden-power-loss
  evidence are `not run` and remain final-release-only.

### P0-5 — Safe Service Worker update

Required safe checkpoint:

```text
candidate verified
  -> all clients informed
  -> mutation freeze
  -> dirty forms empty
  -> IDB/OPFS/hash/sync counters zero
  -> durable local work acknowledged or explicitly quarantined
  -> all clients ACK
  -> activation
  -> migration
  -> verification
  -> user-controlled reload
```

#### Implementation direction

- Active local work forbids `skipWaiting`, `clientsClaim`, forced reload, and
  application navigation takeover.
- Exact version-vector equality is compatibility evidence, not quiescence.
- Bind actual dirty forms, mutations, IDB, OPFS, hashing, upload, and sync work
  to `ClientQuiescence` counters.
- Retain active, candidate, and one verified predecessor app-shell cache.
- Delete only verified app-shell caches after new-shell startup, migration,
  integrity, and client acknowledgement. Never delete IDB/OPFS business data.
- Rollback enters the predecessor shell or read-only recovery without database
  downgrade.

#### Suspended or unresponsive client path

- Freeze a bounded ACK timeout, old-vector support window, security-critical
  classification rule, and named classification owner at Gate 0.
- The normal transition requires every client ACK. A missing ACK moves the
  client to `ACK_TIMEOUT`; it does not silently count as acknowledgement.
- A non-security update may remain waiting only until the frozen old-vector
  support deadline. Before that deadline it never reloads or navigates the
  unresponsive client.
- At a security-critical decision or old-vector deadline, apply this bounded
  security-enforcement path in order:

```text
ACK_TIMEOUT
  -> SERVER_MINIMUM_WRITE_VECTOR_COMMITTED
  -> RESPONSIVE_CLIENTS_FROZEN_AND_ACKED
  -> UNRESPONSIVE_CLIENT_FENCED_READ_ONLY_PENDING_RESUME
  -> SECURITY_UPDATE_ENFORCED_SHELL_PENDING
```

- The server write fence makes security enforcement bounded without pretending
  the unresponsive client's local-work state is known. The waiting worker does
  not activate from timeout or fencing alone.
- `skipWaiting` may be messaged only after the previously unresponsive client
  resumes, freezes, durably acknowledges/quarantines local work, and ACKs the
  normal safe checkpoint. If the client exits, the browser may complete the
  normal Service Worker lifecycle without `skipWaiting`.
- Mutations from the fenced old vector are rejected/quarantined, never silently
  applied. Security enforcement therefore does not depend on the suspended
  document receiving a message.
- When an old suspended client resumes, it enters
  `SECURITY_UPDATE_REQUIRED_READ_ONLY`, preserves committed local work, and
  requires durability acknowledgement or quarantine before user-controlled
  reload.
- Until ACK or client exit, new/old documents may show
  `SECURITY_UPDATE_ENFORCED_SHELL_PENDING` and remain read-only. Candidate
  activation never calls `clientsClaim`, navigates, or force-reloads the old
  document.

#### Acceptance

- Active inspection deploy always remains at `WAITING_FOR_SAFE_CHECKPOINT`.
- Dirty form or non-zero operation counter always blocks activation.
- An unresponsive client cannot block server-side security enforcement, but it
  may legitimately keep shell activation waiting until ACK or exit.
- Evidence proves the server write fence before read-only enforcement and
  proves the resumed old document is read-only before any further mutation.
- Old shell/new backend mismatch produces an explicit read-only/update state,
  not data deletion or an unclassified error.

#### P0-5 handoff resolution — 2026-08-17

The app-shell owner recorded the explicit handoff for the P0-5
Service Worker/update/quiescence source slice. Release provenance, gateway/cache
ownership, and public cutover authority remain with the app-shell/Workspace
release boundary. P0-5 implementation may proceed within the handed-off slice.

#### P0-5 execution record — 2026-08-17

- Replaced automatic Service Worker activation with a safe checkpoint:
  `skipWaiting` is reachable only after every current same-origin window sends
  an exact-vector, zero-dirty-form, zero-counter, durable-work ACK. Missing or
  unresponsive clients keep the candidate waiting.
- Added the unresponsive-client fence state machine:
  `ACK_TIMEOUT` → server-fence/responsive-client acknowledgement →
  `UNRESPONSIVE_CLIENT_FENCED_READ_ONLY_PENDING_RESUME`. Timeout or fencing
  alone never authorizes takeover.
- Added `ClientQuiescence.freezeForSafeCheckpoint()` and explicit ACK payloads.
  New work cannot start after freeze. Activation emits `requiresUserReload`;
  it does not call `window.location.reload()` or navigate legacy clients.
- Updated app-shell artifact checks and handoff documentation to reject the
  retired forced-navigation contract while retaining exact-vector and cache
  integrity checks.
- Focused coordinator/quiescence/SW/manifest tests passed `60/60`; typecheck,
  demo/http/production-offline builds, app-shell artifact scans, and browser
  smoke passed. The in-app Browser built preview rendered a blank DOM, so the
  permitted Playwright fallback was used; it observed `activeWorker=activated`,
  safe-checkpoint messaging, `forcedNavigation=false`, one navigation, and
  zero console errors.
- Result: `verified locally`, `candidate-only`. Physical browser/device,
  process-kill, OS-restart, storage-pressure, and sudden-power-loss evidence
  remain `not run`/`release pending`.

### P0-6 — Outbox, idempotency, and conflict

Every operation carries:

```text
operation_id
operation_sequence
package_revision
device_profile_binding
actor_subject
base_server_revision
payload_hash
request_hash
protocol_version
dependencies
```

`actor_subject` is a lease-bound claim; server authority is derived from the
authenticated principal and must match it.

#### Delivery and ledger

- Transport is at-least-once. Exactly-once domain effect is supplied by the
  PostgreSQL idempotency ledger.
- Same operation plus same request hash replays the same response.
- Same operation plus different request hash produces terminal
  `OPERATION_ID_REUSE` rejection with no domain mutation.
- Retry exhaustion becomes visible `RETRY_EXHAUSTED_REVIEW`; it never deletes
  the row.
- A poison parent quarantines only causal dependents. Unrelated safe operations
  continue.
- Apply the Gate 0 response-retention and tombstone policy.
- Bind pull request, cursor, authorization, and response page to the exact
  subject/profile/device/package scope.

#### Conflict

- Never use last-write-wins.
- Freeze only the affected entity and pull authoritative server state.
- `ACCEPT_SERVER`, `KEEP_LOCAL_AS_NEW_REVISION`, and authorized reviewer
  resolution are separate idempotent, auditable operations.
- A later local edit cannot silently supersede an unresolved conflict.

#### Acceptance

- DB commit followed by lost ACK creates one effect and replays one response.
- Wrong-device/profile pull is denied.
- Poison dependency behavior is deterministic and reviewable.
- Two devices editing one answer create a typed conflict and no silent winner.

#### P0-6 execution record — 2026-08-17

- Froze the OpenAPI source envelope for every field operation:
  `packageRevision`, `actorSubject`, `operationSequence`, `payloadHash`,
  `requestHash`, and causal `dependencies`, while pull requires
  `deviceInstanceId` and `profileKeyId`. Generated TypeScript/Go transports
  were regenerated only after the source contract changed; `contracts-check`
  passed.
- Added forward PostgreSQL migration `000050_offline_sync_hardening` for
  `idempotency_responses.terminal_at` with the proposed 400-day retention and
  profile-bound sync cursor metadata. No downgrade or destructive cleanup was
  added.
- Server operation validation now requires the complete envelope, matches
  `actorSubject` to the authenticated principal, verifies the payload hash,
  persists terminal conflict/forbidden/invalid responses for exact replay, and
  rejects operation-ID reuse with a different request hash.
- Causal dependents of a poisoned local operation become visible
  `QUARANTINED` with `DEPENDENCY_POISONED`; unrelated operations remain
  deliverable. Added typed `RESOLVE_FIELD_CONFLICT` operations with explicit
  `ACCEPT_SERVER`, `KEEP_LOCAL_AS_NEW_REVISION`, and `AUTHORIZED_REVIEWER`
  actions and an auditable server result.
- Focused failing contract tests were written before the source contract
  change. Fresh offline contract tests passed `3/3`, OpenAPI/generated checks
  passed, Go sync/http/migration/unit tests passed, and Web typecheck plus
  field/sync tests passed `36/36`. Result: `verified locally`,
  `candidate-only`.
- Connected PostgreSQL sync integration tests are `blocked`/`not run` because
  the configured disposable database at `127.0.0.1:55432` was unavailable.
  This does not claim server production readiness.

#### P0-7 execution record — 2026-08-17

- Added the profile-bound lease fields to the OpenAPI source first:
  `profileKeyId`, `profilePublicJwk`, `assignmentRevision`,
  `leaseIssuedAt`, and `leaseExpiresAt` are required for checkout/grant
  admission. Every field operation now requires `profileKeyId` and a
  non-empty `authorityProof`; the generated TypeScript and Go transports were
  regenerated from that source. The source contract test passed `4/4`,
  `scripts/check-contracts.sh` passed all `16/16`, and `git diff --check`
  passed.
- Added forward-only migration `000051_profile_authority_offline_lease`.
  It stores the profile public JWK and seven-day lease timestamps without
  downgrading or deleting existing local/server data. The embedded migration
  head remains `51`; no live migration application was run.
- The browser profile authority now creates and persists a non-exportable
  P-256 key, derives a stable public-key thumbprint, signs the canonical
  operation request hash, and refuses to mint a replacement key when durable
  local grants still reference a lost key. Local repository operations include
  the profile binding and proof; sync targets no longer use a pending profile
  placeholder.
- Checkout validates the public JWK/thumbprint pair and caps the server-time
  lease at seven days. Sync validates the payload hash, recomputes the exact
  request hash, verifies the P-256 proof against the grant’s stored public
  JWK, binds pull cursors to the grant profile, and treats the OIDC session as
  online authentication rather than the offline lease authority. Invalid
  proof is a terminal, idempotently replayable forbidden result; local key
  loss and lease expiry preserve local work.
- Focused failing tests were written before the behavior changes. Profile
  authority, key-loss, lease-bound readiness, panel checkout, field repository,
  and sync-engine tests passed `63/63`; Go sync/http/migration tests passed;
  OpenAPI/generated checks passed. Result: `verified locally`,
  `candidate-only`.
- Connected PostgreSQL FieldSync integration was attempted and is
  `blocked`/`not run` because `127.0.0.1:55432` refused the connection.
  Browser/process restart and physical device qualification remain `not run`
  and are release-gate evidence only.

### P0-7 — Offline lease and authority

Random device identity remains correlation metadata. The selected security
authority is a non-exportable proof-of-possession profile key.

#### Profile authority

- Generate a non-exportable P-256 key per qualified browser profile.
- Register the public key/thumbprint with subject and managed profile instance.
- Bind lease and operation request hashes to proof from that key.
- Test CryptoKey persistence, sign/verify, and restart behavior in the runtime
  canary.
- Key loss does not rebind an old lease. It opens read-only/quarantine recovery
  and preserves local work.

#### Lease schema

```text
subject
package
assignment_revision
command_scope
profile_device_binding
issued_at
expires_at
revoked_at
protocol_version
```

#### Lease behavior

- OIDC session is online authentication; offline lease is separate domain
  authority.
- Use the Gate 0 maximum lease, proposed as seven server-time days.
- A new valid OIDC session for the same subject/profile key may synchronize
  work issued under an expired browser session.
- Client clock never creates or extends authority.
- Lease expiry blocks new authoritative offline edits but preserves committed
  work.
- Offline revoke, role change, assignment change, or package revoke cannot be
  applied immediately. Reconnect produces explicit rejection/quarantine and
  never deletes local data.
- Logout and user switch lock local work; they do not clear it.

#### Acceptance

- Multi-day offline work remains synchronizable after original OIDC session
  expiry when the same subject/profile reauthenticates.
- Clock forward/backward cannot extend a lease.
- Revoke/role/assignment/package changes preserve the local sole copy and
  prevent unauthorized application.
- Quarantine access and retention have Security/Records acceptance.

### P0-8 — Resumable attachment upload

Required states:

```text
OPEN
UPLOADING
PARTIALLY_COMMITTED
COMPLETING
COMPLETED
EXPIRED
ABORTED
QUARANTINED
```

Persisted client/server session state:

```text
upload_session_id
session_epoch
part_size
received_parts
acknowledged_offsets
part_hashes
whole_file_sha256
expires_at
object_version
```

#### Protocol

- Begin, part, and complete operation IDs remain stable across retries.
- Part idempotency is `(upload_session_id, session_epoch, part_number,
  part_sha256)`.
- Resume only from server-confirmed parts/offsets.
- Complete verifies ordered parts, total size, whole-file SHA-256, and exact
  object identity/version.
- Session expiry may create a new epoch but not a second logical attachment
  version.
- Correctness never depends on background execution.

#### Object-store/PostgreSQL split-brain recovery

| State | Recovery |
| --- | --- |
| Object completed, PostgreSQL receipt absent | Query exact object version/hash/session metadata and idempotently commit the receipt if it matches; otherwise quarantine. |
| PostgreSQL `COMPLETING`, multipart still open | Query received parts and continue only missing parts. |
| PostgreSQL completed, object missing/mismatched | `QUARANTINED`; no success response, scan, or finalization. |
| Object and DB complete, ACK lost | Replay the same completion receipt. |
| Session expired, verified local object present | Open a new epoch for the same logical attachment/hash and retain the expired-session tombstone. |
| Conflicting object version | Preserve both identities in `PARTIALLY_COMMITTED`/`QUARANTINED`; no automatic overwrite. |

#### Acceptance

- A 50% network cut resumes only missing parts.
- Process or backend restart produces one canonical attachment version.
- Completion receipt contains exact object version, size, and SHA-256.
- Client does not show `COMPLETED` before receipt persistence and readback.

#### P0-8 execution record — 2026-08-17

- Replaced the Inspection Attachment client whole-object path with persisted
  resumable session metadata: stable begin/complete operation IDs, session
  epoch, part size, server-confirmed part numbers/offsets/hashes/object
  versions, and explicit `OPEN`, `UPLOADING`, `PARTIALLY_COMMITTED`,
  `COMPLETING`, and `COMPLETED` transitions. A retry resumes only parts that
  lack a matching server receipt; local bytes are preserved on any failure.
- Changed the OpenAPI source before generated transports. The contract now
  contains part begin/acknowledge operations, receipt-bound completion, and
  exact object version/size/hash output. TypeScript and Go transports were
  regenerated; contract checks passed `5/5` focused offline contract tests
  and all `16/16` repository contract checks.
- Added forward-only migration `000052_resumable_inspection_attachment_upload`
  with session epoch/part metadata and an append-only part receipt table.
  `LatestVersion` is now `52`; no live migration was applied.
- The API validates ordered part layout, exact declared size/hash, each
  server-confirmed part object, and the final composed object before creating
  the immutable attachment version. Expired sessions are retained as
  `EXPIRED` and a new epoch is opened for the same logical attachment; no
  object or local byte is automatically deleted.
- Focused failing tests were written first. Resumable part planning and the
  50%-cut retry path passed; attachment part-layout Go tests, web typecheck,
  API attachment/http/migration tests, builds, generated contract checks, and
  `git diff --check` passed. Result: `verified locally`, `candidate-only`.
- The full Web Vitest suite completed `712/719` tests: seven unrelated
  manager/planning/router UI assertions failed against the already-dirty
  working tree; they are not counted as P0-8 evidence and were not weakened.
- Connected PostgreSQL/object-store split-brain, physical process restart,
  storage-pressure, and sudden-power-loss evidence are `not run`; the P0-8
  production acceptance remains `release pending` and production offline-safe
  remains `blocked`.

### P0-9 — Scan and canonical finalization

#### Finalization prerequisites

- Every accepted operation is terminal.
- No rejected, conflict, poison-dependent, or quarantined operation remains.
- Exact answer revisions/digests match.
- Exact Potential Finding revisions/digests match.
- Exact attachment object versions, sizes, and hashes match.
- Every attachment scan state is `CLEAN`.
- `PENDING`, `FAILED`, or `QUARANTINED` attachment state rejects finalization.
- The server independently computes canonical state from PostgreSQL and
  canonical object metadata. A client manifest is only an assertion.

#### Canonical receipt

```text
inspection_id
package_revision
server_revision
answer_manifest_hash
finding_manifest_hash
attachment_manifest_hash
event_manifest_hash
receipt_id
server_timestamp
```

- Use `avia-finalization-manifest/v1` canonicalization.
- An empty outbox is never finalization evidence.
- Finalization is idempotent; duplicate or lost-ACK retries return one receipt.
- UI displays `FINALIZED` only after local receipt persistence and readback.
- If local receipt persistence fails after server finalization, use
  `FINALIZED_SERVER_RECEIPT_RECOVERY_REQUIRED`; retry retrieves and persists the
  same receipt rather than finalizing again.
- Reopen/correction creates new auditable revisions/events and never overwrites
  the previous receipt.

#### P0-9 execution record — 2026-08-17

- Added the OpenAPI source contract for `finalizeInspection` and the closed
  `InspectionFinalizationReceipt` shape using
  `avia-finalization-manifest/v1`; generated TypeScript/Go transports were
  regenerated afterward. The focused contract checks passed `6/6` and the
  repository contract gate passed `16/16`.
- Added forward-only migration `000053_inspection_finalization_receipts`.
  The receipt is immutable per inspection and stores package/server revision,
  answer/finding/attachment/event manifest hashes, canonicalization version,
  and server timestamp. `LatestVersion` is now `53`; no live migration was
  applied.
- Added a server finalization service that derives manifests from PostgreSQL,
  requires submitted checklist state, rejects every non-`CLEAN` or
  non-`CANONICAL` attachment, persists one receipt and one
  `inspection.finalized` audit event, and replays the same receipt for
  duplicate/lost-ACK operation delivery. Client state is not trusted as
  finalization authority.
- Focused failing canonical-hash tests, Go `internal/...` tests, Web
  typecheck, generated contract checks, and `git diff --check` passed. Result:
  `verified locally`, `candidate-only`.
- Connected PostgreSQL finalization, scan-worker CLEAN transition, receipt
  readback after process restart, physical browser/device qualification, and
  owner approval are `not run`/`blocked`; production offline-safe remains
  `release pending` and `blocked`.

#### Acceptance

- Missing/mismatched entity, non-CLEAN scan, conflict, or quarantine returns one
  precise rejection code.
- One final transition, one canonical receipt, and one authoritative audit
  event exist.
- Support can recompute the receipt without trusting client state.

### P1 — Audit, diagnostics, origin, and rollback

#### Audit

- Use a server-authoritative append-only event chain for the Gate 0 catalog.
- Each event records the previous server event hash, canonical event hash,
  authenticated actor, server revision, and server timestamp.
- Client hash chains are supplementary evidence only, never authority.
- Duplicate operation delivery creates no second domain event.

#### Diagnostics

The user-consented support bundle contains only:

- app, Service Worker, DB, and protocol versions;
- browser/runtime lane and capability fingerprint;
- origin ID and storage persistence/quota state;
- package/outbox/attachment state counts;
- last sync/result and last receipt ID;
- migration/update/quarantine state; and
- last integrity result plus bounded error codes.

It excludes tokens, cookies, credentials, PII, answer/note text, filenames,
raw paths, attachment bytes, secret-bearing URLs, and free-form server errors.
Bundle generation works offline and does not mutate recovery state.

#### Origin and rollback

- Freeze one exact production origin as a permanent storage boundary.
- Any scheme/host/port change requires a separate origin-migration ExecPlan.
- Redirects do not migrate IndexedDB, OPFS, Cache Storage, or Service Worker
  state.
- Rollback never downgrades IndexedDB or reloads active local work.
- Retain active and verified predecessor app-shell artifacts.
- Verify release provenance using source revision, dependency lock, build
  invocation, artifact SHA-256, release signature, and artifact readback.
- Cache GC targets only verified app-shell caches.

#### Acceptance

- Audit catalog, chain, retention, and access are owner-approved.
- Redaction tests prove forbidden diagnostic content is absent.
- Origin mismatch fails closed.
- Mixed build, missing asset, bad hash, rollback, and no-downgrade tests pass.

#### P1 execution record — 2026-08-17

- Added privacy-safe support diagnostics projection. It emits only versions,
  browser lane/capability, origin ID, storage counts, bounded sync/recovery
  codes, and receipt ID; tokens, cookies, credentials, PII, answer/note text,
  filenames, raw paths, bytes, signed URLs, and free-form server errors are
  excluded by projection and redaction tests (`2/2`). It does not mutate local
  recovery state.
- Added exact scheme/host/port origin comparison that fails closed on an
  unconfigured, invalid, alias, or redirect origin. No origin/DNS/gateway
  value was changed; the exact production origin remains unavailable.
- Added deterministic `scripts/test-offline-failure-injection.sh` and
  `scripts/record-offline-release-qualification.mjs` entrypoints. They run
  local candidate tests or print a bounded `not run`/`candidate-only` release
  record and perform no external writes, secrets, deployment, or production
  qualification.
- Added forward-only audit-chain columns and a canonical previous-event-bound
  hash utility. Existing historical events remain nullable until an approved
  connected backfill/chain qualification; no automatic rewrite or deletion
  was performed. Audit catalog, retention, access, and named owners remain
  owner-gated.
- Focused P1 tests, Go audit/migration tests, candidate failure-injection
  (`83/83` Web tests plus Go slices), typecheck, builds, contract checks, and
  `git diff --check` passed. Result: `verified locally`, `candidate-only`.
- Full Web suite remains `712/719`; seven manager/planning/router UI
  assertions fail in unrelated dirty-tree surfaces. Connected audit-chain,
  diagnostics/support acceptance, exact-origin release evidence, physical
  browser/device qualification, and owner approvals are `not run`/`blocked`.

## Demo gate

Only these fast checks may gate the demo:

```bash
npm --prefix apps/web run typecheck
npm --prefix apps/web run build:demo
npm --prefix apps/web run test:e2e:mock -- canonical-scenario.spec.ts
npm --prefix apps/web test -- \
  src/offline/storage-readiness.test.ts \
  src/offline/update-coordinator.test.ts \
  src/features/inspections/offline-readiness-panel.test.tsx \
  tests/offline/field-repository.test.ts \
  tests/offline/opfs-inspection-attachment-store.test.ts \
  tests/offline/sync-engine.test.ts
node apps/web/scripts/assert-app-shell-artifact.mjs apps/web/dist/demo
node tests/demo-boundary-smoke.test.js
```

Optional desktop Chrome candidate smoke, only when needed:

```bash
npm --prefix apps/web run test:e2e:offline -- tests/e2e/offline-startup.spec.ts
```

Expected result:

```text
candidate-only
```

The demo does not require physical devices, browser/process kill, OS restart,
power loss, storage pressure, multi-day offline evidence, production scan,
production object storage, or production release evidence.

## Production verification matrix

### Per-source-slice candidate gate

```bash
npm --prefix apps/web run contracts:check
npm --prefix apps/web run typecheck
npm --prefix apps/web test
npm --prefix apps/web run build:demo
npm --prefix apps/web run build:http
npm --prefix apps/web run check:app-shell
go -C apps/api test -count=1 ./internal/sync ./internal/inspections/attachments ./internal/worker/evidence
node --test api/openapi/tests/contract-examples.test.mjs
npm --prefix apps/web run test:e2e:http -- offline-sync.http.spec.ts
bash scripts/test-cache-update-harness.sh
git diff --check
```

These are local `candidate-only` checks, not production evidence.

P0-9 must add and register deterministic entrypoints with these exact intended
interfaces before use:

```bash
bash scripts/test-offline-failure-injection.sh
node scripts/record-offline-release-qualification.mjs --lane <lane> --artifact <digest>
```

### Final release browser/device gate

Physical device, browser/process kill, OS restart, and storage-pressure tests
run only at the final production release gate. The same gate includes an
approved sudden-power-loss class protocol on disposable qualification hardware
or an owner-approved equivalent power-interruption harness. It is never a demo
gate.

| Lane | Required final evidence |
| --- | --- |
| iOS/iPadOS Safari | iPhone and iPad; qualified WebKit lane; IDB/OPFS kill, quota/storage pressure, safe update, migration, multi-day reconnect, resumable upload, scan, finalization, and approved sudden-power-loss class evidence. |
| iOS/iPadOS Chrome | Same WebKit/capability suite and separate browser-brand admission result; no Chrome-specific implementation. |
| macOS Safari | Process kill, multi-tab, migration, storage pressure, safe update/rollback, upload, scan, finalization, and approved sudden-power-loss class evidence. |
| macOS Chrome | Stable N/N-1; the same domain invariants plus observed Chromium durability and approved sudden-power-loss class evidence. |
| Android Chrome | Stable N/N-1; process/OS kill, low disk, network switch, upload session expiry, scan, finalization, and approved sudden-power-loss class evidence. |
| Windows Chrome | Stable N/N-1; process/OS restart, low disk, safe update/rollback, upload, scan, finalization, and approved sudden-power-loss class evidence. |

Every evidence record includes exact device model, OS/browser build, normal-tab
runtime, artifact digest, precondition, injection, expected state, recovery,
PASS observation, and evidence label.

## Failure-injection gate

Every test record uses:

```text
Precondition
Injection
Expected state
Recovery
PASS criteria
Evidence label
```

Current evidence for every row is `not run`.

| # | Precondition | Injection | Expected state | Recovery | PASS criteria | Evidence label |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | Package download active. | Cut network after an arbitrary item. | `DOWNLOADING` or `RECOVERY_REQUIRED`; never Ready. | Resume from manifest or restart the download. | Partial package is not editable and final digest is exact. | Current `not run`; automation becomes `candidate-only`. |
| 2 | Server has acknowledged 50% of attachment parts. | Cut network. | `UPLOADING`; session, offsets, and part receipts remain durable. | Query server status and send only missing parts. | One canonical attachment version. | Current `not run`; physical lane evidence required at final gate. |
| 3 | Entity+outbox+sequence+receipt transaction active. | Kill browser/process at each fault point. | Complete-old or complete-new transaction. | Startup reconciliation. | No partial record set; every acknowledged mutation survives. | Current `not run`; physical lane evidence required. |
| 4 | OPFS temp write active. | Kill before/after flush, hash, promotion, and metadata commit. | Deterministic matrix state. | Rehash/reconcile or quarantine. | No false Ready and no silent deletion. | Current `not run`; physical lane evidence required. |
| 5 | PostgreSQL committed operation X. | Drop ACK/response. | Client retains retryable X. | Retry the same operation ID/hash. | One effect/event and the same response. | Current `not run`; local integration becomes `candidate-only`. |
| 6 | Valid operation X exists. | Send X twice concurrently. | Idempotency serializes and replays. | None beyond receipt replay. | One domain effect. | Current `not run`; `candidate-only` after local pass. |
| 7 | Operation ID X is already terminal. | Reuse X with different payload/request hash. | Terminal `OPERATION_ID_REUSE`. | Quarantine only causal dependents. | No domain mutation. | Current `not run`; `candidate-only` after local pass. |
| 8 | Outbox middle row is `IN_FLIGHT`. | Close and reopen browser. | Row becomes retryable with dependencies intact. | Foreground sync. | Operation is neither lost nor duplicated. | Current `not run`; physical lane evidence required. |
| 9 | Push/upload/scan is active. | Restart API or worker. | Bounded retry and server status reconciliation. | Resume with stable IDs/session. | No duplicate effect or object version. | Current `not run`; local integration becomes `candidate-only`. |
| 10 | Inspection, dirty form, IDB, OPFS, or sync is active. | Deploy candidate release. | `WAITING_FOR_SAFE_CHECKPOINT`. | Acknowledge/quarantine durable work, then user-controlled reload. | No takeover, navigation, or forced reload. | Current `not run`; physical lane evidence required. |
| 11 | Old Service Worker/app shell is active. | Connect to new backend. | Supported exact compatibility window or explicit read-only `UPDATE_REQUIRED`. | Safe update/reconciliation. | No generic failure or data deletion. | Current `not run`; every official lane required. |
| 12 | IndexedDB migration active or old tab blocks it. | Kill process at each migration phase. | Complete-old, complete-new, or read-only recovery. | Resume deterministic migration. | No clear, downgrade, or partial editable schema. | Current `not run`; physical lane evidence required. |
| 13 | Storage is near/full. | Exhaust quota before entity, package, cache, and attachment writes. | Attempt aborts with storage-full code; earlier work remains. | User/support disposition; no automatic work deletion. | Prior data is unchanged. | Current `not run`; every official lane required. |
| 14 | Local attachment is verified. | Corrupt one byte. | `QUARANTINED`. | Preserve source bytes; reacquire or review disposition. | No upload or finalization. | Current `not run`; local pass becomes `candidate-only`. |
| 15 | Valid lease exists. | Move clock forward/back across lease boundary. | Clock anomaly; new authoritative edits locked. | Reconnect to server time. | Client clock never extends authority. | Current `not run`; physical lane evidence required. |
| 16 | Same inspection is checked out on two qualified profiles. | Edit the same answer. | Typed revision conflict. | Explicit idempotent resolution operation. | No last-write-wins. | Current `not run`; integration and device evidence required. |
| 17 | User works offline. | Disable user, change role/assignment, or revoke package. | Local copy remains; reconnect rejects/quarantines. | Authorized review/recovery. | No unauthorized apply and no local deletion. | Current `not run`; identity integration evidence required. |
| 18 | Local checkout exists. | Clear browser site data. | `LOCAL_DATA_CLEARED`. | Online server re-checkout/support. | UI states the irrecoverable boundary; no false recovery claim. | Current `not run`; explicitly outside the zero-silent-loss guarantee. |
| 19 | Partial upload session exists. | Let session expire. | `EXPIRED`; old-session tombstone retained. | New epoch for same logical attachment/hash. | No second canonical attachment version. | Current `not run`; local pass becomes `candidate-only`. |
| 20 | Attachment scan is `PENDING`. | Request finalization. | Exact scan-pending rejection. | Wait for authoritative CLEAN projection. | No receipt/final state. | Current `not run`; integration evidence required. |
| 21 | Server finalized and returned receipt. | Fail local receipt persistence. | `FINALIZED_SERVER_RECEIPT_RECOVERY_REQUIRED`. | Retrieve, persist, and read back the same receipt. | No second finalize and no premature local success. | Current `not run`; browser evidence required. |
| 22 | Acknowledged package/work exists. | Restart browser or device. | Integrity pass or explicit recovery state. | Startup reconciliation. | Acknowledged mutations are readable or have explicit recovery. | Current `not run`; physical lane evidence required. |
| 23 | Disposable qualification device has acknowledged IDB mutations and OPFS attachments with exact receipts/hashes. | Apply the approved sudden-power-loss class interruption at each durability boundary. | Every acknowledged mutation/attachment survives exactly, or the lane fails production admission. | Restart, run integrity reconciliation, and preserve any mismatched bytes/records for diagnosis. | No acknowledged mutation is silently absent; no partial entity/outbox/receipt set; attachment size/hash is exact. | Current `not run`; final physical release evidence only. |

## Acceptance criteria and final Go/No-Go

### Demo GO

- Online/mock UI works.
- Typecheck and build pass.
- Mock E2E and selected fast offline unit/component tests pass.
- Production offline gates do not block the demo.
- Installed PWA is not used.
- The demo is labeled `candidate-only`.

### Production NO-GO

Production remains offline-safe NO-GO if any item is missing:

- Safe Service Worker update and bounded unresponsive-client path.
- IndexedDB atomicity, pull-page/cursor atomicity, and migration recovery.
- Deterministic OPFS recovery.
- Resumable attachment upload and split-brain recovery.
- Idempotent outbox, poison-dependency handling, and explicit conflict
  resolution.
- Multi-day lease independent of OIDC session lifetime.
- Non-exportable profile proof-of-possession authority.
- Device/profile-bound pull.
- Scan-aware canonical finalization receipt.
- Server-authoritative audit chain and owner-approved retention.
- Privacy-safe diagnostics and support acceptance.
- Exact origin freeze and no-downgrade rollback.
- Release artifact provenance.
- Current physical qualification on every official browser/device lane.
- Security, Records, Product, Support, and Release owner acceptance.

## Risks, dependencies, idempotence, and recovery

- Browser persistence cannot survive explicit site-data clearing or profile/
  device/disk loss; support language must state that boundary.
- IndexedDB and OPFS cannot share one atomic transaction. Manifest-first
  staging, verified promotion, IDB commit markers, reconciliation, and
  quarantine are the recovery protocol.
- At-least-once transport plus server idempotency supplies exactly-once domain
  effects. Do not claim exactly-once transport.
- Client clocks, random device IDs, and client hash chains are metadata, not
  authority.
- Protocol/schema migration is forward-only. Rollback uses compatible code or
  read-only recovery and never downgrades IndexedDB.
- Unknown bytes, rejected operations, and quarantined records remain until an
  approved disposition exists.
- No source-write streams may overlap in the same working tree. Contract review
  and independent read-only verification may run separately.

## Current progress, decisions, discoveries, and outcome

### Progress

- [x] Official browser allowlist and unsupported matrix recorded.
- [x] Demo and production lanes separated in the plan.
- [x] Installed runtime removed as a feature and future phase.
- [x] Mandatory Gate 0, P0-1 through P0-9, and P1 order recorded.
- [x] Required state machines, reconciliation matrix, browser matrix, failure
  injection table, and Go/No-Go criteria recorded.
- [x] Two-pass independent read-only plan review completed; no Critical
  findings. Four Important and two Minor findings were incorporated.
- [x] Documentation verification is `verified locally`: `git diff --check`,
  `node tests/harness-docs-smoke.test.js`,
  `node tests/demo-boundary-smoke.test.js`, and `make harness-check` passed.
- [x] Gate 0 baseline, version vector, policy dispositions, audit catalog,
  origin blocker, durability policy, security-fence order, and evidence-owner
  gaps recorded before source edits.
- [x] P0-1 demo/production separation is `verified locally` with distinct
  artifact/lane contracts and no installed runtime.
- [x] P0-2 official browser/version/capability admission, runtime canary, and
  explicit local-data-cleared classification are `verified locally`.
- [x] P0-3 IndexedDB atomic receipts, strict durability request, lifecycle
  states, pull-page transaction preservation, and forward-only legacy
  reconciliation are `verified locally`.
- [x] P0-4 OPFS manifest-first recovery, deterministic reconciliation, final
  readback, and no-delete quarantine are `verified locally`.
- [x] P0-5 Service Worker safe-update, quiescence ACK, unresponsive-client
  fence, and no-forced-navigation behavior are `verified locally`.
- [x] P0-6 operation envelope, idempotency/reuse handling, profile-bound pull
  cursor, poison-dependent quarantine, and typed conflict resolution are
  `verified locally` as candidate source.
- [x] P0-7 non-exportable profile authority, seven-day server-time lease cap,
  proof-bound operation request hash, profile-bound pull, session-independent
  grant authorization, and key-loss preservation are `verified locally` as
  candidate source.
- [x] P0-8 resumable session/epoch state, server-confirmed part receipts,
  ordered completion, client retry of only missing parts, and no-delete
  recovery path are `verified locally` as a candidate source slice.
- [ ] P0-8 connected object-store/PostgreSQL split-brain and process-restart
  qualification — `not run`.
- [x] P0-9 CLEAN-gated server canonical manifests, idempotent finalization
  receipt, exact revision/hash fields, and finalization audit event are
  `verified locally` as a candidate source slice.
- [x] P1 candidate diagnostics redaction, exact-origin fail-closed helper,
  deterministic qualification entrypoints, and audit-chain field/hash slice
  are `verified locally`.
- [ ] Gate 0 owner decisions — `blocked` for production: exact production
  origin, named Product/Security/Records/Support/Release/Device Lab owners,
  retention approvals, and timeout/deadline/classifier values are unavailable.
- [x] Candidate source implementation through P1 — `verified locally`;
  production qualification remains `candidate-only` and `release pending`.
- [ ] Production physical-device evidence — `not run`.
- [ ] Deployment/release/production actions — `not run` and unauthorized.

### Decisions

- Demo can proceed independently and remains `candidate-only`.
- Production scope is normal browser tabs in exactly six browser families.
- Official allowlist precedes capability admission.
- iOS/iPadOS Safari and Chrome share one WebKit implementation lane.
- Device/profile authority uses a non-exportable proof-of-possession key;
  random device ID is not authority.
- Finalization is a server-computed canonical receipt; empty outbox is not
  completion.
- Real device/process/OS-kill evidence runs only at the final production
  release gate.

### Outcome

```text
Demo can proceed quickly.
Production offline-safe is blocked.
Installed PWA is outside this plan.
Authorized application, contract, generated transport, migration, test, and
documentation source changes were implemented through the candidate P1 slice.
Production evidence is not run.
```

## Execution Prompt

```text
Execute the active AviaSurveil plan at:
docs/exec-plans/active/2026-08-17-offline-first-browser-tab-production-hardening-plan.md

Read apps/surveil/AGENTS.md, docs/PLANS.md, this plan, the plan index, the
verification/output contracts, and the active app-shell convergence plan
completely before acting. Consult the superseded 2026-07-20 plan only as
historical evidence or a source-location map; none of its scope, policy,
acceptance, or release directions are current authority.

Preserve the current branch and all unrelated dirty changes. Do not commit,
push, deploy, change origin/DNS, write secrets, modify cloud infrastructure, or
perform external/GitHub writes without separate exact authority.

Start with Gate 0 only. Record the dirty-tree/version/migration baseline, exact
policy values, named evidence owners, production origin, and app-shell source
handoff. Missing production decisions do not block the online/mock demo.

Then execute P0-1 through P0-9 and P1 in order. Use OpenAPI as transport
authority, forward-only migrations, stable operation/session identities, and
no-delete recovery. Keep all local automation candidate-only. Run physical
device/process/OS-kill/storage-pressure qualification only at the final release
gate. Never use demo success as production offline evidence.
```
