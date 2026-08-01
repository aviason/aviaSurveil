# Department And General Manager Workspaces Design

**Status:** Approved for planning on 10 July 2026
**Build boundary:** Frontend-only clickable demo
**Canonical organization name:** `Fly Namibia`

## Objective

Complete the task-based Department Manager and General Manager workspaces that
match the supplied reference screens while remaining consistent with the
current AviaSurveil360 static demo:

1. `Findings Review`
2. `Inspection Team`
3. `Reports Approval`
4. `CAP Monitoring`
5. `Checklist Management`
6. `Risk Dashboard`
7. Department Manager Dashboard and restricted navigation
8. General Manager Dashboard and restricted navigation

The workspaces must use working controls and shared browser-local demo state.
They must not introduce a backend, database, API, real authentication, real
file storage, real notification delivery, or a production reporting engine.

## Governed checklist authority

For the candidate-only governed checklist workflow, the currently assigned
responsible Department Manager may technically review a complete candidate only
within the manager's department scope. Technical approval and publication are
two separate recorded decisions; publication creates an immutable version only
after the required technical approval. Admin and generation providers can
create or edit candidates but cannot technically approve or publish them.

For governed AGA intake, the manager sees only the server-derived required
Department/unit scope and immutable Draft digest. The manager cannot repair a
missing owner, supply authority by selecting a source, or convert an archive
receipt into a published checklist. `SOURCE_MAPPING_REQUIRED`, stale authority,
incomplete required-owner resolution, or an unavailable scoped assignment blocks
technical approval. Publication is a separate decision and remains distinct
from computed Audit-package eligibility.

## Naming Contract

- Display the service provider/operator as `Fly Namibia` everywhere in the UI,
  report content, stakeholder-facing documentation, and generated demo PDF
  filenames.
- Use `Operator / Service Provider` when the organization type needs to be
  shown.
- Do not display `Air Namibia`, `FlyNamibia`, or `FlyNamibia (Pty) Ltd` as
  alternate organization names.
- Existing internal identifiers such as `ORG-XYZ` may remain unchanged because
  they are not user-facing names.

## Recommended Architecture

Extend the existing HTML, CSS, and Vanilla JavaScript application. The new
manager screens should project the existing mutable demo state instead of
introducing unrelated hard-coded screen datasets.

The core state boundaries are:

- audits and inspections provide organization, dates, department, status, lead
  inspector, and team context;
- findings provide severity, lifecycle status, owner, next action, Due Date,
  CAP state, and evidence state;
- internal users provide Department Manager reporting lines, inspector
  department, email, role, and active status;
- report artifacts preserve Preliminary and Final Reports as separate records,
  each with its own version, status, submission time, comments, attachments,
  approval history, and generated-PDF metadata;
- manager-only UI state stores selected rows, active tabs, filters, open action
  menus, draft comments, and split-pane visibility;
- CAP Monitoring projects CAPs from findings, owners, Due Dates, progress,
  evidence, updates, documents, and history instead of duplicating CAP records.
- Checklist Management stores packages, versions, sections, questions, status,
  and browser-local manager mutations as a focused demo governance model.
- General Manager summary state projects department, Final Report approval, CAP,
  and risk data from shared records; it must not fork a conflicting dashboard
  dataset.

Existing saved browser state must merge safely with the new defaults. Newly
introduced collections and UI state must receive defensive defaults when older
demo state is loaded.

## Department Manager Navigation

The Department Manager sidebar exposes only these entries:

- `Dashboard`
- `Audits`
- `Inspection Team`
- `Findings Review`
- `Reports Approval`
- `Risk Dashboard`
- `CAP Monitoring`
- `Checklist Management`

`Findings Review` replaces the manager-facing `Open Findings` label while
retaining access to status filters. `Reports Approval` combines the current
manager `Audit Reports` and `Preliminary Reports` entries into one approval
queue. Older manager-only Planning, Calendar, Documents, Corrective Actions,
Settings, duplicate checklist, organization, and analytics sidebar entries are
not shown. Compatibility routes may remain internal when required by existing
demo flows, but they are not Department Manager navigation items.

The General Manager sidebar exposes only:

- `Dashboard`
- `Report Approvals`
- `Departments`
- `Risk Dashboard`
- `Settings`

## Findings Review

### Purpose

Answer: “Which inspections under my responsibility have findings, and what
needs management attention?”

### Layout

Use a responsive master-detail workspace:

- left: inspection search, filters, KPI summary, and inspection table;
- right: the selected inspection's findings dossier;
- mobile: the list renders first and the selected dossier opens as the next
  stacked section, without horizontal page overflow.

