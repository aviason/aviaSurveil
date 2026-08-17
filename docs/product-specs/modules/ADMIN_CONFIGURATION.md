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

## Approved AGA catalog administration boundary

Admin may inspect the immutable approved AGA source manifest, inventory, and
catalog provenance. The manifest is deterministic integrity evidence; it is
not a human approval or a deployment/publication gate. The loader's technical
validation is sufficient to make the imported catalog available for New Audit.
Admin does not need to approve the catalog before a Department Manager can
select a valid subset for an Audit.

The legacy source-owner/reviewer assignment and publication workflow is not a
prerequisite for the approved AGA catalog. No extra top-level role, approval
command, bulk approval, or separate publication route is created for this
catalog.

## UX direction

The screen must show status, owner, due date and next action before secondary details. Advanced configuration must stay behind admin permissions.

## MVP acceptance criteria

- Supports the operator audit demo scenario.
- Critical actions are audit logged.
- Auditee-visible and internal information stay separated.
- The user can complete the primary task without leaving the screen.

## Retired duplicate intake paths

The former candidate/exercise Admin projection was physically removed.
Administration now exposes only approved catalog provenance and current
configuration surfaces. Historical receipts authorize no current route,
decision, or runtime.
