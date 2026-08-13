# First-Production Route Families Evidence — 2026-07-21

## Outcome

- Evidence status: `verified locally`
- Artifact status: `candidate-only`
- Release status: `release pending`
- Task 13 local release-candidate packet: `not run`
- Production deployment, traffic cutover, legacy removal, production hosting, and a `production-ready` claim: `blocked`

Task 5 ports the owner-approved Core MVP `first-production` route families into the same React application and capability-composed `Backend` already used by the canonical Cabin Inspection scenario. The root Vanilla JavaScript demo remains intact as the behavioral reference and continues to carry all `later` and `demo-only` concepts.

## Test-First Evidence

The first focused runs failed because the versioned OpenAPI contract had no organization route, the behavior ledger was still version 2, and the reusable Backend contract had no organization or planning capabilities. The first complete real-HTTP run then found two unexpected 404 console records at each viewport: the Department Manager dashboard tried to fetch the canonical Finding before that scenario had created it. The dashboard was corrected to use the server-shaped Finding list projection and select the canonical item only when present.

The final route-family tests cover exact entity identity, role/stage/revision authority, organization scope, decision reasons, idempotent replay, payload-drift rejection, auditee-safe projections, meaningful visible controls, audit-event creation, and the same desktop/tablet/mobile scenario in mock and HTTP profiles.

## Implemented Route Families

- Organization Registry: Department Manager sees active oversight organizations; Auditee access remains restricted to its own organization and internal CAA fields are structurally absent.
- Audit Plan Calendar: Department Manager sees the exact planning item, scheduled date, current status, current owner, budget, and next action.
- Planning authority chain: Finance `APPROVE_BUDGET` -> General Manager `FORWARD_FOR_FINAL_APPROVAL` -> Executive Director `APPROVE_PLAN` -> General Manager `RELEASE_PLAN`. Each command requires a reason and exact expected revision. `RETURN_FOR_REVISION` is an actual server-shaped transition, not a visual placeholder.
- Versioned configuration preview: Admin Preview reads published checklist-template versions and the deterministic Due Date reminder rules at 30, 15, 7, 0, and -1 days. Broad configuration and regulatory authoring remain outside this slice.
- Planning Audit Trail: Admin Preview reads the append-only planning decision projection. This local candidate does not claim production tamper evidence.

The behavior ledger is version 3 with 15 executable entries: the eight role entries, the canonical Cabin workflow action, and six Task 5 route-family actions. Accepted differences remain explicit and do not weaken lifecycle, authority, organization-isolation, or privacy invariants.

## Contract And Authority Boundary

- OpenAPI adds versioned organization, planning, checklist-template-version, reminder-rule, and audit-event projections plus planning decisions.
- `Backend` remains one capability-composed application contract. `MemoryMockStore`/`MockBackend` and the thin generated-transport `HttpBackend` satisfy the same reusable contract.
- The Go planning service locks the exact row, derives the actor from the authenticated principal, enforces role/stage/revision, and records the planning update, audit event, idempotency result, and outbox message in one PostgreSQL transaction.
- Admin configuration and audit projections are server-authorized. The browser route is not treated as an authorization control.
- All canonical mock mutations remain behind `Backend`. Task 5 adds no second repository family and does not bypass the existing field-only `FieldRepository` boundary.
- The normal HTTP build artifact remains free of mock, seed, demo-public, and test-profile inputs.

## Fresh Verification

The following completed with exit code `0`:

- `./scripts/test-http-profile.sh`: direct API/worker builds; full Go `-race` package and live PostgreSQL/OIDC-provider/MinIO integration suite including migration v6 and retained N-1 upgrade; OpenAPI 6/6; clean SQLC regeneration; TypeScript; React/Vitest 16 files and 146/146 tests; demo and HTTP builds; HTTP artifact scan across 12 files and 89 inputs; live HTTP Backend contract 11/11; mock Playwright 4/4; HTTP Playwright 5/5; container/network/volume cleanup
- Focused mock Backend contract: 11/11
- Focused router and Backend tests: 14/14
- Focused fake-fetch `HttpBackend` mapper tests: 7/7
- Focused OpenAPI and behavior-ledger assertions: 8/8
- First-production route matrix alone: mock 3/3 and HTTP 3/3 at 1440x900, 820x1180, and 390x844

Every route-matrix viewport had meaningful role content, no critical horizontal overflow, and zero unexpected browser console warnings/errors. The full HTTP scenario also retained canonical lifecycle, offline-sync, and raw forbidden-field checks. Final static/process gates are recorded in the Task 5 plan result.

## Preserved And Deferred Scope

The root `index.html`, `css/`, `js/`, and legacy tests are unchanged. AI, advanced risk/BI, broad regulatory editing, USOAP/SSP expansion, enforcement case management, and generic workflow surfaces remain `later` or `demo-only` in that intact demo. Task 13 remains `not run` at this checkpoint. Production provider selection, real production OIDC/MFA, MDM, secrets, monitoring/on-call, retention/legal hold, backup/restore and disaster-recovery acceptance, deployment, traffic routing, cutover, and legacy removal remain `blocked` behind a separately approved release/operations plan.