The initial selection is the active Fly Namibia inspection. Fly Namibia must be
visible on first load; it must not depend on completing the live Cabin
Inspection demo first. A small set of pre-existing demo findings may be attached
to the inspection so the manager can review meaningful severity and status
counts without pre-seeding the live hero finding `CAB-2026-001`.

### Inspection list

Each row shows:

- Audit ID
- Organization
- Audit Date
- Team Leader
- Status
- total findings and severity breakdown
- action to select/open the inspection

Search and status/date filters must update the visible list. Export List must
generate a browser-side demo file instead of showing a success toast only.

### Selected inspection dossier

Header fields:

- Audit ID and `Fly Namibia`
- Audit Date
- Team Leader
- Department
- Audit Phase
- Status

Tabs:

1. `Findings Overview`
2. `Findings List`
3. `By Department`
4. `By Level`

The overview uses compact counts and small decision-focused visual summaries.
It must not become a wall of charts. `View All Findings` opens the detailed
finding list. `View Full Report` opens the related report artifact in Reports
Approval.

Every finding shown in a list or detail includes the current owner, next
action, Due Date, status, severity, related audit, and organization. CAP
acceptance remains distinct from closure, and evidence version history remains
preserved.

## Inspection Team

### Purpose

Answer: “Which inspectors report to me, which inspection teams are active, and
what team action is needed?”

### Scope

Only the Department Manager's own reporting-line inspectors and the inspection
teams using those inspectors are shown. Other managers' private workload data
is out of scope for this page.

### Layout

Use the same master-detail pattern:

- summary metrics for total teams, active teams, upcoming inspections, and
  completed inspections this month;
- search and Department, Status, and Audit Date filters;
- inspection-team table on the left;
- selected team detail on the right;
- mobile stacked layout with the selected detail reachable without clipping.

### Action menu behavior

Clicking the row ellipsis opens a real menu anchored to that row. Selecting
`View Team Details` selects the audit and displays the reference-style detail
pane. Other visible menu actions must have a meaningful result:

- `Edit Team`, `Add Inspector`, `Remove Inspector`, and `Change Lead Inspector`
  open focused forms and update browser-local demo state after confirmation;
- `Update Schedule` changes the visible demo date range and records history;
- `View Assignment Package` opens a document preview;
- `View Audit Details` navigates to the existing audit detail;
- `Send Message to Team` opens a compose form and records a mock in-app message;
- `Download Team Assignment` generates a client-side demo PDF;
- `View Activity Log` opens the History tab.

If an action cannot be supported meaningfully in the static demo, it must not
be shown as an enabled control.

### Detail tabs

- `Overview`
- `Team Members`
- `Assignments`
- `Documents`
- `History`

The overview shows team counts and member rows with role, name, department,
email, and status. Team notes are editable browser-local demo text. Attachments
store and display selected filenames only; no file bytes are uploaded or
stored.

## Reports Approval

### Purpose

Answer: “Which Preliminary or Final Reports require my decision?”

### Report artifacts

Preliminary and Final Reports are separate, preserved artifacts. A Preliminary
Report must not be mutated into a Final Report. Each artifact has:

- Report ID and Audit ID
- Organization
- report type and version
- Lead Inspector
- submitted date/time
- current owner and status
- summary, findings, attachments, comments, and history
- manager decision and decision timestamp when present

The initial queue includes a Fly Namibia Preliminary Report pending Department
Manager approval and representative Final Report states for filtering and
review.

### Queue and dossier

Top counters and filters:

- All Reports
- Preliminary Reports
- Final Reports
- Pending My Approval
- Revision Requested
- Approved

Selecting a row opens the report dossier on the right. The dossier provides:

- report summary;
- findings breakdown;
- attachments;
- manager comments;
- approval history;
- a full report preview;
- PDF download options.

### Approval rules

Both report types require a Department Manager decision.

Preliminary Report:

1. Lead Inspector submits the Preliminary Report.
2. Department Manager approves, requests revision, or returns the report.
3. Department Manager approval always forwards the exact Preliminary Report to
   General Manager intermediate review.
4. General Manager may return it or forward it to Executive Director, but
   cannot issue, share, sign, or lock it.
5. Executive Director approval issues and locks the controlled demo copy for
   the report's Service Provider organization.
6. `capRequired` changes only the recipient action after issue; it never
   changes or bypasses the approval chain.

Final Report:

1. Lead Inspector submits the Final Report after the configured CAP/evidence
   preparation stage.
