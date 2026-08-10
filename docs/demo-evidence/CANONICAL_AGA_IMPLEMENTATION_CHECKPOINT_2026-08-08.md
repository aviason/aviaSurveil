# Canonical AGA Implementation Checkpoint — 2026-08-08

This is a local implementation checkpoint for the canonical AGA successor
plan. It is `candidate-only`; it is not stakeholder acceptance, release, or
production evidence. Task 10 external preprod deployment remains explicitly
`not run`.

## Verified locally

| Check | Result |
|---|---|
| OpenAPI bundle, generated Go/TypeScript contracts, and contract suite | `verified locally` — 16/16 checks passed |
| Canonical/OpenAPI/donor-boundary JS/MJS subset | `verified locally` — 70/70 tests passed |
| Focused Go packages (`internal/httpapi`, `internal/assignments`, `internal/application`, `migrations`) | `verified locally` |
| Full non-race Go suite (`go -C apps/api test -count=1 ./...`) | `verified locally` — all packages passed on a task-owned disposable PostgreSQL/MinIO target |
| Empty-database and retained N-1 migration upgrade, including populated 000029 fixture and migration 38 | `verified locally` |
| React typecheck | `verified locally` |
| Full React test suite | `verified locally` — 95 files / 796 tests passed |
| Current-source local runtime images and SBOMs | `verified locally` — all 9 runtime images built and 9 digest-bound CycloneDX SBOMs were generated |
| Donor-free HTTP/Go artifact boundary and local-preprod runtime-role boundary | `verified locally` — normal API/worker/scheduler/migrate dependency and binary-marker scans, focused artifact/compose tests, and disposable PostgreSQL privilege probe |
| Disposable canonical local-preprod stack and privacy-safe demo identity seed | `verified locally` — migration 41, `aga-preprod@1.0.0` catalog (1,310 questions), API readiness, Keycloak realm (9 role-mapped users), MinIO private buckets, ClamAV `PONG`, Gotenberg health, and Mailpit API health |
| Connected canonical OIDC lifecycle | `verified locally` — separate isolated runs passed Manager New Audit selection → Finance → GM → ED → GM Release → Lead/team/coverage → preparation → announced coordination → Inspector start, then checklist/Potential Finding → Preliminary Report issue → Finding → CAP acceptance → real MinIO Evidence upload/ClamAV `CLEAN` → Evidence closure → Final Report issue and Manager dashboard; `1 passed` for each run |
| Consolidated public-HTTPS 1,310-question lifecycle | `verified locally` — after a clean profile regeneration and the runtime/object-store credential split, `make preprod-cloudflare-test-lifecycle` passed `1/1` in 5.0 minutes with distinct OIDC roles: exact 1,310-question New Audit selection; Finance → GM → ED → GM Release; Lead/team/exact per-question coverage; preparation/materialization; pre-start package denial; separate Inspector start; all 1,310 response writes; Potential Finding; Preliminary Report issue/lock; post-issue Finding conversion; CAP acceptance without closure; signed MinIO Evidence upload; ClamAV `PENDING` → `CLEAN`; Evidence-verified closure; Final Report issue/lock; immutable Gotenberg document/version and SHA-256 projections; Auditee privacy; and anonymous Evidence-download denial |
| PostgreSQL and object-store backup/restore | `verified locally` — a task-owned recovery profile produced a non-empty custom-format PostgreSQL dump, restored it into an isolated database with an identical canonical fingerprint, and restored an exact-byte private MinIO sentinel; the profile then removed its containers, volumes, network, and runtime directory |
| Official focused HTTP user-lifecycle E2E | `verified locally` — `1 passed`; outbox drain verified locally |
| Dependency boundary smoke | `verified locally` — MinIO disposable put/get/delete, ClamAV clean acceptance plus EICAR rejection, Gotenberg synthetic HTML→PDF, and authenticated Mailpit SMTP/API delivery |
| Safari/WebKit and OIDC app-shell navigation regression | `verified locally` — local web server `/index.html` returns `200` without `Location`; app-shell worker/manifest version 8 uses `/` only for application navigations and makes `/identity`, `/auth`, API, health, and private-object routes network-only. The focused worker policy, typecheck, HTTP build, and app-shell/HTTP artifact scans passed |
| Disposable Cloudflare Quick Tunnel transport and role-panel E2E | `verified locally` — a fresh `make preprod-cloudflare-link` cold profile created and revalidated an anonymous random `https://*.trycloudflare.com` endpoint backed only by loopback `http://127.0.0.1:8085`; public root/readiness and exact HTTPS OIDC endpoints passed. With Service Workers explicitly enabled, every test first proved that the public app was worker-controlled and that Keycloak rendered visible username/password fields. `make preprod-cloudflare-test-panels` passed all nine role-panel cases plus the provider logout/account-switch case (10/10 in 1.6 minutes) with exact role/organization sessions, `__Host-avia_session` Secure/HttpOnly/SameSite=Strict and Secure CSRF cookies, every role home, Department Manager Question Review, and New Audit discovery of the server-owned 1,310-question catalog. An Admin session was revoked, the browser received only a short-lived encrypted same-origin logout ticket, the API redeemed it into the discovery-bound Keycloak end-session redirect, and the same browser displayed username/password fields and accepted Manager credentials without silently restoring Admin. Provider tokens never entered the application JavaScript response; every new authorization carries `prompt=login` and `max_age=0`. The clean profile also derived the Manager pending-report count from the server projection. No account, token, named tunnel, DNS, AWS, or external-preprod action was used; diagnostic screenshots were temporary and were not retained in repository evidence |
| `git diff --check` | `verified locally` |

