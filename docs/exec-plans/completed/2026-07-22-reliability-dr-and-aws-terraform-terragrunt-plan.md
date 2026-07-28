# Reliability, DR, And AWS Terraform/Terragrunt Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use
> `superpowers:executing-plans` to execute this plan task by task. Do not
> dispatch subagents unless the user explicitly authorizes subagent work.
> Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add complete local observability, alerting, backup/restore, RPO/RTO,
disaster-recovery evidence and runbooks, then prepare and—only after separate
authorization—exercise a reproducible AWS trial using Terraform composed by
Terragrunt.

**Architecture:** Instrument the existing React/Go/worker platform with one
OpenTelemetry contract and run Prometheus, Grafana, Loki, Tempo, and Alertmanager
as optional local Compose services. Use separate pgBackRest stanzas for the
application and Keycloak PostgreSQL databases and a logically isolated local
backup object store for immutable application-object copies. This local store
is not a separate host failure domain. Build reusable AWS
Terraform modules and a Terragrunt environment graph for VPC, ALB, EC2, RDS,
S3, ECR, KMS, Secrets Manager, telemetry, and backups; keep `apply` gated behind
explicit account, region, domain, budget, data-residency, and user approval.

**Tech Stack:** OpenTelemetry SDK/Collector, Prometheus, Grafana, Loki, Tempo,
Alertmanager, structured JSON logs, pgBackRest, MinIO mirror/versioning/object
lock, Docker Compose, Terraform `>= 1.10, < 2.0`, Terragrunt `>= 1.0, < 2.0`,
AWS provider 6.x-compatible modules, native Terraform tests, TFLint, Trivy
configuration and image scanning, CycloneDX SBOMs, Playwright, Go
race/integration tests, and shell-based recovery drills.

**Status:** `completed for the local milestone` — Tasks 1–9 and Task 11 remain
`verified locally`. On 28 July 2026 the user accepted Plan 4 only as a
completed local `candidate-only` milestone. AWS Task 10 remains optional,
unauthorized, and literally `not run`. Release is `release pending`, and
deployment and production readiness are `not run`.

## Stakeholder Closeout — 28 July 2026

The user explicitly authorized the combined Plans 2–4 stakeholder closure. The
canonical implementation and verification basis remains
[Local Reliability, DR, And Infrastructure Evidence](../../demo-evidence/LOCAL_RELIABILITY_AND_DR_2026-07-22.md);
the combined decision and shared exclusions are recorded in the
[Plans 2–4 Stakeholder Disposition](../../demo-evidence/stakeholder/PLANS2_4_STAKEHOLDER_DISPOSITION_2026-07-28.md).

This acceptance is limited to the same-host local milestone. The logically
isolated backup store does not prove host-loss DR. Local measurements do not
establish production SLO, RPO/RTO, alert recipients, or staffed on-call.
Production retention, legal hold, deletion, encryption/KMS ownership,
restoration authority, provider selection, identity federation, external
email, data residency, release, rollback, and operating decisions remain
deferred to their owners.

Task 10 AWS discovery, planning, apply, artifact publication, smoke, rollback,
retain/destroy, and every other AWS action are `not run` and remain
unauthorized. The artifact remains `candidate-only`; release remains `release
pending`; deployment and production readiness remain `not run`; no
`production-ready` claim is made.

On 2026-07-28 the user explicitly replaced the draft per-task Git sequence
below with one Plan 4-scoped commit. The verified local implementation and
evidence were committed as `f9787b3` (`feat(plan4): complete reliability and
infrastructure readiness`). The unchecked exact-message bullets remain literal
historical records because those draft commits were not created. This Git
reconciliation did not authorize or run AWS Task 10.

## Objective

Make local operations measurable and recoverable before any cloud experiment.
The team must be able to detect failures, trace a user action through API and
workers, validate alert delivery, restore authoritative data into an isolated
stack, measure candidate RPO/RTO, and follow reviewed runbooks. Only then may a
separately authorized AWS trial use the same artifacts and contracts.

## Scope

- Define candidate service objectives, telemetry semantics, alert thresholds,
  dashboard ownership, and PII/secret redaction.
- Instrument browser, gateway, Go API, PostgreSQL calls, outbox/jobs, ClamAV,
  SMTP, Gotenberg, MinIO, Keycloak dependency health, and offline sync.
- Run a local LGTM-style stack: OpenTelemetry Collector, Prometheus, Grafana,
  Loki, Tempo, Alertmanager.
- Deliver local alerts to Mailpit and expose alert history/deduplication.
- Back up both application and Keycloak PostgreSQL with separate pgBackRest
  stanzas to a logically isolated local backup store. Production/host-loss
  recovery still requires a separately approved failure domain.
- Back up exact MinIO object versions and metadata to the distinct backup store;
  versioning alone is not treated as backup.
- Automate isolated restore, corruption/missing-service drills, and measured
  candidate RPO/RTO.
- Write operational, incident, backup, restore, credential rotation, release,
  rollback, and local shutdown/runbook documentation.
- Build reusable Terraform modules and Terragrunt dependency layout for an AWS
  trial without embedding environment secrets or account-specific values.
- Run format, validate, unit, policy, security, and plan gates offline/locally.
- Prepare phase-specific AWS wrappers and reviewed plan artifacts without
  creating resources. Remote-state bootstrap, foundation/ECR, artifact
  publication, and data/runtime are separate plan/apply phases.
- Perform any AWS `apply`, smoke, rollback, retain, or destroy only after a new
  explicit authorization for that exact phase, script hash, and plan hash in
  the execution thread.

## Assumptions

- Plans 1–3 are `ready-for-verification`; their stakeholder review is explicitly
  deferred until the end. The user accepted the resulting rework risk for local
  Tasks 1–9 and Task 11 only. Any AWS trial milestone still requires the plan's
  separate owner decisions and exact phase authorization.
