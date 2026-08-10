# AviaSurveil360 Planning Pack, Frontend Demo, And React/Go Candidate

This repository contains a structured planning pack, the original
**frontend-only static clickable demo**, and a separate `candidate-only`
React/Go application for a proposed Civil Aviation Authority surveillance,
audit, Findings, CAP, and Evidence management product.

The intact legacy demo is `index.html` with `css/`, `js/`, browser-local mock
state, and Node smoke tests under `tests/`. The authorized first executable
React application is under `apps/web/`; it uses TypeScript, Vite, build-time
separated mock and HTTP entries, one capability-composed `Backend`, and a
versioned OpenAPI contract under `api/openapi/`. The one-module Go API/worker is
under `apps/api/`. Its local verification profile uses pinned PostgreSQL,
Keycloak, MinIO, and a deterministic scanner adapter.

**Candidate-only / not production-ready:** the Go/PostgreSQL authority layer,
private bounded object upload, deterministic scan worker, full canonical HTTP
scenario, PWA/readiness, atomic offline field repository/outbox, and
manifest-first OPFS Inspection Attachment recovery, typed foreground sync,
approved first-production route families, and the complete local
release-candidate matrix are `verified locally`; they are not deployed
production services. The local recommendation is `GO` for a `candidate-only`
artifact whose release remains `release pending`. Production OIDC/MFA,
production object store/scanner or Evidence records policy, deployment,
cutover, and legacy removal remain `blocked`. The root demo remains the
removal-blocking behavior oracle.

## Product definition

**AviaSurveil360** is a task-based oversight platform for Civil Aviation
Authorities. It helps CAA teams plan audits, run checklists, issue findings,
receive corrective action plans from auditees, review evidence, track due dates,
close findings, and monitor oversight performance.

## Core product thesis

Do not build an EMPIC-like complex enterprise screen first. Build a simple, role-based product where each user immediately understands what to do next:

- Inspector: What do I inspect or review today?
- Auditee: What does the CAA need from my organization?
- Manager: Where are we exposed, delayed or overloaded?
- Admin: Which template or rule must be configured?

## Recommended reading order

1. `docs/index.md`
2. `docs/product-specs/index.md`
3. `docs/demo-handoff/CODEX_DEMO_ONLY_PROMPT.md`
4. `docs/demo-evidence/BUILD_SUMMARY.md`
5. `docs/exec-plans/index.md`
6. `docs/demo-evidence/REACT_MOCK_SLICE_2026-07-20.md`
7. `docs/demo-evidence/BOUNDED_UPLOAD_AND_HTTP_PARITY_2026-07-21.md`
8. `docs/demo-evidence/PWA_OFFLINE_READINESS_2026-07-21.md`
9. `docs/demo-evidence/INDEXEDDB_FIELD_STORAGE_2026-07-21.md`
10. `docs/demo-evidence/OPFS_INSPECTION_ATTACHMENT_RECOVERY_2026-07-21.md`
11. `docs/demo-evidence/LOCAL_RELEASE_CANDIDATE_2026-07-21.md`

For the clickable demo, open `index.html` directly in a browser or serve this
folder with a local static server. See `docs/demo-evidence/BUILD_SUMMARY.md` for current
verification status and known demo limitations.

For the React mock candidate:

```bash
npm --prefix apps/web ci
npm --prefix apps/web run dev:demo
```

To manage the same local server in the background, use `make demo-up`,
`make demo-status`, and `make demo-down`. The default URL is
`http://127.0.0.1:4173/`.

See `docs/demo-evidence/REACT_MOCK_SLICE_2026-07-20.md` for the exact verified
scope, commands, transcript, and exclusions.

For the disposable canonical API-backed preprod profile (PostgreSQL, Keycloak,
Go API, HTTP React shell, and the 1,310-question exercise catalog):

```bash
make preprod-up
```

The command prints the local URL and synthetic login credentials. Startup
builds a fresh target and can take several minutes. Use `make preprod-status`
to check API/web health and the loaded catalog, and `make preprod-down` to
remove the disposable containers, volumes, and temporary credentials. The
obsolete AGA-only stakeholder runtime and its operator aliases were physically
removed after donor-free qualification. This canonical profile remains
`candidate-only`; it does not establish production readiness.

For an explicitly authorized public demo backed by this Mac, the named
Cloudflare Tunnel profile targets `https://demo.aviasurveil.com` while keeping
the origin loopback-only. Create the remotely managed tunnel/public-hostname
route in Cloudflare first, then store only its connector token in macOS
Keychain and start the separate profile. Copy the complete raw `eyJ...` value
from the install command—either its final argument or the value after
`--token`—even when Cloudflare visually wraps the command:

