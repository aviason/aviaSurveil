# AviaSurveil360 Architecture Map

This file maps the repository's current runtime and knowledge boundaries. It is
an orientation surface, not a production-readiness claim.

## Runtime Surfaces

| Surface | Ownership | Boundary |
|---|---|---|
| Root `index.html`, `css/`, `js/` | Accepted static demo and legacy behavior/UI oracle | Browser-local mock behavior; preserve unless an explicit task changes the oracle. |
| `apps/web/` | React + TypeScript + Vite candidate | Separate demo and HTTP build profiles; HTTP artifacts must exclude mock/seed inputs. |
| `apps/api/` | Go API/worker candidate | Local PostgreSQL, session/OIDC, upload/scan, audit, domain slices, immutable report/document versions, native provenance-bound PDF rendering, and a worker-owned reminder controller. `AVIA_DATA=0` remains the standalone default and rejects feed configuration; Workspace Namibia dev explicitly selects `AVIA_DATA_MODE=local-candidate` and the internal AviaData admission client/worker loop. The surface remains `candidate-only`; live connected-data E2E, coordinated recovery, release, and production readiness are not established. |
| `../../shared/auth/` | Shared AviaAuth durable OIDC runtime | Public OIDC/UI listens on 8080; private provider administration listens separately on 8081 with a mounted bearer secret. Auth PostgreSQL owns credentials, MFA, recovery, signing keys, provider sessions, opaque claim projections, idempotency receipts, and redacted audit state. The gateway routes only public `/identity/*` requests and never routes private administration. |
| `apps/api/cmd/preprod-canonical-demo-identity-loader/` | Disposable canonical identity bootstrap | Provisions nine fresh synthetic `usr_...` subjects through private first-party administration, activates them through the SMTP-free local-preprod admin path, and reconciles application identity, membership, lifecycle receipt, revision, and assignment state. Replay is resumable and drift fails closed. |
| `api/openapi/` | Transport contract | Source for checked request/response shapes and generated adapters. |
| `deploy/local/` | Local verification services | Candidate-validation profiles only; AviaWorkspace owns platform deployment authority. The collector-backed full profile keeps browser OTLP enabled and routes `/otel/*` to the private collector; canonical local-preprod and Quick Tunnel builds provision no collector and explicitly compile browser telemetry off. |
| `scripts/start-canonical-preprod.sh`, `scripts/status-canonical-preprod.sh` | Canonical AGA disposable local-preprod operator boundary | Starts and reports the exact `aga-preprod@1.0.0` namespace. The runtime metadata retains a fail-closed `disabled` donor sentinel, while the duplicate donor implementation itself was physically removed under Task 9. Connected OIDC hero receipts are local candidate evidence only. External preprod is outside the local plan and remains `not run` in its separate paused ExecPlan. |
| `scripts/test-canonical-preprod-fault-restart.sh`, `make preprod-test-fault-restart` | Task 8 destructive local qualification boundary | Creates unique disposable integration and local-HTTPS projects, verifies canonical transaction/concurrency guards, cold-restart authoritative-state identity, post-restart role sessions, required/optional dependency semantics, worker recovery, donor/log denial, and exact cleanup. It never targets the named demo project or external infrastructure. |
| `deploy/local/compose.local-http.yaml`, the default `quick` mode of `scripts/start-canonical-preprod-cloudflare.sh`, and `scripts/canonical-preprod-{cloudflare-launcher,quick-tunnel-url}.mjs` | Optional disposable Quick Tunnel qualification boundary | Keeps the canonical HTTPS/8445 profile unchanged while a separate loopback HTTP gateway accepts the random host. It uses only an anonymous, sanitized `cloudflared tunnel --url http://127.0.0.1:<port>` process, validates public-origin OIDC/Secure-cookie/object/CORS wiring and the nine privacy-safe role panels in isolated Chromium, and removes verified task-owned state. Quick mode never configures named tunnels, Cloudflare account/DNS/Access resources, AWS, or external preprod. |
| `make preprod-cloudflare-demo-*`, `scripts/store-canonical-preprod-cloudflare-token.sh`, `scripts/store-cloudflare-tunnel-token-keychain.swift`, `scripts/validate-cloudflare-tunnel-token.mjs`, `scripts/canonical-preprod-cloudflare-named-launcher.mjs`, and the explicit `named` lifecycle mode | Optional `demo.aviasurveil.com` local-origin publication boundary | Uses a separately created remotely managed Tunnel/public-hostname route and a tunnel-scoped connector token stored only in macOS Keychain. The hidden terminal reader and Security-framework writer carry the complete value through memory-only pipes without the `security -w` interactive prompt's 128-byte ceiling. Before building, the lifecycle validates the encoded connector payload without printing or persisting it. The launcher passes the in-memory value through `/dev/fd/3`, while runtime metadata stores only the Keychain reference and exact process identity. A separate loopback port, Compose project, and disposable state bind public HTTPS/OIDC/Secure-cookie/object/CORS behavior without exposing the origin directly. The repository does not create or mutate Cloudflare account, tunnel, DNS, or Access resources; this public candidate remains local-origin and never counts as the separate paused external-preprod release. |
| `tests/` and `apps/web/tests/` | Contract, parity, browser, and regression evidence | Results are local evidence unless an external gate is explicitly exercised. |

