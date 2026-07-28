# Local Preprod Release Candidate Plan

**Status:** `blocked`

**Reviewed task count:** 10

**Objective:** Turn the existing production-like local platform into a
repeatable local preprod release-candidate environment and qualify the complete
application, identity, data-feed, security, performance, observability, and
recovery behavior with realistic synthetic data.

**User-visible outcome:** A stakeholder can start one documented local preprod
environment, sign in as every real Keycloak-backed role, inspect realistic
connected records across all 86 routes, execute complete workflows, and
observe truthful failures, notifications, analytics delivery, and recovery.

**Dependencies:**

- Identity And Realistic Data Foundation is accepted.
- AviaCore And ML Data Readiness has passed its local producer conformance and
  sandbox integration gates.
- Plan 1 visual dispositions are recorded and all required fixes are verified.
- Plans 2-4 remain locally green after successor changes.

## Scope

- A normal-HTTP, normal-OIDC local preprod profile using immutable local image
  digests.
- Real local Keycloak account lifecycle and the accepted synthetic workload
  profiles.
- AviaSurveil API, web, worker, scheduler, data-feed worker, migrations,
  PostgreSQL, Keycloak/PostgreSQL, object storage, malware scanning, SMTP,
  document rendering, observability, alerting, backup, restore, and AviaCore
  local sandbox integration.
- Full role/route/action/workflow coverage.
- Load, spike, concurrency, soak, failure, restart, recovery, privacy,
  accessibility, DAST, dependency/container/IaC, and stakeholder UAT gates.
- A literal local release-candidate evidence package.

## Explicit Exclusions

- AWS or any other paid/remote infrastructure.
- Production customer data, production identity federation, public DNS, public
  traffic, or production email.
- Any test-profile, canonical-header authentication, mock backend, normal API
  reset/seed route, or deterministic fake external service in the accepted
  preprod run.
- Claims of high availability, host-loss recovery, cloud capacity, staffed
  operations, source-bound AviaCore canary, controlled pilot, or
  `production-ready`.
- Automatic legal, enforcement, certificate, closure, or model decision.

## Local Readiness Label

This plan can produce only:

> `local preprod release candidate; verified locally`

It remains `candidate-only` and `release pending`. The label means the
application and artifacts passed the approved local matrix. It does not mean
the deployment environment, operations organization, legal/records decisions,
or cloud capacity are production-ready; production-readiness is not claimed.

## Provisional Local Engineering Targets

These targets qualify local behavior and are not production SLOs:

| Gate | Target |
|---|---|
| Acceptance load | 100 concurrent interactive sessions for 30 minutes |
| Read API latency | p95 at or below 500 ms under acceptance load |
| Mutation API latency | p95 at or below 1,000 ms under acceptance load |
| HTTP error rate | at or below 0.1%, excluding intentional negative tests |
| Spike | 500 concurrent sessions for 5 minutes, no acknowledged data loss |
| Spike recovery | queues and error rate return to baseline within 5 minutes |
| Soak | 12 hours at 100 concurrent sessions using `realistic` data |
| Process growth | no unexplained monotonic growth; RSS growth at or below 10% after warm-up |
| Data-feed acknowledgement | p95 at or below 10 seconds while AviaCore is healthy |
| Local DR | database/object RPO at or below 120 seconds and RTO at or below 120 seconds |

Every run records Mac model, CPU, memory, Docker resources, storage, OS,
browser, network shape, image digests, and workload digest. A material
environment change invalidates comparisons rather than silently moving the
target.

## Ownership Boundaries

| Surface | Accountable owner |
|---|---|
| Role/route/action/workflow truth and UAT | Product/CAA Operations + QA |
| OCI artifacts, source provenance, local PKI and runtime exposure | Platform + Security |
| Keycloak/OIDC/MFA/account lifecycle | Identity + Security |
| Loader profiles and source-to-AviaCore reconciliation | Domain/Data + AviaCore contract governance |
| Load/spike/soak workload and host envelope | Performance + Platform |
| Accessibility matrix and manual AT evidence | Accessibility + QA |
| Alerts, recovery clocks, backup/restore and runbooks | Operations/DBA |
| Local GO/NO-GO and later-plan authorization | Release authority / user |

## Progress

- [x] (2026-07-27) Local-first release-candidate approach approved; AWS
  intentionally deferred.
- [x] (2026-07-27) Existing production-like and reliability/DR evidence
  inspected.
