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
- Non-Compliant or Observation plus a required comment can create an
  audit-scoped Potential Finding for Lead Inspector review
- Only Lead conversion creates the canonical Finding
- Observation CAP, Evidence, and Due Date are optional by configuration
- Submitted checklist reopen requires Inspector/Lead authority, a valid stage,
  and a reason

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
