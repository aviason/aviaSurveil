# AWS Private-Pilot Local Preparation Evidence

**Evidence date:** 10–11 August 2026  
**Plan:** [AWS Single-AZ ARM64 Private-Pilot Production Preparation](../exec-plans/active/2026-08-10-aws-single-az-arm64-private-pilot-production-plan.md)  
**Scope:** Tasks 1–6 local preparation only  
**Result:** `verified locally` for Tasks 1–5 and the available Task 6 gates;
native immutable mixed-workload capacity is `blocked`; `candidate-only`;
`release pending`; `production-ready: not established`

This document is the historical Tasks 1–6 local-preparation record. The later
separately authorized Task 7 read-only calls are recorded in
[AWS Private-Pilot Task 7 Read-Only Discovery Evidence](AWS_PRIVATE_PILOT_TASK7_DISCOVERY_2026-08-11.md).
The no-provider-call statement below applies to the Tasks 1–6 evidence window.

During that historical Tasks 1–6 window, no AWS, Cloudflare, external SMTP,
DNS, certificate, remote-state, lock,
provider-backed plan, apply, publication, RDS migration, identity/data load,
deployment, smoke, recovery, release, traffic, rollback, retain/destroy, or
external residue action was performed. Task 7 was then unauthorized and `not
run`; the later read-only wave is recorded separately.

## Prepared Local Boundary

- The owner decision contract is fail closed. Its committed example is
  intentionally incomplete and exits with `missing-owner-input`.
- Future operator-side AWS CLI, Terraform, and Terragrunt actions require the
  named profile `avia`. `default` and an omitted profile are rejected. EC2
  runtime containers receive no profile or static AWS key and use the IAM
  instance-profile credential-provider chain.
- The production Compose definition contains only gateway, web, API, worker,
  scheduler, enabled data-feed worker, Keycloak, internal Gotenberg Chromium,
  and bounded bootstrap/migration jobs. Local PostgreSQL, Keycloak PostgreSQL,
  MinIO, Mailpit, ClamAV, init/tooling, fixture/loader, backup-MinIO, and LGTM
  services do not appear there.
- API and worker preserve the public HTTPS OIDC issuer while using the
  Compose-internal Keycloak discovery endpoint. Keycloak readiness uses its
  explicit internal management path, and the production scheduler owns a
  bounded `verify-full` loop instead of the local scheduler entrypoint.
- The production object path preserves bucket/key/version/ETag/SHA-256/size.
  Only an exact-version `NO_THREATS_FOUND` GuardDuty tag followed by a matching
  exact-byte verification and a second unchanged exact-version tag read can
  return `CLEAN`; missing, stale, changed, failed, threat, tampered, or
  mismatched results fail closed.
- Public SMTP requires verified implicit TLS or mandatory STARTTLS. The local
  Mailpit plaintext mode remains explicitly private/local and cannot satisfy
  the production profile.
- The 2026-08-11 focused-IaC revision expresses Cloudflare Tunnel → a
  loopback-only gateway on one private dual-stack ARM64 `t4g.small`, zero
  runtime ingress, one egress-only IGW, no ALB/NAT/EIP/public subnet/public
  IPv4/ACM origin certificate, two private RDS subnets, one private Single-AZ
  `db.t4g.micro`, one S3 Gateway Endpoint, private encrypted/versioned S3,
  GuardDuty plans, IPv6-capable ECR, least-privilege IAM/secrets/SSM,
  CloudWatch Agent/logs/metrics, AWS Backup, and Budget. The connector token is
  a separately populated KMS-encrypted Standard SSM parameter and never enters
  Terraform state. Prohibited service/resource mutations fail offline tests.
- The release contract binds all eleven immutable target-account IPv6 ECR
  `linux/arm64` subjects,
  OCI/config/SBOM/provenance/vulnerability evidence, migrations, Compose,
  gateway, decision/policy, RDS CA, runtime environment, and data-feed public
  trust inputs. It rejects runtime secret values and credential/metadata
  endpoint redirects; systemd requires root-owned non-symlink mode-`0600`
  inputs. The separately supervised connector is included as a subject but is
  absent from Compose. Execution remains dry-run-only and rejects unapproved
  remote actions.