- [x] (2026-07-27) Independent architecture, security, artifact, performance,
  accessibility, and recovery plan review completed. It added OCI artifact
  identity, trusted local TLS, environment-bound Playwright/accessibility,
  exhaustive role-route-action coverage, four-profile qualification, concrete
  aggregate commands, and per-store recovery clocks. Runtime verification
  remains `not run`.
- [ ] Predecessor implementation and acceptance.

## Tasks

### Task 1: Freeze The Local Preprod Profile And Acceptance Manifest

**Files**

- Create `docs/operations/LOCAL_PREPROD_PROFILE.md`.
- Create `deploy/local/preprod-policy.json`.
- Create `tests/local-preprod-profile-contract.test.mjs`.
- Create `docs/product-specs/screens/local-preprod-route-role-action-matrix.json`
  from the canonical 86-screen, eight-role, visible-action, and workflow
  authorities.
- Create `scripts/test-local-preprod-profile.sh`.
- Create `scripts/init-local-preprod-pki.sh`,
  `scripts/cleanup-local-preprod-pki.sh`, trusted/untrusted certificate
  fixtures, and isolated browser/CLI trust-store configuration.
- Modify `deploy/local/gateway/Caddyfile` and gateway policy tests.
- Modify `scripts/local-stack.sh` and `deploy/local/compose.yaml` only where the
  accepted profile requires it.
- Modify the local operations index.

**Work**

- [ ] Define exact required, optional, one-shot, and forbidden services,
  networks, volumes, secrets, ports, health checks, dependencies, and startup
  order.
- [ ] Reuse the existing full-profile services and policies; add only the
  identity/data/feed successors absent from the current topology.
- [ ] Require normal OIDC, real Keycloak, real PostgreSQL, private versioned
  object storage, real ClamAV, authenticated SMTP/Mailpit, Gotenberg, and the
  accepted AviaCore local receiver/data platform.
- [ ] Permit the preprod data loader only as a named one-shot operation before
  acceptance tests. It must not remain running.
- [ ] Publish only the one browser-facing HTTPS gateway port. Keep databases,
  object storage, identity administration, telemetry backends, and internal
  services private.
- [ ] Freeze a task-owned local PKI contract: CA identity/custody, hostname and
  SAN, trust install/removal, allowed TLS versions/ciphers, expiry/overlap
  rotation, HTTPS-only/HSTS/cookie behavior, and negative wrong-SAN, expired,
  revoked/old-CA tests. Because only one HTTPS port is published, make no
  HTTP-to-HTTPS redirect claim.
- [ ] Use an isolated task-owned browser/CLI trust store rather than mutating a
  host/global trust store. Acceptance forbids
  `ignoreHTTPSErrors`, insecure curl, and global trust residue.
- [ ] Forbid mock imports, canonical-header auth, test/reset routes, mutable
  image tags, host-mounted source, default passwords, anonymous services, and
  unbounded resources.
- [ ] Define exact clean-start, preserve-data, reset-loader-owned-data,
  backup-before-reset, and complete cleanup commands.
- [ ] Freeze one machine-readable expected result for every role × route
  combination (allow, redirect, 403, or indistinguishable 404), every visible
  enabled/disabled control, all 10 canonical scenario families, identity/
  administration, external-service, offline, feed, degraded, and recovery
  states. Zero skip is the acceptance policy.
- [ ] Freeze an ordered four-profile run ledger with exact pre/post
  fingerprints and whole-namespace reset/reload commands:
  clean `smoke` qualification; clean `acceptance` bootstrap/functional/
  integration; clean `realistic` load/spike; clean `stress`
  data/query/worker/feed-drain; clean `realistic` reload for soak/DR; then clean
  `acceptance` reload for security/accessibility/UAT. Mixed-profile state or
  cross-profile residue fails.

**Verification**

Run:

    node --test tests/local-preprod-profile-contract.test.mjs
    docker compose -f deploy/local/compose.yaml config
    ./scripts/test-local-preprod-profile.sh
    git diff --check

Expected: the profile contract fails on every forbidden mutation and exposes
only the gateway.

**Acceptance**

- One machine-readable policy defines what local preprod is.
- The profile is normal application runtime plus explicit local dependencies,
  not a test harness disguised as preprod.
- The trusted HTTPS/OIDC origin and exhaustive coverage manifest are exact,
  versioned inputs to every later browser, load, security, and UAT run.

### Task 2: Build And Bind Immutable Local Release Artifacts

**Files**

