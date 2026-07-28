# Local Production-Like Services Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use
> `superpowers:executing-plans` to execute this plan task by task. Do not
> dispatch subagents unless the user explicitly authorizes subagent work.
> Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Run the complete 86-screen AviaSurveil360 application locally through
Docker Compose with production-like HTTPS, OIDC/MFA/provisioning, private object
storage, real malware scanning, SMTP delivery, PDF rendering, workers, secrets,
and clean operational boundaries.

**Architecture:** Keep the application as one React artifact and one Go module,
but package the gateway, API, workers, identity, databases, object store,
scanner, email sink, and document renderer as isolated containers. Caddy owns
the one local HTTPS origin. Application commands persist durable jobs first;
workers call typed Keycloak, ClamAV, SMTP, Gotenberg, and MinIO adapters with
idempotent retries and observable terminal state.

**Tech Stack:** Docker Compose, multi-stage Dockerfiles, Caddy, React/Vite
static HTTP artifact, Go API/worker, PostgreSQL 17, Keycloak 26-compatible OIDC
realm with TOTP MFA, MinIO, ClamAV `clamd`/`freshclam`, Mailpit SMTP, Gotenberg,
Docker secrets, SOPS + age, Playwright, and the existing contract/integration
test harness.

**Status:** `completed` — prerequisite acceptance and Tasks 1–9 remain
`verified locally`. On 28 July 2026 the user accepted Plan 3 as a completed
local `candidate-only` milestone. Release is `release pending`, and deployment
and production readiness are `not run`.

## Stakeholder Closeout — 28 July 2026

The user explicitly authorized the combined Plans 2–4 stakeholder closure and
accepted this plan only as a completed local `candidate-only` milestone. The
canonical implementation and verification basis remains
[Local Production-Like Services Evidence](../../demo-evidence/LOCAL_PRODUCTION_LIKE_SERVICES_2026-07-22.md);
the combined decision and shared exclusions are recorded in the
[Plans 2–4 Stakeholder Disposition](../../demo-evidence/stakeholder/PLANS2_4_STAKEHOLDER_DISPOSITION_2026-07-28.md).

The existing Keycloak `CVE-2026-22020` advisory mismatch remains open through
its recorded owner and expiry boundary. Local CA trust, production identity
federation, external email/storage/scanning/document providers, production
records and operating policy, deployment, and release remain owner decisions.
The artifact remains `candidate-only`; release remains `release pending`;
deployment and production readiness remain `not run`; no `production-ready`
claim is made.

## Progress

### Prerequisite Acceptance — 24 July 2026

Plan 1 is accepted for this Plan 3 start with its boundaries preserved
literally. The frozen handoff passed 44/44 focused route/capability tests, the
exact 86-route/two-profile boundary, 5/5 parity-ledger tests, 14/14 canonical
OpenAPI example tests, all 258 tracked baseline PNG hashes, and exact root
oracle source hashes. Its one-shot visual matrix remains `not verified` at
71/259, complete comparison-by-comparison review remains `not run`, and
standalone baseline integrity remains `not verified` because the accepted
manifest audit hash differs from the user-edited canonical audit hash. No
failed comparison was converted into a pass and no accepted baseline, mask,
threshold, semantic identity, authority rule, or root source was changed.

Plan 2 received a fresh evidence-based review in this task. The current
implementation was inspected across the 28 Backend capability registry,
75-path/81-operation generated contract, 10 scenario families and 45 proofs,
test-profile exclusion boundary, identity and organization authorization,
immutable CAP/Evidence/report/document persistence, and durable scan,
notification, document, provisioning, audit, change, and outbox jobs. Fresh
contract generation/drift passed 15/15, SQLC drift passed, selected Go packages
passed under `-race`, and 13 high-value live PostgreSQL/Keycloak/MinIO
integration tests passed under `-race` in a unique task-owned Compose project.
The focused live selection covered schema migration, immutable rows,
transaction linkage, identity/profile authority, provisioning job/session
revocation, organization pre-guarding, full Finding/CAP/Evidence closure
authority, communications/notification delivery state, immutable document
rendering, raw-wire privacy, management/assistant boundaries, and real
Keycloak Authorization Code + PKCE. The scoped containers, volumes, and
network were removed with zero residue. Plan 2 is independently accepted for
the Plan 3 prerequisite; its artifact remains `candidate-only` and
`release pending`.

### Task 1 — 24 July 2026

The secure runtime foundation is `verified locally`. The initial policy suite
failed 0/11 for the intended missing validator, Compose topology, secret
initializer, and SOPS boundaries. The completed suite passes 16/16 and rejects
plaintext secret values, missing secret mounts, mutable external image
references, internal published ports, shared application/identity storage,
unrestricted networks, root or implicit runtime users, unreviewed writable
root filesystems, missing health checks, and missing or unexpected profile
services.

The canonical Compose configs pass for `demo`, `full`, `test`, and `recovery`
with 2, 12, 3, and 2 services respectively. Only `gateway` publishes
`127.0.0.1:8443` in demo/full; test/recovery publish no ports. The optional
tools profile publishes only Mailpit UI on loopback. Seven reviewed upstream
image references are digest locked. Application and Keycloak PostgreSQL data
use separate volumes, all non-gateway runtime networks are internal, generated
credentials use `0700`/`0600` permissions, MinIO uses a bounded random access
identity, SOPS decrypt validation passes with an ignored local age identity,
and an exact generated-secret scan found zero matches across 1,036 present
repository files. Spec-compliance and code-quality self-reviews found no
remaining Critical or Important issue. The planned Task 1 commit is `not run`
because Git staging and commit authorization were not granted.

### Task 2 — 24 July 2026

The containerized application and gateway boundary is `verified locally`.
The first focused boundary run passed 17/20 and failed on the intended absent
bounded web server and writable gateway runtime directories. Subsequent
RED → GREEN cycles covered non-root containerized SBOM/scanner socket access,
bounded writable tool tmpfs, correct gateway SNI health checks, automatic CA
trust-install suppression, SPA redirect behavior, explicit `/__test/*` 404
handling, and removal of an obsolete BusyBox image-lock entry. The completed
focused suite passes 21/21; the Compose policy regression remains 16/16.

