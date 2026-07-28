# Local Reliability, DR, And Infrastructure Evidence

**Evidence date:** 27 July 2026
**Plan:** [Reliability, DR, And AWS Terraform/Terragrunt](../exec-plans/completed/2026-07-22-reliability-dr-and-aws-terraform-terragrunt-plan.md)
**Scope:** Plan 4 Tasks 1-9 and Task 11, local candidate only
**Historical checkpoint status:** `verified locally`, `candidate-only`, `release pending`, and
`ready-for-verification`

No AWS account was queried or changed. AWS discovery, real planning, apply,
artifact publication, smoke, rollback, retain/destroy, and every other Task 10
action are literally `not run`. This evidence does not establish
`production-ready`.

## Stakeholder Closeout — 28 July 2026

The user accepted Plan 4 as completed for the local `candidate-only` milestone
through the
[combined Plans 2–4 stakeholder disposition](stakeholder/PLANS2_4_STAKEHOLDER_DISPOSITION_2026-07-28.md).
Tasks 1–9 and Task 11 plus the historical command, alert, backup, restore,
runbook, image, Terraform, Terragrunt, and cleanup results below remain the
canonical technical basis.

The backup store remains same-host/logically isolated and does not prove
host-loss DR. Local measurements do not establish production SLO, RPO/RTO,
alert recipients, or staffed on-call. Production retention, legal hold,
deletion, encryption/KMS ownership, restoration authority, provider selection,
identity federation, external email, data residency, release, rollback, and
operating decisions remain deferred.

Task 10 is optional, unauthorized, and `not run`. AWS discovery, planning,
apply, artifact publication, smoke, rollback, retain/destroy, and every other
Task 10 action remain `not run`. Release remains `release pending`; deployment
and production readiness remain `not run`; no `production-ready` claim is made.

## Accepted Sequencing Boundary

Plans 1-3 remain `ready-for-verification`. On 26 July 2026 the user explicitly
deferred their stakeholder review until the end and accepted the local rework
risk for Plan 4 Tasks 1-9 and Task 11. That sequencing decision did not
authorize AWS Task 10, deployment, production changes, or release.

Plan 1's retained decoded-pixel result remains literal at 89/259 with 170/258
pixel failures. No baseline, mask, threshold, authority rule, privacy rule, or
semantic truth was weakened while completing Plan 4.

## Required Local Verification Matrix

| Gate | Fresh literal result |
|---|---|
| Operations, observability, backup, recovery, and runbook Node contracts | 43/43 passed |
| Go `-race`, one package at a time | All packages passed inside the canonical PostgreSQL/MinIO/Keycloak profile |
| React typecheck | Passed |
| React unit/component suite | 644/644 across 64 files |
| Clean local full profile | Playwright 1/1; 86/86 HTTP route loads; all 10 real-service scenario families; skipped 0 |
| Runtime failure/restart and cleanup | Real Mailpit delivery and worker restart passed; zero task-owned residue |
| CycloneDX image SBOMs | 9/9 generated and bound to runtime digests |
| Image HIGH/CRITICAL scan gate | 9/9 passed |
| Local observability profile | Collector, Prometheus, Grafana, Loki, Tempo, Alertmanager, all 8 alerts, grouping, recovery, restart persistence, and cleanup passed |
| Backup catalog profile | Full and incremental application/identity pgBackRest chains plus exact object versions passed |
| Isolated RPO/RTO drill | Two complete restores, corrupt-latest fallback, restored MFA/roles, 86 routes, real worker delivery, API restart, and zero residue passed |
| Runbook contracts | Included in the 43/43 Node result; ten owner-scoped runbooks are present |
| Terraform format | Passed |
| Native Terraform tests | 12/12 passed |
| TFLint | Passed with no findings |
| Trivy IaC HIGH/CRITICAL gate | 27 configuration files scanned; 0 findings |
| Terragrunt HCL format | Passed |
| Terragrunt fixture graph | 12/12 validates and 12/12 protected fixture plans; 12/12 empty OPA denial sets; zero apply/destroy |
| Terragrunt and AWS plan/command contracts | 46/46 passed |

The direct sandbox Go command could not bind the required local test port. The
same exact `go -C apps/api test -race -p 1 -count=1 ./...` command passed in the
repository's canonical service harness with PostgreSQL, MinIO, and Keycloak.

The intentionally fail-closed bare `./scripts/check-terragrunt.sh` invocation
returned `missing-owner-input: AVIA_TG_INPUTS_FILE`. The accepted local fixture
run supplied
`infra/terragrunt/fixtures/non-deployable.hcl` and a protected task-owned plan
directory under `/private/tmp`; it validated and planned all 12 units without
AWS access, apply, or destroy.

