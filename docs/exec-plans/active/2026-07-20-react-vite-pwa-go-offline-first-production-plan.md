# React Vite PWA And Go Offline-First Production Implementation Plan


**Goal:** Rebuild AviaSurveil360 as a testable React + TypeScript + Vite browser application, preserve the verified demo as a behavioral oracle, prove one canonical workflow against deterministic mock and real Go-backed modes, and produce a locally verified release candidate for managed-browser offline field inspection without claiming production cutover.

**Architecture:** Keep the current Vanilla JavaScript application intact as the behavioral reference until accepted React parity and cutover. Define the canonical OpenAPI vocabulary before either adapter, then build one React codebase with a capability-composed `Backend`: `MockBackend` supplies demo and deterministic frontend-test behavior, while `HttpBackend` maps the generated transport to the same domain contract. Field inspection routes use a distinct local `FieldRepository`, IndexedDB manifests/outbox, OPFS inspection-attachment bytes, a server-issued offline grant, and typed causal sync; management, Auditee, governance, reporting, administration, official Evidence review, approval, issue, and closure remain online-first. The Go API and worker remain one modular-monolith module with PostgreSQL and private S3-compatible storage.

**Tech Stack:** React, TypeScript, Vite, React Router, TanStack Query, React Hook Form, Zod, Dexie/IndexedDB, Service Worker + Cache Storage, OPFS for staged Inspection Attachments, Vitest, React Testing Library, Playwright, Go modular monolith, `net/http` + `chi`, OpenAPI, PostgreSQL + `pgx`/`sqlc`, S3-compatible object storage, and containerized local integration dependencies.

**Status:** `active` — Tasks 2-13 remain implemented and `verified locally` as a `candidate-only` local application. The corrective [React Legacy UI Parity And Backend Integration plan](2026-07-21-react-legacy-ui-parity-and-backend-integration-plan.md), [86-screen React](../completed/2026-07-22-full-react-86-screen-migration-plan.md), and [full backend parity](../completed/2026-07-22-full-backend-scenario-parity-plan.md) milestones remain recorded. The maintained local identity/runtime successor is the [first-party Go OIDC plan](2026-08-11-first-party-go-oidc-auth-replacement-plan.md). Release remains `release pending`; remote deployment and traffic are `not run`.

## Global Constraints

- Use `AviaSurveil360` as the canonical product name.
- The application must remain browser-delivered. Do not replace the browser client with Flutter, React Native, Capacitor, Electron, Tauri, or platform-native applications in this plan.
- The production web frontend is React + TypeScript + Vite. Do not add Next.js or another server-rendered frontend framework without a separately approved architecture change.
- The production backend is a Go modular monolith. Do not introduce microservices until measured scale, compliance boundaries, or team topology justify extraction.
- Preserve the current root `index.html`, `css/`, `js/`, and `tests/` demo until React behavioral parity and stakeholder cutover are explicitly accepted.
- Do not directly translate the existing global JavaScript architecture into React globals. Port verified behavior into typed feature modules and explicit state/data boundaries.
- Define the minimal versioned OpenAPI contract and canonical examples before `HttpBackend` or handwritten transport DTOs. Generated transport code is never the UI domain model.
- Use one capability-composed application `Backend` with `MockBackend` and `HttpBackend`; do not create duplicate repository implementations for every entity and do not create `DemoSyncTransport`.
- Demo/test selection occurs once at application bootstrap. Do not scatter `if (demoMode)` branches through components or domain logic.
- Demo and HTTP artifacts use build-time-separated entrypoints and public directories. HTTP/production artifacts must have no module-graph path to mock handlers or seed data; their public configuration has no backend-mode field and cannot enable mock at runtime.
- Field inspection is offline-first. Planning, management approvals, Auditee CAP/Evidence submission, report issue, administration, and authoritative closure remain online-first unless a later approved plan expands offline scope.
- Service Worker Background Sync is an enhancement only. Required sync triggers are app start, foreground/resume, browser `online`, explicit `Sync now`, and an application-open retry schedule.
- Use Cache Storage only for the app shell and static assets, IndexedDB for structured offline records/outbox/sync metadata, and OPFS for staged Inspection Attachment bytes.
- Call `navigator.storage.persist()` only as part of an explicit offline package check-out readiness flow. A denied or unsupported persistence request blocks official offline check-out but does not block online use. Treat `navigator.storage.estimate()` as advisory, not a reservation or durability guarantee.
- Do not rely on programmatic private-browsing detection. Official offline check-out requires an owner-approved managed browser/device policy plus IndexedDB, OPFS, restart, persistence, version, quota-headroom, and offline-grant capability checks.
- Server-side authorization is authoritative. Frontend route visibility is not an authorization control.
- The server issues and records every offline grant. Client-supplied actor IDs, device IDs, clocks, or expiry values are metadata only and never establish authority.
- CAP acceptance must never close a Finding. Required Evidence, verification, or an explicit authorized closure path remains mandatory.
- Separate `Comment to Auditee` from `Internal CAA Note` across models, API payloads, local storage, sync commands, UI, reports, and tests.
- Preserve Checklist Template, CAP, Evidence, workflow, and report versions; never overwrite submitted Evidence or superseded CAP revisions.
- Use `InspectionAttachment` for bytes staged during offline field execution. Official Auditee `EvidenceVersion` records remain online-first and are created only through the authorized server upload/version workflow.
- Treat regulatory content as configured references, expected Evidence, and finding bases, not legal advice or automatic enforcement.
- Keep official closure, report issue, and approval transitions server-authoritative. An offline client may save a pending draft command, but must not present it as globally accepted until the server acknowledges it.
- Append exactly one domain audit event in the same PostgreSQL transaction for every successful status transition. Append-only database records must not be described as immutable or tamper-evident without separately verified controls.
- Report decisions target immutable report versions. Department Manager and General Manager may return or forward only; Executive Director alone may issue and lock; report issue never closes Findings.
- Never automatically delete unknown OPFS bytes, submitted Evidence, or canonical records. Cache eviction and legal/records disposition are separate, owner-approved behaviors.
- This plan produces a local release candidate. `production-ready`, deployment, traffic cutover, and release require a separately approved operations/release plan and matching evidence.
- Work on the current branch. Do not create, switch, rename, or delete branches; do not stage, commit, push, deploy, open a PR, or post GitHub comments unless the user separately authorizes that exact action.
- Preserve unrelated untracked `docs/demo-evidence/stakeholder/` and `outputs/` content.
- Keep English canonical implementation docs. Update matching `.turkce.md` companions when a stakeholder-facing canonical product or evidence document changes.
- Use literal verification labels: `verified locally`, `not run`, `blocked`, `candidate-only`, `release pending`, and `production-ready` only when the matching evidence exists.

---

## Objective

Deliver the production transition through independently reviewable slices:

1. Freeze the current demo as a behavioral contract instead of using it as production architecture.
2. Create a React/Vite application beside the legacy demo and establish parity tests before cutover.
3. Define a single typed backend boundary that can use deterministic mock data or the real Go API.
4. Port the canonical Inspection -> Checklist -> Potential Finding -> Finding -> CAP -> Evidence -> CAA Review -> Closure scenario before moving secondary screens.
5. Add PWA app-shell caching and an explicit, testable offline readiness/check-out gate.
6. Make field inspection local-first with atomic IndexedDB response/outbox writes.
7. Stage field Inspection Attachment bytes in OPFS through a manifest-first hash/recovery state machine; keep official Evidence upload online-first and use bounded whole-object retry in the first protocol.
8. Build one Go-module modular monolith with REST/OpenAPI, PostgreSQL, private S3-compatible storage, server-side authorization, versioned workflows, and an audit event for every successful status transition.
9. Implement typed, causally ordered, idempotent push/pull sync with server-issued grants, revision conflicts, filtered projections, and exact acknowledgement replay.
10. Run the same critical browser scenarios in mock mode and against the real Go/PostgreSQL/S3 integration environment.
11. Produce a local release-candidate recommendation only after parity, offline, security, migration, browser, and operational-local gates pass; production cutover remains a separate explicit decision.

## Scope

### In Scope

- A new `apps/web/` React + TypeScript + Vite browser application.
- A new `apps/api/` Go modular monolith with API and worker entrypoints under the same Go module.
- A versioned OpenAPI contract under `api/openapi/`.
- Mock and HTTP backend implementations behind one application-level contract.
- Deterministic in-memory mock storage for the first slice; optional IndexedDB-backed mock-server persistence only after its ownership boundary is proven necessary.
- A classified route inventory (`first-production`, `later`, or `demo-only`) and executable React/legacy behavior parity for each migrated route.
- A local-first field inspection package, checklist response, Potential Finding draft, Inspection Attachment manifest, outbox, and sync model.
- PWA Service Worker, app-shell caching, schema/version checks, offline startup, and update safety.
- Persistent-storage readiness, advisory quota/headroom checks, supported managed-browser policy enforcement, and ephemeral/unmanaged-storage rejection for official offline check-out.
- Manifest-first OPFS Inspection Attachment staging, hashing, crash recovery, and bounded whole-object upload retry in the first production protocol.
- PostgreSQL migrations and generated/checked SQL access.
- Server-side role, organization, section/domain, assignment, object, and transition authorization.
- Real Evidence metadata/versioning, S3 object references, file hash, upload status, and malware-scan state contract.
- A domain audit event for every successful status transition plus transactional server-outbox records.
- Mock, component, Go unit, Go integration, contract, offline, security, migration, and dual-mode browser tests.
- Bilingual stakeholder evidence updates after implementation verification when a matching Turkish companion exists.
- Active-plan, completed-plan, and technical-debt tracking through the repository lifecycle.

### Out Of Scope

- Native or wrapped mobile/desktop applications.
- Microservices, event-streaming platforms, Kubernetes, service mesh, or multi-region active-active infrastructure.
- Automatic conflict merging for Finding severity, Evidence review, closure, report issue, regulatory publication, or enforcement-related decisions.
- AI decision-making, real regulatory ingestion, official USOAP scoring, automatic enforcement, certificate action, or legal advice.
- Full offline capability for Manager, Finance, General Manager, Executive Director, Auditee, Administration, reporting, or regulatory publishing routes.
- A complex generic workflow designer.
- Advanced BI, a general report builder, or a separate search cluster in the first production slice.
- Storage Buckets API, SQLite WASM, CRDTs, PouchDB/CouchDB replication, or a proprietary sync platform in the initial implementation.
- Claiming that browser persistence can survive explicit site-data deletion, profile deletion, device reset, or storage clearing by the user or device administrator.
- Programmatic private-browsing detection as an authorization, durability, or security control.
- Offline Auditee CAP submission, official Evidence upload/review, management approval, report issue, administration, or authoritative closure.
- Porting every current demo screen before the canonical real HTTP vertical slice passes, or treating demo-only AI/advanced-risk/regulatory surfaces as first-production requirements without product approval.
- Multipart upload, notification delivery, production PDF generation, or destructive retention processing in the first backend slice.
- Deleting the legacy demo during implementation. Removal or archival requires a separately approved cutover action.
- Branch operations, commits, pushes, deployment, release, GitHub comments, PRs, or issue writes.

## Assumptions

- Inspectors must use AviaSurveil360 in aircraft, hangars, remote airports, or intermittent-connectivity environments, and an in-progress field inspection must continue without network access.
- The application will be opened online at least once and an assigned inspection package will be explicitly checked out before offline use; first-ever offline loading from an uncached URL is impossible.
- A managed, supported browser/device policy can be established for official offline work. The initial target should be current managed Chromium-based browsers, with Firefox/Safari admitted only after the same offline evidence matrix passes.
- The current local environment baseline is Node `24.16.0`, npm `11.13.0`, and Go `1.26.4`; execution locks exact JavaScript dependencies in `package-lock.json` and Go dependencies in `go.sum` after compatibility checks.
- The current static demo and smoke tests provide useful behavioral evidence but do not prove production authentication, authorization, storage, sync, Evidence chain-of-custody, or browser persistence.
- The canonical OpenAPI contract, not TypeScript or Go persistence structs, is the transport source of truth. UI domain types remain explicitly mapped from generated transport types.
- PostgreSQL is the canonical relational source of truth and S3-compatible object storage is the canonical Evidence/report byte store after server acknowledgement.
- IndexedDB/OPFS data is a recoverable offline working set, not the final authoritative record.
- The Go server creates authoritative status transitions and audit events only after authentication, authorization, idempotency, revision, and validation checks pass.
- Client-generated UUID/ULID operation IDs are acceptable for offline idempotency; the server computes the canonical semantic payload hash, derives the actor, and allocates or confirms public record numbers.
- Background browser APIs may be throttled or unavailable, so visible foreground sync remains mandatory.
- `navigator.storage.persist()` may be denied. Production offline check-out is blocked on that device/browser when persistence is not granted.
- The preferred browser-auth topology is a same-origin Go session/BFF using OIDC Authorization Code flow and secure, HTTP-only cookies so browser code does not retain access or refresh tokens. Security owner approval, provider selection, MFA, CSRF details, and session policy remain blocking decisions for the HTTP/auth slice.
- A Go modular monolith is accepted only after an engineering/platform ADR confirms team ownership, deployment, observability, and on-call capability; browser delivery alone is not the justification for Go.
- An OIDC provider, MFA policy, offline authorization grant duration, local-data protection policy, retention policy, malware scanner, hosting region, RPO, and RTO will be selected by their respective owners before the dependent slice.

## Chosen Architecture

### Runtime Topology

```text
Browser
  React/Vite PWA
  ├─ Demo build entry ──> MockBackend ──> MemoryMockStore
  ├─ HTTP build entry ──> HttpBackend ─────────────────────┐
  └─ Field routes                                          │
      ├─ FieldRepository                                   │
      │   ├─ IndexedDB records/manifests/outbox/cursors     │
      │   └─ OPFS InspectionAttachment bytes               │
      └─ foreground SyncEngine ──> Backend.sync ────────────┤
                                                            v
                                                Go modular monolith
                                                ├─ cmd/api
                                                ├─ cmd/worker
                                                ├─ PostgreSQL
                                                └─ private S3-compatible storage
```

`MockStore` simulates remote canonical state and is reachable only through
`MockBackend`. `FieldRepository` is the browser-local working set in both demo
and HTTP modes. Sync is the only bridge between them; components never dual-write
to a mock store and the local field database.

### Backend Boundary

Create `apps/web/src/backend/backend.ts` with the capability-composed boundary
below. `MockBackend` and `HttpBackend` each implement this single top-level
contract. Capability facets organize the API; they are not separate mock/real
repository families. Generated OpenAPI transport types are mapped into these UI
domain types, and the UI never depends on Go persistence structs.

```ts
export type BackendMode = "mock" | "http";

export interface Backend {
  readonly mode: BackendMode;
  readonly assignments: AssignmentBackend;
  readonly inspections: InspectionBackend;
  readonly potentialFindings: PotentialFindingBackend;
  readonly findings: FindingBackend;
  readonly caps: CapBackend;
  readonly inspectionAttachments: InspectionAttachmentBackend;
  readonly evidence: EvidenceBackend;
  readonly reports: ReportBackend;
  readonly dashboards: DashboardBackend;
  readonly sync: SyncBackend;
}

export interface AssignmentBackend {
  list(input: ListAssignmentsInput): Promise<ListAssignmentsOutput>;
}

export interface InspectionBackend {
  getPackage(input: { packageId: string }): Promise<InspectionPackage>;
  checkout(input: CheckoutInspectionPackageInput): Promise<CheckoutInspectionPackageOutput>;
  upsertChecklistResponse(input: UpsertChecklistResponseInput): Promise<ChecklistResponseView>;
  submitChecklist(input: SubmitChecklistInput): Promise<SubmitChecklistOutput>;
  reopenChecklist(input: ReopenChecklistInput): Promise<SubmitChecklistOutput>;
}

export interface PotentialFindingBackend {
  create(input: CreatePotentialFindingInput): Promise<PotentialFindingView>;
  decide(input: DecidePotentialFindingInput): Promise<PotentialFindingDecisionOutput>;
}

export interface FindingBackend {
  list(input: ListFindingsInput): Promise<ListFindingsOutput>;
  get(input: { findingId: string }): Promise<FindingView>;
  authorizedClose(input: AuthorizedCloseInput): Promise<FindingView>;
}

export interface CapBackend {
  submit(input: SubmitCapInput): Promise<SubmitCapOutput>;
  review(input: ReviewCapInput): Promise<ReviewCapOutput>;
}

export interface InspectionAttachmentBackend {
  beginUpload(input: BeginInspectionAttachmentUploadInput): Promise<BeginInspectionAttachmentUploadOutput>;
  completeUpload(input: CompleteInspectionAttachmentUploadInput): Promise<CompleteInspectionAttachmentUploadOutput>;
}

export interface EvidenceBackend {
  beginUpload(input: BeginEvidenceUploadInput): Promise<BeginEvidenceUploadOutput>;
  completeUpload(input: CompleteEvidenceUploadInput): Promise<CompleteEvidenceUploadOutput>;
  review(input: ReviewEvidenceInput): Promise<ReviewEvidenceOutput>;
}

export interface ReportBackend {
  getVersion(input: { reportVersionId: string }): Promise<ReportVersionView>;
  decide(input: DecideReportInput): Promise<ReportVersionView>;
}

export interface DashboardBackend {
  getManagerProjection(input: { organizationId?: string }): Promise<ManagerDashboardProjection>;
}

export interface SyncBackend {
  pushOperation(input: PushFieldOperationRequest): Promise<PushFieldOperationResult>;
  pull(input: SyncPullRequest): Promise<SyncPullResponse>;
}
```

Before Task 3, `docs/product-specs/data-and-rules/PRODUCTION_CONTRACT_VOCABULARY.md`
must map every source label, legacy value, OpenAPI enum, and UI label. The
following values are the minimum candidate vocabulary; Product and CAA
Operations must approve the mapping before generation. `Due Soon`, `Due Today`,
and `Overdue` are computed `DueState` values, not lifecycle statuses.