- Local reliability targets are engineering acceptance targets, not contractual
  production SLOs:
  - candidate RPO: no more than 15 minutes for application PostgreSQL,
    Keycloak PostgreSQL, and object metadata;
  - candidate RTO: no more than 60 minutes for an isolated complete restore;
  - API p95: reads <= 500 ms and commands <= 1 s under the defined local load,
    excluding asynchronous scan/email/document completion;
  - outbox oldest-ready age: warning at 2 minutes, critical at 10 minutes;
  - scan/document/email job failure: alert after 3 bounded attempts;
  - backup age: warning at 30 minutes for incremental metadata and critical at
    26 hours for the last successful full/differential recovery point.
- Production RPO/RTO/SLO, retention, data residency, on-call staffing, and alert
  recipients require owner approval and are not inferred from local targets.
- AWS region has no default. A region and data-residency decision is a required
  input before planning or applying an actual environment.
- AWS trial uses managed RDS PostgreSQL and S3 through existing adapters rather
  than running PostgreSQL or MinIO inside the application EC2 instance.
- Keycloak and application containers run on EC2 for the trial; production
  identity federation remains a later decision.
- AWS email smoke uses a private trial-only Mailpit container with no public UI
  or external delivery. Choosing SMTP/SES for production remains a separate
  provider and records decision.

## Global Constraints

- Telemetry must not contain passwords, provider tokens, session cookies,
  Evidence bytes, message bodies, Internal CAA Note text, or unnecessary PII.
- Logs use structured event names and stable correlation IDs; user-visible IDs
  may be recorded only where the audit/security contract requires them.
- Liveness, readiness, SLO, audit log, and business KPI remain distinct.
- Alerts require a symptom, threshold, duration, owner, severity, runbook, and
  verified recovery action. Avoid alerting on every log line.
- Backup jobs are immutable, encrypted, access-separated, and restore-tested.
- Restore never overwrites the active local stack; drills use isolated Compose
  project names, networks, volumes, buckets, and credentials.
- Terraform modules contain no account/region/environment literals. Terraform
  owns every AWS resource, including remote-state bucket/KMS/lock resources.
  Terragrunt supplies reviewed environment inputs, composes dependency outputs,
  generates provider/backend configuration, and orchestrates phase commands; it
  does not own AWS resources.
- Remote Terraform state is encrypted/versioned S3 with native lockfile and
  least-privilege access. State may contain sensitive metadata and is never
  committed.
- No AWS resource is created, changed, or destroyed without explicit current-
  task authorization and a reviewed plan output.
- `destroy` targets only resources tagged and owned by the exact trial stack.
- Do not claim production readiness, production on-call, legal approval, or
  disaster-recovery certification from local/AWS trial evidence.
- Work on the current branch and preserve unrelated untracked paths.

## Ownership Boundaries

| Owner | Responsibility |
|---|---|
| Platform/Operations | Telemetry infrastructure, dashboards, alerts, backups, restores, Terraform resource modules, Terragrunt environment composition |
| Backend | Instrumented spans/metrics/logs, health semantics, job/run state, backup consistency hooks |
| Frontend | Navigation/performance/error telemetry without sensitive data; user-visible degraded state |
| Security | Redaction, secrets/state/KMS/IAM review, audit/alert access, image/IaC scanning |
| Records/Legal | Production retention, legal hold, region, backup retention, and deletion approval |
| Product/CAA Operations | Business severity and owner mapping; no automatic enforcement from alerts/indicators |
| Release authority | Explicit approval for AWS plan/apply/rollback/destroy and any later production action |

## Local Telemetry Topology

| Service | Purpose |
|---|---|
| OpenTelemetry Collector | Receive OTLP, enrich bounded resource attributes, redact/drop forbidden fields, route metrics/logs/traces |
| Prometheus | Scrape/store local technical metrics and rule evaluation |
| Grafana | Provisioned dashboards and read-only review |
| Loki | Bounded structured application/gateway/worker logs |
| Tempo | Distributed traces linking gateway, API, PostgreSQL, outbox, and worker adapters |
| Alertmanager | Deduplicate, group, inhibit, and deliver local alerts to Mailpit |
| pgBackRest | Separate application and Keycloak PostgreSQL full/differential/incremental backup stanzas and isolated restore |
| backup MinIO | Separate credentials/volume for a logically isolated local pgBackRest repository and mirrored application objects; not a separate host failure domain |

## AWS Trial Target

- Two-AZ VPC with public ALB subnets and private application/database subnets.
- ALB with ACM certificate and HTTPS-only listener after an approved domain input.
- EC2 Auto Scaling Group for gateway, React artifact, API, workers, scheduler,
  Keycloak, scanner, document renderer, private trial-only Mailpit, and telemetry
  exporters; minimum/desired counts are environment inputs and trial defaults
  are cost-reviewed before apply. Mailpit has no public route and is not a
  production email provider.
- RDS PostgreSQL with encryption, backups, parameter group, private access, and
  separate application/identity databases or instances according to reviewed
  trial cost/security output.
- S3 application and backup buckets with KMS, public-access block, versioning,
  lifecycle policy, and least-privilege roles.
- ECR repositories with immutable tags and scan-on-push.
- Secrets Manager/SSM Parameter Store and instance roles; no user-data secrets.
- CloudWatch logs/metrics/alarms receive OTel exports for the trial. Local
  Grafana/Loki/Tempo remain the developer/rehearsal stack.
- Terraform creates all resources, including remote-state bootstrap resources.
  Terragrunt owns environment composition, dependencies, generated
  provider/backend configuration, remote-state wiring, and phase-scoped
  `run --all` orchestration.

## Phases

1. **Local observability and recovery — Tasks 1–6:** freeze telemetry/SLO
   contracts, instrument the stack, activate dashboards/alerts, prove backups,
   RPO/RTO/DR, and runbooks.
2. **Infrastructure-as-code readiness — Tasks 7–9:** build tested Terraform
   modules, compose them with Terragrunt, and enforce plan/security/cost gates
   without creating AWS resources.
3. **Local evidence and lifecycle — Task 11:** after Tasks 1–9, reconcile local
   evidence and record AWS literally as `not run`. Task 11 does not wait for or
   require Task 10.
4. **Optional separately authorized AWS branch — Task 10:** only after the local
   milestone and a new exact phase authorization, apply one reviewed phase,
   stop at the next boundary, and repeat approval as needed. Numeric ordering
   does not authorize or make Task 10 a prerequisite for Task 11.

