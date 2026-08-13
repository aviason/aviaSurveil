# OPFS Inspection Attachment Recovery Evidence — 2026-07-21

## Outcome

- Evidence status: `verified locally`
- Artifact status: `candidate-only`
- Release status: `release pending`
- Production deployment, traffic cutover, legacy removal, production MDM/local-data controls, owner-approved cache disposition, and a `production-ready` claim: `blocked`
- Task 12 network sync/conflict delivery, Task 5 wider route migration, and Task 13 final local release-candidate gate: `not run`

Task 8 adds manifest-first OPFS staging and startup recovery for field `InspectionAttachment` bytes. The intact root Vanilla JavaScript demo remains the behavioral reference. This task neither creates nor overwrites an official Auditee `EvidenceVersion`, and it does not deliver attachment registration or bytes over the network; Task 12 owns that sync path.

## Test-First Evidence

The first focused run failed because the attachment store, hash worker, and recovery modules did not exist. Subsequent red/green cycles exposed and corrected:

- upload tests that attempted to bypass the causal checklist-response dependency;
- an already-ready manifest whose verified temporary bytes were not restored to the final OPFS path; and
- a recovery module that was implemented but not yet invoked by the React field startup/load path.

The final suite injects termination before or after manifest creation, source hashing, temporary-path creation, every chunk write, flush, stored-byte hashing, final promotion, atomic metadata/outbox readiness, upload start, and acknowledgement.

## Verified Staging And Recovery Boundary

- A subject-scoped IndexedDB manifest is committed in `manifest_created` before an OPFS path is created. Source bytes are hashed in a dedicated module Worker, then written in bounded chunks to a temporary path, flushed, read back, size/hash verified, promoted, and atomically committed as `ready` with one typed `REGISTER_INSPECTION_ATTACHMENT` outbox operation.
- The local lifecycle is `manifest_created -> writing -> ready -> uploading -> acknowledged`; `purge_eligible` exists in the model but cannot be entered because no owner-approved cache/disposition policy exists. Explicit purge remains disabled.
- PDF, JPEG, and PNG are allowed up to 25 MB. Empty/oversized files, unsafe filenames, duplicate active filenames, hash/size mismatch, wrong assignment, wrong package/response/Potential Finding scope, and missing grant command authority fail closed.
- Registration retains causal dependencies on the exact checklist response and, when present, the Potential Finding creation operation. Upload cannot begin while registration is dependency-blocked. Metadata/outbox, upload/in-flight, and acknowledgement transitions commit atomically.
- React field mode stages bytes only through `InspectionAttachmentStore`/`FieldRepository`, lists the filename and staging state, and retains the visible `Saved locally — sync pending` state. No component writes IndexedDB, OPFS, mock seed state, or a remote API directly.
- Startup/load reconciliation compares subject OPFS paths and manifests. Missing referenced bytes become a visible blocking recovery error and field editing is disabled. Verified temporary bytes are promoted; incomplete, mismatched, or unknown bytes receive quarantine metadata and remain present.
- No recovery path automatically deletes referenced, pending, unknown, or quarantined bytes. Acknowledged local bytes also remain present. Explicit browser site-data clearing is stated literally as an irrecoverable boundary for an unsynced sole copy.
- Official Auditee Evidence remains separate and immutable. Task 8 adds no Evidence mutation, review, closure, retention, or deletion behavior.

## Restart Evidence

A dedicated persistent-Chrome test uses an owner-policy-attested checkout, restarts the browser to satisfy the storage canary, loads the checklist, then stops the origin server. It saves the checklist response and stages a PDF while offline. After closing and reopening Chrome with the same profile while the origin remains stopped:

- the checklist shell and worker asset load from the app-shell cache;
- the attachment remains `ready` in IndexedDB;
- its exact OPFS bytes, byte count, and SHA-256 still match;
- the registration outbox remains `BLOCKED_ON_DEPENDENCY` on the response operation; and
- no local byte is deleted.

The existing server-stopped startup, two-client N/N-1 update, and pending/in-flight field restart tests remain green.

## Fresh Verification

The following completed with exit code `0`:

- `npm --prefix apps/web run typecheck`
- `npm --prefix apps/web test`: 15 files, 132/132 tests
- focused attachment staging/recovery suite: 35/35 tests
- `npm --prefix apps/web run test:e2e:offline`: 4/4 real persistent-Chrome tests
- `./scripts/test-http-profile.sh`: full Go race/live PostgreSQL, then-current local OIDC provider, and MinIO integration; OpenAPI 5/5; SQLC clean generation; React 132/132; live HTTP Backend contract 9/9; mock Playwright 1/1; HTTP Playwright 1/1; both builds; HTTP artifact isolation across 12 files and 84 inputs; task-owned dependency cleanup
- `npm --prefix apps/web run check:app-shell`: demo and HTTP artifacts pass across 12 files and 4 assets each, including the hash worker
- Root Vanilla JavaScript smoke suite: 103/103
- `git diff --check`
- `npm --prefix apps/web audit --omit=dev`: 0 production dependency vulnerabilities

The existing full development-dependency audit finding remains `note-open`: two high-severity transitive development-tool findings in `js-yaml` through `@redocly/openapi-core`. Remediation is `not run` in Task 8 and remains assigned to the Task 13 security gate.

Task-owned HTTP containers, network, and volumes were removed by the profile. Browser/process cleanup was checked separately before commit. This evidence supports only the local Task 8 candidate. Task 12 typed network sync is the next binding slice; production release and cutover remain `blocked`.