- Modify `scripts/build-local-images.sh`.
- Modify `scripts/generate-image-sboms.sh`.
- Modify `scripts/scan-local-images.sh`.
- Modify `scripts/check-local-image-evidence.sh`.
- Modify `deploy/local/image-lock.json`.
- Create one versioned OCI promotion-manifest schema shared by local and AWS
  evidence tooling.
- Add contract tests for new runtime image roles.

**Work**

- [ ] Build project-owned web/gateway, API, worker, scheduler, migration,
  data-feed worker, Keycloak customization, and approved one-shot loader
  artifacts once from the exact source state. Pin third-party PostgreSQL,
  Keycloak base, ClamAV, Mailpit, Gotenberg, load, and DAST subjects by upstream
  registry digest; do not claim those upstream images were built here.
- [ ] Separate runtime and one-shot images so the loader cannot be selected as
  the API/worker command.
- [ ] Bind every image digest to its Dockerfile/input digest, dependency lock,
  CycloneDX SBOM, HIGH/CRITICAL scan result, and source tree state.
- [ ] Export every accepted image as a single-platform OCI layout/archive and
  designate the single-platform OCI image-manifest digest as the canonical
  promotion subject. Record platform, optional transport index-to-manifest
  mapping, config digest, manifest digest, archive SHA-256,
  source-tree/content bundle digest, builder identity, build inputs,
  reproducibility result, SBOM/scan/attestation subject digests, and any
  signature status. Never compare a Docker config ID with a registry manifest
  digest.
- [ ] Decide the AWS promotion source boundary: either a separately authorized
  clean committed source plus content digest, or a signed content-addressed
  source bundle followed by a new accepted build. A dirty local candidate may
  be locally qualified but cannot satisfy an AWS "exact source commit" gate by
  omission.
- [ ] Reject mutable tags, rebuilt digest drift, missing SBOM/scan evidence,
  stale exception, unexpected binary, test-profile package, or mock/seed input.
- [ ] Scan npm production/development dependencies, Go modules, containers,
  local Compose, and any added DAST/load-tool images.
- [ ] Preserve the existing time-bounded Keycloak advisory handling only if it
  is still current; expiry or changed evidence fails closed.
- [ ] Run fail-fast static security, secret, licence, provenance, and normal-
  artifact seed/test exclusion checks before any 12-hour or high-volume gate.

**Verification**

Run:

    ./scripts/build-local-images.sh
    ./scripts/generate-image-sboms.sh
    ./scripts/scan-local-images.sh
    ./scripts/check-local-image-evidence.sh
    npm --prefix apps/web audit
    npm --prefix apps/web audit --omit=dev

Expected: every accepted runtime digest has a complete matching evidence chain
and no unapproved HIGH/CRITICAL finding.

**Acceptance**

- The exact images tested later are the exact images recorded in the local
  release candidate.
- The promotion manifest uses OCI subject identity consistently and is
  consumable by the AWS publication verifier without rebuilding or changing
  manifest bytes/media type.

### Task 3: Prove Clean Install, Migration, Identity, And Data Bootstrap

**Files**

- Create `scripts/test-local-preprod-bootstrap.sh`.
- Add Playwright bootstrap/first-login coverage.
- Modify operations runbooks and secret initialization only where required.

**Work**

- [ ] Start from no application, identity, object, or AviaCore data volume and
  run forward migrations exactly once.
- [ ] Install only the task-owned local CA trust, prove the HTTPS gateway and
  OIDC issuer/redirect/cookie chain without TLS bypass, and remove that trust
  during scoped cleanup.
- [ ] Run `smoke` against its clean namespace, reconcile it, dispose it, then
  create a separate clean `acceptance` namespace for bootstrap and Tasks 4-5.
- [ ] Create the application Admin through the approved bootstrap boundary,
  then provision all accepted role accounts through the application lifecycle.
- [ ] Complete real first-login required actions and TOTP for representative
  internal and Auditee users without persisting credentials in evidence.
- [ ] Invoke the one-shot `acceptance` loader and verify its exact manifest,
  counts, object metadata, account bindings, and zero running loader service.
- [ ] Restart the complete stack and prove identity, MFA, sessions, data,
  objects, AviaCore checkpoints, and audit history persist.
- [ ] Repeat on an N-1 schema fixture and prove the supported forward upgrade.
- [ ] Prove normal startup with an empty database remains clean and does not
  auto-seed.