---

### Task 1: Freeze SLO, Telemetry, Redaction, And Alert Contracts

**Files**

- Create `docs/operations/SERVICE_OBJECTIVES.md`
- Create `docs/operations/TELEMETRY_CONTRACT.md`
- Create `docs/operations/ALERT_CATALOG.md`
- Create `docs/operations/OWNERSHIP.md`
- Create `apps/api/internal/platform/telemetry/contract.go`
- Create `apps/api/internal/platform/telemetry/contract_test.go`
- Create `apps/web/src/telemetry/telemetry-contract.ts`
- Create `apps/web/src/telemetry/telemetry-contract.test.ts`
- Create `tests/operations-docs-contract.test.mjs`

**Interfaces**

Every span/log/metric uses a documented name, bounded attributes, correlation
ID, owner, and redaction class. Alert catalog entries require expression,
duration, severity, owner, runbook path, deduplication key, and test fixture.

- [x] Write failing docs/code contract tests for missing objective, owner,
  runbook, unit, histogram boundary, unbounded ID labels, forbidden field names,
  or alert without duration/recovery.
- [x] Run Node, Go, and Vitest contract tests; confirm missing contracts fail.
- [x] Define local targets, resource attributes, trace/log/metric names, redaction
  rules, cardinality limits, alert catalog, ownership, and review cadence.
- [x] Run the contracts plus a repository scan for secrets/PII in telemetry
  declarations; expect zero unowned or forbidden signals.
- [ ] Commit exactly `docs(ops): define reliability contracts` (`not run`;
  Git authorization was not granted).

### Task 2: Instrument Browser, Gateway, API, PostgreSQL, And Workers

**Files**

- Create `apps/api/internal/platform/telemetry/bootstrap.go`
- Create `apps/api/internal/platform/telemetry/http.go`
- Create `apps/api/internal/platform/telemetry/postgres.go`
- Create `apps/api/internal/platform/telemetry/jobs.go`
- Modify `apps/api/internal/httpapi/server.go`
- Modify API/worker/scheduler command entrypoints
- Create `apps/web/src/telemetry/browser-telemetry.ts`
- Modify `apps/web/src/app/bootstrap.tsx`
- Modify `deploy/local/gateway/Caddyfile`
- Create `apps/api/tests/integration/telemetry_pipeline_test.go`

**Interfaces**

W3C trace context propagates gateway → API → PostgreSQL/outbox → worker adapter.
Metrics use bounded status/operation/module labels, never entity IDs. Browser
telemetry records route ID, build profile, Web Vitals, API outcome class, and
handled error boundary without user content.

- [x] Write failing tests for trace propagation, correlation, required spans,
  job links, histogram metrics, sanitized errors, route Web Vitals, shutdown
  flush, unavailable collector, and forbidden high-cardinality/sensitive labels.
- [x] Run focused tests and confirm missing telemetry pipeline.
- [x] Implement OTel bootstrap/middleware/instrumentation, browser exporter,
  structured JSON logs, bounded metrics, job trace links, and graceful flush.
- [x] Run routine/Ad Hoc/Finding/scan/email/PDF scenarios and verify one trace
  graph per action, bounded label sets, no forbidden data, and no request failure
  when the collector is unavailable.
- [ ] Commit exactly `feat(ops): instrument application telemetry` (`not run`;
  Git authorization was not granted).

### Task 3: Add The Local Observability And Alerting Profile

**Files**

- Create `deploy/observability/compose.observability.yaml`
- Create `deploy/observability/otel-collector.yaml`
- Create `deploy/observability/prometheus.yaml`
- Create `deploy/observability/rules/aviasurveil360.yaml`
- Create `deploy/observability/alertmanager.yaml`
- Create `deploy/observability/loki.yaml`
- Create `deploy/observability/tempo.yaml`
- Create `deploy/observability/grafana/provisioning/`
- Create `deploy/observability/grafana/dashboards/`
- Create `scripts/test-observability-profile.sh`
- Create `tests/observability-config-contract.test.mjs`

**Interfaces**

The `observability` profile joins the full local stack without publishing
backend ports beyond loopback tools access. Dashboards are provisioned from Git.
Alertmanager delivers local alert messages to Mailpit with inhibition for a
declared full-stack outage.

- [x] Write failing config tests for unpinned images, missing retention/resource
  limits, open anonymous admin, absent redaction processor, unowned rule,
  missing runbook, alert storm, or internal port exposure.
- [x] Run config contracts; confirm absent profile red.
- [x] Implement immutable-digest Compose services, data volumes, OTel routing,
  scrape/rules, dashboards, datasources, retention, authentication secrets,
  Mailpit receiver, grouping, and inhibition.
- [x] Run the profile, generate every alert fixture, verify dashboard queries,
  trace/log/metric correlation, deduplicated Mailpit receipt, recovery message,
  persistence across restart, and task-owned cleanup.
  - Initial RED: 0/8 config contracts passed because the profile was absent.
  - Current config contract: 9/9 passed, including the OTLP/JSON hex-identifier
    regression contract added after the live harness exposed a base64-encoded
    trace fixture.
  - Pinned Collector `validate`, Prometheus config plus 8/8 alert rules, and
    Alertmanager config checks passed.
  - The 2026-07-26 live profile verified an authenticated provisioned Grafana
    dashboard, trace/log/metric correlation and redaction, one Mailpit delivery
    for a duplicate alert, a resolved recovery delivery, all 8 catalog alert
    fixtures, and Prometheus/Loki/Tempo persistence across restart.
  - OTel and gRPC dependencies were updated to remediated releases; all 8
    digest-bound local images passed the fresh HIGH/CRITICAL vulnerability gate.
  - The accepted amd64-only ClamAV image is explicitly selected on Apple
    Silicon. The focused local policy suite passed 56/56.
  - Every attempt ended with zero task-owned containers, volumes, or networks;
    no unrelated Docker resources were deleted.
- [ ] Commit exactly `feat(ops): add local observability stack`. (`not run`;
  Git authorization was not granted.)

### Task 4: Implement PostgreSQL And Object Backup Pipelines

