# Stakeholder Feedback Remediation Design

**Date:** 2026-07-15
**Status:** approved
**Product:** AviaSurveil360 frontend-only clickable demo
**Source:** Furkan Ozdemir screenshot feedback supplied on 2026-07-15

## Objective

Resolve the nine stakeholder-reported usability and workflow defects across the
Inspector, Lead Inspector, General Manager, Executive Director, and Finance
Review experiences without changing the static HTML, CSS, and Vanilla
JavaScript architecture or weakening the established Finding, CAP, Evidence,
report-authority, and demo-only boundaries.

The result must remove redundant content, make Preliminary and Final Report
screens fit their containers, seed decision-ready GM/ED/Finance examples, and
change the planning approval sequence to the stakeholder-confirmed order:

`Department Manager -> Finance Review -> GM Review -> Executive Director`

If Finance returns a plan for revision, ownership returns to Department
Manager.

## Chosen Approach

Use a targeted render, seed-state, and approval-contract remediation. Keep the
existing role workspaces and shared approval primitive, but update the smallest
set of view, CSS, state, action, test, and product-truth surfaces necessary to
make the requested behavior real.

This is preferred over a CSS-only patch because the empty GM, ED, and Finance
screens and the planning-stage order are state/behavior defects, not only
visual defects. A broad approval-framework rewrite is rejected because the
existing frontend-only demo can support the requested behavior through focused
contract changes.

## Screen Design

### Inspector My Assignments

- Remove the complete `Next inspection` dossier, including its progress donut,
  current-owner metrics, and duplicate action buttons.
- Start the page with the existing assignment KPI row, followed by filters and
  the assignment table.
- Preserve assignment row actions and the In Progress detail view so removing
  the dossier does not remove working behavior.

### Lead Inspector Preliminary Reports

- Keep report-level status in the Preliminary Report list and workflow header.
- Remove Finding lifecycle/CAP status from the Preliminary Report
  `Inspection & Findings` step because CAP work begins only after the report is
  released to the Service Provider.
- The main Findings Review table shows Finding, Level, Due Date, and the Review
  action; it does not show CAP Submitted, CAP Accepted, Waiting for CAP, Closed,
  or another lifecycle-status column.
- The right-side `Findings from Inspection` selection panel shows inclusion,
  severity, Finding ID/title, and area. It does not render a clipped CAP/status
  badge column.
- At desktop widths the two panels remain inside their cards with no nested
  horizontal overflow or clipped text. Mobile keeps the list-before-detail task
  order.

### Preliminary Report Attachments

- Preserve mock filename-only upload behavior and the current attachment
  summary.
- Give filename and description cells explicit usable widths and controlled
  wrapping.
- Prevent filenames, descriptions, uploader names, dates, sizes, and action
  controls from overlapping at the required desktop widths.
- Retain an internal table scroller only where the viewport genuinely requires
  it; the desktop page must not horizontally overflow.

### Final Report Overview

- Convert the organization/CAP summary strip and `Key Findings by Level`
  content to compact metrics with smaller padding, icons, labels, values, and
  action controls.
- Preserve the same counts, CAP/closure boundary wording, and navigation.
- Keep these summaries visually subordinate to the report content and next
  action.

### General Manager Report Approvals

- Seed at least one unlocked Final Report at `submitted_to_gm` with
  `ownerRole: gm` so the initial queue, counters, and selected detail are not
  empty after a demo reset.
- The selected row exposes working `Open Report`, `Return Report`, and
  `Forward to Executive Director` actions.
- Return still requires a General Manager comment. Forwarding changes the exact
  selected report to Executive Director ownership without issuing, signing, or
  locking it.

### Executive Director Final Reports

- Seed at least one separate unlocked Final Report at
  `submitted_to_executive` with `ownerRole: executiveDirector` and select a
  pending report on first load.
- Keep the decision panel visible in the desktop task flow for the pending
  selection.
- Expose working `Approve Report`, `Return for Revision`, `Reject Report`, and
  `Refer for Enforcement Review` choices.
- Approval uses only the existing clearly labelled demo mock approval mark.
  Enforcement referral remains recommendation-only and has no sanction,
  certificate, CAP, Evidence, Finding-closure, or Audit-closure side effect.

### Finance Review

- Change every planning approval rail and its underlying chain to:
  `Department Manager -> Finance Review -> GM Review -> Executive Director`.
