# NCAA Platform V2 And MVP Implementation Plan


**Goal:** Convert the NCAA platform feedback into a practical AviaSurveil360 plan that separates a mock Frontend V2 clickable demo from a production MVP, regulatory intelligence, offline mobile, and AI governance work.

**Architecture:** The current Vanilla JavaScript demo remains a stakeholder UX reference and may be extended for a V2 clickable demo only. The V2 demo should be backend-ready by keeping state changes behind small data-service/storage functions, using API-shaped records, stable IDs, status enums, and an outbox shape that can later map to real sync endpoints. Production work must still be planned as a new secure, API-backed system with a modular monolith backend, PostgreSQL, object storage, server-side authorization, versioned workflows, and append-only audit events. Regulatory, USOAP/SSP, offline mobile, and AI capabilities are separate tracks with explicit governance and verification gates.

**Tech Stack:** Frontend V2 demo: existing static HTML, CSS, and Vanilla JavaScript with mock data, client-side state, `localStorage` demo persistence, and simulated offline sync status. Production recommendation: TypeScript web frontend, modular monolith API, PostgreSQL, S3-compatible object storage, background jobs, full-text regulatory search, PDF/document generation, and a mobile/offline client selected after discovery.

---

- **Date:** 2026-06-23
- **Status:** ready-for-verification
- **Owner:** Product architecture planning
- **Type:** Platform V2 demo and production MVP planning
- **Authority:** Root `AGENTS.md`; `docs/exec-plans/` is the active plan folder.

## Objective

Create an execution-ready plan for the next AviaSurveil360 scope after the NCAA feedback:

1. Extend the current static demo into a richer Frontend V2 customer demo, clearly marked as mock/simulated.
2. Define the production MVP architecture and scope without treating the current Vanilla JavaScript demo as production architecture.
3. Model Regulatory Intelligence for NAMCARS/NAMCATS-style documents, USOAP PQ/CE readiness, SSP/NASP/SPI management, and change impact analysis.
4. Plan offline mobile inspection discovery before selecting PWA, Flutter, React Native, or another client approach.
5. Define AI Inspector Assistant governance so AI can assist drafting and research but cannot publish official checklist, finding, closure, enforcement, or legal decisions without authorized human review.

## Source Inputs

This plan is grounded in:

- The existing AviaSurveil360 source-of-truth docs under `docs/`.
- The current demo-only prototype plan at `docs/exec-plans/active/2026-06-14-aviasurveil-demo-only-prototype-plan.md`.
- The NCAA feedback summarized by the user on 2026-06-23.
- The observed current package mismatch: `README.md` and `MANIFEST.md` still describe a Markdown-only package, while the repo now contains `index.html`, `css/`, and `js/` prototype files.

## Scope

### In Scope

- A Frontend V2 demo plan that adds new screens while preserving the existing static architecture for demo work.
- Demo-only persistence using `localStorage` for CAP submissions, mock evidence filenames, AI decisions, filters, created findings, and simulated offline outbox items.
- A backend-ready demo data boundary: views call helper/data-service functions instead of reading or writing `localStorage` directly, so future API-backed persistence can replace the demo store with less rework.
- A production MVP plan that treats the demo as UX reference only.
- A domain model expansion for regulatory documents, clause versions, checklist mappings, USOAP PQ/CE readiness, SSP/NASP/SPI, CAP effectiveness, AI suggestions, offline packages, and evidence governance.
- Security, permissions, organization isolation, audit trail, evidence versioning, and workflow state-machine requirements for production.
- Verification gates for demo, architecture, regulatory wording, mobile/offline assumptions, and AI governance.

### Out Of Scope

- Implementing the Frontend V2 demo in this plan task.
- Building a production backend, database, API, mobile app, AI system, regulatory ingestion service, or document repository in this plan task.
- Claiming legal validity, enforcement automation, certificate action, USOAP EI improvement, production security readiness, production evidence repository readiness, or offline reliability without later implementation and evidence.
- Directly migrating the current Vanilla JavaScript prototype into production architecture.
- Publishing official checklist templates from AI output without Standards, Legal, or authorized inspector approval.
- Treating `localStorage` demo persistence or simulated offline sync as real backend persistence, real offline-first mobile capability, evidence chain-of-custody, or production audit storage.

## Assumptions

- The current demo is useful as a sales-demo and workflow reference, but not as a production technical foundation.
- The first executable artifact remains frontend-only unless the user explicitly asks for production implementation.
- Frontend V2 can use `localStorage` to preserve demo state across refreshes, with a visible reset control and clear demo-only labels.
- Frontend V2 can simulate offline capture with a local outbox and "will sync when connection returns" messages; this is a product-behavior illustration, not a real sync guarantee.
- Frontend V2 should be written so the future backend implementation can reuse the screen flows, state names, IDs, status vocabulary, and validation rules, even if the storage adapter changes from `localStorage` to API calls.
- NCAA regulatory material may be published as multiple parts, technical standards, updates, and effective-date changes rather than as one stable PDF.
- NAMCARS/NAMCATS references in the product must be modeled as configurable regulatory references and versioned source material, not as legal advice.
- USOAP readiness must track PQ/CE edition, applicability, state, deficiency, CAP, evidence, and verification history separately from normal audit findings.
- Any statement such as a single universal "80% EI target" must be verified against current ICAO/GASP source material before it appears as product copy or system logic.
- Offline mobile requirements depend on NCAA device policy, data retention policy, legal acceptance of signatures, file/media policies, and hosting/security decisions.
- AI outputs are draft assistance only unless an authorized human reviews, edits or accepts them, and the system records that decision.

