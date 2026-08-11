# AWS Private-Pilot Owner Decisions

This is the fail-closed owner-input contract for the AWS Single-AZ ARM64
private-pilot candidate. It records the accepted architecture but does not by
itself authorize a provider plan, apply, secret population, deployment, or
traffic action.

## Accepted Architecture

- Edge: a remotely managed Cloudflare Tunnel is the only public application
  path. A separately supervised, digest-bound ARM64 `cloudflared` container
  opens outbound IPv6 connections on TCP/UDP 7844 and reaches the gateway only
  through `127.0.0.1`. The EC2 security group has zero ingress rules. There is
  no ALB, public subnet, ACM origin certificate, origin-auth header, AWS WAF,
  public IPv4, Elastic IP, or direct-origin route.
- DNS ownership: production Tunnel creation and application DNS cutover are
  separate waves. An existing hostname record is preserved while production
  infrastructure is prepared. The base owner overlay must keep DNS cutover
  authorization `false`; changing or importing the record requires a later
  exact plan and separate authorization.
- Network: one dual-stack application subnet in workload AZ A retains private
  IPv4 for RDS and the S3 Gateway Endpoint and assigns one global IPv6 address
  for outbound traffic through one egress-only Internet Gateway. There is no
  IPv4 default route, NAT Gateway, ordinary Internet Gateway, or interface
  endpoint fleet. The two private RDS subnets in AZ A/B remain structural
  requirements for the DB subnet group.
- Runtime: one On-Demand `t4g.small`, `linux/arm64`, with the approved
  application roles in production Compose and `cloudflared` in a separate
  hardened systemd-owned container. Only the gateway publishes a host port,
  and that port is fixed to loopback. Every runtime subject is an immutable
  `linux/arm64` digest mirrored into the target account's IPv6-capable ECR
  `.on.aws` registry path.
- Data: one private encrypted Single-AZ `db.t4g.micro` in a two-AZ DB subnet
  group, with isolated `aviasurveil360` and `keycloak` databases and runtime
  roles. Bootstrap authority is bounded and removed from normal runtime.
- Objects: private encrypted versioned S3 through the instance profile and AWS
  credential-provider chain. Static production keys and MinIO fallback are
  forbidden. Exact bucket/key/version/ETag/SHA-256/size identity is required.
- Scan: standalone GuardDuty Malware Protection for S3 with result tagging.
  Only the exact version tagged `NO_THREATS_FOUND` may become `CLEAN`.
- Email: the SMTP relay must publish a usable IPv6 AAAA record and pass
  certificate-verified IPv6 TLS/STARTTLS preflight. There is no hidden NAT or
  plaintext fallback when the provider is IPv4-only. The private-pilot target
  has no AviaCore/data-feed runtime integration; its domain source remains
  dormant and its tables/history are preserved.
- Secrets: the Cloudflare connector token is populated only by a separately
  authorized operation into one KMS-encrypted SSM Standard SecureString. The
  Terraform configuration creates a non-runnable placeholder with a
  write-only value and never reads the connector token into Terraform state.
- Recovery: RDS PITR starts at 14 days, AWS Backup at 35 days, CloudWatch logs
  at 30 days, and engineering targets are RPO <= 15 minutes and RTO <= 4
  hours. These are not legal-retention decisions.
- Capacity: a mixed native ARM64 run must retain at least 15% host-memory
  headroom and stay below 70% disk use without sustained swap, OOMs, or
  unexpected restarts. Failure is literally `NO-GO for t4g.small`.
- Cost: the Tunnel revision's expected fixed range is USD 26.91–40.69/month
  while the owner-attested T4g offer applies through 31 December 2026 and USD
  40.93–54.71/month afterward. Usage-driven storage, scans, requests, logs,
  backup, transfer, SMTP, and Cloudflare plan charges remain additional.

Production PostgreSQL, Keycloak PostgreSQL, MinIO, Mailpit, ClamAV,
backup-MinIO, local init/tooling, fixtures/loaders, and LGTM services are
forbidden. Multi-AZ runtime/RDS, a second EC2, autoscaling, ECS/EKS,
amd64/emulation, public RDS/S3, WAF, SES, ALB, NAT, and public IPv4 are also
forbidden.