2. Department Manager approves, requests revision, or returns the report.
3. Department Manager approval forwards the Final Report to the configured
   General Manager intermediate review stage.
4. General Manager may return the report or advance it to Executive Director;
   GM does not issue, sign, or lock it.
5. Only an eligible Executive Director approval may add the mock approval mark,
   issue the report, and lock it. Neither Department Manager nor GM approval
   alone performs those actions.

Request Revision and Return Report require a manager comment. Decisions update
the visible status, approval history, current owner, and mock in-app
notification. No real email or external notification is sent.

### PDF downloads

The Download PDF menu supports:

- Preliminary Report PDF
- Final Report PDF when a Final Report is selected
- Executive Summary PDF

The existing browser-side PDF generator should be generalized to accept report
content and filename inputs. The generated file must have an
`application/pdf` MIME type, a valid PDF header, and a Fly Namibia filename.
This is demo document generation only; it is not a production reporting,
document-storage, signature, or records-management service.

## Department Manager Dashboard

The first manager screen is a compact operational summary, not a generic chart
wall. It shows Total Audits, Reports Awaiting Approval, Open Findings, CAPs In
Progress, Overdue CAPs, and Inspection Team. Task cards link to Audits, Reports
Approval, Risk Dashboard, Inspection Team, Findings Review, CAP Monitoring, and
Checklist Management. Recent high-risk findings and upcoming audits use shared
state and `Fly Namibia` naming.

## CAP Monitoring

The page answers: “Which corrective action plans are open, overdue, or blocked,
and what evidence or owner action is next?” It provides inspection, status,
department, Due Date, and date-range filters; compact CAP counters; a table with
CAP ID, related finding, inspection, department, finding level, status, action
owner, Due Date, days left/overdue, progress, last update, and row actions; and
small status/overdue/upcoming summaries below the table.

The row ellipsis opens a right-side CAP detail drawer for that exact record.
The drawer contains `Overview`, `Action Plan`, `Updates`, `Documents`, and
`History` tabs. Overview shows status, owner, assignee, Due Date, priority,
target closure date, finding description, impact/risk, root cause, configured
regulatory reference, linked finding, progress, latest update, attachment
filenames, and mock notification history. `Add Update` appends a browser-local
update and history entry. CAP acceptance remains separate from finding closure.

## Checklist Management

The Department Manager can manage department checklist packages in a
package-section-question layout matching the supplied reference. The left rail
lists packages and supports create, select, duplicate, archive, and publish new
version. The selected package shows information, sections/questions,
attachments, history, owner, department, effective date, status, and version.

Within the selected package the manager can add/remove/reorder sections and add,
edit, duplicate, activate/deactivate, or remove questions. Question editing
supports question text, configured requirement/reference, guidance note,
evidence methods, likelihood, impact, calculated demo risk level, finding types,
mandatory/critical toggles, and status. All changes persist only in browser-local
demo state. Published versions are preserved; edits create or update a draft
rather than silently overwriting a published version.

## Risk Dashboard

The Department Manager Risk Dashboard provides date, department, inspection,
and risk-level filters plus export of a browser-side demo summary. It shows the
smallest useful decision set: finding totals by risk, trend, risk exposure
matrix, top risky areas, department distribution, overdue CAPs by risk level,
and recent high-risk findings. Risk scores are management indicators only and
must not trigger automatic legal, enforcement, certificate, or closure action.

## General Manager Experience

The General Manager Dashboard provides Pending Final Reports, High Risk
Findings, Reports Awaiting Your Approval, and Overdue CAP counters; a department
overview; a compact risk heat map and distribution; and a Final Report approval
queue. Each queue row can open the report, return it with a required comment, or
advance it to Executive Director. GM is an intermediate reviewer and cannot
issue, sign, or lock a Final Report.

`Report Approvals` opens the General Manager approval queue, `Departments`
opens the department overview, and `Risk Dashboard` opens the cross-department
management risk view. The General Manager does not receive the Department
Manager's team-editing or checklist-editing controls.

## Executive Director Final Authority

Executive Director owns the final report decision. The dedicated Final Reports
workspace may approve, return, reject, or refer an eligible report for
configured enforcement review. Approval records a clearly labelled mock demo
signature, issues the selected report, and locks it. A referral is a
recommendation only and cannot execute a sanction or close an Audit, Finding,
CAP, or Evidence requirement.

Executive Director Planning approval is also a mock approval mark. It does not
release an audit directly; the next action remains `GM / Release to Department`.

## Interaction And Error States

- Filters show a clear empty state when no rows match.
- A missing selected record falls back to the first visible row or an empty
  detail state; it must not throw a render error.
