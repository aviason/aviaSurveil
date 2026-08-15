# AviaSurveil360 Architecture Map

This file maps the repository's current runtime and knowledge boundaries. It is
an orientation surface, not a production-readiness claim.

## Runtime Surfaces

| Surface | Ownership | Boundary |
|---|---|---|
| Root `index.html`, `css/`, `js/` | Accepted static demo and legacy behavior/UI oracle | Browser-local mock behavior; preserve unless an explicit task changes the oracle. |
| `apps/web/` | Connected React + TypeScript + Vite product | Demo and HTTP builds share server-owned route and identity contracts; HTTP artifacts exclude mock/seed inputs and browser storage state. |
| `apps/api/` | Go API/worker and qualification bootstrap | PostgreSQL-backed audit, checklist, Finding, CAP, Evidence, report/document, notification, and approved-source catalog services. Runtime data is target-bound; optional risk/regulatory enrichment remains advisory and cannot approve catalog content. |
| `apps/api/cmd/{curated-foundation-loader,prepared-roster-loader,approved-aga-catalog-loader}/` | Ordered runtime bootstrap | Reconciles foundation, prepared roster, and approved-source catalog through three release-owned target services. Replay is idempotent and drift fails closed. |
| `apps/api/cmd/curated-foundation-loader/`, `prepared-roster-loader/`, `approved-aga-catalog-loader/` | First-party qualification data plane | Reconciles only the selected target's organizations, exact prepared roster, and the immutable `IMPORTED_APPROVED_SOURCE` package. No human approval, publication, bulk approval, or role-switch shortcut is created. |
| `../../shared/auth/` | Shared AviaAuth durable OIDC runtime | Public OIDC/UI listens on 8080; private provider administration listens separately on 8081 with a mounted bearer secret. Auth PostgreSQL owns credentials, MFA, recovery, signing keys, provider sessions, opaque claim projections, idempotency receipts, and redacted audit state. The gateway routes only public `/identity/*` requests and never routes private administration. |
| `api/openapi/` | Transport contract | Source for checked request/response shapes and generated adapters. |
| `deploy/local/` | Target-neutral connected qualification stack | Full, test, recovery, and tools profiles use the same first-party Auth, PostgreSQL, MinIO, bootstrap, API, worker, gateway, and HTTP product boundaries. AviaWorkspace owns cloud topology and release authority. |
| `scripts/local-stack.sh`, `scripts/check-local-runtime.sh` | Local lifecycle and fail-closed health boundary | Creates or consumes task-owned secrets and target manifests, starts only the selected Compose project, checks exact network membership and required dependency health, and removes only that project. |
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
6. Prepared identity bootstrap enters through the private first-party admin
   boundary; provider-generated subjects are persisted separately from manifest
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
   behavior, target manifests, and local qualification evidence.

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
- Exact target binding, one-time loader authorization, provider subject
  retention, idempotent replay, and fail-closed drift detection.
- URL-encoded server-owned identifiers are decoded at canonical HTTP route
  boundaries before authority, projection, and append-only transition checks.
- Root-oracle and accepted-baseline integrity.
- Imported approved-source questions remain traceable to their package, source
  manifest, and catalog root; optional enrichment never becomes an approval or
  publication fact.

Use the [verification matrix](docs/agent-harness/verification-matrix.md) for
the executable checks that protect these boundaries.