Seven local runtime images build from reviewed digest-pinned bases with dirty
source-state evidence. All seven current image digests have digest-bound
CycloneDX SBOMs and pass the fail-closed HIGH/CRITICAL vulnerability policy
with no exception record. Syft and Trivy run from reviewed immutable tool
images under read-only, capability-dropped, no-new-privileges containers with
bounded tmpfs and no host tool dependency.

A clean unique demo project passed healthy Caddy and web checks, HTTPS root and
direct SPA loads, CSP/HSTS/content-type headers, no automatic trust-store
installation attempt, and exit-zero SIGTERM drain. A clean full-profile
gateway/web/API/PostgreSQL slice passed HTTPS root and liveness, browser-like
`/__test/*` and `/api/__test/*` 404 checks, HTTP artifact mock/seed exclusion,
one published gateway port, and exit-zero API/web/gateway drain. Full readiness
remains fail-closed at 503 until the Task 3 identity and Task 4 object-store
providers are active; this was not converted into a pass. The full 12-service
skeleton was also created from clean volumes, with gateway, HTTP web, API,
PostgreSQL, Keycloak PostgreSQL, and Gotenberg reaching their available Task 2
boundaries while later-task provider configuration remained incomplete. Every
Task 2 Compose project, container, network, and volume was removed with zero
task-owned residue.

Spec-compliance self-review found no remaining Critical or Important issue.
Code-quality self-review found and corrected the obsolete web-runtime lock
entry through a fresh RED → GREEN cycle; no Critical or Important issue
remains. These are main-agent self-reviews, not independent reviews. The
planned Task 2 commit is `not run` because Git staging and commit authorization
were not granted.

### Task 3 — 24 July 2026

Production-mode local identity, first-login TOTP MFA, and
application-authorized user provisioning are `verified locally`. The reviewed
realm source and deterministic `0600` runtime builder pass 3/3 contract tests
with exact HTTPS redirect/origin bindings, confidential Authorization Code +
PKCE, no self-registration, no web-client password grant, eight approved realm
roles, an admin-managed `organization_id`, and default `CONFIGURE_TOTP`.
Keycloak runs optimized behind the one gateway origin with its own PostgreSQL
database.

The Go Keycloak adapter, durable lifecycle request/outbox worker, two new
generated OpenAPI operations, HTTP transport, and React administration surface
provision, reconcile, update, suspend, reactivate, expose generic retry state,
record the retained provider subject, and revoke both application and provider
sessions. Focused Go identity/config tests pass under `-race`; five focused
session/lifecycle/HTTP PostgreSQL integration tests pass under `-race`;
contract generation/drift passes 15/15; SQLC drift, frontend typecheck, and
diff checks pass; and the focused frontend transport/UI suite passes 23/23.
The combined local Compose/security regression passes 19/19.

The clean production-mode browser scenario passes 1/1 and proves no
`/__test/*` route, real administrator first-login TOTP, exact session
role/organization claims, CSRF rejection, wrong-role and wrong-organization
rejection, duplicate-email conflict, application provisioning, exact Keycloak
role/organization/required-action mapping, Keycloak restart persistence, local
logout and subsequent provider SSO-or-MFA entry, provisioned-user first-login
TOTP, deactivation, application-session rejection, provider disable, and zero
remaining provider sessions. Session integration separately proves the
30-minute rolling idle policy, eight-hour absolute expiry, opaque credential
hashing, encrypted provider tokens, CSRF validation, logout/revocation, and
one-time OIDC state.

Spec-compliance self-review found and corrected fail-open role/organization
validation at the Keycloak adapter boundary. Code-quality self-review corrected
the stale UI role/organization fixture and added pre-transport validation,
aligned lifecycle idempotency evidence with HTTP `202`, synchronized the
OpenAPI path-order contract, removed committed plaintext test credentials, and
added a real runtime artifact/log scan. The final scan found zero generated
secret matches across API, worker, Keycloak, MinIO, and Playwright artifacts.
The unique Compose project, containers, network, volumes, Vite process,
Playwright worker, and headless browser left zero task-owned residue. These are
main-agent self-reviews, not independent implementation reviews. The planned
Task 3 commit is `not run` because Git staging and commit authorization were
not granted.

### Task 4 — 24 July 2026

Private object storage and real malware scanning are `verified locally`
through strict RED → GREEN cycles. The implementation defines four private,
versioned buckets, separate least-privilege API and worker MinIO identities,
exact quarantine-bucket upload validation, no-overwrite signed uploads,
separate internal and public signing endpoints, typed ClamAV results,
fail-closed signature freshness, clean-only aggregate-specific promotion, and
persisted engine/signature/scan-time metadata. The focused MinIO policy,
ClamAV entrypoint, and Compose policy suite passes 27/27; focused
configuration, object-store, API, worker, and scanner Go packages pass under
`-race`; the focused PostgreSQL upload and scan-pipeline selection passes
under `-race`; frontend HTTP transport passes 21/21; and frontend typecheck,
focused `go vet`, and diff checks pass.

After the host data volume was expanded to 24 GiB free, the previously blocked
real signed MinIO write passed without weakening the storage-safety threshold.
The private-object boundary returned 412 for a repeated signed PUT, 200 for a
signed GET, and 403 for an unsigned GET. A concurrency RED exposed multiple
winners in both direct writes and copies; the atomic conditional-write GREEN
allows exactly one winner for each 24-contender case.

A real ClamAV 1.4.3 adapter passed with signature version 28070. The combined
PostgreSQL, separately credentialed MinIO, ClamAV, and worker pipeline passed
clean exact-version promotion/download, valid PDF embedded-EICAR quarantine
and non-downloadability, immutable crash/retry recovery, and persisted
engine/signature/scan-time metadata. Live failure injection passed with
ClamAV unavailable and with a 250 ms scanner timeout. Restarting both MinIO and
ClamAV restored healthy service and the complete live adapter/pipeline
selection passed again.

