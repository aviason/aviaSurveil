# Local Production-Like Services Evidence

**Evidence date:** 25 July 2026
**Plan:** [Local Production-Like Services](../exec-plans/active/2026-07-22-local-production-like-services-plan.md)
**Scope:** local Docker Compose candidate only
**Status:** Tasks 1–9 are `verified locally` and the plan is
`ready-for-verification`; the artifact is `candidate-only` and
`release pending`. Nothing was deployed and this evidence does not establish
`production-ready`.

## Accepted Prerequisite Boundary

Plan 1 remains `ready-for-verification` with its literal one-shot visual result
unchanged at 71/259. Comparison-by-comparison review is `not run` and
standalone baseline integrity is `not verified`. No failed comparison was
converted into a pass and no accepted baseline was replaced.

Plan 2 is independently accepted for this prerequisite and remains
`ready-for-verification`, `candidate-only`, and `release pending`. The final
Plan 3 regression selection passed generated contract 16/16, canonical
examples 15/15, SQLC drift, complete Go `-race`, web 626/626 across 60 files,
TypeScript typecheck, and root/oracle 108/108.

## Runtime Scope

The local candidate has one browser-facing Caddy HTTPS origin and separate
demo and HTTP React artifacts. Full mode runs the Go API, worker, scheduler,
and one-shot migration process; separate application and Keycloak PostgreSQL
databases; production-mode Keycloak; private versioned MinIO; real ClamAV;
authenticated Mailpit SMTP; and Gotenberg PDF rendering.

Only Caddy publishes a browser-facing port. Full mode contains no mock/seed
input, deterministic scanner, canonical-header authentication, fixture
initializer, or registered `/__test/*` handler. Runtime secrets are generated
under ignored task-owned state and mounted as Docker secrets.

## Required Verification Matrix

| Gate | Literal result |
|---|---|
| Local Compose/image/runtime contracts | 37/37 passed |
| Compose policy | 21/21 passed |
| Runtime image build | 8/8 built |
| CycloneDX SBOM generation | 8/8 generated |
| HIGH/CRITICAL vulnerability gate | 8/8 passed |
| Clean demo profile, run 1 | 86/86 direct loads; Playwright 1/1; skipped 0; 1.2 minutes |
| Clean full profile, run 1 | 86/86 HTTP direct loads; 10/10 scenario families; Playwright 1/1; skipped 0; 31.1 seconds |
| Clean demo profile, run 2 | 86/86 direct loads; Playwright 1/1; skipped 0; 1.2 minutes |
| Clean full profile, run 2 | 86/86 HTTP direct loads; 10/10 scenario families; Playwright 1/1; skipped 0; 32.7 seconds |
| `npm audit` | 0 vulnerabilities |
| `npm audit --omit=dev` | 0 vulnerabilities |
| Complete Go race suite | all packages passed; external OIDC integration skipped only when its four provider variables were entirely absent |
| Profile contract | 11/11 passed |
| OpenAPI generation/drift | 16/16 passed |
| Canonical OpenAPI examples | 15/15 passed |
| SQLC drift | passed |
| Web unit/component suite | 626/626 across 60 files |
| Root/oracle regression | 108/108 passed |

The final clean projects were:

- `aviasurveil360-task-plan3-demo-20260724232442-15829`
- `aviasurveil360-task-plan3-full-20260724232609-15997`
- `aviasurveil360-task-plan3-demo-20260724233044-17198`
- `aviasurveil360-task-plan3-full-20260724233209-17372`

Each finished with exactly zero task-owned containers, volumes, and networks.
The scripts also removed their scoped browser, Playwright, credential, report,
and state directories.

## Image, SBOM, And Scan Evidence

The accepted manifest is `.local/aviasurveil360/image-evidence.json`. It
records source revision
`f33d2bccb95145bbc3a16171937b961295523806`, dirty input `true`, and the
build-time source-state digest
`acc83e3995ee89b0792fd6392ad37356825ab24dee1bbf3647cd94ef3b35ab4d`.
The source-state digest was captured before this evidence document was
synchronized.

