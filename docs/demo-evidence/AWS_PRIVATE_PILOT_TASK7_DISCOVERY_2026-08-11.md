# AWS Private-Pilot Task 7 Discovery And Bootstrap Plan Evidence

**Evidence date:** 11 August 2026  
**Plan:** [AWS Single-AZ ARM64 Private-Pilot Production Preparation](../exec-plans/active/2026-08-10-aws-single-az-arm64-private-pilot-production-plan.md)  
**Scope:** the separately authorized Task 7 read-only discovery waves, the
exact remote-state bootstrap provider-plan wave, and its exact apply  
**Result:** AWS identity, inventory, offer, quota, IAM, current list-price, and
Cloudflare account/zone/hostname/Tunnel discovery completed; the focused
Terraform bootstrap plan is reviewed at 8 additions, 0 changes, and 0 destroys;
the apply created and verified seven resources, then stopped `blocked` on the
denied KMS alias creation; policy v2 is corrected and the residual alias-only
plan was reviewed and applied at 1 addition, 0 changes, and 0 destroys; all
eight bootstrap resources are verified; `candidate-only`; `release pending`;
`production-ready: not established`

The user authorized AWS and Cloudflare Task 7 work on 11 August 2026. These
waves performed identity, topology, offer, permission, and token-validity
reads only and did not initialize remote state, create a lock, or mutate a
resource. The user later authorized only a provider-backed remote-state
bootstrap plan with explicit `avia` and `eu-central-1`. That wave initialized a
local planning directory and generated the protected plan bundle without a
remote backend or state lock. The user then supplied the exact apply
authorization. The apply created seven resources and stopped on a denied KMS
alias operation. It did not publish an image, change DNS/Cloudflare/runtime
secrets, run a migration, deploy, send SMTP, load identity/data, execute
smoke/recovery, route traffic, roll back, retain, destroy, or inspect unrelated
remote residue.

## AWS Results

- The named operator profile is exactly `avia`; its configured default region
  is now `eu-central-1`. The AWS `default` profile was not changed or used.
- STS resolved the expected 12-digit account and the `avia-aws-user` IAM user.
  The exact account and caller ARN are retained only in the ignored,
  mode-`0600` discovery record under
  `.local/aviasurveil360/aws-private-pilot/discovery.json`.
- A later read-only follow-up reconfirmed the same account and caller through
  an explicit `--profile avia --region eu-central-1` STS call. No default or
  omitted profile was used.
- `eu-central-1a`, `eu-central-1b`, and `eu-central-1c` are available to the
  account and each currently offers `t4g.small`.
- The account currently exposes only its default `172.31.0.0/16` VPC through
  the authorized read, so the proposed private-pilot CIDR must not overlap it.
- The newest returned Amazon-owned AL2023 kernel-6.1 ARM64 image was created on
  3 August 2026 and has a 1 November 2026 deprecation time. Its exact AMI ID,
  name, owner, and timestamps are retained in the private discovery record.
- The public SSM AL2023 ARM64 parameter returned version 183 and the same exact
  image ID as the independent Amazon-owned EC2 image-catalog result.
- RDS reports PostgreSQL 18.3 as the current default. `db.t4g.micro` with gp3
  and storage encryption is orderable in the three returned Availability
  Zones. This does not establish application compatibility or capacity.
- Current account quotas are sufficient for the active shape: five running
  On-Demand Standard instances, five VPCs, 40 RDS instances, and 50 DB subnet
  groups. The discovered EIP, NAT Gateway, ALB, and target-group quotas are
  retained only as evidence for the superseded shape. Quota headroom does not
  authorize creation.
- No ACM certificate, S3 bucket, ECR repository, `avia` KMS alias, or `avia`
  SNS topic was returned. ACM is not used by the active Tunnel architecture;
  exact globally unique resource names and the alarm-topic branch remain
  owner inputs.
- A follow-up `avia` read found no bucket at the proposed remote-state name
  `aviasurveil360-private-pilot-tfstate-357601816046-eu-central-1` and zero
  existing aliases named
  `alias/aviasurveil360-private-pilot-terraform-state`. These are unreserved
  candidates only: no bucket/key was created and global S3 name availability
  can change. The later provider-backed bootstrap plan uses these exact names.

