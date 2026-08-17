# Checklist Builder and Runner

## Purpose

Create reusable checklists and let inspectors execute them.

## Key fields

- Template name
- Version
- Section
- Question
- Reference
- Expected evidence
- Default severity
- Answer
- Comment
- Attachment

## Primary actions

- Create template
- Version template
- Start checklist
- Answer item
- Attach file
- Create Potential Finding
- Complete checklist

## Business rules

- Templates are versioned
- Old audits keep old template version
- `question_versions` is the single immutable question body/version authority;
  catalog membership and import lineage may reference it but never copy or
  mutate its body.
- The approved AGA catalog is directly usable after deterministic technical
  integrity/provenance validation. Its source manifest is not a human approval,
  technical approval, or publication gate.
- The approved AGA import preserves all 1,310 immutable question versions.
  Optional risk, regulatory, and historical-review enrichments are advisory
  metadata and never block catalog use or appear as approved conclusions when
  absent.
- A new Audit does not preselect the catalog. The Department Manager selects a
  valid immutable subset from the complete catalog; the selected question
  IDs/versions and catalog root digest are frozen only in that Audit's scope
  snapshot.
- Non-Compliant or Observation plus a required comment can create an
  audit-scoped Potential Finding for Lead Inspector review
- Only Lead conversion creates the canonical Finding
- Observation CAP, Evidence, and Due Date are optional by configuration
- Submitted checklist reopen requires Inspector/Lead authority, a valid stage,
  and a reason
- Materialized checklists are `NOT_STARTED` until the atomic Inspector-start
  command. Before start, responses, Potential Findings, execution-package
  access, offline execution grants/sync, and execution events fail closed.

## Approved AGA catalog intake contract

The loader accepts the Aviation-supplied AGA forms as
`IMPORTED_APPROVED_SOURCE` and records the exact source/package/catalog
digests, form inventory, question count, and ordered question identities. The
loader rejects malformed, duplicate, changed, missing, or extra content, but it
does not create an approval command or require a person to sign the manifest.

The catalog remains searchable and filterable in New Audit. Supplier/provider
scope, regulated target, and application type are selected as a server-owned
cascade before catalog browsing. Application type may add deterministic
advisory focus suggestions; it never removes the Manager's ability to choose
any valid question from the approved catalog.

## UX direction

The screen must show status, owner, due date and next action before secondary details. Advanced configuration must stay behind admin permissions.

## MVP acceptance criteria

- Supports the operator audit demo scenario.
- Critical actions are audit logged.
- Auditee-visible and internal information stay separated.
- The user can complete the primary task without leaving the screen.

## Retired duplicate intake paths

The former candidate, exercise, and duplicate sealed-overlay intake paths were
physically removed. Checklist Builder has one current AGA catalog/import
contract; historical receipts are evidence only and authorize no route,
decision, or runtime behavior.