## Product Rules To Preserve

- The core operational loop remains: Audit Plan -> Checklist -> Finding -> CAP -> Evidence -> CAA Review -> Closure -> Dashboard/Report.
- CAP acceptance is not finding closure.
- A finding closes only after required evidence is accepted, verification is completed, or an authorized closure path is explicitly used and audit-logged.
- Auditee users see only their own organization, own findings, own CAP/evidence requests, auditee-visible comments, and closure status.
- Internal CAA notes, inspector workload, other organizations, internal risk scoring, private dashboard data, and enforcement deliberations are never shown to auditees unless a later authorized source changes that rule.
- `Oversight Health Index` is a management indicator only. It must not trigger automatic legal, enforcement, suspension, or closure decisions.
- Regulatory text uses careful language: `reference`, `regulatory reference`, `configured check`, `finding basis`, `expected evidence`, `review result`, and `configured rule`.

## UI Self-Critique For Frontend V2

### Surface Classification

| Surface | Role | Single Job | Product Shape |
|---|---|---|---|
| Safety Intelligence Dashboard | CAA Manager | Decide which risk, delay, or workload issue needs management attention today. | Governance surface |
| Organization Risk Profile | CAA Manager / Inspector | Understand why one organization needs oversight attention before opening or planning an inspection. | Dossier surface |
| Regulatory Library | Admin / Standards | Inspect current references, versions, effective dates, and change status. | Governance surface |
| Dynamic Inspection Package Builder | Inspector / Lead Inspector | Build an inspection package and understand why each checklist question is included. | Editor surface |
| Offline Field Inspection | Inspector | Show what can be collected offline and what still needs sync. | Field queue surface |
| USOAP Readiness Workspace | Manager / Standards | See PQ/CE gaps, missing evidence, and readiness history without claiming official EI outcome. | Reconciliation surface |
| CAP Effectiveness | Inspector / Manager | Decide whether accepted CAP actions worked after closure and whether recurrence needs attention. | Investigation surface |
| AI Assistant Panel | Inspector / Standards | Review AI-generated draft assistance with source references and accept/edit/reject controls. | Assistant/governance surface |
| SSP/NASP Management Dashboard | Manager / SSP team | Track safety objectives, SPI trends, NASP actions, and responsible sections. | Governance surface |

### Signature Element

Frontend V2 should add a `Regulatory Trace` ribbon wherever a checklist question, finding, evidence expectation, USOAP PQ, or AI suggestion appears. The ribbon shows:

- source document and version
- clause or PQ reference
- effective date
- applicability reason
- linked checklist question or expected evidence
- approval state: draft, under review, published, superseded
- demo label: mock/simulated when applicable

This is the product-specific element that separates AviaSurveil360 from a generic dashboard or checklist demo.

### Generic-Template Critique And Revision

Avoid generic chart walls, decorative risk meters, untraceable AI chat, and broad enterprise menu labels. Each new screen must answer one role question and surface the next operational action. Where a dashboard card appears, it must link to the underlying organization, clause, PQ, finding, CAP, evidence item, or action owner.

## Phase 0 - Intake, Current Demo Verification, And Package Truth

**Purpose:** Establish the current demo as the baseline, document its limits, and prepare the V2 demo and production MVP split.

**Files:**
- Read: `README.md`
- Read: `MANIFEST.md`
- Read: `docs/exec-plans/index.md`
- Read: `docs/exec-plans/active/2026-06-14-aviasurveil-demo-only-prototype-plan.md`
- Read: `docs/demo-evidence/BUILD_SUMMARY.md`
- Potentially modify in later execution: `README.md`
- Potentially modify in later execution: `MANIFEST.md`
- Potentially modify in later execution: `docs/demo-evidence/BUILD_SUMMARY.md`

- [ ] **Step 0.1: Verify package reality**

Run:

```bash
rg --files
```

Expected:

```text
README.md
MANIFEST.md
index.html
css/styles.css
js/app.js
js/data.js
js/helpers.js
js/views.js
docs/...
```

Record that README/MANIFEST currently claim Markdown-only packaging while runnable prototype files exist.

- [ ] **Step 0.2: Smoke the existing demo**

Open `index.html` directly in a browser or serve the folder if browser behavior requires it.

Verify:

- role switch works for CAA Manager, CAA Inspector, Auditee, and Admin Preview
- Airline XYZ audit can create `Finding OPS-2026-001`
- CAP acceptance does not close the finding
- evidence acceptance closes the finding
- auditee cannot see internal CAA notes
- Manager dashboard updates after closure

- [ ] **Step 0.3: Record current technical gaps**

Use the NCAA feedback as the acceptance list and record these demo-vs-production gaps:

- no persistent data
- role switch is not authentication
- server-side organization isolation does not exist
- direct finding view authorization must be enforced in production
- checklist execution uses a single runnable template in the demo
- template versioning is text-only in the demo
- finding form `CAP required` and `Evidence required` behavior needs implementation if V2 demo exposes those choices
- CAP revision/superseded model needs real data shape
- risk calculations are demo indicators only
- mobile viewport requires verification and fixes before customer replay
- package docs need to match the actual prototype contents

**Phase 0 Verification:**

- Current demo baseline is explicitly classified as `verified locally`, `blocked`, or `not run`.
- Package mismatch is captured before any stakeholder handoff.
- Active plan index continues to show the existing demo plan as `ready-for-verification` until stakeholder sign-off occurs.

## Phase 1 - Frontend V2 Clickable Demo Additions

**Purpose:** Extend the current static prototype into a richer NCAA-facing platform demo while making every advanced capability visibly mock/simulated.

**Architecture:** Keep HTML, CSS, and Vanilla JavaScript. Use `localStorage` only behind a small demo data-service/storage boundary for demo persistence and simulated offline/outbox behavior. Do not add backend, database, real authentication, real upload, real AI service, real regulatory ingestion, real notifications, or framework migration.

**Files:**
- Modify: `index.html`
- Modify: `css/styles.css`
- Modify: `js/data.js`
- Modify: `js/helpers.js`
- Modify: `js/views.js`
- Modify: `js/app.js`
- Modify: `docs/demo-evidence/BUILD_SUMMARY.md`
- Modify if package docs are part of this execution: `README.md`
- Modify if package docs are part of this execution: `MANIFEST.md`

### Backend-Ready Demo Architecture

Frontend V2 should stay static, but should be structured as if a backend will replace the demo store later:

- Keep one canonical in-memory `state` object loaded from seed data plus saved demo state.
- Add small storage/data-service helpers rather than calling `localStorage` from view rendering code.
- Use stable entity-shaped records that resemble future API resources:
  - `organization.id`
  - `audit.id`
  - `checklistTemplateVersion.id`
  - `checklistResponse.id`
  - `finding.id`
  - `capRevision.id`
  - `evidenceVersion.id`
  - `offlineOutboxItem.id`
  - `aiSuggestion.id`
- Keep status values as explicit strings/enums, not UI labels.
- Keep transition functions centralized, for example `submitCap`, `acceptCap`, `submitEvidence`, `acceptEvidence`, `queueOfflineAction`, and `markOutboxSynced`.
- Make UI views read from selector/helper functions such as `visibleFindings`, `findingById`, `offlineOutboxItems`, and `regulatoryTraceForQuestion`.
- Put demo persistence behind functions with backend-replaceable names:

```javascript
var DEMO_STORAGE_KEY = 'aviasurveil360:v2-demo-state';

function loadDemoState() {
  var raw = window.localStorage.getItem(DEMO_STORAGE_KEY);
  return raw ? JSON.parse(raw) : null;
}

function saveDemoState(nextState) {
  window.localStorage.setItem(DEMO_STORAGE_KEY, JSON.stringify(nextState));
}

function clearDemoState() {
  window.localStorage.removeItem(DEMO_STORAGE_KEY);
}

function persistAfterAction() {
  saveDemoState(state);
}
```

Later production code should be able to replace those functions with API-backed calls such as `GET /demo-or-api-state`, `POST /findings`, `POST /cap-revisions`, `POST /evidence-versions`, or `POST /sync/outbox` without rewriting the screen flow.

Do not let the demo architecture imply production readiness. The backend-ready boundary is only to reduce future rework.

### V2 Screen Inventory

1. **Safety Intelligence Dashboard**
   - Role: CAA Manager.
   - Shows risk alerts, overdue CAPs, repeat finding trends, section workload, plan slippage, and recommended management action.
   - Every card links to an organization, finding, CAP, audit, or section owner.
   - Label risk as `mock risk indicator`, not enforcement.

2. **Organization Risk Profile**
   - Role: CAA Manager / Inspector.
   - Shows risk score, risk drivers, previous findings, CAP performance, fleet/management changes, occurrence trend placeholder, and audit history.
   - Shows why the organization appears in a risk alert.

3. **Regulatory Library**
   - Role: Admin / Standards.
   - Shows NAMCARS/NAMCATS-style document parts, clause references, versions, effective dates, status, supersedes/superseded-by links, and mock change history.
   - Uses `regulatory reference` and `configured rule`; no legal-advice language.

4. **Dynamic Inspection Package Builder**
   - Role: Inspector / Lead Inspector.
   - Shows selected organization, audit type, domain, risk focus, applicable regulatory references, proposed checklist questions, expected evidence, and "why this question is included".
   - Output is a `mock inspection package`; no official checklist publication.

5. **Offline Field Inspection View**
   - Role: Inspector.
   - Shows simulated offline status, checked-out audit package, local changes, attachment queue, photos/videos/audio/document placeholders, signature placeholder, sync queue, and conflict warning examples.
   - Allows a demo-only offline toggle or browser online/offline event to show: `Internet unavailable - saved locally. It will sync automatically when connection returns.`
   - Stores unsynced demo actions in a `localStorage` outbox until the connection is simulated as restored.
   - Label as `offline simulated`.

