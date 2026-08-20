# Screen Inventory and Form Plan

## Demo screens

1. Role Switch / Login Demo
2. Manager Dashboard
3. Inspector Dashboard
4. Audit Plan Calendar
5. New Inspection Planning Intake
6. Audit Detail
7. Checklist Runner
8. Lead Inspector Potential Finding Review
9. Finding Detail
10. Auditee Portal Dashboard
11. CAP Submission Form
12. Evidence Upload
13. Evidence Review
14. Closed Finding / Report Preview
15. Admin Template Preview

## Inspector dashboard

Main goal: show assigned work.

Cards:

- Today's Inspections
- Checklists In Progress
- CAPs Waiting Review
- Evidence Waiting Review
- Due Soon
- Overdue

Primary buttons:

- Start Inspection
- Continue Checklist
- Review CAP
- Review Evidence

## Manager dashboard

Main goal: show risk and delay.

Cards:

- Oversight Health Index
- Open Findings
- Overdue Findings
- Critical Findings
- Plan Completion
- Average Closure Time

Needs Attention list:

- Critical findings open
- Overdue major findings
- Organizations with repeat findings
- Audits not started

## Auditee dashboard

Main goal: show what the CAA needs.

Cards:

- My Open Findings
- CAP Required
- Evidence Required
- Due Soon
- Overdue
- Closed

Primary buttons:

- Submit CAP
- Upload Evidence
- Respond to CAA

## New Audit Planning Proposal

Main goal: create a governed Planning item before any executable Audit exists.

Fields:

- Inspected Organization
- Provider scope and regulated target, server-resolved from authorized options
- Inspection type
- Purpose preset or editable purpose text
- Planned Date
- Mode: On-site or Remote
- Conditional canonical location or optional online meeting link
- Required inspector count
- Estimated checklist-item count and server workload basis/range
- Requested Budget and Currency
- Approval Path

New Audit submits a Planning proposal to Finance. It does not collect Trigger,
editable Risk Category, Domain, historical recommendation, exact checklist
selection, selection digest, or named inspector assignment. The existing
checklist selector begins only after `GM Release to Department`, at the
post-release Department Manager preparation boundary.

## Checklist Runner form

For each checklist item:

- Question
- Regulation reference
- Expected evidence
- Answer: Compliant / Non-Compliant / Observation / Not Applicable / Not Checked
- Required comment for a Non-Compliant or Observation result
- Mock Evidence filename
- Create Potential Finding button scoped to the exact Audit

## Governed AGA intake and authoring screens (candidate-only contract)

Admin inventory presents archive hash, byte count, ordered file/register
receipt states, identity conflict state, and blocking error codes. It never
exposes raw extracted text outside the Admin-only extraction-review surface and
never treats archive contents as regulatory authority. The client cannot select
required Department owners; it may submit scope hints only.

Checklist authoring visibly distinguishes `EXISTING_CHECKLIST_CANDIDATE`,
`REGULATORY_TRACE`, and `HYBRID_RECONCILED`. Every question shows either an
explicit complete regulated trace or literal `SOURCE_MAPPING_REQUIRED`.
Functional-assignment queues are scoped internal views, never a new role or an
Auditee projection. Technical approval, publication, and Audit-package
eligibility remain separately displayed decisions.

Submitted checklists are read-only. Inspector or Lead Inspector may reopen only
at a valid stage through a reason-required confirmation. Templates with no
configured execution package show an explicitly disabled action.

## Lead Inspector Potential Finding review and Finding conversion form

Minimum fields:

- Title
- Description
- Regulation reference
- Severity
- Due date
- CAP required
- Evidence required
- Return with reason
- Dismiss with reason
- Convert to Finding button

Observation initializes with CAP unchecked, Evidence unchecked, and no Due
Date. The Lead Inspector may explicitly configure those fields before
conversion. Conversion writes the canonical Finding; it does not silently
switch roles.

Advanced fields:

- Risk category
- Repeat finding
- Related previous finding
- Internal CAA note

## CAP form for auditee

Use helper text:

- Why did this happen? Root cause.
- What will you do to fix it? Corrective action.
- What will you change so it does not happen again? Preventive action.
- Who is responsible?
- When will it be completed?
- Upload evidence.

## Evidence review form

Fields:

- Mock Evidence filename and version
- Related finding
- Related CAP
- Previous versions
- Decision: Close / Partially Close / Not Close
- Comment to auditee
- Internal CAA note

`Close` records `Evidence accepted and verified`. `Partially Close` and `Not
Close` keep the Finding open. A Department Manager's reason-required authorized
closure is separate from Evidence review. Finding, Auditee, and Manager
surfaces show organization-scoped reminder history with stage, recipient, date,
`demo_recorded` status, and `Demo in-app event; no real delivery`.

## Retired AGA candidate donor panel

The separate `aga-candidate-demo@1.1.0` Checklist Builder panel was physically
removed under canonical Task 9 after donor-free qualification and the user's
explicit `delete` decision. It is not a current screen, capability, or route.
Historical evidence remains candidate-only; canonical Question Review and New
Audit are the current stakeholder surfaces.
