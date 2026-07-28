# AWS Preprod Validation Plan

**Status:** `paused`

**Reviewed task count:** 8 work packages. Every remote discovery, lock write,
plan, publish, apply, secret population, migration, identity/data operation,
test run, recovery, rollback, retain, destroy, and residue query is a separate
authorization slice; a work-package heading never authorizes the next slice.

**Objective:** Deploy the exact locally accepted release-candidate artifacts to
a separately approved AWS preprod environment, verify environment-specific
security, capacity, identity, data, AviaCore, recovery, and rollback behavior,
and produce a truthful preprod decision plus a bounded production-deployment-
planning recommendation.

**User-visible outcome:** Stakeholders can use a controlled HTTPS preprod
environment with real cloud identity/network/storage/database boundaries and
the same synthetic acceptance scenarios proven locally.

**Intentional deferral:** The user decided that AWS cost is unnecessary until
all application, membership, synthetic-data, AviaCore/ML, and local preprod
work is complete. Document-level planning only is authorized; AWS discovery,
Terraform/Terragrunt plan or lock, apply, publication, deployment, smoke,
rollback, and destroy remain `not run`.

## Entry Gate

This plan cannot start until:

- Local Preprod Release Candidate records GO with the exact label
  `production-ready candidate, verified locally`.
- The exact source commit, image digests, SBOMs, vulnerability results,
  migrations, contracts, workload profiles, and rollback procedure are frozen.
  The source identity also includes the exact source-tree/content-bundle and
  builder-provenance digests; a dirty local candidate cannot be represented by
  a commit alone.
- Plan 1-4 stakeholder dispositions and all Critical/Important successor
  reviews are closed.
- AviaCore provides an approved reachable sandbox/preprod endpoint and exact
  mTLS/source/tenant contract for the intended integration gate.
- Product, Platform, Security, Operations, Data, Legal/Records, QA, and release
  owners provide the inputs required by Task 1.

## Scope

- A separate AWS preprod account or an equally isolated owner-approved account
  boundary.
- Existing Terraform/Terragrunt modules and phase/hash/policy wrappers from
  the Reliability/DR/AWS plan.
- Remote state, networking, private service/database subnets, endpoints, KMS,
  Secrets Manager, ECR, load balancer, TLS, compute, application and identity
  databases, S3 application/backup storage, CloudWatch, AWS Backup, and
  owner-approved edge controls.
- Exact image publication and build-once promotion.
- Forward migration, Keycloak bootstrap/provisioning, synthetic acceptance
  data, all critical E2E scenarios, capacity, failure, backup/restore, rollback,
  and retain/destroy decisions.

## Explicit Exclusions

- Production customer data, production DNS/traffic, customer identity
  federation, production email, or production release.
- Rebuilding artifacts after the local release root.
- Broad AWS permissions, wildcard IAM, public databases/storage, mutable image
  tags, plaintext secrets, direct SSH, manual resource drift, or unreviewed
  Terraform apply.
- Deploying or modifying AviaCore infrastructure from AviaSurveil. AviaCore
  owns its target runtime.
- Multi-region active-active, EKS/Kubernetes, Kafka/RabbitMQ, or a new
  infrastructure architecture without a separate approved ADR and trigger.
- Any `production-ready` claim from this preprod-only plan.

## Authorization Model

One approval covers only one exact action in one topological wave. Every bundle
binds AWS account/region/partition, caller role and session constraints,
action ID, wrapper/policy digests, budget/cost ceiling, expiry, change window,
owner, exact target, expected evidence, next stop, and the applicable
action-specific subject:

| Action | Required subject identity |
|---|---|
| Read-only discovery/residue query | credential/role scope, exact API/query allowlist, discovery or residue script digest, account/region, output schema, and no-write policy |
| Remote plan/lock | Terraform/Terragrunt configuration root, real upstream-output/state serials, backend/lock strategy, variable and wrapper/policy digests, plan action scope, and cost-input digest |
| Apply | the independently reviewed exact binary and JSON plan bytes/hash, prior/current state serial, planned resource/change manifest, wrapper/policy hashes, cost, rollback stop, and source/artifact roots used by runtime changes |
| OCI publish/sign | local OCI archive hash, canonical image-manifest bytes/media type/digest, platform/config digest, destination repository, registry copy tool digest, SBOM/scan/provenance roots, and signing trust policy |
| Secret/migration/identity/loader/mTLS one-shot | exact executable or OCI subject, command/config/schema/data-manifest digest, secret-version references without values, target resource fingerprint, and rollback/forward-fix rule |
| Functional/accessibility/load/spike/stress/soak/DAST/failure/alert test | exact test/tool subject, environment/release root, route-role-action matrix, workload/profile/data manifest, account pool, thresholds, duration, cost, evidence root, and cleanup |
| Restore/failover | exact pre-loss package, backup/object/key/frontier manifests, isolated target, failure action, RPO/RTO clocks, compatibility matrix, and cleanup |
| Rollback/retain/scoped destroy | exact executed rollback/destroy binary+JSON plan or retain manifest, prior image/config, state serial, resource/tag allowlist, cost/expiry, residue query, and recovery/forward-fix boundary |