The focused canonical HTTP Playwright scenario passed 1/1 through the browser,
API signed PUT, private MinIO, durable outbox, real ClamAV worker,
promotion/review, and closure path. Task-owned Compose projects, containers,
networks, volumes, temporary secrets, test binaries, Vite, Playwright, and
browser processes left zero residue. Spec-compliance self-review found no
remaining Critical or Important issue. Code-quality self-review found the
object overwrite TOCTOU defect and corrected it through the concurrency
RED → GREEN cycle; no Critical or Important issue remains. These are
main-agent self-reviews, not independent implementation reviews. The planned
Task 4 commit is `not run` because Git staging and commit authorization were
not granted.

### Task 5 — 24 July 2026

Immutable Gotenberg document rendering is `verified locally` through strict
RED → GREEN cycles. A server-owned versioned HTML template escapes source
content, supports Preliminary, Final, and Closure report kinds, embeds exact
report/source identities, and states explicitly that rendering does not
approve, sign, close, or confer legal validity. The adapter enforces a 1 MiB
immutable-source limit, 2 MiB HTML-request limit, bounded two-minute maximum
timeout, 32 MiB PDF-response limit, `%PDF-` signature, deterministic metadata,
and exact renderer/template/source SHA-256 provenance.

The worker writes a private non-overwriting generated-document object and
transactionally appends its immutable DocumentVersion, provenance, audit
event, authorized change, completion outbox, and job state. Crash-window
recovery compares exact bytes and metadata before finalization. The API lists
no signed URL, authorizes only the exact released clean version, issues a
five-minute signed URL on open, preserves Auditee organization isolation, and
supplies an exact signed Content-Disposition filename. HTTP mode never
fabricates a browser-local PDF; pending renders remain disabled.

The live Gotenberg/PostgreSQL/MinIO matrix passed four focused runs: all three
report kinds, provider-unavailable failure, 250 ms timeout failure, and all
three kinds again after provider restart. It verified private signed download,
exact PDF/renderer/template/source hashes, duplicate suppression, immutable
retry, and restart recovery. The canonical API/worker/browser scenario passed
1/1 with a real Gotenberg PDF, `%PDF-` bytes, exact report filename, and
Auditee-authorized private download. A fixture correction replaced three
prefix-only canonical report hashes with real SHA-256 digests after a focused
RED found 3 invalid versions; the corrected test passes.

Fresh focused regressions pass contract generation/drift 15/15, Gotenberg plus
Compose policy 22/22 and canonical Compose policy 20/20, frontend transport/UI
23/23, frontend typecheck, SQLC drift, focused Go `-race`, `go vet`, and diff
checks. Task-owned containers, volumes, networks, API/worker/Vite/Playwright
processes, and temporary files left zero residue. Spec-compliance self-review
found and corrected the unbounded render-request gap through a new RED → GREEN
cycle. Code-quality self-review found no remaining Critical or Important
issue. These are main-agent self-reviews, not independent implementation
reviews. The planned Task 5 commit is `not run` because Git staging and commit
authorization were not granted.

### Task 6 — 24 July 2026

Local SMTP notification delivery through Mailpit is `verified locally` through
strict RED → GREEN cycles. Server-authoritative recipient resolution uses the
active identity email and organization; an Auditee delivery must match the
notification organization. Separate bounded typed templates keep Auditee-safe
content structurally isolated from Internal CAA context and escape HTML. The
SMTP adapter uses authenticated plaintext only on the explicitly private local
network, preserves one stable Message-ID across retries, classifies permanent
and retryable failures without exposing provider responses or credentials, and
enforces bounded timeouts and message size.

Notification, outbox, audit, and delivery-job state remains transactional.
Delivery provenance records attempt count, accepted time, next retry, safe
failure code, terminal dead-letter state, and provider Message-ID. The default
three-attempt exponential retry is capped at 15 minutes. Due Soon, Due Today,
Overdue, status, duplicate suppression, permanent failure, recipient
isolation, and audit non-leak behavior are covered. The generated OpenAPI,
Go/TypeScript transports, HTTP backend, and Inspector/Lead and Executive
notification views expose normalized `PENDING`, `RETRYING`, `DELIVERED`, and
`FAILED` states while demo records remain explicitly `NOT_CONFIGURED`.

The real Mailpit scenario passed 1/1 with authenticated SMTP, exact recipient,
subject, and database Message-ID metadata, duplicate suppression, provider
stop/retry/restart recovery, two exact receipts after restart, and no
cross-organization or internal-content leak. The canonical HTTP browser
scenario passed 1/1 and rendered the server-authoritative Lead Inspector state
as `Email delivery: Pending` while the Auditee notification list remained
empty. Fresh regressions pass contract generation/drift 15/15, Compose policy
21/21, frontend 626/626, root/oracle 129/129, focused Go `-race`, `go vet`,
SQLC drift, frontend typecheck, docs smoke, and diff checks. Task-owned
containers, volumes, networks, API/Vite/Playwright processes, and browser
workers left zero residue.

Spec-compliance self-review found and corrected the missing live browser
delivery-state proof and HTTP-mode no-email copy through a new RED → GREEN
cycle. Code-quality self-review found no remaining Critical or Important
issue. These are main-agent self-reviews, not independent implementation
reviews. The planned Task 6 commit is `not run` because Git staging and commit
authorization were not granted.

### Task 7 — 24 July 2026

Runtime health, isolation, failure, and restart behavior are `verified locally`
through strict RED → GREEN cycles. Liveness remains downstream-independent.
Readiness probes seven safe named dependencies concurrently within bounded
timeouts: application, PostgreSQL, identity, MinIO, and ClamAV are required;
Gotenberg and Mailpit are optional. Required loss returns `503/not_ready`;
optional loss returns `200/degraded`; neither response exposes credentials or
provider errors.

Production API and scheduler configuration now validates only each process's
owned capabilities. The API discovers Keycloak through the private network
while preserving and explicitly validating the selected public HTTPS issuer;
the scheduler receives only its application database secret and network.
Startup waits for a one-shot, advisory-lock-protected migration. Runtime
containers declare 15-second stop grace, restart policy, two-CPU/two-GiB
limits, and 256-process limits. The exact stack wrapper owns only validated
`aviasurveil360-task-*` projects, scoped state, containers, networks, and
volumes.