6. **USOAP Readiness Workspace**
   - Role: Manager / Standards.
   - Shows PQ, CE, audit area, applicability, readiness status, missing evidence, linked CAP/evidence, verification history, and readiness trend.
   - Does not claim official EI score improvement.

7. **CAP Effectiveness Screen**
   - Role: Inspector / Manager.
   - Shows root cause quality, CAP revision history, effectiveness verification, recurrence indicator, reopen path, and post-closure review result.
   - Keeps CAP acceptance separate from finding closure and post-closure effectiveness.

8. **AI Inspector Assistant Panel**
   - Role: Inspector / Standards.
   - Shows source-referenced draft finding language, risk statement draft, historical comparison, checklist question suggestion, and Accept/Edit/Reject controls.
   - Label every AI suggestion as `AI-generated draft - requires authorized review`.

9. **SSP/NASP Management Dashboard**
   - Role: Manager / SSP team.
   - Shows national safety objectives, SPI trends, NASP actions, responsible section, target date, status, and linked oversight findings where configured.
   - Uses careful language: `supports monitoring`, not automatic state safety determination.

### V2 Mock Data Additions

Add or extend mock data structures in `js/data.js` during implementation:

```javascript
var SEED_REGULATORY_DOCUMENTS = [
  {
    id: 'REG-NAMCARS-OPS-2026',
    family: 'NAMCARS',
    title: 'Flight Operations Requirements',
    version: '2026 mock edition',
    status: 'Published',
    effectiveDate: '2026-01-01',
    clauses: [
      {
        id: 'OPS-TRG-4.2',
        reference: 'OPS 4.2',
        title: 'Crew training records',
        status: 'Published',
        applicability: 'Air operator with scheduled passenger operations',
        expectedEvidence: ['Training record sample', 'Training matrix', 'Responsible manager attestation']
      }
    ]
  }
];

var SEED_RISK_PROFILES = [
  {
    orgId: 'ORG-XYZ',
    score: 74,
    band: 'Needs Attention',
    drivers: ['Repeat training records finding', 'CAP due soon', 'Flight Operations audit scheduled'],
    recommendedAction: 'Prioritize Flight Operations checklist focus on training records and CAP effectiveness.'
  }
];

var SEED_USOAP_READINESS = [
  {
    pqId: 'PQ-OPS-MOCK-001',
    criticalElement: 'CE-7',
    auditArea: 'OPS',
    applicability: 'Applicable in mock demo',
    readinessStatus: 'Missing evidence',
    linkedEvidence: [],
    note: 'Mock PQ readiness record for demo only; not an official ICAO assessment.'
  }
];

var DEMO_PERSISTENCE_CONFIG = {
  storageKey: 'aviasurveil360:v2-demo-state',
  persists: [
    'created findings',
    'CAP submissions',
    'mock evidence filenames',
    'AI accept/edit/reject decisions',
    'selected dashboard filters',
    'offline outbox items'
  ],
  resetLabel: 'Reset demo data',
  disclaimer: 'Stored in this browser for demo only. No backend, real file storage, or production audit trail.'
};

var SEED_OFFLINE_OUTBOX = [
  {
    id: 'OUTBOX-MOCK-001',
    type: 'evidence-upload',
    status: 'Waiting for connection',
    message: 'Internet unavailable - saved locally. It will sync automatically when connection returns.',
    fileName: 'Training_Record_Photo.jpg',
    findingId: 'OPS-2026-001',
    createdAt: '2026-06-23T10:20:00'
  }
];
```

### V2 Demo Persistence And Simulated Offline Sync

Frontend V2 should support light demo persistence without a backend:

- Keep `localStorage` access inside storage/data-service helpers; views and modals should call action functions, not storage APIs directly.
- Save the current demo state to `localStorage` after meaningful actions:
  - role-independent created findings
  - CAP submission content
  - mock evidence filename and status
  - AI suggestion decision: accepted, edited, or rejected
  - selected filters
  - simulated offline outbox items
- Load saved demo state on page start if the storage key exists.
- Keep the existing reset control and make it clear that reset clears the browser's demo state.
- Show a small persistent label such as `Frontend-only demo - saved in this browser`.
- Do not store real files; keep mock evidence as filename, size, type, and status only.
- Do not store secrets, credentials, personal identifiers beyond mock demo data, or real regulatory material.
- Use API-shaped records and stable IDs so later backend endpoints can accept the same conceptual payloads.

Simulated offline sync should work like this:

1. Inspector toggles `Simulate offline` or the browser reports offline.
2. CAP/evidence/checklist notes can still be entered in demo state.
3. The action appears in an `Offline outbox` with status `Waiting for connection`.
4. The UI shows `Internet unavailable - saved locally. It will sync automatically when connection returns.`
5. When the connection is simulated as restored, the outbox item status changes to `Synced to demo state`.
6. The audit log records a demo event such as `Offline item synced (demo)`.
7. The outbox item shape stays close to a future sync API payload: action type, entity type, entity ID, payload summary, created timestamp, status, and last sync attempt.