**Files**

- Create `deploy/recovery/compose.recovery.yaml`
- Create `deploy/recovery/pgbackrest-application.conf`
- Create `deploy/recovery/pgbackrest-identity.conf`
- Create `deploy/recovery/minio-backup-policy.json`
- Create `scripts/backup-postgres.sh`
- Create `scripts/backup-identity-postgres.sh`
- Create `scripts/backup-objects.sh`
- Create `scripts/verify-backup-catalog.sh`
- Create `apps/api/cmd/recovery-fingerprint/`
- Create `scripts/identity-recovery-fingerprint.sh`
- Create `tests/backup-policy-contract.test.mjs`

**Interfaces**

Separate pgBackRest stanzas write encrypted application and Keycloak database
repository data to a logically isolated backup MinIO service with separate
credentials and volume. Every recovery point binds both database backup sets,
the Keycloak realm/user/role/TOTP/provisioning fingerprint, required
configuration/secret references, and the object manifest. Object backup copies
exact key/version, ETag, SHA-256, size, metadata, retention marker, and source
bucket before marking a recovery point complete. Local host/disk loss remains
outside this candidate drill and must not be described as covered.

- [x] Write failing tests for same primary/backup store, plaintext credential,
  mutable backup, missing encryption/version/manifest, partial-success marker,
  absent retention, unverified checksum, missing application or identity stanza,
  missing Keycloak user/TOTP/provisioning fingerprint, or a recovery point
  without both database and application-object fingerprints.
- [x] Run backup policy contracts; confirm current single-store drill is red for
  full policy.
- [x] Implement separate application/identity pgBackRest
  full/differential/incremental schedules, repository stanza/checks, object
  mirror/manifests, retention, application and identity fingerprints, metrics,
  audit records, and lock/idempotency behavior.
- [x] Create two recovery points around controlled database/object changes and
  verify catalog, exact hashes, point selection, age metrics, and alerts.
  - Initial RED: 0/8 backup-policy contracts passed because the isolated store,
    pgBackRest stanzas, immutable manifests, and complete catalog did not exist.
  - Final contract suites passed 14/14 Node and 5/5 focused Go tests.
  - Separate application and Keycloak full backups plus their incremental
    successors completed through encrypted pgBackRest S3 repositories.
  - Two recovery points around controlled database and versioned-object changes
    produced distinct application fingerprints and exact-version manifest
    hashes. Both complete catalogs bind identity/TOTP/provisioning state,
    configuration references, object retention metadata, and checksums.
  - The recovery image joined the digest-bound build/SBOM/scan chain. All 9
    local images passed the fresh HIGH/CRITICAL gate after fixed Alpine package
    versions were pinned and the unused vulnerable `gosu` binary was removed.
  - The final live run consumed the accepted full and recovery image digests,
    performed no ad hoc image rebuild, and ended with zero task-owned
    containers, volumes, or networks.
- [ ] Commit exactly `feat(ops): add verified backup pipelines`. (`not run`;
  Git authorization was not granted.)

### Task 5: Automate Isolated Restore, RPO/RTO, And DR Drills

**Files**

- Create `scripts/restore-isolated-stack.sh`
- Create `scripts/test-rpo-rto-drill.sh`
- Create `deploy/recovery/drill-scenarios.json`
- Create `tests/recovery-drill-contract.test.mjs`
- Create `apps/web/tests/e2e/restored-platform-smoke.spec.ts`
- Create `docs/operations/runbooks/RESTORE.md`
- Create `docs/operations/runbooks/DISASTER_RECOVERY.md`

**Interfaces**

A drill takes a named recovery point and unique isolated prefix. It restores
application PostgreSQL, Keycloak PostgreSQL including provisioned users/roles/
TOTP state, application objects, and required secrets/config references without
reading active volumes. It emits start/end, selected recovery time, calculated
data loss, separate database/object RPO, RTO, fingerprints, scenario results,
and cleanup status. Active sessions may be invalidated, but restored identities
and MFA enrollment must remain verifiable.

- [x] Write failing contracts for active-volume reuse, broad deletion target,
  missing recovery point, checksum mismatch, partial restore marked success,
  absent application/identity backup, identity fingerprint mismatch, missing
  restored TOTP/provisioned-role login, absent RPO/RTO timestamps, skipped
  browser smoke, or missing cleanup.
- [x] Run drill contracts and confirm missing workflow.
- [x] Implement isolated restore orchestration and scenarios for database loss,
  primary object loss, latest-backup corruption fallback, worker backlog, and
  lost application node; never overwrite primary data.
- [x] Run two complete drills and require candidate RPO <= 15 minutes, candidate
  RTO <= 60 minutes, exact application/identity/object fingerprints, normal
  OIDC/TOTP login with restored organization/role scope, 86-route restored
  smoke, and zero isolated residue after evidence capture.
  - Initial RED: all 8 restore/DR contracts failed because the restore workflow,
    scenario catalog, restored browser profile, and runbooks did not exist.
  - Final focused verification passed 25/25 backup/recovery Node contracts,
    focused Go object-backup and recovery-fingerprint tests, TypeScript,
    runbook documentation smoke, shell syntax, and `git diff --check`.
  - Two complete drills restored exact application PostgreSQL, Keycloak
    PostgreSQL, identity/TOTP/provisioned-role state, and retained object
    versions without mounting active application, identity, or object volumes.
  - The final evidence run's latest recovery point measured database/object RPO
    at 3/1 seconds and RTO at 81 seconds. The earlier full recovery point
    measured database/object RPO at 107/102 seconds and RTO at 81 seconds. Both
    are local engineering
    targets, not contractual production commitments.
  - Both drills matched exact application, identity, and object fingerprints;
    delivered one restored `notification.email_requested` backlog item through
    the real worker/Mailpit path; restarted the API node; authenticated with the
    restored TOTP and exact CAA role scope; and direct-loaded all 86 React
    routes without skips.
  - The corrupt-latest-catalog fixture failed closed before target mutation and
    the prior complete point restored successfully. All task-owned containers,
    volumes, networks, state directories, and browser outputs were removed.
  - Full and recovery image digests still match the accepted 9-image SBOM and
    HIGH/CRITICAL vulnerability evidence. The backup store remains
    same-host/logically isolated and does not prove host-loss recovery.
