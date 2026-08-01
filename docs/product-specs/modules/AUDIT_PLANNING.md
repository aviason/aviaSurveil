# Audit Planning

## Purpose

Create annual and ad hoc audit/inspection plans.

## Key fields

- Audit ID
- Organization
- Audit type
- Inspection category
- Advance-notice policy
- Domain
- Planned date
- Location
- Remote/on-site
- Lead inspector
- Team
- Checklist template
- Provider scopes and regulated target
- Scope
- Status

## Primary actions

- Submit annual planning for Finance review
- Forward a Finance-reviewed plan through General Manager and Executive Director approval
- Perform `GM Release to Department` after Executive Director approval
- Prepare the released plan in the Department
- Create audit
- Schedule
- Reschedule
- Assign inspector
- Select checklist
- Publish plan
- Send Service Provider coordination package when advance notice is required
- Confirm a proposed date or accept a Service Provider alternative date

## Business rules

- Manual scheduling is MVP
- Annual planning follows Department Manager -> Finance -> General Manager ->
  Executive Director -> GM Release to Department -> Department preparation.
- Executive Director approval does not release the plan. General Manager
  release remains a separate recorded next action.
- Audit type determines default templates
- Inspection type/configuration determines whether the Service Provider is
  notified in advance
- Routine / Announced inspections share the proposed date, checklist, and
  relevant information after the Lead Inspector is identified
- Ad Hoc / Unannounced inspections skip the Service Provider coordination step
- A Service Provider may confirm the proposed date or suggest an alternative;
  the CAA must accept an alternative before execution is ready
- Reschedule requires reason
- Completed audits cannot be deleted
- One organization may have several active provider scopes. Provider scope is
  separate from the coarse organization type and may target an organization,
  person, facility, device, system, asset, or location.
- Selecting a published checklist requires the applicable provider scope and
  typed target; a Department Manager's technical approval and publication are
  separate decisions made before selection.
- Selecting a published checklist does not itself make it executable. Computed
  Audit-package eligibility is separate from publication and fails closed when
  the exact immutable version, scope, typed target, source currentness,
  required owners, or question trace is incomplete.

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