- [ ] Before functional execution, run an authenticated exposure/secret/header/
  cookie/CSRF/CORS/rate/upload preflight and stop on any Critical/High finding.

**Verification**

Run:

    ./scripts/init-local-preprod-pki.sh <release-run-id>
    ./scripts/test-preprod-data-profile.sh smoke
    ./scripts/test-local-preprod-bootstrap.sh

Expected: clean install, N-1 upgrade, first login, data load, persistence, and
empty-normal startup all pass with zero secret matches, TLS bypass, global
trust mutation, or cross-profile residue in artifacts. The isolated
release-run trust store and certificates remain active for Tasks 4-9 and are
removed only by Task 10's final scoped cleanup.

**Acceptance**

- Local preprod is reproducible without manual database edits or hidden user
  creation.

### Task 4: Execute The Complete Role, Route, Action, And Workflow Matrix

**Files**

- Create `apps/web/tests/e2e/local-preprod-functional.spec.ts`.
- Modify `apps/web/playwright.config.ts` and `apps/web/package.json` to add
  explicit `local-preprod` functional and accessibility projects/scripts.
- Extend shared scenario and visible-action contracts without duplicating
  business logic.
- Create a run-scoped evidence manifest and screenshots.

**Work**

- [ ] Authenticate all eight roles with real sessions and verify exact landing,
  navigation, organization, role, and logout behavior.
- [ ] Execute the canonical role × route matrix and record the expected allow,
  redirect, 403, or indistinguishable 404 result for every pair, including raw
  API list, count, search, pagination, and direct-ID negative cases.
- [ ] Direct-load all 86 routes at desktop, tablet, and mobile.
- [ ] Inventory and execute every visible enabled control. Disabled controls
  must expose the exact accepted reason.
- [ ] Execute planning, assignment, checklist, Potential Finding, Finding, CAP,
  Evidence, report, communications, notification, closure, correction,
  supersession, reopen, and administration lifecycles.
- [ ] Exercise empty, overdue, returned, rejected, unavailable, retrying,
  partially closed, not closed, and authorized-closed states.
- [ ] Verify raw JSON and rendered organization isolation, Internal CAA Note
  exclusion, private risk exclusion, exact record identity, immutable versions,
  and CAP-is-not-closure.
- [ ] Rerun the stakeholder-approved visual set and prove no unresolved
  Critical visual/interaction defect.
- [ ] Use the trusted HTTPS base URL and real OIDC setup with no Playwright
  `webServer`, mock backend, stored canonical-header session, TLS-ignore, or
  skipped test.

**Verification**

Run:

    npm --prefix apps/web run test:e2e:local-preprod:functional

Expected:
86/86 direct loads per viewport, complete action execution, zero unexpected
console/page errors, every role-route result, every enabled/disabled-control
result, all 10 scenario families and named edge states, zero skip, zero
authority/privacy failure, and exact scenario transcripts.

**Acceptance**

- Every supported user workflow is executable against the normal local preprod
  backend with the clean deterministic `acceptance` profile.

### Task 5: Qualify External Services, Offline Recovery, And AviaCore Delivery

**Files**

- Extend `apps/web/tests/e2e/local-preprod-functional.spec.ts`.
- Create `scripts/test-local-preprod-integrations.sh`.
- Modify service-specific tests and runbooks where gaps are found.
- Reuse the cross-repository integration script from the AviaCore plan.

**Work**

- [ ] Upload bounded Inspection Attachments and Evidence through private signed
  paths; prove real ClamAV clean/infected/error/timeout outcomes.
- [ ] Render Preliminary, Final, and Closure PDFs through Gotenberg and verify
  immutable provenance and authorized download.
- [ ] Deliver internal and Auditee-safe email through authenticated SMTP and
  verify retry, deduplication, recipient authority, and message redaction.
- [ ] Exercise offline checkout, IndexedDB/OPFS staging, restart, sync,
  stale-revision conflict, user switch, grant expiry/revoke, and no automatic
  merge.
- [ ] Reconcile each approved domain mutation through the producer outbox,
  mTLS publisher, AviaCore acknowledgement, raw landing, DQ, and governed
  semantic relation.
- [ ] Prove AviaCore loss does not block operational commits, exposes lag, and
  recovers exactly after restart/replay.

**Verification**

Run:

    ./scripts/test-local-preprod-integrations.sh acceptance

The contract-tested aggregate runs the named real-service, OIDC, offline, and
AviaCore local integration projects against one shared accepted profile.
Expected: no mock fallback, no skipped scenario, exact object/document/message/
event identities, exhaustive approved feed coverage, and zero residue after
scoped cleanup.