- [ ] Commit exactly `test(ops): prove local recovery objectives`. (`not run`;
  Git authorization was not granted.)

### Task 6: Write Operational, Incident, Release, And Rollback Runbooks

**Files**

- Create `docs/operations/index.md`
- Create `docs/operations/runbooks/START_STOP.md`
- Create `docs/operations/runbooks/INCIDENT_RESPONSE.md`
- Create `docs/operations/runbooks/IDENTITY_MFA.md`
- Create `docs/operations/runbooks/EVIDENCE_SCAN.md`
- Create `docs/operations/runbooks/EMAIL_DOCUMENT_WORKERS.md`
- Create `docs/operations/runbooks/BACKUP.md`
- Create `docs/operations/runbooks/RELEASE_ROLLBACK.md`
- Create `docs/operations/runbooks/SECRET_ROTATION.md`
- Create `tests/runbook-contract.test.mjs`

**Interfaces**

Every alert references a runbook containing symptoms, safety boundary,
diagnosis commands, expected output, reversible mitigation, escalation owner,
recovery verification, evidence capture, and explicit actions that require new
authorization.

- [x] Write failing tests for broken alert/runbook links, destructive broad
  commands, missing owner/precondition/rollback/verification, production claim,
  or unlabeled local-only evidence.
- [x] Run docs contracts and confirm missing runbooks fail.
- [x] Write and dry-run all procedures against the local stack, replacing broad
  commands with scoped identifiers and adding literal evidence labels.
- [x] Exercise one tabletop and one live local incident per severity class;
  record gaps and rerun until all recovery checks pass.
  - Initial RED passed 0/5 because the operations index, eight new runbooks,
    complete shared contract, and warning/critical drill matrix were absent.
  - The final operations collection links all ten runbooks. Every alert and
    Prometheus rule resolves to an existing owner-scoped runbook with symptoms,
    preconditions, safety boundary, diagnosis, expected output, reversible
    mitigation, recovery verification, evidence capture, escalation, and
    explicit authorization boundaries.
  - A fresh exact full-stack dry-run exercised start/status/check, readiness,
    OIDC discovery, read-only Evidence scan/outbox/email/document diagnosis,
    scoped API/worker/Keycloak restart, image evidence, secret-log scanning,
    stop, and zero-residue checks.
  - The dry-run found and corrected three gaps: missing Node.js preconditions,
    an unavailable `rg` command being misreported as zero secret matches, and
    missing HTTPS port/origin values causing `docker compose up` configuration
    drift. The runtime checker now fails closed with exit 69, both regressions
    have contracts, and the corrected procedures passed on rerun.
  - Warning `api-read-latency` and critical `required-dependency-down`
    tabletops are `verified locally`. The live isolated observability rerun
    proved all eight warning/critical fixtures, duplicate grouping to one
    Mailpit delivery, a resolved message, telemetry persistence across restart,
    secret redaction, and zero task-owned residue.
  - Final focused verification passed 25/25 runbook, operations,
    observability, and runtime contracts; docs smoke, relevant shell syntax,
    accepted full-profile image evidence, and `git diff --check` also passed.
    These are local candidate operations only and do not establish production
    readiness or staffed on-call coverage.
- [ ] Commit exactly `docs(ops): add operational runbooks`. (`not run`; Git
  authorization was not granted.)

### Task 7: Build Reusable AWS Terraform Modules

**Files**

- Create `infra/terraform/versions.tf`
- Create `infra/terraform/modules/network/`
- Create `infra/terraform/modules/security/`
- Create `infra/terraform/modules/ecr/`
- Create `infra/terraform/modules/load-balancer/`
- Create `infra/terraform/modules/compute/`
- Create `infra/terraform/modules/database/`
- Create `infra/terraform/modules/object-storage/`
- Create `infra/terraform/modules/identity-secrets/`
- Create `infra/terraform/modules/observability/`
- Create `infra/terraform/modules/backup/`
- Create `infra/terraform/modules/service-endpoints/`
- Create `infra/terraform/bootstrap/remote-state/`
- Create `infra/terraform/tests/`
- Create `infra/policies/aws-trial.rego`

**Interfaces**

Modules accept explicit environment, region, VPC/AZ, domain/certificate,
capacity, image digest, KMS, retention, backup, logging, and tags. Modules expose
only dependency outputs. No module reads Terragrunt paths or remote state
directly. IAM policies use resource-scoped actions and instance roles. The
Terraform bootstrap root owns the remote-state S3/KMS/native-lock resources and
may temporarily use local state only for its separately reviewed bootstrap
phase; Terragrunt never creates those resources itself.

- [x] Write native Terraform and policy tests for two-AZ network, private
  compute/database, HTTPS-only ALB, encrypted RDS/S3/EBS/state, public-access
  block, ECR immutability, IMDSv2, no SSH ingress, no wildcard IAM, Secrets
  Manager refs, backup retention, mandatory ownership/cost tags, and deletion
  protection policy.
- [x] Run `terraform fmt -check`, `terraform validate`, native tests, TFLint, and
  `trivy config`; confirm expected missing-module failures.
- [x] Implement focused modules, examples, outputs, provider/version constraints,
  lifecycle safeguards, user-data that fetches only secret references and ECR
  digests, and CloudWatch/OTel integration.
- [x] Run all module tests and two fixture plans; expect no public database/
  object access, no plaintext secret, no high/critical IaC finding, and stable
  plan JSON policy results.
- Initial native tests failed because the required module directories were
  absent. The final native Terraform suite passed 12/12, including the
  remote-state bootstrap and private AWS service endpoint egress.
- OPA 1.17.0 policy mutation tests passed 6/6. Two bootstrap and two
  secure-trial fixture plans each produced the same empty policy result hash
  `37517e5f3dc66819f61f5a7bb8ace1921282415f10551d2defa5c3eb0985b570`.
  The plans contained 8 and 81 creates respectively, with zero changes or
  destroys, zero public databases, zero incomplete object public-access
  blocks, and zero plaintext secret-version resources.