This is a stakeholder demonstration of expected behavior only. It is not a production offline-first sync engine, encrypted local store, or evidence chain-of-custody mechanism.

### V2 Interaction Requirements

- [ ] Manager opens Safety Intelligence and navigates to Airline XYZ risk profile.
- [ ] Inspector opens Dynamic Inspection Package Builder and sees why the crew training question is included.
- [ ] Admin opens Regulatory Library and sees clause version/effective date/change history.
- [ ] Inspector opens Offline Field Inspection, toggles simulated offline mode, creates a mock evidence action, sees it saved in the offline outbox, then toggles simulated online mode and sees it marked synced to demo state.
- [ ] Manager opens USOAP Readiness and sees missing evidence without EI overclaiming.
- [ ] Inspector opens AI Assistant, edits a draft suggestion, and records an accept/edit/reject choice in mock state.
- [ ] Manager opens SSP/NASP dashboard and sees objective/SPI/action ownership.
- [ ] Browser refresh preserves created finding, CAP, mock evidence filename, AI decision, selected filters, and offline outbox state through `localStorage`.

### V2 Verification

Run or manually perform:

```bash
open index.html
```

Then verify:

- no console errors during role switch and navigation
- 390px mobile viewport has no horizontal overflow
- every new screen has one main purpose and a primary next action
- every advanced feature is labeled `mock`, `simulated`, `AI-generated draft`, or `not a legal decision` as appropriate
- `localStorage` demo persistence survives refresh and can be cleared through reset
- simulated offline status shows saved-locally and synced-to-demo-state transitions without claiming production sync
- storage calls are isolated behind helper/data-service functions so future backend integration can replace the persistence layer
- entity IDs and status values are stable and not derived only from visible UI labels
- auditee views still do not reveal internal notes, other organizations, internal risk scoring, inspector workload, or AI/regulatory governance data not meant for auditees
- existing demo lifecycle still works end to end

## Phase 2 - Production MVP Architecture And Scope

**Purpose:** Define the real production foundation, using the current demo as UX reference only.

**Architecture Recommendation:** Start with a modular monolith, not microservices. Keep module boundaries clear in code and database schema, but deploy as one system until scale, compliance, or team topology justifies extraction.

### Production Web Frontend

Recommended characteristics:

- TypeScript component architecture using React/Next.js or Vue/Nuxt after engineering selection.
- API-backed state management; no global client-side source of truth.
- Schema-validated forms for audit, checklist, finding, CAP, evidence, regulatory references, and admin configuration.
- Permission-aware routing and server-confirmed authorization on every record view.
- English-canonical product copy with optional localization path.
- WCAG accessibility pass.
- Responsive desktop/tablet-first workbench design with mobile support where appropriate.
- Design system based on task queues, dossiers, lifecycle panels, and governance review surfaces.

### Production Backend

Recommended characteristics:

- Modular monolith API, REST or GraphQL after API design.
- PostgreSQL as canonical relational store.
- S3-compatible object storage for evidence and report artifacts.
- Background jobs for notifications, document generation, retention tasks, and regulatory change impact processing.
- Full-text search for regulatory references, findings, CAPs, evidence metadata, and knowledge base material.
- PDF/document generation for reports and packages.
- Append-only audit events for critical actions.
- Versioned workflow/state machine for audit, finding, CAP, evidence, template, PQ readiness, and regulatory publication states.
- Integration adapters for future NCAA systems, occurrence feeds, identity providers, email/SMS, and document systems.

### Production Security Baseline

Production MVP must include:

- real user identity and organization identity
- MFA for internal users and a policy decision for external users
- optional SSO if NCAA identity architecture requires it
- RBAC plus section/domain/assignment checks
- server-side auditee organization isolation
- object-level authorization for every audit, finding, CAP, evidence, report, regulatory record, and PQ record
- encryption in transit and at rest
- evidence file hash, metadata, version history, and chain-of-custody fields
- virus/malware scanning policy for uploads
- append-only critical action audit events
- backup, disaster recovery, retention, and legal hold policies
- bulk export and report download controls
- AI usage audit logging

### Production Core Modules

| Module | Production MVP Need |
|---|---|
| Identity and Access | Users, roles, org membership, MFA/SSO policy, assignments, permissions |
| Organization Registry | Organizations, certificates/approval references if configured, contacts, risk profile summary |
| Surveillance Planning | Annual/ad hoc audits, assignments, scope, schedule, preparation package |
| Audit Execution | Audit record, checklist run, notes, attachments, status history |
| Checklist Templates | Versioned templates, published snapshots, effective dates, old audit lock-in |
| Findings | Severity, basis, due date, owner, next action, status history, internal/external notes |
| CAP | Revision history, superseded versions, root cause, corrective/preventive action, target date |
| Evidence | Versioned files, review decisions, metadata, hash, retention, access control |
| Auditee Portal | External login, organization isolation, CAP/evidence/messages/report downloads |
| Notifications | Due Date, Due Soon, Overdue, request-more-information, review queues |
| Reports | Audit report, finding closure report, management summaries, export controls |
| Audit Log | Append-only event stream for critical actions and admin/config changes |