```ts
export type LocalDate = string;
export type Instant = string;

export type Role =
  | "inspector"
  | "leadInspector"
  | "manager"
  | "finance"
  | "gm"
  | "executiveDirector"
  | "auditee"
  | "admin";

export type ChecklistAnswer =
  | "COMPLIANT"
  | "NON_COMPLIANT"
  | "OBSERVATION"
  | "NOT_APPLICABLE"
  | "NOT_CHECKED";

export type DueState = "NONE" | "NOT_DUE" | "DUE_SOON" | "DUE_TODAY" | "OVERDUE";

export type PotentialFindingStatus =
  | "PENDING_LEAD_REVIEW"
  | "RETURNED"
  | "DISMISSED"
  | "CONVERTED";

export type FindingStatus =
  | "DRAFT"
  | "OPEN"
  | "WAITING_FOR_CAP"
  | "CAP_SUBMITTED"
  | "CAP_ACCEPTED"
  | "CAP_REJECTED"
  | "CAP_MORE_INFORMATION_REQUESTED"
  | "EVIDENCE_REQUIRED"
  | "EVIDENCE_SUBMITTED"
  | "PENDING_CAA_REVIEW"
  | "EVIDENCE_MORE_INFORMATION_REQUESTED"
  | "PENDING_CLOSURE"
  | "CLOSED"
  | "ESCALATED";

export type CapStatus =
  | "DRAFT"
  | "SUBMITTED"
  | "PENDING_CAA_REVIEW"
  | "ACCEPTED"
  | "REJECTED"
  | "MORE_INFORMATION_REQUESTED"
  | "SUPERSEDED";

export type EvidenceUploadState = "PENDING" | "UPLOADING" | "UPLOADED" | "FAILED";
export type EvidenceScanState = "PENDING" | "CLEAN" | "QUARANTINED" | "FAILED";
export type EvidenceReviewState =
  | "NOT_READY"
  | "PENDING_CAA_REVIEW"
  | "ACCEPTED"
  | "PARTIALLY_ACCEPTED"
  | "REJECTED"
  | "MORE_INFORMATION_REQUESTED";

export type ReportApprovalStatus =
  | "DRAFT"
  | "DEPARTMENT_REVIEW"
  | "GM_REVIEW"
  | "EXECUTIVE_DIRECTOR_REVIEW"
  | "RETURNED"
  | "ISSUED"
  | "LOCKED";

export interface CommandMeta {
  operationId: string;
}

export interface AssignmentSummary {
  auditId: string;
  organizationId: string;
  organizationName: string;
  title: string;
  status: string;
  dueDate: LocalDate | null;
  dueState: DueState;
  nextAction: string;
}

export interface ListAssignmentsInput {
  cursor?: string;
  limit?: number;
  status?: string;
}

export interface ListAssignmentsOutput {
  items: AssignmentSummary[];
  nextCursor: string | null;
}

export interface GetInspectionPackageInput {
  packageId: string;
}

export interface InspectionQuestion {
  id: string;
  sectionId: string;
  prompt: string;
  regulatoryReference: string | null;
  expectedEvidence: string | null;
  allowedAnswers: ChecklistAnswer[];
  commentRequiredFor: ChecklistAnswer[];
  assignedInspectorUserIds: string[];
  currentResponse: ChecklistResponseView | null;
}

export interface InspectionPackage {
  id: string;
  auditId: string;
  organizationId: string;
  packageVersion: number;
  schemaVersion: number;
  protocolVersion: number;
  templateVersionId: string;
  packageDigest: string;
  expiresAt: Instant;
  checklistStatus: "IN_PROGRESS" | "SUBMITTED";
  questions: InspectionQuestion[];
}

export interface ChecklistResponseView {
  id: string;
  questionId: string;
  answer: ChecklistAnswer;
  comment: string;
  revision: number;
  updatedAt: Instant;
}

export interface UpsertChecklistResponseInput extends CommandMeta {
  responseId: string;
  auditId: string;
  questionId: string;
  expectedResponseRevision: number | null;
  answer: ChecklistAnswer;
  comment: string;
}

export interface FindingView {
  id: string;
  findingNumber: string;
  auditId: string;
  organizationId: string;
  title: string;
  description: string;
  regulatoryReference: string | null;
  findingBasis: string;
  severity: "LEVEL_1_CRITICAL" | "LEVEL_2_MAJOR" | "LEVEL_3_MINOR" | "OBSERVATION";
  status: FindingStatus;
  dueDate: LocalDate | null;
  dueState: DueState;
  currentOwnerType: "CAA" | "AUDITEE";
  currentOwnerId: string;
  currentOwnerRole: Role;
  nextAction: string;
  capRequired: boolean;
  evidenceRequired: boolean;
  repeatFinding: boolean;
  createdAt: Instant;
  issuedAt: Instant | null;
  closedAt: Instant | null;
  closureBasis: "EVIDENCE_VERIFIED" | "AUTHORIZED" | null;
  revision: number;
}

export interface ListFindingsInput {
  cursor?: string;
  limit?: number;
  status?: FindingStatus;
}

export interface ListFindingsOutput {
  items: FindingView[];
  nextCursor: string | null;
}

export interface CheckoutInspectionPackageInput extends CommandMeta {
  packageId: string;
  expectedPackageVersion: number;
  deviceInstanceId: string;
}

export interface OfflineGrant {
  grantId: string;
  subjectId: string;
  organizationId: string;
  packageId: string;
  packageVersion: number;
  packageDigest: string;
  allowedCommandTypes: FieldCommandType[];
  assignmentScope: { questionIds: string[] };
  deviceInstanceId: string;
  issuedAt: Instant;
  expiresAt: Instant;
  protocolVersion: number;
}

export interface CheckoutInspectionPackageOutput {
  inspectionPackage: InspectionPackage;
  offlineGrant: OfflineGrant;
}

export interface SubmitChecklistInput extends CommandMeta {
  auditId: string;
  expectedChecklistRevision: number;
}

export interface ReopenChecklistInput extends CommandMeta {
  auditId: string;
  expectedChecklistRevision: number;
  reason: string;
}

export interface SubmitChecklistOutput {
  auditId: string;
  checklistStatus: "IN_PROGRESS" | "SUBMITTED";
  checklistRevision: number;
}

export interface CreatePotentialFindingInput extends CommandMeta {
  auditId: string;
  questionId: string;
  checklistResponseId: string;
  expectedChecklistResponseRevision: number;
  title: string;
  description: string;
  requiredComment: string;
  inspectionAttachmentIds: string[];
}

export interface PotentialFindingView {
  id: string;
  auditId: string;
  questionId: string;
  organizationId: string;
  title: string;
  description: string;
  status: PotentialFindingStatus;
  revision: number;
  convertedFindingId: string | null;
}

export type DecidePotentialFindingInput =
  | (CommandMeta & {
      potentialFindingId: string;
      expectedPotentialFindingRevision: number;
      decision: "RETURN" | "DISMISS";
      reason: string;
    })
  | (CommandMeta & {
      potentialFindingId: string;
      expectedPotentialFindingRevision: number;
      decision: "CONVERT";
      severity: FindingView["severity"];
      capRequired: boolean;
      evidenceRequired: boolean;
      dueDate: LocalDate | null;
    });

export interface PotentialFindingDecisionOutput {
  potentialFinding: PotentialFindingView;
  finding: FindingView | null;
}

export interface SubmitCapInput {
  operationId: string;
  findingId: string;
  expectedFindingRevision: number;
  rootCause: string;
  correctiveAction: string;
  preventiveAction: string;
  responsiblePerson: string;
  targetCompletionDate: LocalDate;
  commentToCaa: string;
}

export interface SubmitCapOutput {
  capRevisionId: string;
  capRevision: number;
  capStatus: "SUBMITTED" | "PENDING_CAA_REVIEW";
  findingStatus: FindingStatus;
  findingRevision: number;
}

export interface ReviewCapInput extends CommandMeta {
  capRevisionId: string;
  expectedCapRevision: number;
  findingId: string;
  expectedFindingRevision: number;
  decision: "ACCEPT" | "REJECT" | "REQUEST_MORE_INFORMATION";
  commentToAuditee: string;
  internalCaaNote: string;
}

export interface ReviewCapOutput {
  capRevisionId: string;
  capRevision: number;
  capStatus: CapStatus;
  findingStatus: FindingStatus;
  findingRevision: number;
}

export interface BeginInspectionAttachmentUploadInput extends CommandMeta {
  inspectionAttachmentId: string;
  packageId: string;
  byteSize: number;
  sha256: string;
  fileName: string;
  declaredMediaType: string;
}

export interface BeginInspectionAttachmentUploadOutput {
  uploadId: string;
  stagingObjectKey: string;
  uploadUrl: string;
  requiredHeaders: Record<string, string>;
  expiresAt: Instant;
  maximumByteSize: number;
}

export interface CompleteInspectionAttachmentUploadInput extends CommandMeta {
  uploadId: string;
  sha256: string;
  byteSize: number;
}

export interface CompleteInspectionAttachmentUploadOutput {
  inspectionAttachmentId: string;
  uploadState: "UPLOADED";
  scanState: "PENDING";
}

export interface ReviewEvidenceInput {
  operationId: string;
  evidenceVersionId: string;
  expectedEvidenceVersionRevision: number;
  findingId: string;
  expectedFindingRevision: number;
  decision: "CLOSE" | "PARTIALLY_CLOSE" | "NOT_CLOSE" | "REQUEST_MORE_INFORMATION";
  commentToAuditee: string;
  internalCaaNote: string;
}

export interface ReviewEvidenceOutput {
  reviewDecisionId: string;
  evidenceVersionId: string;
  evidenceVersionRevision: number;
  findingStatus: FindingStatus;
  findingRevision: number;
}

export interface AuthorizedCloseInput extends CommandMeta {
  findingId: string;
  expectedFindingRevision: number;
  reason: string;
}

export interface BeginEvidenceUploadInput extends CommandMeta {
  findingId: string;
  expectedFindingRevision: number;
  fileName: string;
  declaredMediaType: string;
  byteSize: number;
  sha256: string;
}

export interface BeginEvidenceUploadOutput {
  uploadId: string;
  stagingObjectKey: string;
  uploadUrl: string;
  requiredHeaders: Record<string, string>;
  expiresAt: Instant;
  maximumByteSize: number;
}

export interface CompleteEvidenceUploadInput extends CommandMeta {
  uploadId: string;
  sha256: string;
  byteSize: number;
}

export interface CompleteEvidenceUploadOutput {
  evidenceVersionId: string;
  version: number;
  uploadState: "UPLOADED";
  scanState: "PENDING";
  reviewState: "NOT_READY";
}

export interface ReportVersionView {
  reportVersionId: string;
  reportId: string;
  organizationId: string;
  contentHash: string;
  version: number;
  status: ReportApprovalStatus;
  revision: number;
  issuedAt: Instant | null;
}

export interface DecideReportInput extends CommandMeta {
  reportVersionId: string;
  expectedReportVersionRevision: number;
  decision: "RETURN" | "FORWARD" | "ISSUE_AND_LOCK";
  reason: string;
}

export interface ManagerDashboardProjection {
  generatedAt: Instant;
  openFindings: number;
  overdueFindings: number;
  pendingCapReviews: number;
  pendingEvidenceReviews: number;
}
```

### Build-Time Backend Selection

Do not import mock code behind a runtime branch. Use two explicit entrypoints:

```ts
// apps/web/src/entry/demo.tsx
import { createMockBackend } from "../mock/create-mock-backend";
bootstrap({ backend: createMockBackend(), buildProfile: "demo" });

// apps/web/src/entry/http.tsx
import { createHttpBackend } from "../backend/http-backend";
bootstrap({ backend: createHttpBackend(readPublicHttpConfig()), buildProfile: "http" });
```

`vite.config.ts` selects exactly one entrypoint and exactly one public directory
from a build-time `AVIA_BUILD_PROFILE=demo|http` value. The HTTP build fails if
its Rollup manifest contains a source under `src/mock/` or if its output contains
demo configuration or seed artifacts. Runtime configuration cannot change the
compiled backend family.

### Browser Authentication Boundary

The candidate HTTP topology is same-origin Go session/BFF authentication:

- Go completes the configured OIDC Authorization Code flow.
- Access and refresh tokens remain server-side.
- The browser receives a `Secure`, `HttpOnly`, `SameSite` session cookie and a
  separate CSRF token for state-changing requests.
- `HttpBackend` always sends same-origin credentials and maps `401`, `403`,
  session expiry, CSRF failure, timeout, cancellation, and problem responses.
- A deterministic local test principal is allowed only in an explicit test
  configuration; production startup fails closed when any test identity bypass
  is enabled.
- No session token, refresh token, or server secret is stored in IndexedDB,
  OPFS, Cache Storage, frontend runtime configuration, or the built bundle.

Security owner approval of this topology, the OIDC provider, MFA, session
duration/revocation, CSRF design, and managed local-data controls is a gate for
Task 5. A different topology requires an ADR and matching plan update before
implementation.

### Field Local-First Boundary

Field UI writes only to `FieldRepository`. Structured record changes and their
outbox operations commit in one IndexedDB transaction. Submitted-checklist
reopen remains online-first and reason-required. OPFS/IndexedDB coordination uses
a manifest-first recovery protocol because the two stores cannot share a
transaction.

```ts
export interface FieldRepository {
  checkoutPackage(input: CheckoutInspectionPackageOutput): Promise<CheckedOutPackage>;
  saveChecklistResponse(input: SaveChecklistResponseInput): Promise<ChecklistResponse>;
  savePotentialFindingDraft(input: SavePotentialFindingDraftInput): Promise<PotentialFindingDraft>;
  submitChecklist(input: SubmitLocalChecklistInput): Promise<LocalChecklistState>;
  createAttachmentManifest(input: StageInspectionAttachmentInput): Promise<InspectionAttachmentManifest>;
  writeAttachmentBytes(input: WriteInspectionAttachmentInput): Promise<InspectionAttachmentManifest>;
  listOutbox(): Promise<FieldSyncOperation[]>;
  applyPushResult(result: PushFieldOperationResult): Promise<void>;
  applyPullPage(result: SyncPullResponse): Promise<void>;
}

export interface CheckedOutPackage extends InspectionPackage {
  offlineGrant: OfflineGrant;
  checkedOutAt: Instant;
  syncState: "synced" | "pending" | "conflict" | "rejected";
}

export interface SaveChecklistResponseInput {
  operationId: string;
  packageId: string;
  auditId: string;
  questionId: string;
  baseRevision: number | null;
  answer: ChecklistAnswer;
  comment: string;
  clientOccurredAt: Instant;
}

export interface ChecklistResponse extends SaveChecklistResponseInput {
  id: string;
  updatedAt: Instant;
  syncState: "pending" | "acknowledged" | "conflict" | "rejected";
}

export interface SavePotentialFindingDraftInput {
  operationId: string;
  packageId: string;
  auditId: string;
  questionId: string;
  checklistResponseId: string;
  expectedChecklistResponseRevision: number | null;
  baseRevision: number | null;
  title: string;
  description: string;
  requiredComment: string;
  inspectionAttachmentIds: string[];
  clientOccurredAt: Instant;
}

export interface PotentialFindingDraft extends SavePotentialFindingDraftInput {
  id: string;
  updatedAt: Instant;
  syncState: "pending" | "acknowledged" | "conflict" | "rejected";
}

export interface SubmitLocalChecklistInput {
  operationId: string;
  packageId: string;
  auditId: string;
  baseRevision: number;
  clientOccurredAt: Instant;
}

export interface LocalChecklistState {
  auditId: string;
  status: "IN_PROGRESS" | "SUBMISSION_PENDING" | "SUBMITTED" | "REJECTED";
  revision: number;
}

export interface StageInspectionAttachmentInput {
  attachmentId: string;
  packageId: string;
  auditId: string;
  checklistResponseId: string;
  potentialFindingLocalId: string | null;
  file: File;
}

export interface WriteInspectionAttachmentInput {
  attachmentId: string;
  file: File;
}

export interface InspectionAttachmentManifest {
  attachmentId: string;
  temporaryOpfsPath: string;
  finalOpfsPath: string | null;
  fileName: string;
  mediaType: string;
  byteSize: number;
  sha256: string | null;
  stagingState:
    | "manifest_created"
    | "writing"
    | "ready"
    | "uploading"
    | "acknowledged"
    | "purge_eligible"
    | "quarantined"
    | "missing";
  syncState: "not_ready" | "pending" | "acknowledged" | "rejected";
}

export interface SyncSummary {
  attempted: number;
  accepted: number;
  alreadyApplied: number;
  conflicts: number;
  rejected: number;
  retryable: number;
  completedAt: Instant;
}
```

The sync engine depends on `FieldRepository` and `Backend`; it does not know whether the backend is mock or HTTP.

```ts
export interface SyncEngine {
  syncNow(reason: "startup" | "foreground" | "online" | "manual" | "retry"): Promise<SyncSummary>;
}
```

### Sync Command Envelope

The first production protocol sends one operation at a time in package-causal
order. The repository may coalesce only operations that have never been sent.
Once an operation is in flight, its payload is immutable; a later edit becomes
a new operation after the authoritative revision is known.

