# Module Architecture

## Core modules

| Module | Purpose | MVP priority |
|---|---|---|
| Organization Registry | Master list of audited organizations | Must |
| Surveillance Planning | Annual/ad hoc audit planning | Must |
| Audit Execution | Audit record and progress | Must |
| Checklist Builder/Runner | Template and execution | Must |
| Findings Management | Central finding lifecycle | Must |
| Auditee Portal | External organization actions | Must |
| CAP Management | Root cause and corrective action | Must |
| Evidence Repository | Evidence upload/review/versioning | Must |
| Notifications | Due-date and status messages | Must |
| Dashboard/Reports | Oversight visibility | Must |
| Admin Configuration | Templates, roles and rules | Basic MVP |
| Mobile Inspection App | Field/offline support | Later |
| Risk-Based Planning | Smart frequency/risk suggestions | Later |
| Enforcement Integration | Escalation and legal cases | Later |

## Shared platform services

AviaSurveil360 may share these with other AVIA products:

- Users and roles
- Organization registry
- Document storage
- Notification service
- Audit trail
- Reporting shell
- Risk flags

## Architecture UX rule

Backend can be modular and configurable. Frontend must remain role-based and task-based. Do not expose backend modules as menu items just because they exist.

## Approved target implementation architecture — 22 July 2026

The target is one browser application with two explicit build profiles and one
modular backend. The same 86 React routes must run against either a deterministic
`MockBackend` for demonstrations or a real same-origin `HttpBackend` backed by
Go and PostgreSQL. Demo mode is not a second UI and HTTP mode must not import
mock data, root-demo JavaScript, or test-only authentication.

The selected approach is a modular monolith with external platform services.
Microservices, Kafka, RabbitMQ, Kubernetes, and a second frontend framework are
not justified for the current scope. The API and background workers remain
separate processes built from one Go module so operational scaling does not
split domain ownership prematurely.

Alternatives considered:

1. **Independent microservices per product module — rejected now.** It would add
   distributed transactions, duplicated authorization, service discovery, and
   operational load before measured scale or team boundaries justify them.
2. **One all-in-process monolith including identity, scanning, email, and PDF —
   rejected.** It would couple security/platform lifecycles to domain code and
   prevent independent failure/restart/resource boundaries.
3. **Modular Go monolith plus bounded platform services — selected.** It keeps
   domain transactions and authorization coherent while allowing first-party
   OIDC, SMTP, object storage, and telemetry to operate through separate
   process boundaries. Content scanning is a disabled, fail-closed adapter and
   document rendering is implemented in Go.

### Application components

| Component | Selected technology | Responsibility |
|---|---|---|
| Browser client | React 19, TypeScript, Vite, React Router, TanStack Query, React Hook Form, Zod | All 86 role/task routes, demo and HTTP profiles, accessibility, responsive UI, and truthful action state |
| Demo data boundary | `MockBackend` and deterministic `MemoryMockStore` | Complete 86-route demonstration and repeatable browser tests without a server |
| HTTP data boundary | Generated OpenAPI transport plus `HttpBackend` | Same frontend capability contract mapped to real same-origin HTTP requests |
| Offline field boundary | Dexie/IndexedDB, OPFS, Service Worker | Inspector package, checklist, Potential Finding, Inspection Attachment, and causal foreground synchronization only |
| API | Go 1.26 modular monolith, `chi`, generated OpenAPI types | Authentication boundary, authorization, validation, projections, commands, idempotency, and audit events |
| Persistence | PostgreSQL 17, `pgx`, `sqlc`, forward-only migrations | Authoritative transactional state, append-only audit records, idempotency, change feed, and outbox |
| Workers | Go commands from the same module | Outbox delivery, Evidence scanning, notifications, documents, and scheduled reminder work |
| Identity | First-party Go OIDC with private administration and TOTP MFA | Local OIDC, exact authority revisions, user lifecycle, session revocation, recovery, and MFA |
| Object storage | Private MinIO buckets | Immutable Evidence, Inspection Attachment delivery, generated report versions, quarantine, and backup artifacts |
| Malware scanning | Disabled scanner adapter | Explicit `not-run` result; fail closed before Evidence review or download |
| Email | SMTP adapter with Mailpit locally | Observable local delivery of notification and reminder messages without an external provider |
| Document rendering | Native Go renderer | Versioned PDF rendering from approved report/document HTML templates |
| Local gateway | Caddy | Local HTTPS, static HTTP artifact, same-origin `/api` and `/auth` routing, security headers, and service isolation |
| Secrets | Docker secrets for local runtime; SOPS + age for encrypted configuration; AWS Secrets Manager/SSM later | No committed plaintext runtime credentials |
| Telemetry | OpenTelemetry Collector, Prometheus, Grafana, Loki, Tempo, Alertmanager | Metrics, logs, traces, dashboards, alert routing, and local operational exercises |
| PostgreSQL backup | pgBackRest | Separate application and first-party auth database full/differential/incremental backups, identity/application fingerprints, retention, and restore verification |
| Object backup | MinIO versioning/object lock plus verified mirror to a logically isolated local backup store | Exact object/version recovery without treating versioning as the only backup; local same-host evidence is not host-loss recovery |
| Local orchestration | Docker Compose profiles | `demo`, `full`, `test`, `observability`, and `recovery` execution lanes |
| Future AWS trial | Terraform resource modules composed by Terragrunt | Repeatable VPC, load balancer, EC2 application runtime, RDS PostgreSQL, S3, ECR, KMS, Secrets Manager, telemetry, and backup resources after local acceptance; Terraform owns resources while Terragrunt owns environment composition and generated backend wiring |