- Terraform format/validate, TFLint 0.64.0, and Trivy 0.70.0 passed. Trivy
  reported zero HIGH/CRITICAL findings. The intentionally internet-facing,
  deletion-protected HTTPS ALB is a line-scoped documented exception; runtime
  AWS API and S3 egress is restricted to private VPC endpoints.
- Terraform 1.15.8 and AWS provider 6.56.0 are checksum/provider-lock bound.
  The three Terraform dependency locks have the identical SHA-256
  `c6b44bc69255646a525a3fe7f7f177e32910ced268127d4201ec664519c0b3aa`.
- [ ] Commit exactly `feat(infra): add aws terraform modules`. (`not run`; Git
  authorization was not granted.)

### Task 8: Compose AWS Environments With Terragrunt

**Files**

- Create `infra/terragrunt/terragrunt.hcl`
- Create `infra/terragrunt/catalog/`
- Create `infra/terragrunt/environments/aws-trial/account.hcl`
- Create `infra/terragrunt/environments/aws-trial/environment.hcl`
- Create `infra/terragrunt/environments/aws-trial/region.hcl.example`
- Create component `terragrunt.hcl` files under
  `infra/terragrunt/environments/aws-trial/components/`
- Create `infra/terragrunt/bootstrap/remote-state/terragrunt.hcl` that composes
  `infra/terraform/bootstrap/remote-state/`
- Create `scripts/check-terragrunt.sh`
- Create `tests/terragrunt-contract.test.mjs`

**Interfaces**

Terragrunt generates provider/backend files, composes module dependencies, and
wires the Terraform-created versioned/encrypted/native-lockfile S3 backend into
later phases. The committed trial environment has no account ID, region, domain,
secret, or state bucket default; execution fails with an exact
missing-owner-input message until an approved untracked environment overlay is
supplied. Phase order is bootstrap → foundation/ECR → artifact publication →
data/runtime. A fixture may mock dependencies only for validate/plan and must
fail if the same mock is present during apply.

- [x] Write failing tests for duplicated Terraform code, local state outside
  bootstrap, missing encryption/version/lock, account/region literal, dependency
  mock during apply, broad `run --all destroy`, unpinned module source, or absent
  before/after policy hooks.
- [x] Run Terragrunt HCL formatting/validation and contract tests; confirm absent
  layout failures.
- [x] Implement root includes, catalog components, dependency graph, generated
  provider/backend, remote-state bootstrap/migration procedure, mandatory input
  validation, policy hooks, and plan artifact naming.
- [x] Run `terragrunt hcl fmt --check`, DAG graph, `run --all validate`, and
  fixture `run --all plan` using a non-deployable test overlay; expect stable
  dependency order and zero apply/destroy.
- [ ] Commit exactly `feat(infra): compose aws with terragrunt`. (`not run`; Git
  authorization was not provided.)

Task 8 is `verified locally`: the source contracts pass 7/7, Terragrunt HCL
formatting and validation pass, the deterministic 12-unit DAG validates and
plans sequentially, and all 12 protected binary plan artifacts decode to JSON
with empty OPA denial sets. The checker exits 64 with the exact
`missing-owner-input` reason when the untracked owner overlay is absent. The
fixture uses only local state under `/private/tmp`, permits dependency mocks
only for validate/plan, and performs zero apply/destroy operations.

### Task 9: Enforce AWS Plan, Security, Cost, And Ownership Gates

**Files**

- Create `scripts/aws-trial-plan.sh`
- Create `scripts/check-aws-plan.sh`
- Create `scripts/aws-trial-apply.sh`
- Create `scripts/aws-trial-publish-artifacts.sh`
- Create `scripts/aws-trial-smoke.sh`
- Create `scripts/aws-trial-rollback.sh`
- Create `scripts/aws-trial-destroy.sh`
- Create `infra/policies/aws-plan.rego`
- Create `docs/operations/AWS_TRIAL_DECISIONS.md`
- Create `docs/operations/AWS_TRIAL_RUNBOOK.md`
- Create `tests/aws-trial-plan-contract.test.mjs`
- Create `tests/aws-trial-command-contract.test.mjs`
- Create `apps/web/tests/e2e/aws-trial-smoke.spec.ts`
- Modify `.gitignore`

**Interfaces**

Each phase plan requires explicit account, region/data-residency approval,
domain/certificate where applicable, budget ceiling, capacity, backup retention,
owner contacts, change window, and destroy/retention decision. Bootstrap,
foundation/ECR, artifact publication, and data/runtime produce separate reviewed
plan hashes and require separate apply authorizations. Plan binaries/JSON,
caller identity, and command manifests live only under gitignored
`.local/aviasurveil360/aws-plans/`, use restrictive permissions, are redacted,
age-encrypted when retained, and carry an expiry/cleanup record. The manifest
binds Terraform/Terragrunt versions, lock hashes, wrapper-script hashes,
immutable image digests, CycloneDX SBOM hashes, Trivy image/config results, cost
estimate when credentials permit, policy result, caller identity, and expiry.
An approved plan or wrapper cannot be replaced silently before apply.

- [x] Write failing tests for missing decision, stale plan, wrong caller/account,
  region mismatch, unbounded cost/capacity, public exposure, destroyable data,
  wildcard IAM, unencrypted resource, mutable image tag, unscanned image digest,
  missing SBOM, unprotected/unredacted plan artifact, changed wrapper hash,
  missing phase boundary, broad destroy, or absent rollback.
- [x] Run offline fixture plan contracts and confirm every mutation fails.
- [x] Implement decision schema, phase plan wrapper, JSON policy/security/cost
  gates, protected plan-artifact lifecycle, evidence manifest, script/hash
  checks, image/SBOM gates, and scoped no-op command previews. Apply, smoke,
  rollback, and destroy wrappers must be fully implemented and contract-tested
  here without contacting a mutable AWS environment.
- [x] Run offline fixtures and—only with read-only AWS credentials—caller/quota/
  availability discovery; do not create resources. A real phase plan also
  requires separate current authorization. After producing its protected hash,
  stop for review; do not apply it in Task 9. (Offline fixtures and previews
  passed. Read-only AWS discovery and a real phase plan are `not run`: no
  credentials or separate phase authorization were supplied.)