The full failure matrix passed for PostgreSQL, Keycloak, MinIO, and ClamAV
loss/recovery with liveness retained and fail-closed readiness; Gotenberg and
Mailpit loss/recovery with degraded readiness; and a real worker-process crash
with automatic healthy restart. Only Caddy published a port, runtime network
membership exactly matched the reviewed topology, orphan containers were
zero, generated-secret log matches were zero, and final task-owned residue was
zero. MinIO credential-bearing administration output was suppressed after a
RED test caught an access-key leak; no secret value was retained in evidence.

Fresh checkpoint regressions pass the combined local
Compose/image/runtime/MinIO/Keycloak contract suite 47/47, Compose policy
21/21, runtime contract 6/6, generated OpenAPI contract 16/16, and affected Go
config/health/scanner/httpapi/identity/API packages under `-race`. The
production-like stack reached healthy state on a task-owned loopback port with
public-issuer/private-discovery separation, exact network membership, and zero
cleanup residue. Spec-compliance self-review found and corrected the missing
runtime exact-network proof. Code-quality self-review found and corrected
sequential readiness latency through a new RED → GREEN cycle; no Critical or
Important issue remains. These are main-agent self-reviews, not independent
implementation reviews. The planned Task 7 commit is `not run` because Git
staging and commit authorization were not granted.

### Task 8 — 25 July 2026

Clean demo and full Docker profiles are `verified locally` through strict
RED → GREEN cycles. The profile contracts pass 10/10. Both final scripts ran
twice from fresh scoped volumes with identical outcomes: each demo run proved
86/86 direct loads with Playwright 1/1 and skipped 0; each full run proved
86/86 HTTP direct loads, all 10 scenario families, Playwright 1/1 and skipped
0. Full mode used production-mode Keycloak `CONFIGURE_TOTP`, authorized
application provisioning, private MinIO, real ClamAV scan-clean Evidence
gating, authenticated Mailpit SMTP, immutable Gotenberg PDF rendering, and
normal worker restart/recovery. Each full run observed exactly one expected
Mailpit delivery.

Only Caddy published the browser-facing port. Full-mode `/__test/*` remained
404 and no mock, seed, deterministic scanner, fixture initializer, or
canonical-header authentication entered the artifact. Generated-secret scans
returned zero matches. Every final run removed its exact task-owned
containers, volumes, networks, browser workers, and scoped state with zero
residue.

Spec-compliance self-review found and corrected missing live Mailpit inspection
and incomplete demo partial-start cleanup through new RED → GREEN cycles.
Code-quality self-review found and corrected an unlabeled-volume cleanup gap
through a new RED → GREEN cycle; exact validated project-name fallback now
supplements Compose labels without a global prune. No Critical or Important
issue remains. These are main-agent self-reviews, not independent
implementation reviews. The planned Task 8 commit is `not run` because Git
staging and commit authorization were not granted.

### Task 9 — 25 July 2026

Local production-like service evidence is `verified locally`. The exact
required matrix passes: local Compose/image/runtime contracts 37/37, Compose
policy 21/21, eight runtime image builds, eight digest-bound CycloneDX SBOMs,
eight HIGH/CRITICAL vulnerability gates, two dependency audits with zero
vulnerabilities, and the complete Go suite under `-race`. The final Plan 1–2
regression selection passes contract 16/16, canonical examples 15/15, SQLC
drift, web 626/626 across 60 files, TypeScript typecheck, and root/oracle
108/108.

Two final clean demo runs each pass 86/86 direct loads with Playwright 1/1 and
skipped 0. Two final clean full runs each pass 86/86 HTTP direct loads, all 10
scenario families, Playwright 1/1, skipped 0, exactly one expected Mailpit
delivery, worker restart/recovery, gateway-only publishing, exact network
membership, zero generated-secret log matches, and zero task-owned residue.
Each profile now fails closed unless its runtime image tags match the exact
accepted SBOM/scan digests. This corrected a review finding where the previous
full-profile script rebuilt unscanned image identities after the scan gate.

The canonical evidence records immutable image, SBOM, scan, configuration,
Evidence-object, and Gotenberg PDF provenance hashes; Keycloak/TOTP behavior;
ClamAV 1.4.3/signature 28070 provenance; Mailpit receipts; failure/restart and
resource boundaries; and cleanup results. The one exact Keycloak
`CVE-2026-22020` source-digest exception remains owned, expiring, and tracked.
A root/oracle RED also found the new profile ledger entry outside the approved
classification enum; the minimal GREEN classifies the accepted production
slice as `first-production`. Final main-agent spec-compliance and code-quality
self-reviews found no remaining Critical or Important issue. The planned Task
9 commit and push are `not run` because neither Git action was authorized.

## Objective

Make local Docker the complete executable target for stakeholder demos and real
backend validation. A clean checkout with supported prerequisites must be able
to initialize local secrets, start either deterministic demo or full HTTP mode,
exercise all services, stop cleanly, and recover without silently substituting
mock behavior.

## Scope

- Build reproducible containers for the React demo artifact, React HTTP
  artifact, Go API, Go worker, and scheduled job runner.
- Route the full HTTP application through one local HTTPS origin at
  `https://localhost:8443`.
- Run Keycloak in production mode with its own PostgreSQL database, imported
  realm baseline, application-managed provisioning, TOTP MFA, revocation, and
  role/organization mapping.
- Run private MinIO buckets with versioning and quarantine/clean/generated
  object separation.
- Replace deterministic scan completion with real ClamAV scanning and signature
  readiness.
- Deliver notification emails through the SMTP adapter into Mailpit and expose
  auditable delivery/retry state.
- Render versioned PDFs through Gotenberg and store immutable outputs in MinIO.
- Generate runtime secrets into gitignored files, mount them as Docker secrets,
  and support SOPS + age for encrypted configuration.
- Add health/startup/readiness, network, resource, non-root, filesystem, and
  cleanup gates.
- Generate CycloneDX SBOMs for every built runtime image and fail closed on
  unresolved HIGH/CRITICAL container-image vulnerabilities before full-profile
  acceptance.
- Keep deterministic fixture/reset machinery in a one-shot `test` lane only.
  Normal `full` mode starts from fresh scoped volumes, provisions through
  authorized application/Keycloak flows, and exposes no `/__test/*` route.