- Seed/reset the selected planning item at the Finance stage so Review Queue
  shows a real sample row and selected budget dossier immediately.
- Finance approval advances the exact plan to GM Review.
- Finance `Return for Revision` requires a reason and returns the exact plan to
  Department Manager.
- GM forwards a Finance-reviewed plan to Executive Director. GM
  `Return for Revision` returns the plan to Department Manager; a corrected
  resubmission must pass through Finance Review again before returning to GM.
- Executive Director approval keeps the existing post-approval
  `GM Release to Department` boundary; it does not send the plan directly to
  execution.

## State And Migration

- Add distinct GM-pending and ED-pending Final Report sample artifacts rather
  than representing both queues with one mutable report.
- Add every interactive sample to the canonical report artifact collection and
  its read projection so list, detail, decision, history, preview, and PDF paths
  resolve the same Report ID.
- Use existing audits and organizations only where their report stage is
  internally coherent; do not expose cross-organization records to the Service
  Provider Portal.
- Bump the browser demo-state version when required and merge new seed IDs and
  approval-chain order safely into older saved state.
- Do not reset unrelated user-created browser-local demo records.

## Interaction And Error Handling

- Empty filtered queues remain valid empty states; the default reset state must
  contain the requested GM, ED, and Finance sample records.
- Unsupported actors and wrong-stage decisions must not mutate status, owner,
  history, notifications, or audit logs.
- A missing selected record falls back only to another eligible record in the
  same role/stage, never an unrelated issued or historical record.
- All visible buttons must navigate, open a focused modal, update visible
  browser-local state, or generate the documented mock artifact.
- CAP acceptance remains distinct from Finding closure, and Preliminary/Final
  Report decisions never bypass required CAP, Evidence, or verification work.

## Testing Strategy

Use focused regression coverage for every behavior change:

1. Add render assertions for removal of the My Assignments dossier,
   removal of Preliminary CAP-status columns, and presence of compact layout
   hooks.
2. Add seed/projection assertions proving default GM, ED, and Finance
   decision-ready rows.
3. Add planning-contract assertions for
   `Department Manager -> Finance -> GM -> Executive Director`, Finance approval
   to GM, and Finance return to Department Manager.
4. Implement the smallest correction and rerun the focused test after each
   unit.
5. Run all JavaScript syntax checks, the full Node smoke suite, and
   `git diff --check`.
6. Run rendered browser verification at `1536x864`, `1366x768`, `1024x768`,
   and `390x844`, covering the affected roles and at least one decision per
   changed approval surface.

Rendered checks must cover page identity, meaningful content, console health,
no framework/error overlay, no page or relevant nested overflow, no clipped or
overlapping controls, selected-ID stability, and visible state changes after
interactions. Temporary browser and server processes must be cleaned up.

## Documentation And Tracking

- Update canonical English product/permission/workflow text that still states
  the old planning order, plus matching `.turkce.md` companions where they
  exist.
- Update English/Turkish build summaries only with fresh verification evidence.
- Add or update a focused active execution plan under `docs/exec-plans/active/`
  and keep its index row and next todo synchronized with the actual state.
- Preserve literal evidence labels: `demo-only`, `verified locally`, `not run`,
  and `production-readiness not claimed`.

## Out Of Scope

- Backend, API, database, real authentication, or production authorization.
- Real e-signature, legal approval identity, enforcement execution, file
  upload/storage, notification delivery, reporting service, or immutable audit
  log.
- Framework migration or a broad redesign of unrelated screens.
- Automatic Finding, CAP, Evidence, certificate, enforcement, or Audit closure.
- Branch, commit, push, deployment, PR, or GitHub comment activity without
  separate current-task authorization.

## Success Criteria

- All nine stakeholder notes have a direct automated assertion and rendered
  verification check.
- My Assignments no longer renders the redundant Next Inspection dossier.
- Preliminary Findings content fits and contains no CAP/lifecycle-status
  column; attachment text does not overlap.
- Final Report summaries are visibly compact without losing their data or
  actions.
- GM, ED, and Finance open with meaningful decision-ready sample records.
- GM and ED report decisions operate on distinct exact report artifacts.
- Planning ownership follows Department Manager, Finance, GM, and Executive
  Director in that order; Finance return goes to Department Manager.
- All changed visible controls work, all targeted/full checks pass, and the
  four-viewport browser matrix has no relevant console, clipping, overlap, or
  overflow defect.
- The result remains explicitly `demo-only`; production readiness is not
  claimed.