```ts
export type FieldCommandType =
  | "UPSERT_CHECKLIST_RESPONSE"
  | "CREATE_POTENTIAL_FINDING"
  | "SUBMIT_CHECKLIST"
  | "REGISTER_INSPECTION_ATTACHMENT";

export interface FieldOperationBase<TType extends FieldCommandType, TPayload> {
  operationId: string;
  protocolVersion: number;
  offlineGrantId: string;
  packageId: string;
  packageVersion: number;
  entityId: string;
  commandType: TType;
  baseRevision: number | null;
  deviceInstanceId: string;
  clientOccurredAt: Instant;
  payload: TPayload;
}

export type FieldSyncOperation =
  | FieldOperationBase<"UPSERT_CHECKLIST_RESPONSE", {
      auditId: string;
      questionId: string;
      answer: ChecklistAnswer;
      comment: string;
    }>
  | FieldOperationBase<"CREATE_POTENTIAL_FINDING", {
      auditId: string;
      questionId: string;
      checklistResponseId: string;
      expectedChecklistResponseRevision: number | null;
      title: string;
      description: string;
      requiredComment: string;
      inspectionAttachmentIds: string[];
    }>
  | FieldOperationBase<"SUBMIT_CHECKLIST", {
      auditId: string;
    }>
  | FieldOperationBase<"REGISTER_INSPECTION_ATTACHMENT", {
      auditId: string;
      checklistResponseId: string;
      potentialFindingOperationId: string | null;
      fileName: string;
      mediaType: string;
      byteSize: number;
      sha256: string;
    }>;

export interface PushFieldOperationRequest {
  operation: FieldSyncOperation;
}

export type PushFieldOperationStatus =
  | "accepted"
  | "already_applied"
  | "conflict"
  | "forbidden"
  | "invalid"
  | "retryable";

export interface AuthorizedConflictDescriptor {
  code: "STALE_REVISION" | "PACKAGE_REVOKED" | "ASSIGNMENT_CHANGED";
  entityId: string;
  authoritativeRevision: number | null;
  authoritativeStatus: string | null;
  changedAt: Instant | null;
}

export interface PushFieldOperationResult {
  operationId: string;
  status: PushFieldOperationStatus;
  authoritativeEntityId: string | null;
  authoritativeRevision: number | null;
  errorCode: string | null;
  conflict: AuthorizedConflictDescriptor | null;
  acknowledgedAt: Instant;
}

export interface SyncPullRequest {
  packageId: string;
  offlineGrantId: string;
  cursor: string | null;
  limit?: number;
}

export type AuthorizedSyncChange =
  | { kind: "checklist_response"; value: ChecklistResponseView }
  | { kind: "potential_finding"; value: PotentialFindingView }
  | { kind: "package_revoked"; packageId: string; reasonCode: string; revokedAt: Instant }
  | { kind: "tombstone"; entityType: "checklist_response" | "potential_finding"; entityId: string; revision: number };

export interface SyncPullResponse {
  changes: AuthorizedSyncChange[];
  nextCursor: string | null;
  hasMore: boolean;
  resnapshotRequired: boolean;
  projectionVersion: number;
}
```

The server computes the canonical semantic payload hash and derives the actor
from the authenticated principal. In one PostgreSQL transaction it stores the
idempotency record, domain mutation, exactly one required audit event,
authorized change-feed/server-outbox record, and full replayable response.
Retrying the same operation ID and payload returns the stored response without a
second mutation or audit event; reusing the ID with a different semantic payload
is rejected. Forbidden, invalid, and unresolved conflicts are terminal until
the user changes the command. Retryable outcomes are not persisted as terminal
idempotency results.

Pull cursors are opaque and scoped server-side to principal, organization,
package, grant, projection version, and high-water mark. Responses contain only
authorized projections, tombstones, and revocations; raw domain rows and
`Internal CAA Note` never enter conflict or pull payloads. A page and its cursor
commit atomically in IndexedDB. History expiry or incompatible projection
returns `resnapshotRequired: true` and blocks editing until a safe package
refresh completes.

## File Map

### Preserve During Migration

- Preserve `index.html`, `css/styles.css`, `js/*.js`, and current `tests/*.test.js` as the legacy demo and parity oracle.
- Modify legacy files only when a separately approved defect fix is required; React migration tasks must not quietly change legacy behavior.

### Web Application

- Create `apps/web/package.json`: web scripts and locked dependencies.
- Create `apps/web/vite.config.ts`: build-profile entry/public-directory selection, Vite, test, alias, and PWA integration.
- Create `apps/web/tsconfig.json`, `apps/web/tsconfig.app.json`, `apps/web/tsconfig.node.json`.
- Create `apps/web/index.html`.
- Create `apps/web/src/entry/demo.tsx`, `apps/web/src/entry/http.tsx`, and `apps/web/src/app/bootstrap.tsx`.
- Create `apps/web/src/app/public-http-config.ts` with an allowlist of public, non-secret HTTP configuration.
- Create `apps/web/src/app/router.tsx` and `apps/web/src/app/providers.tsx`.
- Create `apps/web/src/backend/backend.ts`, `http-backend.ts`, `transport-mappers.ts`, and `backend-contracts.ts`.
- Create `apps/web/src/mock/create-mock-backend.ts`, `mock-engine.ts`, `memory-mock-store.ts`, and `seed-data.ts`; defer `indexeddb-mock-store.ts` until a later approved demo-persistence slice.
- Create `apps/web/src/offline/db.ts`, `field-repository.ts`, `outbox.ts`, `sync-engine.ts`, `storage-readiness.ts`, `opfs-inspection-attachment-store.ts`, `attachment-recovery.ts`, and `schema-migrations.ts`.
- Create feature folders under `apps/web/src/features/` for `auth`, `assignments`, `planning`, `inspections`, `checklists`, `findings`, `caps`, `evidence`, `reports`, `organizations`, `governance`, `auditee`, and `admin`.
- Create `apps/web/src/styles/legacy-reference.css` only as a temporary import boundary for verified styling; new reusable tokens/components live under `apps/web/src/ui/`.
- Create `apps/web/src/sw.ts` for app-shell/update behavior; do not put domain conflict logic in the Service Worker.
- Create `apps/web/public/demo/` and `apps/web/public/http/`; each build copies only its selected directory.
- Create `apps/web/scripts/assert-http-artifact.mjs` to reject mock/seed source inputs and demo artifacts through the Vite manifest plus output inventory.

### API Contract And Generation

- Create `docs/product-specs/data-and-rules/PRODUCTION_CONTRACT_VOCABULARY.md` and matching `.turkce.md` companion with approved source/legacy/OpenAPI/UI mappings.
- Create `api/openapi/aviasurveil360.yaml`.
- Create `api/openapi/examples/` with canonical mock and API response examples.
- Create `scripts/generate-contracts.sh` with lock-pinned TypeScript and Go generator commands.
- Create `scripts/check-contracts.sh` to lint OpenAPI, validate canonical examples, regenerate into a temporary directory, and fail on a generated diff.
- Create `apps/web/src/generated/transport/` and `apps/api/internal/httpapi/generated/`; generated files must not be hand-edited.

### Go Application

- Create `apps/api/go.mod`, `apps/api/go.sum`, `apps/api/cmd/api/main.go`, and `apps/api/cmd/worker/main.go` in one Go module.
- Create `apps/api/internal/platform/` packages for configuration, PostgreSQL, object storage, clock, IDs, auth principal, transactions, and audit events.
- Create domain packages under `apps/api/internal/` for `identity`, `organizations`, `planning`, `inspections`, `checklists`, `potentialfindings`, `findings`, `caps`, `evidence`, `reports`, `sync`, `worker`, and `auditlog`.
- Create `apps/api/migrations/` with ordered SQL migrations.
- Create module-owned PostgreSQL query/store packages under `apps/api/internal/<module>/store/postgres/`; `apps/api/sqlc.yaml` may generate only into those owned packages.
- Limit the first worker slice to Evidence upload reconciliation and malware-scan orchestration. Notification delivery, production PDF generation, and retention execution require later approved slices.
- Create `deploy/local/compose.test.yaml` with pinned PostgreSQL and S3-compatible test dependencies, health checks, bucket initialization, and isolated test volumes.
- Create `scripts/test-http-profile.sh` to build API/worker, reset and seed dependencies, start the deterministic test identity/profile, run Go plus shared HTTP browser scenarios, reject skips/zero tests, and tear down task-owned processes and containers.

### Tests

- Create `apps/web/src/**/*.test.ts(x)` for unit/component tests.
- Create `apps/web/tests/contract/` with one parameterized backend contract suite used by `MockBackend` configured with `MemoryMockStore` and by seeded `HttpBackend`.
- Create `apps/web/tests/offline/` for storage, outbox, recovery, and conflict tests.
- Create `apps/web/tests/e2e/` with the same scenario files executed by required Playwright `mock` and `http` projects.
- Create `apps/api/internal/**/*_test.go` for domain, authorization, state-machine, and handler tests.
- Create `apps/api/tests/integration/` for PostgreSQL, object storage, sync, and migration tests.
- Create `tests/parity/behavior-ledger.json` and `tests/parity/react-legacy-parity.test.mjs` for role/route/action/state/visibility comparison and accepted differences.

### Documentation And Tracking

- Modify `README.md`, `README.turkce.md`, `MANIFEST.md`, and `docs/index.md` only when the React/Go artifacts actually exist.
- Modify `docs/agent-harness/verification-matrix.md` during Phase 0 to add a production-application lane while retaining the existing static-demo lane and local-only default unless the user separately approves remote CI.
- Modify matching product/workflow docs only when implementation changes or concretizes their behavior.
- Create bilingual production-transition evidence under `docs/demo-evidence/` only after fresh verification.
- Update this plan, `docs/exec-plans/index.md`, `docs/exec-plans/completed/index.md`, and `docs/exec-plans/tech-debt-tracker.md` as the plan lifecycle changes.

## Delivery Slices

This is an umbrella plan. Each slice must produce working, reviewable software and may be executed only after explicit user approval for that slice.

The binding execution order is the slice order below, even though historical task
numbers are retained for stable references:

1. **Owner/contract gate — Task 1:** Record the required owner values, approve the stack/auth/offline ADRs, add the production-application harness lane, and approve only the next slice.
2. **First executable slice — Tasks 2-4:** Freeze the bilingual vocabulary and minimal OpenAPI contract first; create build-time-separated React entries, one capability-composed `Backend`, deterministic `MockBackend` backed by `MemoryMockStore`, and the canonical React mock scenario. Do not add IndexedDB mock persistence, Service Worker, OPFS, Go domain behavior, real auth/upload, secondary route families, or production claims.
3. **Real HTTP vertical slice — Tasks 9-11:** Build the API and worker in one Go module, PostgreSQL migrations and domain rules, same-origin auth boundary, deterministic local integration profile, bounded whole-object Evidence upload/scan, and the exact canonical HTTP scenario. Run the parameterized backend contract and same Playwright scenario files against mock and HTTP before broad React migration.
4. **Browser offline foundation — Tasks 6-8:** Add the version-fenced PWA shell, conservative readiness/check-out gate, server-issued offline grant, IndexedDB local working set/outbox, and manifest-first OPFS `InspectionAttachment` staging/recovery.
5. **Typed production sync — Task 12:** Connect the local field client through one-operation causal sync, exact idempotency replay, authorized pull projections, explicit conflicts, and foreground fallback.
6. **Incremental React completion — Task 5:** Port only owner-classified `first-production` route families, one family at a time through both mock and HTTP profiles. Keep `later` and `demo-only` routes in the legacy demo.
7. **Local release-candidate verification — Task 13:** Run the mock/HTTP/offline/security/restore/local operational matrices, reconcile evidence, and request explicit acceptance of a local release candidate. Production deployment/cutover remains blocked on a separately approved release/operations plan.

The minimum first executable slice is complete only when the approved vocabulary
and OpenAPI examples validate, the canonical React mock scenario passes, both
build entries type-check, the HTTP artifact manifest contains no mock/seed inputs
or demo public files, the behavior ledger passes for the migrated scenario, and
the root Vanilla JavaScript demo remains unchanged and runnable. This evidence is
`candidate-only` and must not be reported as real API, offline, or production
evidence.

## Phases And Tasks

## Phase 0 — Architecture Review And Owner Gates

### Task 1: Close Review Findings And Approve The Next Slice

**Files:**

- Modify: this plan's Decision Log, status, next todo, and slice gates.
- Modify: `docs/exec-plans/index.md` and `docs/exec-plans/tech-debt-tracker.md`.
- Modify: `docs/agent-harness/verification-matrix.md` after the user approves a production-application verification lane.
- Create: `docs/product-specs/data-and-rules/PRODUCTION_CONTRACT_VOCABULARY.md` and `PRODUCTION_CONTRACT_VOCABULARY.turkce.md` in Task 2 after Product/CAA Operations approve the mapping.
- Test: docs-only checks in this task.

**Interfaces:**

- Consumes: the 2026-07-20 `NO-GO as written` review, current demo/product sources, stakeholder browser-only constraint, and the corrected candidate architecture in this plan.
- Produces: named decisions, a reconciled harness, an explicit GO/NO-GO for one selected slice, and accurate durable blockers.

- [x] **Step 1: Complete the adversarial architecture review with `NOEDIT`.**

  Result: completed 2026-07-20. Verdict was `NO-GO as written` with the React/Vite, one-Backend, field-only offline, Go modular-monolith, PostgreSQL, S3-compatible storage, and transactional-outbox directions retained conditionally. Production-readiness was not claimed.

- [x] **Step 2: Record named owners and accepted values in the Decision Log.**

  Slice reconciliation on 2026-07-20 records the current user / plan owner as
  the explicit authorizer for Tasks 2-4 only. The canonical product docs and
  verified legacy demo are accepted as the slice-local authority for vocabulary,
  the canonical Cabin scenario, the eight role-entry routes, contract generation,
  and build-time demo/HTTP separation. This does not resolve or accept any
  Security, Identity, Records, Platform, Operations, production-hosting, Go,
  offline, upload, sync, release, or cutover decision below.

  Record explicit decisions for:

  ```text
  Product/CAA Operations: first-production/later/demo-only route inventory
  Product/CAA Operations: canonical source/legacy/OpenAPI/UI status and action vocabulary
  Engineering/Platform: React/Vite and Go modular-monolith ownership ADR
  Security: same-origin OIDC session/BFF approval, provider, MFA, cookie, CSRF, session expiry, and revocation
  Security/CAA Operations: supported managed browsers, devices, profiles, and clear-on-exit policy
  Security/Records: browser-local data classification, OS/profile controls or app encryption, key custody, and recovery
  Product/Security: offline grant duration, package duration/size, reassignment, revocation, clock skew, late sync, logout, and user-switch behavior
  Product/CAA Operations: conflict policy and maximum acceptable unsynced-work exposure
  Product/Records: InspectionAttachment versus official Evidence boundary
  Records/Security: maximum Evidence byte size, approved media types, bounded retry versus multipart, checksum, quarantine, and scan policy
  Records/Legal/Standards: retention, legal hold, disposition, chain-of-custody wording, regulatory wording, authorized closure, report approval/versioning, and audit tamper-evidence requirements
  Platform/Operations: PostgreSQL/object-store/scanner providers, hosting region, RPO, RTO, backup, restore, disaster recovery, observability, and on-call owners
  QA/Product: browser/device matrix, parity severity threshold, pilot cohort, rollback trigger, and cutover decision owner
  ```

  Expected: each item has a named human owner and accepted value/policy. An unresolved item remains `note-open`, is marked `blocked` in the Decision Log, and blocks only the dependent slice. Task 2 may proceed only when vocabulary, route scope, generator/build-profile ownership, and harness decisions are accepted.

- [x] **Step 3: Add separate static-demo and production-application harness lanes.**

  Preserve all existing static-demo commands and local-only defaults. Add locked install, OpenAPI generation/diff, React type/unit/build, Go build/race/integration, Playwright mock/HTTP/offline, skip-count, artifact-scan, process cleanup, and restore-drill requirements for future application artifacts. Do not add hosted CI or a remote workflow without separate user approval.

- [x] **Step 4: Approve exactly one next slice.**

  Result: `GO` for Tasks 2-4 only, explicitly authorized by the current user on
  2026-07-20. The approved route scope is the complete canonical Cabin scenario
  plus the eight verified legacy role-entry routes. Tasks 5-13, Go, real HTTP,
  offline storage/PWA/sync, deployment, cutover, and legacy removal are not
  authorized.

  Follow-up 2026-07-21: the current user / plan owner explicitly authorized
  Tasks 5-13 for local release-candidate execution in binding slice order, with
  a separate commit and push after each Task. This follow-up supersedes only the
  earlier implementation-slice limit. It does not authorize production
  deployment, traffic cutover, legacy removal, or a `production-ready` claim.

  Record `GO`, `CONDITIONAL GO`, or `NO-GO` for the selected slice and list every condition. Approval of Task 2-4 does not authorize Go, offline, deployment, release, or route-family expansion.

- [x] **Step 5: Run docs-only verification.**

  Run:

  ```bash
  git diff --check
  node tests/harness-docs-smoke.test.js
  rg -n "MemoryMockStore|MockBackend|HttpBackend|OfflineGrant|InspectionAttachment|Local Release-Candidate Gate|Execution Prompt" docs/exec-plans docs/agent-harness MANIFEST.md
  ```

  Expected: exit `0`; plan/index/tracker/harness links and statuses agree; no implementation file exists yet.

  Result 2026-07-20: `git diff --check`,
  `node tests/harness-docs-smoke.test.js`, and the required `rg` contract-term
  search passed before Task 2 implementation. No React/API implementation file
  existed at this checkpoint.

## Phase 1 — Freeze Legacy Behavior And Establish The React Foundation

### Task 2: Freeze Vocabulary And OpenAPI Before Creating The React Workspace

**Files:**

- Create: `docs/product-specs/data-and-rules/PRODUCTION_CONTRACT_VOCABULARY.md`
- Create: `docs/product-specs/data-and-rules/PRODUCTION_CONTRACT_VOCABULARY.turkce.md`
- Create: `tests/parity/behavior-ledger.json`
- Create: `tests/parity/react-legacy-parity.test.mjs`
- Create: `api/openapi/aviasurveil360.yaml`
- Create: `api/openapi/examples/canonical/*.json`
- Create: `api/openapi/tests/contract-examples.test.mjs`
- Create: `scripts/generate-contracts.sh`
- Create: `scripts/check-contracts.sh`
- Create: `apps/web/package.json`
- Create: `apps/web/package-lock.json`
- Create: `apps/web/vite.config.ts`
- Create: `apps/web/tsconfig.json`
- Create: `apps/web/tsconfig.app.json`
- Create: `apps/web/tsconfig.node.json`
- Create: `apps/web/index.html`
- Create: `apps/web/src/entry/demo.tsx`
- Create: `apps/web/src/entry/http.tsx`
- Create: `apps/web/src/app/bootstrap.tsx`
- Create: `apps/web/src/app/public-http-config.ts`
- Create: `apps/web/src/app/router.tsx`
- Create: `apps/web/src/app/providers.tsx`
- Create: `apps/web/src/generated/transport/api-types.ts`
- Test: `apps/web/src/app/public-http-config.test.ts`