| Image | Runtime digest | SBOM SHA-256 | Scan SHA-256 |
|---|---|---|---|
| gateway | `sha256:8fc2045ddba4c9550c0966b647269b1fd57999b4ec97efe112aedfdee53be29d` | `92d683d0a6230b4417f15826535a568dde9fbfe820e5f1454440f39d146fba0b` | `8ed9e428270a49fe94df5e139d6e5d48b78e9c26add4f788ab85eda3ce3de796` |
| web-demo | `sha256:09e5eb633687d4461e157cd530c80484289ea5e82f1db1fc8bfd47dbbd999895` | `1657cb9039a205867d3a9cfc7edff98c3be69b872b83f415483311f5a90ce558` | `baadd8ef3333e93cd3e4ec4f806efdf66e6f470e2d313a1568b146df75cfdb0b` |
| web-http | `sha256:7a4bec262e99824eea3778d74f1789ba13639b2294a2434c4883e3cd3cc9100f` | `3e30035ce28af8264dbbd1a9bcdb0027fc2240ce5f58fdf9f2ff332518754435` | `76ff1e4c673ad0bb6014d168f1627498f8d92c9d0ef57c3a46c1496cf45fbd28` |
| keycloak | `sha256:65fdad498ac3205f69aea3333d3384fd1c95dc4592cbce6ef8f65af2ef6071a5` | `6194212c09603e3fecbab1de07a684c1e975cf6bae862f655fb7d0b92665cab6` | `a8eb74fc810a6e8abdfc4389df25ad1a096c0adf4874aeba589506f567b55272` |
| api | `sha256:cbb3f82d6665332e6fe159de63da797b4d237dad09a9e1571834b511c07c7db0` | `13f1db2e867f36c2d326cec63cdcc652a1b2675cf4f7ca30cf8880bf7f8ee950` | `5d9aae71b49e7b2d9ae978d93bfaffca120997558f4eb0e5887bc02ab9514183` |
| worker | `sha256:9fb3f06ae6357d21dc083ce6751a84a034fb2fb9ed7fc13c57532f981262aee5` | `4b4b674149f8f89e271d67dff3b7e57fd29ef72dee3ff1cef187b75cf8e4a8d9` | `239df893b568099173e16dd45cd5e3ad6f15a535f5fa65ef0f7c4eec97fbd0b0` |
| scheduler | `sha256:a98fdcda98c262da12f38884f2023eb37106e482f44bcc7500277c5b03663168` | `3adb31e7244834dbd698463b00000615f0d857ec47d69e5f8ca5389941a21b0a` | `4ed908615e126a76a578b619e450e3e6a14702f948167f58345221cc3a3fa13a` |
| migration | `sha256:6417052c35d486cf9f69a5d8431470b13947c36b320aeb6cf7205c1cbfa716b9` | `04890e7abe85fc9927c233346f0da52e418fc327558875cc5d1aacd2ff9d5535` | `bb65451479498dbed1cf059996648b093b663c7fafa5d922bf59ef88e2b35bba` |

The clean-profile scripts now fail closed unless the current image tags match
these exact SBOM- and scan-bound runtime digests. The RED reproduced a gateway
digest drift after an unscanned Compose rebuild; the GREEN removes profile-time
rebuilds and checks the accepted image manifest before startup.

All fixable HIGH/CRITICAL Keycloak findings were remediated. One exact source
digest exception remains for `CVE-2026-22020`, owned by Local Platform
Security and expiring 8 August 2026. It is tracked at
`td-plan3-keycloak-java21-trivy-mismatch` and does not float to another source
digest or vulnerability.

## Configuration Hashes