## Required Private Overlay

The committed repository intentionally contains no deployable decision file.
Create an untracked `0600` JSON file under a `0700` directory and provide its
path to the checker. The following is a shape, not valid owner input:

```yaml
schemaVersion: 1
decisionId: "<required>"
issuedAt: "<required UTC timestamp>"
expiresAt: "<required UTC timestamp>"
authorization:
  scope: local-preparation-only
  remoteActionsAuthorized: false
  productionReleaseAuthorized: false
aws:
  accountId: "<required>"
  operatorProfile: "avia"
  operatorRoleArn: "<required>"
  region: "eu-central-1"
  workloadAvailabilityZone: "<required AZ A>"
  secondaryAvailabilityZone: "<required AZ B>"
cloudflare:
  accountId: "<required>"
  zoneId: "<required>"
  hostname: "<required>"
  tunnelName: "<required>"
  connectorParameterArn: "<exact SSM parameter ARN>"
  apiTokenPermissions:
    - account:cloudflare-tunnel:edit
    - zone:zone:read
    - zone:dns:edit
  dnsCutoverAuthorized: false
  connectorTokenInTerraformState: false
  connectorTokenRotationOwner: "<required>"
network:
  publicSubnets: 0
  dualStackAppSubnets: 1
  privateDatabaseSubnets: 2
  natGateways: 0
  internetGateways: 0
  egressOnlyInternetGateways: 1
  runtimeSecurityGroupIngressRules: 0
  databaseSecurityGroupIngressRules: 1
  ipv4DefaultRoute: false
  ipv6DefaultRoute: egress-only-internet-gateway
  origin: cloudflare-tunnel-ipv6-loopback
  gatewayBindAddress: 127.0.0.1
  cloudflareTunnelEdgeIpVersion: 6
  cloudflareTunnelPort: 7844
  cloudflareTunnelIpv6Cidrs: ["<fresh reviewed IPv6 CIDRs>"]
smtp:
  provider: "<required>"
  hostname: "<required>"
  port: "<465 or 587>"
  transport: "<implicit-tls or starttls>"
  tlsServerName: "<required>"
  credentialSecretArn: "<required reference, never a value>"
  ipv6Required: true
  dnsAAAAVerified: true
  tlsIpv6PreflightVerified: true
  ipv6Cidrs: ["<reviewed relay IPv6 CIDRs>"]
release:
  preprodDisposition: "<required>"
  releaseAuthority: "<required>"
  rollbackOwner: "<required>"
  rollbackPredecessorManifest: "<required digest-bound manifest>"
  windowStart: "<required>"
  windowEnd: "<required>"
  retainOrDestroyDecision: "<required>"
records:
  retentionOwner: "<required>"
  legalHoldOwner: "<required>"
  deletionAuthority: "<required>"
cost:
  monthlyCeilingUsd: "<required>"
  oneRunCeilingUsd: "<required>"
pricingEvidence:
  checkedOn: "<fresh required date>"
  t4gEligible: "<required>"
```

`avia` is the only accepted operator-side AWS shared-config profile and
`eu-central-1` is the selected region. Terraform, Terragrunt, and every future
operator AWS CLI command must select `avia` explicitly; `default`, an omitted
profile, and ambient profile substitution fail closed. EC2 containers receive
no profile or static AWS key and use the instance profile.

The complete machine contract is
`scripts/lib/aws-private-pilot-decision.mjs`; it also requires immutable ARM64
subjects, service ownership, secret references, capacity limits, recovery
owners, budget recipients, and the exact accepted topology.

```bash
./scripts/check-aws-private-pilot-decisions.sh /absolute/private/decision.json
```

Missing, stale, contradictory, secret-bearing, over-budget, IPv4-only,
remotely authorizing, or world-readable input fails closed. Until a current
overlay passes, owner input is `blocked`, external execution is `not run`, the
artifact is `candidate-only`, release is `release pending`, and
`production-ready: not established`.
