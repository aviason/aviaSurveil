# AWS Private-Pilot Runtime Contract

This contract defines the production-only single-host runtime without
deploying it. `deploy/local/compose.yaml` remains the local integration
harness and is not a production surface.

## Ownership And Subjects

| Subject | Owner | Required boundary |
|---|---|---|
| Cloudflare Tunnel | Platform + Security | remotely managed exact hostname, outbound IPv6 only, connector token in one SecureString, four healthy connections |
| Gateway | Platform | loopback-only host port, exact public Host, internal admin/management denial |
| Gateway/API/worker | Product Engineering + Backend | immutable `linux/arm64` subjects, non-root, bounded health/resources/logs, redacted telemetry |
| Keycloak | Identity + Security | internal management plane, separate DB/role/SMTP credential, bounded JVM |
| Native Go PDF renderer | Backend | embedded Noto Sans, deterministic bounded A4 output, provenance-bound jobs |
| RDS | Platform + Recovery | private Single-AZ instance, separate DBs/roles, coordinated recovery |
| S3 and GuardDuty result | Platform + Security | private/versioned/encrypted, exact-version clean gate, no static keys |
| External SMTP | Platform + Identity | verified IPv6 TLS/STARTTLS, scoped credentials, quotas/bounce/incident owner |
| CloudWatch/AWS Backup/Budget | Operations | bounded retention, alerts/on-call, 35-day recovery points, cost ceilings |

Because the owner states that this unused AWS account is dedicated to this
project, the private-pilot Budget is account-wide. It is not filtered by tags,
so untagged data transfer, API, backup, or service charges cannot silently fall
outside the alert ceiling.

## Ports And Networks

- Cloudflare is the only public application listener. The `cloudflared`
  connector establishes outbound TCP/UDP 7844 connections over IPv6 and uses
  the remote exact-hostname configuration to reach
  `http://127.0.0.1:8080`.
- The host publishes exactly the gateway port on `127.0.0.1`. API `8080`,
  Keycloak `8080` and management `9000` are Compose-internal only. The worker
  exposes no listener; its reminder controller is supervised inside the worker.
- The EC2 ENI has private IPv4 plus one Amazon-provided global IPv6 address,
  no public IPv4, and a security group with zero ingress rules. An egress-only
  Internet Gateway carries `::/0`; there is no IPv4 default route, ordinary
  Internet Gateway, NAT Gateway, EIP, public subnet, or ALB.
- Private IPv4 remains deliberate for RDS and the S3 Gateway Endpoint. RDS
  accepts PostgreSQL only from the application security group. S3 denies
  public access and non-TLS requests.
- Cloudflare 7844 destinations are fresh reviewed IPv6 CIDRs. HTTPS reaches
  IPv6-capable AWS/public dependencies. SMTP is restricted to reviewed IPv6
  relay CIDRs. An IPv4-only SMTP, package, image, or AWS dependency produces a
  fail-closed `NO-GO`; no NAT fallback is created. AviaCore/data-feed is not a
  private-pilot dependency.

## Runtime Hardening

Every runtime subject, including `cloudflared`, is an immutable digest with a
verified `linux/arm64` manifest in the target account's `.on.aws` ECR path.
Every container runs non-root with `no-new-privileges`, all capabilities
dropped, bounded PIDs/CPU/memory, explicit health and stop grace, bounded local
log rotation, and the smallest writable tmpfs. No application container is
privileged, uses host networking, mounts the Docker socket, or contains a
secret literal. The separately supervised connector is the only container
allowed to share the host network namespace, solely so it can reach the
loopback gateway and expose loopback-only metrics; the EC2 security group still
has zero ingress.

Runtime secrets arrive through separately authorized instance-profile reads.
Application secret files remain root-owned `0600`. The connector token is held
under the root-owned `0700` runtime directory as a connector-UID-owned `0400`
file so the non-root connector can read it; its value never enters argv,
environment, user data, Terraform state, logs, or evidence. The committed SSM
placeholder cannot authenticate and runtime start rejects it.

Operator-side AWS tooling is pinned to profile `avia` and region
`eu-central-1`. EC2 containers receive no `AWS_PROFILE`, shared-credentials
file, static keys, web-identity override, or container-credential redirect.
IMDSv2 is mandatory, its IPv6 endpoint is enabled, and its response hop limit
is two for the approved Docker bridge credential-provider path.