**Acceptance**

- Every external-service dependency has a truthful success, degraded, retry,
  and recovery path.

### Task 6: Establish Repeatable Load, Spike, And Concurrency Gates

**Files**

- Create `performance/k6/`.
- Create `scripts/test-local-preprod-load.sh`.
- Create `scripts/prepare-local-preprod-profile.sh` to perform the Task 1
  fingerprinted whole-namespace reset/load/reconciliation handoff.
- Create `tests/local-preprod-load-contract.test.mjs`.
- Create workload documentation under `docs/operations/`.

**Work**

- [ ] Pin a proven k6 runtime by immutable digest and keep credentials in
  ephemeral secret mounts.
- [ ] Model weighted read, mutation, login/session refresh, upload metadata,
  sync, reporting, administration, and AviaCore-producing workflows.
- [ ] Bind arrival rate/VUs, concurrency, duration, think time, command mix,
  scenario distribution, account pool, dataset profile, thresholds, and
  generator resource isolation in a versioned workload manifest.
- [ ] Use distinct authorized synthetic users and preserve organization/role
  scope under concurrency.
- [ ] Run the 100-session acceptance profile and 500-session spike against
  `realistic` data.
- [ ] Qualify the `stress` data profile with the same complete workflow
  catalog and a separately frozen query/worker/feed-drain workload. If the
  measured host envelope cannot support it, record a literal blocker; do not
  replace it silently with `realistic`.
- [ ] Dispose the Task 4-5 `acceptance` namespace, load/reconcile a clean
  `realistic` namespace for acceptance/spike, dispose it, load/reconcile clean
  `stress` for `stress-data`, then dispose and reload/reconcile clean
  `realistic` for the soak/DR handoff. Record every before/after fingerprint.
- [ ] Measure endpoint p50/p95/p99, error rate, saturation, DB pool/locks,
  worker lag, feed acknowledgement lag, object/PDF/SMTP latency, and recovery.
- [ ] Add focused races for idempotency, revision conflicts, CAP/Evidence
  versions, report decisions, user lifecycle, sync, outbox claims, and event
  acknowledgement.
- [ ] Fail on threshold breach, silent retries, unexplained 4xx/5xx,
  cross-organization result, duplicate canonical record, or lost
  acknowledgement.

**Verification**

Run:

    node --test tests/local-preprod-load-contract.test.mjs
    ./scripts/prepare-local-preprod-profile.sh realistic
    ./scripts/test-local-preprod-load.sh acceptance
    ./scripts/test-local-preprod-load.sh spike
    ./scripts/prepare-local-preprod-profile.sh stress
    ./scripts/test-local-preprod-load.sh stress-data
    ./scripts/prepare-local-preprod-profile.sh realistic

Expected: all provisional targets pass or the plan records a literal blocker
with endpoint/resource evidence.

**Acceptance**

- Load claims are bound to exact workload and machine profiles and are
  repeatable.

### Task 7: Run The Twelve-Hour Soak And Resource Stability Gate

**Files**

- Create `scripts/test-local-preprod-soak.sh`.
- Add run-scoped metrics queries and evidence schema.
- Modify alerts only through tested rule changes.

**Work**

- [ ] Warm the stack, capture the baseline, then run 12 hours at 100 concurrent
  sessions using `realistic` data.
- [ ] Refuse to start unless the Task 6 handoff fingerprint equals a clean,
  exactly reconciled `realistic` manifest with no `acceptance` or `stress`
  residue.
- [ ] Continuously exercise API, worker, scheduler, identity, object, document,
  SMTP, offline sync, and AviaCore feed paths.
- [ ] Record CPU, RSS, file descriptors, goroutines, DB connections/locks,
  volume growth, queue depth/age, retry counts, cache behavior, latency, and
  error rate.
- [ ] Restart one non-stateful application process during soak and verify
  bounded recovery without data loss.
- [ ] Confirm no monotonic resource leak after warm-up, no stuck lease,
  unbounded retry, dead letter without alert, stale session authority, or
  unreconciled event.
- [ ] Preserve full telemetry but redact user-entered and sensitive values.

**Verification**

Run:

    ./scripts/test-local-preprod-soak.sh 12h

Expected: the complete duration finishes, provisional thresholds pass, queues
return to baseline, and source/AviaCore reconciliation is exact.

