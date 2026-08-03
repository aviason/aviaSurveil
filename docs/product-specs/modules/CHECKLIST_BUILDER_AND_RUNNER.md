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

## AGA candidate demo overlay

`aga-candidate-demo@1.1.0` is a separate, read-only, immutable, preprod-only,
Admin-only Checklist Builder projection. It cannot satisfy Task 9, alter a
frozen synthetic profile, or enter the governed lifecycle. When the exact CAA
Admin capability is present, the panel shows sealed non-authoritative forms,
the 21-form `QUESTION_EXTRACTION_REVIEW_REQUIRED` queue, the 1310-question
`SOURCE_MAPPING_REQUIRED` review material, the explicit 1261/49 source-gap
split, provisional risk distributions, and 14
`EXPERT_RISK_REVIEW_REQUIRED` blockers. It has no import, approve, map,
attest, assign, publish, deliver, Finding, or Audit action. Only successful
Admin reads display `candidate-only`, `release pending`, and
`production-ready: not established`.