## Remote-State Bootstrap Provider Plan

The user's exact authorization was limited to: `avia profili ve eu-central-1
ile yalnız remote-state bootstrap provider planını üret; apply yapma.` The
result was generated with Terraform 1.15.6, Terragrunt 1.0.8, and HashiCorp AWS
provider 6.58.0. The provider was pinned to profile `avia`, allowed account
`357601816046`, and region `eu-central-1`; the default AWS profile was not used.

The first initialization attempt selected OpenTofu implicitly. No plan from
that path was accepted. The bootstrap hook was then hardened to require the
explicit Terraform CLI before planning, and the final saved plan contains:

- 8 additions, 0 changes, and 0 destroys;
- one KMS key with rotation and a 30-day deletion window plus its alias;
- one `force_destroy=false` S3 state bucket with BucketOwnerEnforced ownership,
  all four public-access blocks, versioning, AWS KMS encryption, and a bucket
  policy; and
- approved tags `Owner=platform-operations`,
  `CostCenter=aviasurveil360-private-pilot`, and
  `DataClassification=restricted`.

The protected mode-`0600` bundle is retained under
`.local/aviasurveil360/aws-private-pilot/plans/20260811T064631Z-remote-state-bootstrap/`.
It contains the binary and JSON plans, native Terraform provider lock file,
inputs, authorization receipt, hashes, and review manifest. The binary plan
SHA-256 is
`2b13b5d70fda4b8415e7087bba7609ede75eef7c784ca1cd415c8ed55c3b2bf3`;
the JSON plan SHA-256 is
`14b265957d76c8192b15f74787ae6c02864ced0a105692ee70f77a6f2fde0e53`;
the aggregate source/lock/input/plan SHA-256 is
`87e9ca524f9eb1a7c7714e1789cc6599a630e1eab004054b0dcd96f5f429ed03`.
The secret scan passed, and hash verification passed.

The reviewed expected fixed cost was approximately USD 1/month for the
customer-managed KMS key, plus KMS requests and S3 state bytes, versions, and
requests. That fixed key charge starts once the apply creates the key.

## Remote-State Bootstrap Apply

The user supplied the required exact authorization for account
`357601816046`, region `eu-central-1`, and aggregate SHA-256
`87e9ca524f9eb1a7c7714e1789cc6599a630e1eab004054b0dcd96f5f429ed03`.
An explicit `avia` STS call reconfirmed the expected caller immediately before
the saved binary plan was applied.

Terraform created these seven state-managed resources:

- `aws_kms_key.state`;
- `aws_s3_bucket.state`;
- `aws_s3_bucket_ownership_controls.state`;
- `aws_s3_bucket_policy.state`;
- `aws_s3_bucket_public_access_block.state`;
- `aws_s3_bucket_server_side_encryption_configuration.state`; and
- `aws_s3_bucket_versioning.state`.

`aws_kms_alias.state` failed `blocked` with `AccessDeniedException` for
`kms:CreateAlias`. Read-only inspection of the attached
`AviaPrivatePilotRemainingServicesEuCentral1` policy found that it permits
`alias/avia-private-pilot-*`, not the approved
`alias/aviasurveil360-private-pilot-terraform-state`. No IAM mutation or retry
was authorized or attempted.

The partial local state is retained mode `0600` as
`bootstrap__remote-state.partial.tfstate` in the protected plan bundle. Its
SHA-256 is
`71833f3bc904587ecf81a917e1e59ed68aa0202f8792228f1d0483dba0a1fd2f`.
Post-apply reads verified:

- the KMS key is enabled, single-Region, customer-managed, and has annual
  rotation enabled;
- S3 versioning is enabled;
- all four S3 public-access block controls are true;
- S3 ownership is BucketOwnerEnforced;
- default encryption uses the exact KMS key with bucket keys enabled; and
- the bucket policy status is non-public.

The fixed KMS key charge has now started. The bootstrap remained incomplete
until the alias permission could be corrected and a new residual plan reviewed.

## Residual Alias Provider Plan

The owner updated the customer-managed policy to default version v2. Read-only
verification confirmed that `kms:CreateAlias` now permits the exact
`alias/aviasurveil360-private-pilot-*` prefix. The user separately authorized
only the residual provider plan; apply remained excluded.

Terraform refreshed the protected seven-resource local state and produced
exactly:

- 1 addition: `aws_kms_alias.state`;
- 0 changes; and
- 0 destroys.

The planned alias is
`alias/aviasurveil360-private-pilot-terraform-state` in `eu-central-1` and
targets existing key `ee98bef9-fcd6-4fd4-ba9b-a6fb8ec87850`. The other seven
managed resources are unchanged. The protected mode-`0600` bundle is under
`.local/aviasurveil360/aws-private-pilot/plans/20260811T073238Z-remote-state-bootstrap-residual-alias/`.
Its binary plan SHA-256 is
`eac63dfac3a3ae0396858d5487bf9b2931790a487d1218a713af9b33a04ef427`,
JSON plan SHA-256 is
`1feede93dfdf9d439844841b3e6e3663fe88aaf5ce4623ad805f0f9c50c2771f`,
and aggregate SHA-256 is
`1c525dbd8aa0c3b2317c94a5d2b1457fd83304f1ca244696e169ad2f32eabc2a`.
The secret scan and stored hash verification pass. A KMS alias has no
additional fixed monthly cost. The user subsequently supplied exact apply
authorization bound to the aggregate digest. Terraform applied only
`aws_kms_alias.state` with 1 addition, 0 changes, and 0 destroys. A read-only
KMS lookup verified that the alias resolves to key
`ee98bef9-fcd6-4fd4-ba9b-a6fb8ec87850`.

The protected mode-`0600` final state contains eight resources and has SHA-256
`b2d4016b338b0b08b58701a9a43c92615699f91cc52f94d90923318c869ab18f`.
The protected outputs SHA-256 is
`71fe9c763f7a861d66aef2ead9fab98713b5a9ba0885eeb258cba6bb54d3af00`.
At that checkpoint, remote backend migration and state-lock use remained `not
run`; the following separately authorized wave completed them.

## Remote Backend Migration Preparation

After the owner directed the migration to proceed, a read-only S3 prefix check
confirmed that
`aws-private-pilot/bootstrap/remote-state/` contains no existing state or lock
object. A protected migration bundle pins profile `avia`, account
`357601816046`, region `eu-central-1`, the exact eight-resource state hash,
state key `aws-private-pilot/bootstrap/remote-state/terraform.tfstate`, native
lock key with `.tflock`, and the existing KMS key. Its aggregate SHA-256 is
`d7b2763d9c7ac8b187318a15fb8e397904680884b7ab1ad95ec7b5cc86464792`.

The user supplied the exact authorization bound to that aggregate. Terraform
migrated the eight-resource state to the S3 backend. The backend rebased state
metadata from local lineage/serial
`af48e52c-d75d-e8e8-ba71-26bc968dbe69`/`11` to remote
`56eaf048-41fd-823d-6256-0a6bb4f0cce2`/`1`; after removing those metadata
fields and canonicalizing check-result order, source and remote state are
semantically identical at SHA-256
`a91de8aa8b23d81e23c00bc274937c4bdcf8badb8f83d651d8ae28e665b83990`.

The migrated object is versioned as
`YtCUTRcqO9h6lC18lrNL1DNmMT.Sa38o`, encrypted by the exact KMS key, and uses
an S3 bucket key. The refresh-disabled lock probe returned 0 add/0 change/0
destroy. Migration plus the probe produced two `.tflock` object versions and
two corresponding delete markers; the state object version remained unchanged
during the probe. Terraform apply, provider resource mutation, state deletion,
force-unlock, and all runtime/Cloudflare actions were not run.

## Frankfurt Cost Report And Architecture Revision

AWS Pricing API reads used `--profile avia`; `us-east-1` was only the Pricing
API endpoint and every product filter targeted `EU (Frankfurt)`. At 730 hours,
the following prices were discovered for the initial, now-superseded
ALB/NAT/public-IPv4 shape:

| Component | Current list rate | Monthly handling |
| --- | ---: | ---: |
| One Linux `t4g.small` | USD 0.0192/hour | USD 14.016 before any T4g trial credit |
| One Single-AZ PostgreSQL `db.t4g.micro` | USD 0.019/hour | USD 13.87 |
| One NAT Gateway (superseded) | USD 0.052/hour plus USD 0.052/GB processed | USD 37.96 plus usage |
| One ALB (superseded) | USD 0.027/hour plus USD 0.008/LCU-hour | USD 19.71 plus usage |
| Three public IPv4 addresses (superseded) | USD 0.005/address-hour | USD 10.95 |
| Four customer-managed KMS keys | USD 1/key-month | USD 4.00 plus requests |
| Thirteen Secrets Manager secrets (superseded) | USD 0.40/secret-month | USD 5.20 plus requests |
| Seven standard CloudWatch alarms | USD 0.10/alarm-month list rate | USD 0 within the current 10-alarm-metric free tier |
| EC2 gp3 root, allowed 20–64 GiB | USD 0.0952/GiB-month | USD 1.904–6.0928 |
| RDS gp3, allowed 20–90 GiB | USD 0.137/GiB-month | USD 2.74–12.33 |

After reviewing those fixed charges, the user accepted an outbound-only IPv6
Cloudflare Tunnel. The active local IaC contains no ALB, NAT Gateway, EIP,
public subnet, public IPv4, or ACM origin certificate. It also removes the two
Secrets Manager origin-auth containers, leaving ten runtime secret containers
plus the RDS-managed secret. A Standard SSM connector SecureString adds no
fixed per-parameter charge. One egress-only Internet Gateway has no hourly
gateway charge.

| Active fixed component | Monthly amount |
| --- | ---: |
| One Linux `t4g.small` | USD 0 through 31 December 2026 under the owner-attested offer; USD 14.016 afterward |
| One Single-AZ PostgreSQL `db.t4g.micro` | USD 13.87 |
| Four customer-managed KMS keys | USD 4.00 plus requests |
| Eleven billed Secrets Manager secrets | USD 4.40 plus requests |
| Seven standard CloudWatch alarms | USD 0 within the current 10-alarm-metric free tier |
| EC2 gp3 root, allowed 20–64 GiB | USD 1.904–6.0928 |
| RDS gp3, allowed 20–90 GiB | USD 2.74–12.33 |
| ALB, NAT Gateway, EIP/public IPv4, public subnet, ACM origin certificate, SSM Standard parameter | USD 0 fixed |

The active expected fixed range through 31 December 2026 is therefore **USD
26.914–40.6928/month**, reported as **USD 26.91–40.69/month**. The post-offer
comparator is **USD 40.93–54.71/month**. Removing USD 68.62/month of ALB, NAT,
and public IPv4 charges plus USD 0.80/month of origin-auth secrets saves **USD
69.42/month** versus the superseded shape. The storage bounds are validator
bounds, not an owner-approved allocation.

The architecture still adds S3 data/versions/requests/tags, seven ECR
repositories, GuardDuty scans, EventBridge events, CloudWatch logs, AWS Backup,
API calls, data transfer, external SMTP, Cloudflare plan, and surplus CPU
credit charges according to actual use. Because the owner states that the
otherwise unused account is dedicated to this project, the Budget covers the
whole account rather than filtering on resource tags; untagged service or data
transfer charges therefore remain inside the alert ceiling.

Relevant variable Frankfurt rates captured in the private record include S3
Standard at USD 0.0245/GiB-month for the first 50 TB, ECR at USD
0.10/GiB-month, CloudWatch Standard log ingestion at USD 0.63/GiB and storage
at USD 0.0324/GiB-month, and AWS Backup S3 warm storage at USD
0.06/GiB-month. GuardDuty Malware Protection for S3 includes 1,000 requests
and 1 GiB scanned per account/Region/month; above that allowance the current
Frankfurt rates are USD 0.000308/object and USD 0.129/GiB. S3 APIs, tagging,
and EventBridge remain separately charged.

The [EC2 FAQ](https://aws.amazon.com/ec2/faqs/) currently says all accounts
are eligible for one aggregate 750-hour/month `t4g.small` trial bucket,
including Frankfurt, through 31 December 2026 UTC; surplus CPU credits still
cost money. Account-specific remaining usage could not be read because
`freetier:GetFreeTierUsage` is not granted, so the allowance is supported by
the owner's unused-account attestation rather than an AWS usage-API result.
The post-expiry range remains the required Budget and expiry comparator. The
old public IPv4 price remains historical evidence on the
[VPC pricing page](https://aws.amazon.com/vpc/pricing/); the active design uses
none. AWS documents the malware allowance on the
[GuardDuty pricing page](https://aws.amazon.com/guardduty/pricing/).

## IAM Permission Resolution

The required SSM, Service Quotas, IAM inventory, ACM, S3, ECR, and Access
Analyzer reads now succeed. The verified caller has no inline policies and
seven directly attached policies: the broad EC2, RDS, ECS, S3, ELB, and
CloudFormation AWS-managed policies plus the customer-managed
`AviaPrivatePilotRemainingServicesEuCentral1` gap policy. ECS and
CloudFormation remain unrelated to the accepted topology; Codex did not
detach them because no exact IAM-removal action was authorized.

A single replacement customer-managed provisioning policy was generated only
in the ignored mode-`0600` local area for the superseded ALB/NAT shape. Its
syntax, size, and action names were validated at discovery time, but it is not
the authorization contract for the active Tunnel architecture and must not be
attached or reused without a fresh least-privilege review. The attached
3,955-character gap policy contains 75 action patterns for the remaining
services; AWS Access Analyzer returned zero findings for it. No
self-elevation, policy attachment/detachment, or other IAM mutation was
attempted by Codex.

## Cloudflare Results

- The user supplied a Cloudflare account identifier and later attached the
  metadata for the second identifier. It is the ID of the active
  `restless-block-1a54` API token, not its bearer secret and not a zone ID.
  The metadata reports no prior use and an 8 September 2026 expiry.
- The attached token policy is account-wide, grants a large unrelated set of
  read/write permissions, and does not establish the least-privilege Account
  `Cloudflare Tunnel Edit` plus exact-zone `Zone Read` and `DNS Edit` contract.
  It is rejected as both unusable without the bearer value and over-privileged.
- The old discovery returned 15 IPv4 proxy CIDRs. They are retained only as
  historical evidence and are no longer an IaC input. The active connector
  needs the freshly reviewed Cloudflare Tunnel IPv6 edge ranges on TCP/UDP
  7844.
- A root-private token location exists at
  `/private/tmp/avia-cloudflare.9eCjyW/cloudflare-api-token`; the file is mode
  `0600`. A first bearer-shaped value was rejected with code `1000`, `Invalid
  API Token`. The owner replaced it; the new bearer verified `active`. Neither
  value was printed or recorded.
- Read-only calls resolved the supplied account, one active
  `aviasurveil.com` zone, and the owner-selected `demo.aviasurveil.com`
  hostname. Exact IDs remain in the ignored mode-`0600` discovery record.
- The selected hostname already has one proxied CNAME to the healthy,
  remotely configured `aviasurveil-demo-local` Tunnel. That Tunnel has four
  healthy edge connections and routes the hostname to
  `http://127.0.0.1:8086`; it is the existing local demo, not the new
  private-pilot production Tunnel.
