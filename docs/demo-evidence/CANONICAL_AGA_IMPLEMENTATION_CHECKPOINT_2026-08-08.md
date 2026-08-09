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
| Full React test suite | `verified locally` — 91 files / 769 tests passed |
| Donor-free HTTP/Go artifact boundary and local-preprod runtime-role boundary | `verified locally` — normal API/worker/scheduler/migrate dependency and binary-marker scans, focused artifact/compose tests, and disposable PostgreSQL privilege probe |
| Disposable canonical local-preprod stack and privacy-safe demo identity seed | `verified locally` — migration 41, `aga-preprod@1.0.0` catalog (1,310 questions), API readiness, Keycloak realm (9 role-mapped users), MinIO private buckets, ClamAV `PONG`, Gotenberg health, and Mailpit API health |
| Connected canonical OIDC lifecycle | `verified locally` — separate isolated runs passed Manager New Audit selection → Finance → GM → ED → GM Release → Lead/team/coverage → preparation → announced coordination → Inspector start, then checklist/Potential Finding → Preliminary Report issue → Finding → CAP acceptance → real MinIO Evidence upload/ClamAV `CLEAN` → Evidence closure → Final Report issue and Manager dashboard; `1 passed` for each run |
| Official focused HTTP user-lifecycle E2E | `verified locally` — `1 passed`; outbox drain verified locally |
| Dependency boundary smoke | `verified locally` — MinIO disposable put/get/delete, ClamAV clean acceptance plus EICAR rejection, Gotenberg synthetic HTML→PDF, and authenticated Mailpit SMTP/API delivery |
| Safari/WebKit and OIDC app-shell navigation regression | `verified locally` — local web server `/index.html` returns `200` without `Location`; app-shell worker/manifest version 7 uses `/` only for application navigations and makes `/identity`, `/auth`, API, health, and private-object routes network-only. The focused worker policy, typecheck, HTTP build, and app-shell/HTTP artifact scans passed |
| Disposable Cloudflare Quick Tunnel transport and role-panel E2E | `verified locally` — `make preprod-cloudflare-link` created and revalidated an anonymous random `https://*.trycloudflare.com` endpoint backed only by loopback `http://127.0.0.1:8085`; public root/readiness and exact HTTPS OIDC endpoints passed. With Service Workers explicitly enabled, every test first proved that the public app was worker-controlled and that Keycloak rendered visible username/password fields. `make preprod-cloudflare-test-panels` then passed 9/9 real public Keycloak logins, exact role/organization sessions, `__Host-avia_session` Secure/HttpOnly/SameSite=Strict and Secure CSRF cookies, every role home, Department Manager Question Review, and a New Audit draft through the server-owned 1,310-question catalog, exact one-question preview/confirm, and Step 5 review. The same provider form was confirmed in the Codex in-app WebKit browser. No account, token, named tunnel, DNS, AWS, external-preprod action, screenshot, video, or trace was used |
| `git diff --check` | `verified locally` |

The checkpoint includes the server-owned preparation confirmation revision pin,
restart-safe Department Manager projection, multi-Inspector immutable coverage
key, cumulative governed successor review facts, controlled governed reasons,
disposable-scope authorization for exercise review commands, and the governed
virtual-candidate queue dispatch required for technical approval → publish.
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
  — 108 tests passed; broader recursive discovery remains blocked by existing
  AGA/AviaCore/AWS and paused-donor fixture-contract families.
- Full role/action/negative visible-action matrix, full object upload/scan/
  render/mail matrix, backup/restore, visual/browser viewport evidence, donor
  deletion/requalification, and stakeholder review: `not run`. The connected
  canonical hero lifecycle is verified above; the local stack is intentionally
  left running for the user-owned manual visual pass. The Quick Tunnel
  nine-role browser callback/cookie/panel gate is verified above; only the
  separate 1440x900, 1024x768, and 390x844 visual review remains `not run` and
  user-owned. Safari may require a one-time localhost
  site-data/service-worker clear after this app-shell fix.
- Full recursive root JS/MJS discovery: `blocked` — 110 discovered files,
  459 passed / 34 failed; the failures are existing paused AGA/AviaCore/AWS
  fixture-contract families and the pre-existing Gotenberg/preprod boundary
  contract expectations, not the canonical browser smoke.
- External preprod deployment and all remote infrastructure actions: `not run`
  and unauthorized by this plan execution request.

Sol Ultra's latest independent read-only implementation review remains
`NOT ACCEPTED` pending a fresh post-remediation reread. The final Sol XHigh
code-boundary reread is `ACCEPTED` with 0 Critical / 0 Important findings; it
does not replace connected, visual, or stakeholder gates.
The local implementation remains `candidate-only` and `release pending`.