## Knowledge Surfaces

- [Product specifications](docs/product-specs/index.md) own user behavior,
  vocabulary, roles, and workflow semantics.
- [Execution plans](docs/exec-plans/index.md) own sequencing, progress, and
  acceptance evidence for complex work.
- [Agent harness](docs/agent-harness/index.md) owns repository navigation,
  command discovery, verification routing, output labels, and entropy control.
- [Build evidence](docs/demo-evidence/BUILD_SUMMARY.md) records historical local
  outcomes and limitations; it does not override code, contracts, or plans.
- Governed checklist inventory and browser proof are fail-closed local evidence:
  every named schema, migration, script, package, test, and browser spec must
  exist with a nonzero discovered test count before aggregate acceptance.

## Dependency Direction

1. Product rules and accepted source identities constrain contracts.
2. The OpenAPI contract constrains generated transport types and adapters.
3. Runtime capability interfaces separate React features from mock and HTTP
   implementations.
4. Demo-only state must not leak into HTTP builds.
5. Role and organization projections must filter forbidden data structurally,
   before rendering.
6. Canonical identity bootstrap enters through the private first-party admin
   boundary; provider-generated subjects are persisted separately from fixture
   usernames and exact application membership identity.
7. Tests and visual fixtures prove boundaries; they do not authorize rewriting
   identity or authority to match stale oracle content.
8. The AviaCore v3 producer outbox validates the local locked contract before
  storing an AES-GCM payload envelope, keeps event/attempt history immutable,
  and never reconstructs facts by table scraping. Canonical materialization
  emits `audit.planned` from the released planning identity and the
  server-owned inspection revision, creates a non-executable `NOT_STARTED`
  checklist, and a later atomic Inspector-start transition emits
  `audit.started`. The legacy donor `CreateAuditWorkspace` path and duplicate
  AGA stakeholder runtime were physically removed after donor-free connected
  qualification; every other transition remains an explicit non-event until
  separately mapped.

9. AviaWorkspace owns customer topology, environment selection, immutable image
   release locks, and deployment infrastructure. AviaSurveil owns application
   behavior and candidate-local validation only.

## High-Risk Invariants

- Exact route, role, organization, record, version, revision, and audit identity.
- Application sessions require the current desired-membership revision, fresh
  exact provider authority, and matching OIDC role/organization claims; a
  lifecycle mutation revokes old authority and requires a fresh login.
- Interactive logout atomically revokes the application session, removes its
  encrypted provider tokens, redirects through the discovery-bound OIDC
  end-session endpoint, and forces credential entry on the next authorization
  request; transport choice must not preserve or silently reuse the prior SSO
  identity.
- Append-only Evidence, configuration versions, and audit history.
- Auditee organization isolation and Internal CAA Note exclusion.
- Explicit review and closure authority.
- Build-profile separation and absence of mock/seed inputs in HTTP artifacts.
- Exact disposable-target binding, one-time loader authorization, provider
  subject retention, whole-namespace cleanup, and zero task-owned residue.
- URL-encoded server-owned identifiers are decoded at canonical HTTP route
  boundaries before authority, projection, and append-only transition checks.
- Root-oracle and accepted-baseline integrity.
- A real source-bound OPS/AOC request remains blocked with zero lifecycle
  effects; only the explicit synthetic internal test profile can prove a
  positive local checklist lifecycle.

Use the [verification matrix](docs/agent-harness/verification-matrix.md) for
the executable checks that protect these boundaries.