- Approval actions are disabled after a terminal decision for that manager
  stage.
- Revision/return actions show inline validation when the required comment is
  empty.
- Team changes prevent duplicate members and prevent removal of the current
  Lead Inspector until another lead is selected.
- Downloads report a visible failure state if the browser cannot create a Blob
  URL.
- Closing a menu, modal, or detail pane preserves the underlying selected row
  unless the user selects another row.

## Accessibility And Responsive Behavior

- Use semantic buttons for row actions, tabs, menu items, and decisions.
- Provide visible focus states and `aria-expanded` for ellipsis menus.
- Associate tabs with their selected state.
- Do not communicate severity or status by color alone.
- Preserve the existing restrained aviation-authority visual language.
- Verify desktop at `1536x864` and mobile at `390x844` with no page-level
  horizontal overflow, clipped primary actions, or overlapping controls.

## Verification Design

Implementation follows test-first behavior changes. Focused smoke coverage must
prove:

- canonical `Fly Namibia` naming and absence of legacy visible variants;
- manager navigation and initial Fly Namibia selections;
- Findings Review filters, tabs, aggregates, and detail navigation;
- Inspection Team manager scoping, ellipsis menu, detail pane, member/schedule
  mutations, message record, history, and assignment PDF;
- separate Preliminary and Final Report artifacts;
- Department Manager approve, revision, and return paths for both report types;
- required comments for revision/return;
- PDF headers, MIME type, filenames, and browser download trigger;
- exact Department Manager and General Manager sidebar allowlists;
- CAP Monitoring filters, row ellipsis, drawer tabs, update/history behavior,
  and closure-boundary wording;
- Checklist Management package, version, section, and question mutations;
- Department Manager risk aggregates and navigation from dashboard cards;
- General Manager department/risk projections, required-comment return, and
  intermediate advance to Executive Director;
- Executive Director-only Final Report issue/sign/lock authority and
  non-automatic enforcement referral;
- saved-state migration defaults;
- CAP/evidence lifecycle and auditee privacy regressions remain protected.

After focused tests, run the full direct Node smoke suite, syntax checks,
`git diff --check`, desktop/mobile browser click-through, console review, and
PDF file inspection. Browser automation must use an isolated profile and be
cleaned up afterward.

## Out Of Scope

- Backend, database, or API work
- Real authentication or authorization enforcement
- Real file upload, evidence storage, or document repository
- Real email, SMS, or external notifications
- Production PDF/reporting engine or e-signature
- Automatic legal, enforcement, certificate, or closure decisions
- Framework migration or package-manager setup
- Advanced BI, predictive analytics, or an unrestricted chart builder

## Acceptance Criteria

The design is satisfied when the Department Manager and General Manager role
demos can:

1. open Findings Review and immediately select a Fly Namibia inspection;
2. inspect the finding overview and detailed finding records;
3. open Inspection Team, use the row ellipsis, and see/manage the selected team
   within the allowed demo behaviors;
4. open Reports Approval and review both Preliminary and Final Reports;
5. approve, request revision, or return the appropriate report with visible
   state/history changes;
6. download valid client-side Preliminary, Final, Executive Summary, and team
   assignment demo PDFs;
7. monitor CAPs and open the selected CAP detail drawer from the row ellipsis;
8. create and maintain browser-local checklist packages, sections, questions,
   and preserved versions;
9. use the Department Manager Dashboard and Risk Dashboard with only the
   approved sidebar entries;
10. switch to the General Manager experience, review department/risk summaries,
    and return or advance a Final Report without issuing/signing/locking it;
11. switch to Executive Director, make the final report decision, and preview
    the selected state-backed report with a demo-only approval mark;
12. complete these paths at desktop and mobile sizes without console errors or
    layout overflow.

## Stakeholder Readiness Remediation Evidence — 2026-07-10

Distinct canonical `PR-2026-018` and `FR-2026-018` artifacts now preserve exact selected identity across Department Manager decisions and Lead Final list, preparation, preview, attachments, inspection record, submission, and mock PDF paths. GM remains an intermediate return/forward reviewer; Executive Director alone may issue, add the demo mock approval mark, and lock an eligible Final Report. Report approval leaves open Findings in `Follow-up Open`; CAP acceptance remains separate from Evidence verification and Finding closure.

Status: **demo-only**; focused/static checks and fresh isolated-browser report/authority/PDF/responsive checks at all four required viewports are **verified locally**. Production authorization, signature validity, enforcement execution, release, and external stakeholder acceptance are **not run**; **production-readiness not claimed**.