An earlier combined HTTP harness stopped after its Go, OpenAPI, SQLC, 644/644
web, build, and HTTP contract gates because one Lead mobile inventory region
was stale. The profile-aware ledger was corrected without changing behavior.
Fresh focused mock inventory then passed all three viewports; separate HTTP
desktop, tablet, and mobile inventories passed, and the full HTTP action
execution passed with its outbox assertion.

## Observability And Alert Evidence

The fixed cross-signal fixture used:

- trace ID `00112233445566778899aabbccddeeff`;
- span ID `0011223344556677`;
- correlation ID `plan4-correlation-001`; and
- service instance `plan4-observability-fixture`.

The same correlation value was found in Tempo and Loki while the related
metric was found in Prometheus. Collector output contained none of the injected
forbidden authorization, password, or message-body values. The profile proved
all eight catalog alerts reached Alertmanager, duplicate fixtures grouped into
one Mailpit delivery, `send_resolved` produced the recovery delivery, and
Prometheus, Tempo, and Loki retained evidence across restart. Only the Caddy
gateway published a port, Grafana authentication remained enabled, and cleanup
left zero task-owned containers, volumes, and networks.

| Configuration | SHA-256 |
|---|---|
| Prometheus rules | `b8a7fb0381db4b613fe944ec7caa596831de11b93e572821ba16a8b565c78e14` |
| Grafana overview dashboard | `a64072c6ad745593e247a703d98eb1c78ec1b2878e026d80cacfa33d7d56065f` |
| OpenTelemetry Collector | `a21a73c39a1f192432d7479b958d39a87028569790da1962fe0ad0a297ba41f2` |
| Alertmanager | `06ca783dfe632776a62843c8ad16949a4159b81ebaadce830a4d4f2f74d7ab97` |

These are local engineering signals and local Mailpit receipts. They do not
prove production paging, staffed on-call, external recipients, or contractual
SLOs.

## Backup And Recovery Evidence

The backup catalog profile created two complete points around controlled
database and object changes:

| Recovery point | Type | Catalog SHA-256 |
|---|---|---|
| `rp-20260727T015521Z-full` | full | `a0293d8ebf7bdd9340f835baae45df2bd7c1e79b7ce61e7f11deef498f2f8a1a` |
| `rp-20260727T015531Z-incr` | incremental | `b5a6477fc50b252d0658adb293042a18de9d31a4b7206c5fd1e970f167b05ea5` |

Both catalogs bound separate encrypted application and identity PostgreSQL
chains, exact object version IDs, ETags, byte counts, metadata, retention,
configuration references, and deterministic fingerprints. The failure domain
was exactly `same-host-logically-isolated`.

The final evidence drill created and restored:

| Recovery point | Catalog SHA-256 | DB RPO | Object RPO | RTO |
|---|---|---:|---:|---:|
| `rp-20260727T021314Z-drill2` | `8d617fccbb9d8470539300eff6d9da695efd2293e957abf3bc3bd373e0d7d3bf` | 3 s | 1 s | 81 s |
| `rp-20260727T021304Z-drill1` | `4bcbdca2a508635bcec398959ed118512af3ef86bc3b735e28791abb9b7940fe` | 107 s | 102 s | 81 s |

The newer catalog was also corrupted deliberately and refused before target
mutation. The preceding complete recovery point then restored successfully.
Both results remain below the candidate RPO target of 900 seconds and candidate
RTO target of 3600 seconds.

| Recovery point | Fingerprint | Expected and restored SHA-256 |
|---|---|---|
| `rp-20260727T021314Z-drill2` | application | `5ac648a4a406cdf0deeeddb7da42e6eb7b3b3af19ac8c6ed1e46e5c19f5c0706` |
| `rp-20260727T021314Z-drill2` | identity | `5f692d58d9b15504956ed648d801fe25f205d7d1f30e3177b51ac0ca028b1781` |
| `rp-20260727T021314Z-drill2` | objects | `1db9fc1b6a52d8a4e581252ec1077fba63ab96433e9bfc53e2ba12b6386f127f` |
| `rp-20260727T021304Z-drill1` | application | `e4011e8c5be7fcbcd0240b9b10e9a41069e2dafc45cd8c2962cab80b226f92b0` |
| `rp-20260727T021304Z-drill1` | identity | `5f692d58d9b15504956ed648d801fe25f205d7d1f30e3177b51ac0ca028b1781` |
| `rp-20260727T021304Z-drill1` | objects | `b01f12fde8cd00feec75916d78d58baffbc3bac87f1f7078e76e2b91da712bff` |