**Acceptance**

- Stability is measured over time rather than inferred from short smoke tests.

### Task 8: Exercise Failure, Observability, Backup, And Disaster Recovery

**Files**

- Extend `scripts/test-observability-profile.sh`.
- Extend backup/recovery scripts and operation contracts.
- Create `scripts/test-local-preprod-recovery.sh`.
- Add failure fixtures for data-feed and membership services.
- Update owner/runbook/alert catalogs.

**Work**

- [ ] Exercise loss and recovery of application PostgreSQL, Keycloak
  PostgreSQL, Keycloak, MinIO/S3-compatible storage, ClamAV, Gotenberg, SMTP,
  API, worker, scheduler, data-feed worker, gateway, observability, and
  AviaCore.
- [ ] Prove alert firing, grouping, owner, runbook, notification, silence
  policy, and recovery for every required symptom.
- [ ] Produce a pre-loss recovery package binding both databases, exact object
  versions, identity/MFA state, feed acknowledgement frontier, image/config
  roots, and AviaCore checkpoint references.
- [ ] Freeze RPO/RTO start/end clocks and maximum acknowledged-loss semantics
  separately for application DB, Keycloak DB, object versions, producer
  outbox/acknowledgement frontier, AviaCore candidate state, configuration,
  secrets, and keys. Record source event time, failure time, recovery point,
  service-ready time, and calculated result per store.
- [ ] Restore into an isolated target, verify exact fingerprints and
  permissions, log in with restored MFA, run all roles/routes, deliver worker
  output, reconcile AviaCore, and restart the API.
- [ ] Exercise corrupt-latest fallback and prove local RPO at or below 120
  seconds and RTO at or below 120 seconds.
- [ ] Reconcile the restored `realistic` namespace to its exact manifest,
  assert zero `acceptance`/`stress` residue, then dispose every task-owned
  primary and isolated-restore namespace. Reload and reconcile one clean
  `acceptance` namespace for Task 9; bind before/after fingerprints and do not
  carry restored state into security, accessibility, or UAT evidence.
- [ ] Keep same-host logical isolation explicit; do not claim host-loss or
  regional recovery.

**Verification**

Run:

    ./scripts/test-local-preprod-recovery.sh

The contract-tested aggregate names the observability, backup-catalog, two
coordinated restore-point, corrupt-latest, functional/OIDC, reconciliation,
RPO/RTO, and residue subcommands. Expected: all alerts and both restore points
pass with exact per-store fingerprints/clocks and zero task-owned residue.

**Acceptance**

- Backup is accepted only through a successful isolated functional restore.

### Task 9: Complete Security, Privacy, Accessibility, And UAT Review

**Files**

- Create `scripts/test-local-preprod-security.sh`.
- Create security mutation contracts.
- Add UAT evidence under a create-only stakeholder directory.
- Update threat-model and operations documents where verified gaps exist.

**Work**

- [ ] Refuse to start unless the Task 8 handoff fingerprint equals one clean,
  exactly reconciled `acceptance` namespace and contains no `smoke`,
  `realistic`, `stress`, or isolated-restore residue.
- [ ] Run dependency, Go, container, IaC, secret, TLS, header, cookie, CSRF,
  CORS, rate, upload, authorization, tenant, object-access, and DAST checks.
- [ ] Run OWASP ZAP or an equivalently proven pinned DAST tool against the
  browser-facing gateway using only synthetic data and an approved local
  account set.
- [ ] Exercise IDOR, role/organization drift, stale session, provider outage,
  mTLS spoof, path/body identity drift, mass assignment, oversized input,
  malicious file, log injection, unsafe redirect, and sensitive error cases.
- [ ] Scan logs, traces, metrics, evidence, screenshots, emails, PDFs, AviaCore
  raw/DQ/marts, and loader manifests for forbidden values.
- [ ] Run keyboard, focus, screen-reader semantics, contrast, touch target,
  overflow, zoom, and desktop/tablet/mobile checks on all applicable routes.
- [ ] Run automated accessibility rules on all 86 routes at the three
  viewports against the exact trusted-HTTPS release digest/data; run keyboard,
  focus, dialog/error, visible-control, 200%/400% zoom/reflow, and touch-target
  checks across the action matrix; and record a manual assistive-technology/
  screen-reader ledger for every unique dynamic layout and critical workflow.
- [ ] Run stakeholder UAT for every role using named acceptance scenarios and
  record accept/fix/defer decisions with owners.