The agent stops before every next action. A plan document never grants AWS
authority.

The action ledger includes separate stops for discovery credentials, any
remote-state or S3 lock write, each plan wave, each apply wave, artifact
publication/signing, secret population/rotation, migrations, identity
bootstrap, each loader profile, acceptance load, spike, 12-hour soak, DAST,
failure drills, restore/failover, rollback, retain/destroy, and residue query.
Any cost-bearing operation requires its own explicit current authorization and
cost ceiling even when it shares a work-package heading.

## Progress

- [x] (2026-07-27) AWS deferred until local readiness is complete.
- [x] (2026-07-27) Existing Terraform/Terragrunt, security/cost policy, and
  gated Task 10 wrappers identified.
- [ ] Local release-candidate GO.
- [ ] Task 1 owner inputs and separate read-only discovery authorization.
- [x] (2026-07-27) Independent architecture/security/preprod review completed.
  It found a non-deployable single-container runtime topology, dependency-
  invalid plan phasing, unbound rollback/destroy inputs, OCI/ECR identity
  mismatch, incomplete paid-action authorization, and premature production
  claims. All AWS actions remain `not run`.

## Tasks

### Task 1: Freeze Account, Region, Domain, Cost, Capacity, And Ownership

**Files**

- Create an untracked owner-input overlay following
  `docs/operations/AWS_TRIAL_DECISIONS.md`.
- Create `docs/operations/AWS_PREPROD_DECISIONS.md` without secrets.
- Create `tests/aws-preprod-decision-contract.test.mjs`.
- Create an action × wave authorization-ledger schema.
- Create `scripts/check-aws-preprod-decisions.sh`.
- Modify the technical-debt tracker only with verified decisions.

**Work**

- [ ] Record exact account, region, environment name, data residency, domain,
  certificate, DNS owner, account isolation, and permitted principals.
- [ ] Select one TLS ownership branch: reuse a pre-existing eligible ACM
  certificate, or separately authorize ACM request and DNS validation. Record
  expected hostname/SANs, issuer/trust, key/signature policy, TLS policy,
  HSTS, secure/HttpOnly/SameSite cookies, expiry/renewal/rotation overlap,
  CAA/DNS owner, validation method, and named rotation/incident owner. No later
  task may infer certificate or Route 53 mutation authority.
- [ ] Record monthly and one-run budget ceilings, alert thresholds, expiry,
  capacity profile, service quotas, and automatic stop conditions.
- [ ] Record application/identity database topology, availability, encryption,
  maintenance, backup, PITR, retention, deletion, and legal-hold decisions.
- [ ] Record separate application, identity, object, feed-frontier, secrets/
  keys/configuration RPO/RTO targets, start/end clocks, recovery mechanisms,
  failure-domain requirements, and maximum acknowledged-loss definitions.
- [ ] Record application/backup bucket residency, KMS ownership, versioning,
  object lock, lifecycle, access-log, and restore decisions.
- [ ] Record compute topology, desired/min/max capacity, scaling signals,
  deployment strategy, WAF/rate policy, egress policy, private endpoints,
  observability retention, and on-call owner.
- [ ] Record every required runtime and one-shot role: gateway/web, API,
  worker, scheduler, migration, data-feed worker, Keycloak, scanner, renderer,
  mail integration, loader, and recovery. For each bind image/command, secret
  injection, IAM, network, health/readiness, logging, restart, scaling,
  dependency, and ownership.
- [ ] Record AviaCore endpoint/trust references, producer source/tenant mapping,
  CA/SAN fingerprints, credential rotation, rate limit, SLO, and outage policy.
- [ ] Record exact change window, approvers, rollback owner, and whether the
  final environment is retained or destroyed.
- [ ] Freeze cloud acceptance, spike, and minimum 12-hour soak workload/data
  profiles, arrival/VU/concurrency/duration/thresholds, generator isolation,
  scaling targets, cost ceilings, and cleanup. Acceptance-only data or a
  shorter soak cannot support a capacity/stability recommendation.
- [ ] Fail closed when any required value or named owner is absent.

**Verification**

Run:

    node --test tests/aws-preprod-decision-contract.test.mjs
    ./scripts/check-aws-preprod-decisions.sh

Expected: an incomplete, secret-bearing, contradictory, expired, or
over-budget decision package is rejected without contacting AWS.

**Acceptance**

- Every externally owned decision is explicit before discovery or planning.

### Task 2: Run Read-Only Discovery, Quota, And Cost Preflight

**Files**

- Extend the read-only discovery artifact schema and preflight script only if
  existing Plan 4 wrappers lack an accepted input.
- Create `scripts/aws-preprod-discover.sh` and a no-write command-contract test.
- Create a protected run-scoped discovery evidence root.

**Work**

- [ ] Assume the exact read-only discovery role and verify caller account,
  organization/account status, region, partition, and permission boundary.
- [ ] Inspect service availability and quotas required for VPC, EIP/NAT, ALB,
  EC2/ASG, RDS, S3, KMS, Secrets Manager, ECR, CloudWatch, AWS Backup, ACM,
  Route 53, WAF, and approved endpoints.
- [ ] Verify domain and certificate ownership without changing records or
  requesting a certificate unless separately authorized.
