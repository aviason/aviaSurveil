# Local Release-Candidate Evidence — 2026-07-21

## Outcome

- Local release-candidate recommendation: `GO`
- Evidence status: `verified locally`
- Artifact status: `candidate-only`
- Release status: `release pending`
- Production deployment and cutover decision: `NO-GO` and `blocked`
- Production hosting, production OIDC/MFA, traffic routing, legacy removal, and a `production-ready` claim: `blocked`

Task 13 completes the authorized local verification packet for Tasks 5-13. The React/Vite mock and HTTP entries, one-module Go API/worker, PostgreSQL authority path, then-current local OIDC exchange, private MinIO-compatible object flow, deterministic scanner, field-only browser persistence, foreground sync, and approved route families pass the required local gates. The root Vanilla JavaScript demo remains intact as the removal-blocking behavioral reference.

This decision authorizes no deployment, production traffic, cutover, or legacy archival/removal. A separately approved production release/operations plan does not exist in this authorized slice; therefore production remains `blocked` even though the local candidate receives `GO`.

## Test-First Corrections

The release-candidate work began with focused failures and closed each local defect before the final matrix:

- The API security test did not compile because security-header and rate-limit middleware were absent. The resulting middleware now applies an API-appropriate CSP and defensive headers, bounds login and mutation request classes, uses the socket peer rather than untrusted forwarded headers, and returns a closed `429` problem response with `Retry-After`.
- The worker observability test did not compile because batch processing had no testable reporting boundary. The worker now emits structured completed/failed batch records, while the live profile proves scan work drains without terminal outbox rows.
- The recovery command initially failed because `scripts/test-local-recovery.sh` did not exist. The final local-only drill restores an isolated PostgreSQL backup and exact MinIO object bytes, validates fingerprints/hashes/metadata, and cleans only its dedicated resources.
- The CSP test initially failed because the build policy module was absent. Subsequent red runs exposed TypeScript build inclusion, Vite development style, and keyboard focus-outline defects; the final demo/HTTP artifacts use explicit production CSP values, while local Vite allowances remain development-only.
- The first live conflict browser run stopped at the restart-canary readiness gate. A shared real-IndexedDB test precondition fixed the scenario, which now proves typed conflict presentation, local-draft preservation, explicit re-entry, and authoritative revision advancement.
- The first complete offline run selected two HTTP-only tests because of an overly broad filename pattern. The offline project now explicitly excludes that HTTP file without skipping any selected test; the intended real-offline matrix passes 6/6.
- The initial full dependency audit reported two high-severity development-tool findings in `js-yaml` through `@redocly/openapi-core`. A narrow lock-compatible override to `js-yaml` 4.3.0 replaced the vulnerable transitive version; clean install, contract generation, the complete audit, and the production-only audit all pass with zero vulnerabilities.

## Fresh Verification Matrix

Every successful result below completed with exit code `0` on 2026-07-21:

| Gate | Result |
|---|---|
| Clean dependency install | `npm --prefix apps/web ci`; 158 packages installed, 159 audited, 0 vulnerabilities |
| OpenAPI and generated-contract drift | 6/6; OpenAPI examples, Auditee closed projections, sync unions, first-production routes, generated TypeScript, and generated Go clean |
| React type/unit/component | TypeScript passed; Vitest 17 files, 148/148 tests, 0 skipped |
| Build and artifact boundaries | Demo and HTTP builds passed; HTTP scan passed across 12 files and 89 build inputs; mock/seed/demo-public/test-profile inputs absent |
| App shell and CSP | Demo and HTTP scans passed across 12 files and 4 shell assets each; production policies exclude unsafe inline/eval and wildcard sources |
| Go build/vet/race and live integration | API/worker builds and `go vet` passed; the isolated HTTP profile passed the full Go race suite and live PostgreSQL/OIDC-provider/MinIO integration, migrations, upload/scan, raw authorization, sync, and cleanup |
| Live HTTP Backend contract | 11/11 |
| Mock browser profile | 5/5, including canonical lifecycle, all approved first-production entries at desktop/tablet/mobile, keyboard/focus/target checks, stable reset, and zero unexpected console issues |
| HTTP browser profile | 7/7, including the same parity matrix, lost acknowledgement, foreground recovery, explicit stale-revision conflict resolution, and zero unexpected console issues outside the deliberately aborted acknowledgement |
| Real offline browser profile | 6/6 using isolated Chrome profiles: stopped-origin restart, IndexedDB pending/in-flight recovery, exact OPFS attachment byte/hash recovery, two-client N/N-1 update/rollback, managed-policy/persistence denial, and advisory quota denial |
| Root legacy and parity oracle | 106/106, 0 skipped; root demo unchanged |
| Focused security and worker tests | API CSP/headers, rate limits, session/CSRF/authentication, and worker batch reporting passed |
| Dependency audits | Full npm audit: 0 total; production-only npm audit: 0 total |
| Dependency inventories | CycloneDX 1.5 npm SBOM: 158 components, SHA-256 `1a41728b6fafc6c22d534f67f48e9da13692613efca8988c025215a360f1c584`; Go API/worker runtime inventory: 30 modules, SHA-256 `700e3fe011a93252e95216e33d6400260508cf85337a1b403b7d24ac30676569` |
| PostgreSQL recovery | Isolated dump/restore and canonical fingerprint comparison passed; drill artifact SHA-256 `09db7e26161122de4976d1915e4c4dab7767cf27f00ff2fc0aab9aa1728e82a1` |
| Object-store recovery | Exact private object, 47 bytes, metadata and SHA-256 `ba47f0913c1d12b747062e178b1e346a80a1bf8be2f4b645d08cf0d3cc12d08d` restored and verified |
| Worker/outbox observability | Live profile observed completed scan batches, zero pending scan-request outbox rows, and zero terminal scan-request rows |

