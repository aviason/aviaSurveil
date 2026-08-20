# Audit Planning

## Purpose

Create annual and ad hoc audit/inspection plans.

## Key fields

- Planning proposal ID
- Inspected Organization
- Provider scope and regulated target
- Inspection type
- Purpose
- Planned date and mode
- Conditional canonical location or remote meeting link
- Required inspector count
- Estimated checklist-item count and server workload estimate
- Requested budget and currency
- Planning approval status

## Primary actions

- Submit annual planning for Finance review
- Forward a Finance-reviewed plan through General Manager and Executive Director approval
- Perform `GM Release to Department` after Executive Director approval
- Prepare the released plan in the Department
- Create audit
- Schedule
- Reschedule
- Assign inspector
- Review the Planning proposal dossier before Finance submission
- Publish plan
- Send Service Provider coordination package when advance notice is required
- Confirm a proposed date or accept a Service Provider alternative date

## Business rules

- Manual scheduling is MVP
- Annual planning follows Department Manager -> Finance -> General Manager ->
  Executive Director -> GM Release to Department -> Department preparation.
- Executive Director approval does not release the plan. General Manager
  release remains a separate recorded next action.
- Audit type and provider scope constrain the selectable catalog questions; a
  pre-approval New Audit does not select a checklist template.
- Inspection type/configuration determines whether the Service Provider is
  notified in advance
- Routine / Announced inspections share the released question scope and
  relevant coordination information only after the Lead Inspector is
  identified.
- Ad Hoc / Unannounced inspections skip the Service Provider coordination step
- A Service Provider may confirm the proposed date or suggest an alternative;
  the CAA must accept an alternative before execution is ready
- Reschedule requires reason
- Completed audits cannot be deleted
- One organization may have several active provider scopes. Provider scope is
  separate from the coarse organization type and may target an organization,
  person, facility, device, system, asset, or location.
- Operational selection for an imported Aviation-approved catalog consumes the
  sealed source-approved question versions directly for the applicable
  provider scope and typed target. Optional risk or regulatory enrichment is
  non-authoritative and does not block selection or represent an approved
  conclusion. Internally generated candidates remain separate from the
  imported approved source and must not be presented as its approval gate.
- A disposable preprod exercise may select `PREPROD_EXERCISE` catalog members
  only when the dedicated disposable preprod profile is active. Exercise
  records cannot be published, promoted to operational use, or used to satisfy
  governed package eligibility.
- New Audit freezes an immutable Planning proposal snapshot without exact
  checklist identities. The approved checklist-item estimate is the Planning
  ceiling for later preparation.
- After release, a separate Audit-package scope owns catalog identity, history
  evaluation, exact checklist selection, digest, assignment, and preparation.
  Planning `RELEASED` and Audit-package `FINALIZED` are distinct states.
- Selection does not itself make work executable. Announced materialization
  starts at `AWAITING_AUDITEE_CONFIRMATION`; unannounced materialization starts
  at `SCHEDULED` with notice withheld. Both create a `NOT_STARTED` checklist;
  an authorized Inspector start is a separate atomic transition.

## UX direction

The screen must show status, owner, due date and next action before secondary details. Advanced configuration must stay behind admin permissions.

## MVP acceptance criteria

- Supports the operator audit demo scenario.
- Critical actions are audit logged.
- Auditee-visible and internal information stay separated.
- Advance-notice-required inspections expose only the configured coordination
  package to the matching Service Provider organization.
- Executive Director planning approval leaves `GM Release to Department` as
  the next action.
- The user can complete the primary task without leaving the screen.