- [ ] For a proposed pre-existing ACM certificate, read and bind its ARN,
  domain/SAN set, issuer, status, key/signature type, expiry, in-use state,
  managed-renewal eligibility, validation records, and account/region. If no
  eligible certificate exists, return a literal `CERTIFICATE_ACTION_REQUIRED`;
  discovery never requests a certificate or changes Route 53.
- [ ] Estimate steady-state and bounded test-window cost from owner-approved
  capacity and retention.
- [ ] Detect conflicting CIDRs, names, buckets, state resources, certificates,
  quotas, policies, or pre-existing mutable infrastructure.
- [ ] Write no resource, state, secret, DNS, certificate, role, or budget.
- [ ] Require a separate current authorization for the discovery credentials
  and calls. Do not run Terraform/Terragrunt plan here; native remote-state
  locking is a write and is not "read-only discovery."

**Verification**

Run, only after the exact discovery authorization:

    ./scripts/aws-preprod-discover.sh <discovery-authorization-bundle>

Expected: a protected discovery report with zero AWS mutations and a GO/NO-GO
preflight result.

**Acceptance**

- Plan generation begins only after a clean, current, account/region-bound
  preflight.

### Task 3: Close Runtime/IaC Architecture And Freeze Topological Waves

**Files**

- Modify Terraform modules only for review-proven gaps.
- Modify Terragrunt environment composition and the protected preprod overlay.
- Modify OPA/security/cost tests with every new resource or input.
- Create `docs/operations/AWS_PREPROD_RUNTIME_TOPOLOGY.md`, a machine-readable
  runtime/topology manifest, `scripts/test-aws-preprod-architecture.sh`, and
  wave/authorization contract tests.

**Work**

- [ ] Treat the current AWS candidate as non-deployable: the compute module
  launches one image and does not mount fetched secrets into that container,
  while the current composition models only one database. Close those gaps
  before any real plan.
- [ ] Obtain Platform, Security, Identity, Operations, and Data approval for an
  ADR/topology that models every runtime/one-shot role from Task 1, exact
  secret mounts, per-service IAM/network/health/log/restart/scaling, migration/
  loader isolation, and separate application/identity database and recovery
  scopes.
- [ ] Replace broad phase bundles with a dependency-valid topological wave
  graph. At minimum distinguish bootstrap; identity/KMS and network; ECR/
  object/observability; security; endpoints/load balancer; artifact
  publication; application DB; identity DB; backup; and runtime roles.
- [ ] Plan each wave only after the real outputs of its applied dependencies
  exist. Mock dependency outputs may validate structure but can never become
  an apply-approved plan.
- [ ] Reuse existing network, security, service-endpoint, identity/secrets,
  ECR, load-balancer, database, object-storage, observability, compute, backup,
  and artifact-contract modules.
- [ ] Instantiate separate application and Keycloak database ownership and
  backup scopes. Do not collapse identity recovery into application data.
- [ ] Add only review-proven missing controls such as WAF, access logging,
  deletion protection, Multi-AZ, scaling, or isolated restore resources.
- [ ] Require least-privilege IAM, private data planes, TLS-only access,
  customer-managed encryption where approved, immutable image digests, and
  secret references rather than values.
- [ ] Ensure runtime definitions actually mount/map secret values by stable
  names with least-privilege access and never expose values in user data,
  Terraform state, rendered plans, logs, evidence, or instance metadata.
- [ ] Classify native S3/Dynamo-style state-lock acquisition/release as an AWS
  mutation requiring explicit authorization, or adopt a reviewed no-lock/no-
  concurrency process. Never claim remote planning makes zero writes when a
  lock object is created.
- [ ] Redesign rollback and destroy bundles before use: bind the exact executed
  binary+JSON plan, prior image/config digest, state/resource/tag allowlist,
  caller/account/region, wrapper/policy hashes, cost, expiry, and residue query
  into their own reviewed authorization digest. Environment-selected,
  unbound plan paths are forbidden.
- [ ] Align the shared OCI promotion schema with Task 2 of the local plan:
  config digest, OCI manifest/index digest, archive hash, platform,
  provenance, SBOM, scan, and attestation/signature subjects are distinct.
- [ ] Freeze the TLS/DNS wave graph. An eligible pre-existing ACM certificate
  is read-only input. Otherwise model certificate request and Route 53/DNS
  validation as separate, explicit mutation authorizations with a stop between
  them; model validation/issuance verification and later load-balancer
  attachment as subsequent dependency-valid waves. Bind certificate ARN,
  hostname/SANs, issuer, status, expiry/renewal, validation-record digest,
  CAA, edge TLS policy, HSTS/cookie policy, overlap rotation, alert, and
  rollback/forward-fix evidence. Never make an unapproved DNS change.
- [ ] Run local Terraform format/test/validate, TFLint, Trivy, Terragrunt
  structural validation, OPA, cost-schema, ownership, wave, artifact-
  permission, rollback/destroy-substitution, secret, and broad-destroy denial
  gates with no AWS credentials.

**Verification**

Run:

    ./scripts/test-aws-preprod-architecture.sh

Expected: the complete multi-service topology and dependency-valid wave graph
pass local policy/native tests; no AWS call or plan/apply occurs.