**Interfaces:**

- Consumes: approved Task 1 owner decisions, current `ROLE_ORDER`, route/view inventory, lifecycle smoke tests, canonical product workflows, and local Node/npm baseline.
- Produces: approved canonical vocabulary, minimal versioned OpenAPI, canonical JSON examples, generated TypeScript transport types, a classified behavior ledger, and a bootable backend-neutral React shell.

- [x] **Step 1: Write the failing vocabulary, OpenAPI-example, and behavior-ledger tests.**

  `behavior-ledger.json` records `id`, `classification`, `role`, `route`, `action`, `entityIdRule`, `expectedStatus`, `expectedOwner`, `visibilityInvariant`, `legacyTest`, `reactTest`, and `acceptedDifference`. The initial ledger contains only the canonical scenario plus the eight role-entry routes; each entry fails if its referenced test is missing.

  ```js
  assert.equal(canonicalScenario.classification, 'first-production');
  assert.equal(canonicalScenario.expectedStatus, 'CLOSED');
  assert.equal(canonicalScenario.visibilityInvariant, 'auditee-never-receives-internal-caa-note');
  assert.equal(ledger.legacyRemovalAllowed, false);
  ```

  The OpenAPI test validates every example against its request/response schema and rejects `additionalProperties` on Auditee projections, untyped sync payloads, and lifecycle enum values missing from the approved vocabulary.

- [x] **Step 2: Run the parity test and verify the expected failure.**

  Run:

  ```bash
  node --test api/openapi/tests/contract-examples.test.mjs tests/parity/react-legacy-parity.test.mjs
  ```

  Expected: FAIL because the vocabulary, OpenAPI, examples, and behavior ledger do not exist.

- [x] **Step 3: Write the bilingual vocabulary and minimal OpenAPI contract.**

  The English document is canonical and maps source label -> legacy value -> OpenAPI enum -> UI label. The Turkish companion is stakeholder-facing. The first OpenAPI surface is:

  ```text
  GET  /health/live
  GET  /health/ready
  GET  /v1/assignments
  GET  /v1/inspection-packages/{id}
  POST /v1/inspection-packages/{id}/checkout
  PUT  /v1/checklist-responses/{responseId}
  POST /v1/checklists/{auditId}/submit
  POST /v1/checklists/{auditId}/reopen
  POST /v1/potential-findings
  POST /v1/potential-findings/{id}/decisions
  GET  /v1/findings
  GET  /v1/findings/{id}
  POST /v1/findings/{id}/authorized-closure
  POST /v1/caps
  POST /v1/caps/{capRevisionId}/reviews
  POST /v1/inspection-attachments/{id}/uploads
  POST /v1/inspection-attachments/uploads/{uploadId}/complete
  POST /v1/evidence/uploads
  POST /v1/evidence/uploads/{uploadId}/complete
  POST /v1/evidence/{evidenceVersionId}/reviews
  GET  /v1/report-versions/{id}
  POST /v1/report-versions/{id}/decisions
  GET  /v1/dashboards/manager
  POST /v1/sync/operations
  GET  /v1/sync/changes
  ```

  Every mutating operation includes `operationId`; every decision names its expected entity revisions. Auditee schemas structurally omit internal CAA fields. Sync uses a discriminated command/change union and typed conflict descriptor.

- [x] **Step 4: Add lock-pinned contract generation and validation.**

  `scripts/generate-contracts.sh` generates TypeScript transport types from `api/openapi/aviasurveil360.yaml`. `scripts/check-contracts.sh` lints the document, validates examples, regenerates into a temporary directory, and fails when checked generated output differs. Task 9 extends the same command with Go handler generation after the Go tool lock exists.

- [x] **Step 5: Initialize `apps/web` and lock dependencies.**

  Install React, Vite, TypeScript, router, query, form, validation, Dexie, contract validation, Vitest, React Testing Library, and Playwright dependencies; record exact resolved versions in `apps/web/package-lock.json`. Do not use Create React App.

  Expected scripts:

  ```json
  {
    "scripts": {
      "dev:demo": "AVIA_BUILD_PROFILE=demo vite",
      "dev:http": "AVIA_BUILD_PROFILE=http vite",
      "build:demo": "tsc -b && AVIA_BUILD_PROFILE=demo vite build",
      "build:http": "tsc -b && AVIA_BUILD_PROFILE=http vite build",
      "typecheck": "tsc -b --pretty false",
      "test": "vitest run",
      "test:e2e:mock": "playwright test --project=mock",
      "test:e2e:http": "playwright test --project=http",
      "contracts:check": "../../scripts/check-contracts.sh"
    }
  }
  ```

- [x] **Step 6: Implement the backend-neutral React shell and build-profile selection.**

  ```ts
  export interface PublicHttpConfig {
    apiBaseUrl: string;
    environmentLabel: string;
  }

  export type BuildProfile = "demo" | "http";
  ```

  `vite.config.ts` selects `src/entry/demo.tsx` plus `public/demo/` or `src/entry/http.tsx` plus `public/http/` at build time. `PublicHttpConfig` accepts only the two fields above; unknown keys fail validation so secrets cannot be smuggled into public config.

- [x] **Step 7: Implement the role-aware shell and checked behavior ledger.**

  Expected: all eight roles exist; only ledger-classified routes have React placeholders; the shell imports no legacy globals. `later` and `demo-only` routes explicitly remain on the legacy surface.

- [x] **Step 8: Run contract/foundation verification.**

  Run:

  ```bash
  npm --prefix apps/web ci
  npm --prefix apps/web run contracts:check
  npm --prefix apps/web run typecheck
  npm --prefix apps/web test
  npm --prefix apps/web run build:demo
  npm --prefix apps/web run build:http
  node --test api/openapi/tests/contract-examples.test.mjs tests/parity/react-legacy-parity.test.mjs
  ```

  Expected: all commands pass; generated output is clean; both shells build; no production behavior is claimed; legacy demo remains unchanged.

  Result 2026-07-20: the expected red contract/parity run failed on the
  absent vocabulary, OpenAPI, examples, and ledger. After implementation,
  `npm ci`, contract lint/example/regeneration diff, TypeScript, 11 Vitest
  assertions, demo build, HTTP shell build, and 7 Node contract/parity
  assertions passed. The root Vanilla demo was not modified. Evidence is
  `verified locally` and `candidate-only`; real HTTP and production behavior
  are `not run`.

## Phase 2 — Establish Mock And Real Backend Modes

### Task 3: Implement One Backend Contract With Mock And HTTP Backends

**Files:**

- Create: `apps/web/src/backend/backend.ts`
- Create: `apps/web/src/backend/backend-contracts.ts`
- Create: `apps/web/src/backend/http-backend.ts`
- Create: `apps/web/src/backend/transport-mappers.ts`
- Create: `apps/web/src/mock/create-mock-backend.ts`
- Create: `apps/web/src/mock/mock-engine.ts`
- Create: `apps/web/src/mock/memory-mock-store.ts`
- Create: `apps/web/src/mock/seed-data.ts`
- Create: `apps/web/scripts/assert-http-artifact.mjs`
- Create: `apps/web/tests/contract/backend-contract.ts`
- Create: `apps/web/tests/contract/mock-backend.test.ts`
- Test: `apps/web/src/backend/http-backend.test.ts`
- Test: `apps/web/src/backend/transport-mappers.test.ts`
- Test: `apps/web/src/app/build-profile.test.ts`

**Interfaces:**

- Consumes: approved OpenAPI/generated transport, `PublicHttpConfig`, behavior ledger, and canonical demo records/status rules from Task 2.
- Produces: capability-composed `Backend`, `createMockBackend`, `createHttpBackend`, build-time-separated entries, domain/transport mappers, and one reusable backend conformance suite.

- [x] **Step 1: Write the failing backend contract suite.**

  It must assert stable IDs, exact Audit/question scope, Potential Finding authority, organization isolation, separate CAP submission/review, Evidence version preservation, report/version authority, direct-command idempotency, and sync acknowledgement:

  ```ts
  export function backendContract(createBackend: () => Promise<Backend>) {
    it("does not turn CAP submission into CAA acceptance", async () => {
      const backend = await createBackend();
      const submitted = await backend.caps.submit(canonicalCapSubmission);
      expect(submitted.capStatus).toBe("SUBMITTED");
      expect(submitted.findingStatus).toBe("CAP_SUBMITTED");
    });

    it("keeps the Finding open after authorized CAP acceptance", async () => {
      const backend = await createBackend();
      const accepted = await backend.caps.review(canonicalCapAcceptance);
      expect(accepted.capStatus).toBe("ACCEPTED");
      expect(accepted.findingStatus).toBe("EVIDENCE_REQUIRED");
    });

    it("requires Lead conversion before a canonical Finding exists", async () => {
      const backend = await createBackend();
      const potential = await backend.potentialFindings.create(canonicalPotentialFinding);
      expect(potential.status).toBe("PENDING_LEAD_REVIEW");
      expect(potential.convertedFindingId).toBeNull();
    });
  }
  ```

- [x] **Step 2: Run the contract suite and verify the expected failure.**

  Run:

  ```bash
  npm --prefix apps/web test -- tests/contract/mock-backend.test.ts
  ```

  Expected: FAIL because `MockBackend` does not exist.

- [x] **Step 3: Implement `MockBackend` with a pluggable mock store.**

  The first slice uses only `MemoryMockStore` with an injected clock and ID generator. It simulates remote canonical state and is accessible only through `MockBackend`. Do not add network interception, IndexedDB mock-server persistence, a demo-only Service Worker, component-owned canonical mutations, or a second repository family.

  Implement Potential Finding return/dismiss/convert separately from canonical Findings. `submitCap` yields `CAP_SUBMITTED`; `reviewCap(ACCEPT)` may yield `EVIDENCE_REQUIRED` but never `CLOSED`. Evidence review targets an immutable version. Reusing an operation ID with the same semantic payload returns the stored response; changing the payload rejects reuse.

- [x] **Step 4: Implement `HttpBackend` as a thin typed HTTP client.**

  Requirements:

  ```text
  AbortSignal support
  generated OpenAPI transport types
  explicit transport-to-domain mapping
  JSON content-type validation
  structured problem responses
  correlation/request ID propagation
  same-origin credentials and CSRF token injection
  operation ID on every mutation
  no hidden automatic replay of commands
  no local authorization decisions
  401 / 403 / session-expiry behavior
  ```

  Unit tests use a deterministic fake `fetch` response; they do not claim a real API exists. No mock fixture may be imported into `http-backend.ts` or its build entry.

- [x] **Step 5: Implement build-time-separated demo and HTTP entries.**

  `demo.tsx` imports `createMockBackend`; `http.tsx` imports only `createHttpBackend`. Vite selects exactly one entry and public directory at build time. Runtime configuration cannot switch backend families.

- [x] **Step 6: Verify the HTTP artifact excludes mock code and data.**

  `assert-http-artifact.mjs` reads the Vite manifest and output inventory. It fails when an input path starts with `src/mock/`, when a demo public/config file is present, or when the HTTP entry has a transitive mock/seed module. Seed-name string search is supplemental only.

- [x] **Step 7: Run backend-boundary verification.**

  Run:

  ```bash
  npm --prefix apps/web run typecheck
  npm --prefix apps/web test
  npm --prefix apps/web run build:demo
  npm --prefix apps/web run build:http
  node apps/web/scripts/assert-http-artifact.mjs apps/web/dist
  ```

  Expected: all tests pass; both adapters satisfy the same TypeScript/domain contract; mock conformance executes; HTTP mapper tests execute; the HTTP artifact contains no mock/seed inputs. Real HTTP conformance remains `not run` until Tasks 9-11.

  Result 2026-07-20: the focused mock conformance run first failed because
  `createMockBackend` did not exist. After implementation, TypeScript, 30
  Vitest assertions (including 9 reusable backend-contract cases), demo/HTTP
  builds, and the HTTP artifact scan passed. The scan found 57 HTTP module
  inputs and no mock/seed source or demo-public artifact. `HttpBackend` evidence
  is deterministic fake-fetch mapping only; real HTTP conformance is `not run`.

## Phase 3 — Port The Canonical Scenario Before Secondary Screens

### Task 4: Port The Cabin Inspection Vertical Slice

**Files:**

- Create: `apps/web/src/features/assignments/`
- Create: `apps/web/src/features/inspections/`
- Create: `apps/web/src/features/checklists/`
- Create: `apps/web/src/features/findings/`
- Create: `apps/web/src/features/caps/`
- Create: `apps/web/src/features/evidence/`
- Create: `apps/web/src/features/reports/`
- Create: `apps/web/tests/e2e/canonical-scenario.spec.ts`
- Create: `apps/web/tests/e2e/support/scenario-transcript.ts`
- Modify: `tests/parity/behavior-ledger.json`
- Modify: `tests/parity/react-legacy-parity.test.mjs`

**Interfaces:**

- Consumes: `Backend` from Task 3 and canonical workflow/product rules.
- Produces: React routes/actions and a normalized invariant transcript for the complete Cabin scenario in mock mode; the same scenario file is reused by the HTTP project in Task 11.

- [x] **Step 1: Write the failing Playwright canonical scenario.**

  The test must enter the assigned Inspector role, answer the exact Audit/question, record the required comment, create an audit-scoped Potential Finding, switch explicitly to Lead Inspector, convert it with severity into the canonical Finding, submit CAP as the organization-scoped Auditee, review and accept CAP as CAA without closure, submit a mock Evidence filename/version, review the exact latest version, close only through `Close`, and observe the manager projection/report preview update.

  Additional assertions prove another Inspector's question is read-only, CAP submission remains `CAP_SUBMITTED`, `Partially Close`/`Not Close` keep the Finding open, authorized closure remains a separate reason-required Department Manager path, report issue never closes Findings, and the Auditee never receives another organization's or internal CAA data.

- [x] **Step 2: Run the scenario and verify the expected failure.**

  Run:

  ```bash
  npm --prefix apps/web run test:e2e:mock -- canonical-scenario.spec.ts
  ```

  Expected: FAIL at the first unported route.

- [x] **Step 3: Port routes feature by feature without importing legacy state.**

  Each visible Finding view must render owner, next action, Due Date, status, severity, related Audit, and organization from typed backend/local records.

  No React component mutates canonical mock state directly. All remote-shaped commands and queries use the injected `Backend`; Task 4 does not create `FieldRepository` or offline state.

- [x] **Step 4: Port comment and visibility boundaries.**

  Add component and backend-object tests proving `Internal CAA Note`, workload, internal risk, other organizations, unreleased reports, and enforcement deliberations are absent from every Auditee projection before render. `Comment to Auditee` remains a distinct required field.

- [x] **Step 5: Verify the vertical slice and legacy parity.**

  Run typecheck, unit/component tests, `canonical-scenario.spec.ts` in the mock Playwright project, the relevant legacy Node smoke tests, contract checks, HTTP-artifact scan, and behavior-ledger parity.

  Expected: the canonical React mock scenario and normalized transcript pass; the legacy canonical scenario continues to pass; all evidence is `candidate-only`; real API, offline, and production evidence remain `not run`.

  Result 2026-07-20: the first mock Playwright run failed at the first
  unported `/inspector/inspector-assignments` route. The completed browser
  scenario then passed 1/1 with no page or console warning/error and recorded
  the normalized Audit -> checklist -> Potential Finding -> Finding -> CAP ->
  three immutable Evidence versions -> Evidence-verified closure -> locked
  report -> manager/Auditee projection transcript. CAP acceptance, partial
  close, not close, reason-required authorized closure, and report issue all
  preserved their separate authority boundaries. The focused Auditee
  component/backend-object boundary passed 11/11 assertions; the complete
  Vitest suite passed 32/32 across 8 files; contract generation/check and the
  7 OpenAPI/ledger Node assertions passed; demo and HTTP builds passed; the
  HTTP artifact scan passed for 7 files and 71 module inputs with no mock/seed
  source or demo-public artifact; and the intact legacy suite passed 103/103.
  Evidence is `verified locally` and `candidate-only`. Real HTTP, Go, PWA,
  IndexedDB, OPFS, sync, deployment, cutover, and production behavior are
  `not run`.

### Task 5: Port Approved First-Production Route Families Incrementally

**Files:**

- Create/Modify: owner-approved `first-production` feature folders under `apps/web/src/features/`.
- Modify: `api/openapi/aviasurveil360.yaml`, canonical examples, generated transports, `MockBackend`, Go handlers/domain stores, and `behavior-ledger.json` for each accepted route family.
- Create: shared route-family Playwright files under `apps/web/tests/e2e/` executed by both projects.
- Modify: `tests/parity/react-legacy-parity.test.mjs`.

**Interfaces:**

- Consumes: the proven mock/HTTP canonical vertical from Tasks 4 and 9-12, shared shell, `Backend`, design tokens, and approved route classification.
- Produces: mock/HTTP/React parity for one accepted route family per reviewable increment; `later` and `demo-only` routes remain safely available in the legacy demo.

- [x] **Step 1: Confirm the approved route classification before each increment.**

  The owner-reviewed inventory uses only:

  ```text
  first-production — required for the accepted production slice
  later            — retained in legacy until a later approved slice
  demo-only        — stakeholder concept; not a production commitment
  ```

  The Core MVP families from `MVP_SCOPE_AND_ROADMAP.md` are candidates, not automatic approval. AI, advanced risk/BI, broad regulatory editing, USOAP/SSP expansion, and generic workflow surfaces remain `later` or `demo-only` unless Product changes the classification explicitly.