Each restore recovered normal OIDC/TOTP login, organization `CAA`, the exact
`admin,executiveDirector,finance,gm,inspector,leadInspector,manager` role scope,
all 86 React routes, one pending notification through the real worker and
Mailpit path, and the API-node restart scenario. Every isolated target and its
browser output was removed after evidence capture.

The drill scenario catalog SHA-256 is
`525e9687010a2475e01b0a0319feb6d022b2c7c8061fb1a26ce3257840d55b5a`.
These measurements are candidate local results, not production RPO/RTO
commitments and not host-loss evidence.

## Image, SBOM, And Scan Evidence

The accepted ignored manifest `.local/aviasurveil360/image-evidence.json`
records source revision `3acedc80c7c4545e715fa694ea1e2fbc6163447d`,
dirty input `true`, and source-state SHA-256
`465a4b4a82407c98229c507280f61a69329c0d4c4a5f032ff186c61e51d044f3`.
The source-state digest was captured before this evidence document was added.

| Image | Runtime digest | SBOM SHA-256 | Scan SHA-256 |
|---|---|---|---|
| gateway | `sha256:c3f2209290c4c1072de4e126ea66edd308463dba26d0c9965bac78a7270e78e4` | `47d79bd02daca88e96678e12196f3e9ca331f8f1c589a5b47badbdb30a0bb382` | `9e1ec17160ed3acc7877b862635f18fc08bafc78737d2de824676900be82805c` |
| web-demo | `sha256:eaa6ab1d45dbd6c1ad0db93e1056da60a4c77f85e218a931ddd91196e3d19a1c` | `6e780bf4dc856d8b9e55749de0f972ede8476e73ebb102e38cc42ca91bc72060` | `4349eb189cb796da3363e31d4d5dec67053b09bde2cb2a18addcecff47a82efe` |
| web-http | `sha256:b9d2e35d22f8bd598d24c079ae3c0d4a26ed58cf8ca8fd49ca8e4a9676f5ae5c` | `cbf5c6088028184f65eecb9c97b2f4a7e9c6c67f51f9324b5863755262ebe5a6` | `e888fee03e843b9da823b8c51a175572f7ebff6ebe5228e18c79c4174e8e173a` |
| keycloak | `sha256:468787c0a642789dfd6efc2b116f3d8bd972abad30249eca9add41cf12c640e4` | `948f8f26fc47fdd22d541138b5c40d2cda1f07ca2f7caa4724ec45e28f6a6366` | `7f1a4c4f6bc7cc0e70543ac4caf9419bce8d40c681b7ad588555106ee07e84f3` |
| postgres-recovery | `sha256:bb67eeaa76314487678c0c0c221b427450db1f8c9e12d3c058ecb71593b4fe27` | `13b70da4a2f9023c3731f5b1e17a47b10b24551d6e632002ec0bda0aa4ab826e` | `9f309c925c7f90e509d969c33482c9a769923f412135dfa46da237f91cb3b77a` |
| api | `sha256:9233c1b871bf2824de22bf34f149aeb6ab701579ebb69e47b0146d2f47a34c44` | `a508568f71089604b9bea4309ff1585fc0bd5736f54431b206a630605a4af4d6` | `548bfbacc47c73c65abdfc64e715021d6cc840e923a2e0e781b83ec534483fc0` |
| worker | `sha256:7466a636ba191c66fe55a3d811538843dea5ec6847bd0e727cf8a98e4d3fc169` | `bc58fceeaa5d9864c842c4112f213081ce774cdaf517983b42ad02c3a2b8c777` | `4a66634ef13e1da9e2de72e96051af426bf289e0a824c4abf48db95ca59caf87` |
| scheduler | `sha256:8f02e902ff51d54f14bdbd3fba9386ae84556b54609af568e519954d0b954703` | `b2f269e116bc993f4dca91ce6b232ceb83715b3d161d2c50bf76247ffb7b8884` | `85b3298b0e8aae16f960c91e06b22650a69bca9824bcb502f8971de7c89cb81f` |
| migration | `sha256:d6b43e6df12bdcaa4294d49d79321f5a02eaa68c0262a589af315682e27c186f` | `b31dc35da2d9c9a7771385ebe1bcedd7acba2336d73e9e2aab1c109c71af4a5a` | `7ddd5dca3a6e7bcbd86f8d999fcae5c04e3693c4c962c89e984500ebd0e7bf85` |

