# AWS IPv6 ARM64 Trial Decision Contract

**Status:** `candidate-only`; local contract implementation is `verified locally`;
AWS and Cloudflare actions are `not run`.

This file defines the owner-input boundary for the separate
`aws-ipv6-trial` environment. It is not an authorization to discover, lock,
plan, publish, apply, smoke-test, retain, destroy, or query AWS or Cloudflare.
The owner overlay is intentionally untracked and must never contain a provider
secret value in Git, shell history, logs, plan output, or evidence.

## Overlay location and validation

Create a mode `0700` directory and a mode `0600` JSON file at:

```text
.local/aviasurveil360/aws-ipv6-trial/decision.json
```

The path can be overridden for a single local check with
`AVIA_AWS_IPV6_TRIAL_DECISION_FILE`. The checker performs no network call and
fails closed when the file is absent, world-readable, stale, contradictory, or
over budget:

```bash
./scripts/check-aws-ipv6-trial-decisions.sh
./scripts/check-aws-ipv6-trial-decisions.sh /path/to/decision.json
```

The overlay is a decision package, not an apply authorization. Every remote
action still requires a new exact authorization bound to the account, region,
Cloudflare account/zone, action, reviewed plan or artifact hash, budget,
expiry, and retain/destroy choice.

## Required schema (`schemaVersion: 1`)

| Object / field | Required owner decision |
|---|---|
| `decisionId` | Unique review identifier; it does not grant apply authority. |
| `issuedAt`, `expiresAt` | ISO-8601 planning window. `expiresAt` must be in the future. |
| `aws.accountId` | Exact 12-digit account ID. |
| `aws.operatorRoleArn` | Exact approved read-only/planning operator role ARN. |
| `aws.region` | Approved region; there is no committed default. |
| `aws.availabilityZone` | One approved AZ after capacity discovery. |
| `cloudflare.accountId`, `cloudflare.zoneId` | Exact 32-character hexadecimal account and existing zone IDs. |
| `cloudflare.hostname` | Exact trial hostname. |
| `cloudflare.apiTokenEnv` | The literal environment variable name `CLOUDFLARE_API_TOKEN`; never the token. |
| `cloudflare.connectorTokenParameterName` | AWS SSM parameter name; the connector token value is populated only in an authorized edge/runtime-auth action. |
| `cloudflare.edgeIpVersion` | The literal number `6`; the runtime must set `TUNNEL_EDGE_IP_VERSION=6`. |
| `access.publicExposure` | Explicit `true` or `false`; no inference. |
| `access.allowedIdentities` / `allowedDomains` | Non-empty allowlist when public exposure is false; empty when it is true. |
| `runtime.profile` | The literal `demo`; `full` is not accepted. |
| `runtime.services` | Exactly `cloudflared`, `gateway`, and `web-demo`. |
| `runtime.architecture` | The literal `linux/arm64`. |
| `runtime.instanceType` | The literal `t4g.small`. |
| `runtime.amiSsmParameterName` | Explicit AWS-owned ARM64 AL2023 SSM public parameter. |
| `runtime.imageDigests` | Digest-bound `linux/arm64` subjects for all three services; no mutable tags. |
| `network` prohibitions | Explicit false for public IPv4, EIP, NAT, load balancer, RDS, interface endpoint, inbound SG rules, SSH, amd64, and emulation. |
| `cost.monthlyCeilingUsd`, `oneRunCeilingUsd` | Positive ceilings and projected amounts below each ceiling. |
| `cost.nonComputeCeilingUsd` | Positive bound that includes EBS, ECR, state, logs, transfer, snapshots, and CPU-credit risk. |
| `cost.alertRecipients` | At least one owner-controlled recipient. |
| `pricingEvidence` | Checked date, eligible regions, 750-hour offer, expiry, CPU-credit caveat, post-expiry price, and non-compute estimate. |
| `persistence.rootVolumeGiB` | Positive bounded gp3 size. |
| `persistence.deleteRootVolumeOnTermination` | Explicit deletion decision. |
| `persistence.snapshotRetained` | Explicit snapshot-retention decision. |
| `lifecycle.trialExpiresAt` | Exact trial expiry; it must not outlive the decision. |
| `lifecycle.retainOrDestroy` | `destroy-after-trial` or `retain-with-owner-expiry`. |
| `owners` | Named platform, security, DNS, cost, and rollback/destroy owners. |

The validator rejects secret-like keys (`apiToken`, `connectorToken`,
`password`, `privateKey`, and similar) and secret-looking values. Only source
references such as `apiTokenEnv` and `connectorTokenParameterName` are allowed.

## Example shape (deliberately incomplete)

The following is a documentation shape, not a usable decision. Empty values are
intentional so a copied file cannot accidentally authorize a plan:

```json
{
  "schemaVersion": 1,
  "decisionId": "",
  "issuedAt": "",
  "expiresAt": "",
  "aws": {
    "accountId": "",
    "operatorRoleArn": "",
    "region": "",
    "availabilityZone": ""
  },
  "cloudflare": {
    "accountId": "",
    "zoneId": "",
    "hostname": "",
    "apiTokenEnv": "CLOUDFLARE_API_TOKEN",
    "connectorTokenParameterName": "",
    "edgeIpVersion": 6
  },
  "access": {
    "publicExposure": false,
    "allowedIdentities": [],
    "allowedDomains": []
  },
  "runtime": {
    "profile": "demo",
    "services": ["cloudflared", "gateway", "web-demo"],
    "architecture": "linux/arm64",
    "instanceType": "t4g.small",
    "amiSsmParameterName": "",
    "imageDigests": {
      "cloudflared": "",
      "gateway": "",
      "web-demo": ""
    }
  },
  "network": {
    "ipv6OnlyRuntime": true,
    "publicIpv4": false,
    "elasticIp": false,
    "natGateway": false,
    "loadBalancer": false,
    "rds": false,
    "interfaceVpcEndpoints": false,
    "inboundSecurityGroupRules": false,
    "ssh": false,
    "amd64": false,
    "emulation": false
  },
  "pricingEvidence": {
    "checkedOn": "",
    "eligibleRegions": [],
    "trialHoursPerMonth": 750,
    "offerExpiresOn": "2026-12-31",
    "cpuCreditCaveat": true,
    "onDemandPriceAfterExpiryUsdPerHour": 0,
    "nonComputeMonthlyEstimateUsd": 0
  },
  "cost": {
    "monthlyCeilingUsd": 0,
    "oneRunCeilingUsd": 0,
    "projectedMonthlyUsd": 0,
    "projectedOneRunUsd": 0,
    "nonComputeCeilingUsd": 0,
    "alertRecipients": []
  },
  "persistence": {
    "rootVolumeGiB": 0,
    "deleteRootVolumeOnTermination": true,
    "snapshotRetained": false
  },
  "lifecycle": {
    "trialExpiresAt": "",
    "retainOrDestroy": "destroy-after-trial"
  },
  "owners": {
    "platform": "",
    "security": "",
    "dns": "",
    "cost": "",
    "rollbackDestroy": ""
  }
}
```

## Literal evidence boundary

The contract can establish only `verified locally` for the shape and
fail-closed behavior. It does not establish ARM64 capacity, cloud connectivity,
Cloudflare Access, AWS cost, release readiness, or production readiness. Those
remain `not run`; the trial remains `candidate-only`, `release pending`, and
`production-ready: not established`.