The checkpoint includes the server-owned preparation confirmation revision pin,
restart-safe Department Manager projection, multi-Inspector immutable coverage
key, cumulative governed successor review facts, controlled governed reasons,
disposable-scope authorization for exercise review commands, and the governed
virtual-candidate queue dispatch required for technical approval → publish.
The Department Manager projection also owns the pending-report review count, so
a clean profile does not request or display a synthetic fixed report identity.
Migration 38 reconciles pre-38 confirmed receipts with immutable assignment
revision successors; hydrated confirmations disable repeat submission. Question
Review now returns bounded append-only Draft history, New Audit summary values
are recomputed from immutable catalog memberships, the Lead handoff uses a
canonical pre-materialization route, donor workspace operations are absent from
the normal HTTP contract/client/artifact, and long-lived local-preprod services
use a provisioned non-owner database role.

## Not run or blocked

- Full race Go harness: `blocked` — the official race profile reached the
  existing `internal/agaapplicability` fixture test and timed out after ten
  minutes; the non-race full suite above passed on a task-owned disposable
  database target.
- Root JS/MJS harness smoke (`tests/*.test.js` plus parity): `verified locally`
  — 108 tests passed. Complete recursive discovery also ran 94 files / 451
  tests and returned 418 pass / 33 fail. The exact split is 12 paused AGA donor
  failures; one active governed-intake archive check blocked by absent
  `AGA_CHECKLIST_ARCHIVE`; seven AviaCore failures (three missing external
  sibling/predecessor fixtures and four local registry/decision drifts); and
  13 AWS/OPA-family failures (one stale local AWS fixture and 12 missing local
  OPA executions). The four local AviaCore drifts remain separately governed
  and blocked pending AviaCore owner authority; no owner disposition is
  inferred by this checkpoint.
- Full negative/fault visible-action matrix, full object denial and
  notification matrix, visual/browser viewport evidence, donor
  deletion/requalification, and stakeholder review: `not run`. The connected
  canonical hero lifecycle is verified above; the local stack is intentionally
  left running for the user-owned manual visual pass. The Quick Tunnel
  nine-role browser callback/cookie/panel gate is verified above; only the
  separate 1440x900, 1024x768, and 390x844 visual review remains `not run /
  stakeholder pending` and user-owned. Safari may require a one-time localhost
  site-data/service-worker clear after this app-shell fix.
- Full recursive root JS/MJS discovery: `blocked` — 94 discovered files,
  418 passed / 33 failed with the exact 12 + 1 + 7 + 13 split above. This
  includes four local AviaCore synchronization drifts and 13 local AWS/OPA
  execution/fixture failures, so it is not classified solely as external or
  unauthorized remote work. These failures are not in the canonical
  browser/runtime suites.
- HIGH/CRITICAL local-image vulnerability gate: `blocked` — the sole Keycloak
  Java 21/Java 17 advisory-mismatch exception expired on 2026-08-08. A fresh
  exception-free Trivy scan reproduced one HIGH mapping on
  `java-21-openjdk-headless`, titled as a Java 17 update and with no fixed
  version; Keycloak JAR scanning returned zero findings. No Local Platform
  Security exception extension was inferred.
- External preprod deployment and all remote infrastructure actions: `not run`
  and unauthorized by this plan execution request.

The final read-only Sol XHigh code-boundary reread of the latest changes was
`ACCEPTED` with 0 Critical / 0 Important findings. It does not replace the
blocked image-security, connected-negative, visual, or stakeholder gates.
The local implementation remains `candidate-only` and `release pending`.