- Run all 86 routes and scenario families against the full Compose profile.

## Assumptions

- Plans 1 and 2 are accepted: 86 routes work in demo and HTTP, and durable jobs
  already exist for provisioning, scan, email, and document work.
- Local developer prerequisites are Docker Desktop/Engine with Compose v2,
  Node only for host-side tests, Go only for host-side checks, and `age`/`sops`
  for encrypted configuration maintenance.
- Local HTTPS uses Caddy's internal CA. Trusting that CA in the user's operating
  system is an explicit user action; scripts may print instructions but cannot
  change system trust without approval.
- Mailpit is a real SMTP receiver for local verification, not production email.
- ClamAV signature readiness is required before Evidence upload/review becomes
  ready. A scanner outage fails closed.
- Gotenberg output is a real local PDF artifact but production signing and legal
  validity remain outside scope.

## Global Constraints

- No default password in the full profile and no plaintext runtime secret in
  Git, Compose YAML, container image layers, logs, or browser bundles.
- Images are immutable-digest locked in `deploy/local/image-lock.json`; the
  compose gate rejects an unpinned external image.
- Containers run as non-root where the upstream image supports it; writable
  paths and Linux capabilities are minimized and recorded for exceptions.
- Only Caddy publishes the full application's browser port. PostgreSQL,
  Keycloak, MinIO API, ClamAV, SMTP, and Gotenberg remain on internal networks;
  optional developer consoles bind to loopback under an explicit tools profile.
- HTTP build has no mock/seed/test-profile module path.
- Normal OIDC/full API configuration cannot register seed/reset handlers; direct
  `/__test/*` requests return 404. Test fixtures run only through a scoped
  one-shot command/container that is absent after initialization.
- Provider tokens remain server-side; application cookies remain Secure,
  HttpOnly, SameSite, bounded idle/absolute, and CSRF protected.
- Scan-clean exact Evidence version is required before review/download/closure.
- Worker retries are bounded, idempotent, observable, and never overwrite an
  immutable version.
- Do not add Kubernetes, cloud resources, external SMTP, external AI, external
  identity federation, or production cutover in this plan.
- Work on the current branch and preserve unrelated untracked paths.

## Ownership Boundaries

| Owner | Responsibility |
|---|---|
| Platform | Containers, Compose profiles, gateway, networks, secrets, image lock, health, resource constraints |
| Identity/Security | Keycloak realm, MFA, provisioning, session/revocation, credential and CA handling; image/SBOM vulnerability policy |
| Backend | Typed service adapters, job state, retries, authorization, scan/document/email/provisioning outcomes |
| Frontend | Same-origin runtime configuration, login/enrollment/error presentation, no mock fallback |
| QA | Clean-machine bring-up, full browser matrix, failure injection, restart, cleanup, raw artifact/secret scans |
| Records/Operations | Local evidence review only; production policy and provider approval remain later |

## Compose Topology

| Service | Profile | Published access | Persistence |
|---|---|---|---|
| `gateway` | `demo`, `full` | `127.0.0.1:8443` | Caddy local CA volume |
| `web-demo` | `demo` | internal | immutable image artifact |
| `web-http` | `full` | internal | immutable image artifact |
| `api` | `full` | internal | PostgreSQL/MinIO |
| `worker` | `full` | internal | PostgreSQL/MinIO |
| `scheduler` | `full` | internal | PostgreSQL |
| `fixture-init` | `test` only | none; one-shot | fresh test volumes only |
| `postgres` | `full`, `test`, `recovery` | internal | app database volume |
| `keycloak-postgres` | `full` | internal | identity database volume |
| `keycloak` | `full` | internal via `/identity` | identity database |
| `minio` | `full`, `test`, `recovery` | internal | primary object volume |
| `clamav` | `full` | internal | signature volume |
| `mailpit` | `full`, `tools` | UI loopback only in `tools` | optional message volume |
| `gotenberg` | `full` | internal | ephemeral working data |

The later observability and backup services belong to Plan 4 and join the same
network through separate profiles.

## Phases

1. **Secure runtime foundation — Tasks 1–2:** lock images/secrets/networks and
   containerize the application behind one local HTTPS origin.
2. **Real platform adapters — Tasks 3–6:** activate Keycloak MFA/provisioning,
   MinIO/ClamAV, Gotenberg, and Mailpit SMTP.
3. **Resilience and handoff — Tasks 7–9:** enforce health/failure/restart
   behavior, prove clean demo/full profiles, and record synchronized evidence.

---

### Task 1: Lock Images, Secrets, Profiles, And Network Policy

**Files**

- Create `deploy/local/compose.yaml`
- Create `deploy/local/image-lock.json`
- Create `deploy/local/compose-policy.json`
- Create `deploy/local/secrets/README.md`
- Create `deploy/local/config/application.example.yaml`
- Create `deploy/local/config/application.enc.yaml`
- Create `.sops.yaml`
- Create `scripts/init-local-secrets.sh`
- Create `scripts/check-compose-policy.sh`
- Modify `.gitignore`
- Create `tests/local-compose-policy.test.mjs`

**Interfaces**

`init-local-secrets.sh` writes random local credentials only under
`.local/aviasurveil360/secrets/`, sets restrictive permissions, and refuses to
overwrite without an explicit rotate flag. Compose mounts named secret files;
environment variables contain only non-secret configuration and secret paths.

- [x] Write failing policy tests for plaintext secret values, missing secret
  mounts, external images without digests, published internal ports, shared
  app/identity database, unrestricted networks, root user, writable rootfs
  exceptions without reasons, and absent health checks.
- [x] Run `node --test tests/local-compose-policy.test.mjs`; confirm named
  topology/policy failures.
- [x] Implement the Compose skeleton, profiles, internal networks, volumes,
  encrypted config, secret generator, image-lock resolver, and policy checker.
  Resolve and review current upstream digests during execution, then record
  their immutable values in `image-lock.json`.
- [x] Run Compose config for `demo`, `full`, `test`, and `recovery`, SOPS decrypt
  validation, secret-file scan, and policy tests; expect zero published internal
  services and zero plaintext credentials.
