# Admin Configuration

## Purpose

Configure templates and rules without code changes.

## Key fields

- Audit types
- Checklist templates
- Severity levels
- Due-date rules
- Notification templates
- Roles
- Permissions
- Report templates

## Primary actions

- Create template
- Version template
- Activate/deactivate
- Configure rule
- Preview impact

## Business rules

- Published template changes create new version
- Admin changes audited
- Normal users cannot edit config during audits

## Governed intake administration boundary

Admin may receive a candidate-only AGA archive, review its immutable inventory,
resolve a file identity, and prepare a Draft. Admin cannot establish source
authority, technically approve, publish, or make an Audit package eligible.

`REGULATORY_SOURCE_OWNER` and `CHECKLIST_REVIEWER` are scoped,
effective-dated functional assignments for authenticated internal CAA users;
they are not top-level roles. `REVIEWED_SOURCE_SET` assignment provisioning is
blocked pending a named governance directive. There is no Admin grant/revoke
route and synthetic fixtures only may seed those assignments. Missing,
ambiguous, expired, revoked, or out-of-scope assignments fail closed.

## UX direction

The screen must show status, owner, due date and next action before secondary details. Advanced configuration must stay behind admin permissions.

## MVP acceptance criteria

- Supports the operator audit demo scenario.
- Critical actions are audit logged.
- Auditee-visible and internal information stay separated.
- The user can complete the primary task without leaving the screen.

## Retired AGA candidate donor

The separate `aga-candidate-demo@1.1.0` Admin projection was physically removed
after canonical Task 9 donor-free qualification and the user's explicit
`delete` decision. Administration now uses only canonical governed intake,
catalog provenance, and Question Review boundaries. Historical receipts remain
candidate-only evidence and authorize no current route, decision, or runtime.