- [x] **Step 2: Write failing behavior-ledger, OpenAPI, backend-contract, and browser assertions for one route family.**

  Every action records role, exact entity identity, expected revisions/status/owner, visibility boundary, mock test, HTTP test, React test, legacy test, and accepted difference. A route-name entry alone cannot satisfy parity.

- [x] **Step 3: Extend contract, mock, Go, and React together.**

  Update OpenAPI/examples first, regenerate transport/handler code, implement both backends and server authorization, then port the React route. Do not port the next route family until the current family passes both backend profiles.

- [x] **Step 4: Verify all visible behavior.**

  Do not close a family while a visible control is inert, silently cycles values, or substitutes a toast for the labeled behavior.

- [x] **Step 5: Run the accepted React route/role matrix at desktop, tablet, and mobile in both profiles.**

  Expected: meaningful route content, identical invariant transcripts, zero route/role/organization mismatches, no raw Auditee forbidden fields, no critical horizontal overflow, and no unexpected console warnings/errors.

- [x] **Step 6: Keep the legacy demo and accepted-differences ledger available.**

  Expected: no removal or overwrite of root demo assets; `later`/`demo-only` routes stay reachable there; cutover remains blocked on the separately approved runbook and stakeholder acceptance.

  Result 2026-07-21: red contract and ledger tests first failed on the absent organization route, behavior-ledger v2, and missing organization/planning Backend capabilities. The first complete HTTP run then caught two unexpected 404 console records per viewport because the Department Manager dashboard requested a canonical Finding before the scenario had created it; the dashboard now consumes the server-shaped list projection and selects that Finding only when present. The completed OpenAPI/Backend/Go/React increment adds an organization registry, Audit Plan Calendar, exact Finance -> GM -> Executive Director -> GM plan authority chain, read-only versioned checklist/reminder configuration, and an Admin planning Audit Trail. Planning decisions are reason-required, idempotent, revision- and role/stage-checked, and transactionally record the update, audit event, idempotency result, and outbox message. The behavior ledger is version 3 with 15 executable entries. The fresh full HTTP profile passed API/worker builds, the full Go race/live PostgreSQL/OIDC-provider/MinIO suite including migration v6 and retained N-1 upgrade, OpenAPI 6/6, clean SQLC generation, TypeScript, React/Vitest 146/146, live Backend contract 11/11, mock Playwright 4/4, HTTP Playwright 5/5, both builds, HTTP mock/seed isolation across 12 files and 89 inputs, and cleanup. The first-production matrix passed mock 3/3 and HTTP 3/3 at desktop/tablet/mobile with no unexpected console warnings/errors or critical horizontal overflow. `go vet ./...`, the intact root plus parity suite 106/106, `git diff --check`, and task-owned process/container cleanup also passed. Evidence is [First-Production Route Families](../../demo-evidence/FIRST_PRODUCTION_ROUTE_FAMILIES_2026-07-21.md), `verified locally`, and `candidate-only`. Task 13 subsequently completed the local release-candidate packet; release remains `release pending`, while production deployment, cutover, legacy removal, and `production-ready` evidence remain `blocked`.

## Phase 4 — Add The Browser Offline-First Field Foundation

### Task 6: Implement PWA App Shell And Offline Readiness

**Files:**

- Create: `apps/web/src/sw.ts`
- Create: `apps/web/src/offline/storage-readiness.ts`
- Create: `apps/web/src/offline/update-coordinator.ts`
- Create: `apps/web/src/features/inspections/offline-readiness-panel.tsx`
- Test: `apps/web/src/offline/storage-readiness.test.ts`
- Test: `apps/web/src/offline/update-coordinator.test.ts`
- Create: `apps/web/tests/e2e/offline-startup.spec.ts`
- Create: `apps/web/tests/e2e/offline-update-recovery.spec.ts`

**Interfaces:**

- Consumes: proven HTTP vertical, server-issued `OfflineGrant`, React shell, and approved managed browser/offline/local-data policy.
- Produces: `assessOfflineReadiness()`, version-fenced app-shell caching/update behavior, and an explicit online-only fallback.

- [x] **Step 1: Write failing readiness tests for all result states.**

  ```ts
  export type OfflineReadinessCode =
    | "ready"
    | "unsupported-browser"
    | "managed-policy-unapproved"
    | "ephemeral-or-unmanaged-storage"
    | "service-worker-unavailable"
    | "indexeddb-health-failed"
    | "opfs-health-failed"
    | "persistence-denied"
    | "quota-insufficient"
    | "offline-grant-invalid"
    | "app-version-incompatible"
    | "schema-version-incompatible"
    | "protocol-version-incompatible";
  ```

- [x] **Step 2: Implement the explicit check-out gate.**

  The gate checks secure context, approved browser/device/profile policy, Service Worker readiness, separate IndexedDB and OPFS write/read/hash/delete canaries, `navigator.storage.persisted()`, user-initiated `persist()`, advisory `estimate()`, package/attachment estimate plus owner-approved conservative headroom, a successful browser-restart canary, app/schema/protocol compatibility, and the server-issued grant. It does not claim reserved capacity or infer private browsing.

  A denial blocks checkout with a specific recovery action while preserving online use. After explicit site-data deletion, the server's outstanding checkout record may report `local package missing`; the UI must state that unsynced single-device work cannot be recovered.

- [x] **Step 3: Implement app-shell caching without business-record response caching.**

  Cache only versioned build assets and the offline shell. Do not apply generic stale-while-revalidate to authenticated API responses.

- [x] **Step 4: Implement update safety.**

  Version app shell, IndexedDB schema, package schema, and sync protocol independently with an explicit N/N-1 compatibility range. Do not automatically call `skipWaiting`, claim clients, or delete an old cache while incompatible tabs or unsynced packages exist. Use an approved single migration/update owner across tabs, pause edits during an incompatible migration, use expand/contract schemas, and preserve a read-only recovery path if migration fails.

- [x] **Step 5: Verify actual offline startup.**

  Serve over localhost/HTTPS-equivalent secure context, visit once online, check out a package, stop network/server access, reload, and verify the field shell/package still opens. Do not use DevTools page-only offline mode as the sole evidence.

- [x] **Step 6: Verify multi-client update and rollback safety.**

  Test two tabs on N/N-1 shells, termination at each migration boundary, update with pending outbox and OPFS manifests, rollback to the previous shell without a database downgrade, persistence denial, quota/headroom rejection, unsupported/managed-policy rejection, and explicit site-data clearing. Expected: no silent edit under an incompatible schema and no deletion of pending local bytes.

  Result 2026-07-21: readiness, Service Worker policy, update coordinator, UI, and browser tests were written against absent behavior before implementation. The final gate covers all thirteen readiness codes, explicit user-only persistence requests, IndexedDB and OPFS health canaries, restart survival, exact offline-grant scope, positive N/N-1 versions, app-shell-only caching, migration read-only recovery, pending-work deferral, and a single cross-tab update owner. `npm --prefix apps/web run test:e2e:offline` passed 2/2 using a dedicated persistent Chrome profile: the checked-out Audit shell/package reopened in a fresh browser process after the origin server stopped, and a separate two-page update test preserved N/N-1 caches plus IndexedDB/OPFS sentinels until explicit site-data clearing. The full HTTP profile passed Go race/live integration, OpenAPI 5/5, SQLC regeneration, React 76/76, live HTTP contract 9/9, mock/HTTP Playwright 1/1 each, both builds, and cleanup. App-shell scans passed for demo and HTTP; the HTTP artifact remained mock/seed-free across 10 files and 75 inputs. Evidence is [PWA App Shell And Offline Readiness](../../demo-evidence/PWA_OFFLINE_READINESS_2026-07-21.md), `verified locally`, and `candidate-only`. Tasks 7-8, 12, and 5 subsequently completed atomic field storage/outbox, manifest-first OPFS attachment recovery, typed foreground sync, and approved route-family parity. Production MDM/security operations, deployment, cutover, and `production-ready` evidence remain `blocked`; the next binding slice is Task 13.

### Task 7: Implement Atomic IndexedDB Field Storage And Outbox

**Files:**

- Create: `apps/web/src/offline/db.ts`
- Create: `apps/web/src/offline/schema-migrations.ts`
- Create: `apps/web/src/offline/field-repository.ts`
- Create: `apps/web/src/offline/outbox.ts`
- Create: `apps/web/tests/offline/field-repository.test.ts`
- Create: `apps/web/tests/offline/restart-recovery.spec.ts`

**Interfaces:**

- Consumes: `FieldRepository`, `FieldSyncOperation`, server-issued `OfflineGrant`, and approved per-user local-data policy from the architecture section.
- Produces: subject-scoped atomic offline writes, visible pending state, typed causal outbox, atomic pull-page application, deterministic migrations, and restart recovery.

- [x] **Step 1: Write failing tests for atomic response/outbox behavior.**

  Tests must prove that a checklist response/Potential Finding/checklist submission and its typed outbox operation either both commit or neither commits; duplicate operation IDs cannot create duplicate commands. An unsent response edit may replace its unsent payload atomically, but an in-flight operation is immutable and a later edit waits for its authoritative revision.

- [x] **Step 2: Define the IndexedDB schema.**

  Required stores/indexes:

  ```text
  packages: id, subjectId, auditId, organizationId, packageVersion, schemaVersion, protocolVersion, packageDigest, checkedOutAt, expiresAt, grantId
  offlineGrants: grantId, subjectId, organizationId, packageId, packageVersion, packageDigest, deviceInstanceId, issuedAt, expiresAt, protocolVersion
  checklistResponses: id, subjectId, packageId, auditId, questionId, revision, syncState, updatedAt
  potentialFindingDrafts: id, subjectId, packageId, auditId, questionId, checklistResponseId, baseRevision, syncState, updatedAt
  attachmentManifests: attachmentId, subjectId, packageId, auditId, checklistResponseId, potentialFindingLocalId, temporaryOpfsPath, finalOpfsPath, stagingState, syncState
  outbox: operationId, subjectId, packageId, commandType, entityId, baseRevision, state, createdAt, attemptCount, nextAttemptAt
  syncState: subjectId, packageId, grantId, projectionVersion, cursor, lastSuccessAt, lastErrorCode
  ```

  The store namespace and every query are subject-scoped. User switching or logout locks another subject's local package; it never exposes or silently deletes it. A security-owner-approved recovery path is required before a different subject may resume the package.

- [x] **Step 3: Implement write-through-local field behavior.**

  Field UI waits for the local transaction, not the network. After commit it renders `Saved locally — sync pending` until server acknowledgement.

  `NOT_CHECKED` is a real answer value. Per-question assignment and comment-required rules come from the immutable package snapshot. Another Inspector's question is read-only. Submitted checklist reopen calls the online Backend and requires a reason; it is not an offline command in this protocol.

- [x] **Step 4: Add migration and failure recovery tests.**

  Verify upgrade from every released schema, N/N-1 compatibility, termination at each migration boundary, browser restart with pending/in-flight outbox items, quota errors, duplicate operations, atomic pull page plus cursor, user switch/logout, grant expiry/revocation, and corrupted/incompatible package quarantine. A failed migration opens read-only recovery and never clears stores.

- [x] **Step 5: Run offline storage verification.**

  Expected: tests pass with network absent; no `localStorage` business-record writes exist in `apps/web/src`; no component writes the mock store; no raw actor ID is accepted as authority; no acknowledged or pending record is deleted merely because the user logged out.

  Result 2026-07-21: the red tests first failed on the absent v2 database/repository modules, then exposed premature IndexedDB commits around cryptographic Promises, expiry-write enforcement, and migration-boundary tracking before those defects were corrected. The final schema and repository provide subject-compound stores, exact grant/package/assignment validation, atomic response/Potential Finding/submission plus outbox commits, idempotent replay, unsent coalescing, immutable in-flight operations, causal blocking, atomic pull/cursor updates, lock/quarantine recovery, and released v1-to-v2 migration. React field execution waits for the local transaction and renders `Saved locally — sync pending`; another Inspector's question is read-only and submitted checklist reopen remains online/reason-required. React/Vitest passed 97/97, the focused repository suite 20/20, real persistent-Chrome offline tests 3/3 including server-stopped restart with pending/in-flight recovery, the full HTTP profile and mock/HTTP parity passed, the root Vanilla suite remained 103/103, artifact scans passed, and task-owned processes/containers were cleaned. Evidence is [IndexedDB Field Storage And Outbox](../../demo-evidence/INDEXEDDB_FIELD_STORAGE_2026-07-21.md), `verified locally`, and `candidate-only`. Tasks 8, 12, and 5 subsequently completed OPFS attachment recovery, network sync/conflict delivery, and approved route-family parity. Production controls, deployment, cutover, and `production-ready` evidence remain `blocked`; the next binding slice is Task 13.

### Task 8: Implement Manifest-First OPFS Inspection Attachment Recovery

**Files:**

- Create: `apps/web/src/offline/opfs-inspection-attachment-store.ts`
- Create: `apps/web/src/offline/inspection-attachment-hash-worker.ts`
- Create: `apps/web/src/offline/attachment-recovery.ts`
- Create: `apps/web/tests/offline/opfs-inspection-attachment-store.test.ts`
- Create: `apps/web/tests/e2e/attachment-restart-recovery.spec.ts`

**Interfaces:**

- Consumes: `FieldRepository.createAttachmentManifest()`, `writeAttachmentBytes()`, and approved Inspection Attachment policy.
- Produces: recoverable field attachment bytes with `manifest_created -> writing -> ready -> uploading -> acknowledged -> purge_eligible` lifecycle; it does not create official Auditee Evidence.

- [x] **Step 1: Write failing staging/recovery tests.**

  Inject termination before/after manifest creation, temporary-path creation, each write/flush, hash completion, final-path promotion, metadata readiness, outbox creation, upload start, and acknowledgement. Also cover browser restart, missing referenced bytes, unknown bytes, hash mismatch, quota failure, duplicate filename, and acknowledged cache eligibility.

- [x] **Step 2: Implement staged writes in a dedicated Worker.**

  First commit the IndexedDB manifest in `manifest_created`. Then write to its temporary OPFS path, flush, compute SHA-256, compare byte count, promote to the final path, and atomically mark the manifest `ready` plus create the typed registration outbox operation. The manifest must exist before any OPFS byte can become the sole recoverable copy.

- [x] **Step 3: Implement orphan reconciliation.**

  On startup, compare OPFS paths with manifests. Referenced missing files become a blocking recovery error. Incomplete or unknown bytes move to quarantine metadata and remain recoverable; the first-production implementation never automatically deletes them. Only an owner-approved cache/disposition rule may later mark an acknowledged local copy `purge_eligible`, and purge requires a separate explicit action plus canonical server acknowledgement.

- [x] **Step 4: Verify attachment survival and cleanup.**

  Expected: staged bytes survive page/browser restart at every injected boundary; pending/unknown/quarantined files are never automatically deleted; official Evidence records are never overwritten or deleted; acknowledged local cache eviction remains disabled until its owner policy is approved. Explicit site-data clearing remains an irrecoverable boundary for an unsynced sole copy and is reported literally.

  Result 2026-07-21: red tests first failed on the absent attachment store/hash-worker/recovery modules, then exposed an invalid dependency-bypassing upload fixture, a ready-manifest temporary-path recovery gap, and the missing React startup reconciliation call before those defects were corrected. The final implementation commits a subject-scoped manifest before OPFS bytes, hashes through a dedicated module Worker, performs chunked temporary writes plus read-back size/hash verification, promotes to a final path, and atomically commits `ready` metadata with a typed causal registration outbox. Startup/load reconciliation blocks edits for missing referenced bytes, restores verified temporary bytes, quarantines mismatched/incomplete/unknown bytes, and never automatically deletes local bytes. Upload/acknowledgement atomic boundaries are covered; owner-approved `purge_eligible` policy is absent, so purge remains disabled. React/Vitest passed 132/132, focused attachment tests 35/35, persistent-Chrome offline tests 4/4 with server-stopped exact OPFS hash/byte restart recovery, and the complete HTTP profile passed Go race/live PostgreSQL/OIDC-provider/MinIO integration, OpenAPI 5/5, SQLC clean generation, live HTTP contract 9/9, mock/HTTP Playwright 1/1 each, builds, mock/seed isolation across 12 files and 84 inputs, and cleanup. App-shell scans passed across 12 files and 4 assets including the worker; the root Vanilla suite remained 103/103 and the production dependency audit reported 0 vulnerabilities. Evidence is [OPFS Inspection Attachment Recovery](../../demo-evidence/OPFS_INSPECTION_ATTACHMENT_RECOVERY_2026-07-21.md), `verified locally`, and `candidate-only`. Tasks 12 and 5 subsequently completed network registration/upload/ack delivery, typed conflict recovery, and approved route-family parity. Production controls, deployment, cutover, owner-approved cache disposition, and `production-ready` evidence remain `blocked`; the next binding slice is Task 13.

## Phase 5 — Build The Go Modular Monolith And Real API

### Task 9: Implement The One-Module Go Runtime And PostgreSQL Foundations

**Files:**

- Modify: `api/openapi/aviasurveil360.yaml` only through accepted contract changes.
- Create: `apps/api/go.mod`
- Create: `apps/api/go.sum`
- Create: `apps/api/cmd/api/main.go`
- Create: `apps/api/cmd/worker/main.go`
- Create: `apps/api/internal/platform/config/`
- Create: `apps/api/internal/platform/database/`
- Create: `apps/api/internal/platform/session/`
- Create: `apps/api/internal/platform/idempotency/`
- Create: `apps/api/internal/platform/auditevent/`
- Create: `apps/api/internal/platform/outbox/`
- Create: `apps/api/internal/httpapi/`
- Create: `apps/api/internal/httpapi/generated/`
- Create: `apps/api/migrations/`
- Create: `apps/api/sqlc.yaml`
- Create: module-owned store/query packages under `apps/api/internal/<module>/store/postgres/`.
- Create: `deploy/local/compose.test.yaml`
- Create: `scripts/test-http-profile.sh`
- Test: `apps/api/internal/httpapi/health_test.go`
- Test: `apps/api/tests/integration/migrations_test.go`
- Test: `apps/api/tests/integration/build_graph_test.go`