### Production MVP Verification

Before claiming production MVP readiness:

- automated authorization tests prove auditee organization isolation on list and direct-record access
- workflow tests prove CAP acceptance does not close findings
- template versioning tests prove old audits keep old template snapshots
- evidence tests prove previous versions are preserved and review decisions are recorded
- audit-log tests prove every critical transition creates an immutable event
- security review covers authentication, permissions, uploads, exports, backups, and audit events
- report generation is source-backed and not a mock preview

## Phase 3 - Regulatory Intelligence, NAMCARS/NAMCATS, USOAP, SSP/NASP/SPI

**Purpose:** Model regulatory and state safety oversight domains without making uncontrolled legal or ICAO claims.

### Regulatory Intelligence Domain Model

Add these production entities in later design:

- `RegulatoryDocument`
- `RegulatoryDocumentVersion`
- `RegulatoryPart`
- `RegulatoryChapter`
- `RegulatorySection`
- `RegulatoryClause`
- `RegulatoryEffectiveDate`
- `RegulatoryPublicationStatus`
- `RegulatorySupersession`
- `ApplicabilityRule`
- `ChecklistQuestionMapping`
- `FindingMapping`
- `ExpectedEvidence`
- `RegulatoryChangeImpactAssessment`
- `RegulatoryApprovalDecision`
- `RegulatorySourceAttachment`

### Regulatory Change Workflow

1. Import or register source material into a draft regulatory workspace.
2. Parse or manually structure parts, sections, and clauses.
3. Compare against prior version and mark changed clauses.
4. Identify impacted checklist questions, open/planned audits, findings, report templates, expected evidence, and training/communication needs.
5. Generate draft checklist question changes.
6. Require Standards/Legal/authorized inspector review.
7. Publish a new checklist template version only after approval.
8. Lock already-running audits to their selected template snapshot unless an authorized migration decision is recorded.

### USOAP PQ/CE Domain Model

Add these production entities in later design:

- `UsoapEdition`
- `UsoapCriticalElement`
- `UsoapAuditArea`
- `UsoapProtocolQuestion`
- `UsoapApplicabilityDecision`
- `UsoapReadinessStatus`
- `UsoapDeficiency`
- `UsoapCorrectiveAction`
- `UsoapEvidencePackage`
- `UsoapVerificationEvent`
- `UsoapReadinessSnapshot`

USOAP readiness must track PQ version, applicability, readiness state, deficiency, CAP, evidence, and verification history. The product may support EI readiness analysis, but it must not guarantee any EI score or official ICAO outcome.

### SSP/NASP/SPI Domain Model

Add these production entities in later design:

- `SafetyObjective`
- `NationalAviationSafetyPlanAction`
- `SafetyPerformanceIndicator`
- `SafetyPerformanceTarget`
- `OccurrenceReference`
- `ResponsibleSection`
- `MonitoringPeriod`
- `SspReviewDecision`
- `LinkedOversightFinding`

SSP/NASP/SPI dashboards must show monitoring and management support. They must not claim automatic state safety performance conclusions without configured source data, authority review, and evidence.

### Regulatory Verification

Before regulatory intelligence is demoed externally or implemented in production:

- source owners confirm which NCAA documents are in scope
- current document versions and effective dates are verified from official sources
- Standards/Legal confirms terminology and approval workflow
- PQ/CE edition and access permissions are confirmed
- GASP/EI target language is reviewed against current source material
- product copy is checked for no legal-advice, enforcement, suspension, or guaranteed EI language

## Phase 4 - Offline Mobile Inspection Discovery And Later Implementation

**Purpose:** Avoid choosing PWA or native mobile before field policy, security, device, and sync requirements are known.

### Discovery Questions

- Which devices will inspectors use: managed tablets, personal phones, laptops, or mixed devices?
- Must the app work in aircraft, remote airports, hangars, and low-connectivity field environments?
- What evidence media is allowed: photos, video, audio, scanned documents, signatures, GPS, timestamps?
- What data must be available offline, and how long may it remain on-device?
- What encryption and remote wipe/session revoke policies are required?
- What is the legal status of digital signatures?
- How should conflicts be resolved if two inspectors edit the same record offline?
- What attachment sizes and network conditions must be supported?

### Later Offline Architecture Options

Decision paths:

- PWA with service worker, Cache Storage, IndexedDB, and foreground sync controls.
- Native or cross-platform client if camera, signature, encrypted storage, device management, or background sync requirements exceed reliable browser support.
- Hybrid approach where desktop web handles planning/review and mobile client handles checked-out inspections.

### Required Offline Capabilities

Production offline work must include:

- inspection package check-out
- encrypted local data store
- local attachment queue
- temporary offline IDs
- outbox/inbox synchronization
- revision and conflict detection
- resumable/chunked attachment upload
- visible unsynced state indicators
- photo, video, audio, document, and signature capture if approved
- optional GPS/timestamp only if policy permits
- device loss and session revoke support

### Offline Verification

Before claiming offline readiness:

- airplane-mode field test passes for a checked-out package
- sync resumes after network return
- conflicts are detected and displayed
- attachments survive app restart before sync
- unauthorized user cannot read local data
- lost-device/session revoke process is tested

## Phase 5 - AI Inspector Assistant Governance

**Purpose:** Use AI as reviewable assistance, not an authority.

### AI Assistant Allowed Tasks

- search source-referenced regulatory material
- summarize relevant clauses with citations
- draft finding language from inspector-selected facts
- draft risk statements
- compare a finding to historical findings and CAPs
- suggest checklist question updates for human review
- summarize CAP/evidence review history
- flag possible recurrence patterns

### AI Assistant Prohibited Tasks

AI must not:

- publish official checklist templates
- issue official findings by itself
- accept or reject CAP/evidence by itself
- close findings by itself
- create enforcement decisions
- determine certificate action
- provide legal advice
- claim official USOAP/EI results
- hide or omit source references for regulatory suggestions

### AI Governance Data Model

Add these production entities in later design:

- `AiSuggestion`
- `AiSuggestionSourceReference`
- `AiSuggestionDecision`
- `AiPromptAuditEvent`
- `AiModelConfiguration`
- `AiPolicyVersion`
- `AiReviewQueueItem`

Each AI suggestion must store:

- user request context
- source references used
- generated draft
- confidence or limitation notes if exposed
- reviewer identity
- decision: accepted, edited, rejected
- final human-authored output if accepted or edited
- audit event timestamp

### AI Verification

Before external demo:

- every AI panel visibly says `AI-generated draft - requires authorized review`
- every regulatory answer includes source references or says source unavailable
- Accept/Edit/Reject controls are visible
- no AI action changes official status without a human action

Before production:

- AI output is logged
- source retrieval is permission-scoped
- prompt/output retention is approved
- sensitive data handling is reviewed
- evaluation set covers hallucination, missing citation, wrong organization, and unauthorized source access

## Dependencies

- NCAA process discovery across PEL, OPS, AIR, AGA, ANSSO, AVSEC, SSP, and other relevant sections.
- Authority matrix for who can issue findings, accept CAPs, accept evidence, close findings, approve templates, publish regulatory mappings, and approve AI-suggested changes.
- Official regulatory document access and owner assignment.
- USOAP PQ/CE edition/access decision.
- SSP/NASP/SPI source data availability.
- Hosting, data residency, retention, backup, disaster recovery, and legal hold decisions.
- Identity provider, MFA/SSO policy, and external auditee onboarding policy.
- File storage, scanning, retention, and evidence chain-of-custody policy.
- Mobile device policy and digital signature legal/policy decision.
- AI policy: approved models, source corpus, retention, review, and audit requirements.

## Risks

| Risk | Impact | Mitigation |
|---|---|---|
| Demo scope becomes production promise | Stakeholders may expect unavailable security/offline/regulatory behavior | Label V2 features as mock/simulated and keep production MVP plan separate |
| Regulatory overclaiming | Legal and trust risk | Use careful wording and require Standards/Legal approval before official templates |
| Auditee data leakage | High security risk | Production requires server-side object authorization and tests for direct access |
| Template version gaps | Audits may use wrong regulatory basis | Production checklist versions require immutable published snapshots |
| USOAP/EI overclaiming | Misleading management reporting | Track readiness and evidence, not guaranteed official score outcomes |
| Offline sync conflicts | Lost or inconsistent field records | Design revision/conflict model before mobile implementation |
| Evidence integrity gaps | Weak chain-of-custody | Use object storage metadata, file hash, versioning, malware scanning, and audit events |
| AI hallucination or unauthorized action | Regulatory and workflow risk | Require citations, human review, decision logging, and prohibited-action gates |
| Mobile viewport regression in demo | Weak stakeholder experience | Include 390px responsive smoke verification in V2 demo |
| Package docs mismatch | Handoff confusion | Update README/MANIFEST when prototype contents are part of the deliverable |

## Ownership Boundaries

- Product owner defines demo story, phase order, acceptance criteria, and stakeholder feedback questions.
- CAA section owners validate process flows and section-specific requirements.
- Standards/Legal owns regulatory source approval, clause interpretation, template publication rules, and careful wording.
- Engineering owns production architecture, APIs, data model, authorization, workflow engine, storage, sync, and tests.
- Security owns authentication, MFA/SSO, permissions, encryption, upload scanning, audit trail, backups, exports, and AI data handling review.
- UX/design owns task-first surfaces, role-specific navigation, accessibility, and responsive behavior.
- AI governance owner approves model usage, source corpus, retention, review queues, and audit logging.
- The current static prototype remains a demo artifact; it does not own production backend, security, persistence, offline, regulatory, or AI correctness.

## Verification Plan

### Plan Verification

- `docs/exec-plans/active/2026-06-23-ncaa-platform-v2-and-mvp-plan.md` exists.
- `docs/exec-plans/index.md` has one active row for this plan.
- The existing demo-only plan remains `ready-for-verification` until stakeholder sign-off.
- This plan includes objective, scope, assumptions, phases, verification, risks, dependencies, ownership boundaries, explicit out-of-scope items, and Execution Prompt.

