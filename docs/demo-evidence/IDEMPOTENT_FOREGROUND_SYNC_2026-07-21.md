# Idempotent Foreground Sync Evidence — 2026-07-21

## Outcome

- Evidence status: `verified locally`
- Artifact status: `candidate-only`
- Release status: `release pending`
- Task 5 wider `first-production` route migration and Task 13 local release-candidate packet: `not run`
- Production deployment, traffic cutover, legacy removal, production MDM/security operations, owner-approved local attachment disposition, and a `production-ready` claim: `blocked`

Task 12 connects the field-only React working set to the real Go HTTP profile through typed one-operation push/pull sync. The root Vanilla JavaScript demo remains intact as the behavioral reference. This task adds no automatic conflict merge, background-only delivery, destructive local-byte cleanup, production deployment, or production-readiness claim.

## Test-First Evidence

The first focused runs failed because the server operation service, client foreground engine, reconciliation methods, and HTTP offline-sync scenario did not exist. Later red/green cycles exposed and corrected:

- a later never-sent response edit and an explicit conflict re-entry whose causal dependants still referenced the superseded operation;
- attachment staging attempted only after the browser was disconnected, when its dedicated hash-worker module could no longer be fetched;
- a deliberate lost-ack route abort being counted as an unexpected browser console error instead of the expected transport failure; and
- `go run` leaving a child API binary alive after the test-profile wrapper exited.

The final tests cover duplicate delivery, same-ID/same-semantic replay with a changed client timestamp, different-payload ID reuse, lost acknowledgement, stale revision, repeated edits, an edit while the prior operation is in flight, causal Potential Finding and attachment delivery, actor/device/time trust boundaries, expired and revoked grants, changed assignment, withdrawn package, invalid domain input, cursor replay/scope/restart/history expiry, tombstones, package revocation, forbidden-field scans, attachment retry, two-tab contention, and the absence of Background Sync registration.

## Verified Server Boundary

- The endpoint decodes one closed-union `FieldSyncOperation`, rejects unknown fields, and derives subject, organization, role, and session from the authenticated principal rather than client payload fields.
- Current offline grant, session, device, package ID/version/digest, assignment revision, question scope, and allowed command type are checked before mutation. Expired/revoked grants return terminal typed errors; changed assignment and withdrawn package return authorized typed conflicts.
- The semantic idempotency hash excludes `clientOccurredAt`. The same operation ID and semantic payload replay the exact stored acknowledgement; reusing an ID with a different semantic payload fails.
- Each accepted command stores the domain mutation, full idempotency response, audit event, authorized sync change, and server outbox message in one PostgreSQL transaction. Retryable infrastructure failures are not stored as applied commands.
- Checklist responses use expected revisions. Potential Finding creation verifies the acknowledged response revision. Attachment registration resolves the authoritative Potential Finding through its causal operation ID. Checklist submission remains server-shaped and requires assigned field work.
- Pull cursors are opaque database tokens scoped to subject, organization, package, grant, device, projection version, and high-water mark. Only closed authorized projections, tombstones, and revocations are returned; internal CAA fields are structurally absent. Cursor scope mismatch fails, cursor replay is stable, a new service instance resumes from PostgreSQL state, and expired history returns `resnapshotRequired` without unsafe rows.

## Verified Client Reconciliation

- `FieldRepository` remains the only field mutation boundary. React components do not write IndexedDB, OPFS, mock seed state, or HTTP endpoints directly.
- Never-sent response drafts may coalesce. In-flight payloads are frozen; a later edit becomes a distinct operation and waits for the authoritative revision. Acknowledgement updates authoritative identity/revision without overwriting that later local draft.
- Potential Finding creation waits for its response acknowledgement. Attachment registration waits for its response and Potential Finding operations, then records authoritative IDs before bounded `beginUpload -> PUT -> completeUpload` delivery.
- Expired upload URLs and transport failures return to retryable state. Local attachment bytes remain present after acknowledgement or retry; no sync or recovery path automatically deletes them.
- One foreground package owner is selected with Web Locks; BroadcastChannel publishes status to other tabs. Server idempotency remains the duplicate safeguard.
- Startup, foreground visibility, `online`, manual Sync now, and app-open/page-show triggers run the same engine. No Service Worker Background Sync registration is made or required for success.
- Only retryable transport/infrastructure results are automatically released. `conflict`, `forbidden`, and `invalid` results remain terminal until explicit user action. The UI preserves the local draft, renders an authorized conflict summary, and requires re-entry against the authoritative revision. `resnapshotRequired` locks editing without overwriting pending operations or attachment manifests.

## Real HTTP Scenario

The HTTP Playwright scenario checks out the canonical Cabin package, saves a Non-Compliant response, creates a local Potential Finding, stages a PDF, then makes a later offline edit. It deliberately commits the first sync operation on the API and aborts the response to emulate a lost acknowledgement. A manual foreground retry replays the stored acknowledgement, causally creates authoritative `PF-2026-001`, registers and uploads the attachment, preserves local bytes, and drains the pending work. A subsequent offline Observation edit is delivered by the `online` foreground trigger and reaches authoritative revision 2.

The expected browser `net::ERR_FAILED` record from the deliberate route abort is asserted explicitly. All other warning/error console records remain empty.

## Fresh Verification

The following completed with exit code `0`:

- `./scripts/test-http-profile.sh`: API/worker direct temporary binaries; full Go `-race` package and live PostgreSQL/OIDC-provider/MinIO integration suite; OpenAPI 5/5; SQLC clean generation; TypeScript; React/Vitest 16 files and 143/143 tests; demo and HTTP builds; HTTP artifact scan across 12 files and 86 inputs; live HTTP Backend contract 9/9; mock Playwright 1/1; HTTP Playwright 2/2; container/network/volume cleanup
- Focused sync, field-repository, and attachment suites: 66/66 tests
- `go vet ./...`
- Root Vanilla JavaScript smoke suite: 103/103
- `git diff --check`
- Post-run checks: no Task-owned container, API/worker listener, Playwright, or test Chrome process remained

The HTTP artifact remains free of mock, seed, and test-profile inputs. The deterministic scanner, then-current local OIDC provider, PostgreSQL, and MinIO-compatible object store are test infrastructure only.

## Remaining Boundary

Task 5 route migration and Task 13 final release-candidate packet are `not run`. The known development-tool transitive audit finding remains `note-open` for Task 13. Production identity/MDM, secrets, hosted providers and region, monitoring/on-call, backup/restore/DR acceptance, retention/legal hold, owner-approved attachment cache disposition, deployment, traffic cutover, and legacy removal remain `blocked` behind a separately approved release/operations plan. This evidence supports only a locally verified `candidate-only` Task 12 result.