Digest pulls use the ECR Docker credential helper with
`AWS_USE_DUALSTACK_ENDPOINT=true`, the API endpoint
`ecr.eu-central-1.api.aws`, and the registry form
`<account>.dkr-ecr.eu-central-1.on.aws`. S3 application traffic deliberately
continues over private IPv4 through the Gateway Endpoint rather than bypassing
it through a dual-stack public endpoint.

SSM Agent uses dual-stack endpoints. CloudWatch Agent and operator bootstrap
use IPv6-capable AWS endpoints. Package installation is forced to IPv6; a
repository that cannot be reached over IPv6 fails bootstrap instead of adding
NAT. Before Compose or the connector starts, the runtime preflight verifies
AAAA resolution plus certificate-verified IPv6 TLS for AWS control endpoints,
SMTP. The dormant data-feed package has no private-pilot endpoint, trust
material, secret, or egress prerequisite.

The host CloudWatch Agent exports cloud-init and bounded container logs to the
30-day encrypted host log group and publishes root-disk, memory, and swap
metrics under `AviaSurveil360/PrivatePilot`. A root-only systemd timer reads the
connector's loopback Prometheus endpoint, publishes the active HA-connection
count, and fails whenever it is below four. Cloudflare's local metrics and
CloudWatch records must remain free of token values and request headers.

The Compose unit owns deterministic render/start/health/drain/stop. A separate
hardened systemd unit owns `cloudflared`; it starts only after Compose is
healthy. Migration and database bootstrap are bounded one-shot jobs. Bootstrap
creates the two databases/roles; each migration wave enables the migrator only
for the exact run and returns it to `NOLOGIN` on success or failure. Normal
runtime validation never requires the bootstrap master credential, migration
credential, or Keycloak realm-import file to remain mounted; those bounded
files are removed after their separately authorized one-shot action.

Every PostgreSQL connection uses `verify-full` and the same reviewed,
release-digest-bound AWS RDS CA bundle. Each Go process is capped at four
connections by default and at eight maximum; Keycloak starts with one and is
capped at eight. The long-running total stays below the RDS
`max_connections=50` ceiling with operator headroom.

## Immutable Evidence And Failure Semantics

Uploads remain quarantine-only until their bucket, key, version ID, ETag,
SHA-256, and size match an exact GuardDuty result. Only
`NO_THREATS_FOUND` allows exact-version promotion and `CLEAN`. Missing,
mismatched, stale, tampered, unsupported, access-denied, failed, or threat
results remain non-reviewable and non-downloadable. At-least-once delivery,
lease expiry, retries, duplicate processing, and crash recovery remain
idempotent. Evidence/configuration history is append-only.

SMTP accepts only certificate-verified implicit TLS or mandatory STARTTLS over
IPv6. Keycloak additionally forces JavaMail server-identity checks, mandatory
STARTTLS when selected, and TLS 1.2 or newer. No public plaintext or IPv4/NAT
fallback is permitted.

## Release And Capacity Gate

The release manifest binds exactly seven OCI subjects: `cloudflared`,
`gateway`, `api`, `worker`, `keycloak`, `database-bootstrap`, and `migration`,
plus SBOM/provenance/vulnerability evidence, migrations, Compose, gateway, both
systemd units, runtime scripts, decision/policy digests, RDS CA, and runtime
environment. It rejects profiles/static keys,
credential endpoint redirects, direct secret values, changed subjects,
duplicate keys, unbound files, a non-loopback gateway, or an IPv4 connector.

Native ARM64 evidence must cover OIDC, the canonical workflow, one native
render, one exact scan promotion, email, four Tunnel connections, connection
pools, restarts, CPU, memory, disk, and CPU credits. Data-feed delivery is
retired from this runtime and is not a capacity or release prerequisite. A pass requires
at least 15% memory headroom, less than 70% disk use, no kernel OOM, no
unexpected restart, and no sustained-swap dependency. Otherwise record
`NO-GO for t4g.small`; do not remove roles or weaken controls.

This local contract is `candidate-only`. External validation is `not run`,
release is `release pending`, and `production-ready: not established`.