- [ ] Commit exactly `build(local): define secure compose topology`. `not run`
  because this task does not authorize staging or commit actions.

### Task 2: Containerize Web, API, Workers, And The HTTPS Gateway

**Files**

- Create `apps/web/Dockerfile`
- Create `apps/api/Dockerfile`
- Create `deploy/local/gateway/Dockerfile`
- Create `deploy/local/gateway/Caddyfile`
- Create `deploy/local/gateway/security-headers.caddy`
- Create `deploy/local/api/entrypoint.sh`
- Create `deploy/local/worker/entrypoint.sh`
- Create `deploy/local/scheduler/entrypoint.sh`
- Modify `deploy/local/compose.yaml`
- Create `scripts/build-local-images.sh`
- Create `scripts/generate-image-sboms.sh`
- Create `scripts/scan-local-images.sh`
- Create `tests/local-image-security-policy.test.mjs`
- Create `tests/local-image-boundary.test.mjs`

**Interfaces**

The web image has separate immutable `demo` and `http` targets. The Go image has
`api`, `worker`, `scheduler`, and migration targets built from one module. Caddy
serves `/`, proxies `/api` and `/auth` to Go, and proxies `/identity` to
Keycloak under the one external origin.
Every runtime image is identified by digest, has a reviewed CycloneDX SBOM, and
passes the configured HIGH/CRITICAL vulnerability policy. A finding may be
accepted only through an explicit owner, expiry, rationale, and tracker record.

- [x] Write failing image/boundary tests for root runtime/mock input in HTTP,
  build tools in runtime layers, root user, missing read-only rootfs, missing
  startup/readiness, non-reproducible build metadata, wrong same-origin routes,
  missing SBOM, unscanned digest, and unapproved HIGH/CRITICAL findings.
- [x] Build targets and confirm expected Dockerfile/config red failures.
- [x] Implement multi-stage builds, non-root entrypoints, migrations-before-api
  startup lock, Caddy HTTPS/routing/security headers, CSP, immutable assets,
  compression, bounded shutdown, image SBOM generation, digest-bound scanning,
  and a fail-closed vulnerability policy.
- [x] Start demo then full gateway from clean volumes; verify HTTPS artifact,
  `/health/*`, no mock input in HTTP, no `/__test/*` route in full mode, SIGTERM
  drain, SBOM/vulnerability results, and image boundary tests.
- [ ] Commit exactly `build(local): containerize application runtime`. `not run`
  because this task does not authorize staging or commit actions.

### Task 3: Activate Production-Mode Keycloak, MFA, And Provisioning

**Files**

- Replace `deploy/local/keycloak/realm.json` with generated/reviewed source under
  `deploy/local/keycloak/realm-source.json`
- Create `deploy/local/keycloak/build-realm.mjs`
- Create `deploy/local/keycloak/realm-contract.test.mjs`
- Create `apps/api/internal/identity/keycloak_admin.go`
- Create `apps/api/internal/identity/keycloak_admin_test.go`
- Extend `apps/api/internal/administration/`
- Modify `apps/api/internal/platform/config/config.go`
- Modify `deploy/local/compose.yaml`
- Create `apps/web/tests/e2e/oidc-mfa-provisioning.spec.ts`

**Interfaces**

Keycloak starts in production mode with proxy/hostname/health settings and a
separate database. Application provisioning creates/deactivates users, maps
approved realm roles and organization attributes, requires TOTP configuration
on first login, records external subject, and revokes sessions on disable.

- [x] Write failing realm, Go adapter, and browser tests for exact clients/
  redirect URIs, PKCE, no self-registration, no password grant, required TOTP,
  admin provisioning authorization, duplicate email, role/org mapping,
  deactivation, revocation, and safe application session projection.
- [x] Run realm/Go/OIDC tests; confirm production-mode/provisioning gaps fail.
- [x] Implement deterministic realm generation, Keycloak Admin API adapter,
  outbox handler, status reconciliation, MFA enrollment/required action, role
  mapping, audit events, and failure/retry presentation.
- [x] Run complete login, first-login TOTP, logout, expiry, CSRF, create/disable/
  revoke, wrong-role, wrong-org, restart, and secret/log scans.
- [ ] Commit exactly `feat(identity): activate local mfa provisioning`. `not run`
  because this task does not authorize staging or commit actions.

### Task 4: Activate Private Object Storage And Real Malware Scanning

**Files**

- Create `deploy/local/minio/init.sh`
- Create `deploy/local/minio/bucket-policy-contract.test.mjs`
- Create `apps/api/internal/platform/scanner/scanner.go`
- Create `apps/api/internal/platform/scanner/clamav.go`
- Create `apps/api/internal/platform/scanner/clamav_test.go`
- Extend `apps/api/internal/worker/evidence/worker.go`
- Modify `apps/api/internal/evidence/upload_service.go`
- Modify `apps/api/internal/inspections/attachments/upload_service.go`
- Modify `deploy/local/compose.yaml`
- Create `apps/api/tests/integration/clamav_object_pipeline_test.go`

**Interfaces**

Buckets are private and separated as `evidence-quarantine`, `evidence-clean`,
`inspection-attachments`, and `generated-documents`. Object keys never
overwrite. ClamAV receives exact bytes, records engine/signature version and
scan time, and promotes only clean exact versions. Infected/error/timeout remain
quarantined and non-downloadable.

- [x] Write failing policy/integration tests for public access, overwrite,
  wrong bucket, MIME/size/hash mismatch, EICAR detection, clean promotion,
  scanner unavailable/timeout, stale signature readiness, download/review
  denial, retry idempotency, and immutable version history.
- [x] Run focused tests; confirm absent real scanner/bucket policies.
- [x] Implement MinIO initialization, least-privilege service credentials,
  versioning, scanner adapter, worker handler, health/readiness, quarantine and
  promotion transaction semantics.
- [x] Run real clean/infected/error/timeout/restart cases through browser, API,
  ClamAV, worker, and MinIO; expect no infected/public/downloadable object.
- [ ] Commit exactly `feat(evidence): activate local malware scanning`. `not
  run` because this task does not authorize staging or commit actions.

### Task 5: Activate Gotenberg Document Rendering

**Files**