| Configuration | SHA-256 |
|---|---|
| `deploy/local/compose.yaml` | `0c29ad0a96a3dccdb58373a1a6749313d1c77c12f831390cf52f288942799233` |
| `deploy/local/image-lock.json` | `219715b47b1aecc3aa831bf64d3a2218371a812fa9f0e6e3fcb72029a58b3cad` |
| `deploy/local/compose-policy.json` | `6ba43768b44226114f26c971af1fc0a7d6a7d9400e5d6b8783850f0fe19adf70` |
| Keycloak realm source | `07f5491a52880f5423da8974b7af5b308b3bc6a89a9728df0accbf8e39c01ed6` |
| Keycloak runtime patches | `b932daf6e0118b4ceb47b45c9bfcaa33fd5212b2de180ba42a9667f8af947ad9` |
| Vulnerability policy | `2ce379a3e7f00b3687e569e2690e97553746ad90619e62d35624e0f8e716408f` |
| Report template v1 | `d0ac76576b3c1608fa7d4215f8725d87006511705391a7131116fb44b3a99f14` |
| Auditee HTML/text notification templates | `2223c315983fd5c59b70de4f9ee7843d5dcd7be5ee4357059bc7ab3dc93eada7` / `f328882b2771b8aced07fb7a225ad5b595624c2d328e160319ee6433f1734885` |
| Internal CAA HTML/text notification templates | `94ed9752b52a3f37b64d65fbd7b55250921309477a027d3dc38cce39aabbd015` / `3f8a9cf6b6ebb18dd019bb601df3f741d8ee31ca2a949aa9fa28177bf71ea1bf` |

## Real-Service Proofs

- Keycloak 26.7.0 ran in production mode behind the one HTTPS origin.
  Administrator and provisioned Auditee first-login flows each completed
  `CONFIGURE_TOTP`; role, organization, PKCE, CSRF, disable, session revocation,
  restart, and no-test-route boundaries passed.
- Private versioned MinIO stored the 71-byte Evidence object with
  `sha256:dc6c95daec00e242b194297223ebfd6a717320563a50a16bc718542a33e35d10`.
  Unsigned access remained denied and overwrite prevention remained atomic.
- The last separately captured live ClamAV provenance was engine 1.4.3,
  signature version 28070. Final clean profiles required fresh signature
  readiness, persisted engine/signature/scan-time metadata, and allowed
  closure only for the exact scan-clean Evidence version.
- Each final full run inspected the private Mailpit API and found exactly one
  application delivery containing `Plan 3 full-profile SMTP delivery`.
  Authenticated SMTP, stable message identity, retry/restart, organization
  scope, and Internal CAA separation remained green.
- The first final scanned-image run persisted a private immutable Gotenberg PDF
  with content
  `sha256:e16b5e72295a2cfd6e4ffe8337b52f411ea4be605d188874ee0046c52db7be06`,
  renderer
  `sha256:56c47f7b913f3b978554115a0191c4a9dcc2558f9090f27f3f13f28a7c2f8329`,
  template
  `sha256:d0ac76576b3c1608fa7d4215f8725d87006511705391a7131116fb44b3a99f14`,
  and source
  `sha256:52e6a51e0ac942495920f191090eba7c9384848a36ad5f51544f584d8d991d0f`.
  The exact PDF bytes downloaded with `%PDF-`; rendering did not approve,
  sign, close, or confer legal validity.

## Failure, Resource, And Cleanup Boundary

The runtime matrix passed required PostgreSQL, Keycloak, MinIO, and ClamAV
loss/recovery; optional Gotenberg and Mailpit degraded loss/recovery; scanner
unavailable and 250 ms timeout; Gotenberg unavailable and 250 ms timeout;
authenticated SMTP retry/restart; private-object overwrite races; and worker
crash/restart recovery.

Runtime services declare a 15-second stop grace period, two CPU/two GiB limits,
and a 256-process limit where applicable. Readiness probes seven named
dependencies concurrently; liveness remains downstream-independent. Only
Caddy published a port, network membership matched policy, orphan containers
were zero, generated-secret log matches were zero, and all final task-owned
residue counts were zero. Peak CPU/memory utilization was `not run`; the
declared resource limits and bounded shutdown behavior were `verified locally`.

## Review And Release Boundary

Task-level and final spec-compliance and code-quality checks are main-agent
self-reviews, not independent reviews. The final spec-compliance review found
no remaining Critical or Important requirement gap. The separate final
code-quality review found no remaining Critical or Important implementation
issue; changed Go files are formatted, complete `go vet` passes, shell syntax
passes, profile/image security contracts pass 24/24, and docs/diff checks pass.

No Git staging, commit, push, deployment, production infrastructure change, or
operating-system trust-store change was performed at this Plan 3 checkpoint.
Plan 4 was unstarted at that checkpoint; its later local status is recorded in
[Local Reliability, DR, And Infrastructure Evidence](LOCAL_RELIABILITY_AND_DR_2026-07-22.md).
