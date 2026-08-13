# Canonical AGA Implementation Checkpoint — 2026-08-08

This is a local implementation checkpoint for the canonical AGA successor
plan. It is `candidate-only`; it now records stakeholder acceptance of the
manual local viewport milestone, but it is not release or production evidence.
External preprod deployment was `not run` and, by the
user's 2026-08-10 sequencing decision, is now outside this plan and tracked by
the separate paused
[Canonical AGA External Preprod Release And Handoff](../exec-plans/active/2026-08-10-canonical-aga-external-preprod-release-plan.md).

## Verified locally

| Check | Result |
|---|---|
| OpenAPI bundle, generated Go/TypeScript contracts, and contract suite | `verified locally` — 16/16 checks passed |
| Canonical/OpenAPI/donor-boundary JS/MJS subset | `verified locally` — post-deletion generated-contract, static-boundary, and normal-artifact checks passed |
| Focused Go packages (`internal/httpapi`, `internal/assignments`, `internal/application`, `migrations`) | `verified locally` |
| Full non-race Go suite (`go -C apps/api test -count=1 ./...`) | `verified locally` — all packages passed on a task-owned disposable PostgreSQL/MinIO target |
| Empty-database and retained N-1 migration upgrade, including populated 000029 fixture and migration 38 | `verified locally` |
| React typecheck | `verified locally` |
| Full React test suite | `verified locally` — post-deletion 85 files / 748 tests passed |
| Current-source local runtime images and SBOMs | `verified locally` — all 9 runtime images built and 9 digest-bound CycloneDX SBOMs were generated |
| Donor-free HTTP/Go artifact boundary and local-preprod runtime-role boundary | `verified locally` — normal API/worker/scheduler/migrate dependency and binary-marker scans, focused artifact/compose tests, and disposable PostgreSQL privilege probe |
| Disposable canonical local-preprod stack and privacy-safe demo identity seed | `verified locally` — migration 41, `aga-preprod@1.0.0` catalog (1,310 questions), API readiness, then-current OIDC provider (9 role-mapped users), MinIO private buckets, ClamAV `PONG`, Gotenberg health, and Mailpit API health |
| Connected canonical OIDC lifecycle | `verified locally` — separate isolated runs passed Manager New Audit selection → Finance → GM → ED → GM Release → Lead/team/coverage → preparation → announced coordination → Inspector start, then checklist/Potential Finding → Preliminary Report issue → Finding → CAP acceptance → real MinIO Evidence upload/ClamAV `CLEAN` → Evidence closure → Final Report issue and Manager dashboard; `1 passed` for each run |
| Consolidated public-HTTPS 1,310-question lifecycle | `verified locally` — after a clean profile regeneration and the runtime/object-store credential split, `make preprod-cloudflare-test-lifecycle` passed `1/1` in 5.0 minutes with distinct OIDC roles: exact 1,310-question New Audit selection; Finance → GM → ED → GM Release; Lead/team/exact per-question coverage; preparation/materialization; pre-start package denial; separate Inspector start; all 1,310 response writes; Potential Finding; Preliminary Report issue/lock; post-issue Finding conversion; CAP acceptance without closure; signed MinIO Evidence upload; ClamAV `PENDING` → `CLEAN`; Evidence-verified closure; Final Report issue/lock; immutable Gotenberg document/version and SHA-256 projections; Auditee privacy; and anonymous Evidence-download denial |
| PostgreSQL and object-store backup/restore | `verified locally` — a task-owned recovery profile produced a non-empty custom-format PostgreSQL dump, restored it into an isolated database with an identical canonical fingerprint, and restored an exact-byte private MinIO sentinel; the profile then removed its containers, volumes, network, and runtime directory |
| Official focused HTTP user-lifecycle E2E | `verified locally` — `1 passed`; outbox drain verified locally |
| Dependency boundary smoke | `verified locally` — MinIO disposable put/get/delete, ClamAV clean acceptance plus EICAR rejection, Gotenberg synthetic HTML→PDF, and authenticated Mailpit SMTP/API delivery |
| Safari/WebKit and OIDC app-shell navigation regression | `verified locally` — local web server `/index.html` returns `200` without `Location`; app-shell worker/manifest version 8 uses `/` only for application navigations and makes `/identity`, `/auth`, API, health, and private-object routes network-only. The focused worker policy, typecheck, HTTP build, and app-shell/HTTP artifact scans passed |
| Disposable Cloudflare Quick Tunnel transport and role-panel E2E | `verified locally` — a fresh `make preprod-cloudflare-link` cold profile created and revalidated an anonymous random `https://*.trycloudflare.com` endpoint backed only by loopback `http://127.0.0.1:8085`; public root/readiness and exact HTTPS OIDC endpoints passed. With Service Workers explicitly enabled, every test first proved that the public app was worker-controlled and that the then-current provider rendered visible username/password fields. `make preprod-cloudflare-test-panels` passed all nine role-panel cases plus the provider logout/account-switch case (10/10 in 1.6 minutes) with exact role/organization sessions, `__Host-avia_session` Secure/HttpOnly/SameSite=Strict and Secure CSRF cookies, every role home, Department Manager Question Review, and New Audit discovery of the server-owned 1,310-question catalog. An Admin session was revoked, the browser received only a short-lived encrypted same-origin logout ticket, the API redeemed it into the discovery-bound provider end-session redirect, and the same browser displayed username/password fields and accepted Manager credentials without silently restoring Admin. Provider tokens never entered the application JavaScript response; every new authorization carries `prompt=login` and `max_age=0`. The clean profile also derived the Manager pending-report count from the server projection. No account, token, named tunnel, DNS, AWS, or external-preprod action was used; diagnostic screenshots were temporary and were not retained in repository evidence |
| Task 8 negative/fault/restart and cleanup matrix | `verified locally` — `make preprod-test-fault-restart` used a unique disposable local-HTTPS project, passed the selected real-PostgreSQL transaction/fault/concurrency suite in 37.846 seconds, repeated the full 1,310-question OIDC lifecycle (`1/1`, 4.8 minutes), preserved authoritative database SHA-256 `3c13f4999eba1d942a079117912af42f961c897046271ca37bc3fb3c0f6e333e` across a cold full-stack restart, and passed the post-restart nine-role plus forced-credential logout matrix (`10/10`, 1.3 minutes). PostgreSQL/OIDC-provider/MinIO/ClamAV loss produced live `200` plus ready `503/not_ready`; Gotenberg/Mailpit loss produced ready `200/degraded`; all recovered to `ready`; an injected worker crash incremented restart count 1→2; pre-auth/authenticated donor probes failed closed; generated secrets were absent from logs; and cleanup left zero labelled containers, volumes, networks, task-owned processes, or runtime root. The separately running `demo.aviasurveil.com` profile remained ready locally and publicly. |
| Task 9 physical donor deletion and post-deletion requalification | `verified locally` — after the user explicitly selected `delete`, 153 tracked donor/obsolete compatibility files were removed and the sealed package reader was moved into canonical import ownership. Migration 42 removes the fixed-ID mutable Inspection Package Draft table. Full Go passed on disposable PostgreSQL/MinIO (integration 212.490 seconds); current React passed 85/85 files and 748/748 tests; demo/HTTP builds, generated contracts, normal dependency/binary and donor-artifact scans passed. Recursive root discovery improved to 373/394 with all 12 donor failures gone. The final current-image matrix passed selected PostgreSQL transaction/concurrency tests in 47.484 seconds, the 1,310-question OIDC lifecycle `1/1` in 5.9 minutes, a real Auditee→CAA notification with worker `DELIVERED` state and Mailpit receipt, restart fingerprint `6902377347b88cf11c6558af7af2a594f4542b0b81bda8491ce143cc696b1ddd`, role/logout `10/10` in 1.4 minutes, every dependency loss/recovery case, worker restart 2→3, donor fail-closed probes, and zero residue. A final isolated HTTP publication-boundary run passed typecheck, focused transport `3/3`, Task 9 Go integration, and real Admin import → inspect → submit → Department Manager technical approval → publication browser checks at 1440x900 and 390x844 (`2/2`), then left zero task-owned Docker/browser residue. The live demo profile was not changed. |
| Privacy-safe Question Review and New Audit viewport capture | `verified locally` — [nine current-worktree images](canonical-aga-manual-review-2026-08-10/README.md) cover New Audit selection, Question Review queue + populated Decision file, and its history/decision controls at 1440x900, 1024x768, and 390x844. The capture selected three exact immutable versions from the 1,310-question disposable catalog, recorded a scoped exercise `RETAIN`, retained no real question body or visible catalog label, and reported zero document overflow, zero product HTTP errors, zero browser console/page errors, and zero request failures. The user accepted the manual review on 2026-08-11. |
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
  — 108 tests passed. Complete post-deletion recursive discovery ran 88 files /
  394 tests and returned 373 pass / 21 fail. All 12 paused AGA donor failures
  are gone. The remaining exact split is one active governed-intake archive
  check blocked by absent
  `AGA_CHECKLIST_ARCHIVE`; seven AviaCore failures (three missing external
  sibling/predecessor fixtures and four local registry/decision drifts); and
  13 AWS/OPA-family failures (one stale local AWS fixture and 12 missing local
  OPA executions). The four local AviaCore drifts remain separately governed
  and blocked pending AviaCore owner authority; no owner disposition is
  inferred by this checkpoint.