- Create `apps/api/internal/documents/gotenberg_renderer.go`
- Create `apps/api/internal/documents/gotenberg_renderer_test.go`
- Create `apps/api/internal/documents/templates/`
- Create `apps/api/internal/documents/template_contract_test.go`
- Extend the document worker command
- Modify `deploy/local/compose.yaml`
- Create `apps/api/tests/integration/gotenberg_document_pipeline_test.go`
- Create `apps/web/tests/e2e/generated-document.http.spec.ts`

**Interfaces**

Rendering consumes a versioned server-owned HTML/template/data snapshot and
returns PDF bytes plus renderer/template/source hashes. The worker stores a new
immutable generated-document version and never treats rendering as report
approval or signature.

- [x] Write failing template/adapter/integration/browser tests for escaped
  content, deterministic metadata, exact source/report version, PDF signature,
  private object, duplicate job, renderer timeout/error, retry, download scope,
  and no legal/e-signature claim.
- [x] Run focused tests and confirm missing adapter/templates.
- [x] Implement approved templates, Gotenberg adapter, bounded request, worker
  handler, immutable object/metadata, audit event, and UI status/download.
- [x] Run real render/download/retry/restart for Preliminary, Final, and Closure
  report versions; expect exact version/hash evidence.
- [ ] Commit exactly `feat(documents): activate local pdf rendering`. `not run`
  because this task does not authorize staging or commit actions.

### Task 6: Activate SMTP Delivery Through Mailpit

**Files**

- Create `apps/api/internal/notifications/smtp_sender.go`
- Create `apps/api/internal/notifications/smtp_sender_test.go`
- Create `apps/api/internal/notifications/templates/`
- Create `apps/api/internal/notifications/template_contract_test.go`
- Extend notification worker and scheduler commands
- Modify `deploy/local/compose.yaml`
- Create `apps/api/tests/integration/mailpit_notification_pipeline_test.go`
- Create `apps/web/tests/e2e/notification-delivery.http.spec.ts`

**Interfaces**

Recipient resolution is server-authoritative and organization-scoped. Templates
receive bounded typed data and separate CAA-internal from Auditee-safe content.
Outbox delivery records message ID, attempt, accepted/failed state, next retry,
and audit event without logging body or credentials.

- [x] Write failing tests for Due Soon/Overdue/status notifications, recipient
  isolation, template escaping, forbidden internal content, duplicate
  suppression, SMTP refusal/timeout/retry, permanent failure, Mailpit receipt,
  and visible in-app delivery state.
- [x] Run focused tests and confirm missing SMTP/templates.
- [x] Implement templates, SMTP adapter, worker/scheduler handlers, retry policy,
  dead-letter state, audit events, and Mailpit tools profile.
- [x] Run real Mailpit delivery/failure/restart scenarios and query exact message
  metadata through its local API; expect no duplicate or privacy leak.
- [ ] Commit exactly `feat(notifications): activate local email delivery`.
  `not run` because this task does not authorize staging or commit actions.

### Task 7: Enforce Runtime Health, Isolation, Failure, And Restart Behavior

**Files**

- Extend `apps/api/internal/httpapi/health.go`
- Extend `apps/api/internal/httpapi/health_test.go`
- Create `apps/api/internal/platform/health/dependencies.go`
- Modify all Compose health checks and dependency conditions
- Create `scripts/local-stack.sh`
- Create `scripts/check-local-runtime.sh`
- Create `tests/local-runtime-contract.test.mjs`
- Create `apps/web/tests/e2e/local-service-failures.http.spec.ts`

**Interfaces**

Liveness never depends on downstream services. Startup waits for migrations and
required configuration. Readiness reports named required dependencies without
secrets. Optional delivery failures degrade only their capabilities; identity,
database, and required scan readiness fail closed where needed.

- [x] Write failing tests for dependency loss, restart order, migration lock,
  stale ClamAV signature, Keycloak loss, MinIO loss, Gotenberg loss, SMTP loss,
  worker crash, bounded shutdown, orphan container, and secret/log leakage.
- [x] Run runtime contracts and confirm current topology cannot meet them.
- [x] Implement dependency probes, degraded/readiness semantics, restart policy,
  resource limits, stop grace periods, one-shot migration, stack wrapper, and
  exact cleanup ownership.
- [x] Execute failure injection and restart matrix without deleting user data;
  verify recovery, visible capability states, no cross-network access, and no
  leftover task-owned processes/containers.
- [ ] Commit exactly `test(local): enforce runtime resilience boundary`.
  `not run` because this task does not authorize staging or commit actions.

### Task 8: Prove Clean Demo And Full Local Docker Profiles

**Files**

- Create `scripts/test-local-demo-profile.sh`
- Create `scripts/test-local-full-profile.sh`
- Create `apps/web/tests/e2e/local-full-platform.spec.ts`
- Modify `apps/web/playwright.config.ts`
- Modify `tests/parity/behavior-ledger.json`
- Modify `MANIFEST.md`

**Interfaces**

Both scripts create unique Compose project names, use fresh scoped volumes,
trap only task-owned resources, emit a machine-readable summary, and fail if a
required test/project is skipped. Full mode uses normal OIDC/MFA and real local
service adapters; it never enables canonical-header auth, deterministic scan,
test reset routes, or test-profile fixture handlers. Its initial state is
created through normal authorized provisioning/application commands. The
one-shot `fixture-init` lane belongs only to the separate `test` profile.

- [x] Write failing script contract tests for reused project name, shared fixed
  volume, missing trap, skipped Playwright project, mock import, deterministic
  scanner, registered `/__test/*` route, test fixture container in full mode,
  direct internal port, or missing cleanup assertion.
- [x] Run script contracts and confirm expected failures.
- [x] Implement clean demo/full orchestration, seed/provision flow, browser
  profiles, 86 direct loads, 10 scenarios, service adapters, artifact checks,
  restart/failure cases, and cleanup summaries. Full-mode setup must use fresh
  volumes plus normal provisioning/application commands, never a reset API.
- [x] Run both exact scripts from clean state twice; expect identical outcome,
  86 demo and 86 HTTP routes, real MFA/scan/email/PDF evidence, and zero residue.