- [ ] Require no open Critical/High security finding and no P0/P1 product
  defect; any accepted lower finding needs an owner, expiry, and mitigation.

**Verification**

Run:

    ./scripts/test-local-preprod-security.sh
    npm --prefix apps/web run test:e2e:local-preprod:accessibility
    git diff --check

Expected: zero forbidden-data finding, zero unresolved Critical/High security
finding, 86-route environment-bound accessibility evidence with no skip, and a
complete role/UAT/manual-AT ledger.

**Acceptance**

- Security, privacy, accessibility, and stakeholder evidence cover the same
  exact release artifacts and dataset.

### Task 10: Produce The Local Release-Candidate Decision

**Files**

- Create
  `docs/demo-evidence/LOCAL_PREPROD_RELEASE_CANDIDATE_2026-07-27.md` during
  execution.
- Modify `docs/demo-evidence/BUILD_SUMMARY.md`, this plan, the plan index, and
  the technical-debt tracker.
- Create a machine-readable release manifest under a create-only evidence run.
- Create `scripts/test-local-preprod-release-candidate.sh` and its command/
  evidence contract test.
- Create `scripts/verify-local-preprod-cleanup.sh` to verify task-owned PKI/
  trust, process, network, volume, namespace, and secret cleanup from the exact
  release-run manifest without deleting evidence.

**Work**

- [ ] Start one final release run from a single clean Task 2 build, freeze the
  OCI promotion manifest, and run all Tasks 3-9 without rebuilding or changing
  source, configuration, contracts, workload, trust, or data manifests.
- [ ] Execute the frozen Task 1 four-profile ledger in order. Every subordinate
  gate verifies its required predecessor fingerprint, records its post-state,
  and fails on a reused evidence root, mixed-profile rows/accounts/objects/jobs,
  or residue from an isolated restore. The final release root includes the
  entire transition ledger and scoped cleanup attestation.
- [ ] Bind source state, dirty state, image/SBOM/scan roots, migrations,
  contracts, workload profiles, Keycloak realm, mTLS references, AviaCore
  roots, test outputs, screenshots, performance runs, alerts, restore points,
  findings, UAT decisions, and residue checks.
- [ ] Require 100 percent critical E2E scenarios, zero authority/privacy
  breach, zero acknowledged data loss, exact AviaCore reconciliation, accepted
  local performance/DR targets, and complete rollback instructions.
- [ ] Obtain independent specification, code, security/privacy, operations,
  data-platform, Data/ML, QA, and stakeholder review.
- [ ] Fix every Critical and Important issue and rerun the affected complete
  gate rather than editing evidence into a pass.
- [ ] Treat any source, image, runtime configuration, contract, workload,
  synthetic-data schema/generator, PKI/trust, or migration correction as a new
  release subject: restart at Task 2 and rerun every dependent Task 3-9 gate.
  Only an evidence-verifier-only correction that cannot affect subject,
  runtime, inputs, or measured behavior may use a dependency-graph-bounded
  rerun, and the independent reviewer must approve and record that graph.
- [ ] Publish the literal decision as GO or NO-GO for the local milestone.
- [ ] After every Task 3-9 gate stops, remove the exact release-run PKI and
  isolated browser/CLI trust store through the scoped cleanup command. Prove
  their before/after fingerprints, no host/global trust mutation or residue,
  and no task-owned process/network/volume/profile residue before finalizing
  the decision root.

**Verification**

Run:

    ./scripts/test-local-preprod-release-candidate.sh run <release-run-id>
    ./scripts/cleanup-local-preprod-pki.sh <release-run-id>
    ./scripts/verify-local-preprod-cleanup.sh <release-run-id>
    ./scripts/test-local-preprod-release-candidate.sh finalize <release-run-id>
    node tests/harness-docs-smoke.test.js
    git diff --check

The aggregate command must enumerate every subordinate command and expected
evidence path, profile-transition fingerprint, invalidation edge, and cleanup
attestation. Expected: one self-contained, verifier-derived release root with
no stale, skipped, mock-bound, TLS-bypassed, rebuilt, mixed-profile, restored-
state, or unbound evidence.

**Acceptance**

- GO permits only the label
  `local preprod release candidate; verified locally`.
- NO-GO records exact blockers and recovery actions.
- Neither outcome authorizes AWS, production traffic, or source-bound customer
  data.

## Risks And Mitigations