**Interfaces:**

- Consumes: Task 2 OpenAPI/generated transport, approved Go/auth ADR, frontend Backend operations, and canonical domain vocabulary.
- Produces: generated Go handler interfaces, one Go module containing API and worker commands, forward-only expand/contract migrations, module-owned stores, transaction/idempotency/audit/outbox boundaries, readiness/liveness endpoints, and reproducible PostgreSQL integration dependencies.

- [x] **Step 1: Write failing generated-contract, build-graph, migration, and health tests.**

  Required assertions:

  ```text
  OpenAPI lint/examples/TypeScript generation still pass
  Go handlers regenerate from the same OpenAPI without a diff
  cmd/api and cmd/worker both build inside apps/api/go.mod
  empty install and every retained N-1 upgrade fixture pass
  health/live does not depend on PostgreSQL
  health/ready fails closed when migrations or required dependencies are unavailable
  production configuration rejects test identity, test session, and dev-secret bypasses
  ```

- [x] **Step 2: Create the Go module and dependency lock.**

  Use Go `1.26` language mode, `chi` for routing, `pgx` for PostgreSQL, `sqlc` for checked module-owned queries, and lock-pinned generated OpenAPI request/response/handler types. Keep HTTP framework code in `internal/httpapi`; both commands import only packages beneath the same `apps/api` module.

  Extend `scripts/generate-contracts.sh` and `scripts/check-contracts.sh` so TypeScript and Go generation run from one OpenAPI source and a clean regeneration diff is mandatory.

- [x] **Step 3: Add forward-only migrations and transaction helpers.**

  Initial schemas cover identity/session references, organizations, inspections, immutable checklist template/package snapshots, checklist responses, Potential Findings, Findings, CAP revisions, Evidence versions, review decisions, report versions/decisions, offline grants, idempotency responses, authorized sync changes/cursors, object metadata, audit events, and transactional server outbox.

  Each domain owns its generated query/store package. `platform/database` exposes only the pool and transaction primitive. Migrations use expand/contract compatibility; application rollback never depends on a database downgrade.

- [x] **Step 4: Implement the configuration and deterministic local profile.**

  `deploy/local/compose.test.yaml` pins PostgreSQL, isolates its volumes, exposes health checks, and is reset by `scripts/test-http-profile.sh`. A deterministic test principal/session bootstrap exists only under explicit test configuration and cannot start in production mode. The candidate OIDC/session decision is accepted; its real handler and local-provider integration remain `not run` until Task 10.

- [x] **Step 5: Run local integration foundations.**

  Run:

  ```bash
  docker compose -f deploy/local/compose.test.yaml up -d postgres
  GOCACHE=/private/tmp/aviasurveil360-go-cache go -C apps/api build ./cmd/api ./cmd/worker
  GOCACHE=/private/tmp/aviasurveil360-go-cache go -C apps/api test -race ./...
  ./scripts/check-contracts.sh
  ```

  Expected: both commands build; migrations apply from empty and retained upgrade fixtures; generated outputs are clean; health/config tests pass; teardown leaves no task-owned process or container. Domain behavior remains `not run` until Task 10.

  Result 2026-07-21: the required red runs failed on the absent Go generator/runtime, missing 503 readiness contract, missing platform boundaries, and missing local profile. The first complete profile then caught and rejected a generated request-type collision before the generator was corrected. The final `./scripts/test-http-profile.sh` run built both commands, passed the full Go race suite including empty-install and retained N-1 PostgreSQL upgrades, passed OpenAPI/TypeScript/Go clean regeneration, passed SQLC clean regeneration, and removed its task-owned container, network, and volume. Evidence is [Go And PostgreSQL Foundation](../../demo-evidence/GO_POSTGRES_FOUNDATION_2026-07-21.md), `verified locally`, and `candidate-only`. Canonical domain transitions, real OIDC, object upload/scan, real HTTP parity, offline behavior, sync, deployment, and production behavior are `not run` in Task 9.

### Task 10: Implement Canonical Domain, Session, Authorization, And Audit Rules

**Files:**

- Create/Modify: `apps/api/internal/{identity,organizations,planning,inspections,checklists,potentialfindings,findings,caps,evidence,reports,sync,auditlog}/`
- Create: authorization matrix tests in each module.
- Create: `apps/api/tests/integration/auditee_isolation_test.go`
- Create: `apps/api/tests/integration/idempotent_commands_test.go`
- Create: `apps/api/tests/integration/offline_grant_test.go`
- Create: `apps/api/tests/integration/audit_event_test.go`
- Create: `apps/api/tests/integration/report_approval_test.go`

**Interfaces:**

- Consumes: generated OpenAPI interfaces, platform transaction/idempotency/audit/outbox primitives, approved vocabulary, product permission rules, and approved OIDC/session/offline-grant policy.
- Produces: authoritative domain transitions, same-origin session principal, server-issued offline grants, object/field-level authorization, exact audit events, and Auditee-safe projections.

- [x] **Step 1: Write failing domain state-machine tests.**

  Cover every allowed and blocked transition, including:

  ```text
  Inspector creates an Audit/question/response-scoped Potential Finding only
  Lead Inspector alone returns, dismisses, or converts it; return/dismiss require reason
  Lead conversion selects severity and atomically creates one canonical Finding/public number
  Auditee CAP submission produces CAP_SUBMITTED, never CAP acceptance
  CAA CAP review accepts/rejects/requests more information against exact revisions
  CAP acceptance leaves the Finding open
  Evidence review targets an exact scan-clean immutable EvidenceVersion
  Close records evidence-verified closure; partial/not-close/request-more-info stay open
  Department Manager authorized closure is separate and reason-required
  submitted Evidence and superseded CAP/Evidence versions are preserved
  submitted checklist is read-only; online reopen requires permitted role/stage/reason
  report decisions bind exact versions; DM/GM return/forward only; ED issues/locks
  report issue never closes Findings
  checklist/template/package snapshots remain immutable for existing Audits
  ```

- [x] **Step 2: Write failing authorization tests.**

  Required assertions include list filtering and direct-ID access for Auditee organization isolation, per-question assignment scope for Inspectors, Lead Potential Finding authority, intermediate-only GM permissions, Finance budget-only permissions, Executive Director report issue authority, and Department Manager authorized closure.

  Raw JSON assertions cover Findings, CAPs, Evidence, reports, assignments, messages/notifications when added, dashboard projections, files, sync changes, and conflict payloads. Auditee responses must not contain any forbidden key/value for `Internal CAA Note`, other organizations, internal workload/risk, unreleased reports, or enforcement deliberations.

- [x] **Step 3: Implement domain services independently of HTTP.**

  HTTP handlers parse/validate generated inputs and map problems; domain/application services own transitions, authorization context, module-owned store calls, and transactions. Every mutating command supplies `operationId` plus explicitly named expected revisions.

  The same PostgreSQL transaction records the idempotency key/canonical server-computed payload hash/full response, domain mutation, exactly one audit event for every successful status transition, authorized sync change, and server-outbox record. Same ID/hash replays the original response; different payload reuse fails. Forbidden/invalid/conflicting commands create no successful-transition event or mutation; security telemetry is separate.

  Each audit event records authenticated actor, organization, entity/version, before/after status, reason, server timestamp, operation/correlation ID, and closure basis where relevant. Append-only is the only claim until Records/Security approve stronger tamper evidence.

- [x] **Step 4: Implement authentication boundary.**

  Implement the approved same-origin session/BFF boundary: Go completes OIDC Authorization Code flow, retains provider tokens server-side, issues a Secure/HttpOnly/SameSite session cookie, validates CSRF on state changes, maps the session to a server principal, and enforces expiry/revocation. Permit deterministic test sessions only in explicit test configuration; production startup fails when any test/dev identity bypass is enabled.

- [x] **Step 5: Implement server-issued offline grants.**

  Checkout authenticates and authorizes the current assignment, records the grant server-side, and returns the exact `OfflineGrant` from the architecture section. Tests cover expiry, clock skew, assignment change, package withdrawal, user switch, logout, session revoke, device-loss revoke, and late sync. Client actor/device/time fields never override the authenticated principal or server authorization.

- [x] **Step 6: Run Go unit, integration, race, authorization, and raw-wire tests.**

  Expected: zero failures; both Go commands build; direct-object/list/field isolation and forbidden-key tests pass; lost acknowledgement yields one mutation/transition event; unauthorized transitions produce no mutation or successful-transition audit event; CAP acceptance/report issue cannot close a Finding.

  Result 2026-07-21: the required domain, authorization, projection, idempotency, session, OIDC, migration, and offline-grant tests were written against missing behavior before implementation. The complete `./scripts/test-http-profile.sh` run built both commands, passed the full Go race and live PostgreSQL suite, applied empty and retained N-1 migrations, completed a real Authorization Code + PKCE exchange against the then-current pinned local OIDC provider, passed OpenAPI and all module-owned SQLC clean-regeneration checks, and removed every Task-owned container, network, and volume. `go vet ./...`, React/Vitest 32/32, the intact root Vanilla suite 103/103, both React builds, and `git diff --check` also passed. Evidence is [Canonical Authority Foundation](../../demo-evidence/CANONICAL_AUTHORITY_FOUNDATION_2026-07-21.md), `verified locally`, and `candidate-only`. Object bytes/storage, malware scanning, real HTTP scenario parity, browser offline persistence, sync, deployment, and production behavior remain `not run` in Task 10.

### Task 11: Implement Bounded Attachment/Evidence Upload, Scan, And Real HTTP Parity

**Files:**

- Create: `apps/api/internal/evidence/` upload/version services.
- Create: `apps/api/internal/inspections/attachments/` upload/link services that never create an official Evidence version implicitly.
- Create: `apps/api/internal/platform/objectstore/`.
- Modify: `apps/api/cmd/worker/main.go`.
- Create: `apps/api/internal/worker/evidence/` for upload reconciliation and malware-scan orchestration only.
- Modify: `deploy/local/compose.test.yaml` with pinned S3-compatible storage, bucket initialization/CORS, and deterministic scan adapter.
- Modify: `scripts/test-http-profile.sh`.
- Test: `apps/api/tests/integration/evidence_upload_test.go`.
- Test: `apps/api/tests/integration/inspection_attachment_upload_test.go`.
- Test: `apps/api/tests/integration/evidence_scan_worker_test.go`.
- Test: `apps/web/tests/contract/http-backend-live.test.ts`.
- Test: `apps/web/tests/e2e/canonical-scenario.spec.ts` through the HTTP Playwright project.

**Interfaces:**

- Consumes: distinct Inspection Attachment and Evidence metadata/version models, approved maximum size/media/checksum/storage/scanning policy, transactional outbox, and Tasks 9-10 HTTP/domain foundation.
- Produces: idempotent server-authorized bounded whole-object upload sessions, hash-verified field attachments, immutable official Evidence versions, separate upload/scan/review states, private download authorization, idempotent worker recovery, and the first complete real HTTP canonical transcript.

- [x] **Step 1: Write failing Evidence tests.**

  Verify organization/object authorization, package/grant/assignment scope for field attachments, operation-ID replay, declared-size limits, extension plus server-side MIME sniffing, media allowlist, archive limits, URL expiry/retry, hash/size mismatch rejection, incomplete upload recovery, non-overwriting object keys, malware-scan quarantine/failure/timeout, prior Evidence-version preservation, clean-only review/download/closure, lost acknowledgements, and process restart. Prove that an Inspection Attachment does not become an official EvidenceVersion automatically.

- [x] **Step 2: Implement upload session and completion endpoints.**

  Browser receives one short-lived, write-only staging instruction after authorization. First-production upload is explicitly bounded whole-object retry; multipart/resume is out of scope. The server generates a unique non-overwriting quarantine key. Completion verifies server-observed object size and checksum, declared versus sniffed type, and operation idempotency. Official Auditee upload creates a new immutable Evidence version with `uploadState=UPLOADED`, `scanState=PENDING`, and `reviewState=NOT_READY`; field upload updates only the matching Inspection Attachment record and its scan state.

  No object is public. Download instructions are server-authorized and issued only for `scanState=CLEAN`. A pending, quarantined, failed, or superseded Evidence version cannot be reviewed or support closure. Promotion or linking of a clean Inspection Attachment to later official Evidence requires a separate authorized domain command and new EvidenceVersion; it never overwrites or aliases the attachment record.

- [x] **Step 3: Implement transactional worker claims.**

  Server outbox rows contain event ID/type/version, idempotency key, payload, `available_at`, attempt count, lease owner/expiry, and terminal state. Claim in a short transaction; handlers are idempotent across crash-after-scan-effect/before-ack. Clean scan advances the exact Evidence version to `reviewState=PENDING_CAA_REVIEW`; quarantine/failure remains non-reviewable and records operator-visible state.

  Do not implement notification delivery, production PDF generation, retention deletion, or legal disposition here.

- [x] **Step 4: Complete the deterministic HTTP integration profile.**

  The script starts clean PostgreSQL and object storage, initializes the private quarantine/canonical locations, applies migrations, seeds exact canonical IDs, starts API and worker, enables the fail-closed local test session/scanner adapter, waits for readiness, and supplies Playwright base URLs. It always tears down task-owned processes and containers.

- [x] **Step 5: Run Evidence, worker, Backend-contract, and shared browser verification.**

  Run both Go command builds, Go race/integration tests, the parameterized backend contract against seeded `HttpBackend`, and the same `canonical-scenario.spec.ts` under `mock` and `http`. Fail on unavailable dependencies, skipped tests, zero tests, differing invariant transcripts, or leftover task-owned processes.

  Expected: upload bytes, metadata, checksum, scan state, review state, Evidence versions, Finding transitions, report/dashboard projections, and audit events remain consistent across retry/restart. Mock and HTTP agree on public invariants; production OIDC/MFA, managed-device offline, deployment, and production-readiness remain `not run`.

  Result 2026-07-21: upload, object-store, worker, HTTP-contract, and shared-browser tests were written against absent behavior before implementation. The fresh `./scripts/test-http-profile.sh` run built API and worker, passed the full Go race and live PostgreSQL/OIDC-provider/MinIO integration suite, passed OpenAPI generation 5/5 and all module-owned SQLC clean generation, passed React/Vitest 32/32, passed the live `HttpBackend` contract 9/9, and passed the same canonical Playwright scenario under mock 1/1 and HTTP 1/1. The normal HTTP artifact scan passed across 7 files and 71 inputs without mock/seed or test-profile code. Upload expiry/retry, non-overwriting keys, hash/type/size enforcement, immutable Evidence versions, Inspection Attachment separation, scan clean/quarantine/failure/timeout, crash recovery, clean-only review/download, organization isolation, and task-owned cleanup are covered. `go vet ./...`, the intact root Vanilla suite 103/103, and `git diff --check` also passed. Evidence is [Bounded Upload And HTTP Parity](../../demo-evidence/BOUNDED_UPLOAD_AND_HTTP_PARITY_2026-07-21.md), `verified locally`, and `candidate-only`. The deterministic scanner and local S3-compatible profile are not production services. At the Task 11 checkpoint, browser offline Tasks 6-8, Task 12 sync, Task 5 wider route migration, Task 13 release packet, deployment, cutover, and `production-ready` evidence were `not run` or `blocked` as applicable. Tasks 6-8, 12, and 5 subsequently completed the local PWA/readiness, field-storage, attachment-recovery, foreground-sync, and approved route-family foundations; the current binding next slice is Task 13.

## Phase 6 — Implement Production Sync And Conflict Handling

### Task 12: Implement Idempotent Push/Pull Sync

**Files:**

- Create: `apps/api/internal/sync/`.
- Modify: `apps/web/src/offline/sync-engine.ts`.
- Create: `apps/api/tests/integration/sync_test.go`.
- Create: `apps/web/tests/offline/sync-engine.test.ts`.
- Create: `apps/web/tests/e2e/offline-sync.http.spec.ts`.

**Interfaces:**

- Consumes: closed-union `FieldSyncOperation`, subject/package-scoped outbox, `Backend.sync.pushOperation/pull`, server-issued grants, domain services, and the transactional idempotency/audit/change-feed boundary.
- Produces: one-operation causal delivery, exact acknowledgement replay, typed authorized conflicts/changes, scoped opaque cursors, revocation/resnapshot handling, visible retry state, and safe local reconciliation without automatic byte deletion.

- [x] **Step 1: Write failing sync tests.**

  Cover duplicate delivery, same-ID/same-payload replay, same-ID/different-payload rejection, lost acknowledgement after commit, stale expected revision, repeated offline edits to one response, edit while prior operation is in flight, causal Potential Finding dependency, client actor/time/device tampering, expired/revoked grant, changed assignment, withdrawn package, server validation failure, server restart, cursor page replay, cursor scope mismatch, tombstone, package revocation, history expiry/resnapshot, raw forbidden-field scan, attachment upload retry, two tabs, and absent Background Sync.

- [x] **Step 2: Implement server-side idempotency and revision checks.**

  Process one operation per request. Derive actor/organization from the authenticated session and validate the recorded grant, current assignment, package/version/digest, allowed command type, and expected entity revision. Compute the canonical semantic payload hash on the server.

  In one PostgreSQL transaction, persist the idempotency record with full response, domain mutation, exactly one status-transition audit event when applicable, and authorized change-feed/server-outbox entry. Reusing the ID with the same payload replays the full response; different payload reuse fails. Do not store a retryable infrastructure result as a terminal applied command.