- [ ] Commit exactly `test(local): prove complete docker profiles`.
  `not run` because this task does not authorize staging or commit actions.

### Task 9: Record Local Production-Like Service Evidence

**Files**

- Create `docs/demo-evidence/LOCAL_PRODUCTION_LIKE_SERVICES_2026-07-22.md`
- Modify `docs/demo-evidence/BUILD_SUMMARY.md`
- Modify `docs/index.md`
- Modify `MANIFEST.md`
- Modify `docs/exec-plans/index.md`
- Modify `docs/exec-plans/tech-debt-tracker.md`
- Modify this plan
- Create `scripts/check-local-image-evidence.sh`
- Modify `scripts/test-local-demo-profile.sh`
- Modify `scripts/test-local-full-profile.sh`
- Modify `tests/local-profile-contract.test.mjs`
- Modify `tests/parity/behavior-ledger.json`

- [x] Run clean image build, Compose policy, secret scan, demo/full profiles,
  normal MFA OIDC, provisioning, real scan, Mailpit, Gotenberg, failure/restart,
  dependency audits, digest-bound image SBOM/vulnerability gates, absence of
  full-profile test routes, and cleanup from fresh scoped volumes.
- [x] Record immutable image digests, configuration hashes, exact test counts,
  Keycloak realm/MFA behavior, scanner signature/engine, SMTP receipts, PDF and
  object hashes, failure results, timings, resource observations, and cleanup.
- [x] Keep literal labels: `verified locally`, `candidate-only`, and
  `release pending`; do not claim production deployment or production-ready.
- [x] Set this plan `ready-for-verification` only after all local service gates
  pass and Plans 1–2 remain green.
- [ ] Commit exactly `docs(evidence): record local production services` and push
  only when explicitly authorized.

## Required Verification Matrix

```bash
node --test tests/local-compose-policy.test.mjs tests/local-image-boundary.test.mjs tests/local-runtime-contract.test.mjs
./scripts/check-compose-policy.sh
./scripts/build-local-images.sh
./scripts/generate-image-sboms.sh
./scripts/scan-local-images.sh
./scripts/test-local-demo-profile.sh
./scripts/test-local-full-profile.sh
npm --prefix apps/web audit
npm --prefix apps/web audit --omit=dev
GOCACHE=/private/tmp/aviasurveil360-local-services-go-cache go -C apps/api test -race -p 1 -count=1 ./...
```

Expected final result: clean reproducible demo and full profiles, one HTTPS
origin, 86 full HTTP routes, normal MFA OIDC, real ClamAV/Mailpit/Gotenberg/
MinIO behavior, no plaintext secrets, no published internal services, and zero
task-owned residue.

## Risks And Controls

| Risk | Control |
|---|---|
| Compose becomes an unmaintainable production substitute | Profiles, policy tests, one ownership file, AWS remains separate Plan 4 |
| Local credentials leak | Generated gitignored secret files, Docker secrets, SOPS+age, artifact/log scans |
| Keycloak dev mode weakens evidence | Production-mode startup, separate DB, MFA and provisioning browser tests |
| Infected Evidence becomes reviewable | Quarantine, exact-version scan, fail-closed readiness/download/review tests |
| Mail or PDF retries duplicate versions | Durable job idempotency and output/message identity constraints |
| Gateway bypass exposes internal services | Internal networks, no published ports, same-origin browser tests |
| Image tags drift | Digest lock and compose policy gate |
| A pinned image contains a known critical vulnerability | Digest-bound SBOM and image scan, fail-closed HIGH/CRITICAL policy, explicit expiring exception record only |
| Test reset authority leaks into the full stack | Fresh volumes and normal provisioning in full mode; one-shot test-only fixture lane; `/__test/*` 404 contract |
| Cleanup stops user resources | Unique Compose project names and task-owned trap assertions |

## Dependencies

- Plans 1 and 2 accepted with 86 demo/HTTP routes and durable worker jobs.
- Docker Engine/Desktop and Compose v2.
- User-approved local CA trust if warning-free interactive HTTPS is required.
- Plan 4 adds observability, backup, RPO/RTO, DR, Terraform, and Terragrunt.

## Out Of Scope

- Production external IdP federation, SCIM, SMS/hardware MFA, external email
  provider, legal e-signature, external malware service, or external AI.
- Monitoring/alerts, backup/DR objectives, Terraform/Terragrunt/AWS deployment,
  public DNS, traffic cutover, or production on-call.
- Automatic modification of the user's operating-system trust store.

## Execution Prompt

```text
Execute docs/exec-plans/completed/2026-07-22-local-production-like-services-plan.md task by task with superpowers:executing-plans only after the 86-screen React and full backend parity plans are complete, ready-for-verification, and independently accepted. Do not overlap unfinished Plan 1 or Plan 2 work. Do not dispatch subagents unless explicitly authorized. Work on the current branch and preserve unrelated .superpowers/, docs/demo-evidence/stakeholder/, and outputs/ content.

Build a production-like local Docker Compose system with one Caddy HTTPS origin, separate React demo/HTTP artifacts, Go API/worker/scheduler, separate app and Keycloak PostgreSQL databases, Keycloak production-mode TOTP MFA and application provisioning, private MinIO, real ClamAV, Mailpit SMTP, and Gotenberg PDF rendering. Use Docker secrets and SOPS+age; never commit or log plaintext credentials. Pin all external images by reviewed immutable digest.

Keep server authority, Auditee privacy, immutable versions, scan-clean Evidence gating, idempotent jobs, and exact audit events. Full mode must never import mock/seed, register /__test reset routes, run the test fixture initializer, or enable canonical-header authentication/deterministic scanning. Start it from fresh volumes and use normal authorized provisioning/application commands. Generate digest-bound image SBOMs and fail closed on unresolved HIGH/CRITICAL image findings. Test real failure, timeout, retry, restart, health/readiness, and cleanup behavior.

Use the plan's TDD order and exact commit messages only when Git actions are separately authorized. Inspect upstream, allowlist, cached names, full cached diff, and diff check before every commit. Do not add cloud resources, modify system trust without approval, deploy, or claim production readiness. Finish with synchronized local-service evidence and stakeholder verification as the next todo.
```