- [ ] Commit exactly `test(infra): enforce aws trial plan gates`. (`not run`;
  Git authorization was not provided.)

Task 9 is `verified locally` without AWS access: 46/46 Terragrunt, plan, and command
contracts pass, including 10/10 unsafe Terraform plan mutations and fail-closed
decision, caller, account/region, certificate, data-residency, cost/capacity,
change-window, artifact-permission, stale/hash, redaction, phase, image
scope/digest/SBOM/scan, and wrapper-hash mutations. All 12 Task 8 fixture plan
JSON artifacts have empty AWS plan OPA denial sets. Seven scoped previews
contacted no AWS service, the AWS Playwright project lists exactly two frozen
smoke tests, React typecheck passes, and Trivy reports zero HIGH/CRITICAL IaC
findings. ShellCheck is unavailable locally; all shell sources pass `bash -n`.
AWS planning, apply, publication, smoke, rollback, destroy, and read-only
discovery remain literally `not run`.

### Task 10: Execute The Gated AWS Trial, Smoke, Rollback, And Destroy

**Authorization gate:** This task is `not authorized` by plan creation,
completion of Tasks 1–9, Task 11, or this execution prompt. Stop and request
explicit approval of the exact phase, account, region, reviewed plan hash,
wrapper-script hashes, image/SBOM hashes, budget, change window, and
cleanup/retention action. One approval authorizes one phase only. After each
phase, stop, produce/review the next plan, and request a new authorization.

**Files**

- Use the frozen, contract-tested `scripts/aws-trial-*.sh` wrappers from Task 9
- Use the frozen `apps/web/tests/e2e/aws-trial-smoke.spec.ts` from Task 9
- Create `docs/demo-evidence/AWS_TRIAL_EVIDENCE.md` only after execution; its
  header and evidence manifest record the actual execution date/time

- [ ] Verify the approved phase/plan and wrapper hashes, current caller/account,
  region, budget, artifact image/SBOM/vulnerability evidence, database migration
  backup, domain/ACM where applicable, and trial tags.
- [ ] Apply only the approved phase graph. Bootstrap creates only remote-state
  resources; foundation creates network/ECR; artifact publication pushes and
  verifies immutable digests through the frozen publication wrapper;
  data/runtime follows only after a newly reviewed plan. Stream sanitized
  outputs and stop on any policy, health, budget, ownership, or phase mismatch.
- [ ] Run HTTPS/OIDC/MFA, 86-route read smoke, canonical mutation scenario,
  scan/private-trial-Mailpit/PDF, telemetry/alerts, backup/restore sample,
  security headers, public exposure, image digest/SBOM/vulnerability checks.
- [ ] Exercise reviewed application rollback without data loss and rerun smoke.
- [ ] Perform the separately approved retain-or-destroy action. Before destroy,
  resolve exact tagged resources and preserve required evidence/backups; after
  destroy, prove no trial-owned billable residue. Never use broad account-wide
  deletion commands.

### Task 11: Record Reliability, DR, And Infrastructure Evidence

**Files**

- Create `docs/demo-evidence/LOCAL_RELIABILITY_AND_DR_2026-07-22.md`
- Modify `docs/demo-evidence/BUILD_SUMMARY.md`
- Modify `docs/index.md`
- Modify `MANIFEST.md`
- Modify `docs/exec-plans/index.md`
- Modify `docs/exec-plans/tech-debt-tracker.md`
- Modify this plan

- [x] After Tasks 1–9, run the clean full local 86-route/10-scenario profile
  together with fresh telemetry, alert, dual-database backup, restore, RPO/RTO,
  runbook, Terraform, Terragrunt, image/IaC policy, and cleanup gates. Record AWS
  Task 10 literally as `not run` unless separately authorized and completed.
- [x] Record exact objective results, alert receipts/recovery, trace examples,
  dashboard/rule hashes, backup catalog/recovery-point hashes, RPO/RTO, restore
  fingerprints, tool/module locks, Terraform/Terragrunt fixture plan results,
  security findings, and residue checks.
- [x] Reconcile owners, open production decisions, active index, and tracker.
- [x] Set this plan and the local reliability/IaC milestone to
  `ready-for-verification` when Tasks 1–9, Task 11, and all local gates pass.
  Task 10 may remain intentionally gated/`not run`; it is not a completion
  prerequisite and requires a separate current authorization if later pursued.
- [ ] Commit exactly `docs(evidence): record reliability and infrastructure` and
  push only when explicitly authorized. (`not run` in Task 11.)

Task 11 is `verified locally` on 2026-07-27. The clean full profile passed
86/86 HTTP direct loads and all 10 real-service scenarios; React passed
644/644, operations/runbook contracts passed 43/43, and the canonical Go race
suite passed in its service harness. All eight alert fixtures, grouped and
resolved Mailpit delivery, cross-signal correlation/redaction, dual-database
catalogs, two complete isolated restores, exact fingerprints, corrupt-latest
fallback, 9/9 image/SBOM/scan bindings, 12/12 Terraform tests, 12/12
Terragrunt fixture plans, 46/46 infrastructure contracts, and zero task-owned
residue passed. The final drill measured 3/1-second database/object RPO and
81-second RTO for the latest point, and 107/102-second RPO and 81-second RTO
for the fallback point. Trivy reported zero HIGH/CRITICAL image or IaC
findings. AWS discovery and every Task 10 action are literally `not run`.

## Required Local Verification Matrix