- [x] **Step 3: Implement causal local delivery and one sync owner.**

  Coalesce only never-sent drafts. Freeze in-flight operations. A later edit waits for the authoritative revision and becomes a new operation. Potential Finding creation waits for its checklist response acknowledgement. Attachment registration references the Potential Finding creation operation ID; after acknowledgement the repository records the authoritative Potential Finding/attachment IDs, then uses `Backend.inspectionAttachments` for bounded byte upload and completion. Use the approved managed-browser locking/broadcast mechanism so one foreground tab owns a package sync cycle; server idempotency remains the final duplicate safeguard.

- [x] **Step 4: Implement authorized pull and atomic cursor application.**

  Scope opaque cursors server-side to principal, organization, package, grant, projection version, and high-water mark. Return only typed authorized projections, tombstones, and revocations. Never return raw rows or internal CAA fields in changes/conflicts. Apply a full page and next cursor in one IndexedDB transaction. `resnapshotRequired` blocks editing until a safe full package refresh; it never silently overwrites pending operations or attachment manifests.

- [x] **Step 5: Implement client sync triggers and backoff.**

  Required triggers: startup, foreground, `online`, manual, and app-open retry. Retry only `retryable` results; never retry `forbidden`, `invalid`, or unresolved `conflict` automatically.

- [x] **Step 6: Implement explicit conflict UI.**

  The first production protocol performs no automatic conflict merge. Show the typed authorized conflict summary, preserve the local draft, and require user resolution or re-entry against the authoritative revision. Finding severity, CAP/Evidence decisions, closure, report issue, regulatory publication, and enforcement-related state remain online/server-authoritative and never enter field auto-merge.

- [x] **Step 7: Verify foreground-only fallback and recovery.**

  Disable Background Sync and prove that opening/foregrounding the browser completes pending sync. Repeat after browser and API restart, lost acknowledgement, two-tab contention, cursor replay, grant rejection, and attachment URL expiry. Background Sync cannot be the only successful path and cannot mutate OPFS/IndexedDB outside the same reconciliation rules.

  Result 2026-07-21: the red tests began against absent server/client sync implementations, then exposed causal dependency rewiring after never-sent response coalescing and explicit conflict re-entry, dedicated hash-worker availability during offline staging, expected browser transport noise for deliberate lost acknowledgements, persistent cross-tab status listening, and a test-profile `go run` child-process leak before those defects were corrected. The final implementation accepts one closed-union operation per request, derives authority from the authenticated session plus current grant/package/assignment state, excludes client time from the semantic idempotency hash, and transactionally stores the exact acknowledgement, domain mutation, audit event, authorized change, and outbox message. The foreground client freezes in-flight work, queues later edits against acknowledged revisions, resolves Potential Finding and attachment IDs causally, uploads registered bytes with bounded retry without local deletion, applies typed authorized pull pages and opaque scoped cursors atomically, and exposes terminal conflict/resnapshot state without automatic merge. Web Locks and BroadcastChannel select one package owner and publish its result to other tabs; startup, foreground, `online`, manual, and app-open triggers work without Background Sync. The fresh full HTTP profile passed the Go race/live PostgreSQL/OIDC-provider/MinIO integration suite, OpenAPI 5/5, SQLC clean generation, React 143/143, live HTTP contract 9/9, mock Playwright 1/1, HTTP Playwright 2/2 including lost-ack replay and foreground recovery, both builds, mock/seed isolation across 12 files and 86 inputs, and dependency cleanup. `go vet ./...`, the intact root Vanilla suite 103/103, and `git diff --check` also passed. Evidence is [Idempotent Foreground Sync](../../demo-evidence/IDEMPOTENT_FOREGROUND_SYNC_2026-07-21.md), `verified locally`, and `candidate-only`. Task 5 subsequently completed approved first-production route parity; Task 13 remains `not run`. Production deployment, cutover, legacy removal, and `production-ready` evidence remain `blocked`. The next binding slice is Task 13.

## Phase 7 — Dual-Mode Verification, Cutover Evidence, And Plan Lifecycle

### Task 13: Produce The Local Release-Candidate Verification Packet

**Files:**

- Modify/Create: `apps/web/tests/e2e/*.spec.ts`.
- Create: bilingual evidence documents under `docs/demo-evidence/` after fresh results exist.
- Modify: `README.md`, `README.turkce.md`, `MANIFEST.md`, `docs/index.md`, and build summaries to reflect actual artifacts only.
- Modify: this plan, active index, completed index, and tracker as status changes.
- Create or link: a separately approved production release/operations plan before any deployment or cutover action.

**Interfaces:**

- Consumes: completed approved React, HTTP, Evidence, offline, sync, and local-security slices.
- Produces: evidence-backed `GO`, `CONDITIONAL GO`, or `NO-GO` for a local release candidate, explicit external production gaps, and reconciled plan tracking. It does not produce a production cutover authorization.

- [x] **Step 1: Run static/type/unit verification.**

  ```bash
  npm --prefix apps/web ci
  npm --prefix apps/web run contracts:check
  npm --prefix apps/web run typecheck
  npm --prefix apps/web test
  npm --prefix apps/web run build:demo
  npm --prefix apps/web run build:http
  node apps/web/scripts/assert-http-artifact.mjs apps/web/dist
  GOCACHE=/private/tmp/aviasurveil360-go-cache go -C apps/api build ./cmd/api ./cmd/worker
  GOCACHE=/private/tmp/aviasurveil360-go-cache go -C apps/api test -race ./...
  ./scripts/test-http-profile.sh
  node --test tests/*.test.js
  git diff --check
  ```

  Expected: exit `0` for every command; generated output is clean; no test is skipped or reports zero executed cases; the HTTP artifact has no mock/seed input; both Go commands build; cleanup leaves no task-owned processes or containers.

- [x] **Step 2: Run mock-mode browser verification.**

  Verify every approved `first-production` ledger entry, canonical lifecycle, stable reset, no production claims, desktop/tablet/mobile, no critical overflow, accessibility/keyboard basics, and zero unexpected console warnings/errors. Persistent IndexedDB mock-server behavior is not required unless separately approved.

- [x] **Step 3: Run HTTP-mode browser verification against real local dependencies.**

  Run the exact same scenario files against Go/PostgreSQL/S3-compatible storage. Fail when the HTTP profile is unavailable or skipped. Compare normalized invariant transcripts for exact IDs/scopes, lifecycle transitions, revisions, Potential Finding authority, CAP review, Evidence versions/scan/review, report decisions, audit events, dashboards, and denial outcomes. Add raw-wire forbidden-field, direct-ID/list isolation, migration, upload, and worker-retry paths.

- [x] **Step 4: Run real offline browser/device verification.**

  Required cases:

  ```text
  first online load and package check-out
  server-issued grant validation and current assignment
  full browser restart in real network-off/server-stopped mode
  checklist and Potential Finding draft writes while offline
  Inspection Attachment survival across every crash boundary
  foreground/manual sync after network return
  conflict presentation and resolution
  expired/revoked offline grant, reassignment, logout, and user switch
  persistence denied and quota insufficient gates
  absent Background Sync
  N/N-1 app/schema/protocol update with pending unsynced work and two tabs
  failed migration read-only recovery and prior-shell rollback
  explicit site-data clearing and irrecoverable unsynced-copy boundary
  managed-browser clear-on-exit/policy enforcement
  ```

- [x] **Step 5: Run security and operational verification.**

  Include same-origin session/CSRF/authorization review, raw Auditee response scans, upload/quarantine/scan/download controls, approved local-data/key handling, CSP, rate limits, dependency/SBOM review, PostgreSQL plus object-store backup/restore drill, worker/outbox observability, supported managed-browser evidence, and rollback rehearsal. Real configured OIDC/MFA, production hosting, disaster recovery, retention/legal-hold execution, external penetration review, and production monitoring remain separate evidence unless actually run.

- [x] **Step 6: Reconcile local release-candidate status.**

  Move this plan at most to `ready-for-verification` after the selected implementation objective and every required local gate pass. Do not label it `production-ready`, archive the legacy demo, deploy, route traffic, or mark production cutover complete. Required local gates cannot pass through a documented gap; only explicitly non-blocking external gaps may remain and each must have a named owner, tracker note, and separately approved production release/operations plan.

  Result 2026-07-21: the Task 13 red runs first exposed absent API security/rate-limit middleware, an untestable worker batch boundary, a missing recovery drill, an absent build CSP source, a checkout test without its restart-canary precondition, and an offline Playwright pattern that selected HTTP-only tests. Those defects were corrected without weakening readiness or adding skips. A narrow lock-compatible `js-yaml` 4.3.0 override also closes the two prior high-severity development-tool audit findings while preserving clean contract generation. The fresh clean-install packet passed OpenAPI/generated contracts 6/6, TypeScript, React/Vitest 148/148, API/worker build and `go vet`, the full Go race/live PostgreSQL/OIDC-provider/MinIO suite, HTTP Backend contract 11/11, mock Playwright 5/5, HTTP Playwright 7/7 including typed stale-revision presentation/local-draft preservation/explicit re-entry, real offline Playwright 6/6, HTTP isolation across 12 files and 89 inputs, both app-shell scans, worker/outbox drain, root legacy/parity 106/106, full and production-only npm audits with 0 vulnerabilities, CycloneDX npm SBOM review, Go runtime inventory, and task-owned cleanup. The isolated recovery drill restored the canonical PostgreSQL fingerprint and exact 47-byte private object with matching metadata/hash. Evidence is [Local Release-Candidate Evidence](../../demo-evidence/LOCAL_RELEASE_CANDIDATE_2026-07-21.md), `verified locally`, and `candidate-only`. The local decision is `GO`; release remains `release pending`. Production deployment/cutover is `NO-GO` and `blocked`: no production release/operations plan was authorized or created in this slice, and production Identity, hosting, records, security, monitoring/on-call, pilot, and release-authority evidence remains external.

## Verification Strategy

### Frontend Unit And Component

- HTTP build graph and public directory contain no mock/seed input or demo configuration; runtime selection cannot re-enable mock.
- `MockBackend` backed by `MemoryMockStore` satisfies the shared domain contract with deterministic clock/IDs.
- `HttpBackend` maps success/problem/cancel/timeout responses without hidden retries.
- Auditee backend objects and renders never contain internal CAA or other-organization data.
- CAP submission remains `CAP_SUBMITTED`; only separate authorized review may accept it.
- CAP acceptance never produces closed Finding UI.
- Inspector Potential Finding creation does not issue a Finding or select severity; Lead conversion does.
- Offline state labels distinguish local commit, sync pending, conflict, rejected, and server acknowledged.

### Offline Storage

- Response/Potential Finding/checklist submission and typed outbox operation write atomically.
- Schema migrations preserve pending/in-flight work and provide read-only recovery on failure.
- Managed-policy, persistence-denied, OPFS/IndexedDB health, advisory quota-headroom, version, and grant failures block official check-out.
- Browser restart retains checked-out package/grant, responses, outbox, manifests, and Inspection Attachments.
- Manifest-first OPFS/IndexedDB reconciliation never automatically deletes referenced, unknown, unsynced, or quarantined bytes.
- Multi-tab N/N-1 Service Worker/app/schema/protocol updates do not edit incompatible packages or delete old recovery assets.
- Background Sync absence does not block foreground/manual sync.

### Go Domain And Security

- Domain transitions are independent of HTTP handlers.
- Object and field authorization applies to list, direct-ID, download, conflict, and pull access.
- Unauthorized or conflicting operations produce no mutation.
- Idempotency full response, mutation, required status-transition audit event, authorized change, and background outbox record share the domain transaction.
- Evidence versions, hashes, scan states, and review decisions are preserved.
- Only scan-clean exact Evidence versions may be reviewed or support closure.
- Report decisions bind exact versions and preserve DM -> GM -> ED authority without closing Findings.
- Template snapshots remain immutable for existing Audits.

### Contract And Integration

- OpenAPI validates examples; TypeScript and Go generated outputs regenerate cleanly.
- `MockBackend`/`MemoryMockStore` fixtures and Auditee projections pass contract validation.
- Real Go handlers pass the same parameterized Backend contract and exact Playwright scenario files.
- HTTP dependencies being unavailable, skipped tests, or zero tests fail the gate.
- PostgreSQL migrations pass empty install and upgrade fixtures.
- Bounded S3-compatible uploads and scan worker survive URL expiry, lost acknowledgement, retry, and process restart.
- Direct commands and sync operation idempotency survive lost acknowledgements without duplicate mutations/audit events.
- Authorized cursor/tombstone/revocation/resnapshot behavior passes raw forbidden-field tests.

### Browser And Scenario

- Same Playwright scenario set runs in `mock` and `http` profiles.
- Approved `first-production` roles/routes/actions remain reachable only by permitted actors; `later`/`demo-only` routes remain legacy-only.
- Canonical lifecycle works end to end.
- Offline field scenario survives actual reload/browser restart and network loss.
- Desktop, tablet, and mobile views remain readable and actionable.
- Task-owned browsers, servers, containers, and test processes are cleaned up.

## Acceptance Gates

### Contract And React Candidate Gate

- Approved vocabulary, OpenAPI examples, generation, and behavior ledger pass.
- Every migrated role/route/action has executable state/visibility assertions; route-name inventory alone does not pass.
- Canonical mock scenario passes in React.
- Legacy scenario remains available and passing.
- No React route relies on legacy global state.
- HTTP artifact contains no mock/seed inputs or demo public configuration.

### Real HTTP Vertical Gate

- API and worker build from one Go module; generated handlers match OpenAPI.
- Go/PostgreSQL/object-storage/test-scanner integration starts from clean state and tears down cleanly.
- The parameterized Backend contract and exact same canonical Playwright scenario pass in mock and HTTP profiles with matching invariant transcripts.
- Raw Auditee JSON isolation, direct/list/download authorization, lifecycle, idempotency, Evidence, report, and audit-event tests pass.

### Offline Gate

- Official check-out requires approved managed policy, persistent storage, separate IndexedDB/OPFS health, advisory quota headroom, compatible app/schema/protocol versions, and a valid server-issued grant.
- Field writes are local-first and atomic with outbox creation.
- Browser restart, N/N-1 update/rollback, manifest recovery, multi-tab ownership, and foreground sync pass on supported target devices.
- Unsynced state is always visible.
- Explicit site-data clearing and device/profile deletion remain disclosed irrecoverable boundaries for an unsynced sole copy.

### Incremental Route Parity Gate

- Only Product-approved `first-production` families migrate.
- Each family extends OpenAPI, mock, Go, authorization, React, ledger, and both browser profiles in one reviewable increment.
- No visible control is inert or falsely represented as working.
- `later` and `demo-only` behavior remains available in the legacy demo.

### Local Release-Candidate Gate

- Every required local contract, build, mock, HTTP, offline, security, migration, upload/scan, backup/restore, responsive/accessibility, and cleanup check passes; a required local check cannot be waived by documenting its gap.
- Stakeholder/user acceptance is explicit for the local release candidate only.
- Remaining external gaps are literal, owner-assigned, and tracked.
- Result may be `ready-for-verification`; it is not `production-ready` and does not authorize deployment or cutover.

### External Production Cutover Gate — Separate Approval Required

- A separately approved operations/release plan defines trusted clean builds, artifact provenance, production OIDC/MFA, environment/secrets, staging, migration compatibility, backup/restore and disaster-recovery evidence, monitoring/SLO/on-call, pilot cohort, routing, rollback triggers, legal/records approvals, and release authority.
- Deployment, traffic routing, legacy archival/removal, and production-readiness claims remain blocked until that plan passes and the user explicitly authorizes the exact action.

## Risks And Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| Big-bang React rewrite loses behavior | Broken stakeholder workflows | Preserve legacy demo, port one canonical vertical, use an executable role/route/action/state/visibility ledger, prove HTTP before broad route migration |
| Mock/Go behavior drifts | Demo passes while production fails | OpenAPI before adapters, generated transport/handlers, one Backend contract, parameterized conformance, exact shared Playwright files and invariant transcripts |
| Browser evicts local data | Unsynced inspection loss | Require granted persistence plus managed policy, IDB/OPFS/restart canaries, advisory quota headroom, early sync, and explicit unsynced warnings; never claim reservation/durability |
| User clears site data | Local work loss | Explicit product warning, MDM/browser policy, no false guarantee, sync as soon as possible |
| Background Sync unavailable | Pending work never transmits | Startup/foreground/online/manual/app-open retry are mandatory |
| Service Worker update breaks pending packages | Field interruption/data incompatibility | Separate app/schema/package/protocol versions, N/N-1 fence, one multi-tab migration owner, old-cache retention, update deferral, read-only recovery, rollback tests |
| OPFS and IndexedDB diverge | Missing or orphaned Inspection Attachment | Manifest before bytes, hash/size verification, crash injection at every boundary, quarantine unknown bytes, no automatic deletion |
| Repeated offline edits conflict with themselves | Avoidable conflict/data loss | One-operation causal delivery, coalesce only unsent drafts, freeze in-flight payload, use authoritative revision for later edit |
| Two inspectors edit the same record | Silent overwrite | Expected revisions, server assignment authorization, typed conflict UI, no automatic merge in the first protocol |
| Offline actor loses authority | Unauthorized later sync | Server-issued/recorded time-limited grant, current assignment/permission check on every operation, visible terminal rejection/recovery |
| Mock code enters HTTP artifact | Security/data-truth failure | Build-time-separated entry/public graph, manifest/metafile source assertion, output inventory, runtime cannot switch backend family |
| Go modular monolith becomes tangled | Slow change and weak ownership | One module, capability/domain packages, module-owned stores, platform transaction primitive only, allowed dependency graph |
| Worker crash duplicates external effects | Duplicate scan/upload side effects | Transactional outbox identity/version/lease plus handler-level idempotency across crash-after-effect/before-ack |
| Evidence upload weakens chain-of-custody | Untrusted or lost files | Bounded whole-object retry, unique private quarantine key, server-observed checksum/size/type, immutable versions, separate scan/review states, clean-only review/download/closure |
| Auditee-safe UI receives internal data | Confidentiality breach | Auditee-specific OpenAPI projections, raw JSON forbidden-field tests, organization scoping for list/direct/download/sync/conflict paths |
| Local candidate is mistaken for production | Unsafe cutover | Rename local gate, literal evidence labels, separate approved release/operations plan with trusted build/staging/restore/rollback/monitoring evidence |
| Browser-only constraint conflicts with hard remote-wipe guarantees | Security policy failure | Managed device/browser controls, limited server-issued offline grant, explicit owner acceptance; block production if policy demands cannot be met |