**Acceptance**

- No AWS call occurs in this task.
- Every later wave has an exact plan/review/authorization/apply protocol, but
  its real plan is generated just in time from current upstream outputs.

### Task 4: Bootstrap Infrastructure And Publish Exact Artifacts

This is a tracking umbrella. Every plan-lock, plan, review, apply, publication,
signing, and verification action below is a separate current-task
authorization slice with a literal stop.

**Files**

- No source or local release-artifact change is permitted after local GO. An
  application source/image/config/contract/workload/data-schema correction
  invalidates AWS execution and returns to Local Preprod Task 2, every affected
  Task 3-9 gate, and Task 10 GO. An IaC-only correction repeats this plan's
  Task 3 architecture gate and every affected plan/apply/verify wave; its
  dependency graph is recorded before resumption.
- Create protected phase evidence and publication manifests.
- Create `scripts/aws-preprod-wave.sh` with `plan`, `apply`, and `verify`
  subcommands and substitution-mutation tests.
- Create `scripts/aws-preprod-publish-artifacts.sh` with separate `copy`,
  `sign`, and credential-free `verify` actions; retire and forbid the broader
  `scripts/aws-trial-publish-artifacts.sh` and
  `scripts/lib/aws-trial-control.mjs` from preprod authorization bundles.

**Work**

- [ ] For each Task 3 topological foundation wave: obtain plan/lock-write
  authorization; generate binary+JSON plan from current real upstream outputs;
  release the lock; independently review policy/cost/security and record the
  hash; obtain a separate apply authorization; execute only that bundle;
  verify actual state/cost; then stop before planning the next wave.
- [ ] Verify state encryption, locking, access, logging, recovery, and break-
  glass denial before the foundation phase.
- [ ] If Task 1 selected new certificate issuance, execute certificate request,
  DNS validation mutation, issuance verification, and later attachment as
  separate authorized waves. If it selected a pre-existing certificate, prove
  its bound identity/currentness again before attachment. Neither branch
  inherits network or load-balancer apply authority.
- [ ] Obtain a separate artifact-publication authorization, verify each OCI
  archive hash and canonical image-manifest digest, then use a pinned
  digest-preserving OCI registry-copy implementation that uploads exact blobs
  and original manifest bytes/media type without rebuilding or format
  conversion. Compare ECR's returned/raw manifest digest and bytes—not a Docker
  config ID or mutable tag—to the promotion manifest, then separately
  authorize signing/attestation of that exact ECR digest under the approved
  trust policy.
- [ ] Verify repository immutability, scan-on-push policy, lifecycle, cross-
  account denial, and digest equality.
- [ ] Stop before data/runtime apply and record actual cost/resource state.
- [ ] Reject any swapped plan bytes/path, JSON, manifest, image/archive,
  previous digest, account/region, wrapper/policy, expiry, or authorization
  input before Terraform or AWS runs.

**Verification**

Run each separately authorized slice as:

    ./scripts/aws-preprod-wave.sh plan <wave> <authorization-bundle>
    ./scripts/aws-preprod-wave.sh apply <wave> <reviewed-bundle>
    ./scripts/aws-preprod-wave.sh verify <wave> <reviewed-bundle>
    ./scripts/aws-preprod-publish-artifacts.sh copy <reviewed-copy-bundle>
    ./scripts/aws-preprod-publish-artifacts.sh sign <reviewed-signing-bundle>
    ./scripts/aws-preprod-publish-artifacts.sh verify <publication-evidence-root>

Expected: only approved wave resources exist; lock writes are recorded;
published OCI subject roots equal the local release root; each command stops
before the next slice.

**Acceptance**

- Build-once promotion is proven.
- Every applied wave has a state-bounded recovery disposition:
  forward-fix, retain with owner/cost/expiry, scoped destroy from an exact
  reviewed plan, or service/data recovery. Binary rollback is promised only
  for application runtime when N/N-1 image/config/schema compatibility is
  proven; forward-only database truth is never described as generically
  rollbackable.

### Task 5: Deploy Data And Runtime, Then Bootstrap Identity And Synthetic Data

This is also a tracking umbrella, not one combined authorization. Application
data, identity data, backup, runtime, secret population, migration, identity
bootstrap, loader, and mTLS configuration are separate slices.

**Files**

- Use only approved runtime configuration, secret references, migrations, and
  release artifacts.
- Produce protected deployment, migration, identity, and loader evidence.
- Reuse `scripts/aws-preprod-wave.sh`; create exact one-shot operation wrappers
  `scripts/aws-preprod-populate-secrets.sh`,
  `scripts/aws-preprod-migrate.sh`,
  `scripts/aws-preprod-bootstrap-identity.sh`,
  `scripts/aws-preprod-load-data.sh`, and
  `scripts/aws-preprod-configure-aviacore-mtls.sh`, plus command-contract tests.
- Create `scripts/test-aws-preprod-smoke.sh` with the same action-specific
  authorization/evidence contract as later cloud tests.

**Work**

- [ ] Generate/review/authorize/apply application DB, identity DB, backup, and
  runtime waves separately from current real upstream outputs. Each action
  binds its own plan, lock mutation, cost, rollback, and stop.