## Fresh Verification Matrix

| Gate | Fresh result |
|---|---|
| Private-pilot decision, Compose, IaC, and release Node contracts | 90/90 passed after the Tunnel/IPv6 revision; unsafe topology, `default`/omitted operator profile, malformed/cross-account/cross-region identities, runtime profiles/static keys, changed digests, external/mutable images, insecure SMTP, token-in-state, and remote authority were rejected |
| Terraform/Terragrunt formatting | Passed for the focused module, environment, and fixture |
| Shell syntax and committed JSON parsing | Passed |
| Trivy IaC HIGH/CRITICAL | Passed with 0 findings using embedded checks; vulnerability database/check updates were not attempted |
| Focused Go race suites | Passed for object store, scanner, Evidence worker, notifications, config, API, and worker; the previously recorded scheduler, data-feed worker, and document results remain unchanged |
| Broad Go packages | `go test -count=1 ./internal/... ./cmd/...` passed |
| Local PostgreSQL/MinIO/ClamAV/Mailpit integration | Fresh rerun passed through migration 43, exact object identity, duplicate/crash recovery, clean and EICAR scan paths, scanner-loss fail close, delivery/outage/restart, exact metadata, and zero task-owned Docker residue; ClamAV remains local-only |
| OpenAPI contracts | 16/16 passed |
| React typecheck and unit/component suite | Typecheck passed; 85 files / 745 tests passed |
| React demo/HTTP builds and app-shell inventories | Both builds passed; demo 78 files / 73 assets and HTTP 79 files / 74 assets passed |
| Root legacy JavaScript | 103/103 passed |
| Selected OpenAPI, operations, recovery, local-image/profile, runtime, and parity contracts | 113/113 passed |
| Final focused harness/operations/runbook/recovery/local-profile contracts | 83/83 passed after plan, index, tracker, runbook, and evidence synchronization |
| Docker task-owned cleanup | Zero task-owned private-pilot integration containers, volumes, or networks remained |
| Diff hygiene | `git diff --check` passed at the recorded closeout gate |

The local host reported `arm64` with Go `1.26.4 darwin/arm64`, Node
`24.19.0`, npm `11.17.0`, Terraform `1.15.6`, Terragrunt `1.0.8`, and Trivy
`0.72.0`. TFLint and OPA executables were unavailable, so their direct gates
are `not run`; the Rego source and unsafe-plan mutations were still exercised
by offline Node contracts.

`npm ci` restored the exact locked web dependencies and reported six existing
dependency advisories (one moderate and five high). No automatic audit fix or
dependency mutation was authorized. This result is not production image
vulnerability evidence.

## Blocked And Not-Run Gates

The machine is ARM64, but that does not establish EC2 capacity. The committed
owner overlay is absent and the release image lock intentionally contains no
resolved immutable ARM64 subjects, SBOMs, provenance, or vulnerability
evidence. Running a supposed production mixed workload would therefore test a
different artifact. The native production container/capacity gate is
`blocked`; no memory/headroom, CPU-credit, disk, restart, connection-pool,
render, or mixed-workload claim is made, and no `t4g.small` GO or `NO-GO` was
recorded.

Terraform provider initialization/tests/validation/plan and Terragrunt
provider-backed planning are `not run`. AWS/Cloudflare discovery, price/quota
checks, remote state/locks, SMTP transport against a public provider, image
publication/signing, real GuardDuty behavior, RDS bootstrap/migration,
CloudWatch delivery, backup/restore, RPO/RTO, production identity/MFA/data,
deployment, smoke, release, traffic, rollback, retain/destroy, and residue
inspection are all `not run`.

The AWS account, `eu-central-1` region, candidate AZs, and owner-attested T4g
eligibility were resolved in the later read-only wave. The complete deployable
owner overlay still lacks the exact domain, Cloudflare zone/Tunnel, scoped
bearer token, SMTP endpoint, globally unique names, storage/budget allocations,
immutable release inputs, records/legal/retention/deletion decisions,
alert/on-call owner, recovery acceptance, preprod disposition, and release
authority. Those inputs remain `blocked`. The recorded Task 7 AWS read-only
wave does not authorize another action; a separate exact authorization is
required before provider planning or any remote/cost-bearing action.