## Dependencies

- Stakeholder approval that browser-delivered field inspection must work offline.
- Product/CAA Operations approval of the canonical vocabulary, Potential Finding/CAP/report rules, and `first-production` route inventory.
- Engineering/Platform ADR accepting React/Vite and a one-module Go modular monolith with named maintenance/on-call ownership.
- Supported managed browser/device/profile and clear-on-exit policy decision.
- Security approval for browser-local sensitive data controls, key handling/recovery if app encryption is required, and offline-grant behavior.
- Same-origin session/BFF approval, OIDC/MFA provider, organization identity mapping, cookie/session/CSRF/revocation policy, and test-identity boundary.
- PostgreSQL, private S3-compatible storage, malware scanner, hosting region, RPO, RTO, backup, restore, disaster-recovery, and observability decisions.
- Product owner approval of one-operation causal sync, no automatic first-release merge, conflict UX, and offline-authoritative boundaries.
- Records/Legal/Standards approval of Inspection Attachment versus official Evidence, size/media/checksum/scan policy, regulatory wording, Evidence/closure terminology, report versioning, retention/legal hold/disposition, and audit tamper-evidence claims.
- Harness-owner approval of the local production-application lane; remote CI remains separately authorized.
- Engineering capacity for React parity, Go domain work, offline browser engineering, and dual-mode test automation.
- Docker-compatible local integration environment for PostgreSQL and object storage verification.
- A separately authorized production plan before deployment, traffic routing,
  cutover, or `production-ready` claims.

## Ownership Boundaries

- Product owner: route classification, offline scope, check-out semantics, user messages, vocabulary, behavior-ledger differences, local-candidate acceptance, and phase priority.
- CAA operations owners: field device reality, assignment/package duration and size, checklist/Inspection Attachment needs, conflict-resolution policy, and pilot acceptance.
- Security/Identity: supported browsers/devices/profiles, same-origin OIDC/MFA session, CSRF, offline grant, local data/key lifecycle, CSP, upload/download, session/device revoke, and threat review.
- Records/Legal/Standards: Inspection Attachment/Evidence boundary, Evidence versioning/retention/disposition, chain-of-custody language, signature status, regulatory references, legal hold, report authority, audit tamper-evidence requirements, and authorized closure rules.
- Frontend engineering: React migration, PWA, IndexedDB/OPFS, local repository, sync UI, accessibility, and browser tests.
- Backend engineering: OpenAPI mapping, one Go module, module-owned PostgreSQL stores, domain transitions, authorization, idempotency/sync, object storage, workers, Auditee projections, and audit events.
- Platform/operations: Go ownership/on-call, environments, secrets, database/object storage/scanner operations, backups, restore/DR, monitoring, trusted build/release, rollback, and supported browser/device distribution.
- QA: executable behavior ledger, dual backend profiles, fail-on-skip truth, real offline/browser restart, conflict, multi-tab update/migration, security regression, restore rehearsal, and evidence capture.
- Current static demo: behavioral/stakeholder reference only; it does not own production persistence, security, sync, or Evidence correctness.

## Decision Log

Record a named human owner, decision date, accepted value/policy, and evidence
pointer before changing a row to `accepted`. `blocked` means the dependent slice
must not start; it does not block unrelated earlier slices.

| Decision | Required Owner | Status | Accepted Value / Evidence |
|---|---|---|---|
| 2026-07-20 adversarial review disposition | Plan owner | accepted | `NO-GO as written`; corrected plan retains React/Vite, one Backend, field-only offline, Go modular monolith, PostgreSQL/S3, and transactional outbox conditionally. Production-readiness not claimed. |
| Tasks 2-4 mock-data first executable slice | Current user / plan owner | accepted | Explicit authorization dated 2026-07-20. Tasks 2-4 are implemented and `verified locally`; evidence is [React Mock Slice](../../demo-evidence/REACT_MOCK_SLICE_2026-07-20.md), commit `59a2bcf`. Result remains `candidate-only`. |
| Tasks 5-13 local release-candidate execution | Current user / plan owner | accepted | Explicit authorization dated 2026-07-21. Execute every remaining Task in binding slice order, commit and push after each Task, and stop at a local release-candidate decision. This does not authorize production deployment, traffic cutover, legacy removal, or a `production-ready` claim. |
| Canonical vocabulary and lifecycle mapping | Product + CAA Operations | accepted | Current user / plan owner acceptance dated 2026-07-21: the canonical English product docs, matching Turkish companions, verified legacy behavior oracle, and versioned OpenAPI remain authoritative across Tasks 5-13. Regulatory references remain configured references, not legal advice. |
| `first-production` / `later` / `demo-only` route inventory | Product | accepted | Current user / plan owner acceptance dated 2026-07-21: the Core MVP families in `MVP_SCOPE_AND_ROADMAP.md` are `first-production` together with the eight role entries; AI, advanced risk/BI, broad regulatory editing, USOAP/SSP expansion, enforcement case management, and generic workflow surfaces remain `later` or `demo-only` in the intact legacy demo. Task 5 implemented and locally verified the approved organization, planning authority/calendar, versioned configuration, reminder-rule, and audit-trail families; see [route-family evidence](../../demo-evidence/FIRST_PRODUCTION_ROUTE_FAMILIES_2026-07-21.md). |
| Contract generation and demo/HTTP build-profile ownership | Current user / plan owner | accepted | Tasks 2-4 established the minimal versioned OpenAPI, checked TypeScript generation, and build-time-separated React/Vite demo/HTTP entries. Tasks 9-11 added checked Go generation and a real local API profile; the HTTP artifact excludes mock/seed and test-profile inputs. This is `candidate-only`, not a production API authorization. |
| React/Vite and one-module Go ownership ADR | Engineering + Platform | accepted | Current user / plan owner acceptance dated 2026-07-21 for the local candidate: React/Vite browser client plus one Go `1.26` modular-monolith module with API and worker commands. Production maintenance, on-call, and deployment ownership remain external release blockers. |
| Same-origin OIDC session/BFF, MFA, cookie, CSRF, expiry, and revocation | Security + Identity | accepted | Candidate policy accepted 2026-07-21: provider-neutral OIDC Authorization Code BFF, then-current local OIDC integration, provider-enforced MFA, server-side provider tokens, Secure/HttpOnly/SameSite cookie, CSRF on mutations, 30-minute idle and 8-hour absolute session, explicit logout/revoke, and fail-closed production configuration. |
| Managed browser/device/profile and local-data protection | Security + CAA Operations + Records | accepted | Candidate policy accepted 2026-07-21: current managed Chrome on an encrypted OS/profile, clear-on-exit disabled, subject-scoped local records, no cross-subject render, and explicit refusal of official checkout when policy attestation or storage health is absent. Tasks 6-7 locally verify policy/readiness plus subject-scoped field storage, lock/quarantine preservation, and real restart recovery; see [PWA evidence](../../demo-evidence/PWA_OFFLINE_READINESS_2026-07-21.md) and [field-storage evidence](../../demo-evidence/INDEXEDDB_FIELD_STORAGE_2026-07-21.md). App-level encryption and production MDM evidence remain external release decisions. |
| Offline grant/package duration, scope, reassignment/revoke/late-sync behavior | Product + Security + CAA Operations | accepted | Candidate policy accepted 2026-07-21: 24-hour grant, 72-hour checked-out package, 5-minute clock-skew tolerance, exact subject/device/package/assignment scope, late or revoked sync rejected without deleting local recovery data, logout locks the package, and user switch never exposes it. Tasks 7 and 12 verify local expiry/revoke/lock/quarantine behavior plus authoritative network rejection, typed conflicts, and foreground retry without deleting local recovery data. |
| Inspection Attachment versus official Evidence; size/media/checksum/upload/scan | Product + Records + Security | accepted | Candidate policy accepted 2026-07-21: offline `InspectionAttachment` remains distinct from online official `EvidenceVersion`; PDF/JPEG/PNG only, 25 MB per object, SHA-256, server MIME sniffing, private non-overwriting object keys, bounded whole-object retry, and clean-scan gate. Archives and multipart/resume remain out of scope. |
| Conflict policy | Product + CAA Operations | accepted | Candidate policy accepted 2026-07-21: no automatic merge; preserve the local draft, return an authorized typed conflict, and require explicit user re-entry/resolution against the authoritative revision. |
| Retention/legal hold/disposition and audit tamper-evidence claim | Records + Legal + Security | accepted | Candidate policy accepted 2026-07-21: no automatic retention deletion or legal disposition; preserve submitted/superseded versions; describe audit storage only as append-only. Production retention, legal hold, and tamper-evidence controls remain external blockers. |
| Hosting region, PostgreSQL/object storage/scanner, RPO/RTO, restore/DR, monitoring | Platform + Operations | accepted | Candidate policy accepted 2026-07-21: isolated pinned local Docker dependencies use PostgreSQL, private MinIO-compatible storage, and deterministic scanner/worker behavior with a local backup/restore rehearsal. Production providers, region, RPO/RTO, monitoring, and on-call remain external release blockers. |
| Production release/cutover ownership and runbook | Product + Platform + Operations + QA | blocked | No production release/cutover plan is authorized. Remote discovery/apply, preprod disposition, capacity/recovery evidence, owner inputs, deployment, release, and traffic cutover remain separate blocked gates. |

## Plan Lifecycle

- Current status: `active`; Tasks 2-13 are implemented and `verified locally` for the `candidate-only` local release candidate, and the linked 17-surface React parity remediation is `ready-for-verification`.
- Review status: the initial 2026-07-20 adversarial review is complete; verdict was `NO-GO as written`, and its plan-level corrections are incorporated in this revision.
- Current next todo: independently review the four 22 July full-platform follow-up plans, then execute Full React 86-Screen Migration Task 1; AWS and production actions remain separately authorized gates.
- Move to `ready-for-verification` only after the selected implementation objective and every required local gate pass. A required local gate cannot pass through a documented gap.
- Do not move to `completed/` merely because a local release candidate exists. Completion requires objective completion, required local verification, explicit stakeholder/user acceptance, completed-index entry, tracker reconciliation, and an explicit disposition for the separate production release/operations dependency.
- `production-ready`, deployment, traffic routing, cutover, and legacy removal remain blocked until the separately approved production release/operations plan passes and the user authorizes the exact action.
- If a narrower replacement plan supersedes a phase, keep this umbrella plan active or mark it `superseded` only with explicit replacement links and accurate next todo.

## Review Prompt

```text
Re-review the corrected AviaSurveil360 production transition plan at docs/exec-plans/active/2026-07-20-react-vite-pwa-go-offline-first-production-plan.md.

NOEDIT. Do not modify files, create branches, commit, push, deploy, open a PR, or post GitHub comments.

Read AGENTS.md first, then README.md, MANIFEST.md, docs/index.md, docs/product-specs/index.md, docs/product-specs/product-plan/MVP_SCOPE_AND_ROADMAP.md, docs/product-specs/data-and-rules/CONCEPTUAL_DATA_MODEL.md, docs/product-specs/data-and-rules/STATUS_PERMISSION_SECURITY.md, docs/product-specs/workflows/AUDIT_CHECKLIST_WORKFLOW.md, docs/product-specs/workflows/FINDING_CAP_EVIDENCE_WORKFLOW.md, docs/agent-harness/index.md, docs/agent-harness/output-contract.md, docs/agent-harness/verification-matrix.md, docs/exec-plans/index.md, docs/exec-plans/tech-debt-tracker.md, and docs/demo-evidence/BUILD_SUMMARY.md.

Review the corrected plan adversarially but practically. Check:
1. Whether OpenAPI and the bilingual vocabulary now precede both adapters and generated TypeScript/Go drift fails clean regeneration.
2. Whether the capability-composed Backend expresses the canonical checklist -> Potential Finding -> Finding -> CAP submit/review -> Evidence version/review -> closure/report/dashboard path without duplicate repositories.
3. Whether build-time demo/HTTP separation provably excludes mock handlers, seed data, and demo public configuration from the HTTP artifact.
4. Whether the same-origin session/BFF and server-issued OfflineGrant boundaries prevent client actor/device/time/expiry claims from becoming authority.
5. Whether field execution alone is offline-first, official Auditee CAP/Evidence and critical decisions remain online/server-authoritative, and InspectionAttachment is distinct from EvidenceVersion.
6. Whether IndexedDB manifest/outbox, manifest-first OPFS bytes, Cache Storage shell, persistence/advisory-quota/readiness gates, site-data-loss language, multi-tab updates, migrations, restart, and Background Sync fallback are production-plausible without durability overclaim.
7. Whether one-operation causal sync, full transactional idempotency replay, expected revisions, typed conflicts, authorized cursor/tombstone/revocation/resnapshot, lost acknowledgements, and repeated edits are safe.
8. Whether Auditee raw-wire isolation, CAP acceptance, Evidence versions/scan gate, report-version authority, authorized closure, regulatory wording, and every-transition audit events remain intact.
9. Whether API and worker are buildable/tested in one Go module, module-owned stores prevent cross-domain writes, PostgreSQL/outbox recovery is explicit, and bounded upload/scan is sufficient without premature multipart, notifications, PDF, retention, or microservices.
10. Whether the real HTTP profile is deterministic and fail-closed, the exact same contract/browser scenarios run in both profiles without skips, and invariant transcripts detect semantic drift.
11. Whether the corrected execution order proves a real HTTP vertical before broad route migration and uses an executable behavior ledger rather than route-name parity.
12. Whether the Local Release-Candidate Gate and separate production release/operations dependency prevent local/demo evidence from being mislabeled as production readiness.

Return findings first, ordered by severity and labeled Blocking, Suggestion, Nit, or FYI. For every Blocking or Suggestion finding, cite the exact plan section and provide a concrete replacement. Then return:
- GO, CONDITIONAL GO, or NO-GO
- required owner decisions before implementation
- the minimum first executable slice
- verification gaps
- a concise corrected architecture summary

Do not implement the plan. Do not treat the current frontend-only demo as production evidence. Use literal evidence labels and state production-readiness not claimed.
```

## Execution Prompt

```text
Execute only the explicitly user-authorized slice from docs/exec-plans/active/2026-07-20-react-vite-pwa-go-offline-first-production-plan.md for AviaSurveil360 after Task 1 records named owner decisions and the selected slice has `GO` or `CONDITIONAL GO` with every slice-specific Blocking condition resolved.

Respect AGENTS.md and the plan's Global Constraints. Work task-by-task within the plan's defined scope. Do not dispatch subagents unless the user explicitly authorizes subagent work. Work on the current branch; do not create, switch, rename, or delete branches. Do not stage, commit, push, deploy, open a PR, or post GitHub comments unless the user separately authorizes that exact action. Preserve unrelated untracked docs/demo-evidence/stakeholder/ and outputs/ content.

Start with Task 1. Update the Decision Log, active index, and tracker before implementation. Then execute only the approved slice in the binding Delivery Slices order; do not silently implement the entire umbrella plan. Preserve the root Vanilla JavaScript demo as the behavioral reference until accepted cutover and a separately authorized archival/removal action.

Freeze the approved bilingual vocabulary and minimal OpenAPI contract before either adapter. Use React + TypeScript + Vite with build-time-separated demo/HTTP entries. Implement one capability-composed Backend with MockBackend backed by MemoryMockStore for deterministic demo/tests and HttpBackend mapped from generated transport for the real Go API. FieldRepository remains a distinct local working-set boundary; MockStore simulates remote canonical state and is never dual-written. Do not add IndexedDB mock persistence in the first slice, DemoSyncTransport, duplicate per-entity mock/real repository families, Next.js, native wrappers, microservices, Storage Buckets, SQLite WASM, CRDTs, or a proprietary sync platform without a separately approved plan change.

Use the approved same-origin session/BFF for HTTP auth and a server-issued/recorded OfflineGrant for field checkout. For field execution, write structured records and typed outbox operations atomically to IndexedDB; create the attachment manifest before writing InspectionAttachment bytes to OPFS; cache only the app shell in Cache Storage; and require managed-policy, persistence, IndexedDB/OPFS/restart, advisory quota-headroom, version, and grant readiness. Treat Background Sync as optional; startup, foreground, online, manual, and app-open retry paths are mandatory. Keep official Auditee Evidence, CAP/Evidence review, approval, report issue, and closure online/server-authoritative.

Build cmd/api and cmd/worker inside one apps/api Go module. Process sync one typed operation at a time in causal order. In one PostgreSQL transaction store exact idempotency response, domain mutation, every required status-transition audit event, authorized change feed, and server outbox. Use private bounded whole-object Evidence upload, server-observed checksum/size/type, immutable versions, quarantine/scan, and clean-only review/download/closure. Do not add notification delivery, production PDF generation, multipart upload, or destructive retention in the first backend slice.

Use focused tests and broader gates for every task. Keep mock and HTTP behavior aligned through OpenAPI, backend contract tests, Go integration tests, and the same critical Playwright scenarios in both modes.

After every material plan-state change, reconcile docs/exec-plans/index.md, this plan's Decision Log/next todo/status, docs/exec-plans/tech-debt-tracker.md, and any evidence/build summaries actually affected. Use verified locally, not run, blocked, candidate-only, release pending, and production-ready literally. Task 13 can produce only a local release-candidate recommendation. Do not deploy, route traffic, archive legacy, or claim production security, offline reliability, Evidence chain-of-custody, or production readiness without the separately approved release/operations plan, matching fresh evidence, and exact user authorization.
```