### Frontend V2 Verification

- Existing lifecycle still works end to end.
- All nine new V2 screens are reachable by role-appropriate navigation.
- Advanced features are labeled mock/simulated/AI draft/not legal decision.
- Auditee isolation remains visibly preserved in all auditee views.
- 390px mobile viewport has no horizontal overflow.
- Browser console has no errors during the demo path.

### Production MVP Verification

- Authorization tests cover list and direct-object access.
- Workflow tests cover every allowed and blocked transition.
- Evidence/versioning tests prove superseded versions are preserved.
- Regulatory template tests prove audit template snapshots are immutable.
- Audit event tests cover critical actions.
- Security review covers uploads, exports, backups, retention, and AI logging.

### Regulatory/USOAP/SSP Verification

- Official source document versions are confirmed before use.
- PQ/CE edition and applicability rules are owner-approved.
- Product copy avoids unverified EI target claims.
- Change impact assessment lists impacted clauses, checklist questions, planned/open audits, findings, evidence expectations, and report templates.

### Offline Mobile Verification

- Demo-only offline simulation shows local outbox behavior, but is not counted as production offline readiness.
- Offline package check-out and airplane-mode editing pass on target devices before any production offline readiness claim.
- Local encrypted store and session revoke behavior pass security review.
- Sync queue, conflict detection, and attachment upload recovery pass test cases.

### AI Verification

- AI suggestions include source references or explicit source-unavailable limits.
- Accept/Edit/Reject decisions are audit logged.
- AI cannot publish official regulatory, finding, CAP, evidence, closure, enforcement, or USOAP outputs by itself.

## Index Next Todo

The next concrete todo for this plan is:

> Frontend V2 is implemented and verified locally. Next: stakeholder review/sign-off, then choose the follow-up track: README/MANIFEST package-truth cleanup, production MVP architecture execution, regulatory owner packet, offline mobile discovery, or AI governance discovery.

## Execution Prompt

```text
Execute docs/exec-plans/active/2026-06-23-ncaa-platform-v2-and-mvp-plan.md for AviaSurveil360.

Respect AGENTS.md. Use docs/exec-plans/ as the source of truth for plan tracking. Do not create, switch, rename, delete, or otherwise perform branch operations. Do not treat the current Vanilla JavaScript demo as production architecture.

Start with Phase 0:
1. Verify the current repo/package reality with rg --files.
2. Open or serve index.html and smoke the existing demo lifecycle.
3. Record the current demo/package gaps: no persistence, no real auth, no server-side organization isolation, direct finding authorization gap, single runnable checklist, text-only template versioning, CAP/evidence required options needing behavior if exposed, no CAP revision model, mock risk indicator, mobile overflow risk, and README/MANIFEST package mismatch.
4. Confirm the owner questions for NCAA regulatory documents, USOAP PQ/CE edition/access, SSP/NASP/SPI source data, mobile device policy, digital signature policy, and AI governance.
5. Produce the detailed backend-ready Frontend V2 screen/data inventory before implementation, including localStorage demo persistence for created findings, CAP submissions, mock evidence filenames, AI decisions, filters, and simulated offline outbox items.

Then implement only the chosen next phase:
- If building Frontend V2, keep HTML/CSS/Vanilla JS, mock data, client-side state, localStorage demo persistence, and simulated offline outbox behavior only. Make it backend-ready by isolating localStorage behind storage/data-service helper functions, using API-shaped records, stable IDs, explicit status values, and centralized transition functions. Add Safety Intelligence Dashboard, Organization Risk Profile, Regulatory Library, Dynamic Inspection Package Builder, Offline Field Inspection, USOAP Readiness Workspace, CAP Effectiveness, AI Assistant Panel, and SSP/NASP Management Dashboard. Label advanced features as mock, simulated, AI-generated draft, or not a legal decision. Preserve CAP accepted != finding closed and auditee isolation. The offline demo may show "Internet unavailable - saved locally. It will sync automatically when connection returns.", but must not claim production offline sync or evidence chain-of-custody.
- If planning Production MVP, create a separate implementation plan for the secure API-backed system: TypeScript web frontend, modular monolith API, PostgreSQL, object storage, workflow engine, append-only audit events, real authentication, server-side authorization, evidence versioning, reporting, notifications, and security review.
- If planning Regulatory Intelligence, create a source-bound model for document versions, clauses, effective dates, applicability rules, checklist mappings, expected evidence, change impact, USOAP PQ/CE readiness, SSP/NASP/SPI, and human approval workflow.
- If planning Offline Mobile, complete device/security/sync discovery before selecting PWA or native technology.
- If planning AI Assistant, implement only source-referenced draft assistance with Accept/Edit/Reject controls and audit logging; do not allow AI to publish official regulatory, finding, CAP, evidence, closure, enforcement, or USOAP decisions.

Verification must keep demo-only claims separate from production readiness. Do not claim legal validity, USOAP EI improvement, enforcement readiness, production security, evidence repository completeness, offline reliability, or AI correctness without matching implementation and evidence.

When work changes plan status, update docs/exec-plans/index.md with the actual status and the single next concrete todo.
```