| Risk | Mitigation |
|---|---|
| Local test profile is mistaken for preprod | Normal OIDC/HTTP only, no test routes, policy mutation tests |
| Rebuild changes the artifact after tests | Build once, digest lock, run all gates against exact images |
| High-volume tests exhaust the host | Declared Docker resource envelope, bounded profiles, monitored fail-closed runs |
| Short tests miss leaks | Mandatory 12-hour soak and post-warm-up growth checks |
| Service outage blocks user work silently | Failure matrix, truthful degraded states, alerts, bounded retries |
| Backup evidence omits identity/feed state | Two databases, object versions, MFA, acknowledgement frontier, functional restore |
| Local success is promoted to production | Fixed local-only label and separate AWS plan |
| DAST or load evidence leaks credentials | Synthetic accounts, ephemeral mounts, value scans, create-only redacted evidence |
| Docker config ID is mistaken for a promotable image digest | OCI config/manifest/archive identities and one shared promotion schema |
| HTTPS passes only because the client ignores errors | Task-owned trusted CA, hostname/rotation negatives, no TLS-ignore, and trust cleanup |
| Mock accessibility evidence is attached to preprod | Dedicated environment-bound project over the exact HTTPS digest/data |
| Dirty local source is mistaken for an AWS commit subject | Explicit source-content provenance and a separate AWS source-boundary decision |

## Idempotence And Recovery

- Clean-start and preserve-data modes are separate commands.
- One release run owns exact project names, ports, networks, volumes,
  certificates, users, loader run, and evidence directory.
- Failure cleanup targets only those exact identities.
- A release run cannot reuse a prior evidence root or image lock.
- Long-running load/soak/DR commands preserve partial evidence on failure and
  report a safe resume or clean-restart procedure.

## Decisions

- Complete all local readiness work before spending on AWS.
- Reuse and extend the existing production-like local services.
- Use proven pinned tooling for load and DAST rather than hand-written engines.
- Require realistic data for performance and acceptance data for deterministic
  functional diagnosis.
- Require all four profiles to retain the same scenario/role/route catalog;
  profile scale cannot remove workflow coverage.
- Treat OCI manifest digest, config digest, and archive digest as distinct
  identities.
- Reserve `production-ready` for the later environment/release gate.

## Discoveries

- Plans 3 and 4 already provide real local Keycloak, MinIO, ClamAV, Mailpit,
  Gotenberg, observability, backup/restore, Terraform contracts, and clean
  86-route/10-scenario evidence.
- Current local DR evidence proves same-host logical isolation, not host-loss.
- Existing full mode deliberately excludes mock/seed inputs and test-profile
  routes; the local preprod profile must preserve that boundary.
- Current local image evidence is based on Docker config IDs, while later ECR
  promotion requires OCI registry manifest identity; Task 2 must repair this
  evidence boundary before local GO.
- The existing accessibility package command targets a mock project and
  Playwright can be configured to ignore HTTPS errors; neither is accepted for
  the preprod gate.

## Outcome Notes

Planning and independent review only. Local profile implementation, image
build, identity/data load, load/soak, DAST, UAT, release evidence, Git
publication, and deployment are `not run`.

## Execution Prompt

```text
Execute docs/exec-plans/active/2026-07-27-local-preprod-release-candidate-plan.md only after both predecessor plans pass and Plan 1 visual decisions are recorded. Read AGENTS.md, docs/PLANS.md, the complete plan, Plans 1-4 evidence, the identity/data evidence, and the AviaCore/ML evidence before editing.

Use the normal HTTP/OIDC application and the exact immutable local OCI subjects. Do not expose test/reset routes, canonical-header authentication, mock/seed inputs, databases, object stores, identity administration, or telemetry ports. Use only synthetic data. Build once and bind every test, workload, finding, screenshot, alert, restore, and UAT decision to the exact OCI manifest/archive, source-content, contract, workload, data, and trusted-TLS roots. Never use TLS-ignore or a mock-bound accessibility project.

Use task-by-task RED -> GREEN and independent review. Execute the complete role-route-action-workflow matrix with zero skip, and qualify all four data profiles at their exact gates. Preserve authority, tenant isolation, append-only versions, Internal CAA Note privacy, CAP-is-not-closure, advisory-only analytics/ML, and unrelated worktree changes. Do not run AWS, use production data, claim host-loss recovery, claim production-ready, commit, or push without separate authorization. Stop on any Critical/Important finding, workload threshold breach, privacy mismatch, unexplained event gap, stale evidence, or cleanup failure.
```