- [ ] Prove the runtime starts every approved gateway/web, API, worker,
  scheduler, data-feed, Keycloak, scanner, renderer, mail-integration, and
  observability role with exact image/command, private network, least-privilege
  IAM, mounted secrets, health/readiness, logs, restart, and scaling policy.
- [ ] Run one-shot forward migrations before application traffic and verify
  schema fingerprint, database roles, extensions, RLS/policies, and no seed
  data.
- [ ] Start normal runtime from immutable images and verify health, readiness,
  TLS, headers, private endpoints, secret mounts, and no debug/test routes.
- [ ] Bootstrap application Admin, provision all representative roles, and
  complete first-login/MFA through the real preprod domain.
- [ ] Configure the exact AviaCore mTLS producer identity and prove a safe
  handshake before any profile workload.
- [ ] Run a separately approved one-shot clean `smoke` loader, reconcile its
  complete identity/data/object/job manifest, run the authorized smoke action,
  then separately authorize and execute its whole-namespace cleanup. Prove the
  empty fingerprint before separately loading/reconciling clean `acceptance`.
  Do not run `realistic` or `stress` until their own data and capacity/cost
  approvals.
- [ ] Obtain separate explicit authorization immediately before secret
  population/rotation, migration, identity bootstrap, each `smoke`/
  `acceptance` load and cleanup, and mTLS credential configuration. None
  inherits runtime-apply authority.

**Verification**

Run the exact operation wrappers named by the reviewed bundles, then:

    ./scripts/aws-preprod-populate-secrets.sh <reviewed-bundle>
    ./scripts/aws-preprod-migrate.sh <reviewed-bundle>
    ./scripts/aws-preprod-bootstrap-identity.sh <reviewed-bundle>
    ./scripts/aws-preprod-configure-aviacore-mtls.sh <reviewed-bundle>
    ./scripts/aws-preprod-load-data.sh smoke <reviewed-bundle>
    ./scripts/test-aws-preprod-smoke.sh <reviewed-smoke-bundle>
    ./scripts/aws-preprod-load-data.sh cleanup <reviewed-smoke-cleanup-bundle>
    ./scripts/aws-preprod-load-data.sh acceptance <reviewed-bundle>
    ./scripts/aws-preprod-wave.sh verify runtime <reviewed-bundle>

Expected: healthy complete normal runtime, separate application/identity
recovery scopes, exact identities/data, and no public/internal/one-shot
boundary violation.

**Acceptance**

- The environment is ready for bounded preprod verification, not production
  traffic.

### Task 6: Run Cloud E2E, Capacity, Security, And AviaCore Verification

**Files**

- Reuse local tests with an environment-bound configuration.
- Create protected preprod test evidence and redacted findings.
- Create `scripts/test-aws-preprod-release.sh` and its command/evidence
  contract; it consumes the canonical route-role-action-workflow matrix.

**Work**

- [ ] Execute all eight roles, 86 positive direct loads at three viewports,
  every role × route allow/redirect/403/404 result, every visible action/
  disabled reason, all 10 scenario families and named edge/identity/admin/feed/
  external/offline workflows, and raw API privacy/authority negatives with
  zero skip.
- [ ] Execute environment-bound accessibility against the exact HTTPS release:
  all-route automated rules, action-matrix keyboard/focus/dialog, contrast/
  touch/200%/400% reflow, and the accepted manual AT ledger.
- [ ] Refuse functional/accessibility/security execution unless Task 5 hands
  off one clean reconciled `acceptance` namespace with no `smoke` residue.
- [ ] Obtain separate cost authorization and data-loader authorization for a
  representative `realistic` manifest, then separate authorizations for
  acceptance load, bounded spike, a complete `stress` data/query/worker/feed-
  drain action, and a minimum 12-hour soak. Freeze numeric workload/threshold/
  capacity/scaling/cost inputs; do not treat local targets as production SLOs.
- [ ] After acceptance functional/accessibility/DAST, separately authorize
  whole-namespace cleanup; load/reconcile clean `realistic` for load/spike;
  clean it; load/reconcile clean `stress` for the stress action; clean it; then
  load/reconcile clean `realistic` for soak, failure/alert drills, and Task 7
  recovery. Record every before/after fingerprint. A mixed-profile namespace
  or a load action without its own current authorization fails.
- [ ] Measure ALB, compute, RDS, storage, identity, worker, data-feed, and
  external-service latency, saturation, scaling, errors, lag, and cost.
- [ ] Run cloud-specific DAST, WAF/rate, IAM, network exposure, TLS,
  certificate, secret rotation, object policy, backup, logging, and audit
  probes using synthetic data.
- [ ] Prove the public edge presents only the bound hostname/SAN certificate
  with the expected issuer, chain, key/signature type, TLS policy, expiry and
  managed-renewal/rotation state; verify CAA/validation records, HSTS,
  Secure/HttpOnly/SameSite cookies, wrong-host/old-certificate rejection, and
  certificate-expiry/renewal alerts. Never bypass TLS verification.
- [ ] Reconcile AviaSurveil source, producer outbox, AviaCore acknowledgement,
  raw landing, DQ, and governed products through the approved remote
  integration level.