```bash
node --test tests/operations-docs-contract.test.mjs tests/observability-config-contract.test.mjs tests/backup-policy-contract.test.mjs tests/recovery-drill-contract.test.mjs tests/runbook-contract.test.mjs
GOCACHE=/private/tmp/aviasurveil360-ops-go-cache go -C apps/api test -race -p 1 -count=1 ./...
npm --prefix apps/web run typecheck
npm --prefix apps/web test
./scripts/test-local-full-profile.sh
./scripts/generate-image-sboms.sh
./scripts/scan-local-images.sh
./scripts/test-observability-profile.sh
./scripts/verify-backup-catalog.sh
./scripts/test-rpo-rto-drill.sh
terraform fmt -check -recursive infra/terraform
terraform -chdir=infra/terraform test
tflint --recursive --chdir infra/terraform
trivy config --severity HIGH,CRITICAL --exit-code 1 infra
terragrunt hcl fmt --check
AVIA_TG_PLAN_DIR="$(mktemp -d /private/tmp/aviasurveil360-tg-plans.XXXXXX)"
AVIA_TG_INPUTS_FILE="$PWD/infra/terragrunt/fixtures/non-deployable.hcl" \
  AVIA_TG_PLAN_DIR="$AVIA_TG_PLAN_DIR" \
  ./scripts/check-terragrunt.sh
rm -rf -- "$AVIA_TG_PLAN_DIR"
node --test tests/terragrunt-contract.test.mjs tests/aws-trial-plan-contract.test.mjs tests/aws-trial-command-contract.test.mjs
```

AWS `plan` with real read-only credentials and every AWS apply/change/destroy
command remains separately authorized. Fixture plan success is not cloud
deployment evidence.

## Risks And Controls

| Risk | Control |
|---|---|
| Telemetry leaks sensitive aviation/identity data | Allowlisted attributes, collector redaction/drop, forbidden-field tests |
| Metrics create unbounded cardinality/cost | No entity IDs in metric labels, contract tests, retention/resource limits |
| Alerts are noisy or ownerless | Symptom/duration/owner/runbook catalog, grouping/inhibition, fixture exercises |
| Backup exists but cannot restore | Isolated exact-fingerprint restore is the acceptance gate |
| Local logical isolation is mistaken for host-loss recovery | Label the backup store as same-host/logically isolated; production requires a separate failure domain |
| Identity data is omitted from recovery | Separate Keycloak pgBackRest stanza, identity fingerprint, restored user/role/TOTP login gate |
| Versioning is mistaken for backup | Logically isolated backup store and exact manifest/recovery-point policy |
| DR drill damages active data | Unique isolated prefixes and destructive-target contract tests |
| Terraform and Terragrunt duplicate ownership | Terraform owns all resources including state bootstrap; Terragrunt composes environments and generates/wires backend configuration |
| One AWS plan hides bootstrap/artifact dependency changes | Separate reviewed hashes and explicit apply authorization for bootstrap, foundation/ECR, artifact publication, and data/runtime phases |
| Approved plan is executed by changed destructive wrappers | Task 9 contract-tests and hashes every wrapper; Task 10 rejects a hash mismatch |
| Plan artifacts leak sensitive metadata | Gitignored restrictive storage, redaction, age encryption when retained, expiry and cleanup contract |
| AWS trial creates uncontrolled cost/residue | Budget/capacity decision, plan policy, immutable tags, scoped retain/destroy evidence |
| Local targets are presented as production commitments | Literal candidate labels and owner decisions before production SLO/RPO/RTO |

## Dependencies

- Plan 3 complete local production-like Compose profile; stakeholder review is
  deferred with user-accepted rework risk. Task 11 reruns its clean
  86-route/10-scenario full-profile gate after observability/recovery changes.
- Plans 1–2 remain green during restore and any later AWS smoke tests.
- Explicit owner decisions for AWS account, region/data residency, domain,
  budget, capacity, backup retention, and cleanup before real planning/apply.
- Tasks 1–9 plus Task 11 may complete without AWS. Each Task 10 phase requires a
  new current-task approval for exact plan and wrapper hashes.

## Out Of Scope

- Production traffic, production release approval, staffed on-call rotation,
  contractual SLO, production data residency/legal hold, external penetration
  test, compliance certification, or irreversible migration.
- Kubernetes/EKS, microservices, Kafka/RabbitMQ, multi-region active-active,
  automatic enforcement, or external AI.
- Any AWS resource action during plan writing or Tasks 1–9.

## Execution Prompt

```text
Execute docs/exec-plans/completed/2026-07-22-reliability-dr-and-aws-terraform-terragrunt-plan.md with superpowers:executing-plans only after the full React, backend parity, and local production-like service plans are complete and accepted. Execute Tasks 1-9, then Task 11; Task 10 is an optional separately authorized future branch and is not required for the local milestone. Do not dispatch subagents unless explicitly authorized. Work on the current branch and preserve unrelated .superpowers/, docs/demo-evidence/stakeholder/, and outputs/ content.

Complete local observability, alerting, separate application/Keycloak PostgreSQL backups, isolated identity/application/object restore, candidate RPO/RTO, DR drills, and runbooks before cloud work. Use OpenTelemetry Collector, Prometheus, Grafana, Loki, Tempo, Alertmanager, pgBackRest, and a logically isolated same-host backup object store; do not claim host-loss recovery. Never expose secrets, Evidence bytes, message bodies, Internal CAA Note text, provider tokens, or high-cardinality entity IDs in telemetry. Rerun the clean 86-route/10-scenario full profile before Task 11 evidence.

Build reusable AWS Terraform modules and compose them with Terragrunt. Terraform owns all resources including remote-state bootstrap; Terragrunt owns environment inputs, dependencies, generated provider/backend wiring, and phase orchestration. Prepare and contract-test the apply/smoke/rollback/destroy wrappers in Task 9. Require separate reviewed plan hashes for bootstrap, foundation/ECR, artifact publication, and data/runtime, plus explicit region/data-residency, account, domain, budget, capacity, retention, owner, and rollback decisions. Protect plan artifacts, bind image SBOM/vulnerability evidence, and run format/validate/test/TFLint/Trivy and fixture plans without creating resources.

Task 10 AWS apply is not authorized by this prompt, by Tasks 1-9, or by Task 11. Finish Task 11 with AWS recorded as not run. If Task 10 is later requested, one approval covers only one exact phase and must bind account, region, reviewed plan hash, wrapper hashes, image/SBOM hashes, budget, window, and retain/destroy action; stop again before every next phase. Never run broad destroy commands. Use exact commit messages only when Git actions are separately authorized, and inspect upstream/allowlist/cached diff before each commit. Do not claim production readiness.
```
