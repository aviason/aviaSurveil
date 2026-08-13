# Bounded Upload And HTTP Parity Evidence — 2026-07-21

## Outcome

- Evidence status: `verified locally`
- Artifact status: `candidate-only`
- Release status: `release pending`
- Production object storage, malware scanning, OIDC/MFA, deployment, traffic
  cutover, legacy removal, and a `production-ready` claim: `blocked`
- Browser offline foundation (Tasks 6-8), production sync (Task 12), the wider
  role-entry migration (Task 5), final release-candidate packet (Task 13), and
  production operations: `not run`

Task 11 implements and verifies bounded attachment/Evidence bytes, deterministic
local scanning, and the first real HTTP canonical scenario. It uses pinned local
PostgreSQL, the then-current local OIDC provider, and S3-compatible object storage. The deterministic scan
adapter and local object-storage profile are test infrastructure, not a
production malware scanner or production records repository. The root Vanilla
JavaScript demo remains intact as the behavioral reference.

## Test-First Evidence

Upload, storage, worker, HTTP contract, and shared browser tests were added
against absent behavior before implementation. The red/green cycle caught and
corrected:

- filename validation that incorrectly rejected valid names containing `0` or
  `x`;
- accidental inclusion of the test-only HTTP entry in the normal HTTP build;
- mock/server differences in Finding next action and post-scan review state;
- public HTTP errors that leaked internal sentinel prefixes;
- an expected validation denial that generated browser console noise;
- worker terminal failure/timeout paths that were not yet visible and
  non-reviewable;
- upload expiry/retry and clean/failed Inspection Attachment separation that
  needed explicit regression coverage; and
- Go build temporary/cache behavior that could exhaust a constrained local
  disk. The profile now contains task-owned `GOTMPDIR`, reuses content-addressed
  cache entries through hard links, forces fresh test execution with
  `-count=1`, and removes only task-owned links and temporary output.

## Verified Upload And Worker Boundaries

- Authorized upload sessions are short-lived, idempotent, bounded to 25 MB,
  restricted to PDF/JPEG/PNG, and issued with unique non-overwriting quarantine
  keys.
- Completion verifies the server-observed object size and SHA-256 digest, the
  declared extension/media type, and server-side MIME sniffing before a record
  can advance.
- Expired incomplete sessions remain non-reviewable. A retry receives a fresh
  key and cannot overwrite the earlier staged object.
- Official Auditee submission creates a new immutable Evidence version with
  separate upload, scan, and review states. Earlier versions are preserved.
- Inspection Attachments remain Audit/question/response/package/grant scoped.
  A clean or failed attachment never becomes an official Evidence version
  implicitly.
- Objects are private. Download instructions and Evidence review are available
  only for the exact scan-clean version; pending, quarantined, failed, or
  superseded versions cannot support review or closure.
- The worker claims outbox work with a lease and idempotency key. Clean scans
  promote the exact object/version and set review to `PENDING_CAA_REVIEW`.
  Quarantine, scanner failure, and scanner timeout remain non-reviewable and
  expose deterministic operator-visible state.
- Crash-after-copy/before-acknowledgement recovery does not duplicate an
  Evidence version or overwrite an object.
- Scan state, Finding transition, audit event, authorized change, object
  metadata, and terminal outbox state are committed consistently.
- Notification delivery, production PDF generation, retention deletion, legal
  disposition, production scanner/provider policy, and production hosting are
  not implemented or claimed.

## Verified Real HTTP Parity

- The Go API and worker expose the canonical Cabin Inspection scenario through
  the capability-composed HTTP Backend contract.
- The React HTTP build uploads the selected Evidence bytes through a signed PUT,
  completes the server record, waits for the deterministic worker, and reviews
  only the clean immutable version.
- The same Playwright scenario passes under both `mock` and `http` profiles,
  including Potential Finding authority, CAP submission/review separation,
  organization isolation, Internal CAA Note separation, Evidence versioning,
  closure, report, dashboard, and denial invariants.
- The production-shaped HTTP artifact scan passed across 7 files and 71 build
  inputs. It contains no mock/seed implementation, local test token/header, or
  test-profile source.

## Fresh Verification

The primary command was:

```bash
./scripts/test-http-profile.sh
```

It completed with exit code `0` and proved:

- API and worker production-command builds: pass
- Full fresh Go race suite and live integration tests: pass
- PostgreSQL migrations, then-current pinned local OIDC+PKCE, and private local
  object-storage integration: pass
- Evidence expiry/retry, hash/type/size enforcement, immutable versions,
  scanner clean/quarantine/failure/timeout, and crash recovery: pass
- Inspection Attachment scope and official-Evidence separation: pass
- OpenAPI examples and TypeScript/Go clean generation: 5/5 pass
- All module-owned SQLC clean generation: pass
- React/Vitest: 32/32 pass
- Demo and HTTP builds: pass
- HTTP artifact isolation: pass (7 files, 71 inputs)
- Live `HttpBackend` contract: 9/9 pass
- Shared Playwright scenario: mock 1/1 and HTTP 1/1 pass
- Task-owned API/worker/browser processes, containers, networks, volumes,
  cache links, and temporary directories cleanup: pass

Additional fresh gates:

- `go vet ./...`: pass
- Root Vanilla JavaScript smoke suite: 103/103 pass
- `git diff --check`: pass

This evidence supports only the local Task 11 candidate. Tasks 6-8 are
subsequently `verified locally`; Task 12 typed network sync is the next binding
slice. Production release and cutover remain outside this authorization and
`blocked`.