- [ ] Exercise instance replacement, process restart, RDS connection
  interruption, object-service retry, certificate overlap rotation, and
  AviaCore outage without destructive regional experiments.
- [ ] Under separately authorized, bounded failure actions, prove every
  required CloudWatch/application alert fires with the expected owner/runbook,
  correlation and redaction; prove grouping/deduplication, notification
  delivery, silence/inhibition policy, recovery notification, and no alert
  storm or sensitive value.
- [ ] Require zero Critical/High security finding, zero P0/P1 product defect,
  zero privacy/tenant breach, and zero unexplained acknowledged data loss.

**Verification**

Run one separately authorized action at a time:

    ./scripts/test-aws-preprod-release.sh functional <reviewed-bundle>
    ./scripts/test-aws-preprod-release.sh accessibility <reviewed-bundle>
    ./scripts/test-aws-preprod-release.sh dast <reviewed-bundle>
    ./scripts/aws-preprod-load-data.sh cleanup <reviewed-acceptance-cleanup-bundle>
    ./scripts/aws-preprod-load-data.sh realistic <reviewed-realistic-bundle>
    ./scripts/test-aws-preprod-release.sh load <reviewed-bundle>
    ./scripts/test-aws-preprod-release.sh spike <reviewed-bundle>
    ./scripts/aws-preprod-load-data.sh cleanup <reviewed-realistic-cleanup-bundle>
    ./scripts/aws-preprod-load-data.sh stress <reviewed-stress-bundle>
    ./scripts/test-aws-preprod-release.sh stress-data <reviewed-bundle>
    ./scripts/aws-preprod-load-data.sh cleanup <reviewed-stress-cleanup-bundle>
    ./scripts/aws-preprod-load-data.sh realistic <reviewed-recovery-realistic-bundle>
    ./scripts/test-aws-preprod-release.sh soak <reviewed-bundle>
    ./scripts/test-aws-preprod-release.sh failure <reviewed-bundle>
    ./scripts/test-aws-preprod-release.sh alerts <reviewed-bundle>

After all remote actions have stopped, run the credential-free, no-AWS-call
aggregate only as:

    ./scripts/test-aws-preprod-release.sh verify <protected-evidence-root>

Expected: every approved action passes with its own subject/evidence/cost and
the offline aggregate proves the ordered four-profile ledger; otherwise the
environment remains NO-GO with specific rollback/recovery.

**Acceptance**

- Capacity and security claims are bound to the exact AWS topology and
  workload, not inferred from local runs.

### Task 7: Prove Backup, Isolated Restore, Failover, And Rollback

Restore, failover, rollback, retain, destroy, and residue inspection are
separate authorization slices. Approval of a recovery drill never authorizes a
destructive or continuing-cost action.

**Files**

- Use approved runbooks and create protected recovery evidence.
- Create `scripts/test-aws-preprod-recovery.sh` and protected rollback/destroy
  bundle verifiers.
- Create `scripts/aws-preprod-lifecycle.sh` with separately authorized
  `rollback`, `retain`, `scoped-destroy`, and `residue` actions. Only one of
  `retain` or `scoped-destroy` may be selected for a decision root; `residue`
  always has its own read-only authorization.
- Modify runbooks or infrastructure only through a new reviewed plan cycle if
  a gap is found.

**Work**

- [ ] Refuse to start unless Task 6 hands off a clean, exactly reconciled
  `realistic` namespace and proves no `smoke`, `acceptance`, or `stress`
  residue.
- [ ] Create two coordinated, immutable pre-loss recovery packages with
  distinct application/identity PITR positions and state changes between them.
  Each binds application DB, Keycloak DB, exact object versions, keys/secret
  versions, image/config roots, feed acknowledgement frontier, AviaCore
  checkpoint/publication references, workload/profile digest, and provider
  artifact checksums.
- [ ] Verify PITR and backup-provider artifacts in a separate failure domain
  allowed by residency policy.
- [ ] Restore both coordinated points into separate clean isolated targets
  without relying on the live primary. Corrupt or make the newest approved
  package unavailable in a bounded drill and prove deterministic fallback to
  the prior trusted point without synthesizing or rewriting either package.
- [ ] Verify exact fingerprints, roles/permissions, Keycloak users/roles/TOTP,
  real OIDC/MFA for all eight roles, all 86 positive route loads, object
  versions, worker delivery, data-feed replay, and AviaCore reconciliation at
  both points.
- [ ] Exercise approved AZ/instance failover and measure owner-approved RPO/RTO.
- [ ] Calculate RPO/RTO with the Task 1 clocks for application DB, identity DB,
  objects, secrets/keys/config, producer acknowledgement frontier, and AviaCore
  checkpoint/publication; record source-event, failure, recovery-point, and
  service-ready times per store.
- [ ] Before any binary rollback claim, prove N and N-1 application images
  against the current forward schema and prove N against the prior supported
  schema using exact API/OIDC/worker/feed smoke. If the previous image is not
  compatible with forward-only database truth, prohibit binary rollback and
  use the reviewed roll-forward/recovery path.