- Manual stakeholder visual disposition: `accepted` on 2026-08-11. The connected canonical hero and final Task 8
  negative/fault/restart matrix are verified above; the local stack is
  intentionally available for the current public demo. The Quick Tunnel
  nine-role browser callback/cookie/panel gate is verified above. The separate
  nine-image 1440x900, 1024x768, and 390x844 capture package is prepared and
  `verified locally`; the user accepted that manual review for the local
  `candidate-only` milestone.
  Task 9 no longer blocks it. Safari may
  require a one-time localhost site-data/service-worker clear after this
  app-shell fix.
- Full recursive root JS/MJS discovery: `blocked` — 88 discovered files,
  373 passed / 21 failed with the exact 1 + 7 + 13 split above. This
  includes four local AviaCore synchronization drifts and 13 local AWS/OPA
  execution/fixture failures, so it is not classified solely as external or
  unauthorized remote work. These failures are not in the canonical
  browser/runtime suites.
- External preprod deployment and all remote infrastructure actions: outside
  this checkpoint's plan, `not run`, unauthorized, and tracked by the separate
  paused external-preprod ExecPlan.

The final read-only Sol XHigh code-boundary reread of the latest post-deletion
implementation and HTTP publication-boundary changes was `ACCEPTED` with 0
Critical / 0 Important findings. It does not
replace or pass the blocked image-security, root-discovery, or full-race gates.
The local implementation remains `candidate-only` and `release pending`.