- No DNS or Tunnel mutation ran. The production Terraform module now permits
  creating its distinct Tunnel while managing zero application DNS records.
  The existing DNS record can be imported and moved only in a later exact
  cutover wave, after the production origin is healthy.
- Current Cloudflare guidance confirms that token secrets are shown once and
  should be restricted to the exact zone; see [Create API token](https://developers.cloudflare.com/fundamentals/api/get-started/create-token/).

## Next Gate

The AWS and Cloudflare discovery blockers are resolved. The owner approved
`Owner=platform-operations`, `CostCenter=aviasurveil360-private-pilot`,
`DataClassification=restricted`, and hostname `demo.aviasurveil.com`. The
remaining owner overlay inputs—including globally unique object names, SNS
alarm topic, SMTP, exact storage sizes, budget recipients, records/recovery,
immutable release subjects, and release authority—remain `blocked`.

The dependency-valid remote-state bootstrap provider plan was reviewed and its
exact apply created seven resources before stopping on the alias IAM mismatch.
The permission is now corrected and the residual alias-only plan is reviewed at
1 addition, 0 changes, and 0 destroys. Its exact apply completed and all eight
bootstrap resources are verified. The exact-authorized remote-backend migration
and native lock probe also completed. The next gate is separate authorization
of the next dependency-valid plan. Full runtime planning remains blocked on the
remaining owner overlay and upstream outputs.

## Discovery And Local Architecture-Revision Verification

- `jq empty .local/aviasurveil360/aws-private-pilot/discovery.json` —
  `verified locally`; the ignored evidence file remains mode `0600` and now
  preserves both the active Tunnel range and the superseded ALB/NAT range.
- AWS Access Analyzer validation of the attached remaining-services policy —
  `verified locally` through the authorized read API with zero findings.
- Explicit `avia` STS identity refresh and Cloudflare bearer verification —
  AWS identity and the replacement Cloudflare bearer `verified`; the exact
  zone, hostname, DNS record, Tunnel, configuration, and four healthy
  connections were read without mutation. No secret value was logged.
- Private-pilot decision, Compose, infrastructure, and release contracts —
  92/92 `verified locally`; the new bootstrap-plan scope and
  premature-DNS-cutover cases fail closed,
  and the mutation matrix rejects ALB, NAT, ordinary
  Internet Gateway, EIP, public IPv4, IPv4 default route, runtime ingress,
  connector-token state, mutable/external images, and remote authority.
- Focused AWS adapter Go race suites, API/worker packages, and the task-owned
  PostgreSQL/MinIO/ClamAV/Mailpit integration — `verified locally`; the local
  integration ended with zero task-owned Docker residue. ClamAV was exercised
  only as the retained local harness and is absent from production Compose and
  IaC.
- Terraform/Terragrunt formatting, shell syntax, committed JSON parsing, and
  Trivy HIGH/CRITICAL IaC scan — `verified locally`; Trivy reported zero
  findings using embedded checks without a check update.
- Focused Terraform remote-state bootstrap provider plan — `verified locally`
  as a reviewed protected 8-add/0-change/0-destroy bundle; native lock file,
  secret scan, JSON validation, and stored hashes pass. Its exact apply created
  seven resources and stopped `blocked` on the denied alias.
- Post-apply KMS/S3 reads — `verified`: key/rotation, private bucket policy,
  versioning, ownership, public-access blocks, and exact KMS encryption are
  active. The protected partial state hash and mode are `verified locally`.
- Residual alias provider plan — reviewed at 1 add/0 change/0 destroy;
  policy-v2 prefix, existing key target, protected pre-plan state, secret scan,
  JSON, and hashes are `verified locally`. Exact apply completed at 1/0/0; the
  alias target, eight-resource final state, and protected output hashes are
  verified.
- Remote backend migration and native S3 lock — `verified`: the exact
  eight-resource state is semantically preserved in one versioned KMS-encrypted
  object; the refresh-disabled plan is 0/0/0; two lock versions and matching
  delete markers prove lock acquisition/release while the state object version
  stayed unchanged.
- The separate `remote-state-managed` Terragrunt entry point is `verified
  locally`: it pins `avia`, account, region, exact state key/KMS key, native S3
  lockfile, and always-on remote initialization while every future
  plan/apply/destroy/import still fails closed without new authority.
- `node tests/harness-docs-smoke.test.js` — `verified locally`.
- `git diff --check` — `verified locally`.
- Full-stack provider-backed planning and every AWS/Cloudflare mutation outside
  the eight recorded bootstrap resources plus the exact state/lock object
  writes — `not run`.