- [ ] Create rollback and destroy as separate protected bundles. Each bundle
  binds the actual executed binary+JSON plan, previous image/config digest,
  exact state/resource/tag allowlist, caller/account/region, wrapper/policy
  hashes, cost/expiry, and residue query. Resolve no plan from an unbound
  environment path.
- [ ] Obtain a new explicit authorization immediately before rollback and,
  separately, before retain or scoped destroy. Retain authorization binds its
  continuing-cost ceiling/expiry; destroy policy inspects JSON and permits only
  enumerated addresses/resources/tags.

**Verification**

Run each approved recovery action through:

    ./scripts/test-aws-preprod-recovery.sh restore-point-1 <reviewed-bundle>
    ./scripts/test-aws-preprod-recovery.sh restore-point-2 <reviewed-bundle>
    ./scripts/test-aws-preprod-recovery.sh corrupt-latest-fallback <reviewed-bundle>
    ./scripts/test-aws-preprod-recovery.sh failover <reviewed-bundle>
    ./scripts/test-aws-preprod-recovery.sh compatibility <reviewed-bundle>
    ./scripts/test-aws-preprod-recovery.sh rollback-or-roll-forward <reviewed-bundle>
    ./scripts/aws-preprod-lifecycle.sh rollback <reviewed-rollback-bundle>
    ./scripts/aws-preprod-lifecycle.sh retain <reviewed-retain-bundle>
    ./scripts/aws-preprod-lifecycle.sh scoped-destroy <reviewed-destroy-bundle>
    ./scripts/aws-preprod-lifecycle.sh residue <reviewed-residue-bundle>

`rollback` runs only when the compatibility result permits it. Exactly one of
`retain` or `scoped-destroy` runs; the other must be recorded as
`NOT_SELECTED`, not skipped evidence. `residue` follows the selected branch
under a new read-only authorization. Every command stops before the next.

Expected: two independently usable restored environments, deterministic
corrupt-latest fallback, measured per-store RPO/RTO, a truthful compatible
rollback or mandatory roll-forward result, no substituted plan/input, and no
resource outside the reviewed scope changed.

**Acceptance**

- A backup is accepted only after isolated functional restore.
- Rollback is claimed only when the exact compatibility matrix passes;
  otherwise the accepted recovery is explicitly roll-forward.

### Task 8: Issue The Preprod Decision And Deployment-Planning Recommendation

**Files**

- Create a protected AWS preprod evidence root.
- Create
  `docs/demo-evidence/AWS_PREPROD_VALIDATION_2026-07-27.md` with redacted
  references only.
- Create `scripts/verify-aws-preprod-evidence.sh` and its mutation contract.
- Modify this plan, the plan index, build summary, and technical-debt tracker.

**Work**

- [ ] Bind account/region, decisions, approvals, plans, applies, resources,
  costs, source/image/contract roots, migrations, identities, dataset,
  AviaCore evidence, tests, findings, capacity, alerts, recovery, rollback,
  residue, and retain/destroy result.
- [ ] Bind every authorization to its action-specific subject from the
  Authorization Model and verify the complete ordered
  `smoke -> acceptance -> realistic -> stress -> realistic recovery`
  transition ledger, empty fingerprints between profiles, isolated-restore
  cleanup, and zero mixed-profile residue.
- [ ] Reject the decision root if an application release change did not return
  through Local Preprod Task 2/affected gates/Task 10, or an IaC change did not
  repeat this plan's Task 3 and every affected wave.
- [ ] Obtain independent Product, Platform, Security, Operations, QA,
  Data/ML, Legal/Records, stakeholder, and release-authority review.
- [ ] Require all P0/P1 findings closed, no unexpired Critical/High finding,
  accepted preprod performance/RPO/RTO/capacity/cost, exact rollback, and
  staffed runbook/on-call ownership before `preprod verified`.
- [ ] Reconcile every open production debt item to an explicit later gate,
  owner, evidence requirement, and expiry. This includes production
  federation/email/data, signing/enforcement, Evidence storage/scanning/
  retention/legal hold/deletion/incident operations, source-bound AviaCore
  canary/pilot, MDM/device controls, production target/operations, release, and
  cutover.
- [ ] Distinguish:
  `preprod verified`, `qualified for production deployment planning`,
  application `production-ready`, production deployment authorized, and
  production traffic cutover authorized. This plan can issue only the first
  two; none implies another.
- [ ] Record GO or NO-GO with exact blockers and expiry.
- [ ] Verify the separately authorized Task 7 retain/destroy outcome and cost/
  resource residue. Do not execute either action from this decision task.

**Verification**

Run:

    ./scripts/verify-aws-preprod-evidence.sh
    node tests/harness-docs-smoke.test.js
    git diff --check

Expected: one independently verified, non-self-referential decision root with
no stale, missing, waived, mixed-profile, or unbound P0/P1 item. The verifier
has no AWS credentials and performs no remote query, retain, or destroy
action; it consumes only already protected outputs.

**Acceptance**

- `preprod verified` may be used only for the exact tested environment and
  release root.
- `qualified for production deployment planning` is not `production-ready`.
- `production-ready`, deployment, and traffic cutover require a separate
  production release/cutover plan and evidence; owner signatures cannot waive
  absent production-equivalent gates by themselves.

