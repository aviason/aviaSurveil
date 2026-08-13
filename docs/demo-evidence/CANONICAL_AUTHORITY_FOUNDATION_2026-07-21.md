# Canonical Authority Foundation Evidence — 2026-07-21

## Outcome

- Evidence status: `verified locally`
- Artifact status: `candidate-only`
- Release status: `release pending`
- Production OIDC/MFA, deployment, traffic cutover, and legacy removal: `blocked`
- Object upload/storage, malware scanning, real HTTP scenario parity, IndexedDB, OPFS, PWA offline behavior, sync push/pull, and production operations: `not run`

Task 10 implements and verifies the server-side authority foundation. It does
not claim Task 11 attachment/Evidence bytes, Task 12 sync, Task 13 release
verification, or production behavior. The root Vanilla JavaScript demo remains
intact as the behavioral reference.

## Test-First Evidence

Domain state-machine, authorization, raw-projection, idempotency, migration,
session, OIDC, and offline-grant tests were written against missing behavior
before their implementations. The integration profile then exposed two real
local-provider defects: the seeded external-provider user required a complete profile,
and an unsupported `offline_access` request consumed the authorization code.
The realm fixture and requested scopes were corrected, and the complete profile
was rerun successfully. A final review also found that a fixed browser-cookie
expiry would end an otherwise active rolling-idle session early. A regression
test failed first; browser-session cookies now leave idle and absolute expiry
under server authority, and the targeted test passed before the full rerun.

## Verified Authority Foundation

- Domain rules are separated into `identity`, `organizations`, `planning`,
  `inspections`, `checklists`, `potentialfindings`, `findings`, `caps`,
  `evidence`, `reports`, `sync`, and `auditlog` modules with checked,
  module-owned PostgreSQL stores.
- An Inspector can create only Audit/question/response-scoped Potential
  Findings. Lead authority, reason-required return/dismiss, severity-selected
  conversion, and atomic canonical Finding/public-number creation are enforced.
- CAP submission and CAA review are distinct exact-revision commands. CAP
  acceptance leaves the Finding open; rejected and more-information CAPs can be
  resubmitted as preserved revisions.
- Evidence review binds an exact scan-clean immutable Evidence version.
  Evidence-verified closure, partial/not-close/request-more-information results,
  and reason-required Department Manager authorized closure remain distinct.
- Submitted checklists are read-only until a permitted, stage-valid,
  reason-required online reopen. Checklist template and inspection package
  snapshots, CAP revisions, Evidence versions, review decisions, and report
  versions reject overwrite attempts.
- Report decisions bind exact versions; Department Manager and General Manager
  can return/forward only, while Executive Director issue locks the report and
  never closes a Finding.
- Every successful status transition atomically stores the mutation,
  server-computed semantic idempotency hash and full response, exactly one
  domain audit event, an authorized sync change, and an outbox message. A lost
  acknowledgement replays the original response; changed-payload operation-ID
  reuse fails without another mutation or transition event.
- Audit rows are database-enforced append-only and record actor, organization,
  entity/version, before/after state, reason, server time,
  operation/correlation ID, and closure basis where applicable. No
  tamper-evidence claim is made.
- Auditee list and direct-object access is organization-scoped. Closed raw JSON
  projections cover Findings, CAPs, Evidence, released reports, assignments,
  dashboard data, sync changes, and safe conflicts without Internal CAA Notes,
  other organizations, private workload/risk, unreleased reports, or
  enforcement deliberations.
- The same-origin BFF completes OIDC Authorization Code with PKCE, one-time
  state and nonce verification. Provider tokens remain server-side under
  AES-GCM; opaque browser and CSRF tokens are stored only as hashes. Browser
  sessions use Secure, HttpOnly, SameSite cookies, mutation CSRF checks,
  30-minute rolling idle expiry, eight-hour absolute expiry, and explicit
  revocation.
- The then-current pinned local OIDC profile completes a real Authorization Code + PKCE
  exchange with issuer, signature, audience, nonce, organization, and canonical
  role checks. This is local-provider evidence, not production OIDC/MFA
  evidence.
- Server-issued offline grants derive subject and organization from the active
  session and bind device, package version/digest, assignment revision,
  questions, and allowed command types. Expiry/skew, reassignment, package
  withdrawal, user switch, logout/session revoke, device-loss revoke, and late
  authorization are fail-closed.
- Forward migration `000003_authority_foundation` and the retained N-1 fixture
  pass against live PostgreSQL; all twelve module store outputs regenerate
  without drift.

## Fresh Verification

The primary command was:

```bash
./scripts/test-http-profile.sh
```

It completed with exit code `0` and proved:

- API and worker builds: pass
- Full Go race suite, including domain and live PostgreSQL integration tests: pass
- Empty install and retained N-1 migration upgrade: pass
- Then-current pinned local OIDC Authorization Code + PKCE integration: pass
- OpenAPI examples and TypeScript/Go generation drift: pass
- All module-owned SQLC generation drift: pass
- Task-owned OIDC-provider/PostgreSQL containers, volume, and network cleanup: pass

Additional fresh gates:

- `go vet ./...`: pass
- React/Vitest: 32/32 pass
- Root Vanilla JavaScript smoke suite: 103/103 pass
- React demo build: pass
- React HTTP build: pass
- `git diff --check`: pass

This evidence supports only the local Task 10 authority candidate. The next
binding slice is Task 11. A `production-ready` claim remains blocked by the
separate release/operations gate.
