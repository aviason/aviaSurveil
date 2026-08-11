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
- A governed operational checklist can use only separately published,
  package-eligible versions. `PREPROD_EXERCISE` questions are confined to a
  dedicated disposable preprod profile and cannot be promoted or published.
- Question Review has explicit governed and exercise command boundaries. The
  exercise boundary records review/classification facts only and cannot invoke
  technical approval or publication.
- Non-Compliant or Observation plus a required comment can create an
  audit-scoped Potential Finding for Lead Inspector review
- Only Lead conversion creates the canonical Finding
- Observation CAP, Evidence, and Due Date are optional by configuration
- Submitted checklist reopen requires Inspector/Lead authority, a valid stage,
  and a reason
- Materialized checklists are `NOT_STARTED` until the atomic Inspector-start
  command. Before start, responses, Potential Findings, execution-package
  access, offline execution grants/sync, and execution events fail closed.

## Governed AGA checklist-intake contract (candidate-only)

The governed lifecycle is `EXISTING_CHECKLIST_CANDIDATE` or
`REGULATORY_TRACE` intake, optional `HYBRID_RECONCILED` Draft revision,
source-authority acceptance, candidate mapping attestation, Department Manager
technical approval, separate Department Manager publication, and separately
computed Audit-package eligibility. Each fact is immutable and append-only.

Existing checklist material is non-authoritative candidate input. A question
without a complete current official chain is visibly `SOURCE_MAPPING_REQUIRED`;
it cannot be published, technically approved, automatically deferred, or used
in an executable Audit package. A resolved question requires a complete
`OFFICIAL_CHECKLIST_SOURCE_CHAIN_V1` and never accepts a client citation string
as source authority.

## UX direction

The screen must show status, owner, due date and next action before secondary details. Advanced configuration must stay behind admin permissions.

## MVP acceptance criteria

- Supports the operator audit demo scenario.
- Critical actions are audit logged.
- Auditee-visible and internal information stay separated.
- The user can complete the primary task without leaving the screen.

## Retired AGA candidate donor

The separate `aga-candidate-demo@1.1.0` Checklist Builder projection and panel
were physically removed after canonical Task 9 qualification and the user's
explicit `delete` decision. Checklist Builder no longer exposes a duplicate
sealed-candidate product. Historical overlay receipts remain evidence only;
canonical catalog/import provenance and Question Review own current behavior.