### Domain ownership

The Go module keeps bounded packages for identity, organizations, planning,
inspections, teams/assignments, checklists/templates/questions, Potential
Findings, Findings, CAP, Evidence, reports, documents, communications,
notifications/reminders, risk/analytics projections, administration,
configuration, audit log, sync, and the bounded Inspector-assistant draft
provider. Packages may read another module only through an application service
or an authorized projection. They must not update another module's tables.

The Inspector assistant remains advisory. Its local provider is deterministic
and server-hosted; it cannot create a Finding, change severity, close work, or
make an enforcement decision. A future external model provider must implement
the same audited draft interface and requires a separate governance decision.

### Request and event flow

1. Caddy terminates local HTTPS and serves the HTTP React artifact.
2. The first-party provider completes OIDC; the Go BFF stores provider tokens server-side and
   issues the Secure, HttpOnly, SameSite application session.
3. React calls only same-origin `/api` and `/auth` routes through `HttpBackend`.
4. The API authorizes object and field scope before loading or mutating domain
   state.
5. A command transaction writes the domain mutation, audit event, idempotency
   response, authorized change record, and outbox item together.
6. Workers claim outbox work and call the disabled scanner adapter, SMTP/Mailpit
   locally, the native Go renderer, or MinIO through typed adapters. Retries
   are idempotent and observable.
7. React invalidates typed queries or processes authorized sync changes; no UI
   action relies on a toast as its only durable effect.
8. Deterministic reset/seed behavior exists only in a scoped test-profile
   one-shot lane. The normal OIDC/full API registers no `/__test/*` route.

### Local acceptance boundary

The local system is accepted only when all 86 screens work in both demo and
HTTP profiles, every visible action has a real mock and HTTP outcome, all
required multi-role scenarios replay against PostgreSQL, the complete Compose
stack starts from a clean machine, normal OIDC with MFA works, scan/email/PDF
workers are observable, application and first-party auth database plus object
backup/restore and RPO/RTO drills pass, normal full mode exposes no test reset
route, and no task-owned process or container is left behind.

Local acceptance is not production deployment. Plans 1–4 were accepted on 28
July 2026 only as completed local `candidate-only` milestones. Release remains
`release pending`; deployment and production readiness remain `not run`. AWS
Task 10 is an optional branch outside the local completion gate and remains
unauthorized and `not run`; any future discovery, planning, bootstrap,
foundation/ECR, artifact-publication, data/runtime, smoke, rollback, or
retain/destroy action requires its own explicit authorization and reviewed
inputs. Traffic cutover, production legal/records policy, external penetration
testing, production identity federation, external email and other provider
contracts, data residency, production SLO/RPO/RTO, and on-call staffing remain
separate owner approval gates.

### Binding execution plans

1. `docs/exec-plans/completed/2026-07-22-full-react-86-screen-migration-plan.md`
2. `docs/exec-plans/completed/2026-07-22-full-backend-scenario-parity-plan.md`
3. `docs/exec-plans/active/2026-08-11-first-party-go-oidc-auth-replacement-plan.md`