```bash
make preprod-cloudflare-demo-token
make preprod-cloudflare-demo-up
make preprod-cloudflare-demo-status
make preprod-cloudflare-demo-users
make preprod-cloudflare-demo-down
```

The terminal prompt is hidden and its Security-framework writer supports the
complete long connector value without the macOS `security -w` prompt's
128-byte ceiling. The token is not stored in the repository, shell history,
process arguments, logs, or runtime files. See the
[start/stop runbook](docs/operations/runbooks/START_STOP.md#named-cloudflare-tunnel-at-demoaviasurveilcom)
for the exact Cloudflare dashboard route, security boundary, rotation, and
cleanup behavior. This remains a public `candidate-only` local-origin demo, not
an external preprod deployment or production-readiness evidence.

For the complete local HTTP candidate profile:

```bash
./scripts/test-http-profile.sh
```

See
`docs/demo-evidence/BOUNDED_UPLOAD_AND_HTTP_PARITY_2026-07-21.md` for the
bounded upload/scan contract, real HTTP parity, fresh local gates, and explicit
production exclusions.

For the dedicated persistent-profile PWA/offline foundation check:

```bash
npm --prefix apps/web run test:e2e:offline
```

See `docs/demo-evidence/PWA_OFFLINE_READINESS_2026-07-21.md` for the exact
readiness, app-shell caching, browser-restart, two-client update, and explicit
site-data-loss evidence and exclusions.

See `docs/demo-evidence/INDEXEDDB_FIELD_STORAGE_2026-07-21.md` for the Task 7
atomic field-record/outbox, migration, server-stopped restart-recovery, and
explicit sync/OPFS exclusions at that checkpoint.

See
`docs/demo-evidence/OPFS_INSPECTION_ATTACHMENT_RECOVERY_2026-07-21.md` for the
Task 8 manifest-first OPFS lifecycle, startup reconciliation, no-delete policy,
and server-stopped attachment restart evidence.

For the complete local release-candidate decision, security/operational matrix,
dependency/SBOM review, browser profiles, and PostgreSQL/object-store recovery
drill, see
`docs/demo-evidence/LOCAL_RELEASE_CANDIDATE_2026-07-21.md`. The decision is
local `GO`, `candidate-only`, and `release pending`; production remains
`blocked` pending a separately approved release/operations plan and explicit
authorization.


## Source Notes
- EMPIC Solutions — surveillance layer, checklists, findings, CAP, evidence, external stakeholder access: https://www.empic.aero/solutions/
- EMPIC-EAP booklet — external stakeholder web client and corrective action handling: https://www.empic.aero/wp-content/uploads/2022/05/Booklet_202205_eng.pdf
- TrustFlight Centrik 5 QMS — aviation quality, safety, risk, workflows, documents and dashboards: https://www.trustflight.com/products/centrik-5-qms/
- ASQS/iQSMS Quality Management Module via Aircraft IT — internal/external audits, auditee guest users, due-date emails and finding status: https://www.aircraftit.com/vendors/comply365-2/quality-management-module/
- ICAO / UK CAAi audit training material — oversight activity, audit reports, findings and observations: https://www.icao.int/sites/default/files/APAC/Meetings/2025/2025%20COSCAP%20SEA%20AND%20UK%20CAAi%20AGA%20CERT%20AND%20SURV/5-Presentations/08-Conducting-Audits_v02_COSCAP.pdf
- Irish Aviation Authority UAM-020 — root cause, CAP, target completion and evidence expectations: https://www.iaa.ie/docs/default-source/publications/advisory-memoranda/uas-advisory-memoranda-%28uam%29/uam-020---guidance-on-the-competent-authority-oversight-audits.pdf
- ANAO CASA surveillance audit report — planning, data, tracking and reporting weaknesses in regulator surveillance: https://www.anao.gov.au/work/performance-audit/civil-aviation-safety-authority-planning-and-conduct-surveillance-activities
- UK CAA compliance monitoring sample checklist/finding form: https://www.caa.co.uk/media/0g3ekh5a/sample-compliance-monitioring-audit-checklists-and-findings-form-1.docx
- FOCA supervision review — Level 1/Level 2 finding handling examples: https://www.newsd.admin.ch/newsd/message/attachments/66693.pdf
- GOV.UK Design System / Service Manual — simple form and question-page patterns: https://design-system.service.gov.uk/patterns/question-pages/ and https://www.gov.uk/service-manual/design/form-structure
- Nielsen Norman Group — complex application and dashboard usability principles: https://www.nngroup.com/articles/complex-application-design/ and https://www.nngroup.com/articles/dashboards-preattentive/