The supported local browser evidence uses Google Chrome `150.0.7871.129` with isolated test profiles. The toolchain was Node `24.16.0`, npm `11.13.0`, and Go `1.26.4` on Darwin arm64.

## Security And Operational Boundary Review

- Same-origin session, OIDC state/PKCE, secure cookie, CSRF, expiry/revocation, role, organization, assignment, direct-ID, list, pull, and conflict boundaries pass focused and live Go tests. The then-current configured local OIDC flow is test evidence, not production Identity evidence.
- Raw Auditee JSON tests scan Finding, CAP, Evidence, report, assignment, dashboard, direct-ID/list, and sync projections for forbidden Internal CAA, other-organization, workload/risk, unreleased-report, and enforcement material.
- Official Evidence upload remains private, bounded to PDF/JPEG/PNG and 25 MB, server-observed, non-overwriting, immutable by version, quarantined until scan-clean, and gated for review/download/closure. Offline Inspection Attachments remain distinct from official Evidence.
- Field records are subject-scoped; official checkout refuses missing managed-policy, persistence, storage-health, version, grant, or advisory-capacity gates. Site-data clearing remains an explicit irrecoverable unsynced-copy boundary. App-level encryption and production MDM evidence are not claimed.
- CSP and request-rate controls are locally verified. Production reverse-proxy limits, WAF policy, distributed counters, and external penetration testing remain external evidence.
- The local PostgreSQL/object-store restore and prior-shell rollback rehearsals passed. They do not establish production RPO/RTO, provider recovery, or disaster-recovery ownership.
- Worker logs and outbox state are observable in the local profile. Production metrics, alerts, SLOs, paging, and on-call are not configured.

## External Production Gaps

These gaps do not block the local candidate; they do block production. Each remains recorded in the [tech-debt tracker](../exec-plans/tech-debt-tracker.md):

| Gap | Owner | Status |
|---|---|---|
| Production OIDC/MFA, secrets, identity operations, and external security review | Security + Identity | `blocked` |
| Production provider/region, trusted build/provenance, deployment, migrations, backup/restore, RPO/RTO, monitoring, SLO, and on-call | Platform + Operations | `blocked` |
| Retention, legal hold, disposition, records classification, and tamper-evidence acceptance | Records + Legal + Security | `blocked` |
| Production managed-device/browser policy, app-level local-data encryption/key ownership, and site-data incident procedure | Security + CAA Operations + Records | `blocked` |
| Pilot acceptance, release authority, routing, rollback thresholds, and legacy cutover/removal decision | Product + Platform + Operations + QA | `blocked` |
| Separately approved production release/operations plan | Product + Platform + Operations + QA | `blocked`; not authorized or created in this slice |

## Reproduction Commands

```bash
npm --prefix apps/web ci
npm --prefix apps/web run contracts:check
npm --prefix apps/web run typecheck
npm --prefix apps/web test
npm --prefix apps/web run build:demo
npm --prefix apps/web run build:http
node apps/web/scripts/assert-http-artifact.mjs apps/web/dist
npm --prefix apps/web run check:app-shell
GOCACHE=/private/tmp/aviasurveil360-go-cache go -C apps/api build ./cmd/api ./cmd/worker
GOCACHE=/private/tmp/aviasurveil360-go-cache go -C apps/api vet ./...
./scripts/test-http-profile.sh
npm --prefix apps/web run test:e2e:offline
./scripts/test-local-recovery.sh
node --test tests/*.test.js tests/parity/react-legacy-parity.test.mjs
```

Network-backed dependency audits were run separately with `npm audit --json` and `npm audit --omit=dev --json`. SBOM and runtime inventories were generated into task-owned temporary paths and were not added as release artifacts.

## Final Scope Statement

Tasks 5-13 are complete for the authorized local candidate and are `verified locally`. The result is `candidate-only` and `release pending`. Production deployment, production traffic, cutover, legacy removal, and `production-ready` remain `blocked` pending explicit authorization and a separately approved production release/operations plan with fresh external evidence.