## Risks And Mitigations

| Risk | Mitigation |
|---|---|
| AWS spend begins before value exists | Entire plan paused until local GO; per-phase budgets and authorizations |
| Fixture plan is mistaken for real deployment | Read-only discovery, protected real plan hashes, phase-specific apply evidence |
| Tested artifacts differ from local candidate | Build-once digest publication and equality checks |
| Identity and application share one recovery fate | Separate database/backup ownership and joint functional restore |
| Public exposure exceeds gateway | Private subnets/endpoints, policy probes, exposure mutation tests |
| Cloud test overclaims production | Separate preprod, readiness, deployment, and cutover decisions |
| AviaCore is deployed from the wrong repository | AviaCore owns its endpoint; producer only configures approved mTLS delivery |
| Destroy removes unrelated resources | Exact allowlist, phase state, protected plan, scoped wrapper, residue audit |
| Single-container candidate is mistaken for deployable topology | Blocking multi-service/runtime/secret/two-database architecture gate |
| Dependent plans use mock or stale upstream outputs | Just-in-time topological plan/review/authorize/apply waves |
| Rollback/destroy plan is substituted after approval | Executed plan and resource manifest live inside their own authorization digest |
| Docker config ID is compared with ECR manifest digest | Shared OCI promotion schema and registry-subject equality |
| Preprod is called production-ready | Fixed preprod/deployment-planning labels and a separate production release/cutover gate |

## Idempotence And Recovery

- Discovery is read-only. Remote planning may acquire/release a state lock and
  is therefore a separately authorized mutation with its own evidence.
- An approved action bundle is immutable; any changed plan, JSON, upstream
  output, resource manifest, wrapper/policy, artifact, account/region, cost, or
  expiry creates a new review and authorization cycle.
- Apply wrappers inspect current state and refuse stale or mismatched plans.
- Every wave records resources before the next wave.
- Rollback/destroy targets only exact wave-owned state and resource IDs.
- Plan locking, publication, rollback, retain, destroy, and residue queries
  never inherit apply authority.
- Failed test/deploy evidence remains preserved and cannot be rewritten into a
  passing root.

## Decisions

- Do not spend on AWS until the local release candidate is complete.
- Use a separate preprod account boundary and the existing gated
  Terraform/Terragrunt foundation only after the runtime/topology and wave
  architecture gaps in Task 3 close.
- Build once locally and promote exact digests.
- Use synthetic acceptance data first; larger profiles require a separate
  capacity/cost approval.
- Keep AviaCore as a separately owned runtime and contract authority.

## Discoveries

- Plan 4 already provides reusable Terraform modules, Terragrunt composition,
  OPA/security/cost gates, protected plan wrappers, and apply/smoke/rollback/
  destroy command contracts.
- All real AWS actions in Plan 4 Task 10 remain `not run`.
- Existing local evidence cannot prove AWS quotas, network exposure, managed
  service behavior, host/AZ loss, target cost, or staffed operations.
- The current compute module runs one image and does not mount downloaded
  secret files into that container; the current composition models one
  database and cannot yet realize separate application/identity recovery.
- Current broad phase planning can use dependency mocks/stale values and native
  remote planning can write a state-lock object; it is not a zero-mutation
  reviewed-plan solution.
- Current rollback/destroy authorization does not bind the separately selected
  plan path, and current local Docker config IDs are not ECR manifest digests.

## Outcome Notes

Plan and independent-review artifact. AWS discovery, planning, apply, artifact publication,
deployment, migration, tests, restore, rollback, retain/destroy, Git
publication, and production actions are `not run`.

## Execution Prompt

```text
Execute docs/exec-plans/active/2026-07-27-aws-preprod-validation-plan.md only after the Local Preprod Release Candidate records GO and the user explicitly authorizes the exact next AWS action. Read AGENTS.md, docs/PLANS.md, this complete plan, the Plan 4 AWS decision/command contracts, the local release root, and all current owner inputs before any AWS call.

One approval authorizes only one exact action in one topological wave. Bind the common account/region/caller/cost/window/owner/next-stop fields and the action-specific subject required by the Authorization Model: discovery query/credential/script, plan inputs/state/lock, apply binary+JSON plan, OCI manifest bytes/copy/signing policy, one-shot executable/config/data/target, test tool/workload/profile/environment, restore package/target/clocks, or rollback/retain/destroy plan/allowlist/residue query. Plan locking, certificate request, DNS validation, publication/signing, secret population, migration, identity/data load and cleanup, every paid test, restore, rollback, retain/destroy, and residue queries require separate authorization. Stop before each next action. Never rebuild artifacts, use production/customer data, expose databases/storage/identity administration, use mutable tags, write secrets to files/evidence, perform broad destroy, deploy AviaCore from this repository, or infer production/cutover authority from preprod.

Use read-only discovery first, close the local multi-service/topological-wave architecture gate second, then plan/review/authorize/apply one dependency-valid wave at a time from real upstream outputs. Preserve evidence on failure and record literal GO/NO-GO. This plan can issue `preprod verified` and `qualified for production deployment planning`, never `production-ready`. Do not commit, push, deploy to production, or route traffic without separate authorization.
```