Every scan status is `passed` for HIGH/CRITICAL severity. The committed image
lock SHA-256 is
`65796e22c1f98ee534e91a848c9d507f52b954714a2cd4b438d877f7fb368608`.

## Terraform And Terragrunt Evidence

The local toolchain was Terraform 1.15.8, Terragrunt 1.0.4, TFLint 0.64.0,
Trivy 0.70.0, Go 1.26.4, Node.js 24.16.0, and npm 11.13.0.

Native Terraform tests passed 12/12. TFLint was clean. Trivy scanned 27 IaC
configuration files at HIGH/CRITICAL severity with zero findings. The
non-deployable Terragrunt fixture validated and planned all 12 units, produced
12 protected binary plans and 12 empty OPA denial sets, and executed no apply
or destroy operation. The 46/46 plan/command contracts include ten unsafe
Terraform mutations, protected-plan permission/hash/age checks, owner decision
checks, image/SBOM/scan binding, phase controls, no broad destroy, and
non-executing AWS wrapper previews.

| Artifact or lock | SHA-256 |
|---|---|
| Go module sum | `c19dcef31d50284bebe2ff8882d44899c2f68941e8b4f1af2b15978d53aa6ef8` |
| Web package lock | `33d24f34664c27f864a49e334f5ce28100a94091054baaea049a938eea54801f` |
| Common Terraform/AWS provider lock | `c6b44bc69255646a525a3fe7f7f177e32910ced268127d4201ec664519c0b3aa` |
| Artifact-publication local-provider lock | `c238d583dbe6d1395d52a3b9894930c8340f3b998652c549485679cc03839fb2` |
| Terragrunt component catalog | `3ba4ef8126e358b22c55dd925b7dcde1f600dfa09fed0fbc42716c04c0e136b6` |
| AWS plan OPA policy | `5821670cd5fc84927a04cee3e8bf682e9d801a05e7862b455ecbee1947e9a954` |
| Full-profile test wrapper | `188b0d043a8e017d0d71cfe579ad35ad9c6a7702fd81dd06cccf4bfa4c6b3997` |
| Observability test wrapper | `4165eecc322a41493d5b0eff707168165844d27102c7a3bd5b3211712458fe2f` |
| Backup catalog wrapper | `698ebb57e98e6996bb7f799cc72e337e579abddac4e5b1fbaad33a02c5faa444` |
| RPO/RTO drill wrapper | `871de91b6934a17ddd243aa3ccbe7a377e5e2ff0699e308fd60af472beee1cda` |
| Terragrunt checker | `f5994d3ce45ca6330247ea13593b171f83ae822e764ffe800aff13d26a91d034` |

The common provider-lock hash covers the Terraform root, bootstrap and secure
examples, remote-state bootstrap, and every AWS-trial Terragrunt component
except artifact publication. Artifact publication has a second lock because it
also uses the local provider.

## Ownership And Open Decisions

| Decision or residual risk | Owner | Status |
|---|---|---|
| Production SLO, RPO/RTO, alert recipients, on-call staffing, and escalation | Product, Platform/Operations, Security | `not run` |
| Independent backup failure domain and host-loss recovery | Platform/Operations, Security, Records/Legal | `not run` |
| Production backup retention, legal hold, deletion, encryption/KMS ownership, and restoration authority | Records/Legal, Security, Platform/Operations | `not run` |
| AWS account, region/data residency, domain/certificate, budget, capacity, change window, and retain/destroy choice | Named business, security, finance, and platform owners | `not run` |
| Production identity federation, email provider, secrets, trusted artifacts, staging, release, rollback, and legacy cutover | Identity, Platform/Operations, QA, Product | `not run` |
| Plans 1-4 local stakeholder acceptance | Stakeholders named in the plan handoffs | Accepted 2026-07-28 for local `candidate-only` milestones only |

The same-host backup store is only logically isolated, local Mailpit is not
external delivery, fixture plans are not cloud plans, and local runbooks do not
establish staffed production operations.

## Historical Checkpoint Disposition

At the 27 July 2026 checkpoint, Plan 4 Tasks 1-9 and Task 11 were `verified
locally`. The local reliability, recovery, runbook, Terraform, Terragrunt,
image, and policy milestone was `ready-for-verification`, `candidate-only`,
and `release pending`.

AWS Task 10 is literally `not run` and remains optional. Any future AWS phase
still requires a new exact authorization bound to account, region, owner
decisions, reviewed plan hash, wrapper hashes, immutable image/SBOM/scan
evidence, budget, window, and retain/destroy action. Production deployment and
the `production-ready` claim remain blocked.
